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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveInputStdin is test-list case 4: "-" reads all of the injected
// stdin reader and reports source=stdin.
func TestResolveInputStdin(t *testing.T) {
	want := "# hello from stdin\n"
	md, source, err := resolveInputFrom("-", strings.NewReader(want))
	if err != nil {
		t.Fatalf("resolveInputFrom(\"-\", ...): %v", err)
	}
	if md != want {
		t.Errorf("md = %q, want %q", md, want)
	}
	if source != stdinSource {
		t.Errorf("source = %v, want stdinSource", source)
	}
}

// TestResolveInputFile is test-list case 5 (file half): a path reads that
// file's content and reports source=file.
func TestResolveInputFile(t *testing.T) {
	want := "# hello from a file\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	md, source, err := resolveInputFrom(path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolveInputFrom(%q, ...): %v", path, err)
	}
	if md != want {
		t.Errorf("md = %q, want %q", md, want)
	}
	if source != fileSource {
		t.Errorf("source = %v, want fileSource", source)
	}
}

// TestResolveInputMissingFile is test-list case 5 (error half): a missing
// file path returns a clear, non-nil error rather than silently reading
// stdin or panicking.
func TestResolveInputMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.md")

	_, source, err := resolveInputFrom(path, strings.NewReader(""))
	if err == nil {
		t.Fatal("resolveInputFrom(missing file): got nil error, want a clear error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not mention the missing path %q", err.Error(), path)
	}
	if source != fileSource {
		t.Errorf("source = %v, want fileSource (even on error, the attempted source is observable)", source)
	}
}
