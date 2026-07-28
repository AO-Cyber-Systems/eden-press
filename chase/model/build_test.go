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

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
	pmath "github.com/AO-Cyber-Systems/eden-press/press/math"
)

// structuralSignature renders a deterministic, order-stable string
// describing doc's node-kind tree shape plus each node's own
// mutation-relevant fields (Section.ID/Attrs, Heading.Level,
// Comment(Node|Inline).Raw). This is the non-mutation invariant test's (Test-
// list case 6) proof mechanism.
//
// ast.Node.Dump (goldmark's own structural dumper) is deliberately NOT used
// for this: ast.DumpHelper renders each node's "extra fields" from a Go map,
// and Go map iteration order is randomized per-process -- calling Dump twice
// in a row on the exact same, never-mutated doc was empirically confirmed to
// sometimes produce two DIFFERENT strings (field order differs), which would
// make a Dump-text-diff a flaky false-positive detector of "mutation",
// unrelated to anything Build actually does. structuralSignature instead
// walks the tree itself and only ever appends fields in a fixed, literal
// order chosen by this function -- never a map range.
func structuralSignature(doc ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			b.WriteByte(')')
			return ast.WalkContinue, nil
		}
		fmt.Fprintf(&b, "(%s", n.Kind().String())
		switch node := n.(type) {
		case *markdown.Section:
			fmt.Fprintf(&b, "#%d", node.ID)
			for _, a := range node.Attrs {
				fmt.Fprintf(&b, "[%s=%s]", a.Name, a.Value)
			}
		case *ast.Heading:
			fmt.Fprintf(&b, "@%d", node.Level)
		case *markdown.CommentNode:
			fmt.Fprintf(&b, "{%s}", node.Raw)
		case *markdown.CommentInline:
			fmt.Fprintf(&b, "{%s}", node.Raw)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// TestBuildThreeSlideDeck covers Test-list case 1: Build on a 3-slide deck
// -> len(Document.Sections) == 3, each Section's ID matches its 1-based
// index.
func TestBuildThreeSlideDeck(t *testing.T) {
	md := "# Slide 1\n\n---\n\n# Slide 2\n\n---\n\n# Slide 3\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 3 {
		t.Fatalf("len(Sections) = %d, want 3 (Sections: %+v)", len(d.Sections), d.Sections)
	}
	for i, sec := range d.Sections {
		if sec.ID != i+1 {
			t.Errorf("Sections[%d].ID = %d, want %d", i, sec.ID, i+1)
		}
	}
}

// TestBuildOutlineHeadingsGroupedBySection covers Test-list case 2: a deck
// with "# H1" / "## H2" headings -> Document.Outline lists each heading
// with correct Level, materialized Text, and AutoHeadingID Slug, in
// document order and grouped under the owning slide.
func TestBuildOutlineHeadingsGroupedBySection(t *testing.T) {
	md := "# H1\n\n## H2\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(d.Sections))
	}

	want := []OutlineEntry{
		{SectionID: 1, Level: 1, Text: "H1", Slug: "h1"},
		{SectionID: 1, Level: 2, Text: "H2", Slug: "h2"},
	}
	if len(d.Outline) != len(want) {
		t.Fatalf("len(Outline) = %d, want %d: %+v", len(d.Outline), len(want), d.Outline)
	}
	for i, w := range want {
		if d.Outline[i] != w {
			t.Errorf("Outline[%d] = %+v, want %+v", i, d.Outline[i], w)
		}
	}
}

// TestBuildNotesVsDirectiveComment covers Test-list case 3: a slide
// containing "<!-- just a presenter note -->" -> that text appears in the
// owning Section's Notes; a recognized directive comment
// ("<!-- paginate: true -->") does NOT appear in Notes.
func TestBuildNotesVsDirectiveComment(t *testing.T) {
	md := "# Slide with note\n\n<!-- just a presenter note -->\n\n<!-- paginate: true -->\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(d.Sections))
	}
	notes := d.Sections[0].Notes
	if len(notes) != 1 || notes[0] != "just a presenter note" {
		t.Fatalf("Notes = %+v, want exactly [%q]", notes, "just a presenter note")
	}
	for _, n := range notes {
		if strings.Contains(n, "paginate") {
			t.Errorf("recognized directive comment leaked into Notes: %q", n)
		}
	}
}

