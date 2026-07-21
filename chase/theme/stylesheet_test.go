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

package theme

import (
	"testing"

	"github.com/tdewolff/parse/v2/css"
)

// tok builds a css.Token literal for hand-constructed model fixtures below —
// these tests exercise the Stylesheet/Rule/Declaration MODEL directly (no
// Parse call: parse.go's grammar-stream builder is a separate task/file),
// proving the model's structural shape (token preservation, nesting depth)
// and its String() serializer independently of how the model gets built.
func tok(tt css.TokenType, data string) css.Token {
	return css.Token{TokenType: tt, Data: []byte(data)}
}

// TestStylesheetModelSingleRule covers Test-list case 1: a single ruleset
// with one declaration serializes back to canonical CSS text and preserves
// its Rule/Declaration counts.
func TestStylesheetModelSingleRule(t *testing.T) {
	r := Rule{
		SelectorTokens: []css.Token{tok(css.IdentToken, "section")},
		Declarations: []Declaration{
			{Property: "color", Value: []css.Token{tok(css.IdentToken, "red")}},
		},
	}
	sheet := Stylesheet{Rules: []Rule{r}}

	if len(sheet.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(sheet.Rules))
	}
	if len(sheet.Rules[0].Declarations) != 1 {
		t.Fatalf("len(Declarations) = %d, want 1", len(sheet.Rules[0].Declarations))
	}
	want := "section { color: red; }"
	if got := sheet.String(); got != want {
		t.Fatalf("Stylesheet.String() = %q, want %q", got, want)
	}
}

// TestStylesheetModelMultiSelectorRetainsComma covers Test-list case 2: a
// multi-selector ruleset (`h1, h2 { margin: 0 }`) keeps its selector as ONE
// walkable token list containing the comma — it must NOT be pre-split into
// two Rules, since chase/theme/selector (THEME-04) needs to walk the whole
// comma-separated list itself.
func TestStylesheetModelMultiSelectorRetainsComma(t *testing.T) {
	r := Rule{
		SelectorTokens: []css.Token{
			tok(css.IdentToken, "h1"),
			tok(css.CommaToken, ","),
			tok(css.IdentToken, "h2"),
		},
		Declarations: []Declaration{
			{Property: "margin", Value: []css.Token{tok(css.NumberToken, "0")}},
		},
	}
	sheet := Stylesheet{Rules: []Rule{r}}

	if len(sheet.Rules) != 1 {
		t.Fatalf("multi-selector list must stay ONE rule (walkable token list), got %d rules", len(sheet.Rules))
	}
	foundComma := false
	for _, tkn := range sheet.Rules[0].SelectorTokens {
		if tkn.TokenType == css.CommaToken {
			foundComma = true
		}
	}
	if !foundComma {
		t.Fatalf("SelectorTokens lost the comma — selector must remain walkable, not pre-split")
	}
	want := "h1,h2 { margin: 0; }"
	if got := sheet.String(); got != want {
		t.Fatalf("Stylesheet.String() = %q, want %q", got, want)
	}
}

// TestStylesheetModelNestingDepth covers Test-list case 3: a nested rule
// (`section { & h1 { color: red } }`) is captured as a Child of its parent
// Rule with NestingDepth=1 — NOT flattened into a second top-level Rule.
// Down-leveling nesting into a flat selector is TRD 01-04's job, not this
// model's.
func TestStylesheetModelNestingDepth(t *testing.T) {
	child := Rule{
		SelectorTokens: []css.Token{
			tok(css.DelimToken, "&"),
			tok(css.WhitespaceToken, " "),
			tok(css.IdentToken, "h1"),
		},
		Declarations: []Declaration{
			{Property: "color", Value: []css.Token{tok(css.IdentToken, "red")}},
		},
		NestingDepth: 1,
	}
	parent := Rule{
		SelectorTokens: []css.Token{tok(css.IdentToken, "section")},
		Children:       []Rule{child},
	}
	sheet := Stylesheet{Rules: []Rule{parent}}

	if len(sheet.Rules) != 1 {
		t.Fatalf("nested rule must NOT be flattened into a second top-level rule, got %d top-level rules", len(sheet.Rules))
	}
	if len(sheet.Rules[0].Children) != 1 {
		t.Fatalf("expected 1 nested child, got %d", len(sheet.Rules[0].Children))
	}
	if got := sheet.Rules[0].Children[0].NestingDepth; got != 1 {
		t.Fatalf("Children[0].NestingDepth = %d, want 1", got)
	}
	want := "section { & h1 { color: red; } }"
	if got := sheet.String(); got != want {
		t.Fatalf("Stylesheet.String() = %q, want %q", got, want)
	}
}

// TestModelDeclarationImportantString exercises the !important round-trip
// through Declaration.String().
func TestModelDeclarationImportantString(t *testing.T) {
	d := Declaration{
		Property:  "color",
		Value:     []css.Token{tok(css.IdentToken, "red")},
		Important: true,
	}
	want := "color: red !important"
	if got := d.String(); got != want {
		t.Fatalf("Declaration.String() = %q, want %q", got, want)
	}
}

// TestModelAtRuleString exercises AtRule.String()'s round-trip of a
// RECORDED (not resolved) at-rule statement.
func TestModelAtRuleString(t *testing.T) {
	a := AtRule{Name: "import", Prelude: `"x.css"`}
	want := `@import "x.css"`
	if got := a.String(); got != want {
		t.Fatalf("AtRule.String() = %q, want %q", got, want)
	}
}

// TestModelMetaResolveSizeDefault exercises Meta.ResolveSize's 1280x720
// default fallback, both when no @size table exists at all and when a name
// isn't found in a populated table.
func TestModelMetaResolveSizeDefault(t *testing.T) {
	var m Meta
	if w, h := m.ResolveSize(""); w != 1280 || h != 720 {
		t.Fatalf("ResolveSize(\"\") on empty Meta = (%d,%d), want (1280,720)", w, h)
	}

	m.Sizes = map[string]Size{"16:9": {Name: "16:9", WidthPx: 1280, HeightPx: 720}}
	if w, h := m.ResolveSize("missing"); w != 1280 || h != 720 {
		t.Fatalf("ResolveSize(\"missing\") = (%d,%d), want default (1280,720)", w, h)
	}
	if w, h := m.ResolveSize("16:9"); w != 1280 || h != 720 {
		t.Fatalf("ResolveSize(\"16:9\") = (%d,%d), want (1280,720)", w, h)
	}
}
