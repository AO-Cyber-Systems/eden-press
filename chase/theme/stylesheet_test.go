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

// --- Task 2: Parse (css token-stream -> Stylesheet), THEME-01 -------------
//
// The tests below drive parse.go's Parse function directly (rather than
// hand-building the model, as the tests above do), covering Test-list
// cases 1, 2, 3 (again, now via the real tokenizer) and case 4 (at-rule
// capture). Parse is purely structural — it does NOT require or interpret
// @theme metadata (that's meta.go's ParseMeta/ParseTheme, Task 3) — so
// these fixtures intentionally carry no theme header.

// TestParseSingleRule covers Test-list case 1 via the real tokenizer.
func TestParseSingleRule(t *testing.T) {
	sheet, err := Parse(`section { color: red }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(sheet.Rules))
	}
	r := sheet.Rules[0]
	if got := tokensText(r.SelectorTokens); got != "section" {
		t.Fatalf("selector = %q, want %q", got, "section")
	}
	if len(r.Declarations) != 1 {
		t.Fatalf("len(Declarations) = %d, want 1", len(r.Declarations))
	}
	d := r.Declarations[0]
	if d.Property != "color" || tokensText(d.Value) != "red" {
		t.Fatalf("Declaration = {%q: %q}, want {color: red}", d.Property, tokensText(d.Value))
	}
}

// TestParseMultiSelectorRetainsComma covers Test-list case 2 via the real
// tokenizer: `h1, h2 { margin: 0 }` must parse into ONE Rule whose
// SelectorTokens still contain the comma (walkable), not two Rules.
func TestParseMultiSelectorRetainsComma(t *testing.T) {
	sheet, err := Parse(`h1, h2 { margin: 0 }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1 (selector list must not be pre-split)", len(sheet.Rules))
	}
	foundComma := false
	for _, tkn := range sheet.Rules[0].SelectorTokens {
		if tkn.TokenType == css.CommaToken {
			foundComma = true
		}
	}
	if !foundComma {
		t.Fatalf("SelectorTokens = %v, want a retained Comma token", sheet.Rules[0].SelectorTokens)
	}
}

// TestParseNestedRuleDepth covers Test-list case 3 via the real tokenizer:
// `section { & h1 { color: red } }` must capture the inner ruleset as a
// Child with NestingDepth=1, NOT flattened.
func TestParseNestedRuleDepth(t *testing.T) {
	sheet, err := Parse(`section { & h1 { color: red } }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1 top-level rule", len(sheet.Rules))
	}
	parent := sheet.Rules[0]
	if len(parent.Children) != 1 {
		t.Fatalf("len(Children) = %d, want 1", len(parent.Children))
	}
	child := parent.Children[0]
	if child.NestingDepth != 1 {
		t.Fatalf("child.NestingDepth = %d, want 1", child.NestingDepth)
	}
	if len(child.Declarations) != 1 || child.Declarations[0].Property != "color" {
		t.Fatalf("child.Declarations = %+v, want [{color red}]", child.Declarations)
	}
}

// TestParseAtRuleCapture covers Test-list case 4: `@import`/`@import-theme`
// statements are RECORDED as AtRule entries, never resolved.
func TestParseAtRuleCapture(t *testing.T) {
	sheet, err := Parse(`@import "x.css"; @import-theme "y";`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Atoms) != 2 {
		t.Fatalf("len(Atoms) = %d, want 2", len(sheet.Atoms))
	}
	if sheet.Atoms[0].Name != "import" || sheet.Atoms[0].Prelude != `"x.css"` {
		t.Fatalf("Atoms[0] = %+v, want {import \"x.css\"}", sheet.Atoms[0])
	}
	if sheet.Atoms[1].Name != "import-theme" || sheet.Atoms[1].Prelude != `"y"` {
		t.Fatalf("Atoms[1] = %+v, want {import-theme \"y\"}", sheet.Atoms[1])
	}
	if len(sheet.Rules) != 0 {
		t.Fatalf("len(Rules) = %d, want 0 (no rulesets in this fixture)", len(sheet.Rules))
	}
}

// scaffoldThemeCSS is 01-RESEARCH.md's "Scaffold theme CSS" fixture,
// verbatim — chase/theme's own baseline reset, safe to hardcode per the
// TRD's research_context.
const scaffoldThemeCSS = `section { width: 1280px; height: 720px; box-sizing: border-box; overflow: hidden; position: relative; scroll-snap-align: center center; -webkit-text-size-adjust: 100%; text-size-adjust: 100%; }
section::after { bottom: 0; content: attr(data-marpit-pagination); padding: inherit; pointer-events: none; position: absolute; right: 0; }
section:not([data-marpit-pagination])::after { display: none; }
:where(h1) { font-size: 2em; margin-block: 0.67em; }
video::-webkit-media-controls { will-change: transform; }`

// TestParseScaffoldThemeRuleCount covers Task 2's verify note: parsing the
// RESEARCH scaffold CSS yields the expected rule count (5 top-level rules,
// no nesting, no at-rules).
func TestParseScaffoldThemeRuleCount(t *testing.T) {
	sheet, err := Parse(scaffoldThemeCSS)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Rules) != 5 {
		t.Fatalf("len(Rules) = %d, want 5", len(sheet.Rules))
	}
	if len(sheet.Atoms) != 0 {
		t.Fatalf("len(Atoms) = %d, want 0", len(sheet.Atoms))
	}
}

// stressThemeCSS is 01-RESEARCH.md's "Recommended synthetic stress theme"
// fixture, verbatim — exercises :root, implicit-& nesting (2 levels deep in
// one branch), :where()/:is(), and ::backdrop. Parse is purely structural,
// so the leading `@theme`/`@size` metadata comment is simply skipped here
// (meta.go/Task 3 owns interpreting it).
const stressThemeCSS = `/**
 * @theme stress
 * @size 4:3
 */
:root { --accent: teal; }
section {
  & h1, & h2 { color: var(--accent); }
  &.lead { text-align: center; }
}
:where(h1, h2) { margin: 0; }
:is(h3, h4) + p { margin-top: 0; }
::backdrop { background: #048; }`

// TestParseStressThemeRuleCount covers Task 2's verify note: parsing the
// stress theme yields the expected rule count and nesting shape.
func TestParseStressThemeRuleCount(t *testing.T) {
	sheet, err := Parse(stressThemeCSS)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Rules) != 5 {
		t.Fatalf("len(Rules) = %d, want 5 top-level rules (:root, section, :where, :is, ::backdrop)", len(sheet.Rules))
	}
	// sheet.Rules[1] is `section`, with 2 nested children (`& h1, & h2`
	// and `&.lead`), each at NestingDepth 1.
	section := sheet.Rules[1]
	if got := tokensText(section.SelectorTokens); got != "section" {
		t.Fatalf("Rules[1] selector = %q, want %q", got, "section")
	}
	if len(section.Children) != 2 {
		t.Fatalf("len(section.Children) = %d, want 2", len(section.Children))
	}
	for i, c := range section.Children {
		if c.NestingDepth != 1 {
			t.Fatalf("section.Children[%d].NestingDepth = %d, want 1", i, c.NestingDepth)
		}
	}
	// CustomPropertyGrammar (`--accent: teal;`) inside :root must also be
	// captured as a Declaration.
	root := sheet.Rules[0]
	if len(root.Declarations) != 1 || root.Declarations[0].Property != "--accent" {
		t.Fatalf("root.Declarations = %+v, want a single --accent custom property", root.Declarations)
	}
}
