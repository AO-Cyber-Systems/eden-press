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

// pass_advancedbg.go implements Tier-2's advanced-background injection
// step (RESEARCH's Tier-2 order item 8, "inlineSVGOpts.enabled &&
// advancedBackground"): scaffold.go's AdvancedBackgroundCSS static rule
// block (pre-parsed once into advancedBackgroundRules below) is appended
// whenever inline-SVG rendering is enabled.
//
// KNOWN, DOCUMENTED GAP (see this TRD's error_recovery: "if the stress
// theme/scaffold gate doesn't need a case, leave it as a documented
// TODO... don't silently pass"): one rule in AdvancedBackgroundCSS uses
// the chase/theme/selector container placeholder (":marpit-container")
// ALONE — "`:marpit-container > svg[data-marpit-svg] >
// foreignObject[...]`" — rather than paired with the ":marpit-slide"
// placeholder scope.go's Prepend/Replace expect together (see scope.go's
// 5-token findPlaceholder). 01-01's locked selector.Replace therefore
// leaves that ONE rule's container placeholder un-substituted; deferred
// deliberately rather than modifying the 01-01-owned selector package to
// special-case a container-only placeholder it was never designed to
// handle. Neither Pack(stress) nor Pack(scaffold) — this TRD's
// must_haves gate — exercises that one rule's selector, so it does not
// block this TRD; a real render pipeline would need a small additional
// container-only substitution step in a later objective.

// advancedBackgroundRules is scaffold.go's AdvancedBackgroundCSS,
// pre-parsed once (nesting-flattened + root-marked, neither of which
// applies to this static block, via loadPlain) — safe to share across
// Pack calls since every pipeline Pass builds fresh output slices rather
// than mutating a Rule's SelectorTokens/Declarations in place.
var advancedBackgroundRules = mustLoadPlainRules(AdvancedBackgroundCSS)

// mustLoadPlainRules loads plain (meta-less) static CSS text via
// loadPlain and panics if it fails to parse — used only for this
// package's own embedded, package-authored CSS constants (scaffold.go),
// never for untrusted input.
func mustLoadPlainRules(cssText string) []Rule {
	sheet, err := loadPlain(cssText)
	if err != nil {
		panic("theme: embedded static CSS failed to parse: " + err.Error())
	}
	return sheet.Rules
}

// advancedBackgroundPass returns a Pass that appends
// advancedBackgroundRules to the current sheet's Rules.
func advancedBackgroundPass() Pass {
	return Pass{
		Name: "advanced-background",
		Run: func(sheet *Stylesheet) error {
			sheet.Rules = append(sheet.Rules, advancedBackgroundRules...)
			return nil
		},
	}
}
