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
	"encoding/xml"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// xmlElem is a minimal recursive MathML element used for STRUCTURAL DOM
// assertions: parse the emitted <math> string and inspect element shape/child
// order rather than substring-matching. `,any` captures every child element;
// `,chardata` captures the immediate text (numeric char refs like &#x0005B; are
// decoded to their runes by encoding/xml); `,any,attr` captures every attribute
// so fence-sizing (minsize/maxsize) and cell alignment (columnalign) can be
// asserted structurally rather than by substring.
type xmlElem struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []xmlElem  `xml:",any"`
	Chardata string     `xml:",chardata"`
}

// attr returns the value of the named attribute and whether it was present.
func (e xmlElem) attr(name string) (string, bool) {
	for _, a := range e.Attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// findElem returns the first descendant (depth-first, self included) whose local
// element name matches local.
func findElem(e xmlElem, local string) (xmlElem, bool) {
	if e.XMLName.Local == local {
		return e, true
	}
	for _, c := range e.Children {
		if found, ok := findElem(c, local); ok {
			return found, true
		}
	}
	return xmlElem{}, false
}

// findAll returns every descendant (depth-first, self included) whose local
// element name matches local, in document order.
func findAll(e xmlElem, local string) []xmlElem {
	var out []xmlElem
	if e.XMLName.Local == local {
		out = append(out, e)
	}
	for _, c := range e.Children {
		out = append(out, findAll(c, local)...)
	}
	return out
}

// parseMathML unmarshals an emitted <math> string into the xmlElem tree, failing
// the test on malformed XML.
func parseMathML(t *testing.T, got string) xmlElem {
	t.Helper()
	var root xmlElem
	if err := xml.Unmarshal([]byte(got), &root); err != nil {
		t.Fatalf("parse emitted MathML: %v\n%q", err, got)
	}
	return root
}

// fenceParens returns, in document order, every <mo> whose decoded text is a
// round paren "(" or ")". The converter emits fences as <mo> with a numeric
// char ref (&#x00028;/&#x00029;) that encoding/xml decodes back to the rune, so
// a matched stretchy fence pair is exactly ["(", ")"].
func fenceParens(root xmlElem) []xmlElem {
	var out []xmlElem
	for _, mo := range findAll(root, "mo") {
		switch strings.TrimSpace(mo.Chardata) {
		case "(", ")":
			out = append(out, mo)
		}
	}
	return out
}

// flatText concatenates the element's own text with all descendant text.
func flatText(e xmlElem) string {
	s := e.Chardata
	for _, c := range e.Children {
		s += flatText(c)
	}
	return s
}

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

// assertSizedFencePair asserts that mos is EXACTLY an opening "(" followed by a
// CLOSING ")", each carrying BOTH minsize and maxsize (the content-height stretchy
// sizing). This is the criterion-1 shape for \binom and pmatrix: the unpatched
// converter emitted the opening "(" as the closing fence too (<mo>(…<mo>( — two
// opening parens), and pmatrix's fence carried no sizing at all.
func assertSizedFencePair(t *testing.T, label string, mos []xmlElem, got string) {
	t.Helper()
	if len(mos) != 2 {
		t.Fatalf("%s: got %d paren <mo> fences, want EXACTLY 2 (one '(' + one ')'): %q", label, len(mos), got)
	}
	if strings.TrimSpace(mos[0].Chardata) != "(" {
		t.Errorf("%s: opening fence = %q, want '(': %q", label, mos[0].Chardata, got)
	}
	if strings.TrimSpace(mos[1].Chardata) != ")" {
		t.Errorf("%s: CLOSING fence = %q, want ')' (unpatched bug reused '(' as the close): %q", label, mos[1].Chardata, got)
	}
	for i, mo := range mos {
		if _, ok := mo.attr("minsize"); !ok {
			t.Errorf("%s: fence[%d] (%q) missing minsize (content-height sizing): %q", label, i, mo.Chardata, got)
		}
		if _, ok := mo.attr("maxsize"); !ok {
			t.Errorf("%s: fence[%d] (%q) missing maxsize (content-height sizing): %q", label, i, mo.Chardata, got)
		}
	}
}

// TestBinomFence is Objective-8 spike case 5 (PROPOSAL §11): \binom must emit a
// MATCHED, content-sized stretchy fence pair — an opening <mo>( and a distinct
// CLOSING <mo>), both carrying minsize/maxsize. The unpatched converter reused
// the opening '(' char for the closing fence (appendPostfixElement passed
// `\lparen` instead of `\rparen`), yielding <mo>(…<mo>( . Asserted STRUCTURALLY.
func TestBinomFence(t *testing.T) {
	got := renderMathML(`\binom{n}{k}`, true)
	root := parseMathML(t, got)
	if _, ok := findElem(root, "mfrac"); !ok {
		t.Fatalf(`\binom{n}{k}: expected the <mfrac> binomial body: %q`, got)
	}
	assertSizedFencePair(t, `\binom{n}{k}`, fenceParens(root), got)
}

// TestPmatrixFence is Objective-8 spike case 6 (PROPOSAL §11): pmatrix (both the
// \begin{pmatrix}…\end{pmatrix} environment and the \pmatrix{…} shorthand) must
// emit a matched, content-sized stretchy fence pair around its <mtable> — the
// same fix as \binom (the unpatched pmatrix branch passed an EMPTY attribute map,
// so its fence carried no sizing AND reused '(' as the close). The both-in-one
// sub-case guards research Open Q2: a \binom and a \pmatrix in the SAME expression
// each render their own correct fence, no cross-contamination.
func TestPmatrixFence(t *testing.T) {
	t.Run("environment", func(t *testing.T) {
		got := renderMathML(`\begin{pmatrix}1&0\\0&1\end{pmatrix}`, true)
		root := parseMathML(t, got)
		if _, ok := findElem(root, "mtable"); !ok {
			t.Fatalf(`pmatrix: expected an <mtable> body: %q`, got)
		}
		assertSizedFencePair(t, `\begin{pmatrix}`, fenceParens(root), got)
	})

	t.Run("shorthand", func(t *testing.T) {
		got := renderMathML(`\pmatrix{1&0\\0&1}`, true)
		root := parseMathML(t, got)
		assertSizedFencePair(t, `\pmatrix{…}`, fenceParens(root), got)
	})

	t.Run("both_in_one_expression", func(t *testing.T) {
		// research Open Q2: shared convertAndAppendCommand fence path must serve
		// BOTH branches independently — 4 fences in order ( ) ( ) , each sized.
		got := renderMathML(`\binom{n}{k}\begin{pmatrix}1&0\\0&1\end{pmatrix}`, true)
		root := parseMathML(t, got)
		mos := fenceParens(root)
		if len(mos) != 4 {
			t.Fatalf(`\binom+\pmatrix: got %d paren fences, want 4 (binom '(' ')' then pmatrix '(' ')'): %q`, len(mos), got)
		}
		wantSeq := []string{"(", ")", "(", ")"}
		for i, mo := range mos {
			if strings.TrimSpace(mo.Chardata) != wantSeq[i] {
				t.Errorf("both: fence[%d] = %q, want %q (cross-contamination or reused-open bug): %q", i, mo.Chardata, wantSeq[i], got)
			}
			if _, ok := mo.attr("minsize"); !ok {
				t.Errorf("both: fence[%d] (%q) missing minsize: %q", i, mo.Chardata, got)
			}
			if _, ok := mo.attr("maxsize"); !ok {
				t.Errorf("both: fence[%d] (%q) missing maxsize: %q", i, mo.Chardata, got)
			}
		}
	})
}

// moTexts returns the decoded text of every <mo> in the tree, in document order.
func moTexts(root xmlElem) []string {
	var out []string
	for _, mo := range findAll(root, "mo") {
		out = append(out, strings.TrimSpace(mo.Chardata))
	}
	return out
}

// TestMatrixFenceRegression guards the anti-pattern: the pmatrix/binom fence fix
// must NOT regress \begin{bmatrix} (square brackets) or bare \begin{matrix} (no
// fence) — both already KaTeX-quality per PROPOSAL §11. Fences are emitted as
// numeric char refs, so the assertion is on the PARSED/decoded <mo> text.
func TestMatrixFenceRegression(t *testing.T) {
	// bmatrix keeps its matched square-bracket fence [ … ] and no round parens.
	bm := renderMathML(`\begin{bmatrix}1&0\\0&1\end{bmatrix}`, true)
	bmRoot := parseMathML(t, bm)
	fences := moTexts(bmRoot)
	if len(fences) != 2 || fences[0] != "[" || fences[1] != "]" {
		t.Errorf(`\begin{bmatrix}: expected matched square-bracket fence [ ], got %v: %q`, fences, bm)
	}
	if len(fenceParens(bmRoot)) != 0 {
		t.Errorf(`\begin{bmatrix}: must NOT emit round parens (pmatrix fence leaked in): %q`, bm)
	}
	// bare matrix has NO fence at all.
	m := renderMathML(`\begin{matrix}1&0\\0&1\end{matrix}`, true)
	mRoot := parseMathML(t, m)
	if got := len(moTexts(mRoot)); got != 0 {
		t.Errorf(`\begin{matrix}: expected NO fence <mo>, got %d: %q`, got, m)
	}
}

// mtdAligns returns the columnalign value of every <mtd> in the tree that
// carries one, in document order.
func mtdAligns(root xmlElem) []string {
	var out []string
	for _, mtd := range findAll(root, "mtd") {
		if v, ok := mtd.attr("columnalign"); ok {
			out = append(out, v)
		}
	}
	return out
}

// TestAlignedTable is Objective-8 spike case 7 (PROPOSAL §11): \begin{aligned}
// must render as an <mtable> with the alignment column split (per-<mtd>
// columnalign "right"/"left", mirroring the working \align* / ALIGN path), NOT
// the unpatched literal <mi>&</mi> + <mspace linebreak> fallthrough. Asserted
// STRUCTURALLY on the parsed tree.
func TestAlignedTable(t *testing.T) {
	got := renderMathML(`\begin{aligned}a&=b\\c&=d\end{aligned}`, true)
	root := parseMathML(t, got)

	mtable, ok := findElem(root, "mtable")
	if !ok {
		t.Fatalf(`aligned: expected an <mtable> (got the unpatched literal fallthrough): %q`, got)
	}
	if rows := findAll(mtable, "mtr"); len(rows) != 2 {
		t.Errorf(`aligned: <mtable> has %d <mtr> rows, want 2 (a&=b // c&=d): %q`, len(rows), got)
	}

	// The alignment split: at least one right-aligned and one left-aligned cell.
	aligns := mtdAligns(root)
	var hasRight, hasLeft bool
	for _, a := range aligns {
		hasRight = hasRight || a == "right"
		hasLeft = hasLeft || a == "left"
	}
	if !hasRight || !hasLeft {
		t.Errorf(`aligned: expected the "right"/"left" columnalign split, got %v: %q`, aligns, got)
	}

	// The unpatched fallthrough emitted the '&' column-break as a literal <mi>&</mi>
	// and each '\\' as an <mspace linebreak> — neither may survive the fix.
	for _, mi := range findAll(root, "mi") {
		if strings.TrimSpace(mi.Chardata) == "&" {
			t.Errorf(`aligned: literal <mi>&</mi> survived (unpatched bug): %q`, got)
		}
	}
	for _, sp := range findAll(root, "mspace") {
		if _, ok := sp.attr("linebreak"); ok {
			t.Errorf(`aligned: <mspace linebreak> survived (unpatched bug): %q`, got)
		}
	}
}

// TestMathvariantCodepoint is Objective-8 spike case 8 (PROPOSAL §11):
// \mathbb/\mathbf/\mathcal must emit the actual Unicode Mathematical-Alphanumeric
// CODEPOINT as element text (ℝ U+211D, bold v U+1D42F, script L U+2112) — NOT a
// mathvariant attribute (which MathML Core ignores; the unpatched fork dropped
// the styling entirely, emitting a plain <mi>R</mi>). Asserted by decoding the
// emitted numeric char ref via encoding/xml and checking NO mathvariant attr.
func TestMathvariantCodepoint(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want rune // the Mathematical-Alphanumeric codepoint the letter must become
	}{
		{"mathbb_R", `\mathbb{R}`, 0x211D},   // DOUBLE-STRUCK CAPITAL R (named hole)
		{"mathbf_v", `\mathbf{v}`, 0x1D42F},  // MATHEMATICAL BOLD SMALL V
		{"mathcal_L", `\mathcal{L}`, 0x2112}, // SCRIPT CAPITAL L (named hole)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := renderMathML(c.raw, false)
			if strings.Contains(got, "mathvariant") {
				t.Errorf(`%s: emitted a mathvariant attribute (MathML Core ignores it); want the Unicode codepoint: %q`, c.raw, got)
			}
			root := parseMathML(t, got)
			mi, ok := findElem(root, "mi")
			if !ok {
				t.Fatalf(`%s: expected an <mi>: %q`, c.raw, got)
			}
			gotText := strings.TrimSpace(flatText(mi))
			if gotText != string(c.want) {
				t.Errorf(`%s: <mi> text = %q (%U), want %q (%U): %q`, c.raw, gotText, []rune(gotText), string(c.want), c.want, got)
			}
		})
	}
}

