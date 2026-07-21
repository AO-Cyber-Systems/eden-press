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
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

// decodeRunTexts walks fragment (a DrawingML shape/paragraph XML fragment)
// and returns the DECODED CharData of every <a:t> run, in document order --
// the round-trip proof that user text was correctly XML-escaped (parsing
// yields back the ORIGINAL text, never a corrupted document). The fragment is
// wrapped in a root that declares the a:/p: prefixes so the decoder resolves
// them cleanly.
func decodeRunTexts(t *testing.T, fragment string) []string {
	t.Helper()
	wrapped := `<root xmlns:a="` + nsDrawingML + `" xmlns:p="` + nsPresentationML + `" xmlns:r="` + nsRelationships + `">` + fragment + `</root>`
	dec := xml.NewDecoder(strings.NewReader(wrapped))
	var texts []string
	inT := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decode fragment (proves malformed/corrupt XML): %v", err)
		}
		switch e := tok.(type) {
		case xml.StartElement:
			if e.Name.Local == "t" {
				inT = true
			}
		case xml.EndElement:
			if e.Name.Local == "t" {
				inT = false
			}
		case xml.CharData:
			if inT {
				texts = append(texts, string(e))
			}
		}
	}
	return texts
}

// TestTextBoxShapeTitle is Test-list case 1: an Outline-heading title maps to
// a real, editable <p:sp> text box whose <a:t> carries the heading text,
// placed at exact EMU xfrm, sz in centipoints -- NOT a screenshot image.
func TestTextBoxShapeTitle(t *testing.T) {
	off := Point{X: Inches(0.5), Y: Inches(0.3)}
	ext := Extent{CX: Inches(9), CY: Inches(1.2)}
	var sb strings.Builder
	buildTextBox(&sb, textBox{
		id:    2,
		name:  "Title 1",
		off:   off,
		ext:   ext,
		paras: []paragraph{{text: "Heading text", sz: Centipoints(44), bold: true}},
	})
	got := sb.String()

	// Real editable run carrying the exact title text (round-trip decode).
	texts := decodeRunTexts(t, got)
	if len(texts) != 1 || texts[0] != "Heading text" {
		t.Fatalf("run texts = %v, want [\"Heading text\"]", texts)
	}
	// Exact EMU placement (criterion 3: EMU-verified positions).
	wantXfrm := `<a:xfrm><a:off x="457200" y="274320"/><a:ext cx="8229600" cy="1097280"/></a:xfrm>`
	if !strings.Contains(got, wantXfrm) {
		t.Errorf("missing exact EMU xfrm %q in:\n%s", wantXfrm, got)
	}
	// sz is centipoints (44pt => 4400), bold title.
	if !strings.Contains(got, `sz="4400"`) || !strings.Contains(got, `b="1"`) {
		t.Errorf("title run missing sz=\"4400\"/b=\"1\" in:\n%s", got)
	}
	// cNvPr id threaded through.
	if !strings.Contains(got, `<p:cNvPr id="2" name="Title 1"/>`) {
		t.Errorf("missing cNvPr id=2 name=Title 1 in:\n%s", got)
	}
	// It is a text-box shape, not an image (criterion 2).
	if !strings.Contains(got, `txBox="1"`) {
		t.Errorf("shape is not marked txBox=\"1\" in:\n%s", got)
	}
	assertNoImage(t, got)
}

// TestShapeParagraphBodyRun is Test-list case 2: a paragraph Block maps to a
// <p:sp>/<a:p>/<a:r>/<a:t> body run carrying the paragraph's plain text.
func TestShapeParagraphBodyRun(t *testing.T) {
	var sb strings.Builder
	buildTextBox(&sb, textBox{
		id:    2,
		name:  "Body 2",
		off:   Point{X: Inches(0.5), Y: Inches(1.7)},
		ext:   Extent{CX: Inches(9), CY: Inches(0.6)},
		paras: []paragraph{{text: "The quick brown fox.", sz: Centipoints(18)}},
	})
	got := sb.String()
	texts := decodeRunTexts(t, got)
	if len(texts) != 1 || texts[0] != "The quick brown fox." {
		t.Fatalf("run texts = %v, want [\"The quick brown fox.\"]", texts)
	}
	// Plain paragraph carries NO bullet pPr.
	if strings.Contains(got, "<a:buChar") || strings.Contains(got, "<a:buAutoNum") {
		t.Errorf("plain paragraph unexpectedly emitted a bullet pPr:\n%s", got)
	}
	assertNoImage(t, got)
}

// TestListShapeBulletAndOrdered is Test-list case 3: an unordered list Block
// emits <a:pPr lvl>...<a:buChar> per item at its nesting Level; an ordered
// list emits <a:buAutoNum type="arabicPeriod"/>.
func TestListShapeBulletAndOrdered(t *testing.T) {
	// Unordered, two nesting levels.
	var ub strings.Builder
	buildTextBox(&ub, textBox{
		id:   2,
		name: "Body 2",
		off:  Point{X: Inches(0.5), Y: Inches(1.7)},
		ext:  Extent{CX: Inches(9), CY: Inches(1.2)},
		paras: []paragraph{
			{text: "Top item", sz: Centipoints(18), bullet: bulletChar, level: 0},
			{text: "Nested item", sz: Centipoints(18), bullet: bulletChar, level: 1},
		},
	})
	uget := ub.String()
	if !strings.Contains(uget, `<a:pPr lvl="0" marL="457200" indent="-457200"><a:buChar char="`+bulletGlyph+`"/></a:pPr>`) {
		t.Errorf("missing level-0 bullet pPr in:\n%s", uget)
	}
	if !strings.Contains(uget, `<a:pPr lvl="1" marL="914400" indent="-457200"><a:buChar char="`+bulletGlyph+`"/></a:pPr>`) {
		t.Errorf("missing level-1 bullet pPr (marL scaled per level) in:\n%s", uget)
	}
	if texts := decodeRunTexts(t, uget); len(texts) != 2 || texts[0] != "Top item" || texts[1] != "Nested item" {
		t.Errorf("bullet run texts = %v, want [Top item, Nested item]", texts)
	}

	// Ordered.
	var ob strings.Builder
	buildTextBox(&ob, textBox{
		id:   2,
		name: "Body 2",
		off:  Point{X: Inches(0.5), Y: Inches(1.7)},
		ext:  Extent{CX: Inches(9), CY: Inches(0.6)},
		paras: []paragraph{
			{text: "First", sz: Centipoints(18), bullet: bulletAutoNum, level: 0},
		},
	})
	oget := ob.String()
	if !strings.Contains(oget, `<a:buAutoNum type="arabicPeriod"/>`) {
		t.Errorf("ordered list missing <a:buAutoNum type=\"arabicPeriod\"/> in:\n%s", oget)
	}
	if strings.Contains(oget, "<a:buChar") {
		t.Errorf("ordered list must not emit a <a:buChar> bullet:\n%s", oget)
	}
}

