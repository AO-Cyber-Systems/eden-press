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

package selector

import "github.com/tdewolff/parse/v2/css"

// scope.go implements Marpit's two-step placeholder scoping (mirrors
// postcss/pseudo_selector/{prepend,replace}.js): (1) Prepend injects a
// placeholder container/slide chain (":marpit-container > :marpit-slide")
// in front of every compound selector; (2) Replace later substitutes that
// placeholder with the REAL element chain — "div.marpit > svg >
// foreignObject > section" in inline-SVG render mode, "div.marpit >
// section" otherwise. Splitting scoping into these two steps (rather than
// prepending the real chain directly) is what lets root.go's
// IncreasingSpecificity run strictly AFTER Replace and still find the
// marker adjacent to the real "section" element (see root.go).

var (
	placeholderChain = mustParseTokens(":marpit-container > :marpit-slide")
	inlineSVGChain   = mustParseTokens("div.marpit > svg > foreignObject")
	nonSVGChain      = mustParseTokens("div.marpit")
	slideChain       = mustParseTokens("section")
)

// InlineSVGContainerChain returns the container-chain tokens for
// Marpit's inline-SVG render mode: "div.marpit > svg > foreignObject".
func InlineSVGContainerChain() []css.Token { return inlineSVGChain }

// NonSVGContainerChain returns the container-chain tokens for Marpit's
// non-SVG render mode: "div.marpit".
func NonSVGContainerChain() []css.Token { return nonSVGChain }

// SlideChain returns the slide-element tokens: "section", the element
// both render modes scope theme rules onto.
func SlideChain() []css.Token { return slideChain }

// Prepend injects the placeholder container/slide chain in front of a
// single compound selector (one entry from SplitList), never descending
// into a FunctionToken's arguments.
//
// Three cases, mirroring prepend.js exactly:
//  1. compound already starts with the ":marpit-container" placeholder —
//     idempotent no-op (test-list case 9, Prepend-level reading).
//  2. compound's FIRST token is literally the ident "section" — FUSED:
//     the placeholder replaces "section" itself (they become the SAME
//     element), and the rest of the compound (e.g. a trailing
//     ":where(.lead)" or "[data-x=...]") is appended directly with no
//     separator. This is also why root.go's MarkRoot emits the literal
//     text "section:marpit-root" — so a :root rule fuses here exactly
//     like a literal "section" rule does.
//  3. Any other compound (a bare tag like "h1", a class, a pseudo-class,
//     an :is()/:where() call, ...) — SPACED: the placeholder is prepended
//     as an ancestor via the descendant combinator, since the rule
//     targets a DESCENDANT of the slide element, not the slide itself.
func Prepend(compound []css.Token) []css.Token {
	if len(compound) == 0 {
		return compound
	}
	if isContainerSentinel(compound) {
		return compound
	}

	if compound[0].TokenType == css.IdentToken && string(compound[0].Data) == "section" {
		rest := compound[1:]
		out := make([]css.Token, 0, len(placeholderChain)+len(rest))
		out = append(out, placeholderChain...)
		out = append(out, rest...)
		return out
	}

	out := make([]css.Token, 0, len(placeholderChain)+1+len(compound))
	out = append(out, placeholderChain...)
	out = append(out, css.Token{TokenType: css.WhitespaceToken, Data: []byte(" ")})
	out = append(out, compound...)
	return out
}

// isContainerSentinel reports whether compound already begins with the
// ":marpit-container" placeholder (i.e. Prepend already ran on it).
func isContainerSentinel(compound []css.Token) bool {
	return len(compound) >= 2 &&
		compound[0].TokenType == css.ColonToken &&
		compound[1].TokenType == css.IdentToken &&
		string(compound[1].Data) == "marpit-container"
}

// Replace substitutes the ":marpit-container > :marpit-slide" placeholder
// (however Prepend positioned it) with the real container/slide chain:
// container + " > " + slide. If no placeholder is present, Replace is a
// safe no-op — it returns compound unchanged, so callers can run it
// idempotently on selectors Prepend never touched.
func Replace(compound []css.Token, container, slide []css.Token) []css.Token {
	idx := findPlaceholder(compound)
	if idx < 0 {
		return compound
	}

	chain := make([]css.Token, 0, len(container)+1+len(slide))
	chain = append(chain, container...)
	chain = append(chain, css.Token{TokenType: css.DelimToken, Data: []byte(">")})
	chain = append(chain, slide...)

	out := make([]css.Token, 0, idx+len(chain)+(len(compound)-idx-5))
	out = append(out, compound[:idx]...)
	out = append(out, chain...)
	out = append(out, compound[idx+5:]...)
	return out
}

// findPlaceholder locates the 5-token ":marpit-container > :marpit-slide"
// placeholder sequence within tokens, returning its start index or -1 if
// absent. Prepend always places it at index 0, but Replace scans
// generically rather than assuming that, so it stays a safe no-op when
// called on anything Prepend never touched.
func findPlaceholder(tokens []css.Token) int {
	for i := 0; i+4 < len(tokens); i++ {
		if tokens[i].TokenType == css.ColonToken &&
			tokens[i+1].TokenType == css.IdentToken && string(tokens[i+1].Data) == "marpit-container" &&
			tokens[i+2].TokenType == css.DelimToken && string(tokens[i+2].Data) == ">" &&
			tokens[i+3].TokenType == css.ColonToken &&
			tokens[i+4].TokenType == css.IdentToken && string(tokens[i+4].Data) == "marpit-slide" {
			return i
		}
	}
	return -1
}