// TestSqrtRootChildOrder is Objective-8 spike case 3 (PROPOSAL §11): the
// \sqrt[n]{radicand} radicand-loss bug. The unpatched walker read the '['
// OPENING_BRACKET itself as the radicand (misassembling it into the <mroot>) and
// leaked the real base out as a sibling. The fix must emit an <mroot> with
// EXACTLY 2 element children in MathML order [radicand, index] — base first.
// This is asserted STRUCTURALLY by parsing the emitted <math> with encoding/xml
// and inspecting <mroot>'s child count and order (not substring-matching).
func TestSqrtRootChildOrder(t *testing.T) {
	got := renderMathML(`\sqrt[3]{x}`, true)
	if !strings.Contains(got, "<mroot>") {
		t.Fatalf(`\sqrt[3]{x}: expected an <mroot>: %q`, got)
	}

	var root xmlElem
	if err := xml.Unmarshal([]byte(got), &root); err != nil {
		t.Fatalf("parse emitted MathML: %v\n%q", err, got)
	}
	mroot, ok := findElem(root, "mroot")
	if !ok {
		t.Fatalf(`\sqrt[3]{x}: no <mroot> in parsed tree: %q`, got)
	}
	if len(mroot.Children) != 2 {
		t.Fatalf(`\sqrt[3]{x}: <mroot> has %d element children, want EXACTLY 2 [radicand, index]: %q`, len(mroot.Children), got)
	}

	// MathML <mroot> order is [base, index]: radicand x first, index 3 second.
	base := strings.TrimSpace(flatText(mroot.Children[0]))
	index := strings.TrimSpace(flatText(mroot.Children[1]))
	if !strings.Contains(base, "x") {
		t.Errorf(`\sqrt[3]{x}: <mroot> first child (radicand) = %q, want to contain "x": %q`, base, got)
	}
	if !strings.Contains(index, "3") {
		t.Errorf(`\sqrt[3]{x}: <mroot> second child (index) = %q, want to contain "3": %q`, index, got)
	}
	// Regression against the unpatched bug: the '[' bracket-marker (U+005B) must
	// NEVER be misassembled as the radicand (which is exactly what leaked the
	// real base out as a sibling of <mroot>).
	if strings.Contains(base, "[") {
		t.Errorf(`\sqrt[3]{x}: radicand leaked the '[' bracket-marker (unpatched bug): base=%q`, base)
	}
	// The radicand 'x' must live INSIDE <mroot>, not as a stray sibling: the only
	// 'x' in the whole tree is the one reachable from <mroot>.
	if strings.Count(flatText(root), "x") != 1 || !strings.Contains(flatText(mroot), "x") {
		t.Errorf(`\sqrt[3]{x}: radicand 'x' must be the <mroot> base, not leaked as a sibling: %q`, got)
	}
}