// TestGroupShapeIdentity is Test-list case 4 (criterion 3): a <p:grpSp> whose
// xfrm has chOff==off and chExt==ext (06-02 identity transform), wrapping a
// child <p:sp> whose off/ext are literal slide EMU -- all EMU-asserted.
func TestGroupShapeIdentity(t *testing.T) {
	off := Point{X: Inches(1), Y: Inches(1.5)}
	ext := Extent{CX: Inches(4), CY: Inches(3)}
	xf := IdentityGroupTransform(off, ext)

	// The identity relationship itself, at the source: chOff==off, chExt==ext.
	if xf.ChOff != off || xf.ChExt != ext {
		t.Fatalf("IdentityGroupTransform not identity: %+v", xf)
	}

	child := textBox{
		id:    3,
		name:  "Child 1",
		off:   off, // literal slide EMU (identity => unchanged)
		ext:   ext,
		paras: []paragraph{{text: "grouped", sz: Centipoints(18)}},
	}
	var sb strings.Builder
	buildGroupShape(&sb, groupShape{id: 2, name: "Body Group", xf: xf, children: []textBox{child}})
	got := sb.String()

	if !strings.Contains(got, "<p:grpSp>") || !strings.Contains(got, "</p:grpSp>") {
		t.Fatalf("output is not a <p:grpSp>:\n%s", got)
	}
	// off (914400,1371600), ext (3657600,2743200); chOff==off, chExt==ext.
	wantXfrm := `<a:xfrm>` +
		`<a:off x="914400" y="1371600"/>` +
		`<a:ext cx="3657600" cy="2743200"/>` +
		`<a:chOff x="914400" y="1371600"/>` +
		`<a:chExt cx="3657600" cy="2743200"/>` +
		`</a:xfrm>`
	if !strings.Contains(got, wantXfrm) {
		t.Errorf("group xfrm not identity-EMU-correct.\nwant substring: %s\ngot:\n%s", wantXfrm, got)
	}
	// The child text box lives inside the group and carries a real run.
	if !strings.Contains(got, `<p:sp>`) {
		t.Errorf("group has no child <p:sp>:\n%s", got)
	}
	if texts := decodeRunTexts(t, got); len(texts) != 1 || texts[0] != "grouped" {
		t.Errorf("group child run texts = %v, want [grouped]", texts)
	}
	assertNoImage(t, got)
}

// TestXMLEscaping is Test-list case 5: user text containing <, &, " produces
// WELL-FORMED XML that parses back to the original text (no corruption).
func TestXMLEscaping(t *testing.T) {
	raw := `A < B & "quoted" > C`
	var sb strings.Builder
	buildTextBox(&sb, textBox{
		id:   2,
		name: `Danger < & " Name`,
		off:  Point{X: 0, Y: 0},
		ext:  Extent{CX: Inches(1), CY: Inches(1)},
		paras: []paragraph{
			{text: raw, sz: Centipoints(18)},
		},
	})
	got := sb.String()

	// The dangerous metacharacters must NOT appear raw in element content.
	if strings.Contains(got, "<a:t>A < B") {
		t.Errorf("raw unescaped '<' leaked into <a:t> content:\n%s", got)
	}
	if !strings.Contains(got, "&lt;") || !strings.Contains(got, "&amp;") {
		t.Errorf("expected &lt; and &amp; escaping in:\n%s", got)
	}

	// The proof: parse the emitted XML; the decoded run text equals the input.
	texts := decodeRunTexts(t, got)
	if len(texts) != 1 || texts[0] != raw {
		t.Fatalf("round-trip decode = %v, want [%q]", texts, raw)
	}

	// The whole shape (including the escaped name attribute) is well-formed:
	// wrap and fully unmarshal without error.
	wrapped := `<root xmlns:a="` + nsDrawingML + `" xmlns:p="` + nsPresentationML + `">` + got + `</root>`
	if err := xml.Unmarshal([]byte(wrapped), new(struct{})); err != nil {
		t.Fatalf("emitted shape XML is not well-formed (escaping bug): %v", err)
	}
}

// assertNoImage asserts a shape fragment carries no picture/image element --
// criterion 2's negative: text is real <a:t> runs, never one screenshot per
// slide.
func assertNoImage(t *testing.T, fragment string) {
	t.Helper()
	for _, bad := range []string{"<p:pic", "<a:blip", "<pic:pic"} {
		if strings.Contains(fragment, bad) {
			t.Errorf("shape unexpectedly contains image element %q (criterion 2 violation):\n%s", bad, fragment)
		}
	}
}
