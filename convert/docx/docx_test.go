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

package docx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// readPart returns the named entry's bytes from an OPC zip package.
func readPart(t *testing.T, pkg []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		defer rc.Close()
		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(b)
	}
	t.Fatalf("part %q not present; have %s", name, partNames(t, pkg))
	return ""
}

func partNames(t *testing.T, pkg []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	var out []string
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	return out
}

// docWith wraps blocks in a single-section Document.
func docWith(blocks ...model.Block) *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections:      []model.Section{{ID: 1, Blocks: blocks}},
	}
}

// TestToDOCXNilDoc: a nil document is a caller error, not a panic.
func TestToDOCXNilDoc(t *testing.T) {
	if _, err := ToDOCX(nil, Options{}); err == nil {
		t.Fatal("ToDOCX(nil) = nil error, want an error")
	}
}

// TestPackageGraph: the minimal OPC part graph a Word-openable .docx requires
// is present, and [Content_Types].xml covers every part in it.
func TestPackageGraph(t *testing.T) {
	pkg, err := ToDOCX(docWith(model.Block{Kind: model.BlockParagraph, Text: "hi"}), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}

	for _, want := range []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/document.xml",
		"word/_rels/document.xml.rels",
		"word/styles.xml",
		"word/numbering.xml",
		"docProps/core.xml",
		"docProps/app.xml",
	} {
		found := false
		for _, got := range partNames(t, pkg) {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required part %q (have %v)", want, partNames(t, pkg))
		}
	}

	ct := readPart(t, pkg, "[Content_Types].xml")
	for _, want := range []string{
		"/word/document.xml",
		"/word/styles.xml",
		"/word/numbering.xml",
		"/docProps/core.xml",
		"/docProps/app.xml",
	} {
		if !strings.Contains(ct, want) {
			t.Errorf("[Content_Types].xml has no Override for %q: %s", want, ct)
		}
	}
	// The officeDocument relationship is what Word follows first; without it
	// the package opens as "corrupt" even with every part present.
	rels := readPart(t, pkg, "_rels/.rels")
	if !strings.Contains(rels, "word/document.xml") {
		t.Errorf("root rels does not point at word/document.xml: %s", rels)
	}
}

