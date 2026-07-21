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
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// parseDocWithContext runs Phase 1 with an explicit parser.Context, so
// tests can inject HeadingDividerKey before parsing.
func parseDocWithContext(md goldmark.Markdown, src string, pc parser.Context) ast.Node {
	reader := text.NewReader([]byte(src))
	return md.Parser().Parse(reader, parser.WithContext(pc))
}

// countChildrenOfKind counts direct children of doc matching kind.
func countChildrenOfKind(doc ast.Node, kind ast.NodeKind) int {
	n := 0
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == kind {
			n++
		}
	}
	return n
}

// parseDoc runs ONLY Phase 1 (Parser().Parse()) of the two-phase seam --
// never md.Convert() (RESEARCH/anti_patterns: "DO NOT run md.Convert() in
// production paths").
func parseDoc(md goldmark.Markdown, src string) ast.Node {
	reader := text.NewReader([]byte(src))
	return md.Parser().Parse(reader)
}

// findNode returns the first node of the given kind found via a pre-order
// walk, or nil if none exists.
func findNode(doc ast.Node, kind ast.NodeKind) ast.Node {
	var found ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == kind {
			found = n
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

// Test-list case 5: `<!-- foo: bar -->` -> a hidden CommentNode carrying raw
// kv {foo: bar}; not rendered as visible text; not yet recognized as a
// directive.
func TestCommentBlockDetection(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "<!-- foo: bar -->\n")

	n := findNode(doc, KindCommentNode)
	if n == nil {
		t.Fatalf("expected a CommentNode, found none in:\n%v", doc)
	}
	cn, ok := n.(*CommentNode)
	if !ok {
		t.Fatalf("expected *CommentNode, got %T", n)
	}
	if !cn.Hidden {
		t.Fatalf("expected CommentNode.Hidden == true")
	}
	if got, want := cn.KV["foo"], "bar"; got != want {
		t.Fatalf("KV[\"foo\"] = %q, want %q (KV=%v)", got, want, cn.KV)
	}

	// GOTCHA (task 1): assert the produced node KIND is CommentNode, NOT
	// goldmark's own HTMLBlock -- i.e. our parser won the priority race.
	if bad := findNode(doc, ast.KindHTMLBlock); bad != nil {
		t.Fatalf("comment was parsed as ast.HTMLBlock, not CommentNode -- priority race lost")
	}
}

// Test-list case 6: inline comment mid-paragraph (`text <!-- x --> more`) ->
// detected as an inline CommentNode.
func TestCommentInlineDetection(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "text <!-- x --> more\n")

	n := findNode(doc, KindCommentInline)
	if n == nil {
		t.Fatalf("expected a CommentInline node, found none in:\n%v", doc)
	}
	ci, ok := n.(*CommentInline)
	if !ok {
		t.Fatalf("expected *CommentInline, got %T", n)
	}
	if !ci.Hidden {
		t.Fatalf("expected CommentInline.Hidden == true")
	}
	if ci.Raw != "x" {
		t.Fatalf("Raw = %q, want %q", ci.Raw, "x")
	}

	if bad := findNode(doc, ast.KindRawHTML); bad != nil {
		t.Fatalf("inline comment was parsed as ast.RawHTML, not CommentInline -- priority race lost")
	}
}

// Test-list case 1: a deck splits into slides on "---" thematic breaks;
// each slide is a *Section with a 1-based ID.
func TestSlideSplitBasic(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "# A\n\nfirst\n\n---\n\n# B\n\nsecond\n")

	if got, want := countChildrenOfKind(doc, KindSection), 2; got != want {
		t.Fatalf("got %d Section children, want %d (doc=%v)", got, want, doc)
	}

	first, ok := doc.FirstChild().(*Section)
	if !ok {
		t.Fatalf("doc.FirstChild() is not *Section, got %T", doc.FirstChild())
	}
	if first.ID != 1 {
		t.Fatalf("first section ID = %d, want 1", first.ID)
	}
	second, ok := doc.FirstChild().NextSibling().(*Section)
	if !ok {
		t.Fatalf("doc's second child is not *Section, got %T", doc.FirstChild().NextSibling())
	}
	if second.ID != 2 {
		t.Fatalf("second section ID = %d, want 2", second.ID)
	}

	// The thematic break itself must not survive as a doc-level child --
	// it is consumed (dropped) by the slide-splitter.
	if bad := findNode(doc, ast.KindThematicBreak); bad != nil {
		t.Fatalf("a ThematicBreak survived slide-splitting: %v", doc)
	}
}

// Test-list case 2: a deck with no "---" is a single slide.
func TestSlideSplitSingleSlideWhenNoBreaks(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "# A\n\njust one slide\n")

	if got, want := countChildrenOfKind(doc, KindSection), 1; got != want {
		t.Fatalf("got %d Section children, want %d (doc=%v)", got, want, doc)
	}
	sec, ok := doc.FirstChild().(*Section)
	if !ok {
		t.Fatalf("doc.FirstChild() is not *Section, got %T", doc.FirstChild())
	}
	if sec.ID != 1 {
		t.Fatalf("section ID = %d, want 1", sec.ID)
	}
}

