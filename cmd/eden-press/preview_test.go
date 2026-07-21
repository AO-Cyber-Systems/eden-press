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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPreviewDeck = "# Hello Preview\n\nPreview content.\n"

// TestPreviewOpensRenderedFile is test-list case 1: with openURL swapped to
// capture, `runPreview deck.md` converts to a temp file:// html and calls
// openURL with a path whose contents are the standalone doc -- proving the
// launch is behind a testable seam and never spawns a real browser.
func TestPreviewOpensRenderedFile(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(testPreviewDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	orig := openURL
	var captured string
	openURL = func(url string) error {
		captured = url
		return nil
	}
	t.Cleanup(func() { openURL = orig })

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"preview", path})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.HasPrefix(captured, "file://") {
		t.Fatalf("openURL captured = %q, want a file:// prefix", captured)
	}
	filePath := strings.TrimPrefix(captured, "file://")
	t.Cleanup(func() { os.Remove(filePath) })

	base := filepath.Base(filePath)
	if !strings.HasPrefix(base, "eden-press-") || !strings.HasSuffix(base, ".html") {
		t.Errorf("temp file name = %q, want an \"eden-press-*.html\" pattern", base)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", filePath, err)
	}
	got := string(content)
	if !strings.Contains(got, "<!doctype html>") {
		t.Errorf("temp file missing <!doctype html>: %q", got)
	}
	if !strings.Contains(got, "Hello Preview") {
		t.Errorf("temp file missing rendered deck content: %q", got)
	}
}

// TestPreviewRejectsStdin is test-list case 2: `runPreview -` returns a
// clear error without converting or calling openURL at all -- like watch
// (04-06), preview needs a real file path to (re-)open.
func TestPreviewRejectsStdin(t *testing.T) {
	resetCfg()

	orig := openURL
	var called bool
	openURL = func(string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { openURL = orig })

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(testPreviewDeck))
	root.SetArgs([]string{"preview", "-"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute: want an error for `preview -`, got nil")
	}
	if !strings.Contains(err.Error(), "cannot preview stdin") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "cannot preview stdin")
	}
	if called {
		t.Error("openURL was called for a rejected stdin preview, want no call")
	}
}
