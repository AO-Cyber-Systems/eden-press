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
	"strconv"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// notesDoc returns a 3-section fixture where ONLY the middle section (ID 2)
// has speaker notes -- Test-list case 3's conditional-emission fixture.
func notesDoc() *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{{Kind: model.BlockHeading, Level: 1, Text: "One"}}},
			{ID: 2, Blocks: []model.Block{{Kind: model.BlockHeading, Level: 1, Text: "Two"}},
				Notes: []string{"first note", "second"}},
			{ID: 3, Blocks: []model.Block{{Kind: model.BlockHeading, Level: 1, Text: "Three"}}},
		},
		Outline: []model.OutlineEntry{
			{SectionID: 1, Level: 1, Text: "One"},
			{SectionID: 2, Level: 1, Text: "Two"},
			{SectionID: 3, Level: 1, Text: "Three"},
		},
	}
}

// TestNotesSlideBodyRuns is Test-list case 1: a notes body renders one
// <a:p> per note string, in a <p:ph type="body"> placeholder shape, with
// escaped text (round-trip decode proves it).
func TestNotesSlideBodyRuns(t *testing.T) {
	xml := buildNotesSlide([]string{"first note", "second"})
	if xml == nil {
		t.Fatal("buildNotesSlide returned nil for a non-empty notes slice")
	}
	got := decodeSlideRunTexts(t, xml)
	want := []string{"first note", "second"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("notes slide run texts = %v, want %v", got, want)
	}
	s := string(xml)
	if !strings.Contains(s, `<p:ph type="body" idx="1"/>`) {
		t.Errorf("notes slide missing <p:ph type=\"body\"/> placeholder:\n%s", s)
	}
	if !strings.Contains(s, "<p:notes") {
		t.Errorf("notes slide root is not <p:notes>:\n%s", s)
	}
}

// TestNotesSlideEmptyReturnsNil proves buildNotesSlide is CONDITIONAL at the
// single-section level: a Section with no notes must never produce a notes
// part (anti-pattern: no dangling empty notes part).
func TestNotesSlideEmptyReturnsNil(t *testing.T) {
	if xml := buildNotesSlide(nil); xml != nil {
		t.Errorf("buildNotesSlide(nil) = %q, want nil", xml)
	}
	if xml := buildNotesSlide([]string{}); xml != nil {
		t.Errorf("buildNotesSlide([]string{}) = %q, want nil", xml)
	}
}

// TestNotesSlideEscapesUserText proves note text funnels through the same
// escapeXML seam as every other user string in this package.
func TestNotesSlideEscapesUserText(t *testing.T) {
	xml := buildNotesSlide([]string{`Tom & "Jerry" < 5`})
	got := decodeSlideRunTexts(t, xml)
	want := []string{`Tom & "Jerry" < 5`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("escaped round-trip = %v, want %v", got, want)
	}
	if !strings.Contains(string(xml), "&amp;") || !strings.Contains(string(xml), "&lt;") {
		t.Errorf("expected &amp;/&lt; escaping in the notes slide XML")
	}
}