// Test-list case 3: the setext-H2 trap. "Intro\n---" is CommonMark setext
// syntax for an H2 heading, NOT a slide break -- goldmark's own
// SetextHeadingParser (priority 100) already consumes it into an
// *ast.Heading before any ASTTransformer runs (priority 100/200), so this
// requires ZERO special-case code in slide.go.
func TestSlideSplitSetextH2TrapIsNotABreak(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "Intro\n---\n\nbody\n")

	if got, want := countChildrenOfKind(doc, KindSection), 1; got != want {
		t.Fatalf("got %d Section children, want %d -- setext-H2 trap mistaken for a slide break (doc=%v)", got, want, doc)
	}

	h := findNode(doc, ast.KindHeading)
	if h == nil {
		t.Fatalf("expected an *ast.Heading (setext H2), found none in:\n%v", doc)
	}
	heading, ok := h.(*ast.Heading)
	if !ok || heading.Level != 2 {
		t.Fatalf("expected a level-2 heading, got %#v", h)
	}
}

// Test-list case 4: headingDivider inserts synthetic breaks before
// qualifying headings, which the slide-splitter then consumes uniformly.
func TestHeadingDividerInsertsSyntheticBreaks(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	pc := parser.NewContext()
	pc.Set(HeadingDividerKey, []int{1})
	doc := parseDocWithContext(md, "# A\n\nfirst\n\n# B\n\nsecond\n", pc)

	// headingDivider:1 -> a break before every level-1 heading EXCEPT the
	// very first document child (guarded in headingdivider.go) -- so "# A"
	// (first child) gets no break, "# B" does: two slides total.
	if got, want := countChildrenOfKind(doc, KindSection), 2; got != want {
		t.Fatalf("got %d Section children, want %d (doc=%v)", got, want, doc)
	}
}

// Test-list case 7 (partial -- Task 2): the two-phase seam's finalized AST
// is inspectable BETWEEN Parse() and Render(): Section nodes must already
// exist on the doc returned by Parse(), before Render() is ever called.
func TestTwoPhaseSeamSectionsExistBeforeRender(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "# A\n\n---\n\n# B\n")

	// This assertion runs BEFORE any call to md.Renderer().Render(),
	// proving the AST is fully split into slides by Parse() alone -- the
	// seam is real, not merely the same effective execution order.
	if doc.FirstChild() == nil || doc.FirstChild().Kind() != KindSection {
		t.Fatalf("expected doc's first child to be *Section immediately after Parse(), got %T", doc.FirstChild())
	}
}

// Test-list case 7 (Task 3): the full two-phase seam, driven exactly as
// callers must -- Parser().Parse() followed, in a SEPARATE step, by
// Renderer().Render() -- proving the rendered HTML wraps sections in a
// ".marpit" container (must_haves: "A deck splits into slides on `---`
// thematic breaks; each slide renders as <section id=\"{index+1}\"> inside
// a <div class=\"marpit\"> container").
func TestTwoPhaseSeamRendersMarpitContainerAndSections(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	source := []byte("# A\n\nfirst\n\n---\n\n# B\n\nsecond\n")

	// Phase 1: Parse.
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	// Inspect the finalized AST BETWEEN phases, before any rendering has
	// happened -- this is the seam the TRD requires to be provable.
	if got, want := countChildrenOfKind(doc, KindSection), 2; got != want {
		t.Fatalf("between phases: got %d Section children, want %d", got, want)
	}

	// Phase 2: Render (separate call, same doc/source).
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()

	if !strings.HasPrefix(out, `<div class="marpit">`) {
		t.Fatalf("output does not open with .marpit container, got:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), `</div>`) {
		t.Fatalf("output does not close .marpit container, got:\n%s", out)
	}
	if !strings.Contains(out, `<section id="1">`) {
		t.Fatalf("output missing <section id=\"1\">, got:\n%s", out)
	}
	if !strings.Contains(out, `<section id="2">`) {
		t.Fatalf("output missing <section id=\"2\">, got:\n%s", out)
	}
	if !strings.Contains(out, "first") || !strings.Contains(out, "second") {
		t.Fatalf("output missing expected slide content, got:\n%s", out)
	}
}

// Test-list: a hidden comment must never leak into rendered HTML output.
// Uses the two-phase seam directly (Parse then Render) -- never
// md.Convert() -- consistent with every other production-shaped call in
// this suite.
func TestCommentNeverLeaksIntoRenderedOutput(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	source := []byte("<!-- foo: bar -->\n\ntext <!-- x --> more\n")

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "foo") || strings.Contains(out, "bar") || strings.Contains(out, "<!--") {
		t.Fatalf("hidden comment leaked into rendered output:\n%s", out)
	}
}
