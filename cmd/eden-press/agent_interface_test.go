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

// agent_interface_test.go proves the whole agent-facing CLI surface
// documented in the repo-root AGENTS.md, driven end-to-end through the
// actual compiled dispatch path (newRootCmd/SetArgs) rather than isolated
// per-function units: test-list case 1 (--format pptx produces a REAL,
// valid OOXML zip package), case 2 (--format json's full envelope
// structure, re-proven one level up from format_test.go's own per-function
// coverage, now via the actual pptx SIBLING case wired into the SAME
// emitFormat switch), and case 4 (both standing import-boundary gates stay
// green). Task 2 (AGENTS.md) adds test-list case 3 (a real subprocess exit
// code) to this same file.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// agentInterfaceTestDeck is the hand-built fixture (no generated/LLM test
// data) test-list cases 1+2 share: front matter, a heading (non-empty
// model.outline), a fenced ```go code block, and a display "$$...$$" math
// block -- exactly the schema-v2 surfaces (Block{Kind:"code"|"math"}) both
// the JSON envelope and the pptx exporter must carry through.
const agentInterfaceTestDeck = "---\n" +
	"marp: true\n" +
	"---\n\n" +
	"# Agent Interface\n\n" +
	"```go\nfmt.Println(\"agent\")\n```\n\n" +
	"$$E=mc^2$$\n"

// TestFormatPPTXValidZip is test-list case 1: `eden-press <deck> -o out.pptx
// --format pptx` writes a REAL OOXML package -- a zip archive/zip can open,
// whose parts include the two that prove it's a genuine PowerPoint package:
// [Content_Types].xml and ppt/presentation.xml. Written to a file (-o) and
// read back from disk -- pptx is binary, never asserted against stdout
// text.
func TestFormatPPTXValidZip(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	inPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(inPath, []byte(agentInterfaceTestDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	outPath := filepath.Join(dir, "out.pptx")

	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	// --output/-o is registered on the explicit "convert" subcommand
	// (registerConvertFlags), never on root's bare default action -- see
	// integration_test.go's "file to -o" case for the same pattern.
	root.SetArgs([]string{"convert", inPath, "--output", outPath, "--format", "pptx"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, errBuf.String())
	}
	if out.String() != "" {
		t.Errorf("stdout = %q, want empty (pptx bytes must go to --output, never stdout)", out.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", outPath, err)
	}
	if len(data) == 0 {
		t.Fatal("pptx output file is empty")
	}
	if !bytes.HasPrefix(data, []byte("PK")) {
		t.Errorf("pptx output does not start with the zip magic \"PK\": %x", data[:2])
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v (not a valid zip/OOXML package)", err)
	}

	names := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"[Content_Types].xml", "ppt/presentation.xml"} {
		if !names[want] {
			t.Errorf("pptx package missing required OPC part %q; got: %v", want, names)
		}
	}
}

// TestFormatPPTXStdout is a companion happy-path check: with NO -o, pptx
// bytes still go to stdout (writeOutputBytes' documented default pairing)
// and are themselves a valid zip -- proving writePPTX reuses
// writeOutputBytes' stdout leg, not just the file leg.
func TestFormatPPTXStdout(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	inPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(inPath, []byte(agentInterfaceTestDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs([]string{inPath, "--format", "pptx"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, errBuf.String())
	}

	data := out.Bytes()
	if len(data) == 0 {
		t.Fatal("stdout pptx output is empty")
	}
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("zip.NewReader(stdout pptx): %v", err)
	}
}

// TestFormatJSONEndToEndAgentDeck is test-list case 2: re-asserts 04.1-01's
// JSON envelope contract one level up, through THIS TRD's own
// front-matter-bearing fixture and the actual compiled --format pptx
// sibling case wired into the SAME emitFormat switch -- html/json/pptx are
// all now exercised from the identical binary path.
func TestFormatJSONEndToEndAgentDeck(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(agentInterfaceTestDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs([]string{path, "--format", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, errBuf.String())
	}

	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal envelope: %v\nraw: %s", err, out.String())
	}
	for _, key := range []string{"html", "css", "model", "comments", "meta"} {
		if _, ok := envelope[key]; !ok {
			t.Errorf("envelope missing top-level key %q: %v", key, envelope)
		}
	}

	model, ok := envelope["model"].(map[string]any)
	if !ok {
		t.Fatalf("envelope[model] is not an object: %T", envelope["model"])
	}
	sections, _ := model["sections"].([]any)
	outline, _ := model["outline"].([]any)
	if len(outline) == 0 {
		t.Error("model.outline empty, want the deck's heading entry")
	}

	var sawCode, sawMath bool
	for _, s := range sections {
		sec, _ := s.(map[string]any)
		blocks, _ := sec["blocks"].([]any)
		for _, b := range blocks {
			block, _ := b.(map[string]any)
			switch block["kind"] {
			case "code":
				sawCode = true
				if block["language"] != "go" {
					t.Errorf("code block language = %v, want %q", block["language"], "go")
				}
				text, _ := block["text"].(string)
				if !strings.Contains(text, "fmt.Println") {
					t.Errorf("code block text = %q, want raw source", text)
				}
				if _, ok := block["source"]; ok {
					t.Errorf("code block has a forbidden `source` key: %v", block)
				}
			case "math":
				sawMath = true
				if disp, _ := block["display"].(bool); !disp {
					t.Errorf("math block display = %v, want true", block["display"])
				}
				text, _ := block["text"].(string)
				if !strings.Contains(text, "E=mc^2") {
					t.Errorf("math block text = %q, want raw TeX", text)
				}
				if _, ok := block["tex"]; ok {
					t.Errorf("math block has a forbidden `tex` key: %v", block)
				}
			}
		}
	}
	if !sawCode {
		t.Error("no code block found end-to-end")
	}
	if !sawMath {
		t.Error("no math block found end-to-end")
	}
}

// TestGateEnforcement is test-list case 4: both standing import-boundary
// gates stay green from this test binary's own point of view -- the
// no-chromedp gate now scans ./cmd/... too, and check-cli-imports.sh is
// unchanged. Run as scripts, mirroring exactly how CI/make invoke them, so
// this test fails loudly if a future change reintroduces chromedp (or a
// direct chase/profiles import) into the CLI.
func TestGateEnforcement(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	repoRoot := filepath.Join(wd, "..", "..")

	for _, script := range []string{"check-no-chromedp.sh", "check-cli-imports.sh"} {
		script := script
		t.Run(script, func(t *testing.T) {
			cmd := exec.Command("bash", filepath.Join("scripts", script))
			cmd.Dir = repoRoot
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bash scripts/%s: %v\n%s", script, err, out)
			}
		})
	}
}
