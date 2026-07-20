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

package cssdiff

// CONF-03 comparator. Equal parses two CSS strings into the normalized Stylesheet
// model (build.go) and compares them:
//
//   - FORMAT-INSENSITIVE: whitespace, indentation, and comments are erased by
//     parsing, so ".a{color:red}" and ".a {\n  color: red;\n}" compare equal.
//   - ORDER-SENSITIVE: rule order and in-rule declaration order are preserved and
//     significant, because the CSS cascade is order-dependent (a later rule of
//     equal specificity wins; a repeated property's last value wins). Any reorder
//     is therefore reported as a difference — Marpit's theme output is
//     deterministic, so a reorder signals a real generation change, not noise.
//   - VALUE / SELECTOR / !important SENSITIVE: a changed value, changed selector,
//     or dropped `!important` is a real cascade change and is reported.
//
// This is the CSS analogue of conformance/htmldiff.Equal (DOM-normalized HTML
// diff): it lets the theme-CSS acceptance gate ignore cosmetic formatting while
// catching every semantically meaningful difference.

import (
	"fmt"
	"strings"
)

// Equal reports whether two CSS strings are semantically equal under the rules
// documented on the package, returning a human-readable unified-style diff of the
// first differences when they are not. A parse error on either side yields
// (false, "<side>: parse error: ...").
func Equal(expected, actual string) (equal bool, diff string) {
	exp, err := Parse(expected)
	if err != nil {
		return false, fmt.Sprintf("expected: parse error: %v", err)
	}
	act, err := Parse(actual)
	if err != nil {
		return false, fmt.Sprintf("actual: parse error: %v", err)
	}
	return diffStylesheets(exp, act)
}

func diffStylesheets(exp, act Stylesheet) (bool, string) {
	var b strings.Builder
	equal := true

	n := len(exp.Rules)
	if len(act.Rules) > n {
		n = len(act.Rules)
	}
	for i := 0; i < n; i++ {
		switch {
		case i >= len(exp.Rules):
			equal = false
			fmt.Fprintf(&b, "+ rule[%d] present only in actual:   %s\n", i, ruleHeader(act.Rules[i]))
		case i >= len(act.Rules):
			equal = false
			fmt.Fprintf(&b, "- rule[%d] present only in expected: %s\n", i, ruleHeader(exp.Rules[i]))
		default:
			if d := diffRule(i, exp.Rules[i], act.Rules[i]); d != "" {
				equal = false
				b.WriteString(d)
			}
		}
	}
	return equal, b.String()
}

func diffRule(i int, e, a Rule) string {
	var b strings.Builder
	if e.AtRule != a.AtRule {
		fmt.Fprintf(&b, "  rule[%d] at-rule: expected %q, actual %q\n", i, e.AtRule, a.AtRule)
	}
	if e.Selector != a.Selector {
		fmt.Fprintf(&b, "  rule[%d] selector: expected %q, actual %q\n", i, e.Selector, a.Selector)
	}

	m := len(e.Declarations)
	if len(a.Declarations) > m {
		m = len(a.Declarations)
	}
	for j := 0; j < m; j++ {
		switch {
		case j >= len(e.Declarations):
			fmt.Fprintf(&b, "  rule[%d] decl[%d] present only in actual:   %s\n", i, j, declString(a.Declarations[j]))
		case j >= len(a.Declarations):
			fmt.Fprintf(&b, "  rule[%d] decl[%d] present only in expected: %s\n", i, j, declString(e.Declarations[j]))
		case e.Declarations[j] != a.Declarations[j]:
			fmt.Fprintf(&b, "  rule[%d] decl[%d]: expected %s, actual %s\n", i, j,
				declString(e.Declarations[j]), declString(a.Declarations[j]))
		}
	}
	return b.String()
}

func ruleHeader(r Rule) string {
	if r.AtRule != "" {
		return "@" + r.AtRule + " " + r.Selector
	}
	return r.Selector
}

func declString(d Declaration) string {
	s := d.Property + ": " + d.Value
	if d.Important {
		s += " !important"
	}
	return s
}
