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
	"fmt"
	"strings"

	"github.com/tdewolff/parse/v2/css"

	"github.com/AO-Cyber-Systems/eden-press/chase/theme/selector"
)

// pack.go implements Tier-2 of 01-RESEARCH.md's two-tier design:
// ThemeSet.Pack, run at RENDER time (every render, not once-per-theme
// like Tier-1's Load), producing the final, fully-scoped CSS text for a
// named, already-added theme.
//
// The Tier-2 pass order this implements (verbatim from
// 01-RESEARCH.md's numbered `runPostCSS` plugin list, collapsed to the
// steps this TRD's must_haves actually require — see the TRD's
// anti_patterns: "do NOT simplify pass order into one linear pipeline",
// meaning don't drop or reorder these, not that every one of Marpit's 20
// steps needs its own literal implementation here):
//
//  1. @import-theme resolve, recursively, cycle detection (pass_import.go)
//  2. scaffold prepend, skipped for the scaffold theme itself (pass_scaffold.go)
//  3. advanced-background static CSS append, inline-SVG only (pass_advancedbg.go)
//  4. pagination comment-out (pass_pagination.go)
//  5. SECOND ":root" -> sentinel pass, over the now-injected CSS (pass_root.go's rootMarkPass)
//  6. selector-scope: prepend placeholder, then replace with the real
//     container/slide chain, delegated entirely to chase/theme/selector
//     (01-01, locked) — see scopePass/scopeSelector below
//  7. increasing-specificity, run STRICTLY AFTER step 6 (pass_root.go's specificityPass)
//
// "hoist @charset/@import" (RESEARCH items 4/19) has no separate Pass
// here: this package's Stylesheet model already keeps recorded Atoms in
// their own ordered slice, rendered before Rules regardless of source
// position (see stylesheet.go's String / this file's renderPacked) — the
// model itself satisfies "hoisted to the front" by construction, so a
// discrete hoist step would be a no-op.
//
// The unit-element ident, the scaffold CSS, and the advanced-background
// CSS are all caller-supplied at ThemeSet construction time (TRD 02-03,
// MODEL-04's de-hardcoding move — see profiles/slides, which supplies
// all three today): chase/theme has no profile-specific value of its own,
// only the profile-agnostic passes that consume them.

// PackOptions controls a single Pack call's render-mode-dependent
// behavior.
type PackOptions struct {
	// InlineSVG selects Marpit's inline-SVG render mode: the container
	// chain becomes "div.marpit > svg > foreignObject"
	// (selector.InlineSVGContainerChain) instead of "div.marpit"
	// (selector.NonSVGContainerChain), and the advanced-background CSS
	// supplied to NewThemeSet is appended.
	InlineSVG bool
}

// ThemeSet is a named registry of loaded Themes (see theme.go's Load),
// used to Pack any one of them into final, fully-scoped CSS text.
//
// Every ThemeSet auto-registers an internal scaffold theme identity (the
// caller-supplied scaffoldCSS, under the reserved ScaffoldThemeName) at
// construction — see NewThemeSet — so Pack can compare a theme's identity
// against it (pointer equality) to implement Test-list case 8's "skipped
// when packing the scaffold theme itself". unit and advancedBackground
// are likewise supplied by the caller at construction (TRD 02-03,
// MODEL-04's de-hardcoding move: chase/theme owns no profile-specific
// value of its own — see profiles/slides, which supplies all three
// today).
type ThemeSet struct {
	themes             map[string]*Theme
	scaffold           *Theme
	unit               string
	advancedBackground []Rule

	// containerNonSVG / containerSVG are the profile-supplied container
	// chains Pack scopes rules under. Nil means "use chase/theme/selector's
	// built-in Marp chains", which is what every caller got before
	// SetContainerChains existed — see that method's doc.
	containerNonSVG []css.Token
	containerSVG    []css.Token
}

// SetContainerChains supplies the container-chain selectors the active
// Profile declares (profile.Profile.Container(false) and Container(true)),
// replacing chase/theme/selector's built-in "div.marpit" / "div.marpit > svg >
// foreignObject" defaults.
//
// This closes a gap that went unnoticed while exactly one profile existed:
// Profile.Container()'s own documentation says it "de-hardcodes
// chase/theme/selector/scope.go's inlineSVGChain / nonSVGChain", but nothing
// ever passed its value here, so those package vars stayed the real source of
// truth and a second profile's rules were scoped under the SLIDES container —
// CSS that could never match its own markup.
//
// An empty string leaves the corresponding default in place, so existing
// callers that never call this behave exactly as before.
func (ts *ThemeSet) SetContainerChains(nonSVG, inlineSVG string) error {
	if nonSVG != "" {
		toks, err := selector.ParseSelectorTokens(nonSVG)
		if err != nil {
			return fmt.Errorf("theme: SetContainerChains: non-SVG chain %q: %w", nonSVG, err)
		}
		ts.containerNonSVG = toks
	}
	if inlineSVG != "" {
		toks, err := selector.ParseSelectorTokens(inlineSVG)
		if err != nil {
			return fmt.Errorf("theme: SetContainerChains: inline-SVG chain %q: %w", inlineSVG, err)
		}
		ts.containerSVG = toks
	}
	return nil
}

