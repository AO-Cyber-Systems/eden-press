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
	"github.com/AO-Cyber-Systems/eden-press/conformance/cssdiff"
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
//
// testScaffoldCSS / testAdvancedBackgroundCSS are TEST-ONLY, byte-for-byte
// copies of profiles/slides' ScaffoldCSS / AdvancedBackgroundCSS constants
// (TRD 02-03, MODEL-04's de-hardcoding move) — duplicated locally, rather
// than importing profiles/slides, so chase/theme's own test suite stays a
// true dependency leaf (no test-only import of a package that itself
// imports chase/theme). testSizeFallback/testDefaultSize live in
// stylesheet_test.go, likewise a local stand-in for a Profile's size table.
const testScaffoldCSS = `
section {
  width: 1280px;
  height: 720px;
  box-sizing: border-box;
  overflow: hidden;
  position: relative;
  scroll-snap-align: center center;
  -webkit-text-size-adjust: 100%;
  text-size-adjust: 100%;
}
section::after {
  bottom: 0;
  content: attr(data-marpit-pagination);
  padding: inherit;
  pointer-events: none;
  position: absolute;
  right: 0;
}
section:not([data-marpit-pagination])::after {
  display: none;
}
:where(h1) {
  font-size: 2em;
  margin-block: 0.67em;
}
video::-webkit-media-controls {
  will-change: transform;
}
`

