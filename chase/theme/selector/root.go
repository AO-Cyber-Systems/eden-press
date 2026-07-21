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

// root.go owns the ":root" handling described in 01-RESEARCH.md §"The
// :root specificity trick": step (1), the add-time ":root" -> sentinel
// rewrite (MarkRoot), and step (4), the LAST, post-scoping rewrite of
// that sentinel to the actual specificity-trick token sequence
// (IncreasingSpecificity). Steps (2)/(3) (a render-time second pass, and
// the scope-prefix mechanics themselves) belong to the render pipeline
// (01-04) and scope.go respectively — this file exposes (1) and (4) as
// two separate functions specifically so the pipeline can sequence them
// around Prepend/Replace, per the gotcha that ordering is load-bearing
// (RESEARCH Pitfall 1): IncreasingSpecificity MUST run after Replace, or
// the specificity trick ends up scoped to the WRONG (placeholder, not
// real) element.

// sectionMarkerTokens is the literal fused replacement MarkRoot inserts
// for a ":root"/"section:root" occurrence: the text "section:marpit-root".
// Emitting a LITERAL leading "section" ident (not just a bare
// ":marpit-root" pseudo-class) is deliberate: it makes scope.go's Prepend
// treat a :root rule EXACTLY like a literal "section" rule (the fused,
// same-element branch), rather than nesting it as a descendant — a
// :root-authored rule must apply to the SAME element as a plain "section"
// rule, not one of its children.
var sectionMarkerTokens = mustParseTokens("section:marpit-root")

// rootRewriteTokens is the literal specificity-trick token sequence
// IncreasingSpecificity substitutes for the ":marpit-root" sentinel,
// applied only AFTER scope-prefixing:
// `:where(section):not([\20 root])`.
//
// `:where()` contributes ZERO specificity while still always matching
// every "section" element; `:not([\20 root])` contributes exactly one
// attribute-selector specificity unit (0-1-0), outranking a plain
// "section" rule (0-0-1) — but `[\20 root]` is a CSS-escaped attribute
// selector for an attribute literally named "<SPACE>root", which is
// impossible to author in HTML, so the :not() always matches too. The
// escape must be emitted verbatim as token text, never "normalized"
// away (01-RESEARCH.md error_recovery).
var rootRewriteTokens = mustParseTokens(`:where(section):not([\20 root])`)

// MarkRoot rewrites every ":root" (or fused "section:root") occurrence in
// tokens to the literal text "section:marpit-root" — the add-time
// sentinel step. Unlike scope.go's Prepend, MarkRoot scans the ENTIRE
// flat token stream, including tokens nested inside a FunctionToken's
// arguments (`:is(:root, h1)`), because a real gaia-theme selector can
// carry the marker nested inside `:where(:is(...))` (test-list case 7)
// and it must still be found here so it survives to be scope-prefixed
// and later rewritten by IncreasingSpecificity.
//
// A match is only rewritten when it sits at a "valid boundary" —
// mirroring Marpit's own regex `/(^|[\s>+~(])(?:section)?:root\b/g`:
// start-of-compound, a descendant-combinator WhitespaceToken, an explicit
// combinator (">"/"+"/"~"), or immediately after a FunctionToken/open-
// paren (so ":is(:root, h1)" matches the FIRST argument). A CommaToken is
// also accepted as a valid boundary here — a pragmatic superset of the
// JS regex (which, operating on raw text, can see a literal space after
// a comma that tdewolff's tokenizer drops from function-argument commas;
// see selector.go's String doc comment) that only ever broadens matching
// for a legitimate ":root" argument, never mis-rewrites unrelated text.
func MarkRoot(tokens []css.Token) []css.Token {
	out := make([]css.Token, 0, len(tokens))
	for i := 0; i < len(tokens); {
		if n := rootMarkerAt(tokens, i); n > 0 && validRootBoundary(tokens, i) {
			out = append(out, sectionMarkerTokens...)
			i += n
			continue
		}
		out = append(out, tokens[i])
		i++
	}
	return out
}

