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

// verify_test.go is the objective's ACCEPTANCE GATE (06-05 Task 2): it
// exercises the full 06-01 -> 06-04 -> notes stack end-to-end through the
// public ToPPTX on one realistic model.Document fixture (titles, paragraphs,
// a nested list, and notes on one section), at BOTH SlideSize16x9 and
// SlideSize4x3, and asserts -- structurally, never visually -- that the
// result is a valid, editable, correctly positioned presentation:
// OBJECTIVE.md criteria 1 (from the model, zero chromedp), 2 (editable
// <p:sp>/<a:t> shapes, never a screenshot), and 4 (opens with elements in
// their expected positions, on both aspect ratios). An OPTIONAL,
// SKIP-guarded LibreOffice-headless convert-to-pdf smoke provides an
// independent-consumer open proof beyond this package's own structural
// asserter.
package pptx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// acceptanceDoc is the hand-built (no generated/property-based data) model
// fixture Test-list cases 4/5 drive: three sections with Outline titles, a
// plain paragraph, a nested (2-level) list, and Notes on the MIDDLE section
// only -- titles + paragraphs + a nested list + notes on >=1 section.
func acceptanceDoc() *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Introduction"},
				{Kind: model.BlockParagraph, Text: "Welcome to the deck."},
			}},
			{ID: 2, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Roadmap"},
				{Kind: model.BlockList, Ordered: true, Items: []model.ListItem{
					{Text: "Phase one"},
					{Text: "Phase one detail", Level: 1},
					{Text: "Phase two"},
				}},
			}, Notes: []string{"Mention the Q3 budget.", "Pause for questions here."}},
			{ID: 3, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Summary"},
				{Kind: model.BlockParagraph, Text: "Thanks for watching."},
			}},
		},
		Outline: []model.OutlineEntry{
			{SectionID: 1, Level: 1, Text: "Introduction"},
			{SectionID: 2, Level: 1, Text: "Roadmap"},
			{SectionID: 3, Level: 1, Text: "Summary"},
		},
	}
}

// offExt is one <a:off>/<a:ext> EMU pair extracted from a slide's XML, in
// document order.
type offExt struct {
	off Point
	ext Extent
}

// offExtPattern matches a shape/group's <a:xfrm> off immediately followed by
// its ext -- true of every xfrm this package emits (buildTextBox,
// buildGroupShape): off and ext are always adjacent with no intervening
// element. buildGroupShape's chOff/chExt use DIFFERENT tag names, so they
// never falsely match this pattern.
var offExtPattern = regexp.MustCompile(`<a:off x="(-?\d+)" y="(-?\d+)"/><a:ext cx="(\d+)" cy="(\d+)"/>`)

// extractOffExtPairs returns every <a:off>/<a:ext> pair in slideXML, in
// document order: the title shape's (if any), then the body group's own,
// then each grouped child's -- exactly buildSlide's emission order.
func extractOffExtPairs(slideXML []byte) []offExt {
	matches := offExtPattern.FindAllSubmatch(slideXML, -1)
	pairs := make([]offExt, 0, len(matches))
	for _, m := range matches {
		x, _ := strconv.ParseInt(string(m[1]), 10, 64)
		y, _ := strconv.ParseInt(string(m[2]), 10, 64)
		cx, _ := strconv.ParseInt(string(m[3]), 10, 64)
		cy, _ := strconv.ParseInt(string(m[4]), 10, 64)
		pairs = append(pairs, offExt{off: Point{X: x, Y: y}, ext: Extent{CX: cx, CY: cy}})
	}
	return pairs
}