// TestBuildCommentFormSizeMathNotNotes covers 03-07 Task 1 Test-list case 2
// (CORE-02): once chase/directive.CoerceGlobal recognizes "size"/"math" as
// GLOBAL directives, a COMMENT-form "<!-- size: 4:3 -->" / "<!-- math:
// mathml -->" must classify as a recognized directive -- not a presenter
// note -- exactly mirroring TestBuildNotesVsDirectiveComment's existing
// paginate coverage, but for the two Marp-Core (not Marpit) global
// directives this TRD adds. A genuine free-form note in the same slide must
// still land in Notes untouched.
func TestBuildCommentFormSizeMathNotNotes(t *testing.T) {
	md := "# Slide\n\n<!-- size: 4:3 -->\n\n<!-- math: mathml -->\n\n<!-- just a note -->\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(d.Sections))
	}
	notes := d.Sections[0].Notes
	if len(notes) != 1 || notes[0] != "just a note" {
		t.Fatalf("Notes = %+v, want exactly [%q] (comment-form size/math must classify as directives, not notes)", notes, "just a note")
	}
}

// TestBuildMetaFromFrontMatter covers Test-list case 4: a deck with front
// matter (theme: gaia, size: 16:9, class: lead) -> Document.Meta carries
// those resolved values.
func TestBuildMetaFromFrontMatter(t *testing.T) {
	md := "---\ntheme: gaia\nsize: 16:9\nclass: lead\n---\n\n# Slide\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	want := map[string]string{"theme": "gaia", "size": "16:9", "class": "lead"}
	for k, v := range want {
		if got := d.Meta.Directives[k]; got != v {
			t.Errorf("Meta.Directives[%q] = %q, want %q", k, got, v)
		}
	}
}

// TestBuildNonMutation covers Test-list case 6 -- MODEL-01's core proof:
// building the model does NOT change the HTML output. Render the SAME
// already-parsed doc to HTML, call Build against that identical doc, render
// it again -> byte-identical HTML. A deterministic structural signature of
// the tree shape (node kinds + Section.ID/Attrs + Heading.Level +
// Comment.Raw) is also compared before/after, as a second, independent
// non-mutation check.
func TestBuildNonMutation(t *testing.T) {
	md := "# Slide 1\n\nSome text.\n\n<!-- speaker note -->\n\n---\n\n# Slide 2\n"
	source := []byte(md)

	doc, pc := markdown.Parse(md)

	sigBefore := structuralSignature(doc)

	eng := markdown.NewEngine()
	var renderBefore bytes.Buffer
	if err := eng.Renderer().Render(&renderBefore, source, doc); err != nil {
		t.Fatalf("Render (before Build): %v", err)
	}

	_ = Build(doc, source, pc)

	sigAfter := structuralSignature(doc)
	if sigBefore != sigAfter {
		t.Fatalf("AST structural signature changed after Build:\n--- before ---\n%s\n--- after ---\n%s", sigBefore, sigAfter)
	}

	var renderAfter bytes.Buffer
	if err := eng.Renderer().Render(&renderAfter, source, doc); err != nil {
		t.Fatalf("Render (after Build): %v", err)
	}
	if renderBefore.String() != renderAfter.String() {
		t.Fatalf("rendered HTML changed after Build:\n--- before ---\n%s\n--- after ---\n%s", renderBefore.String(), renderAfter.String())
	}

	// Also confirm the package-level Render entrypoint (a fresh Parse+Render)
	// is unaffected by the Build call above having run against a separately
	// parsed doc.
	htmlA, err := markdown.Render(md, nil)
	if err != nil {
		t.Fatalf("Render A: %v", err)
	}
	htmlB, err := markdown.Render(md, nil)
	if err != nil {
		t.Fatalf("Render B: %v", err)
	}
	if htmlA != htmlB {
		t.Fatalf("markdown.Render(md, nil) not stable across calls:\nA: %s\nB: %s", htmlA, htmlB)
	}
}

// blocksOfKind returns every Block of kind k in blocks, preserving order.
func blocksOfKind(blocks []Block, k BlockKind) []Block {
	var out []Block
	for _, b := range blocks {
		if b.Kind == k {
			out = append(out, b)
		}
	}
	return out
}

