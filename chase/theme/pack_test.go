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
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/theme/selector"
)

// pack_test.go holds this TRD's full Test-list (cases 1-10), added
// incrementally across the TRD's three tasks: Task 1 covers cases 1, 2,
// 3, 4, and the Task-1-scoped (embedding-level) half of case 8; Task 2
// extends this file with cases 5, 6, 7, the full Pack-level half of case
// 8, and 9; Task 3 extends it further with the cssdiff.Equal acceptance
// gate (case 10).

// stressThemeCSS and scaffoldThemeCSS (01-RESEARCH.md's synthetic stress
// theme and Marpit-scaffold fixtures) are already defined package-level in
// stylesheet_test.go (01-03) — reused here verbatim rather than
// redeclared, since this pipeline exercises the SAME fixtures 01-03's
// structural Parse tests already cover, one layer down (nesting-down-
// level, root-mark, scope, Pack), not a different synthetic theme.

// findRuleByProperty returns the first Rule in rules carrying a
// declaration named prop, and reports whether one was found — used
// throughout these tests to locate a rule by its distinguishing
// declaration rather than brittle selector-text/position matching.
func findRuleByProperty(rules []Rule, prop string) (Rule, bool) {
	for _, r := range rules {
		for _, d := range r.Declarations {
			if d.Property == prop {
				return r, true
			}
		}
	}
	return Rule{}, false
}

// selText renders a rule's selector back to text (trimmed) via
// chase/theme/selector.String, this package's canonical selector-to-text
// path throughout the pipeline (see pack.go's renderPacked).
func selText(r Rule) string {
	return strings.TrimSpace(selector.String(r.SelectorTokens))
}

// ---- Task 1: Tier-1 (Load) + nesting down-level + scaffold embed ----