// expectedShapePositions independently recomputes buildSlide's OWN layout
// contract for section (title off/ext, then the body group's own off/ext,
// then each body child's off/ext, in document order) by calling the EXACT
// SAME helpers (sectionTitle, skipTitleHeading, buildBodyShapes) buildSlide
// itself uses -- so the position assert proves the generated XML matches the
// WRITER'S OWN declared EMU layout, never a hand-guessed pixel value
// (error_recovery: "assert the EMU values the writer INTENDS").
func expectedShapePositions(section model.Section, outline []model.OutlineEntry, size SlideSize) []offExt {
	gen := newShapeIDGen()
	width := size.CX - 2*slideMarginX

	// Every slide's spTree opens with its OWN mandatory group shape (id 1),
	// whose <a:xfrm> is the fixed, always-zero off/ext (buildSlide emits this
	// unconditionally, before the title) -- the first off/ext pair on every
	// slide, model content notwithstanding.
	want := []offExt{{off: Point{X: 0, Y: 0}, ext: Extent{CX: 0, CY: 0}}}

	title, titleLevel, hasTitle := sectionTitle(section, outline)
	bodyBlocks := section.Blocks
	if hasTitle {
		gen.nextID()
		want = append(want, offExt{off: Point{X: slideMarginX, Y: titleTopY}, ext: Extent{CX: width, CY: titleHeightY}})
		bodyBlocks = skipTitleHeading(section.Blocks, titleLevel, title)
	}
	if len(bodyBlocks) > 0 {
		gen.nextID() // the body group shape's own id
		children, bottomY := buildBodyShapes(bodyBlocks, gen, width, bodyTopY)
		want = append(want, offExt{off: Point{X: slideMarginX, Y: bodyTopY}, ext: Extent{CX: width, CY: bottomY - bodyTopY}})
		for _, c := range children {
			want = append(want, offExt{off: c.off, ext: c.ext})
		}
	}
	return want
}

// wantAcceptanceSlideRuns is the expected per-slide (1-based) editable run
// text for acceptanceDoc(): title (from Outline) followed by body runs, in
// document order.
var wantAcceptanceSlideRuns = map[int][]string{
	1: {"Introduction", "Welcome to the deck."},
	2: {"Roadmap", "Phase one", "Phase one detail", "Phase two"},
	3: {"Summary", "Thanks for watching."},
}

// wantAcceptanceNotes is the expected notesSlide2.xml body run text (section
// 2 is the only section with Notes).
var wantAcceptanceNotes = []string{"Mention the Q3 budget.", "Pause for questions here."}

// assertAcceptanceDeck runs the full Task 2 acceptance gate against a deck
// built from acceptanceDoc() at size: structural openability, editable
// content (title + body runs per slide, notes-slide body text), and EMU
// shape positions matching the writer's own declared layout -- criterion 4
// at whichever aspect ratio size is.
func assertAcceptanceDeck(t *testing.T, size SlideSize) {
	t.Helper()
	doc := acceptanceDoc()
	deck, err := ToPPTX(doc, Options{SlideSize: size})
	if err != nil {
		t.Fatalf("ToPPTX(size=%v): %v", size, err)
	}
	parts := unzipParts(t, deck)

	// (a) structural openability: every part content-type-covered, every
	// r:id resolves, every .rels Target exists in the zip (notes included).
	assertStructurallyOpenable(t, parts)
	// (b) the requested aspect ratio landed in presentation.xml.
	assertSlideSize(t, parts, size)

	for i, sec := range doc.Sections {
		num := i + 1
		name := fmt.Sprintf("ppt/slides/slide%d.xml", num)
		slideXML, ok := parts[name]
		if !ok {
			t.Fatalf("deck (size=%v) missing %s", size, name)
		}

		// (c) editable content: title + body runs, in order, exactly --
		// real editable text, never a screenshot image.
		got := decodeSlideRunTexts(t, slideXML)
		if !reflect.DeepEqual(got, wantAcceptanceSlideRuns[num]) {
			t.Errorf("%s (size=%v) run texts = %v, want %v", name, size, got, wantAcceptanceSlideRuns[num])
		}

		// (d) POSITIONS: every shape's EMU off/ext matches the writer's own
		// declared layout for this section, in document order (criterion 4).
		gotPos := extractOffExtPairs(slideXML)
		wantPos := expectedShapePositions(sec, doc.Outline, size)
		if !reflect.DeepEqual(gotPos, wantPos) {
			t.Errorf("%s (size=%v) shape positions = %+v, want %+v", name, size, gotPos, wantPos)
		}
	}

	// (e) notes reach the notes slide: only section 2 (slide 2) has notes --
	// its notesSlide body carries the exact note text; sections 1 and 3 have
	// NO notesSlide part at all (conditional emission).
	notesXML, ok := parts["ppt/notesSlides/notesSlide2.xml"]
	if !ok {
		t.Fatalf("deck (size=%v) missing ppt/notesSlides/notesSlide2.xml", size)
	}
	gotNotes := decodeSlideRunTexts(t, notesXML)
	if !reflect.DeepEqual(gotNotes, wantAcceptanceNotes) {
		t.Errorf("notesSlide2.xml (size=%v) run texts = %v, want %v", size, gotNotes, wantAcceptanceNotes)
	}
	for _, n := range []int{1, 3} {
		if _, ok := parts[fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", n)]; ok {
			t.Errorf("deck (size=%v) unexpectedly has a notesSlide%d.xml (section %d has no notes)", size, n, n)
		}
	}
}