// TestMathRegressionBaseline is Objective-8 spike case 4: a guard proving the
// two 08-02 converter patches did NOT disturb the already-KaTeX-quality cases.
// The big-op stacking is strictly scoped to (big-operator OR \lim-family) AND a
// script command AND display style — so ordinary scripts, single-arg radicals,
// inline sums, and limitless operators are all untouched.
func TestMathRegressionBaseline(t *testing.T) {
	// Ordinary superscript is unchanged.
	if got := renderMathML(`x^2`, false); !strings.Contains(got, "<msup>") {
		t.Errorf(`x^2: expected <msup>: %q`, got)
	}
	// Fractions are unchanged.
	if got := renderMathML(`\frac{a}{b}`, true); !strings.Contains(got, "<mfrac>") {
		t.Errorf(`\frac{a}{b}: expected <mfrac>: %q`, got)
	}
	// Single-arg root is UNTOUCHED by the \sqrt[n] fix: <msqrt>, never <mroot>.
	if got := renderMathML(`\sqrt{x}`, true); !strings.Contains(got, "<msqrt>") || strings.Contains(got, "<mroot>") {
		t.Errorf(`\sqrt{x}: expected <msqrt> and NOT <mroot>: %q`, got)
	}
	// A limitless display \sum must NOT gain a spurious stacked wrapper.
	if got := renderMathML(`\sum`, true); strings.Contains(got, "<munderover>") || strings.Contains(got, "<munder>") {
		t.Errorf(`\sum (no limits): must NOT emit a spurious munderover/munder: %q`, got)
	}
	// Inline \sum with limits stays side-by-side <msubsup> (stacking is display-only).
	if got := renderMathML(`\sum_{i=1}^{n}`, false); !strings.Contains(got, "<msubsup>") || strings.Contains(got, "<munderover>") {
		t.Errorf(`inline \sum_{i=1}^{n}: expected side-by-side <msubsup>, NOT stacked: %q`, got)
	}
}

