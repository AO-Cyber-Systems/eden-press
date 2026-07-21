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
	"io"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// Parse builds a Stylesheet from raw CSS by walking the tdewolff/parse/v2/css
// grammar-TOKEN stream (see stylesheet.go's package doc — there is no
// p.AST() to consume, so this function materializes Rules/Atoms itself via
// the documented p.Next()/p.Values() walk, the same approach
// conformance/cssdiff already proved out — see that package's Parse, whose
// selector/value-joined-to-string model this one deliberately does NOT
// reuse, see stylesheet.go's package doc for why).
//
// Parse is PURELY STRUCTURAL: it never requires, reads, or validates
// @theme/@size/@auto-scaling metadata, so a plain, theme-less CSS string
// parses through it without error. @theme's required-ness (THEME-02) is
// enforced one layer up, by meta.go's ParseMeta/ParseTheme — the TRD's
// test-list cases 1-4 all exercise plain CSS with no theme header at all,
// which would break here if Parse itself demanded @theme.
//
// Nested rulesets (CSS Nesting's implicit "&" child-combinator shorthand,
// e.g. `section { & h1 { ... } }`) are captured as Rule.Children with an
// incrementing NestingDepth, never flattened — down-leveling nesting into
// flat selectors is TRD 01-04's pipeline concern, not this one's (see the
// TRD's anti_patterns).
//
// Rule and at-rule order is preserved exactly as authored: CSS cascade
// order is cascade-significant, so nothing here sorts or set-compares
// rules, declarations, or atoms.
func Parse(cssText string) (Stylesheet, error) {
	p := css.NewParser(parse.NewInputString(cssText), false)

	var sheet Stylesheet

	// stack holds the ancestor chain of currently open rulesets, deepest
	// last (nil for the top level). A nil *Rule entry marks a nested
	// ruleset found INSIDE an at-rule block (atDepth > 0 when it was
	// opened) — recording an at-rule block's contents is out of this
	// TRD's scope (see AtRule's doc) — pushed only so its matching
	// EndRulesetGrammar pops the stack symmetrically.
	var stack []*Rule
	var atDepth int

	for {
		gt, _, data := p.Next()

		switch gt {
		case css.ErrorGrammar:
			if err := p.Err(); err != nil && err != io.EOF {
				return Stylesheet{}, err
			}
			return sheet, nil

		case css.CommentGrammar:
			// A top-level comment (e.g. the theme's leading `/** @theme
			// ... */` metadata block) — not modeled here. meta.go's
			// ParseMeta scans the raw source text directly for that,
			// independent of this grammar walk (see this function's doc).

		case css.AtRuleGrammar:
			// A standalone at-rule statement, e.g. `@import "x.css";` or
			// `@import-theme "y";`. RECORDED, never resolved/interpreted
			// — see AtRule's doc.
			if atDepth == 0 && len(stack) == 0 {
				sheet.Atoms = append(sheet.Atoms, newAtRule(data, p.Values()))
			}

		case css.BeginAtRuleGrammar:
			// The opening of a block at-rule, e.g. `@media (...) {`. Only
			// the outermost at-rule is recorded; its block contents
			// (including any nested rulesets) are never modeled — see
			// AtRule's doc.
			if atDepth == 0 && len(stack) == 0 {
				sheet.Atoms = append(sheet.Atoms, newAtRule(data, p.Values()))
			}
			atDepth++

		case css.EndAtRuleGrammar:
			if atDepth > 0 {
				atDepth--
			}

		case css.BeginRulesetGrammar, css.QualifiedRuleGrammar:
			if atDepth > 0 {
				stack = append(stack, nil)
				continue
			}
			newRule := Rule{
				SelectorTokens: cloneTokens(p.Values()),
				NestingDepth:   len(stack),
			}
			if len(stack) == 0 {
				sheet.Rules = append(sheet.Rules, newRule)
				stack = append(stack, &sheet.Rules[len(sheet.Rules)-1])
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, newRule)
				stack = append(stack, &parent.Children[len(parent.Children)-1])
			}

		case css.DeclarationGrammar:
			if atDepth > 0 || len(stack) == 0 || stack[len(stack)-1] == nil {
				continue
			}
			value, important := extractImportant(cloneTokens(p.Values()))
			current := stack[len(stack)-1]
			current.Declarations = append(current.Declarations, Declaration{
				Property:  string(data),
				Value:     value,
				Important: important,
			})

		case css.CustomPropertyGrammar:
			// A custom property (`--name: value;`). tdewolff gives this
			// its own GrammarType — never DeclarationGrammar — since a
			// custom property's value is captured as a single raw
			// CustomPropertyValueToken rather than parsed as a CSS value
			// (verified against the pinned v2.8.13 source's
			// parseCustomProperty). Modeled as an ordinary Declaration
			// here regardless; chase/theme has no need to distinguish the
			// two at this layer.
			if atDepth > 0 || len(stack) == 0 || stack[len(stack)-1] == nil {
				continue
			}
			current := stack[len(stack)-1]
			current.Declarations = append(current.Declarations, Declaration{
				Property: string(data),
				Value:    cloneTokens(p.Values()),
			})

		case css.EndRulesetGrammar:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case css.TokenGrammar:
			// A stray top-level token (e.g. a CDO/CDC "<!-- -->" marker,
			// or a token inside an unrecognized at-rule's opaque body):
			// not modeled.
		}
	}
}

