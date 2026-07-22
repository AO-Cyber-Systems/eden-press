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

package math

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// parseMathNodes parses src with the math option and returns every *mathNode in
// document order. It drives the parser directly (Parser().Parse) rather than a
// full Convert, so the $/$$ InlineParser + custom AST node can be exercised in
// isolation from the NodeRenderer (which lands in Task 2).
func parseMathNodes(t *testing.T, opt goldmark.Option, src string) []*mathNode {
	t.Helper()
	md := goldmark.New(opt)
	doc := md.Parser().Parse(text.NewReader([]byte(src)))
	var nodes []*mathNode
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if m, ok := n.(*mathNode); ok {
				nodes = append(nodes, m)
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		t.Fatalf("ast.Walk: %v", err)
	}
	return nodes
}

// TestMathParse drives the bespoke $-trigger InlineParser: inline $…$ and block
// $$…$$ each parse to exactly one mathNode carrying the raw LaTeX and the
// inline/block distinction.
func TestMathParse(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$x^2$`)
		if len(nodes) != 1 {
			t.Fatalf("got %d math nodes, want 1", len(nodes))
		}
		if nodes[0].Raw != "x^2" {
			t.Errorf("Raw = %q, want %q", nodes[0].Raw, "x^2")
		}
		if nodes[0].Block {
			t.Errorf("Block = true, want false for inline $…$")
		}
	})

	t.Run("block", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$$\frac{a}{b}$$`)
		if len(nodes) != 1 {
			t.Fatalf("got %d math nodes, want 1", len(nodes))
		}
		if nodes[0].Raw != `\frac{a}{b}` {
			t.Errorf("Raw = %q, want %q", nodes[0].Raw, `\frac{a}{b}`)
		}
		if !nodes[0].Block {
			t.Errorf("Block = false, want true for block $$…$$")
		}
	})

	t.Run("block_aligned", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$$\begin{aligned}a&=b\\c&=d\end{aligned}$$`)
		if len(nodes) != 1 {
			t.Fatalf("got %d math nodes, want 1", len(nodes))
		}
		if !nodes[0].Block {
			t.Errorf("Block = false, want true")
		}
		if !strings.Contains(nodes[0].Raw, `\begin{aligned}`) {
			t.Errorf("Raw = %q, want it to contain \\begin{aligned}", nodes[0].Raw)
		}
	})

	t.Run("two_inline", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$a$ and $b$`)
		if len(nodes) != 2 {
			t.Fatalf("got %d math nodes, want 2", len(nodes))
		}
		if nodes[0].Raw != "a" || nodes[1].Raw != "b" {
			t.Errorf("Raw = %q,%q want a,b", nodes[0].Raw, nodes[1].Raw)
		}
	})
}

// TestCurrency is the currency mis-trigger guard (error_recovery): a bare `$`
// used as a currency sign must NOT open math. Per the Pandoc inline-math rule
// the opening `$` needs a non-space to its right AND the closing `$` a non-space
// to its left and no digit to its right — `$5 and $10` satisfies none.
func TestCurrency(t *testing.T) {
	cases := []string{
		`price is $5 and $10`,
		`it costs $100`,
		`from $5 to $10 per item`,
	}
	for _, src := range cases {
		nodes := parseMathNodes(t, Option(""), src)
		if len(nodes) != 0 {
			t.Errorf("parse(%q): got %d math nodes, want 0 (currency must stay literal)", src, len(nodes))
		}
	}
}

// TestMathOff asserts the MathMode "off" composition: Option("off") is a no-op
// goldmark.Option, so the $-parser is never registered and `$x^2$` stays literal
// text (zero math nodes). Full press.Options wiring lands in 03-09.
func TestMathOff(t *testing.T) {
	nodes := parseMathNodes(t, Option("off"), `$x^2$`)
	if len(nodes) != 0 {
		t.Errorf("MathMode off: got %d math nodes, want 0 (math disabled)", len(nodes))
	}
}

