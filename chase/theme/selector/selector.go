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

// Package selector is the standalone, independently unit-tested CSS
// selector-rewriter subsystem (THEME-04) for the Chase theme-scoping
// pipeline (see 01-04). It operates on tdewolff/parse/v2/css TOKENS end
// to end — never flattening a selector to a string mid-pipeline — because
// the scope-prepend + :root specificity rewrite must be applied per
// COMPOUND selector and must never descend into a FunctionToken's
// (`:is(...)`, `:where(...)`) arguments during prepending. A naive string
// concatenation ("div.marpit > svg > foreignObject > section " + rawText)
// breaks on multi-selector lists and function arguments (01-RESEARCH.md
// Pitfall 3); this package exists specifically to avoid that.
//
// This package imports ONLY tdewolff/parse and the stdlib — zero
// markdown/goldmark import, zero import of the parent chase/theme
// pipeline — so it can be built and unit-tested in complete isolation
// before any theme-pipeline code depends on it.
//
// selector.go (this file) provides the token-level primitives: splitting
// a selector list into its top-level comma-separated compounds
// (SplitList), a depth-aware token visitor (Walk) that CAN see inside
// FunctionToken arguments on demand, literal re-serialization (String),
// and a test/pipeline convenience for going from raw CSS selector text to
// a token slice (ParseSelectorTokens). scope.go and root.go build the
// actual rewrite behaviors on top of these primitives.
package selector

