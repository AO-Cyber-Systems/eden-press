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

import (
	"io"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// discardedRule is a ruleStack sentinel marking a nested ruleset found
// inside an at-rule block this spike deliberately does not model (see the
// AtRule handling below); declarations attributed to a discarded rule are
// dropped rather than mis-attached to an unrelated Rule.
const discardedRule = -1

// Parse builds a normalized Stylesheet from raw CSS by walking the
// tdewolff/parse/v2/css grammar-token stream.
//
// tdewolff/parse/v2/css exposes a grammar-TOKEN stream, not a navigable
// AST — there is no p.AST() to consume — so this function materializes the
// model itself via the documented p.Next()/p.Values() walk over
// css.GrammarType constants (see `go doc github.com/tdewolff/parse/v2/css`
// for the pinned v2.8.13 API this was written against).
//
// Rule and declaration order is preserved exactly as authored: CSS cascade
// order is significant, so nothing here sorts or set-compares rules or
// declarations. Only within-node normalization is applied while values are
// captured: hex colors are lowercased, redundant whitespace collapses
// (tdewolff's own lexer already collapses whitespace RUNS to a single
// token; this function does not re-introduce or discard the separators
// that survive that pass), comments are stripped (dropped at the top
// level; already excluded from selector/declaration token streams by the
// underlying lexer), and quote style is canonicalized to double quotes.
//
// At-rule contents (@media, @keyframes, @import, etc.) are captured only
// as a raw, opaque prelude string on Rule.AtRule — deliberately minimal
// per this spike's scope (00-RESEARCH.md Open Question 2); nested rules
// inside an at-rule block are not modeled individually.
func Parse(cssText string) (Stylesheet, error) {
	p := css.NewParser(parse.NewInputString(cssText), false)

	var sheet Stylesheet
	var ruleStack []int
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
			// Top-level comment between rules: stripped, not modeled.

		case css.AtRuleGrammar:
			// A standalone at-rule statement (e.g. "@import url(x.css);").
			if atDepth == 0 {
				sheet.Rules = append(sheet.Rules, Rule{AtRule: atRuleText(data, p.Values())})
			}

		case css.BeginAtRuleGrammar:
			// The opening of a block at-rule (e.g. "@media (...) {"). Only
			// the outermost at-rule we encounter gets a Rule entry; nested
			// at-rules (e.g. "@layer { @layer base {} }") just add depth.
			if atDepth == 0 {
				sheet.Rules = append(sheet.Rules, Rule{AtRule: atRuleText(data, p.Values())})
			}
			atDepth++

		case css.EndAtRuleGrammar:
			if atDepth > 0 {
				atDepth--
			}

		case css.BeginRulesetGrammar, css.QualifiedRuleGrammar:
			if atDepth > 0 {
				// Nested ruleset inside an at-rule block this spike
				// deliberately does not model; push a sentinel so the
				// matching EndRulesetGrammar can pop it symmetrically.
				ruleStack = append(ruleStack, discardedRule)
				continue
			}
			sheet.Rules = append(sheet.Rules, Rule{Selector: normalizeSelector(p.Values())})
			ruleStack = append(ruleStack, len(sheet.Rules)-1)

		case css.DeclarationGrammar:
			if atDepth > 0 || len(ruleStack) == 0 {
				continue
			}
			idx := ruleStack[len(ruleStack)-1]
			if idx == discardedRule {
				continue
			}
			sheet.Rules[idx].Declarations = append(
				sheet.Rules[idx].Declarations,
				normalizeDeclaration(string(data), p.Values()),
			)

		case css.EndRulesetGrammar:
			if len(ruleStack) > 0 {
				ruleStack = ruleStack[:len(ruleStack)-1]
			}

		case css.TokenGrammar:
			// Stray top-level token (e.g. CDO/CDC "<!-- -->" markers); not
			// modeled.
		}
	}
}

// atRuleText reconstructs an at-rule's raw name + prelude text (e.g.
// "@media(max-width:400px)") from the at-rule name returned by Next() and
// its prelude tokens from Values(), mirroring the concatenation approach
// css.Parser's own tests use to round-trip canonical CSS text.
func atRuleText(name []byte, values []css.Token) string {
	return strings.TrimSpace(string(name) + joinTokens(values))
}

// normalizeSelector builds a selector string from a ruleset's Values()
// tokens. tdewolff's lexer already collapses whitespace runs to a single
// token and drops comments while lexing a selector, so this only needs to
// concatenate tokens (canonicalizing quote style on any string literals,
// e.g. in attribute selectors) and trim incidental leading/trailing space.
func normalizeSelector(values []css.Token) string {
	return strings.TrimSpace(joinTokens(values))
}

// joinTokens concatenates token data in source order, canonicalizing quote
// style on string literals. It does not touch hex colors: a Hash token
// here may be an ID selector (case-sensitive), not a color.
func joinTokens(tokens []css.Token) string {
	var b strings.Builder
	for _, t := range tokens {
		if t.TokenType == css.StringToken {
			b.WriteString(normalizeQuote(string(t.Data)))
		} else {
			b.Write(t.Data)
		}
	}
	return b.String()
}

// normalizeDeclaration builds a normalized Declaration from a
// DeclarationGrammar event: property is the (already lowercased by
// css.Parser) property name, and value is built from the value tokens
// with a trailing "!important" extracted and within-node normalization
// applied.
func normalizeDeclaration(property string, values []css.Token) Declaration {
	values, important := extractImportant(values)
	return Declaration{
		Property:  property,
		Value:     strings.TrimSpace(joinValueTokens(values)),
		Important: important,
	}
}

// joinValueTokens is joinTokens plus hex-color lowering. Hex lowering is
// scoped to declaration VALUES only (never selectors): a Hash token in a
// value is a color literal, but a Hash token in a selector is an ID
// selector and must stay case-sensitive.
func joinValueTokens(tokens []css.Token) string {
	var b strings.Builder
	for _, t := range tokens {
		switch t.TokenType {
		case css.StringToken:
			b.WriteString(normalizeQuote(string(t.Data)))
		case css.HashToken:
			b.WriteString(strings.ToLower(string(t.Data)))
		default:
			b.Write(t.Data)
		}
	}
	return b.String()
}

// extractImportant strips a trailing "!important" marker from a
// declaration's value tokens and reports whether it was present.
// css.Parser already collapses any whitespace surrounding "!" (verified
// against its own test suite: "red !important" and "red ! important" both
// tokenize identically), so the marker is always the last two tokens: a
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

// normalizeQuote canonicalizes a CSS string token — whose Data includes
// its own surrounding quote characters — to a double-quoted form, so that
// single- and double-quoted equivalents normalize identically.
func normalizeQuote(s string) string {
	if len(s) < 2 {
		return s
	}
	quote := s[0]
	if quote != '\'' && quote != '"' {
		return s
	}
	inner := s[1 : len(s)-1]
	if quote == '\'' {
		inner = strings.ReplaceAll(inner, `"`, `\"`)
	}
	return `"` + inner + `"`
}