// TestComprehensiveAcceptance16x9 is Test-list case 4: the realistic
// titles+paragraphs+nested-list+notes fixture, at 16:9, passes the full
// structural + editable-content + position acceptance gate -- the
// objective's (06-convert-pptx) criterion-4 evidence at the widescreen
// aspect ratio.
func TestComprehensiveAcceptance16x9(t *testing.T) {
	assertAcceptanceDeck(t, SlideSize16x9)
}

// TestComprehensiveAcceptance4x3 is Test-list case 5: the SAME fixture, only
// the SlideSize argument differing, passes the SAME acceptance gate at 4:3 --
// criterion 4 proven on BOTH aspect ratios from one fixture.
func TestComprehensiveAcceptance4x3(t *testing.T) {
	assertAcceptanceDeck(t, SlideSize4x3)
}

// TestAcceptanceDeckLibreOfficeSmoke is Test-list case 6 (optional): if
// soffice is on PATH, converts BOTH the 16:9 and 4:3 acceptance decks to PDF
// headlessly -- an independent-consumer open proof beyond this package's own
// structural asserter -- each invocation given a UNIQUE
// -env:UserInstallation (06-RESEARCH anti-pattern: a shared profile
// directory causes lock hangs; t.TempDir() already gives each subtest its
// own directory). Skips cleanly when soffice is absent so CI without
// LibreOffice still passes on the structural/position asserts alone.
func TestAcceptanceDeckLibreOfficeSmoke(t *testing.T) {
	sofficePath := findSoffice()
	if sofficePath == "" {
		t.Skip("no LibreOffice binary found; skipping LibreOffice-headless acceptance smoke")
	}

	doc := acceptanceDoc()
	for _, tc := range []struct {
		name string
		size SlideSize
	}{
		{"16x9", SlideSize16x9},
		{"4x3", SlideSize4x3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deck, err := ToPPTX(doc, Options{SlideSize: tc.size})
			if err != nil {
				t.Fatalf("ToPPTX: %v", err)
			}

			tmpDir := t.TempDir()
			deckPath := filepath.Join(tmpDir, "acceptance.pptx")
			if err := os.WriteFile(deckPath, deck, 0o644); err != nil {
				t.Fatalf("write deck: %v", err)
			}

			userInstall := "file://" + filepath.Join(tmpDir, "soffice-profile")
			cmd := exec.Command(sofficePath,
				"--headless",
				"--convert-to", "pdf",
				"--outdir", tmpDir,
				"-env:UserInstallation="+userInstall,
				deckPath,
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("soffice --headless --convert-to pdf failed: %v\noutput: %s", err, out)
			}

			pdfPath := filepath.Join(tmpDir, "acceptance.pdf")
			if _, err := os.Stat(pdfPath); err != nil {
				t.Fatalf("expected soffice to produce %s: %v", pdfPath, err)
			}
		})
	}
}
