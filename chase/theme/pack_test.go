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
