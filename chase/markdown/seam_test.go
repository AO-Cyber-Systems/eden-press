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

package markdown

import (
	"bytes"
	"testing"
)

// TestSeamCriterion1ASTAndContextInspectableBetweenPhases covers 01-08's
// Test-list case 1: Parse (seam.go) returns a finalized *ast.Document --
// Section nodes AND their directive-resolved Attrs already materialized --
// plus its parser.Context, BOTH inspectable BEFORE Render ever runs; then
// calling Render on the same markdown produces exactly the same HTML a
// caller gets by driving the two phases manually off the SAME engine,
// proving Render is the real two-phase seam and not a disguised
// md.Convert() call.
func TestSeamCriterion1ASTAndContextInspectableBetweenPhases(t *testing.T) {
	md := "<!-- _class: lead -->\n\n# Slide 1\n\nfirst\n\n---\n\n# Slide 2\n\nsecond\n"

	doc, pc := Parse(md)
	if pc == nil {
		t.Fatalf("Parse returned a nil parser.Context")
	}
	if got, want := countChildrenOfKind(doc, KindSection), 2; got != want {
		t.Fatalf("between phases: got %d Section children, want %d", got, want)
	}

	first, ok := doc.FirstChild().(*Section)
	if !ok {
		t.Fatalf("doc.FirstChild() is not *Section, got %T", doc.FirstChild())
	}
	foundClass := false
	for _, a := range first.Attrs {
		if a.Name == "class" && a.Value == "lead" {
			foundClass = true
		}
	}
	if !foundClass {
		t.Fatalf("first slide's directive-resolved Attrs missing class=lead BEFORE render, got %#v", first.Attrs)
	}

	var buf bytes.Buffer
	if err := defaultEngine.Renderer().Render(&buf, []byte(md), doc); err != nil {
		t.Fatalf("manual Render() error: %v", err)
	}
	manual := buf.String()

	out, err := Render(md, nil)
	if err != nil {
		t.Fatalf("seam Render() error: %v", err)
	}
	if out != manual {
		t.Fatalf("seam.Render() output != manual two-phase render:\nseam:\n%s\nmanual:\n%s", out, manual)
	}
}

// TestRenderFuncAdapterMatchesRender covers the RenderFunc adapter: it must
// produce output identical to calling Render directly, with the exact
// (markdown string, opts map[string]any) (string, error) shape
// conformance/runner.RenderFunc requires.
func TestRenderFuncAdapterMatchesRender(t *testing.T) {
	md := "# Hello\n"

	rf := RenderFunc()
	out, err := rf(md, map[string]any{"requires_engine": "marp-core"})
	if err != nil {
		t.Fatalf("RenderFunc()(): %v", err)
	}
	want, err := Render(md, nil)
	if err != nil {
		t.Fatalf("Render(): %v", err)
	}
	if out != want {
		t.Fatalf("RenderFunc adapter output = %q, want %q", out, want)
	}
}

// TestNewEngineRoundTripsGFMTable covers NewEngine's extension surface: the
// chase engine must ALSO carry GFM (tables etc.), not just chase/markdown's
// own Extender -- verified via a GFM table producing a real <table>, not
// escaped/raw-text fallback.
func TestNewEngineRoundTripsGFMTable(t *testing.T) {
	out, err := Render("| a | b |\n|---|---|\n| 1 | 2 |\n", nil)
	if err != nil {
		t.Fatalf("Render(): %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("<table>")) {
		t.Fatalf("expected a rendered <table>, got:\n%s", out)
	}
}