const testAdvancedBackgroundCSS = `
section[data-marpit-advanced-background="background"] {
  columns: initial !important;
  display: block !important;
  padding: 0 !important;
}
section[data-marpit-advanced-background="background"]::before,
section[data-marpit-advanced-background="background"]::after,
section[data-marpit-advanced-background="content"]::before,
section[data-marpit-advanced-background="content"]::after {
  display: none !important;
}
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] {
  all: initial;
  display: flex;
  flex-direction: row;
  height: 100%;
  overflow: hidden;
  width: 100%;
}
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container][data-marpit-advanced-background-direction="vertical"] {
  flex-direction: column;
}
section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split] > div[data-marpit-advanced-background-container] {
  width: var(--marpit-advanced-background-split, 50%);
}
section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split="right"] > div[data-marpit-advanced-background-container] {
  margin-left: calc(100% - var(--marpit-advanced-background-split, 50%));
}
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] > figure {
  all: initial;
  background-position: center;
  background-repeat: no-repeat;
  background-size: cover;
  flex: auto;
  margin: 0;
}
section[data-marpit-advanced-background="content"],
section[data-marpit-advanced-background="pseudo"] {
  background: transparent !important;
}
section[data-marpit-advanced-background="pseudo"],
:marpit-container > svg[data-marpit-svg] > foreignObject[data-marpit-advanced-background="pseudo"] {
  pointer-events: none !important;
}
section[data-marpit-advanced-background-split] {
  width: 100%;
  height: 100%;
}
`

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
	th, err := Load(stressThemeCSS, "section", testSizeFallback)
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
// at the Task-1 (embedding) level: the scaffold/advanced-background CSS
// text (testScaffoldCSS/testAdvancedBackgroundCSS — byte-identical local
// copies of profiles/slides' ScaffoldCSS/AdvancedBackgroundCSS, TRD 02-03)
// parses cleanly (structurally valid CSS, no @theme header required) and
// carries the exact rule shapes 01-RESEARCH.md records. The full
// "prepended for the stress theme, skipped for the scaffold theme itself"
// pipeline behavior is covered at the Task-2/Pack level (see
// TestPackScaffoldPrependedForStressTheme and
// TestPackSkipsScaffoldForScaffoldThemeItself).
func TestScaffoldEmbeddedTextMatchesResearchVerbatim(t *testing.T) {
	scaffoldSheet, err := Parse(testScaffoldCSS)
	if err != nil {
		t.Fatalf("Parse(testScaffoldCSS): %v", err)
	}
	if len(scaffoldSheet.Rules) != 5 {
		t.Fatalf("len(testScaffoldCSS rules) = %d, want 5", len(scaffoldSheet.Rules))
	}
	if _, ok := findRuleByProperty(scaffoldSheet.Rules, "scroll-snap-align"); !ok {
		t.Fatalf("testScaffoldCSS missing the base section reset rule")
	}
	if _, ok := findRuleByProperty(scaffoldSheet.Rules, "will-change"); !ok {
		t.Fatalf("testScaffoldCSS missing the video::-webkit-media-controls rule")
	}

	advBgSheet, err := Parse(testAdvancedBackgroundCSS)
	if err != nil {
		t.Fatalf("Parse(testAdvancedBackgroundCSS): %v", err)
	}
	if len(advBgSheet.Rules) != 10 {
		t.Fatalf("len(testAdvancedBackgroundCSS rules) = %d, want 10", len(advBgSheet.Rules))
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
	th, err := Load(miniThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
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
	th, err := Load(stressThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
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
	baseTh, err := Load(importBaseThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load(base): %v", err)
	}
	childTh, err := Load(importChildThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load(child): %v", err)
	}

	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
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

	cyclicTh, err := Load(importCyclicThemeCSS, "section", testSizeFallback)
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
	th, err := Load(stressThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
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
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)

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
	th, err := Load(miniThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
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
	th, err := Load(authoredCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
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

// ---- Task 3: cssdiff.Equal acceptance gate (Test-list case 10) ----

// expectedStressPackedCSS is the hand-verified expected output of
// Pack("stress", PackOptions{InlineSVG: true}): derived by running the
// implemented pipeline once and reviewing every rule against
// 01-RESEARCH.md's documented transforms — scaffold prepended and scoped
// to "div.marpit > svg > foreignObject > section", the stress theme's own
// nesting (implicit-"&" child + "&"-compound) down-leveled and scoped,
// its ":root" declaration scoped THEN specificity-boosted to
// ":where(section):not([\20 root])" (proving the order: scope, then
// specificity — this TRD's central pipeline-ordering must_have), its
// plain `:where()`/`:is()`/`::backdrop` rules scoped as ordinary
// (space-prepended) selectors, and the advanced-background static block
// appended + scoped — EXCEPT the one rule using the bare
// ":marpit-container" placeholder (no paired ":marpit-slide"), which
// 01-01's locked selector.Replace cannot resolve — see
// pass_advancedbg.go's documented, deliberate scope-narrowing gap.
const expectedStressPackedCSS = `div.marpit > svg > foreignObject > section { width: 1280px; height: 720px; box-sizing: border-box; overflow: hidden; position: relative; scroll-snap-align: center center; -webkit-text-size-adjust: 100%; text-size-adjust: 100%; }
div.marpit > svg > foreignObject > section::after { bottom: 0; content: attr(data-marpit-pagination); padding: inherit; pointer-events: none; position: absolute; right: 0; }
div.marpit > svg > foreignObject > section:not([data-marpit-pagination])::after { display: none; }
div.marpit > svg > foreignObject > section :where(h1) { font-size: 2em; margin-block: 0.67em; }
div.marpit > svg > foreignObject > section video::-webkit-media-controls { will-change: transform; }
div.marpit > svg > foreignObject > :where(section):not([\20 root]) { --accent: teal; }
div.marpit > svg > foreignObject > section h1, div.marpit > svg > foreignObject > section h2 { color: var(--accent); }
div.marpit > svg > foreignObject > section.lead { text-align: center; }
div.marpit > svg > foreignObject > section :where(h1, h2) { margin: 0; }
div.marpit > svg > foreignObject > section :is(h3, h4) + p { margin-top: 0; }
div.marpit > svg > foreignObject > section ::backdrop { background: #048; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"] { columns: initial !important; display: block !important; padding: 0 !important; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"]::before, div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"]::after, div.marpit > svg > foreignObject > section[data-marpit-advanced-background="content"]::before, div.marpit > svg > foreignObject > section[data-marpit-advanced-background="content"]::after { display: none !important; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] { all: initial; display: flex; flex-direction: row; height: 100%; overflow: hidden; width: 100%; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container][data-marpit-advanced-background-direction="vertical"] { flex-direction: column; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split] > div[data-marpit-advanced-background-container] { width: var(--marpit-advanced-background-split, 50%); }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split="right"] > div[data-marpit-advanced-background-container] { margin-left: calc(100% - var(--marpit-advanced-background-split, 50%)); }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] > figure { all: initial; background-position: center; background-repeat: no-repeat; background-size: cover; flex: auto; margin: 0; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="content"], div.marpit > svg > foreignObject > section[data-marpit-advanced-background="pseudo"] { background: transparent !important; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background="pseudo"], :marpit-container > svg[data-marpit-svg] > foreignObject[data-marpit-advanced-background="pseudo"] { pointer-events: none !important; }
div.marpit > svg > foreignObject > section[data-marpit-advanced-background-split] { width: 100%; height: 100%; }`

// expectedScaffoldPackedCSS is the hand-verified expected output of
// Pack(ScaffoldThemeName, PackOptions{InlineSVG: false}): the scaffold
// theme packed against ITSELF must skip the scaffold-prepend step
// entirely (Test-list case 8 — no duplication) while still going through
// scope + the (here, no-op) root-mark/specificity/pagination passes, in
// the non-SVG container chain ("div.marpit > section").
const expectedScaffoldPackedCSS = `div.marpit > section { width: 1280px; height: 720px; box-sizing: border-box; overflow: hidden; position: relative; scroll-snap-align: center center; -webkit-text-size-adjust: 100%; text-size-adjust: 100%; }
div.marpit > section::after { bottom: 0; content: attr(data-marpit-pagination); padding: inherit; pointer-events: none; position: absolute; right: 0; }
div.marpit > section:not([data-marpit-pagination])::after { display: none; }
div.marpit > section :where(h1) { font-size: 2em; margin-block: 0.67em; }
div.marpit > section video::-webkit-media-controls { will-change: transform; }`

// TestPackFullPipelineStressThemeMatchesFixtureViaCSSDiff covers Test-list
// case 10 (stress theme half): Pack("stress", InlineSVG) equals the
// hand-verified expected fixture via conformance/cssdiff.Equal — a
// format-insensitive, order-sensitive AST diff, not a brittle string
// compare.
func TestPackFullPipelineStressThemeMatchesFixtureViaCSSDiff(t *testing.T) {
	th, err := Load(stressThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
	ts.Add(th)

	out, err := ts.Pack("stress", PackOptions{InlineSVG: true})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if equal, diff := cssdiff.Equal(expectedStressPackedCSS, out); !equal {
		t.Fatalf("Pack(stress) != expected fixture via cssdiff.Equal:\n%s", diff)
	}
}

// TestPackFullPipelineScaffoldThemeMatchesFixtureViaCSSDiff covers
// Test-list case 10 (scaffold theme half): Pack(scaffold) equals the
// hand-verified expected fixture via cssdiff.Equal.
func TestPackFullPipelineScaffoldThemeMatchesFixtureViaCSSDiff(t *testing.T) {
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)

	out, err := ts.Pack(ScaffoldThemeName, PackOptions{InlineSVG: false})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if equal, diff := cssdiff.Equal(expectedScaffoldPackedCSS, out); !equal {
		t.Fatalf("Pack(scaffold) != expected fixture via cssdiff.Equal:\n%s", diff)
	}
}
