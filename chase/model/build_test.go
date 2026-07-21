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
	"fmt"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
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