// NewThemeSet constructs an empty ThemeSet with its internal scaffold
// theme identity already registered under ScaffoldThemeName. unit is the
// unit-element ident chase/theme's scoping passes scope theme rules
// onto; scaffoldCSS is the base CSS text prepended before every packed
// theme's own rules (except the scaffold theme itself); advancedBackgroundCSS
// is the static CSS block appended when a Pack call has InlineSVG
// enabled (pass "" to omit it entirely). All three values originate in
// the active Profile (see chase/profile) — chase/theme has no default of
// its own.
func NewThemeSet(unit, scaffoldCSS, advancedBackgroundCSS string) *ThemeSet {
	ts := &ThemeSet{themes: make(map[string]*Theme), unit: unit}
	ts.scaffold = &Theme{
		Name:  ScaffoldThemeName,
		Sheet: Stylesheet{Rules: mustLoadPlainRules(scaffoldCSS, unit)},
	}
	ts.themes[ScaffoldThemeName] = ts.scaffold
	if advancedBackgroundCSS != "" {
		ts.advancedBackground = mustLoadPlainRules(advancedBackgroundCSS, unit)
	}
	return ts
}

// Add registers th under th.Name, replacing any existing theme of that
// name (including — deliberately — ScaffoldThemeName, letting a caller
// override the built-in scaffold if it ever needs to).
func (ts *ThemeSet) Add(th *Theme) {
	ts.themes[th.Name] = th
}

// Get returns the registered theme named name, and whether it was found.
func (ts *ThemeSet) Get(name string) (*Theme, bool) {
	th, ok := ts.themes[name]
	return th, ok
}

// Pack renders the named theme's final, fully-scoped CSS text: @import-
// theme resolution, scaffold-prepend, advanced-background injection,
// pagination neutralization, the second :root-mark pass, selector-scope
// (prepend + replace), and the post-scoping increasing-specificity
// rewrite — see this file's package doc for the full ordered list.
func (ts *ThemeSet) Pack(name string, opts PackOptions) (string, error) {
	th, ok := ts.Get(name)
	if !ok {
		return "", fmt.Errorf("theme: Pack: unknown theme %q", name)
	}

	sheet, err := resolveImportTheme(ts, name, map[string]bool{})
	if err != nil {
		return "", err
	}

	container := ts.containerNonSVG
	if container == nil {
		container = selector.NonSVGContainerChain()
	}
	if opts.InlineSVG {
		container = ts.containerSVG
		if container == nil {
			container = selector.InlineSVGContainerChain()
		}
	}
	slide := selector.UnitChain(ts.unit)

	skipScaffold := th == ts.scaffold
	passes := []Pass{scaffoldPass(ts.scaffold.Sheet.Rules, skipScaffold)}
	if opts.InlineSVG {
		passes = append(passes, advancedBackgroundPass(ts.advancedBackground))
	}
	passes = append(passes,
		paginationPass,
		rootMarkPass(ts.unit),
		scopePass(container, slide, ts.unit),
		specificityPass(ts.unit),
	)

	if err := RunPasses(&sheet, passes...); err != nil {
		return "", err
	}

	return renderPacked(sheet), nil
}

// scopePass returns a Pass applying chase/theme/selector's two-step
// placeholder scoping (Prepend, then Replace with the real
// container/slide chain) to every rule's selector — see scopeSelector.
// unit is the caller-supplied unit-element ident Prepend compares
// against.
func scopePass(container, slide []css.Token, unit string) Pass {
	return Pass{
		Name: "scope",
		Run: func(sheet *Stylesheet) error {
			sheet.Rules = scopeRulesAll(sheet.Rules, container, slide, unit)
			return nil
		},
	}
}

// scopeRulesAll applies scopeSelector to every rule's SelectorTokens,
// returning a new slice (never mutating rules in place).
//
// A block at-rule has no selector of its own — scoping its empty
// SelectorTokens would synthesize a bogus one — so it is descended INTO
// instead: the rules inside @media/@supports need exactly the same scoping as
// their top-level siblings, or they would target unscoped elements and leak
// out of the deck.
func scopeRulesAll(rules []Rule, container, slide []css.Token, unit string) []Rule {
	out := make([]Rule, len(rules))
	for i, r := range rules {
		if r.At != nil {
			r.Children = scopeRulesAll(r.Children, container, slide, unit)
			out[i] = r
			continue
		}
		r.SelectorTokens = scopeSelector(r.SelectorTokens, container, slide, unit)
		out[i] = r
	}
	return out
}