// TestBuildParagraphBlocks covers Test-list case 3: a section with two
// paragraphs yields two Block{Kind: paragraph} in document order, each carrying
// plain editable text.
func TestBuildParagraphBlocks(t *testing.T) {
	md := "# Title\n\nFirst paragraph.\n\nSecond paragraph.\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(d.Sections))
	}
	paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph)
	if len(paras) != 2 {
		t.Fatalf("paragraph blocks = %d, want 2 (all blocks: %+v)", len(paras), d.Sections[0].Blocks)
	}
	if paras[0].Text != "First paragraph." {
		t.Errorf("paras[0].Text = %q, want %q", paras[0].Text, "First paragraph.")
	}
	if paras[1].Text != "Second paragraph." {
		t.Errorf("paras[1].Text = %q, want %q", paras[1].Text, "Second paragraph.")
	}

	// Document order: the heading Block precedes both paragraph Blocks.
	blocks := d.Sections[0].Blocks
	if len(blocks) < 3 || blocks[0].Kind != BlockHeading || blocks[1].Kind != BlockParagraph || blocks[2].Kind != BlockParagraph {
		t.Errorf("block order = %+v, want [heading, paragraph, paragraph]", blocks)
	}
}

// TestBuildListBlocks covers Test-list case 4: a nested bullet list yields one
// Block{Kind: list, Ordered: false} whose Items carry per-item Text + 0-based
// nesting Level in document order; an ordered list sets Ordered: true.
func TestBuildListBlocks(t *testing.T) {
	t.Run("nested_bullet", func(t *testing.T) {
		md := "# T\n\n- a\n  - b\n- c\n"
		doc, pc := markdown.Parse(md)

		d := Build(doc, []byte(md), pc)

		lists := blocksOfKind(d.Sections[0].Blocks, BlockList)
		if len(lists) != 1 {
			t.Fatalf("list blocks = %d, want 1 (all: %+v)", len(lists), d.Sections[0].Blocks)
		}
		lb := lists[0]
		if lb.Ordered {
			t.Errorf("Ordered = true, want false for a bullet list")
		}
		want := []ListItem{{Text: "a", Level: 0}, {Text: "b", Level: 1}, {Text: "c", Level: 0}}
		if len(lb.Items) != len(want) {
			t.Fatalf("Items = %+v, want %+v", lb.Items, want)
		}
		for i, w := range want {
			if lb.Items[i] != w {
				t.Errorf("Items[%d] = %+v, want %+v", i, lb.Items[i], w)
			}
		}

		// No loose paragraph blocks leaked from the list's item text.
		if paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph); len(paras) != 0 {
			t.Errorf("list item text leaked as %d paragraph block(s): %+v", len(paras), paras)
		}
	})

	t.Run("ordered", func(t *testing.T) {
		md := "# T\n\n1. one\n2. two\n"
		doc, pc := markdown.Parse(md)

		d := Build(doc, []byte(md), pc)

		lists := blocksOfKind(d.Sections[0].Blocks, BlockList)
		if len(lists) != 1 {
			t.Fatalf("list blocks = %d, want 1", len(lists))
		}
		if !lists[0].Ordered {
			t.Errorf("Ordered = false, want true for a numbered list")
		}
		want := []ListItem{{Text: "one"}, {Text: "two"}}
		if len(lists[0].Items) != len(want) {
			t.Fatalf("Items = %+v, want %+v", lists[0].Items, want)
		}
		for i, w := range want {
			if lists[0].Items[i] != w {
				t.Errorf("Items[%d] = %+v, want %+v", i, lists[0].Items[i], w)
			}
		}
	})
}

