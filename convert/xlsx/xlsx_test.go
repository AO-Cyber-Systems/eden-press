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

package xlsx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

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
	t.Fatalf("part %q not present; have %v", name, partNames(t, pkg))
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

func tableDoc(blocks ...model.Block) *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections:      []model.Section{{ID: 1, Blocks: blocks}},
	}
}

var metricsTable = model.Block{
	Kind:    model.BlockTable,
	Headers: []string{"Metric", "Q2", "Q3"},
	Rows: [][]string{
		{"p95 latency", "840", "550"},
		{"Error rate", "1.2", "0.4"},
	},
}

// TestToXLSXNilDoc: a nil document is a caller error, not a panic.
func TestToXLSXNilDoc(t *testing.T) {
	if _, err := ToXLSX(nil, Options{}); err == nil {
		t.Fatal("ToXLSX(nil) = nil error, want an error")
	}
}

// TestPackageGraph: the minimal OPC graph Excel requires is present and every
// part is declared in [Content_Types].xml.
func TestPackageGraph(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(metricsTable), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	for _, want := range []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"xl/workbook.xml",
		"xl/_rels/workbook.xml.rels",
		"xl/worksheets/sheet1.xml",
		"xl/styles.xml",
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
	for _, want := range []string{"/xl/workbook.xml", "/xl/worksheets/sheet1.xml", "/xl/styles.xml"} {
		if !strings.Contains(ct, want) {
			t.Errorf("[Content_Types].xml has no Override for %q: %s", want, ct)
		}
	}
}

// TestDeterminism: same document twice, byte-identical package.
func TestDeterminism(t *testing.T) {
	d := tableDoc(metricsTable)
	a, err := ToXLSX(d, Options{})
	if err != nil {
		t.Fatalf("ToXLSX #1: %v", err)
	}
	b, err := ToXLSX(d, Options{})
	if err != nil {
		t.Fatalf("ToXLSX #2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("ToXLSX is not deterministic: %d vs %d bytes", len(a), len(b))
	}
}

// TestTableToCells: a table's header and body land in the right cells, with A1
// references that match their row/column position.
func TestTableToCells(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(metricsTable), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	sheet := readPart(t, pkg, "xl/worksheets/sheet1.xml")

	for _, want := range []string{`r="A1"`, `r="B1"`, `r="C1"`, `r="A2"`, `r="C3"`} {
		if !strings.Contains(sheet, want) {
			t.Errorf("missing cell reference %s: %s", want, sheet)
		}
	}
	for _, want := range []string{"Metric", "Q2", "Q3", "p95 latency", "Error rate"} {
		if !strings.Contains(sheet, want) {
			t.Errorf("missing cell text %q: %s", want, sheet)
		}
	}
}

// TestNumericCellsAreNumbers is the point of exporting a spreadsheet at all: a
// workbook whose numbers are text cannot be summed, charted or sorted. Numeric
// cell values must be emitted as typed numbers, and non-numeric text must not.
func TestNumericCellsAreNumbers(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(metricsTable), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	sheet := readPart(t, pkg, "xl/worksheets/sheet1.xml")

	// 840 is numeric -> <v>840</v> with no t="inlineStr".
	if !strings.Contains(sheet, "<v>840</v>") {
		t.Errorf("numeric cell 840 not emitted as a number: %s", sheet)
	}
	if !strings.Contains(sheet, "<v>0.4</v>") {
		t.Errorf("decimal cell 0.4 not emitted as a number: %s", sheet)
	}
	// Header text must be an inline string, never a number.
	if !strings.Contains(sheet, `t="inlineStr"`) {
		t.Errorf("text cells are not inline strings: %s", sheet)
	}
	// "p95 latency" must NOT be coerced to a number.
	if strings.Contains(sheet, "<v>p95 latency</v>") {
		t.Errorf("text was emitted as a numeric value: %s", sheet)
	}
}

// TestNumericDetection pins the boundary cases directly: what counts as a
// number must not swallow things that merely start with a digit.
func TestNumericDetection(t *testing.T) {
	for _, tc := range []struct {
		in      string
		numeric bool
	}{
		{"840", true},
		{"0.4", true},
		{"-12.5", true},
		{"+3", true},
		{"1e3", true},
		{"", false},
		{"1.2%", false},  // a percentage is not a bare number
		{"550ms", false}, // a unit suffix is not a number
		{"p95", false},
		{"1,200", false}, // thousands separators are locale-dependent
		{"  7  ", true},  // surrounding whitespace is not meaningful
		{"NaN", false},   // parses as a float in Go, but is not a spreadsheet number
		{"Inf", false},
	} {
		if got := isNumeric(tc.in); got != tc.numeric {
			t.Errorf("isNumeric(%q) = %v, want %v", tc.in, got, tc.numeric)
		}
	}
}

// TestColumnRef covers the A..Z -> AA rollover, the classic off-by-one in
// spreadsheet writers.
func TestColumnRef(t *testing.T) {
	for _, tc := range []struct {
		idx  int
		want string
	}{
		{0, "A"}, {1, "B"}, {25, "Z"}, {26, "AA"}, {27, "AB"}, {51, "AZ"}, {52, "BA"}, {701, "ZZ"}, {702, "AAA"},
	} {
		if got := columnRef(tc.idx); got != tc.want {
			t.Errorf("columnRef(%d) = %q, want %q", tc.idx, got, tc.want)
		}
	}
}

// TestOneSheetPerSection: each section carrying a table becomes its own sheet;
// a section with no table is skipped rather than producing an empty sheet.
func TestOneSheetPerSection(t *testing.T) {
	d := &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Metrics"},
				metricsTable,
			}},
			{ID: 2, Blocks: []model.Block{
				{Kind: model.BlockParagraph, Text: "prose only, no table"},
			}},
			{ID: 3, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Costs"},
				{Kind: model.BlockTable, Headers: []string{"Item"}, Rows: [][]string{{"x"}}},
			}},
		},
	}
	pkg, err := ToXLSX(d, Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	names := partNames(t, pkg)
	sheets := 0
	for _, n := range names {
		if strings.HasPrefix(n, "xl/worksheets/sheet") {
			sheets++
		}
	}
	if sheets != 2 {
		t.Errorf("worksheets = %d, want 2 (the table-free section must be skipped): %v", sheets, names)
	}

	wb := readPart(t, pkg, "xl/workbook.xml")
	if !strings.Contains(wb, `name="Metrics"`) || !strings.Contains(wb, `name="Costs"`) {
		t.Errorf("sheets not named from their section headings: %s", wb)
	}
}

