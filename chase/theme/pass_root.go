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

import "github.com/AO-Cyber-Systems/eden-press/chase/theme/selector"

// pass_root.go wires chase/theme/selector's (01-01, locked) two ":root"
// rewrite steps into this package's Pass pipeline: MarkRoot (the add-time
// ":root" -> "<unit>:marpit-root" sentinel rewrite, run once at Tier-1
// Load, and AGAIN at Tier-2 Pack over the freshly-injected scaffold/
// advanced-background CSS — 01-RESEARCH.md's ":root is replaced twice"
// requirement) and IncreasingSpecificity (the final ":marpit-root" ->
// ":where(<unit>):not([\20 root])" specificity-trick rewrite, which
// pack.go's Pack sequences strictly AFTER selector-scoping — see this
// TRD's anti_patterns and RESEARCH Pitfall 1).
//
// Both passes now take the caller-supplied unit-element ident (TRD 02-03,
// MODEL-04's de-hardcoding move) and are therefore constructor FUNCTIONS
// rather than fixed package-level Pass values.

// rootMarkPass returns a Pass that rewrites every ":root"/"<unit>:root"
// occurrence across sheet.Rules to the "<unit>:marpit-root" sentinel via
// selector.MarkRoot. Used both by theme.go's Load (Tier-1, first pass)
// and pack.go's Pack (Tier-2, second pass, over injected CSS).
func rootMarkPass(unit string) Pass {
	return Pass{
		Name: "root-mark",
		Run: func(sheet *Stylesheet) error {
			sheet.Rules = markRootAll(sheet.Rules, unit)
			return nil
		},
	}
}

// specificityPass returns a Pass that rewrites every ":marpit-root"
// sentinel across sheet.Rules to the final
// ":where(<unit>):not([\20 root])" specificity-trick sequence via
// selector.IncreasingSpecificity. pack.go's Pack MUST sequence this
// strictly after its scope-prefix pass (Prepend + Replace) — see this
// file's package doc and root.go's own doc comment.
func specificityPass(unit string) Pass {
	return Pass{
		Name: "increasing-specificity",
		Run: func(sheet *Stylesheet) error {
			sheet.Rules = increasingSpecificityAll(sheet.Rules, unit)
			return nil
		},
	}
}

// markRootAll applies selector.MarkRoot to every rule's SelectorTokens,
// recursing into Children defensively (harmless no-op once flattenNesting
// has already run — see pass_nesting.go — since a flat rule list has none
// by construction, but this keeps the helper correct if ever invoked
// before flattening).
func markRootAll(rules []Rule, unit string) []Rule {
	out := make([]Rule, len(rules))
	for i, r := range rules {
		// A block at-rule carries no selector; only its contents do.
		if r.At == nil {
			r.SelectorTokens = selector.MarkRoot(r.SelectorTokens, unit)
		}
		if len(r.Children) > 0 {
			r.Children = markRootAll(r.Children, unit)
		}
		out[i] = r
	}
	return out
}

// increasingSpecificityAll applies selector.IncreasingSpecificity to every
// rule's SelectorTokens, recursing into Children defensively (see
// markRootAll's doc for why).
func increasingSpecificityAll(rules []Rule, unit string) []Rule {
	out := make([]Rule, len(rules))
	for i, r := range rules {
		// A block at-rule carries no selector; only its contents do.
		if r.At == nil {
			r.SelectorTokens = selector.IncreasingSpecificity(r.SelectorTokens, unit)
		}
		if len(r.Children) > 0 {
			r.Children = increasingSpecificityAll(r.Children, unit)
		}
		out[i] = r
	}
	return out
}