// TestBuildCodeBlocks covers Test-list case 5: a fenced code block yields
// Block{Kind: code, Language: "go", Text: <RAW source>} -- pre-chroma, with no
// HTML span markup. An indented code block yields a code Block with empty
// Language.
func TestBuildCodeBlocks(t *testing.T) {
	t.Run("fenced_go", func(t *testing.T) {
		md := "# T\n\n```go\nfmt.Println()\n```\n"
		doc, pc := markdown.Parse(md)

		d := Build(doc, []byte(md), pc)

		codes := blocksOfKind(d.Sections[0].Blocks, BlockCode)
		if len(codes) != 1 {
			t.Fatalf("code blocks = %d, want 1 (all: %+v)", len(codes), d.Sections[0].Blocks)
		}
		if codes[0].Language != "go" {
			t.Errorf("Language = %q, want %q", codes[0].Language, "go")
		}
		if codes[0].Text != "fmt.Println()\n" {
			t.Errorf("Text = %q, want %q (RAW source)", codes[0].Text, "fmt.Println()\n")
		}
		if strings.Contains(codes[0].Text, "<span") || strings.Contains(codes[0].Text, "class=") {
			t.Errorf("code Text must be RAW (pre-chroma), got HTML markup: %q", codes[0].Text)
		}
	})

	t.Run("indented_no_language", func(t *testing.T) {
		md := "# T\n\n    indented code\n"
		doc, pc := markdown.Parse(md)

		d := Build(doc, []byte(md), pc)

		codes := blocksOfKind(d.Sections[0].Blocks, BlockCode)
		if len(codes) != 1 {
			t.Fatalf("code blocks = %d, want 1 (all: %+v)", len(codes), d.Sections[0].Blocks)
		}
		if codes[0].Language != "" {
			t.Errorf("Language = %q, want empty for indented code", codes[0].Language)
		}
		if codes[0].Text != "indented code\n" {
			t.Errorf("Text = %q, want %q", codes[0].Text, "indented code\n")
		}
	})
}

// TestBuildHeadingBlocks covers Test-list case 6: an `## H2` inside a section
// produces BOTH the existing Outline entry (unchanged) AND a
// Block{Kind: heading, Level: 2, Text: "..."}.
func TestBuildHeadingBlocks(t *testing.T) {
	md := "# H1\n\n## H2\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	// Outline unchanged (mirrors TestBuildOutlineHeadingsGroupedBySection).
	wantOutline := []OutlineEntry{
		{SectionID: 1, Level: 1, Text: "H1", Slug: "h1"},
		{SectionID: 1, Level: 2, Text: "H2", Slug: "h2"},
	}
	if len(d.Outline) != len(wantOutline) {
		t.Fatalf("len(Outline) = %d, want %d", len(d.Outline), len(wantOutline))
	}
	for i, w := range wantOutline {
		if d.Outline[i] != w {
			t.Errorf("Outline[%d] = %+v, want %+v", i, d.Outline[i], w)
		}
	}

	// Heading Blocks alongside the Outline.
	headings := blocksOfKind(d.Sections[0].Blocks, BlockHeading)
	wantHeadings := []Block{
		{Kind: BlockHeading, Level: 1, Text: "H1"},
		{Kind: BlockHeading, Level: 2, Text: "H2"},
	}
	if len(headings) != len(wantHeadings) {
		t.Fatalf("heading blocks = %+v, want %+v", headings, wantHeadings)
	}
	for i, w := range wantHeadings {
		if !reflect.DeepEqual(headings[i], w) {
			t.Errorf("heading block[%d] = %+v, want %+v", i, headings[i], w)
		}
	}
}

// TestBuildBlocksPreserveExistingOutput is the Task-2 regression: adding block
// extraction leaves the pre-existing Attrs/Notes/Outline output byte-for-byte
// unchanged for a mixed fixture (heading + note + directive + paragraph).
func TestBuildBlocksPreserveExistingOutput(t *testing.T) {
	md := "# Slide\n\nBody text.\n\n<!-- just a presenter note -->\n\n<!-- paginate: true -->\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(d.Sections))
	}
	// Notes unchanged: exactly the one non-directive note.
	notes := d.Sections[0].Notes
	if len(notes) != 1 || notes[0] != "just a presenter note" {
		t.Fatalf("Notes = %+v, want exactly [%q]", notes, "just a presenter note")
	}
	// Outline unchanged: the single H1.
	wantOutline := []OutlineEntry{{SectionID: 1, Level: 1, Text: "Slide", Slug: "slide"}}
	if len(d.Outline) != 1 || d.Outline[0] != wantOutline[0] {
		t.Fatalf("Outline = %+v, want %+v", d.Outline, wantOutline)
	}
	// New Blocks present: a heading block + a paragraph block (order preserved).
	if paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph); len(paras) != 1 || paras[0].Text != "Body text." {
		t.Errorf("paragraph blocks = %+v, want one %q", paras, "Body text.")
	}
	if hs := blocksOfKind(d.Sections[0].Blocks, BlockHeading); len(hs) != 1 || hs[0].Text != "Slide" {
		t.Errorf("heading blocks = %+v, want one %q", hs, "Slide")
	}
}

