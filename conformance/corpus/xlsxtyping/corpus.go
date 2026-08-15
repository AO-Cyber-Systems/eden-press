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

// Package xlsxtyping publishes the ONE cell-typing rule two independent XLSX
// writers must agree on: eden-press's convert/xlsx (input: markdown table
// text) and AODex's workbook_exporter.dart (input: a parsed 6-way CellValue
// union).
//
// The writers are deliberately NOT unified -- routing a parsed workbook
// through convert/xlsx would stringify every value and re-infer its type, and
// strconv.ParseFloat("007", 64) succeeds, so Excel would read the identifier
// "007" as the number 7. Unifying also loses dates, booleans and formulas: a
// formula cell would arrive as the literal string "=SUM(A1:A5)".
//
// What IS shared is the classification rule, published here as DATA rather
// than prose. Three docstrings in convert/xlsx and its model had already
// drifted from their code before this package existed; a prose contract would
// have been the fourth. The point is to make the rule a thing that FAILS, not
// a thing that is written down.
//
// The Dart rule wins for user data: its input is a cell a person authored, and
// preserving what they typed outranks convenience.
//
// # Shape, and why it differs from the sibling corpus package
//
// The sibling package conformance/corpus loads a directory of render fixtures
// (input.md + options.json + expected.html) because its cases are documents.
// These cases are string->type pairs, so they live in Go directly:
//
//   - Cases and StructuralCases are the source of truth. Any Go consumer --
//     including AODex, once its go.mod can reach this module -- imports them.
//     No copying, no checksum, no drift possible.
//   - cases.json is a GENERATED projection for suites that are not Go (the
//     Dart one). Regenerate with `go generate ./conformance/corpus/xlsxtyping`.
//     Never hand-edit it: TestCorpusJSONIsCurrent asserts the committed file is
//     byte-identical to what CasesJSON returns right now, exactly as
//     `make openapi-verify` does for an API spec. A hand-maintained JSON is a
//     second source of truth wearing a fixture's clothes.
package xlsxtyping

import (
	"bytes"
	"encoding/json"
)

//go:generate go run ./internal/gencases

// SchemaID identifies the JSON projection's shape. A consumer that pins this
// string finds out at load time that the corpus moved under it, rather than
// silently reading zero cases and passing.
const SchemaID = "eden-press.xlsxtyping/v1"

// Class is how a cell's raw text must be written.
type Class string

const (
	// ClassNumber is a real spreadsheet number: a typed value that can be
	// summed, sorted and charted.
	ClassNumber Class = "number"
	// ClassText is text, whatever it looks like.
	ClassText Class = "text"
	// ClassEmpty is an absent value, distinct from a cell holding "".
	ClassEmpty Class = "empty"
)

// Case is one agreed classification, with the reason it is agreed. The reason
// is not decoration: every case here exists because some implementation got it
// wrong or could, and the reason is what stops a future reader "simplifying"
// the case away.
type Case struct {
	Input  string `json:"input"`
	Class  Class  `json:"class"`
	Reason string `json:"reason"`
}

