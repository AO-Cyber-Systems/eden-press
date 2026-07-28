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
	"fmt"

	"github.com/tdewolff/parse/v2/css"

	"github.com/AO-Cyber-Systems/eden-press/chase/theme/selector"
)

// pass_nesting.go hand-rolls a minimal CSS Nesting Level 1 down-leveler.
//
// No Go library performs this flattening (01-RESEARCH.md Pitfall 2:
// tdewolff/parse/v2/css only TOKENIZES nesting syntax — chase/theme's own
// Parse in parse.go preserves it as Rule.Children — it never flattens it),
// so this file hand-rolls exactly the shapes the synthetic stress theme
// needs, per this TRD's anti_patterns/error_recovery: implicit-"&" child
// nesting (`section { & h1 { ... } }`), "&"-compound fusion (`&.lead {
// ... }`), and a nested comma-separated selector list (`& h1, & h2 { ...
// }`). A nested rule with its OWN further-nested children (2+ levels deep)
// is a documented, deliberate scope-narrowing gap: flattenRule returns an
// error naming the offending selector rather than silently mishandling it
// or attempting general CSS Nesting spec compliance.

// passNesting is the Tier-1 Pass wrapper (see theme.go's Load) around
// flattenNesting.
var passNesting = Pass{
	Name: "nesting",
	Run: func(sheet *Stylesheet) error {
		flat, err := flattenNesting(sheet.Rules)
		if err != nil {
			return err
		}
		sheet.Rules = flat
		return nil
	},
}

// flattenNesting down-levels every (possibly-nested) Rule in rules into
// flat, top-level Rules with no Children.
func flattenNesting(rules []Rule) ([]Rule, error) {
	var out []Rule
	for _, r := range rules {
		flat, err := flattenRule(r)
		if err != nil {
			return nil, err
		}
		out = append(out, flat...)
	}
	return out, nil
}

// flattenRule flattens a single (possibly-nested) Rule into zero or more
// flat Rules: the rule's own selector+declarations (if it carries any
// direct declarations, or has no children at all) followed by each
// Children entry's selector, expanded against the parent's selector list
// via expandNestedSelector.
func flattenRule(r Rule) ([]Rule, error) {
	// A block at-rule is a CONTAINER, not a nested ruleset: its children are
	// top-level rules that happen to live inside @media/@supports, so they are
	// flattened WITHIN it and the at-rule wrapper survives. Treating it like an
	// ordinary nested rule would both hoist its contents out of the block and
	// trip the depth>1 guard on any at-rule containing nesting.
	if r.At != nil {
		inner, err := flattenNesting(r.Children)
		if err != nil {
			return nil, err
		}
		if len(inner) == 0 && len(r.Declarations) == 0 {
			return nil, nil
		}
		return []Rule{{At: r.At, Declarations: r.Declarations, Children: inner}}, nil
	}
	var out []Rule
	if len(r.Declarations) > 0 || len(r.Children) == 0 {
		out = append(out, Rule{SelectorTokens: r.SelectorTokens, Declarations: r.Declarations})
	}
	for _, child := range r.Children {
		if len(child.Children) > 0 {
			return nil, fmt.Errorf(
				"theme: nesting depth >1 not supported (selector %q nested under %q) — "+
					"see 01-04-SUMMARY.md's documented scope-narrowing note",
				selector.String(child.SelectorTokens), selector.String(r.SelectorTokens),
			)
		}
		expanded, err := expandNestedSelector(r.SelectorTokens, child.SelectorTokens)
		if err != nil {
			return nil, err
		}
		out = append(out, Rule{SelectorTokens: expanded, Declarations: child.Declarations})
	}
	return out, nil
}

// expandNestedSelector down-levels a single nested ruleset's selector list
// (childTokens, e.g. "& h1, & h2") against its parent's selector list
// (parentTokens, e.g. a bare unit-element compound), producing the flat,
// comma-joined replacement selector token list (e.g. "<parent> h1,
// <parent> h2").
//
// Both lists are split into their top-level comma-separated compounds via
// chase/theme/selector.SplitList (depth-aware — never splits inside a
// FunctionToken/bracket, so a nested `:is(h3, h4) + p`-shaped child stays
// one compound, see substituteAmpersand); every (child compound, parent
// compound) pair is expanded via substituteAmpersand, child-major, and the
// results are joined via joinCompounds.
func expandNestedSelector(parentTokens, childTokens []css.Token) ([]css.Token, error) {
	parents := selector.SplitList(parentTokens)
	children := selector.SplitList(childTokens)
	if len(parents) == 0 || len(children) == 0 {
		return nil, fmt.Errorf("theme: nesting: empty selector list (parent %q, child %q)",
			selector.String(parentTokens), selector.String(childTokens))
	}

	var expanded [][]css.Token
	for _, child := range children {
		for _, parent := range parents {
			expanded = append(expanded, substituteAmpersand(child, parent))
		}
	}
	return joinCompounds(expanded), nil
}

// substituteAmpersand implements CSS Nesting's substitution rule for a
// single compound selector: every top-level "&" (a DelimToken "&" at
// depth 0 — chase/theme/selector.Walk's depth tracking means one found
// INSIDE a FunctionToken's arguments is deliberately left untouched, out
// of this TRD's narrowed scope) is replaced by parent's tokens, FUSED
// directly (no added combinator) — this is what makes "&.lead" fuse into
// "section.lead" while "& h1" (an explicit descendant WhitespaceToken
// already present after the "&") expands into "section h1". If compound
// contains no top-level "&" at all, CSS Nesting's implicit rule applies:
// parent is prepended as a descendant ancestor ("parent" + " " +
// compound) — this is also test-list case 3's passthrough boundary check
// (a plain, non-nested `:is(h3, h4) + p` compound never reaches this
// function at all, since it has no Children entry to begin with).
func substituteAmpersand(compound, parent []css.Token) []css.Token {
	var out []css.Token
	found := false
	selector.Walk(compound, func(tok css.Token, _ int, depth int) bool {
		if depth == 0 && tok.TokenType == css.DelimToken && string(tok.Data) == "&" {
			out = append(out, parent...)
			found = true
			return true
		}
		out = append(out, tok)
		return true
	})
	if found {
		return out
	}
	result := make([]css.Token, 0, len(parent)+1+len(compound))
	result = append(result, parent...)
	result = append(result, css.Token{TokenType: css.WhitespaceToken, Data: []byte(" ")})
	result = append(result, compound...)
	return result
}

// joinCompounds re-joins already-processed compound selectors (e.g.
// expandNestedSelector's or pack.go's scopeSelector's per-compound output)
// into a single selector-list token slice, inserting a bare (unpadded)
// CommaToken between each — chase/theme/selector.String re-pads it with a
// single trailing space when the list is finally rendered to text (see
// that function's doc comment on function-argument-comma vs top-level-
// list-separator conventions).
func joinCompounds(compounds [][]css.Token) []css.Token {
	var out []css.Token
	for i, c := range compounds {
		if i > 0 {
			out = append(out, css.Token{TokenType: css.CommaToken, Data: []byte(",")})
		}
		out = append(out, c...)
	}
	return out
}