// TestMathML exercises the native-MathML render path (latex2mathml). Simple
// constructs must produce a well-formed <math> element with the right display
// mode. Per error_recovery, BASELINE only asserts well-formedness of simple
// cases — the 8 known latex2mathml converter bugs are Objective 8's fix.
func TestMathML(t *testing.T) {
	inline := renderMathML("x^2", false)
	if !strings.HasPrefix(inline, "<math") || !strings.Contains(inline, "</math>") {
		t.Errorf("renderMathML inline: not a well-formed <math> element: %q", inline)
	}
	if !strings.Contains(inline, `display="inline"`) {
		t.Errorf("renderMathML inline: missing display=\"inline\": %q", inline)
	}
	if !strings.Contains(inline, "<msup>") {
		t.Errorf("renderMathML(x^2): expected an <msup> superscript: %q", inline)
	}

	block := renderMathML(`\frac{a}{b}`, true)
	if !strings.Contains(block, `display="block"`) {
		t.Errorf("renderMathML block: missing display=\"block\": %q", block)
	}
	if !strings.Contains(block, "<mfrac>") {
		t.Errorf("renderMathML(\\frac): expected an <mfrac>: %q", block)
	}
}

// TestBigOperatorStacking is Objective-8 spike cases 1-2 (PROPOSAL §11): big
// n-ary operators and the \lim-family, when carrying limits in DISPLAY mode,
// must render their scripts STACKED — a <munderover> (sub+super) / <munder>
// (sub only) wrapping the operator — NOT the side-by-side <msubsup>/<msub> the
// unpatched vendored converter emitted. Open Q1 (movablelimits vs tag-switch)
// resolved empirically to the munderover TAG-SWITCH: the operator renders as an
// <mi>/<mo> and Chromium MathML-Core only stacks via munder*/mover*, never by
// repositioning msubsup on a movablelimits attribute (see 08-02-SUMMARY). The
// assertion is STRUCTURAL (element shape of the emitted <math>) — no MathJax-SVG
// oracle (marp-math is permanently blocked).
func TestBigOperatorStacking(t *testing.T) {
	t.Run("sum", func(t *testing.T) {
		got := renderMathML(`\sum_{i=1}^{n}`, true)
		if !strings.Contains(got, "<munderover>") {
			t.Errorf("display \\sum_{i=1}^{n}: expected a stacked <munderover>: %q", got)
		}
		if strings.Contains(got, "<msubsup>") {
			t.Errorf("display \\sum_{i=1}^{n}: must NOT emit side-by-side <msubsup>: %q", got)
		}
	})
	t.Run("prod", func(t *testing.T) {
		got := renderMathML(`\prod_{i=1}^{n}`, true)
		if !strings.Contains(got, "<munderover>") {
			t.Errorf("display \\prod_{i=1}^{n}: expected a stacked <munderover>: %q", got)
		}
		if strings.Contains(got, "<msubsup>") {
			t.Errorf("display \\prod_{i=1}^{n}: must NOT emit side-by-side <msubsup>: %q", got)
		}
	})
	t.Run("lim", func(t *testing.T) {
		got := renderMathML(`\lim_{x \to 0}`, true)
		if !strings.Contains(got, "<munder>") {
			t.Errorf("display \\lim_{x \\to 0}: expected a stacked <munder>: %q", got)
		}
		if strings.Contains(got, "<msub>") {
			t.Errorf("display \\lim_{x \\to 0}: must NOT emit side-by-side <msub>: %q", got)
		}
	})
}

// TestFallback exercises the PNG-only fallback render path (go-latex/latex).
// A construct go-latex CAN raster (\frac) proves the real base64 PNG data-URI
// path; a construct it CANNOT (\begin{aligned}, which panics inside mtex)
// proves the graceful documented stub — an alt-only <img>, never a crash and
// never a silent drop. Both are PNG-only: drawtex has no SVG canvas.
func TestFallback(t *testing.T) {
	// Real raster path (called directly; in production \frac routes to MathML).
	img := renderFallbackIMG(`\frac{a}{b}`, false)
	if !strings.Contains(img, "<img") || !strings.Contains(img, `class="math-fallback"`) {
		t.Errorf("renderFallbackIMG: not a math-fallback <img>: %q", img)
	}
	if !strings.Contains(img, "data:image/png;base64,") {
		t.Errorf("renderFallbackIMG(\\frac): expected a base64 PNG data-URI (real raster path): %q", img)
	}

	// Graceful stub path: go-latex panics on \begin{aligned}; must degrade to
	// an alt-only <img>, not crash.
	stub := renderFallbackIMG(`\begin{aligned}a&=b\\c&=d\end{aligned}`, true)
	if !strings.Contains(stub, "<img") || !strings.Contains(stub, "math-fallback") {
		t.Errorf("renderFallbackIMG(aligned): expected a graceful math-fallback <img> stub: %q", stub)
	}
	if !strings.Contains(stub, "alt=") {
		t.Errorf("renderFallbackIMG(aligned): stub must carry alt text of the raw LaTeX: %q", stub)
	}
}