// TestBuildMathBlocks covers Test-list case 7: driven with a MATH-BATTERY
// engine (mirroring press.Render's markdown.NewEngine(pmath.Option("mathml"))),
// Build extracts each `$…$`/`$$…$$` construct as a Block{Kind: math} carrying
// its RAW TeX (not the rendered MathML) and its display flag -- reached via the
// duck-typed rawMath seam, with chase/model NOT importing press/math in its
// non-test code.
//
// The math node exists ONLY under the battery engine; the default
// markdown.Parse (no math battery) leaves `$$x$$` as literal paragraph text
// (exercised by the other build tests). This is why the battery engine is
// wired explicitly here, via the additive markdown.ParseWithEngine seam.
func TestBuildMathBlocks(t *testing.T) {
	engine := markdown.NewEngine(pmath.Option("mathml"))
	md := "# Math\n\n$$E=mc^2$$\n\n$x$\n"

	doc, pc := markdown.ParseWithEngine(md, engine)
	d := Build(doc, []byte(md), pc)

	if len(d.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(d.Sections))
	}
	maths := blocksOfKind(d.Sections[0].Blocks, BlockMath)
	if len(maths) != 2 {
		t.Fatalf("math blocks = %d, want 2 (all: %+v)", len(maths), d.Sections[0].Blocks)
	}

	// Display math $$E=mc^2$$ -> RAW TeX + Display true.
	if maths[0].Text != "E=mc^2" {
		t.Errorf("display math Text = %q, want %q (RAW TeX)", maths[0].Text, "E=mc^2")
	}
	if !maths[0].Display {
		t.Errorf("display math Display = false, want true for $$…$$")
	}

	// Inline math $x$ -> RAW TeX + Display false.
	if maths[1].Text != "x" {
		t.Errorf("inline math Text = %q, want %q", maths[1].Text, "x")
	}
	if maths[1].Display {
		t.Errorf("inline math Display = true, want false for $…$")
	}

	// The Text is RAW TeX, never the lossy rendered MathML.
	for _, m := range maths {
		if strings.Contains(m.Text, "<math") || strings.Contains(m.Text, "<mrow") {
			t.Errorf("math Text must be RAW TeX, not MathML: %q", m.Text)
		}
	}

	// The display-math-only paragraph did NOT ALSO double-emit its raw TeX as a
	// paragraph Block (error_recovery: the math case owns that node).
	for _, p := range blocksOfKind(d.Sections[0].Blocks, BlockParagraph) {
		if strings.Contains(p.Text, "E=mc^2") || strings.Contains(p.Text, "$") {
			t.Errorf("math raw TeX leaked into a paragraph Block: %q", p.Text)
		}
	}
}

// TestBuildEmptyDocument covers Test-list case 7: an empty document ("").
//
// Deviation from the TRD's literal wording ("nil Sections"): the TRD's own
// must_haves require "Sections matching slide count/AST" -- i.e. Build must
// faithfully reflect the AST it walks, never fabricate a shape that isn't
// there. chase/markdown/slide.go's slideSplitTransformer unconditionally
// appends one (possibly empty) run after its loop, so markdown.Parse("")
// legitimately produces ONE *markdown.Section (empty, no attrs, no
// comments) -- confirmed empirically. Asserting nil Sections here would
// mean Build lies about the real AST shape, which directly contradicts
// MODEL-01 ("direct walk of the SAME finalized AST", never fabricated).
// Outline remains nil since the empty Section has no headings. See
// 02-01-SUMMARY.md Deviations for the full rationale.
func TestBuildEmptyDocument(t *testing.T) {
	md := ""
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	if d == nil {
		t.Fatal("Build returned nil Document for empty input")
	}
	if d.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", d.SchemaVersion, SchemaVersion)
	}
	if len(d.Sections) != 1 {
		t.Fatalf("Sections = %+v, want exactly one empty Section (matches the real AST: slideSplitTransformer always emits >=1 run)", d.Sections)
	}
	if d.Sections[0].ID != 1 || d.Sections[0].Attrs != nil || d.Sections[0].Notes != nil {
		t.Errorf("Sections[0] = %+v, want {ID:1, Attrs:nil, Notes:nil}", d.Sections[0])
	}
	if d.Outline != nil {
		t.Errorf("Outline = %+v, want nil", d.Outline)
	}
}

