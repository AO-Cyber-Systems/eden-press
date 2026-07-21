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

// Package theme implements chase's owned CSS engine over
// github.com/tdewolff/parse/v2/css — the theme-CSS analogue of chase/directive,
// with ZERO dependency on markdown/goldmark: theme stylesheets are parsed
// completely independently of any slide content.
//
// tdewolff/parse/v2/css exposes a grammar-TOKEN stream, not a navigable AST —
// there is no p.AST() to consume — so this package materializes its own
// Stylesheet/Rule/Declaration model on top of that stream (see parse.go's
// Parse function), the same approach conformance/cssdiff already proved out.
// This package's model is deliberately RICHER than cssdiff's: Rule.Selector
// is kept as a token list ([]css.Token), not a joined string, so the
// selector-rewriter subsystem (chase/theme/selector, THEME-04) can hand-walk
// :is()/:where() FunctionToken arguments; nested rulesets are preserved as
// Rule.Children with a NestingDepth marker rather than flattened (down-
// leveling nesting is TRD 01-04's concern, not this one's).
//
// Metadata (@theme/@size/@auto-scaling) is a separate concern, extracted by
// meta.go and merged onto Stylesheet.Meta by ParseTheme.
package theme

import (
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// Declaration is a single normalized CSS property/value pair within a Rule.
//
// Value keeps the declaration's value TOKEN LIST (not a joined string) so
// later passes (rem-unit conversion, var() resolution — both TRD 01-04
// concerns) can walk it without re-tokenizing. Important reports whether the
// declaration carried a trailing "!important" in the source; when true,
// Value has already had the "!important" marker stripped.
type Declaration struct {
	Property  string
	Value     []css.Token
	Important bool
}

// String renders the declaration back to "property: value" CSS text (with a
// trailing " !important" when Important is set), in this package's
// canonical form — not necessarily byte-identical to the original source
// spacing (see Rule.String's doc for why that's the right tradeoff here).
func (d Declaration) String() string {
	value := tokensText(d.Value)
	if d.Important {
		return d.Property + ": " + value + " !important"
	}
	return d.Property + ": " + value
}

// Rule is a single CSS ruleset: a selector TOKEN LIST plus its ordered
// Declarations and any nested rulesets (Children).
//
// SelectorTokens is never flattened to a string — it must stay walkable so
// chase/theme/selector (THEME-04) can hand-walk compound selectors and
// :is()/:where() FunctionToken arguments. Children holds nested rulesets
// exactly as parsed (e.g. CSS Nesting's implicit "&" child combinator
// shorthand): this package PRESERVES nesting depth, it does not flatten it
// — down-leveling nested rules into flat selectors is TRD 01-04's pipeline
// concern (see parse.go's package doc and the TRD's anti_patterns).
//
// NestingDepth is 0 for a top-level ruleset (a direct member of
// Stylesheet.Rules) and N for a ruleset nested N levels deep inside its
// ancestors (a member of some ancestor chain of Children).
type Rule struct {
	SelectorTokens []css.Token
	Declarations   []Declaration
	Children       []Rule
	NestingDepth   int
}

// String renders the rule back to CSS text, recursively rendering any
// nested Children INSIDE the same braces (since nesting is preserved, not
// flattened, by this package). The rendering is this package's own
// canonical form (tokens joined verbatim, single space around braces and
// after each ";") rather than a byte-exact echo of the original source —
// deliberately so: source whitespace is not itself meaningful once tokenized,
// and TRD 01-03's recovery guidance is to normalize to a canonical form (the
// same tradeoff conformance/cssdiff already makes) rather than chase
// original-source byte spacing.
func (r Rule) String() string {
	var b strings.Builder
	b.WriteString(tokensText(r.SelectorTokens))
	b.WriteString(" {")
	for _, d := range r.Declarations {
		b.WriteString(" ")
		b.WriteString(d.String())
		b.WriteString(";")
	}
	for _, c := range r.Children {
		b.WriteString(" ")
		b.WriteString(c.String())
	}
	b.WriteString(" }")
	return b.String()
}

// Size is a single named slide-size entry declared via a theme's
// `@size <name> <W>px <H>px` metadata line -- a name paired with its own
// width/height in pixels.
type Size struct {
	Name     string
	WidthPx  int
	HeightPx int
}

// Meta captures a theme's identity metadata, extracted by meta.go from the
// theme CSS's leading comment block (mirroring Marpit's postcss/meta.js).
//
// Name comes from the required `@theme <name>` line — THEME-02 treats a
// theme CSS string with no `@theme` metadata as an error, never silently
// defaulting it (see meta.go's ParseMeta). Sizes is the named-size table
// built from (possibly repeated) `@size` lines; ResolveSize derives the
// active slide dimensions from it, falling back to a caller-supplied
// default size. AutoScaling is the raw `@auto-scaling` value, captured
// verbatim (no coercion — that is a rendering-layer concern, out of this
// TRD's scope). Raw is the theme's original leading comment text, kept
// for diagnostics.
type Meta struct {
	Name        string
	Sizes       map[string]Size
	AutoScaling string
	Raw         string
}

// ResolveSize returns the active slide's pixel dimensions for the named
// size. If name is empty, or no @size entry with that name was parsed, it
// returns fallback's pixel dimensions — this default is caller-supplied
// (originating in the active Profile, e.g. profiles/slides) rather than
// hardcoded here, so chase/theme carries no profile-specific size
// assumption of its own (TRD 02-03, MODEL-04's de-hardcoding move).
func (m Meta) ResolveSize(name string, fallback Size) (widthPx, heightPx int) {
	if name != "" {
		if sz, ok := m.Sizes[name]; ok {
			return sz.WidthPx, sz.HeightPx
		}
	}
	return fallback.WidthPx, fallback.HeightPx
}

// AtRule is a RECORDED (not resolved) at-rule statement or block header,
// e.g. `@import "x.css"`, `@import-theme "y"`, `@charset "utf-8"`, or an
// `@media (...) { ... }` block's opening. Resolution (import fetching,
// media-query evaluation) and modeling of a block at-rule's nested rules
// are both explicitly OUT of this TRD's scope (see the TRD's anti_patterns:
// "DO NOT attempt @import RESOLUTION here — only RECORD"); that is TRD
// 01-04's pipeline concern.
type AtRule struct {
	// Name is the at-rule name without its leading '@', lowercased (e.g.
	// "import", "import-theme", "media", "charset").
	Name string
	// Prelude is the raw prelude text as authored, trimmed (e.g.
	// `"x.css"` for @import, `(max-width: 400px)` for @media).
	Prelude string
}

// String renders a recorded at-rule back to its opening CSS text (e.g.
// `@import "x.css"`), WITHOUT a trailing ";" or "{" — Stylesheet.String
// adds those. Note this is best-effort: a block at-rule's nested contents
// were never modeled (RECORD only, see AtRule's doc), so round-tripping an
// AtRule that was originally a block (e.g. @media) will lose its body —
// this package's round-trip guarantee (TRD 01-03's Task 1 "done" criterion)
// covers plain/nested rulesets only, not at-rule block contents.
func (a AtRule) String() string {
	if a.Prelude == "" {
		return "@" + a.Name
	}
	return "@" + a.Name + " " + a.Prelude
}

// Stylesheet is chase/theme's owned, token-preserving CSS model: a theme
// CSS string parses into this Stylesheet{Meta, Rules, Atoms} shape (see
// parse.go's Parse for Rules/Atoms and meta.go's ParseTheme for the full,
// Meta-populated result).
//
// Rule and at-rule order is preserved exactly as authored — CSS cascade
// order is cascade-significant — mirroring conformance/cssdiff's own
// ordering guarantee.
type Stylesheet struct {
	Meta  Meta
	Rules []Rule
	Atoms []AtRule
}

// String renders the Stylesheet back to CSS text: recorded Atoms first (see
// AtRule.String's round-trip caveat for block at-rules), then Rules in
// order, one per line, in this package's canonical form.
func (s Stylesheet) String() string {
	var b strings.Builder
	for _, a := range s.Atoms {
		b.WriteString(a.String())
		b.WriteString(";\n")
	}
	for i, r := range s.Rules {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(r.String())
	}
	return b.String()
}

// tokensText concatenates token data in source order and trims incidental
// leading/trailing whitespace. It deliberately does NOT canonicalize quote
// style or hex-color casing the way conformance/cssdiff's joinTokens does —
// this package's String() is a structural round-trip aid for its own model
// (see Rule.String's doc), not a byte-for-byte CSS-diff comparator; that
// finer-grained canonicalization belongs to whatever TRD 01-04 layers on
// top (or to conformance/cssdiff itself, unchanged).
func tokensText(tokens []css.Token) string {
	var b strings.Builder
	for _, t := range tokens {
		b.Write(t.Data)
	}
	return strings.TrimSpace(b.String())
}
