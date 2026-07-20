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

// Package cssdiff implements the normalized, order-preserving CSS-AST model
// used to make real changes in theme-CSS output mechanically detectable.
//
// tdewolff/parse/v2/css exposes a grammar-TOKEN stream, not a navigable AST
// — there is no off-the-shelf Go CSS-AST diff library and no p.AST() call to
// consume — so this package materializes its own thin Stylesheet/Rule/
// Declaration model on top of that stream (see build.go's Parse function)
// rather than depending on one.
//
// Declaration and rule ORDER is cascade-significant in CSS and is therefore
// preserved exactly as authored; only within-node values are normalized
// (hex color casing, whitespace, comments, quote style). This is a
// deliberate departure from general-purpose, order-independent stylesheet-
// equivalence tooling: for verifying the same theme-CSS engine's output
// across a change, a reordered declaration is a real, detectable change.
//
// This package is the CONF-03 spike: it proves the model + grammar-stream
// builder make declaration-value changes and reorders detectable at the
// model level (see spike_test.go). The full comparator and theme negative
// tests are built on top of this model in a later objective.
package cssdiff

// Declaration is a single normalized CSS property/value pair within a Rule.
//
// Important reports whether the declaration carried a trailing
// "!important" in the source; when true, Value has already had the
// "!important" marker stripped.
type Declaration struct {
	Property  string
	Value     string
	Important bool
}

// Rule is a single CSS ruleset: a selector plus its ordered Declarations.
//
// AtRule carries the raw, minimally-processed text of the at-rule this Rule
// was found under (e.g. "@media(max-width:400px)"), or is empty for a plain
// ruleset. At-rule semantics (nested-rule modeling, @media/@keyframes-
// specific behavior) are deliberately out of scope for this spike; AtRule
// exists only so the shape is forward-compatible without implementing that
// semantics now (see 00-RESEARCH.md Open Question 2). When Rule itself
// represents a standalone at-rule statement (e.g. "@import url(x.css);"),
// Selector and Declarations are both zero-valued.
type Rule struct {
	Selector     string
	Declarations []Declaration
	AtRule       string
}

// Stylesheet is the normalized, order-preserving model of a parsed CSS
// stylesheet: an ordered list of Rules exactly as they appear in the
// source, with within-node normalization applied (see Parse in build.go).
// Rule order is never sorted or otherwise reshuffled.
type Stylesheet struct {
	Rules []Rule
}