import (
	"io"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// ParseSelectorTokens parses a bare CSS selector (e.g. "h1, h2" or
// ":is(h3, h4) + p") and returns the raw token slice tdewolff's grammar
// walk captures at BeginRulesetGrammar for it — the same p.Values() this
// package's callers (and conformance/cssdiff/build.go, for stylesheet
// diffing) work from. Unlike cssdiff, this NEVER flattens the result to a
// string: callers get []css.Token so segmentation/scoping/root-rewriting
// can operate at the token level throughout.
//
// A selector alone is not valid top-level CSS, so this wraps it in a
// throwaway declaration block ("<selector>{x:y}") purely to make
// css.Parser emit the BeginRulesetGrammar event; the declaration itself
// is discarded.
func ParseSelectorTokens(selectorText string) ([]css.Token, error) {
	p := css.NewParser(parse.NewInputString(selectorText+"{x:y}"), false)
	for {
		gt, _, _ := p.Next()
		switch gt {
		case css.BeginRulesetGrammar, css.QualifiedRuleGrammar:
			return p.Values(), nil
		case css.ErrorGrammar:
			if err := p.Err(); err != nil && err != io.EOF {
				return nil, err
			}
			return nil, io.EOF
		}
	}
}

// mustParseTokens parses a fixed, package-internal selector literal (a
// placeholder chain, a container/slide chain, or the :root rewrite
// target) via ParseSelectorTokens and panics on error. It exists so
// scope.go/root.go's constant token sequences are built by the SAME
// tokenizer used everywhere else in this package — dogfooding the parser
// — rather than hand-authored as css.Token{} struct literals, which would
// be one more place to introduce a typo in an escape sequence like
// `\20 `. It is only ever called with fixed, package-authored strings at
// package-var initialization time, never with untrusted input.
func mustParseTokens(selectorText string) []css.Token {
	tokens, err := ParseSelectorTokens(selectorText)
	if err != nil {
		panic("selector: mustParseTokens(" + selectorText + "): " + err.Error())
	}
	return tokens
}

// SplitList splits a full selector-list token stream into its
// comma-separated compound selectors, splitting ONLY on TOP-LEVEL commas.
// A comma nested inside a FunctionToken's arguments (`:is(a, b)`), inside
// an attribute selector (`[data-x="a,b"]`), or inside a quoted string is
// never a split point — depth is tracked across FunctionToken/
// LeftParenthesisToken/LeftBracketToken opens and their matching closes;
// a StringToken's internal comma is not even a separate token in
// tdewolff's lexer, so it is naturally never seen as one here.
//
// Each returned compound has its incidental leading/trailing whitespace
// trimmed. A wholly empty or whitespace-only input returns a nil/empty
// slice (zero compounds) rather than one phantom empty compound — Test-
// list case 9 (empty selector is a safe no-op).
func SplitList(tokens []css.Token) [][]css.Token {
	var result [][]css.Token
	var current []css.Token
	depth := 0

	for _, t := range tokens {
		switch t.TokenType {
		case css.RightParenthesisToken, css.RightBracketToken:
			if depth > 0 {
				depth--
			}
		}

		if t.TokenType == css.CommaToken && depth == 0 {
			result = append(result, trimWhitespace(current))
			current = nil
		} else {
			current = append(current, t)
		}

		switch t.TokenType {
		case css.FunctionToken, css.LeftParenthesisToken, css.LeftBracketToken:
			depth++
		}
	}

	last := trimWhitespace(current)
	if len(result) > 0 || len(last) > 0 {
		result = append(result, last)
	}
	return result
}

// trimWhitespace strips leading/trailing css.WhitespaceToken entries from
// a token slice, returning nil if nothing but whitespace remains.
func trimWhitespace(tokens []css.Token) []css.Token {
	start := 0
	for start < len(tokens) && tokens[start].TokenType == css.WhitespaceToken {
		start++
	}
	end := len(tokens)
	for end > start && tokens[end-1].TokenType == css.WhitespaceToken {
		end--
	}
	if start == end {
		return nil
	}
	return tokens[start:end]
}

// Walk visits every token in tokens in source order, invoking fn with the
// token, its index, and the current nesting depth (0 at the top level of
// the compound selector; incremented for tokens that lie inside a
// FunctionToken's arguments, a parenthesized group, or an attribute
// selector's brackets).
//
// tdewolff's css.Token stream is already FLAT: a FunctionToken's
// arguments are ordinary tokens in the SAME slice, terminated by a
// RightParenthesisToken — there is no separate nested slice to descend
// into. Walk's only job is to track that depth as it scans, which is
// exactly what lets root.go find (and rewrite) a `:root` marker nested
// inside `:is(...)`/`:where(...)` function arguments — the gaia
// regression case — something SplitList and scope.go's Prepend
// deliberately do NOT do (they only ever act at depth 0).
//
// If fn returns false, Walk stops early.
func Walk(tokens []css.Token, fn func(tok css.Token, index int, depth int) bool) {
	depth := 0
	for i, t := range tokens {
		switch t.TokenType {
		case css.RightParenthesisToken, css.RightBracketToken:
			if depth > 0 {
				depth--
			}
		}
		if !fn(t, i, depth) {
			return
		}
		switch t.TokenType {
		case css.FunctionToken, css.LeftParenthesisToken, css.LeftBracketToken:
			depth++
		}
	}
}

// String serializes a token slice back to literal CSS text.
//
// tdewolff's css.Parser DROPS whitespace immediately adjacent to an
// explicit combinator (`>`, `~`, `+`) and around a FunctionToken's
// argument commas — confirmed empirically: `"h1 > h2"` tokenizes as
// Ident/Delim(">")/Ident with NO WhitespaceToken at all, and
// `":is(h1, h2)"` tokenizes with a bare CommaToken and no following
// Whitespace. Only the plain DESCENDANT combinator (an actual space with
// no explicit delimiter, e.g. `"h1 h2"` or `"section :is(...)"`) survives
// as a real WhitespaceToken, because there it IS the combinator.
//
// Real Marpit output (conformance/corpus/cases/marp-theme-gaia/expected.css)
// nonetheless has single-space-padded combinators (`"div.marpit > svg"`)
// and single-space-padded function-arg commas (`":is(h1, marp-h1)"`), so
// String reinserts that canonical padding here — the one place selector
// data is turned back into text — rather than requiring every token
// builder in scope.go/root.go to hand-construct WhitespaceTokens around
// every combinator/comma it emits. A comma reaching String is guaranteed
// (by SplitList's contract) to be a nested function-arg comma, never a
// top-level list separator, since SplitList always consumes those; a
// top-level list is instead re-joined via JoinList, which uses a bare
// "," with NO padding, matching the corpus's minified list separator.
//
// Any other token (idents, colons, strings, hashes, brackets, `.`/`*`/`=`
// delimiters, the FunctionToken itself, etc.) is written byte-for-byte —
// this preserves CSS escape sequences like `\20 ` in `:not([\20 root])`
// verbatim, exactly as required.
func String(tokens []css.Token) string {
	var b strings.Builder
	for _, t := range tokens {
		switch {
		case t.TokenType == css.DelimToken && isCombinatorDelim(t.Data):
			b.WriteByte(' ')
			b.Write(t.Data)
			b.WriteByte(' ')
		case t.TokenType == css.CommaToken:
			b.Write(t.Data)
			b.WriteByte(' ')
		default:
			b.Write(t.Data)
		}
	}
	return b.String()
}

// isCombinatorDelim reports whether a DelimToken's data is one of the
// three explicit CSS combinator characters (child, sibling, adjacent-
// sibling). Every other DelimToken use (class-selector `.`, universal
// `*`, attribute `=`, etc.) must NOT get space padding — it fuses
// directly onto its neighbors.
func isCombinatorDelim(data []byte) bool {
	return len(data) == 1 && (data[0] == '>' || data[0] == '~' || data[0] == '+')
}

// JoinList re-joins a selector list's already-processed compounds (e.g.
// the output of SplitList, each independently scoped) into a single
// selector-list string, using a bare "," separator with NO padding —
// matching the corpus's minified top-level list-separator convention
// (contrast the single-space-padded comma used INSIDE a function's
// arguments, which String applies automatically).
func JoinList(compounds [][]css.Token) string {
	parts := make([]string, len(compounds))
	for i, c := range compounds {
		parts[i] = String(c)
	}
	return strings.Join(parts, ",")
}
