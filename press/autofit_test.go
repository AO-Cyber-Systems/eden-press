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

package press

import (
	"bytes"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

// renderWithAutofit is the shared TDD harness for this file: build a
// markdown.NewEngine carrying autofitOption() (exactly the composition
// 03-09/press.Render folds it into), drive it through the two-phase
// Parse/Render seam (never md.Convert()), and return the rendered HTML.
func renderWithAutofit(t *testing.T, md string) string {
	t.Helper()
	engine := markdown.NewEngine(autofitOption())
	doc, _ := markdown.ParseWithEngine(md, engine)

	var buf bytes.Buffer
	if err := engine.Renderer().Render(&buf, []byte(md), doc); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// TestAutofitFitHeaderMarker covers Test-list case 4 (CORE-09): a fitting
// header "# <!--fit-->" renders an <h1> carrying the stable fit marker
// (data-auto-scaling="fit"), with the <!--fit--> marker comment consumed
// (never leaked into the rendered HTML as literal text).
func TestAutofitFitHeaderMarker(t *testing.T) {
	html := renderWithAutofit(t, "# <!--fit-->\n")

	if !strings.Contains(html, `data-auto-scaling="fit"`) {
		t.Fatalf("expected the fit marker attribute on the heading, got %q", html)
	}
	if strings.Contains(html, "<!--fit-->") {
		t.Fatalf("expected the <!--fit--> marker comment to be consumed, got %q", html)
	}
	if !strings.Contains(html, "<h1") {
		t.Fatalf("expected an <h1> element, got %q", html)
	}
}

// TestAutofitNormalHeadingCarriesNoFitMarker covers the negative half of
// Test-list case 5: a normal heading with no <!--fit--> marker carries no
// fit marker at all.
func TestAutofitNormalHeadingCarriesNoFitMarker(t *testing.T) {
	html := renderWithAutofit(t, "# Just a normal heading\n")

	if strings.Contains(html, "data-auto-scaling") {
		t.Fatalf("expected no fit marker on a normal heading, got %q", html)
	}
}

// TestAutofitShrinkMarkersOnCodeAndMathBlocks covers Test-list case 5
// (CORE-09): a fenced code block and a "$$...$$" math block both carry the
// stable shrink marker (wrapped in the marp-fit-shrink class) -- MARKERS
// only, never a runtime-JS layout pass.
func TestAutofitShrinkMarkersOnCodeAndMathBlocks(t *testing.T) {
	md := "```go\nfmt.Println(1)\n```\n\n$$E=mc^2$$\n"
	html := renderWithAutofit(t, md)

	if got := strings.Count(html, `class="`+autofitShrinkClass+`"`); got != 2 {
		t.Fatalf("expected exactly 2 shrink-marked blocks (code + math), got %d in %q", got, html)
	}
	if !strings.Contains(html, "<pre><code") {
		t.Fatalf("expected the fenced code block's own <pre><code> to still render, got %q", html)
	}
	if !strings.Contains(html, "E=mc^2") {
		t.Fatalf("expected the math block's own text to still render, got %q", html)
	}
}

// TestAutofitOptionIsComposableGoldmarkOption confirms autofitOption()
// composes with other goldmark options/extensions without panicking or
// erroring -- the shape 03-09's NewEngine(pressExtraOpts...) fold-in
// requires (a plain, composable goldmark.Option, not a bespoke type).
func TestAutofitOptionIsComposableGoldmarkOption(t *testing.T) {
	engine := markdown.NewEngine(autofitOption())
	doc, _ := markdown.ParseWithEngine("# <!--fit-->\n\nSome text.\n", engine)

	var buf bytes.Buffer
	if err := engine.Renderer().Render(&buf, []byte("# <!--fit-->\n\nSome text.\n"), doc); err != nil {
		t.Fatalf("render with composed autofitOption(): %v", err)
	}
}
