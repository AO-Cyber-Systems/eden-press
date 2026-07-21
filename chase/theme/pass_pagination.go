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

import "strings"

// pass_pagination.go implements Tier-2's pagination step (RESEARCH's
// Tier-2 order item 10, "pagination comment-out non-marpit content on
// section::after"): an author-authored `content` declaration that
// targets a `::after` pseudo-element and is NOT the scaffold's own
// default `content: attr(data-marpit-pagination)` pagination text is
// neutralized (dropped — "commented out" in Marpit's own terms) so it
// can never silently clobber the page-number content scaffold.go's own
// `section::after` rule injects. The default pagination declaration
// itself, and any rule that isn't ::after-targeting at all, is left
// untouched.

// paginationPass is a Pass that neutralizes non-default `content`
// declarations on ::after-targeting rules across the current sheet.
var paginationPass = Pass{
	Name: "pagination",
	Run: func(sheet *Stylesheet) error {
		sheet.Rules = commentOutPagination(sheet.Rules)
		return nil
	},
}

// commentOutPagination applies the pagination neutralization to every
// rule in rules, returning a new slice (never mutating rules in place).
func commentOutPagination(rules []Rule) []Rule {
	out := make([]Rule, len(rules))
	for i, r := range rules {
		if targetsAfterPseudo(r) {
			r.Declarations = dropNonDefaultPaginationContent(r.Declarations)
		}
		out[i] = r
	}
	return out
}

// targetsAfterPseudo reports whether r's selector targets a "::after"
// pseudo-element anywhere in its (rendered) text.
func targetsAfterPseudo(r Rule) bool {
	return strings.Contains(selectorText(r.SelectorTokens), "::after")
}

// dropNonDefaultPaginationContent returns decls with any "content"
// declaration removed UNLESS it is exactly the default pagination
// content (attr(data-marpit-pagination)) — all other declarations pass
// through unchanged.
func dropNonDefaultPaginationContent(decls []Declaration) []Declaration {
	out := make([]Declaration, 0, len(decls))
	for _, d := range decls {
		if d.Property == "content" && !isDefaultPaginationContent(d) {
			continue // neutralized ("commented out")
		}
		out = append(out, d)
	}
	return out
}

// isDefaultPaginationContent reports whether d is exactly the scaffold's
// default pagination content declaration.
func isDefaultPaginationContent(d Declaration) bool {
	return tokensText(d.Value) == "attr(data-marpit-pagination)"
}