// --- EPD-R1 (AODex Objective 11 wave 1): table / image / quote block kinds ---
//
// Motivation, measured before these tests were written: a GFM table renders
// into Output.HTML but its CONTENT is absent from the docmodel entirely -- a
// consumer reading Document (which is what convert/docx, a future convert/xlsx
// and bind/dart all do) cannot recover it by any means. Images behave the same
// way. A blockquote is separately DEGRADED: its text survives, but only as an
// indistinguishable paragraph Block, so a quote cannot be styled as a quote
// downstream.

// TestBuildTableBlocks covers EPD-R1 case 1: a GFM table yields exactly one
// Block{Kind: table} carrying Headers, Rows and per-column Aligns, and its cell
// text does NOT also leak out as loose paragraph blocks.
func TestBuildTableBlocks(t *testing.T) {
	md := "# T\n\n| Metric | Q2 | Q3 |\n|:---|---:|:---:|\n| p95 | 840ms | 550ms |\n| errs | 1.2% | 0.4% |\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	tables := blocksOfKind(d.Sections[0].Blocks, BlockTable)
	if len(tables) != 1 {
		t.Fatalf("table blocks = %d, want 1 (all: %+v)", len(tables), d.Sections[0].Blocks)
	}
	tb := tables[0]

	wantHeaders := []string{"Metric", "Q2", "Q3"}
	if len(tb.Headers) != len(wantHeaders) {
		t.Fatalf("Headers = %+v, want %+v", tb.Headers, wantHeaders)
	}
	for i, w := range wantHeaders {
		if tb.Headers[i] != w {
			t.Errorf("Headers[%d] = %q, want %q", i, tb.Headers[i], w)
		}
	}

	wantRows := [][]string{{"p95", "840ms", "550ms"}, {"errs", "1.2%", "0.4%"}}
	if len(tb.Rows) != len(wantRows) {
		t.Fatalf("Rows = %+v, want %+v", tb.Rows, wantRows)
	}
	for i, wr := range wantRows {
		if len(tb.Rows[i]) != len(wr) {
			t.Fatalf("Rows[%d] = %+v, want %+v", i, tb.Rows[i], wr)
		}
		for j, w := range wr {
			if tb.Rows[i][j] != w {
				t.Errorf("Rows[%d][%d] = %q, want %q", i, j, tb.Rows[i][j], w)
			}
		}
	}

	// Alignment is load-bearing for convert/docx and convert/xlsx -- a
	// right-aligned numeric column must stay right-aligned in the export.
	wantAligns := []string{"left", "right", "center"}
	if len(tb.Aligns) != len(wantAligns) {
		t.Fatalf("Aligns = %+v, want %+v", tb.Aligns, wantAligns)
	}
	for i, w := range wantAligns {
		if tb.Aligns[i] != w {
			t.Errorf("Aligns[%d] = %q, want %q", i, tb.Aligns[i], w)
		}
	}

	// Same invariant the list case asserts: cell text must not ALSO surface as
	// loose prose, or every table would be duplicated in a DOCX export.
	if paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph); len(paras) != 0 {
		t.Errorf("table cell text leaked as %d paragraph block(s): %+v", len(paras), paras)
	}
}

// TestBuildTableRegression is the direct regression for the measured Wave 0
// finding: the string "p95 latency" appeared nowhere in the serialized model.
func TestBuildTableRegression(t *testing.T) {
	md := "# T\n\n| Metric | Q3 |\n|---|---|\n| p95 latency | 550ms |\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	blob, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(blob), "p95 latency") {
		t.Errorf("table payload absent from serialized model; got %s", blob)
	}
}

