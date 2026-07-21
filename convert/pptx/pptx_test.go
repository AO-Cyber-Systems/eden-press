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
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// wantSlideRuns is the expected per-slide (1-based) editable run text for the
// threeSectionDoc fixture -- title (from Outline, once) followed by body runs.
var wantSlideRuns = map[int][]string{
	1: {"Alpha", "Alpha body text."},
	2: {"Beta", "one", "two"},
	3: {"Gamma", "Gamma body."},
}

// wickedDoc is a one-section fixture whose title and paragraph carry XML
// metacharacters (< > & "), exercising the escaping trap through ToPPTX.
var wickedDoc = model.Document{
	SchemaVersion: model.SchemaVersion,
	Sections: []model.Section{
		{ID: 1, Blocks: []model.Block{
			{Kind: model.BlockHeading, Level: 1, Text: `Tom & "Jerry"`},
			{Kind: model.BlockParagraph, Text: `if a < b && c > d`},
		}},
	},
	Outline: []model.OutlineEntry{
		{SectionID: 1, Level: 1, Text: `Tom & "Jerry"`},
	},
}

// TestToPPTXEndToEnd16x9 is Test-list case 8 (outermost): an inline
// model.Document -> ToPPTX at 16:9 returns a valid zip that passes 06-03's
// structural asserter with N model-driven slides of REAL editable shapes.
func TestToPPTXEndToEnd16x9(t *testing.T) {
	doc := threeSectionDoc()
	deck, err := ToPPTX(doc, Options{SlideSize: SlideSize16x9})
	if err != nil {
		t.Fatalf("ToPPTX: %v", err)
	}
	parts := unzipParts(t, deck)

	// (a) structurally openable: content-types cover every part, every r:id
	// resolves, every rels Target exists.
	assertStructurallyOpenable(t, parts)
	// (b) 16:9 slide size threaded into presentation.xml.
	assertSlideSize(t, parts, SlideSize16x9)

	// (c) exactly N model-driven slide parts, each carrying its section's
	// editable title + body runs; NO screenshot image anywhere.
	for i := 1; i <= len(doc.Sections); i++ {
		name := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		slideXML, ok := parts[name]
		if !ok {
			t.Fatalf("deck missing %s", name)
		}
		got := decodeSlideRunTexts(t, slideXML)
		if !reflect.DeepEqual(got, wantSlideRuns[i]) {
			t.Errorf("%s run texts = %v, want %v", name, got, wantSlideRuns[i])
		}
		if bytes.Contains(slideXML, []byte("<p:pic")) || bytes.Contains(slideXML, []byte("<a:blip")) {
			t.Errorf("%s contains an image element (criterion 2 violation)", name)
		}
		if !bytes.Contains(slideXML, []byte("<p:grpSp>")) {
			t.Errorf("%s missing the grouped-shape case (criterion 3)", name)
		}
	}
	// A 4th slide must NOT exist.
	if _, ok := parts["ppt/slides/slide4.xml"]; ok {
		t.Errorf("unexpected extra slide4.xml for a 3-section deck")
	}
}

// TestToPPTXDefaultsTo16x9 proves the zero-value Options defaults SlideSize to
// 16:9 (per the documented API contract).
func TestToPPTXDefaultsTo16x9(t *testing.T) {
	deck, err := ToPPTX(threeSectionDoc(), Options{})
	if err != nil {
		t.Fatalf("ToPPTX: %v", err)
	}
	assertSlideSize(t, unzipParts(t, deck), SlideSize16x9)
}

// TestToPPTXAspect4x3 is Test-list case 8's aspect-ratio arm: the SAME fixture
// at SlideSize4x3 sets <p:sldSz> to 4:3 -- only the size argument differs.
func TestToPPTXAspect4x3(t *testing.T) {
	deck, err := ToPPTX(threeSectionDoc(), Options{SlideSize: SlideSize4x3})
	if err != nil {
		t.Fatalf("ToPPTX: %v", err)
	}
	parts := unzipParts(t, deck)
	assertStructurallyOpenable(t, parts)
	assertSlideSize(t, parts, SlideSize4x3)
}

// TestToPPTXDeterministic is Test-list case 8's determinism arm: rebuilding the
// same document yields byte-identical output (fixed timestamp, zip.Store,
// stable ordering -- no time.Now/map-ordered leak).
func TestToPPTXDeterministic(t *testing.T) {
	doc := threeSectionDoc()
	a, err := ToPPTX(doc, Options{SlideSize: SlideSize16x9})
	if err != nil {
		t.Fatalf("ToPPTX #1: %v", err)
	}
	b, err := ToPPTX(doc, Options{SlideSize: SlideSize16x9})
	if err != nil {
		t.Fatalf("ToPPTX #2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("ToPPTX output is not deterministic: len(a)=%d len(b)=%d", len(a), len(b))
	}
}

// TestToPPTXEscapesUserText proves user metacharacters survive the full
// pipeline into a well-formed deck (the escaping trap at the content layer).
func TestToPPTXEscapesUserText(t *testing.T) {
	doc := &wickedDoc
	deck, err := ToPPTX(doc, Options{})
	if err != nil {
		t.Fatalf("ToPPTX: %v", err)
	}
	parts := unzipParts(t, deck)
	assertStructurallyOpenable(t, parts) // would fail if the XML were corrupt
	slideXML := parts["ppt/slides/slide1.xml"]
	got := decodeSlideRunTexts(t, slideXML)
	want := []string{`Tom & "Jerry"`, `if a < b && c > d`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("escaped round-trip = %v, want %v", got, want)
	}
	if !strings.Contains(string(slideXML), "&lt;") || !strings.Contains(string(slideXML), "&amp;") {
		t.Errorf("expected &lt;/&amp; escaping in slide1.xml")
	}
}