// TestNestingImplicitAmpersandChildDownLevels covers Test-list case 1:
// `section { & h1 { color: red } }` down-levels to `section h1 { color:
// red }`.
func TestNestingImplicitAmpersandChildDownLevels(t *testing.T) {
	sheet, err := Parse(`section { & h1 { color: red; } }`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	flat, err := flattenNesting(sheet.Rules)
	if err != nil {
		t.Fatalf("flattenNesting: %v", err)
	}
	r, ok := findRuleByProperty(flat, "color")
	if !ok {
		t.Fatalf("flattened rules missing a color declaration: %#v", flat)
	}
	if got, want := selText(r), "section h1"; got != want {
		t.Fatalf("selector = %q, want %q", got, want)
	}
}

// TestNestingAmpersandCompoundDownLevels covers Test-list case 2:
// `section { &.lead { text-align: center } }` -> `section.lead { ... }`.
func TestNestingAmpersandCompoundDownLevels(t *testing.T) {
	sheet, err := Parse(`section { &.lead { text-align: center; } }`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	flat, err := flattenNesting(sheet.Rules)
	if err != nil {
		t.Fatalf("flattenNesting: %v", err)
	}
	r, ok := findRuleByProperty(flat, "text-align")
	if !ok {
		t.Fatalf("flattened rules missing a text-align declaration: %#v", flat)
	}
	if got, want := selText(r), "section.lead"; got != want {
		t.Fatalf("selector = %q, want %q", got, want)
	}
}

// TestNestingIsPseudoClassSelectorPassesThrough covers Test-list case 3: a
// plain (non-nested) `:is(h3, h4) + p` rule is a flatten passthrough —
// unmodified, and NOT mis-split by the comma inside :is()'s arguments.
func TestNestingIsPseudoClassSelectorPassesThrough(t *testing.T) {
	sheet, err := Parse(`:is(h3, h4) + p { margin-top: 0; }`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	flat, err := flattenNesting(sheet.Rules)
	if err != nil {
		t.Fatalf("flattenNesting: %v", err)
	}
	if len(flat) != 1 {
		t.Fatalf("len(flat) = %d, want 1 (no over-broad split)", len(flat))
	}
	if got, want := selText(flat[0]), ":is(h3, h4) + p"; got != want {
		t.Fatalf("selector = %q, want %q", got, want)
	}
}

// TestThemeLoadMarksRootSentinelTier1 covers Test-list case 4 (Tier1):
// `:root { --accent: teal }` -> the "section:marpit-root" sentinel is
// present after Load, and the stress theme's nesting/passthrough shapes
// all load successfully together.
func TestThemeLoadMarksRootSentinelTier1(t *testing.T) {
	th, err := Load(stressThemeCSS)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if th.Name != "stress" {
		t.Fatalf("Name = %q, want %q", th.Name, "stress")
	}
	r, ok := findRuleByProperty(th.Sheet.Rules, "--accent")
	if !ok {
		t.Fatalf("loaded theme missing the --accent declaration: %#v", th.Sheet.Rules)
	}
	if got, want := selText(r), "section:marpit-root"; got != want {
		t.Fatalf("selector = %q, want %q (sentinel not marked)", got, want)
	}
}

// TestScaffoldEmbeddedTextMatchesResearchVerbatim covers Test-list case 8
// at the Task-1 (embedding) level: scaffold.go's ScaffoldCSS and
// AdvancedBackgroundCSS constants parse cleanly (structurally valid CSS,
// no @theme header required) and carry the exact rule shapes
// 01-RESEARCH.md records. The full "prepended for the stress theme,
// skipped for the scaffold theme itself" pipeline behavior is covered at
// the Task-2/Pack level (see TestPackScaffoldPrependedForStressTheme and
// TestPackSkipsScaffoldForScaffoldThemeItself).
func TestScaffoldEmbeddedTextMatchesResearchVerbatim(t *testing.T) {
	scaffoldSheet, err := Parse(ScaffoldCSS)
	if err != nil {
		t.Fatalf("Parse(ScaffoldCSS): %v", err)
	}
	if len(scaffoldSheet.Rules) != 5 {
		t.Fatalf("len(ScaffoldCSS rules) = %d, want 5", len(scaffoldSheet.Rules))
	}
	if _, ok := findRuleByProperty(scaffoldSheet.Rules, "scroll-snap-align"); !ok {
		t.Fatalf("ScaffoldCSS missing the base section reset rule")
	}
	if _, ok := findRuleByProperty(scaffoldSheet.Rules, "will-change"); !ok {
		t.Fatalf("ScaffoldCSS missing the video::-webkit-media-controls rule")
	}

	advBgSheet, err := Parse(AdvancedBackgroundCSS)
	if err != nil {
		t.Fatalf("Parse(AdvancedBackgroundCSS): %v", err)
	}
	if len(advBgSheet.Rules) != 10 {
		t.Fatalf("len(AdvancedBackgroundCSS rules) = %d, want 10", len(advBgSheet.Rules))
	}
}

// ---- Task 2: Tier-2 (Pack) — scope, root (2nd pass + specificity),
// import, scaffold (full), advanced-background (full), pagination ----

// miniThemeCSS is a minimal single-rule theme used wherever a test only
// needs to observe scaffold/advanced-background/scope behavior without
// the stress theme's extra nesting/pseudo-class shapes.
const miniThemeCSS = `/**
 * @theme mini
 */
h1 { color: red; }
`

// TestPackScopesEverySelectorInlineSVG covers Test-list case 5: Pack
// scopes every rule to "div.marpit > svg > foreignObject > section" in
// inline-SVG render mode — both the packed theme's own rule and the
// prepended scaffold's own "section" rule.
func TestPackScopesEverySelectorInlineSVG(t *testing.T) {
	th, err := Load(miniThemeCSS)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet()
	ts.Add(th)

	out, err := ts.Pack("mini", PackOptions{InlineSVG: true})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if !strings.Contains(out, "div.marpit > svg > foreignObject > section h1") {
		t.Fatalf("Pack output missing scoped h1 rule; got:\n%s", out)
	}
	if !strings.Contains(out, "div.marpit > svg > foreignObject > section {") {
		t.Fatalf("Pack output missing scoped scaffold section rule; got:\n%s", out)
	}
}

// TestRootIncreasingSpecificityRunsAfterScoping covers Test-list case 6:
// the ":marpit-root" sentinel is rewritten to the final
// ":where(section):not([\20 root])" specificity-trick sequence AFTER
// selector-scoping — the packed output carries the fully-scoped,
// fully-boosted selector, never the raw ":marpit-root" sentinel or an
// un-replaced container placeholder.
func TestRootIncreasingSpecificityRunsAfterScoping(t *testing.T) {
	th, err := Load(stressThemeCSS)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet()
	ts.Add(th)

	out, err := ts.Pack("stress", PackOptions{InlineSVG: false})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	want := `div.marpit > :where(section):not([\20 root])`
	if !strings.Contains(out, want) {
		t.Fatalf("Pack output missing scoped+specificity-boosted root rule (want substring %q); got:\n%s", want, out)
	}
	if strings.Contains(out, ":marpit-root") {
		t.Fatalf("Pack output still contains the raw :marpit-root sentinel "+
			"(specificity pass didn't run, or ran before scoping):\n%s", out)
	}
	if strings.Contains(out, ":marpit-container") {
		t.Fatalf("Pack output still contains the unscoped placeholder sentinel:\n%s", out)
	}
}

// importBaseThemeCSS / importChildThemeCSS / importCyclicThemeCSS are
// Test-list case 7's fixtures: "child" recursively imports "base" via
// @import-theme, and "cyclic" imports itself.
const importBaseThemeCSS = `/**
 * @theme base
 */
h1 { color: blue; }
`

const importChildThemeCSS = `/**
 * @theme child
 */
@import-theme "base";
h2 { color: green; }
`

const importCyclicThemeCSS = `/**
 * @theme cyclic
 */
@import-theme "cyclic";
`

// TestImportThemeResolvesRecursivelyAndDetectsCycle covers Test-list case
// 7: @import-theme resolves recursively (child's Pack output carries both
// its own and its imported base theme's rules), and a self-referential
// @import-theme ERRORS via cycle detection rather than recursing forever.
func TestImportThemeResolvesRecursivelyAndDetectsCycle(t *testing.T) {
	baseTh, err := Load(importBaseThemeCSS)
	if err != nil {
		t.Fatalf("Load(base): %v", err)
	}
	childTh, err := Load(importChildThemeCSS)
	if err != nil {
		t.Fatalf("Load(child): %v", err)
	}

	ts := NewThemeSet()
	ts.Add(baseTh)
	ts.Add(childTh)

	out, err := ts.Pack("child", PackOptions{})
	if err != nil {
		t.Fatalf("Pack(child): %v", err)
	}
	if !strings.Contains(out, "color: blue") {
		t.Fatalf("Pack(child) missing imported base theme's rule; got:\n%s", out)
	}
	if !strings.Contains(out, "color: green") {
		t.Fatalf("Pack(child) missing its own rule; got:\n%s", out)
	}

	cyclicTh, err := Load(importCyclicThemeCSS)
	if err != nil {
		t.Fatalf("Load(cyclic): %v", err)
	}
	ts.Add(cyclicTh)
	if _, err := ts.Pack("cyclic", PackOptions{}); err == nil {
		t.Fatalf("Pack(cyclic): expected a circular @import-theme error, got nil")
	}
}

// TestPackScaffoldPrependedForStressTheme and
// TestPackSkipsScaffoldForScaffoldThemeItself together cover the full
// Pack-level half of Test-list case 8: scaffold.go's ScaffoldCSS is
// prepended for an ordinary theme, but Pack never double-prepends it when
// the theme being packed IS the scaffold theme itself.
func TestPackScaffoldPrependedForStressTheme(t *testing.T) {
	th, err := Load(stressThemeCSS)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet()
	ts.Add(th)

	out, err := ts.Pack("stress", PackOptions{})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if !strings.Contains(out, "scroll-snap-align") {
		t.Fatalf("Pack(stress) missing the prepended scaffold reset rule; got:\n%s", out)
	}
}

func TestPackSkipsScaffoldForScaffoldThemeItself(t *testing.T) {
	ts := NewThemeSet()

	out, err := ts.Pack(ScaffoldThemeName, PackOptions{})
	if err != nil {
		t.Fatalf("Pack(scaffold): %v", err)
	}
	if count := strings.Count(out, "scroll-snap-align"); count != 1 {
		t.Fatalf("Pack(scaffold) scaffold-reset rule count = %d, want exactly 1 "+
			"(scaffold must not be double-prepended onto itself); got:\n%s", count, out)
	}
}

// TestAdvancedBgInjectedMatchesResearchVerbatim covers Test-list case 9:
// the advanced-background static CSS block is injected verbatim when
// inline-SVG rendering is enabled, and omitted entirely otherwise.
func TestAdvancedBgInjectedMatchesResearchVerbatim(t *testing.T) {
	th, err := Load(miniThemeCSS)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet()
	ts.Add(th)

	svgOut, err := ts.Pack("mini", PackOptions{InlineSVG: true})
	if err != nil {
		t.Fatalf("Pack(InlineSVG=true): %v", err)
	}
	for _, want := range []string{
		"columns: initial !important",
		"margin-left: calc(100% - var(--marpit-advanced-background-split, 50%))",
		"background: transparent !important",
		"pointer-events: none !important",
	} {
		if !strings.Contains(svgOut, want) {
			t.Fatalf("Pack(InlineSVG=true) missing advanced-background declaration %q; got:\n%s", want, svgOut)
		}
	}

	nonSVGOut, err := ts.Pack("mini", PackOptions{InlineSVG: false})
	if err != nil {
		t.Fatalf("Pack(InlineSVG=false): %v", err)
	}
	if strings.Contains(nonSVGOut, "columns: initial !important") {
		t.Fatalf("Pack(InlineSVG=false) unexpectedly includes advanced-background CSS:\n%s", nonSVGOut)
	}
}

// TestPaginationNeutralizesNonDefaultAfterContent exercises pass_pagination.go
// directly (Test-list wording: "pagination comment-out non-marpit content
// on section::after"): a non-default `content` declaration on a
// "::after"-targeting rule is neutralized, its sibling declarations
// survive, and the scaffold's own default pagination content is
// untouched.
func TestPaginationNeutralizesNonDefaultAfterContent(t *testing.T) {
	authoredCSS := `/**
 * @theme paginated
 */
section::after { content: "custom"; color: red; }
`
	th, err := Load(authoredCSS)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet()
	ts.Add(th)

	out, err := ts.Pack("paginated", PackOptions{})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if strings.Contains(out, `content: "custom"`) {
		t.Fatalf("Pack output still contains a non-default ::after content declaration that should have been neutralized:\n%s", out)
	}
	if !strings.Contains(out, "color: red") {
		t.Fatalf("Pack output dropped an unrelated declaration on the same ::after rule:\n%s", out)
	}
	if !strings.Contains(out, "content: attr(data-marpit-pagination)") {
		t.Fatalf("Pack output missing the scaffold's default pagination content:\n%s", out)
	}
}