// TestNotesWiringClosure is Test-list case 2: a deck with notes closes the
// FULL 4-way notes rels chain -- notesMaster1.xml (+rels->theme1) present
// once, the notes slide's rels -> slideN + notesMaster1, slideN.xml.rels ->
// notesSlideN, content-types Overrides for notesMaster + the notesSlide,
// presentation.xml.rels carries the notesMaster rId, and the WHOLE deck
// still passes the general structural asserter with notes present.
func TestNotesWiringClosure(t *testing.T) {
	doc := notesDoc()
	deck, err := ToPPTX(doc, Options{SlideSize: SlideSize16x9})
	if err != nil {
		t.Fatalf("ToPPTX: %v", err)
	}
	parts := unzipParts(t, deck)

	// The general structural asserter (content-types coverage + every r:id
	// resolves + every rels Target exists) must pass with notes present.
	assertStructurallyOpenable(t, parts)

	masterXML, ok := parts["ppt/notesMasters/notesMaster1.xml"]
	if !ok {
		t.Fatal("deck is missing ppt/notesMasters/notesMaster1.xml")
	}
	if len(masterXML) == 0 {
		t.Error("ppt/notesMasters/notesMaster1.xml is empty")
	}
	masterRels, ok := parts["ppt/notesMasters/_rels/notesMaster1.xml.rels"]
	if !ok {
		t.Fatal("deck is missing ppt/notesMasters/_rels/notesMaster1.xml.rels")
	}
	if rels, err := parseRelsXML(masterRels); err != nil {
		t.Fatalf("parseRelsXML(notesMaster1.xml.rels): %v", err)
	} else if !relsTargetPresent(rels, "../theme/theme1.xml") {
		t.Errorf("notesMaster1.xml.rels missing -> ../theme/theme1.xml relationship: %+v", rels)
	}

	// Only section 2 has notes -> exactly one notesSlideN.xml.
	notesXML, ok := parts["ppt/notesSlides/notesSlide2.xml"]
	if !ok {
		t.Fatal("deck is missing ppt/notesSlides/notesSlide2.xml")
	}
	got := decodeSlideRunTexts(t, notesXML)
	if want := []string{"first note", "second"}; !reflect.DeepEqual(got, want) {
		t.Errorf("notesSlide2.xml run texts = %v, want %v", got, want)
	}

	notesRels, ok := parts["ppt/notesSlides/_rels/notesSlide2.xml.rels"]
	if !ok {
		t.Fatal("deck is missing ppt/notesSlides/_rels/notesSlide2.xml.rels")
	}
	rels, err := parseRelsXML(notesRels)
	if err != nil {
		t.Fatalf("parseRelsXML(notesSlide2.xml.rels): %v", err)
	}
	if !relsTargetPresent(rels, "../slides/slide2.xml") {
		t.Errorf("notesSlide2.xml.rels missing -> ../slides/slide2.xml: %+v", rels)
	}
	if !relsTargetPresent(rels, "../notesMasters/notesMaster1.xml") {
		t.Errorf("notesSlide2.xml.rels missing -> ../notesMasters/notesMaster1.xml: %+v", rels)
	}

	slide2Rels, ok := parts["ppt/slides/_rels/slide2.xml.rels"]
	if !ok {
		t.Fatal("deck is missing ppt/slides/_rels/slide2.xml.rels")
	}
	rels2, err := parseRelsXML(slide2Rels)
	if err != nil {
		t.Fatalf("parseRelsXML(slide2.xml.rels): %v", err)
	}
	if !relsTargetPresent(rels2, "../notesSlides/notesSlide2.xml") {
		t.Errorf("slide2.xml.rels missing -> ../notesSlides/notesSlide2.xml: %+v", rels2)
	}

	ct, err := parseContentTypesXML(parts["[Content_Types].xml"])
	if err != nil {
		t.Fatalf("parseContentTypesXML: %v", err)
	}
	if got := ct.Overrides["/ppt/notesMasters/notesMaster1.xml"]; got != ctNotesMaster {
		t.Errorf("[Content_Types].xml notesMaster1.xml Override = %q, want %q", got, ctNotesMaster)
	}
	if got := ct.Overrides["/ppt/notesSlides/notesSlide2.xml"]; got != ctNotesSlide {
		t.Errorf("[Content_Types].xml notesSlide2.xml Override = %q, want %q", got, ctNotesSlide)
	}

	presRels, err := parseRelsXML(parts["ppt/_rels/presentation.xml.rels"])
	if err != nil {
		t.Fatalf("parseRelsXML(presentation.xml.rels): %v", err)
	}
	if !relsTargetPresent(presRels, "notesMasters/notesMaster1.xml") {
		t.Errorf("presentation.xml.rels missing the notesMaster rId: %+v", presRels)
	}
}

// TestConditionalNotesEmission is Test-list case 3: only the section that
// HAS notes gets a notesSlideN + a slide-rels entry; sibling notes-free
// slides get NEITHER. A deck with zero notes anywhere emits NO notesMaster
// and NO notesSlides at all.
func TestConditionalNotesEmission(t *testing.T) {
	doc := notesDoc()
	deck, err := ToPPTX(doc, Options{})
	if err != nil {
		t.Fatalf("ToPPTX: %v", err)
	}
	parts := unzipParts(t, deck)

	for _, num := range []int{1, 3} {
		name := "ppt/notesSlides/notesSlide" + strconv.Itoa(num) + ".xml"
		if _, ok := parts[name]; ok {
			t.Errorf("notes-free section %d unexpectedly produced %s", num, name)
		}
		relsName := "ppt/slides/_rels/slide" + strconv.Itoa(num) + ".xml.rels"
		rels, err := parseRelsXML(parts[relsName])
		if err != nil {
			t.Fatalf("parseRelsXML(%s): %v", relsName, err)
		}
		if relsTargetPresent(rels, "../notesSlides/notesSlide"+strconv.Itoa(num)+".xml") {
			t.Errorf("%s unexpectedly references a notesSlide", relsName)
		}
	}
	if _, ok := parts["ppt/notesSlides/notesSlide2.xml"]; !ok {
		t.Error("section 2 (which HAS notes) is missing its notesSlide2.xml")
	}
	if _, ok := parts["ppt/notesMasters/notesMaster1.xml"]; !ok {
		t.Error("deck with a notes section is missing notesMaster1.xml")
	}

	// A deck with ZERO notes anywhere emits no notes parts whatsoever.
	noNotesDoc := threeSectionDoc()
	noNotesDeck, err := ToPPTX(noNotesDoc, Options{})
	if err != nil {
		t.Fatalf("ToPPTX (no notes): %v", err)
	}
	noNotesParts := unzipParts(t, noNotesDeck)
	for name := range noNotesParts {
		if strings.Contains(name, "notesSlide") || strings.Contains(name, "notesMaster") {
			t.Errorf("notes-free deck unexpectedly contains part %q", name)
		}
	}
	assertStructurallyOpenable(t, noNotesParts)
}

// relsTargetPresent reports whether any relationship in rels has the exact
// given Target.
func relsTargetPresent(rels []relationship, target string) bool {
	for _, r := range rels {
		if r.Target == target {
			return true
		}
	}
	return false
}
