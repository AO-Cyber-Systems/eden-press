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

// goCodeFence is the Test-list's baseline fixture: a KNOWN language (go) so
// chroma's lexer lookup never falls back to a plain/analyse lexer (research
// error_recovery).
const goCodeFence = "```go\nfunc main() {\n\tx := 42\n\tvar s string = \"hi\"\n\t// comment\n\tfmt.Println(x, s)\n}\n```\n"

// TestHighlightClasses covers Test-list case 1: a goldmark engine built with
// highlightOption(""), reused via markdown.NewEngine(highlightOption("")),
// renders a fenced ```go code block to chroma-highlighted markup using CSS
// CLASSES (chromahtml.WithClasses(true)) -- never inline style="..." --
// reusing github.com/yuin/goldmark-highlighting/v2's NewHighlighting rather
// than a hand-rolled fenced-code NodeRenderer.
func TestHighlightClasses(t *testing.T) {
	engine := markdown.NewEngine(highlightOption(""))

	var buf bytes.Buffer
	if err := engine.Convert([]byte(goCodeFence), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, `class="`) {
		t.Fatalf("expected class-based chroma output (class=\"...\"), got none: %s", got)
	}
	if strings.Contains(got, `style="`) {
		t.Fatalf("expected NO inline style=\"...\" (WithClasses(true) must be wired through WithFormatOptions), got: %s", got)
	}
	// chroma's own PreWrapper class ("chroma", chroma.StandardTypes[PreWrapper])
	// is the wrapper goldmark-highlighting's reused formatter always emits in
	// classes mode -- confirms the reused extension is actually engaged, not
	// goldmark's plain default <pre><code> fenced-code renderer.
	if !strings.Contains(got, `class="chroma"`) {
		t.Fatalf(`expected chroma's <pre class="chroma"> wrapper (proves goldmark-highlighting is wired, not goldmark's default renderer), got: %s`, got)
	}
	// A known keyword token (chroma short code "kd" for Go's KeywordDeclaration
	// -- "func"/"var") must appear pre-remap here; the .hljs-* rewrite is a
	// separate post-pass (highlight_remap.go / Test-list case 2), not part of
	// this wiring.
	if !strings.Contains(got, `class="kd"`) {
		t.Fatalf(`expected a raw chroma short class (e.g. class="kd") before the .hljs-* remap post-pass, got: %s`, got)
	}
}

// TestHighlightStyleOmittable covers Test-list case 5: highlightOption
// accepts a custom chroma style name and wires it through without breaking
// the class-based reuse, AND -- the actual point of this test -- the
// highlighting battery is fully OMITTABLE: a caller that never composes
// highlightOption at all (the shape press.Options{NoHighlight: true} takes
// in 03-09's full wiring) gets plain, unhighlighted fenced-code markup back,
// proving the option-based composition is a clean on/off toggle at the
// call site rather than something highlightOption itself must branch on.
func TestHighlightStyleOmittable(t *testing.T) {
	// A valid, non-default chroma style name is accepted and produces
	// classed, chroma-wrapped output exactly like highlightOption("") does --
	// the style name flows through to highlighting.WithStyle without
	// disturbing the classes-not-inline-styles wiring.
	styledEngine := markdown.NewEngine(highlightOption("monokai"))
	var styledBuf bytes.Buffer
	if err := styledEngine.Convert([]byte(goCodeFence), &styledBuf); err != nil {
		t.Fatalf("Convert (custom style): %v", err)
	}
	styled := styledBuf.String()
	if !strings.Contains(styled, `class="chroma"`) {
		t.Fatalf(`highlightOption("monokai") did not produce chroma-wrapped output: %s`, styled)
	}
	if strings.Contains(styled, `style="`) {
		t.Fatalf(`highlightOption("monokai") leaked inline style=, want classes only: %s`, styled)
	}

	// Omitting highlightOption entirely (the NoHighlight=true shape) yields
	// goldmark's plain fenced-code renderer: no chroma wrapper at all.
	bareEngine := markdown.NewEngine()
	var bareBuf bytes.Buffer
	if err := bareEngine.Convert([]byte(goCodeFence), &bareBuf); err != nil {
		t.Fatalf("Convert (no highlight option): %v", err)
	}
	bare := bareBuf.String()
	if strings.Contains(bare, `class="chroma"`) {
		t.Fatalf("expected NO chroma wrapper when highlightOption is omitted (simulates NoHighlight), got: %s", bare)
	}
}