// TestDeterminism: the same document twice yields byte-identical packages --
// the same guarantee convert/pptx makes, and a precondition for caching an
// export by content hash.
func TestDeterminism(t *testing.T) {
	d := docWith(
		model.Block{Kind: model.BlockHeading, Level: 1, Text: "T"},
		model.Block{Kind: model.BlockParagraph, Text: "body"},
	)
	a, err := ToDOCX(d, Options{})
	if err != nil {
		t.Fatalf("ToDOCX #1: %v", err)
	}
	b, err := ToDOCX(d, Options{})
	if err != nil {
		t.Fatalf("ToDOCX #2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("ToDOCX is not deterministic: %d vs %d bytes", len(a), len(b))
	}
}

// TestHeadingBlocks: heading levels map onto Word's built-in Heading1..6
// styles, which is what makes Word's navigation pane and TOC field work.
func TestHeadingBlocks(t *testing.T) {
	pkg, err := ToDOCX(docWith(
		model.Block{Kind: model.BlockHeading, Level: 1, Text: "Title"},
		model.Block{Kind: model.BlockHeading, Level: 3, Text: "Sub"},
	), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")
	if !strings.Contains(body, `<w:pStyle w:val="Heading1"/>`) {
		t.Errorf("no Heading1 style: %s", body)
	}
	if !strings.Contains(body, `<w:pStyle w:val="Heading3"/>`) {
		t.Errorf("no Heading3 style: %s", body)
	}
	if !strings.Contains(body, "<w:t>Title</w:t>") {
		t.Errorf("heading text missing: %s", body)
	}
}

// TestParagraphAndQuoteBlocks: prose is Normal; an EPD-R1 quote block gets the
// distinct Quote style rather than being indistinguishable from prose -- the
// whole point of adding the kind upstream.
func TestParagraphAndQuoteBlocks(t *testing.T) {
	pkg, err := ToDOCX(docWith(
		model.Block{Kind: model.BlockParagraph, Text: "ordinary"},
		model.Block{Kind: model.BlockQuote, Text: "quoted"},
	), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")
	if !strings.Contains(body, "<w:t>ordinary</w:t>") {
		t.Errorf("paragraph text missing: %s", body)
	}
	if !strings.Contains(body, `<w:pStyle w:val="Quote"/>`) {
		t.Errorf("quote block did not get the Quote style: %s", body)
	}
	// The quote run carries xml:space="preserve" (quotes may be multi-line),
	// so match the text content rather than a bare <w:t>.
	if !strings.Contains(body, ">quoted</w:t>") {
		t.Errorf("quote text missing: %s", body)
	}
}

// TestListBlocks: bullet and ordered lists reference the two numbering
// definitions in numbering.xml, and per-item nesting Level becomes w:ilvl.
func TestListBlocks(t *testing.T) {
	pkg, err := ToDOCX(docWith(
		model.Block{Kind: model.BlockList, Items: []model.ListItem{
			{Text: "a"}, {Text: "b", Level: 1},
		}},
		model.Block{Kind: model.BlockList, Ordered: true, Items: []model.ListItem{
			{Text: "one"},
		}},
	), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")
	if !strings.Contains(body, `<w:numId w:val="1"/>`) {
		t.Errorf("bullet list does not reference numId 1: %s", body)
	}
	if !strings.Contains(body, `<w:numId w:val="2"/>`) {
		t.Errorf("ordered list does not reference numId 2: %s", body)
	}
	if !strings.Contains(body, `<w:ilvl w:val="1"/>`) {
		t.Errorf("nested item did not become ilvl 1: %s", body)
	}

	// numbering.xml declares the instances as <w:num w:numId="N">, which is a
	// different spelling from document.xml's <w:numId w:val="N"/> reference.
	// Both must exist or Word silently drops the list formatting.
	num := readPart(t, pkg, "word/numbering.xml")
	for _, want := range []string{`<w:num w:numId="1">`, `<w:num w:numId="2">`} {
		if !strings.Contains(num, want) {
			t.Errorf("numbering.xml missing %s: %s", want, num)
		}
	}
}

// TestCodeBlock: a fenced code block keeps its line structure. Word has no
// <pre>, so each source line must become its own run separated by <w:br/>;
// joining them into one run would collapse the whole listing onto one line.
func TestCodeBlock(t *testing.T) {
	pkg, err := ToDOCX(docWith(
		model.Block{Kind: model.BlockCode, Language: "go", Text: "line one\nline two\n"},
	), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")
	if !strings.Contains(body, `<w:pStyle w:val="Code"/>`) {
		t.Errorf("code block did not get the Code style: %s", body)
	}
	if !strings.Contains(body, "line one") || !strings.Contains(body, "line two") {
		t.Errorf("code text missing: %s", body)
	}
	if !strings.Contains(body, "<w:br/>") {
		t.Errorf("code lines were not separated by <w:br/>: %s", body)
	}
	// Leading whitespace must survive -- xml:space="preserve" or Word trims it.
	if !strings.Contains(body, `xml:space="preserve"`) {
		t.Errorf("code runs lack xml:space=preserve: %s", body)
	}
}

// TestTableBlock is the payoff for EPD-R1: a table that previously could not
// reach an exporter at all now becomes a real Word table with a header row and
// per-column alignment.
func TestTableBlock(t *testing.T) {
	pkg, err := ToDOCX(docWith(model.Block{
		Kind:    model.BlockTable,
		Headers: []string{"Metric", "Q3"},
		Rows:    [][]string{{"p95 latency", "550ms"}, {"errors", "0.4%"}},
		Aligns:  []string{"left", "right"},
	}), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")

	if !strings.Contains(body, "<w:tbl>") {
		t.Fatalf("no <w:tbl> emitted: %s", body)
	}
	for _, want := range []string{"Metric", "Q3", "p95 latency", "550ms", "errors", "0.4%"} {
		if !strings.Contains(body, "<w:t>"+want+"</w:t>") {
			t.Errorf("table cell %q missing: %s", want, body)
		}
	}
	// Right-aligned numeric column must carry its justification.
	if !strings.Contains(body, `<w:jc w:val="right"/>`) {
		t.Errorf("right alignment not applied: %s", body)
	}
	// Header row is repeated across page breaks -- the reason tblHeader exists.
	if !strings.Contains(body, "<w:tblHeader/>") {
		t.Errorf("header row not marked as a repeating header: %s", body)
	}
}

// TestTableRaggedRows: rows shorter or longer than the header are padded and
// truncated to the header's column count. The model reports rows as authored
// (chase/model deliberately does not normalize), so the EXPORTER must, or Word
// rejects the table as malformed.
func TestTableRaggedRows(t *testing.T) {
	pkg, err := ToDOCX(docWith(model.Block{
		Kind:    model.BlockTable,
		Headers: []string{"A", "B", "C"},
		Rows:    [][]string{{"1"}, {"1", "2", "3", "4"}},
	}), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")
	// Every <w:tr> must contain exactly 3 <w:tc> -- header + both body rows.
	rows := strings.Count(body, "<w:tr>")
	cells := strings.Count(body, "<w:tc>")
	if rows != 3 {
		t.Fatalf("rows = %d, want 3 (header + 2 body): %s", rows, body)
	}
	if cells != 9 {
		t.Errorf("cells = %d, want 9 (3 rows x 3 cols, ragged rows normalized): %s", cells, body)
	}
	// The overflowing 4th cell is dropped, not silently shifted into a new row.
	if strings.Contains(body, "<w:t>4</w:t>") {
		t.Errorf("overflow cell leaked into the table: %s", body)
	}
}

// TestXMLEscaping: user text is untrusted. A title containing XML
// metacharacters must not be able to corrupt (or inject into) document.xml.
func TestXMLEscaping(t *testing.T) {
	pkg, err := ToDOCX(docWith(
		model.Block{Kind: model.BlockParagraph, Text: `a & b < c > d "e" <w:p/>`},
	), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	body := readPart(t, pkg, "word/document.xml")
	if strings.Contains(body, "<w:t>a & b") {
		t.Errorf("raw ampersand left unescaped: %s", body)
	}
	if !strings.Contains(body, "&amp;") || !strings.Contains(body, "&lt;") {
		t.Errorf("text not XML-escaped: %s", body)
	}
	// The injected markup must appear as text, never as a second paragraph.
	if strings.Count(body, "<w:p>") != 1 {
		t.Errorf("injected <w:p/> created a real paragraph: %s", body)
	}
	// And the result must still be well-formed XML.
	if err := xmlWellFormed(body); err != nil {
		t.Errorf("document.xml is not well-formed after escaping: %v", err)
	}
}

// TestSectionPageBreaks: each model Section after the first starts a new page
// by default (a Section is a slide/page boundary upstream), and the behavior is
// switchable for continuous-flow reports.
func TestSectionPageBreaks(t *testing.T) {
	d := &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{{Kind: model.BlockParagraph, Text: "one"}}},
			{ID: 2, Blocks: []model.Block{{Kind: model.BlockParagraph, Text: "two"}}},
		},
	}

	pkg, err := ToDOCX(d, Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	if got := strings.Count(readPart(t, pkg, "word/document.xml"), `<w:br w:type="page"/>`); got != 1 {
		t.Errorf("page breaks = %d, want 1 between 2 sections", got)
	}

	pkg, err = ToDOCX(d, Options{ContinuousSections: true})
	if err != nil {
		t.Fatalf("ToDOCX continuous: %v", err)
	}
	if got := strings.Count(readPart(t, pkg, "word/document.xml"), `<w:br w:type="page"/>`); got != 0 {
		t.Errorf("page breaks = %d, want 0 with ContinuousSections", got)
	}
}

// TestDocumentWellFormed: every part this package emits must parse as XML.
func TestDocumentWellFormed(t *testing.T) {
	pkg, err := ToDOCX(docWith(
		model.Block{Kind: model.BlockHeading, Level: 2, Text: "H"},
		model.Block{Kind: model.BlockParagraph, Text: "p"},
		model.Block{Kind: model.BlockQuote, Text: "q"},
		model.Block{Kind: model.BlockCode, Text: "c\n"},
		model.Block{Kind: model.BlockList, Items: []model.ListItem{{Text: "i"}}},
		model.Block{Kind: model.BlockTable, Headers: []string{"h"}, Rows: [][]string{{"r"}}},
		model.Block{Kind: model.BlockImage, Src: "x.png", Text: "alt"},
		model.Block{Kind: model.BlockMath, Text: "E=mc^2", Display: true},
	), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}
	for _, name := range partNames(t, pkg) {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".rels") {
			continue
		}
		if err := xmlWellFormed(readPart(t, pkg, name)); err != nil {
			t.Errorf("%s is not well-formed: %v", name, err)
		}
	}
}