// renderHTML runs the full goldmark pipeline (parse + the wired NodeRenderer).
func renderHTML(t *testing.T, opt goldmark.Option, src string) string {
	t.Helper()
	md := goldmark.New(opt)
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		t.Fatalf("Convert(%q): %v", src, err)
	}
	return buf.String()
}

// TestMathRender is the end-to-end routing assertion: with the battery wired,
// `$x^2$` renders native MathML and `$$\begin{aligned}…$$` routes to the PNG
// fallback <img> — the routing decision made on the RAW source by needsFallback
// BEFORE any conversion.
func TestMathRender(t *testing.T) {
	mathml := renderHTML(t, Option(""), `$x^2$`)
	if !strings.Contains(mathml, "<math") {
		t.Errorf("`$x^2$` should render native MathML: %q", mathml)
	}
	if strings.Contains(mathml, "<img") {
		t.Errorf("`$x^2$` should NOT route to the fallback <img>: %q", mathml)
	}

	fallback := renderHTML(t, Option(""), `$$\begin{aligned}a&=b\\c&=d\end{aligned}$$`)
	if !strings.Contains(fallback, "<img") || !strings.Contains(fallback, "math-fallback") {
		t.Errorf("heavy `\\begin{aligned}` should route to the fallback <img>: %q", fallback)
	}
	if strings.Contains(fallback, "<math") {
		t.Errorf("heavy `\\begin{aligned}` should NOT emit MathML: %q", fallback)
	}
}

// TestMathRawDisplayAccessors covers Test-list case 8 (the 06-01 additive
// getter seam): MathRaw()/MathDisplay() return the mathNode's Raw/Block fields
// unchanged, and a *mathNode satisfies the exact duck-typed interface
// chase/model reaches raw TeX through -- WITHOUT chase/model ever importing
// press/math. This test IS that interface contract: if these getters or their
// signatures drift, chase/model's math-block extraction silently stops
// compiling against the seam, so this test pins the shape DART-04 depends on.
func TestMathRawDisplayAccessors(t *testing.T) {
	display := newMathNode(`E=mc^2`, true)
	if display.MathRaw() != `E=mc^2` {
		t.Errorf("MathRaw() = %q, want %q", display.MathRaw(), `E=mc^2`)
	}
	if !display.MathDisplay() {
		t.Errorf("MathDisplay() = false, want true for $$…$$ (block) math")
	}

	inline := newMathNode(`x`, false)
	if inline.MathRaw() != "x" {
		t.Errorf("MathRaw() = %q, want %q", inline.MathRaw(), "x")
	}
	if inline.MathDisplay() {
		t.Errorf("MathDisplay() = true, want false for $…$ (inline) math")
	}

	// The duck-typed seam: an *ast.Node value type-asserts to the exact
	// interface chase/model declares locally (as `rawMath`). This proves
	// chase/model can recover raw TeX + display flag from a math node reached
	// only as an ast.Node, with no press/math import.
	var node ast.Node = display
	rm, ok := node.(interface {
		MathRaw() string
		MathDisplay() bool
	})
	if !ok {
		t.Fatal("*mathNode does not satisfy the rawMath duck-typed interface (MathRaw/MathDisplay)")
	}
	if rm.MathRaw() != `E=mc^2` || !rm.MathDisplay() {
		t.Errorf("via interface: MathRaw()=%q MathDisplay()=%v, want %q true", rm.MathRaw(), rm.MathDisplay(), `E=mc^2`)
	}
}
