// Copyright (c) 2026 AO Cyber Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/AO-Cyber-Systems/eden-press/cmd/eden-press/reload"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// newWatchCmd registers the "watch" subcommand.
func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch [flags] <in.md>",
		Short: "Watch a Markdown deck and rebuild its HTML on change",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd, args)
		},
	}
	registerWatchFlags(cmd)
	return cmd
}

// runWatch is CLI-02's capstone: `eden-press watch <in.md>` rebuilds the
// output whenever the source changes, and broadcasts a reload signal to any
// open browser over a stdlib SSE channel (cmd/eden-press/reload).
//
// Watch scope (research Pattern 3 / must_haves): the input file's PARENT
// DIRECTORY is what gets Add()'d to the fsnotify watcher -- never the file
// itself, or an editor's atomic write-temp-then-rename save silently kills
// the watch after the first edit (Pitfall 2, the classic Vim bug). Events
// are then filtered down to the exact file(s) this session cares about (see
// watchScope/eventTriggersRebuild), and rapid bursts are debounced (~300ms)
// so one logical save produces exactly one rebuild.
//
// Pitfall 8: stdin ("-") has no path to (re-)watch, so it is rejected here,
// early and explicitly, before any watcher is even constructed.
func runWatch(cmd *cobra.Command, args []string) error {
	arg := "-"
	if len(args) == 1 {
		arg = args[0]
	}
	if arg == "-" {
		return fmt.Errorf("watch: cannot watch stdin (\"-\"); pass a file path")
	}

	absIn, err := filepath.Abs(arg)
	if err != nil {
		return fmt.Errorf("watch: resolve %q: %w", arg, err)
	}
	if _, err := os.Stat(absIn); err != nil {
		return fmt.Errorf("watch: %w", err)
	}

	dirs, watched := watchScope(absIn, cfg.Strings("theme-set"))

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watch: new watcher: %w", err)
	}
	defer w.Close()

	dirSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		if err := w.Add(d); err != nil {
			return fmt.Errorf("watch: add %q: %w", d, err)
		}
		dirSet[filepath.Clean(d)] = true
	}

	hub := reload.NewHub()
	if _, err := hub.Start(); err != nil {
		return fmt.Errorf("watch: start reload server: %w", err)
	}
	defer hub.Close()

	rebuild := debounced(300*time.Millisecond, func() { rebuildOnce(cmd, absIn, hub) })

	rebuildOnce(cmd, absIn, hub) // initial build, unconditional -- watch mode always renders once up front

	fmt.Fprintf(cmd.ErrOrStderr(), "watch: watching %s (reload: %s)\n", absIn, hub.URL())

	ctx := cmd.Context()
	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if eventTriggersRebuild(ev, watched, dirSet) {
				rebuild()
			}
		case werr, ok := <-w.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "watch:", werr)
		case <-ctx.Done():
			return nil
		}
	}
}

// watchScope resolves the set of directories runWatch must Add to the
// fsnotify watcher (dirs), and the exact set of absolute file paths this
// session cares about (watched): the input file's own parent directory,
// plus the parent directory of every already-resolved --theme-set path
// (must_haves: "input-file dir + any loaded theme-set file dir").
//
// Keeping this resolution in one small, pure, independently-testable
// function is deliberate (research Pattern 3 / Open Question 3): widening
// v1's scope to a full recursive walk later is a change to THIS function
// only, never a rewrite of runWatch's event loop.
func watchScope(absIn string, themeSetPaths []string) (dirs []string, watched map[string]bool) {
	watched = map[string]bool{absIn: true}
	dirSet := map[string]bool{filepath.Dir(absIn): true}

	for _, p := range themeSetPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue // themeCSS (themeset.go) already surfaces an unreadable-path error at render time
		}
		watched[abs] = true
		dirSet[filepath.Dir(abs)] = true
	}

	dirs = make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	return dirs, watched
}

