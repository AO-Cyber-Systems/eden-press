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

const testDeck = "# Hello Convert\n\nWorld content.\n"

// TestRunConvertFileToStdout is test-list case 4: a file argument (no
// subcommand, root's default action) writes a standalone zero-JS document
// to stdout containing the deck's rendered content.
func TestRunConvertFileToStdout(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(testDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{path})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "<!doctype html>") {
		t.Errorf("stdout missing <!doctype html>: %q", got)
	}
	if !strings.Contains(got, "Hello Convert") {
		t.Errorf("stdout missing rendered deck content: %q", got)
	}
	if strings.Contains(got, "<script") {
		t.Errorf("default convert output contains <script>, want zero-JS: %q", got)
	}
}

// TestRunConvertStdinToStdout is test-list case 5: "-" reads injected
// stdin and emits the same standalone doc to stdout.
func TestRunConvertStdinToStdout(t *testing.T) {
	resetCfg()

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(testDeck))
	root.SetArgs([]string{"-"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "<!doctype html>") {
		t.Errorf("stdout missing <!doctype html>: %q", got)
	}
	if !strings.Contains(got, "Hello Convert") {
		t.Errorf("stdout missing rendered deck content: %q", got)
	}
	if strings.Contains(got, "<script") {
		t.Errorf("default convert output contains <script>, want zero-JS: %q", got)
	}
}

// TestRunConvertOutputFile is test-list case 6: `convert --output out.html`
// writes the doc to the file, NOT stdout.
func TestRunConvertOutputFile(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	inPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(inPath, []byte(testDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	outPath := filepath.Join(dir, "out.html")

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"convert", inPath, "--output", outPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if out.String() != "" {
		t.Errorf("stdout = %q, want empty (output should go to the file)", out.String())
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", outPath, err)
	}
	got := string(written)
	if !strings.Contains(got, "<!doctype html>") {
		t.Errorf("output file missing <!doctype html>: %q", got)
	}
	if !strings.Contains(got, "Hello Convert") {
		t.Errorf("output file missing rendered deck content: %q", got)
	}
	if strings.Contains(got, "<script") {
		t.Errorf("default convert output contains <script>, want zero-JS: %q", got)
	}
}