// scopeSelector scopes a single (possibly multi-compound, comma-
// separated) selector token list: split into its top-level compounds via
// chase/theme/selector.SplitList, Prepend + Replace each independently,
// then re-join via joinCompounds.
func scopeSelector(tokens []css.Token, container, slide []css.Token, unit string) []css.Token {
	compounds := selector.SplitList(tokens)
	scoped := make([][]css.Token, len(compounds))
	for i, c := range compounds {
		scoped[i] = selector.Replace(selector.Prepend(c, unit), container, slide)
	}
	return joinCompounds(scoped)
}

// selectorText renders a selector token list back to canonical CSS text
// (trimmed) via chase/theme/selector.String — this package's single
// selector-to-text path, used by both renderPacked/renderRule (the
// output serializer) and pass_pagination.go's ::after detection.
//
// Deliberately NOT stylesheet.go's tokensText: that helper concatenates
// token Data with no combinator/comma spacing (fine for chase/theme's own
// structural round-trip, see Rule.String's doc), whereas this pipeline's
// final output must match real-world scoped-selector spacing conventions
// (padded combinators, padded function-argument commas) for
// conformance/cssdiff.Equal-based comparison against hand-built fixtures.
func selectorText(tokens []css.Token) string {
	return strings.TrimSpace(selector.String(tokens))
}

// renderPacked serializes a fully-processed Stylesheet to final CSS text:
// recorded Atoms first (see this file's package doc on why no separate
// hoist pass is needed), then Rules in order.
func renderPacked(sheet Stylesheet) string {
	var b strings.Builder
	for _, a := range sheet.Atoms {
		b.WriteString(a.String())
		b.WriteString(";\n")
	}
	// renderRule may return "" (an at-rule block emptied of all its children),
	// so separation tracks what was actually written rather than the index.
	wrote := false
	for _, r := range sheet.Rules {
		text := renderRule(r)
		if text == "" {
			continue
		}
		if wrote {
			b.WriteString("\n")
		}
		b.WriteString(text)
		wrote = true
	}
	return b.String()
}

// renderRule renders a single flat (no Children — Pack's pipeline always
// flattens nesting during Tier-1 Load, see pass_nesting.go) Rule to CSS
// text, using selectorText for the selector and declarationText for each
// declaration (not stylesheet.go's Rule.String/Declaration.String, which
// use the unpadded tokensText — see selectorText's doc).
func renderRule(r Rule) string {
	var b strings.Builder
	// A block at-rule keeps its Children after flattening (flattenRule
	// flattens WITHIN the block rather than hoisting out of it), so it is the
	// one Rule shape here that is not flat. Rendering it via the flat path
	// would emit a bare " { }" and silently drop the whole block — which is
	// exactly what happened to themes/uncover.css's @media print.
	if r.At != nil {
		if len(r.Children) == 0 && len(r.Declarations) == 0 {
			return ""
		}
		b.WriteString(r.At.String())
		b.WriteString(" {")
		// Descriptors declared directly on the at-rule (@page, @font-face).
		for _, d := range r.Declarations {
			b.WriteString(" ")
			b.WriteString(declarationText(d))
			b.WriteString(";")
		}
		for _, c := range r.Children {
			inner := renderRule(c)
			if inner == "" {
				continue
			}
			b.WriteString(" ")
			b.WriteString(inner)
		}
		b.WriteString(" }")
		return b.String()
	}
	b.WriteString(selectorText(r.SelectorTokens))
	b.WriteString(" {")
	for _, d := range r.Declarations {
		b.WriteString(" ")
		b.WriteString(declarationText(d))
		b.WriteString(";")
	}
	b.WriteString(" }")
	return b.String()
}

// declarationText renders a declaration back to "property: value" CSS
// text, repadding any bare (function-argument) comma in the value with a
// single trailing space — mirroring selector.String()'s existing comma/
// combinator repadding discipline, so packed output matches a caller's
// static CSS text byte-for-byte (Test-list case 9) rather than
// chase/theme's own tokensText, which — like tdewolff itself — drops the
// whitespace token adjacent to a function-argument comma entirely (see
// stylesheet.go's tokensText doc).
func declarationText(d Declaration) string {
	var b strings.Builder
	for _, t := range d.Value {
		b.Write(t.Data)
		if t.TokenType == css.CommaToken {
			b.WriteByte(' ')
		}
	}
	value := strings.TrimSpace(b.String())
	if d.Important {
		return d.Property + ": " + value + " !important"
	}
	return d.Property + ": " + value
}