// Cases is the agreed cell-typing rule.
//
// The rule itself, stated once: trim surrounding whitespace; an empty result
// is ClassEmpty; a value that is longer than one character, starts with "0" or
// "+", and contains no "." is ClassText (it is an authored identifier, not a
// quantity); "NaN"/"Inf" are ClassText; otherwise ClassNumber when it parses
// as a float and ClassText when it does not.
var Cases = []Case{
	{"007", ClassText, "an identifier, not the number 7 -- silently storing it as 7 is the single most common way a spreadsheet tool corrupts data"},
	{"+3", ClassText, "a leading sign on an integer reads as authored input; eden-press previously pinned this as a number and AODex as text -- this case is the reconciliation (decision D6: the Dart rule wins for user data)"},
	{"0.4", ClassNumber, "a leading zero followed by a decimal point is a fraction, not an identifier -- do NOT let the 007 rule swallow it, or every fractional value in every document becomes text"},
	{"+3.5", ClassNumber, "the same escape hatch on the sign branch: a signed value with a decimal point is a quantity, so the guard is (leading 0 OR leading +) AND no dot, never either half alone"},
	{"0", ClassNumber, "single-character zero is the number zero -- the identifier guard is length > 1"},
	{"840", ClassNumber, ""},
	{"-12.5", ClassNumber, ""},
	{"1e3", ClassNumber, ""},
	{"  7  ", ClassNumber, "surrounding whitespace is not meaningful"},
	{"", ClassEmpty, "empty is distinct from a cell holding an empty string"},
	{"1,200", ClassText, "thousands separators are locale-dependent; guessing corrupts"},
	{"1.2%", ClassText, "a percentage needs a number format, not a bare value"},
	{"550ms", ClassText, "a unit suffix is not a number"},
	{"p95", ClassText, ""},
	{"NaN", ClassText, "parses as a float in Go and as double.nan in Dart, but is not a spreadsheet number -- a <v>NaN</v> cell cannot be evaluated"},
	{"Inf", ClassText, "as NaN"},
	{"true", ClassText, "convert/xlsx has no boolean cell type; AODex's parser does and keeps it -- the two inputs differ, and this case records that the TEXT path is what a markdown table gets"},
}

// StructuralCase is a workbook-shape agreement rather than a cell
// classification. Kept in its own slice so it is never fed to a typing
// assertion by accident.
type StructuralCase struct {
	// ID names the case for a failure message.
	ID string `json:"id"`
	// Input describes the input precisely enough to reconstruct it.
	Input string `json:"input"`
	// Expect is the required outcome, in words -- these cases are about
	// workbook structure, which has no single scalar answer to pin.
	Expect string `json:"expect"`
	// Reason is why the case exists; as with Case.Reason, load-bearing.
	Reason string `json:"reason"`
}

// StructuralCases are the shape agreements both writers must hold.
var StructuralCases = []StructuralCase{
	{
		ID:     "sheet-name-over-31-runes",
		Input:  `a section heading of 40 non-ASCII runes, e.g. strings.Repeat("é", 40)`,
		Expect: "the sheet name is truncated to exactly 31 RUNES (not 31 bytes), stays valid UTF-8, and the workbook opens",
		Reason: "Excel's limit is characters, and slicing a multi-byte name mid-rune emits invalid UTF-8, which Excel reports as a corrupt file rather than as a long name",
	},
	{
		ID:     "column-rollover-z-to-aa",
		Input:  "the 27th column, 0-based index 26",
		Expect: `the column reference is "AA"; index 51 is "AZ" and index 52 is "BA"`,
		Reason: `spreadsheet columns are bijective base-26 with no zero digit; the classic writer bug takes an ordinary remainder and emits "BA" at the Z->AA rollover`,
	},
	{
		ID:     "ragged-row",
		Input:  "a table with 3 headers whose body rows hold 1 and 4 cells",
		Expect: "the short row is padded to 3 cells and the long row truncated to 3; no cell is emitted in column D",
		Reason: "the model preserves rows exactly as authored, so the exporter must decide; a ragged sheet puts values under the wrong headers, which is worse than dropping the overflow",
	},
}

// Corpus is the JSON projection's top-level shape.
type Corpus struct {
	Schema     string           `json:"schema"`
	Cells      []Case           `json:"cells"`
	Structural []StructuralCase `json:"structural"`
}

// CasesJSON renders the corpus to the exact bytes cases.json must contain.
//
// Determinism is the whole mechanism: slice order is the declaration order,
// key order is the struct field order, the indent is fixed, and HTML escaping
// is off so a reason mentioning a <v> element stays readable instead of
// turning into <v>. Relax any of those and TestCorpusJSONIsCurrent
// flaps.
func CasesJSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(Corpus{
		Schema:     SchemaID,
		Cells:      Cases,
		Structural: StructuralCases,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