// eventTriggersRebuild is the pure predicate runWatch's event loop applies
// to every raw fsnotify.Event, implementing must_haves' filtering rules in
// one independently-testable place:
//
//   - fsnotify.Chmod is always ignored (fires spuriously; Linux inotify in
//     particular emits it ahead of a delayed Remove).
//   - Editor backup/swap noise (isBackupOrSwap) is always ignored.
//   - An event whose (cleaned) Name is exactly one of this session's
//     watched files (the input file, or a loaded theme-set file) triggers a
//     rebuild -- this is what survives an atomic-save rename (Pitfall 2):
//     the rename's Create/Rename event lands on absIn itself.
//   - An event whose Name is the watched DIRECTORY itself (not a specific
//     file within it) is treated as a re-scan signal and also triggers a
//     rebuild, rather than being silently dropped -- directory-level Write
//     semantics differ across kqueue/Windows ("directory contents changed")
//     vs. Linux inotify, and rebuilding is cheap/idempotent here (pure-Go,
//     no headless browser), so re-running the pipeline is the correct,
//     low-cost response to "something changed, re-check."
//   - Anything else (a sibling file this session does not care about) is
//     ignored.
func eventTriggersRebuild(ev fsnotify.Event, watched map[string]bool, dirs map[string]bool) bool {
	if ev.Op&fsnotify.Chmod != 0 {
		return false
	}
	name := filepath.Clean(ev.Name)
	if isBackupOrSwap(name) {
		return false
	}
	if watched[name] {
		return true
	}
	if dirs[name] {
		return true
	}
	return false
}

// isBackupOrSwap reports whether path names a common editor backup/swap
// artifact that must never itself trigger a rebuild: a "~"-suffixed backup
// copy, a ".swp"/".swx" Vim swap file, or Vim's own numeric "4913"
// writable-probe file it creates and immediately removes before every
// atomic save.
func isBackupOrSwap(path string) bool {
	base := filepath.Base(path)
	if strings.HasSuffix(base, "~") {
		return true
	}
	if strings.HasSuffix(base, ".swp") || strings.HasSuffix(base, ".swx") {
		return true
	}
	if base == "4913" {
		return true
	}
	return false
}

// debounced wraps fn so that repeated calls within d of each other collapse
// into a single eventual call, fired once the calls go quiet -- the
// standard idiom for coalescing an editor's several raw fsnotify events per
// logical save into exactly one rebuild (research Pattern 3).
func debounced(d time.Duration, fn func()) func() {
	var mu sync.Mutex
	var t *time.Timer
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if t != nil {
			t.Stop()
		}
		t = time.AfterFunc(d, fn)
	}
}

// rebuildOnce re-runs 04-03's render pipeline (resolveInput -> buildOptions
// -> press.Render -> assembleHTML) against path, splicing the reload
// client through assembleHTML's InjectScripts seam (watch-session output
// ONLY -- the default convert path never calls this), writes the result via
// writeWatchOutput, then broadcasts a reload over hub. Any error along the
// way is logged to cmd's stderr and the watch session KEEPS RUNNING rather
// than exiting -- a single bad save should not kill an otherwise-working
// watch loop.
func rebuildOnce(cmd *cobra.Command, path string, hub *reload.Hub) {
	md, _, err := resolveInput(path)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "watch: rebuild:", err)
		return
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "watch: rebuild:", err)
		return
	}

	out, err := press.Render(md, opts)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "watch: rebuild:", err)
		return
	}

	doc := assembleHTML(out, htmlDocOptions{
		AutoFitScript: cfg.Bool("auto-fit-script"),
		InjectScripts: []string{reload.ClientJS(hub.URL())},
	})

	if err := writeWatchOutput(cmd, path, doc); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "watch: rebuild:", err)
		return
	}

	hub.Broadcast("reload")
}

// writeWatchOutput writes doc to the --output/-o path if set, or to the
// default "<input-stem>.html" derived from inputPath otherwise -- watch's
// own default output pairing (distinct from convert's stdout default,
// since a watch session has nowhere useful to stream repeated stdout
// writes to).
func writeWatchOutput(cmd *cobra.Command, inputPath string, doc string) error {
	path := ""
	if f := cmd.Flags().Lookup("output"); f != nil {
		path = f.Value.String()
	}
	if path == "" {
		ext := filepath.Ext(inputPath)
		stem := strings.TrimSuffix(inputPath, ext)
		path = stem + ".html"
	}

	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("writeWatchOutput: write file %q: %w", path, err)
	}
	return nil
}
