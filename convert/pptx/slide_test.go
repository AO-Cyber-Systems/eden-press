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

package pptx

import (
	"reflect"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// decodeSlideRunTexts strips the XML prolog from a full slide document and
// returns every <a:t> run text in document order (title first, then body).
func decodeSlideRunTexts(t *testing.T, slideXML []byte) []string {
	t.Helper()
	body := strings.TrimPrefix(string(slideXML), xmlDeclaration)
	return decodeRunTexts(t, body)
}

// threeSectionDoc is the inline (hand-built, no generated data) model fixture
// Test-list case 6 drives: three Sections, each with a heading (-> Outline +
// a heading Block) plus body content.
func threeSectionDoc() *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Alpha"},
				{Kind: model.BlockParagraph, Text: "Alpha body text."},
			}},
			{ID: 2, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Beta"},
				{Kind: model.BlockList, Ordered: false, Items: []model.ListItem{
					{Text: "one"},
					{Text: "two", Level: 1},
				}},
			}},
			{ID: 3, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Gamma"},
				{Kind: model.BlockParagraph, Text: "Gamma body."},
			}},
		},
		Outline: []model.OutlineEntry{
			{SectionID: 1, Level: 1, Text: "Alpha"},
			{SectionID: 2, Level: 1, Text: "Beta"},
			{SectionID: 3, Level: 1, Text: "Gamma"},
		},
	}
}

// TestSlidePerSectionCarriesTitleAndBody is Test-list case 6: each Section
// becomes a slide whose shapes carry that section's title (from Outline, once)
// and its body block text -- the title heading is NOT double-rendered as a
// body shape, list items appear as runs.
func TestSlidePerSectionCarriesTitleAndBody(t *testing.T) {
	doc := threeSectionDoc()
	want := map[int][]string{
		1: {"Alpha", "Alpha body text."},
		2: {"Beta", "one", "two"},
		3: {"Gamma", "Gamma body."},
	}
	for _, sec := range doc.Sections {
		slideXML, relsXML := buildSlide(sec, doc.Outline, SlideSize16x9)

		got := decodeSlideRunTexts(t, slideXML)
		if !reflect.DeepEqual(got, want[sec.ID]) {
			t.Errorf("section %d run texts = %v, want %v", sec.ID, got, want[sec.ID])
		}
		// Real editable text, never a screenshot image.
		assertNoImage(t, string(slideXML))
		// Title present exactly once (from Outline) -- no double render.
		title := want[sec.ID][0]
		if n := countRun(got, title); n != 1 {
			t.Errorf("section %d: title %q appears %d times among runs, want exactly 1", sec.ID, title, n)
		}
		// The slide carries at least one grouped shape (criterion 3).
		if !strings.Contains(string(slideXML), "<p:grpSp>") {
			t.Errorf("section %d slide has no <p:grpSp> body group", sec.ID)
		}
		// Slide rels point at the layout.
		if !strings.Contains(string(relsXML), "slideLayouts/slideLayout1.xml") {
			t.Errorf("section %d slide rels do not target slideLayout1.xml:\n%s", sec.ID, relsXML)
		}
	}
}

// TestSlideBetaListIsBulleted asserts Test-list case 6's list section renders
// bullet pPr at the item nesting levels (not a screenshot, not plain runs).
func TestSlideBetaListIsBulleted(t *testing.T) {
	doc := threeSectionDoc()
	slideXML, _ := buildSlide(doc.Sections[1], doc.Outline, SlideSize16x9) // Beta
	s := string(slideXML)
	if !strings.Contains(s, `<a:pPr lvl="0"`) || !strings.Contains(s, `<a:pPr lvl="1"`) {
		t.Errorf("Beta list missing per-level bullet pPr:\n%s", s)
	}
	if !strings.Contains(s, "<a:buChar") {
		t.Errorf("Beta unordered list missing <a:buChar>:\n%s", s)
	}
}

// TestUntitledSectionEmitsBodyOnly is Test-list case 7: a Section with body
// blocks but NO Outline heading emits body shapes and NO fabricated title.
func TestUntitledSectionEmitsBodyOnly(t *testing.T) {
	section := model.Section{ID: 5, Blocks: []model.Block{
		{Kind: model.BlockParagraph, Text: "Orphan body."},
	}}
	// Outline has NO entry for section 5.
	slideXML, _ := buildSlide(section, nil, SlideSize16x9)

	got := decodeSlideRunTexts(t, slideXML)
	if !reflect.DeepEqual(got, []string{"Orphan body."}) {
		t.Fatalf("untitled section run texts = %v, want [Orphan body.]", got)
	}
	if strings.Contains(string(slideXML), `name="Title 1"`) {
		t.Errorf("untitled section fabricated a title shape:\n%s", slideXML)
	}
}

// TestSlideMultiHeadingKeepsExtraHeadings locks the error_recovery contract:
// the lowest-Level heading becomes the title; ADDITIONAL headings are rendered
// as body shapes, never silently dropped.
func TestSlideMultiHeadingKeepsExtraHeadings(t *testing.T) {
	section := model.Section{ID: 1, Blocks: []model.Block{
		{Kind: model.BlockHeading, Level: 1, Text: "Main"},
		{Kind: model.BlockHeading, Level: 2, Text: "Sub"},
		{Kind: model.BlockParagraph, Text: "Body para."},
	}}
	outline := []model.OutlineEntry{
		{SectionID: 1, Level: 1, Text: "Main"},
		{SectionID: 1, Level: 2, Text: "Sub"},
	}
	slideXML, _ := buildSlide(section, outline, SlideSize16x9)
	got := decodeSlideRunTexts(t, slideXML)

	// Title "Main" (lowest level) once; "Sub" kept as a body heading shape.
	want := []string{"Main", "Sub", "Body para."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("multi-heading run texts = %v, want %v", got, want)
	}
	if countRun(got, "Main") != 1 {
		t.Errorf("title heading double-rendered: %v", got)
	}
}

// countRun counts how many entries of runs exactly equal s.
func countRun(runs []string, s string) int {
	n := 0
	for _, r := range runs {
		if r == s {
			n++
		}
	}
	return n
}
