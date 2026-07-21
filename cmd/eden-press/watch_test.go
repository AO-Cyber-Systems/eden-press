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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/AO-Cyber-Systems/eden-press/cmd/eden-press/reload"
)

// TestDebounced is test-list case 1: N rapid calls within the window fire
// the target fn exactly once, after the quiet period.
func TestDebounced(t *testing.T) {
	var calls int32
	rebuild := debounced(50*time.Millisecond, func() { atomic.AddInt32(&calls, 1) })

	for range 10 {
		rebuild()
		time.Sleep(5 * time.Millisecond) // well inside the 50ms window -- keeps resetting the timer
	}

	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("calls = %d before the quiet period elapsed, want 0", got)
	}

	time.Sleep(150 * time.Millisecond) // safely past the last reset + window

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d after the quiet period, want exactly 1", got)
	}
}

// TestIsBackupOrSwap is test-list case 2: isBackupOrSwap rejects
// deck.md~, .deck.md.swp, 4913 (Vim's writable-probe file) and accepts
// deck.md.
func TestIsBackupOrSwap(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"deck.md~", true},
		{"/tmp/dir/deck.md~", true},
		{".deck.md.swp", true},
		{"/tmp/dir/.deck.md.swp", true},
		{"4913", true},
		{"/tmp/dir/4913", true},
		{"deck.md", false},
		{"/tmp/dir/deck.md", false},
	}
	for _, c := range cases {
		if got := isBackupOrSwap(c.path); got != c.want {
			t.Errorf("isBackupOrSwap(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestEventTriggersRebuild is test-list case 3 (name filter) plus the
// Chmod/backup-swap/directory-rescan filtering rules from must_haves: an
// event on a sibling file in the watched dir does NOT trigger a rebuild; an
// event on the target file does; Chmod and backup/swap noise never
// triggers; a directory-level event (its Name equal to the watched dir
// itself) is treated as a re-scan signal and DOES trigger.
func TestEventTriggersRebuild(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "deck.md")
	sibling := filepath.Join(dir, "notes.md")
	watched := map[string]bool{target: true}
	dirs := map[string]bool{dir: true}

	cases := []struct {
		name string
		ev   fsnotify.Event
		want bool
	}{
		{"target file Write", fsnotify.Event{Name: target, Op: fsnotify.Write}, true},
		{"target file Create (atomic-rename)", fsnotify.Event{Name: target, Op: fsnotify.Create}, true},
		{"target file Rename", fsnotify.Event{Name: target, Op: fsnotify.Rename}, true},
		{"sibling file Write", fsnotify.Event{Name: sibling, Op: fsnotify.Write}, false},
		{"target file Chmod ignored", fsnotify.Event{Name: target, Op: fsnotify.Chmod}, false},
		{"target backup file ignored", fsnotify.Event{Name: target + "~", Op: fsnotify.Write}, false},
		{"directory-level Write is a re-scan signal", fsnotify.Event{Name: dir, Op: fsnotify.Write}, true},
		{"unrelated path", fsnotify.Event{Name: "/somewhere/else.md", Op: fsnotify.Write}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := eventTriggersRebuild(c.ev, watched, dirs); got != c.want {
				t.Errorf("eventTriggersRebuild(%+v) = %v, want %v", c.ev, got, c.want)
			}
		})
	}
}

// TestWatchScopeResolvesInputAndThemeSetDirs proves the scope resolver
// (must_haves: "input-file dir + any loaded theme-set file dir") includes
// both the input file's own directory and every theme-set path's
// directory, deduplicated, so widening scope later is a change to this one
// function (research Pattern 3 / Open Question 3).
func TestWatchScopeResolvesInputAndThemeSetDirs(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "themes")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(dir, "deck.md")
	theme := filepath.Join(sub, "custom.css")

	dirs, watched := watchScope(in, []string{theme})

	if !watched[in] {
		t.Errorf("watched set missing input file %q: %v", in, watched)
	}
	if !watched[theme] {
		t.Errorf("watched set missing theme-set file %q: %v", theme, watched)
	}

	dirSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		dirSet[d] = true
	}
	if !dirSet[dir] {
		t.Errorf("dirs missing input file's parent dir %q: %v", dir, dirs)
	}
	if !dirSet[sub] {
		t.Errorf("dirs missing theme-set file's parent dir %q: %v", sub, dirs)
	}
}

