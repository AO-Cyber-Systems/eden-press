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

// pass_scaffold.go implements Tier-2's scaffold-prepend step (RESEARCH's
// Tier-2 order item 7, "theme !== scaffoldTheme && scaffold"): every
// packed theme's rule list is prefixed with Marpit's base slide-reset CSS
// (scaffold.go's ScaffoldCSS, pre-parsed into Rules by NewThemeSet) —
// UNLESS the theme being packed IS the scaffold theme itself (Test-list
// case 8: "skipped when packing the scaffold theme itself"), where
// self-prepending would duplicate its own rules.

// scaffoldPass returns a Pass that prepends scaffoldRules to the current
// sheet's Rules, unless skip is true (packing the scaffold theme itself)
// or scaffoldRules is empty.
func scaffoldPass(scaffoldRules []Rule, skip bool) Pass {
	return Pass{
		Name: "scaffold",
		Run: func(sheet *Stylesheet) error {
			sheet.Rules = prependScaffold(sheet.Rules, scaffoldRules, skip)
			return nil
		},
	}
}

// prependScaffold prepends scaffoldRules in front of rules, unless skip
// is true or there is nothing to prepend.
func prependScaffold(rules, scaffoldRules []Rule, skip bool) []Rule {
	if skip || len(scaffoldRules) == 0 {
		return rules
	}
	out := make([]Rule, 0, len(scaffoldRules)+len(rules))
	out = append(out, scaffoldRules...)
	out = append(out, rules...)
	return out
}