// TestSheetNameSanitization: Excel rejects a workbook whose sheet names break
// its rules, so names are sanitized, length-capped and de-duplicated.
func TestSheetNameSanitization(t *testing.T) {
	long := strings.Repeat("x", 40)
	d := &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: `A/B:C\D?E*F[G]H`},
				metricsTable,
			}},
			{ID: 2, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: long},
				metricsTable,
			}},
			{ID: 3, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: long},
				metricsTable,
			}},
		},
	}
	pkg, err := ToXLSX(d, Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	wb := readPart(t, pkg, "xl/workbook.xml")

	for _, bad := range []string{"/", ":", "\\", "?", "*", "[", "]"} {
		if strings.Contains(wb, `name="`) && strings.Contains(extractNames(wb), bad) {
			t.Errorf("sheet name retains the illegal character %q: %s", bad, wb)
		}
	}
	for _, n := range strings.Split(extractNames(wb), "\x00") {
		if len([]rune(n)) > 31 {
			t.Errorf("sheet name %q exceeds Excel's 31-character limit", n)
		}
	}
	// The two identical over-long names must not collide.
	names := strings.Split(extractNames(wb), "\x00")
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate sheet name %q (Excel rejects the workbook): %v", n, names)
		}
		seen[n] = true
	}
}

// extractNames pulls the name="..." attribute values out of workbook.xml,
// NUL-joined so the caller can split them unambiguously.
func extractNames(wb string) string {
	var out []string
	for _, chunk := range strings.Split(wb, `<sheet name="`)[1:] {
		if i := strings.Index(chunk, `"`); i >= 0 {
			out = append(out, chunk[:i])
		}
	}
	return strings.Join(out, "\x00")
}

// TestNoTablesFallback: a workbook must contain at least one sheet, so a
// document with no tables still produces a valid (if sparse) package rather
// than an empty workbook Excel would reject.
func TestNoTablesFallback(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(
		model.Block{Kind: model.BlockHeading, Level: 1, Text: "Notes"},
		model.Block{Kind: model.BlockParagraph, Text: "no tables here"},
	), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	sheet := readPart(t, pkg, "xl/worksheets/sheet1.xml")
	if !strings.Contains(sheet, "no tables here") {
		t.Errorf("fallback sheet did not carry the document's text: %s", sheet)
	}
}

// TestRaggedRows: rows shorter than the header are padded; the sheet stays
// rectangular.
func TestRaggedRows(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(model.Block{
		Kind:    model.BlockTable,
		Headers: []string{"A", "B", "C"},
		Rows:    [][]string{{"1"}, {"1", "2", "3", "4"}},
	}), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	sheet := readPart(t, pkg, "xl/worksheets/sheet1.xml")
	// The overflowing 4th value must not appear in a D column.
	if strings.Contains(sheet, `r="D`) {
		t.Errorf("overflow cell created a D column beyond the header width: %s", sheet)
	}
}

// TestXMLEscaping: cell text is untrusted and must not be able to corrupt the
// sheet.
func TestXMLEscaping(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(model.Block{
		Kind:    model.BlockTable,
		Headers: []string{`a & b <c>`},
		Rows:    [][]string{{`"quoted" <row/>`}},
	}), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}
	sheet := readPart(t, pkg, "xl/worksheets/sheet1.xml")
	if !strings.Contains(sheet, "&amp;") || !strings.Contains(sheet, "&lt;") {
		t.Errorf("cell text not XML-escaped: %s", sheet)
	}
	if err := xmlWellFormed(sheet); err != nil {
		t.Errorf("sheet is not well-formed after escaping: %v", err)
	}
}

// TestAllPartsWellFormed runs every emitted part through a real XML decoder.
func TestAllPartsWellFormed(t *testing.T) {
	pkg, err := ToXLSX(tableDoc(metricsTable), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
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