// newAtRule builds a recorded AtRule from an AtRuleGrammar/BeginAtRuleGrammar
// event's name bytes (already lowercased by css.Parser) and prelude tokens.
// The prelude is read directly via tokensText WITHOUT going through
// cloneTokens: unlike SelectorTokens/Declaration.Value, Prelude is a plain
// string built and fully consumed within this single call, so it never
// outlives the Values() buffer it was read from — see cloneTokens' doc for
// when cloning is (and is not) required.
func newAtRule(name []byte, values []css.Token) AtRule {
	return AtRule{
		Name:    strings.TrimPrefix(string(name), "@"),
		Prelude: tokensText(values),
	}
}

// cloneTokens deep-clones a token slice AND every token's own Data byte
// slice.
//
// This is NOT optional: p.Values() — and each Token.Data slice within it —
// are backed by the css.Parser/Lexer's OWN REUSED internal buffers,
// confirmed both by reading the pinned tdewolff/parse/v2@v2.8.13 source and
// by a throwaway reproduction: storing raw p.Values() slices across loop
// iterations (without cloning) silently corrupts earlier-stored
// selector/value tokens once a later Next() call overwrites the same
// backing array. Any token slice stored beyond the current loop iteration
// (Rule.SelectorTokens, Declaration.Value) MUST go through this helper
// first; a Prelude string built and consumed within a single call (see
// newAtRule) does not need it.
func cloneTokens(tokens []css.Token) []css.Token {
	if len(tokens) == 0 {
		return nil
	}
	out := make([]css.Token, len(tokens))
	for i, t := range tokens {
		data := make([]byte, len(t.Data))
		copy(data, t.Data)
		out[i] = css.Token{TokenType: t.TokenType, Data: data}
	}
	return out
}

// extractImportant strips a trailing "!important" marker from a
// declaration's (already-cloned) value tokens and reports whether it was
// present. css.Parser's own declaration parsing already collapses any
// whitespace surrounding "!" (verified against the pinned source's
// parseDeclaration, which explicitly drops a WhitespaceToken adjacent to a
// single-byte "!" token: "red !important" and "red ! important" tokenize
// identically), so the marker is always exactly the last two tokens: a
// DelimToken "!" immediately followed by an IdentToken "important".
func extractImportant(tokens []css.Token) ([]css.Token, bool) {
	n := len(tokens)
	if n >= 2 &&
		tokens[n-1].TokenType == css.IdentToken &&
		strings.EqualFold(string(tokens[n-1].Data), "important") &&
		tokens[n-2].TokenType == css.DelimToken &&
		string(tokens[n-2].Data) == "!" {
		return tokens[:n-2], true
	}
	return tokens, false
}