// TestBuildImageBlocks covers EPD-R1 case 2: a standalone image yields
// Block{Kind: image} with Src, alt text in Text, and the optional title.
func TestBuildImageBlocks(t *testing.T) {
	t.Run("standalone", func(t *testing.T) {
		md := "# T\n\n![Q3 chart](https://example.com/c.png \"Quarterly\")\n"
		doc, pc := markdown.Parse(md)

		d := Build(doc, []byte(md), pc)

		imgs := blocksOfKind(d.Sections[0].Blocks, BlockImage)
		if len(imgs) != 1 {
			t.Fatalf("image blocks = %d, want 1 (all: %+v)", len(imgs), d.Sections[0].Blocks)
		}
		if got, want := imgs[0].Src, "https://example.com/c.png"; got != want {
			t.Errorf("Src = %q, want %q", got, want)
		}
		if got, want := imgs[0].Text, "Q3 chart"; got != want {
			t.Errorf("Text (alt) = %q, want %q", got, want)
		}
		if got, want := imgs[0].Title, "Quarterly"; got != want {
			t.Errorf("Title = %q, want %q", got, want)
		}

		// An image-only paragraph must not also emit an empty/alt-text
		// paragraph Block -- same class of bug isMathOnlyParagraph prevents.
		if paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph); len(paras) != 0 {
			t.Errorf("image-only paragraph leaked %d paragraph block(s): %+v", len(paras), paras)
		}
	})

	t.Run("inline_with_prose_keeps_both", func(t *testing.T) {
		md := "# T\n\nSee ![chart](c.png) for detail.\n"
		doc, pc := markdown.Parse(md)

		d := Build(doc, []byte(md), pc)

		if imgs := blocksOfKind(d.Sections[0].Blocks, BlockImage); len(imgs) != 1 {
			t.Errorf("image blocks = %d, want 1 (all: %+v)", len(imgs), d.Sections[0].Blocks)
		}
		// Mixed prose+image keeps the prose, exactly as mixed prose+math does.
		if paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph); len(paras) != 1 {
			t.Errorf("paragraph blocks = %d, want 1 (all: %+v)", len(paras), d.Sections[0].Blocks)
		}
	})
}

// TestBuildQuoteBlocks covers EPD-R1 case 3: a blockquote yields
// Block{Kind: quote} instead of being silently flattened into an
// indistinguishable paragraph Block.
func TestBuildQuoteBlocks(t *testing.T) {
	md := "# T\n\n> Synthesized from the sync.\n\nOrdinary prose.\n"
	doc, pc := markdown.Parse(md)

	d := Build(doc, []byte(md), pc)

	quotes := blocksOfKind(d.Sections[0].Blocks, BlockQuote)
	if len(quotes) != 1 {
		t.Fatalf("quote blocks = %d, want 1 (all: %+v)", len(quotes), d.Sections[0].Blocks)
	}
	if got, want := quotes[0].Text, "Synthesized from the sync."; got != want {
		t.Errorf("quote Text = %q, want %q", got, want)
	}

	// The quote's own text must NOT also appear as a paragraph Block; only the
	// genuine prose paragraph outside the quote should.
	paras := blocksOfKind(d.Sections[0].Blocks, BlockParagraph)
	if len(paras) != 1 {
		t.Fatalf("paragraph blocks = %d, want 1 (all: %+v)", len(paras), d.Sections[0].Blocks)
	}
	if got, want := paras[0].Text, "Ordinary prose."; got != want {
		t.Errorf("paragraph Text = %q, want %q", got, want)
	}

	// Document order: heading, quote, paragraph.
	blocks := d.Sections[0].Blocks
	if len(blocks) != 3 || blocks[0].Kind != BlockHeading || blocks[1].Kind != BlockQuote || blocks[2].Kind != BlockParagraph {
		t.Errorf("block order = %+v, want [heading, quote, paragraph]", blocks)
	}
}

// TestSchemaVersionV3 pins the deliberate version bump. Adding table/image is
// additive, but re-classifying a blockquote from paragraph to quote CHANGES the
// JSON a v2 consumer would have seen for the same input, so this is v3 and not
// a silent v2 extension. AGENTS.md's envelope schema documents the new kinds.
func TestSchemaVersionV3(t *testing.T) {
	if got, want := SchemaVersion, "eden-press.model/v3"; got != want {
		t.Errorf("SchemaVersion = %q, want %q", got, want)
	}
	md := "# T\n\ntext\n"
	doc, pc := markdown.Parse(md)
	if got := Build(doc, []byte(md), pc).SchemaVersion; got != "eden-press.model/v3" {
		t.Errorf("Document.SchemaVersion = %q, want v3", got)
	}
}