// TestRunWatchRejectsStdin is test-list case 4 (Pitfall 8): watch on stdin
// ("-") is rejected early with a clear error, and returns immediately
// (never blocks trying to construct a watcher against a non-existent
// path).
func TestRunWatchRejectsStdin(t *testing.T) {
	resetCfg()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"watch", "-"})

	done := make(chan error, 1)
	go func() { done <- root.Execute() }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Execute() = nil error, want a stdin-rejection error")
		}
		if !strings.Contains(err.Error(), "stdin") {
			t.Errorf("error = %q, want it to mention stdin", err.Error())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runWatch did not return promptly when passed stdin (\"-\") -- Pitfall 8 guard missing")
	}
}

// TestRebuildOnceInjectsReloadClientOnly is test-list case 6: the assembled
// watch-session HTML contains the EventSource snippet (via InjectScripts),
// and the file it writes is exactly the default "<input-stem>.html" when
// --output is unset. The companion half of case 6 (default convert output
// still carries no <script>) is already regression-locked by
// TestRunConvertFileToStdout/TestRunConvertStdinToStdout in convert_test.go.
func TestRebuildOnceInjectsReloadClientOnly(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	inPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(inPath, []byte(testDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	wantOut := filepath.Join(dir, "deck.html")

	cmd := newTestConvertCmd() // persistent + auto-fit-script flag surface is enough; watch reads the same cfg seam
	registerWatchFlags(cmd)
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)

	hub := reload.NewHub()
	if _, err := hub.Start(); err != nil {
		t.Fatalf("hub.Start: %v", err)
	}
	t.Cleanup(func() { hub.Close() })

	rebuildOnce(cmd, inPath, hub)

	written, err := os.ReadFile(wantOut)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v (stderr: %q)", wantOut, err, errBuf.String())
	}
	got := string(written)

	if !strings.Contains(got, "Hello Convert") {
		t.Errorf("output missing rendered deck content: %q", got)
	}
	if !strings.Contains(got, "EventSource") {
		t.Errorf("watch-session output missing the injected EventSource reload client: %q", got)
	}
	if n := strings.Count(got, "<script"); n != 1 {
		t.Errorf("<script> count = %d, want exactly 1 (the reload client only)", n)
	}
}

// TestRunWatchRebuildsOnAtomicSave is an end-to-end, poll-with-timeout
// tolerant proof of Pitfall 2: watching the PARENT DIRECTORY (never the
// file itself) survives an editor's write-temp-then-rename save, and the
// rebuilt output reflects the new content. Bounded via cmd.Context()
// cancellation so the test can never hang forever even if fsnotify
// misbehaves in this environment (documented per-OS manual verification is
// still owned by the TRD's own verification section; this is the automated
// macOS proof for this dev host).
func TestRunWatchRebuildsOnAtomicSave(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	inPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(inPath, []byte("# v1\n\nfirst\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	outPath := filepath.Join(dir, "out.html")

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"watch", inPath, "--output", outPath})

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- root.ExecuteContext(ctx) }()

	waitForFileContent(t, outPath, "first", 2*time.Second)

	// Atomic-save-safe rewrite: write to a temp file in the SAME directory,
	// then rename over the original -- the exact editor pattern Pitfall 2
	// targets. A naive single-file watch loses the watch here; watching the
	// parent dir + name-filtering must not.
	tmp := filepath.Join(dir, ".deck.md.tmp")
	if err := os.WriteFile(tmp, []byte("# v2\n\nsecond\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(tmp): %v", err)
	}
	if err := os.Rename(tmp, inPath); err != nil {
		t.Fatalf("os.Rename: %v", err)
	}

	waitForFileContent(t, outPath, "second", 2*time.Second)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runWatch did not return after context cancellation")
	}
}

// waitForFileContent polls path until its content contains want or timeout
// elapses -- the TRD's own recommended tolerance for fsnotify's inherently
// non-deterministic timing (research: "keep the end-to-end file-change test
// tolerant").
func waitForFileContent(t *testing.T, path, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	var lastContent string
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err == nil {
			lastContent = string(b)
			if strings.Contains(lastContent, want) {
				return
			}
		} else {
			lastErr = err
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q to contain %q; last content = %q; last read err = %v", path, want, lastContent, lastErr)
}
