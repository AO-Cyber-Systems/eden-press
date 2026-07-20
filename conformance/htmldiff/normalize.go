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

// Package htmldiff compares two HTML fragments after DOM normalization so that a
// NARROW, NAMED allow-list of provably-cosmetic differences does not count as a
// structural divergence. It is the single comparison primitive (Equal) reused by
// the spec-sweep runner (00-04) and the Marp corpus runner (00-05).
//
// The allow-list is EXACTLY these four classes and nothing more:
//
//  1. Void-element syntax — <br> vs <br/> vs <br />, <hr> vs <hr/> vs <hr /> all
//     canonicalize identically (golang.org/x/net/html collapses them at parse
//     time; htmldiff asserts it as a guaranteed contract).
//  2. <hr> void syntax — same class as (1), named separately per CONF-02 wording.
//  3. Attribute order — attributes are sorted before serialization, so
//     <a href="x" title="y"> and <a title="y" href="x"> match.
//  4. Inter-block whitespace — whitespace-only text between block elements, and
//     leading/trailing whitespace, is collapsed away (strings.Fields).
//
// This is NOT a general "ignore all whitespace" rule (Pitfall 13). Text inside
// <pre> and <code> is compared VERBATIM (whitespace-significant); over-normalizing
// it would mask real fidelity bugs. htmldiff's negative tests guard that boundary.
//
// The parser is golang.org/x/net/html (WHATWG tree construction) — htmldiff never
// hand-rolls an HTML tokenizer.
package htmldiff

import (
	"sort"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// normalize parses an HTML fragment and emits a canonical token stream so that
// cosmetic differences (void-element syntax <br> vs <br/>, inter-block
// whitespace, attribute order) don't count as structural divergence.
//
// html.ParseFragment needs a context node; a <body> atom node is used so bare
// block-level content parses correctly.
func normalize(fragment string) (string, error) {
	ctx := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(fragment), ctx)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, n := range nodes {
		walk(&sb, n, false)
	}
	return sb.String(), nil
}

// walk serializes n into a canonical token stream. It sorts attributes
// alphabetically (attribute-order class), collapses inter-block whitespace via
// strings.Fields (inter-block-whitespace class), and preserves <pre>/<code> text
// VERBATIM through the inPre flag (the whitespace-significant boundary the
// allow-list must not cross).
func walk(sb *strings.Builder, n *html.Node, inPre bool) {
	switch n.Type {
	case html.ElementNode:
		attrs := make([]string, 0, len(n.Attr))
		for _, a := range n.Attr {
			attrs = append(attrs, a.Key+"="+a.Val)
		}
		sort.Strings(attrs)
		sb.WriteString("<" + n.Data)
		if len(attrs) > 0 {
			sb.WriteString(" " + strings.Join(attrs, " "))
		}
		sb.WriteString(">\n")
		cin := inPre || n.Data == "pre" || n.Data == "code"
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(sb, c, cin)
		}
		sb.WriteString("</" + n.Data + ">\n")
	case html.TextNode:
		if inPre {
			// VERBATIM: pre/code text is whitespace-significant. Preserving the
			// raw Data is what lets the negative tests catch a real regression.
			if n.Data != "" {
				sb.WriteString("TEXT:" + n.Data + "\n")
			}
		} else {
			// Inter-block / intra-inline whitespace is collapsed to single spaces;
			// whitespace-only text nodes vanish entirely.
			t := strings.Join(strings.Fields(n.Data), " ")
			if t != "" {
				sb.WriteString("TEXT:" + t + "\n")
			}
		}
	case html.CommentNode:
		sb.WriteString("<!--" + n.Data + "-->\n")
	}
}

// Equal reports whether two HTML fragments are structurally equivalent after DOM
// normalization, i.e. they differ only by the narrow cosmetic allow-list
// documented on the package. When they are NOT equal, diff is a human-readable
// line-oriented view of the two canonical token streams (for test output). When
// they ARE equal, diff is empty.
//
// A parse error on either side is surfaced through diff (and equal=false) rather
// than panicking, so a malformed fixture fails loudly instead of silently
// comparing empty streams.
func Equal(expected, actual string) (equal bool, diff string) {
	ne, err := normalize(expected)
	if err != nil {
		return false, "normalize(expected) error: " + err.Error()
	}
	na, err := normalize(actual)
	if err != nil {
		return false, "normalize(actual) error: " + err.Error()
	}
	if ne == na {
		return true, ""
	}
	return false, unifiedDiff(ne, na)
}

// unifiedDiff renders a compact, line-oriented side view of the two canonical
// token streams. It is intentionally simple (not a minimal-edit LCS): the goal is
// legible test output that shows which canonical tokens diverged, not a perfect
// patch.
func unifiedDiff(expected, actual string) string {
	el := strings.Split(strings.TrimRight(expected, "\n"), "\n")
	al := strings.Split(strings.TrimRight(actual, "\n"), "\n")
	var b strings.Builder
	b.WriteString("--- expected (canonical)\n")
	b.WriteString("+++ actual (canonical)\n")
	max := len(el)
	if len(al) > max {
		max = len(al)
	}
	for i := 0; i < max; i++ {
		var e, a string
		if i < len(el) {
			e = el[i]
		}
		if i < len(al) {
			a = al[i]
		}
		if e == a {
			b.WriteString("  " + e + "\n")
			continue
		}
		if i < len(el) {
			b.WriteString("- " + e + "\n")
		}
		if i < len(al) {
			b.WriteString("+ " + a + "\n")
		}
	}
	return b.String()
}