// TestSpikeCorpus is the CRITERION-1 completion gate for Objective 8: all EIGHT
// PROPOSAL §11 spike cases — the constructs the vendored converter rendered wrong
// — promoted into a single permanent structural-regression set. Each row asserts
// the case's target MathML-DOM SHAPE (parsed with encoding/xml, never a Marp
// MathJax-SVG byte-diff, which is permanently blocked). Cases 1–4 landed in
// 08-02 (big-op stacking + \sqrt[n]); cases 5–8 land in this TRD (08-03). The
// checker greps for TestSpikeCorpus as the criterion-1 evidence — keep it the
// single source of truth for "the 8 cases render at KaTeX-parity".
func TestSpikeCorpus(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		block  bool
		assert func(t *testing.T, got string, root xmlElem)
	}{
		{
			// 1. \sum with display limits → stacked <munderover> (08-02).
			name: "1_sum_stacked", raw: `\sum_{i=1}^{n}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				if _, ok := findElem(root, "munderover"); !ok {
					t.Errorf("case 1 \\sum: expected stacked <munderover>: %q", got)
				}
				if _, ok := findElem(root, "msubsup"); ok {
					t.Errorf("case 1 \\sum: must NOT emit side-by-side <msubsup>: %q", got)
				}
			},
		},
		{
			// 2. \prod with display limits → stacked <munderover> (08-02).
			name: "2_prod_stacked", raw: `\prod_{i=1}^{n}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				if _, ok := findElem(root, "munderover"); !ok {
					t.Errorf("case 2 \\prod: expected stacked <munderover>: %q", got)
				}
				if _, ok := findElem(root, "msubsup"); ok {
					t.Errorf("case 2 \\prod: must NOT emit side-by-side <msubsup>: %q", got)
				}
			},
		},
		{
			// 3. \lim with display subscript → stacked <munder> (08-02).
			name: "3_lim_stacked", raw: `\lim_{x \to 0}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				if _, ok := findElem(root, "munder"); !ok {
					t.Errorf("case 3 \\lim: expected stacked <munder>: %q", got)
				}
				if _, ok := findElem(root, "msub"); ok {
					t.Errorf("case 3 \\lim: must NOT emit side-by-side <msub>: %q", got)
				}
			},
		},
		{
			// 4. \sqrt[3]{x} → <mroot> with EXACTLY [radicand x, index 3] (08-02).
			name: "4_sqrt_index", raw: `\sqrt[3]{x}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				mroot, ok := findElem(root, "mroot")
				if !ok {
					t.Fatalf("case 4 \\sqrt[3]{x}: expected <mroot>: %q", got)
				}
				if len(mroot.Children) != 2 {
					t.Fatalf("case 4 \\sqrt[3]{x}: <mroot> has %d children, want 2 [radicand, index]: %q", len(mroot.Children), got)
				}
				if !strings.Contains(flatText(mroot.Children[0]), "x") {
					t.Errorf("case 4 \\sqrt[3]{x}: radicand (child 0) = %q, want 'x': %q", flatText(mroot.Children[0]), got)
				}
				if !strings.Contains(flatText(mroot.Children[1]), "3") {
					t.Errorf("case 4 \\sqrt[3]{x}: index (child 1) = %q, want '3': %q", flatText(mroot.Children[1]), got)
				}
			},
		},
		{
			// 5. \binom{n}{k} → matched, content-sized stretchy fence ( … ) (08-03).
			name: "5_binom_fence", raw: `\binom{n}{k}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				assertSizedFencePair(t, "case 5 \\binom", fenceParens(root), got)
			},
		},
		{
			// 6. pmatrix → sized fence with the CORRECT closing delimiter ) (08-03).
			name: "6_pmatrix_fence", raw: `\begin{pmatrix}1&0\\0&1\end{pmatrix}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				if _, ok := findElem(root, "mtable"); !ok {
					t.Fatalf("case 6 pmatrix: expected <mtable> body: %q", got)
				}
				assertSizedFencePair(t, "case 6 pmatrix", fenceParens(root), got)
			},
		},
		{
			// 7. aligned → <mtable> with the "right"/"left" column-alignment split (08-03).
			name: "7_aligned_mtable", raw: `\begin{aligned}a&=b\\c&=d\end{aligned}`, block: true,
			assert: func(t *testing.T, got string, root xmlElem) {
				mtable, ok := findElem(root, "mtable")
				if !ok {
					t.Fatalf("case 7 aligned: expected <mtable> (not literal <mi>&</mi>): %q", got)
				}
				if rows := findAll(mtable, "mtr"); len(rows) != 2 {
					t.Errorf("case 7 aligned: %d <mtr> rows, want 2: %q", len(rows), got)
				}
				var hasRight, hasLeft bool
				for _, a := range mtdAligns(root) {
					hasRight = hasRight || a == "right"
					hasLeft = hasLeft || a == "left"
				}
				if !hasRight || !hasLeft {
					t.Errorf("case 7 aligned: expected right/left columnalign split: %q", got)
				}
				for _, mi := range findAll(root, "mi") {
					if strings.TrimSpace(mi.Chardata) == "&" {
						t.Errorf("case 7 aligned: literal <mi>&</mi> survived: %q", got)
					}
				}
			},
		},
		{
			// 8. \mathbb{R}/\mathbf{v}/\mathcal{L} → Unicode codepoints, NO mathvariant (08-03).
			name: "8_mathvariant_codepoint", raw: `\mathbb{R}`, block: false,
			assert: func(t *testing.T, got string, root xmlElem) {
				variants := []struct {
					raw  string
					want rune
				}{
					{`\mathbb{R}`, 0x211D},
					{`\mathbf{v}`, 0x1D42F},
					{`\mathcal{L}`, 0x2112},
				}
				for _, v := range variants {
					out := renderMathML(v.raw, false)
					if strings.Contains(out, "mathvariant") {
						t.Errorf("case 8 %s: emitted a mathvariant attr (Core ignores it): %q", v.raw, out)
					}
					vr := parseMathML(t, out)
					mi, ok := findElem(vr, "mi")
					if !ok {
						t.Fatalf("case 8 %s: expected <mi>: %q", v.raw, out)
					}
					if strings.TrimSpace(flatText(mi)) != string(v.want) {
						t.Errorf("case 8 %s: <mi> = %q, want %q (%U): %q", v.raw, flatText(mi), string(v.want), v.want, out)
					}
				}
			},
		},
	}

	if len(cases) != 8 {
		t.Fatalf("TestSpikeCorpus must lock EXACTLY the 8 PROPOSAL §11 spike cases, have %d", len(cases))
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := renderMathML(c.raw, c.block)
			root := parseMathML(t, got)
			c.assert(t, got, root)
		})
	}
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