// rootMarkerAt reports how many tokens starting at tokens[i] form a
// ":root" (2 tokens: Colon, Ident("root")) or fused "section:root" (3
// tokens: Ident("section"), Colon, Ident("root")) occurrence — 0 if
// neither matches at i.
func rootMarkerAt(tokens []css.Token, i int) int {
	if i+1 < len(tokens) && isColonIdent(tokens[i], tokens[i+1], "root") {
		return 2
	}
	if i+2 < len(tokens) &&
		tokens[i].TokenType == css.IdentToken && string(tokens[i].Data) == "section" &&
		isColonIdent(tokens[i+1], tokens[i+2], "root") {
		return 3
	}
	return 0
}

// isColonIdent reports whether the two tokens are a Colon immediately
// followed by an IdentToken whose data equals name.
func isColonIdent(colon, ident css.Token, name string) bool {
	return colon.TokenType == css.ColonToken &&
		ident.TokenType == css.IdentToken &&
		string(ident.Data) == name
}

// validRootBoundary reports whether position i in tokens is a valid
// place for a ":root"/"section:root" match to start — see MarkRoot's doc
// comment for the exact boundary set.
func validRootBoundary(tokens []css.Token, i int) bool {
	if i <= 0 {
		return true
	}
	prev := tokens[i-1]
	switch prev.TokenType {
	case css.WhitespaceToken, css.FunctionToken, css.LeftParenthesisToken, css.CommaToken:
		return true
	case css.DelimToken:
		return isCombinatorDelim(prev.Data)
	}
	return false
}

// IncreasingSpecificity rewrites every ":marpit-root" sentinel (bare or
// fused as "<ident>:marpit-root") in tokens to the literal
// `:where(section):not([\20 root])` specificity-trick sequence. This
// MUST run AFTER scope.go's Replace — see this file's package doc comment
// and RESEARCH Pitfall 1.
//
// It uses Walk (rather than a depth-blind scan) specifically so the
// nested-in-`:where(:is(...))` gaia case (test-list case 7) is
// unambiguously exercised via the same traversal root.go documents
// elsewhere as depth-aware, even though marker DETECTION itself only
// looks at immediate token adjacency (tdewolff's stream is already flat,
// so a plain index scan would find the same matches) — Walk gives the
// scan a depth value for free, which callers/tests can use as evidence
// the marker really was nested, not just present at the top level.
func IncreasingSpecificity(tokens []css.Token) []css.Token {
	type match struct {
		start int
		n     int
	}
	var matches []match

	Walk(tokens, func(_ css.Token, index int, _ int) bool {
		if len(matches) > 0 {
			last := matches[len(matches)-1]
			if index < last.start+last.n {
				return true // inside an already-recorded match; skip
			}
		}
		if n := marpitRootMarkerAt(tokens, index); n > 0 {
			matches = append(matches, match{start: index, n: n})
		}
		return true
	})

	if len(matches) == 0 {
		return tokens
	}

	out := make([]css.Token, 0, len(tokens))
	i, mi := 0, 0
	for i < len(tokens) {
		if mi < len(matches) && matches[mi].start == i {
			out = append(out, rootRewriteTokens...)
			i += matches[mi].n
			mi++
			continue
		}
		out = append(out, tokens[i])
		i++
	}
	return out
}

// marpitRootMarkerAt reports how many tokens starting at tokens[i] form a
// ":marpit-root" sentinel — 2 tokens bare (Colon, Ident("marpit-root")),
// or 3 tokens fused ("section", Colon, Ident("marpit-root")) — 0 if
// neither matches at i.
func marpitRootMarkerAt(tokens []css.Token, i int) int {
	if i+1 < len(tokens) && isColonIdent(tokens[i], tokens[i+1], "marpit-root") {
		return 2
	}
	if i+2 < len(tokens) &&
		tokens[i].TokenType == css.IdentToken && string(tokens[i].Data) == "section" &&
		isColonIdent(tokens[i+1], tokens[i+2], "marpit-root") {
		return 3
	}
	return 0
}
