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

package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// HeadingDividerKey is the parser.Context key chase/markdown reads its
// resolved headingDivider levels from. The value MUST already be a fully
// resolved []int of qualifying heading levels (e.g. headingDivider: 2 ->
// []int{1, 2}, already expanded by chase/directive.CoerceGlobal's
// coerceHeadingDivider) -- this transformer does not re-derive the
// scalar-to-range expansion itself.
//
// A nil/absent value or an empty []int disables headingDivider entirely
// (no synthetic breaks are inserted); this is also what a bare
// "headingDivider: false" coercion should resolve to upstream.
//
// This is a parser.Context key (not a New() constructor Option) so the
// value can vary per-parse, mirroring the two-phase seam's own
// parser.WithContext(pc) injection pattern (01-RESEARCH.md
// "headingDivider ordering").
var HeadingDividerKey = parser.NewContextKey()

// headingDividerTransformer inserts a synthetic *ast.ThematicBreak before
// every *ast.Heading whose level qualifies per HeadingDividerKey, so the
// slide-split transformer (priority 200, run AFTER this one at priority
// 100) can consume ALL breaks -- synthetic and author-written "---" alike
// -- uniformly, with zero special-case code in slide.go
// (01-RESEARCH.md "headingDivider ordering"; must_haves: "headingDivider
// inserts synthetic breaks before qualifying headings, then the
// slide-splitter consumes ALL breaks uniformly").
type headingDividerTransformer struct{}

// newHeadingDividerTransformer returns chase/markdown's headingDivider
// parser.ASTTransformer.
func newHeadingDividerTransformer() parser.ASTTransformer {
	return &headingDividerTransformer{}
}

// Transform implements parser.ASTTransformer.
func (t *headingDividerTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	levels := resolveHeadingDividerLevels(pc)
	if len(levels) == 0 {
		return
	}

	var next ast.Node
	for c := doc.FirstChild(); c != nil; c = next {
		// Capture NextSibling before any mutation below -- InsertBefore
		// does not disturb c's own NextSibling pointer, but this guards
		// against any future change to that invariant.
		next = c.NextSibling()

		h, ok := c.(*ast.Heading)
		if !ok {
			continue
		}
		if !levelQualifies(levels, h.Level) {
			continue
		}

		prev := c.PreviousSibling()
		if prev == nil {
			// Never insert a break before the document's very first child,
			// even if it is a qualifying heading -- that would produce an
			// empty leading slide (confirmed against the
			// marp-heading-divider corpus fixture: 3 expected sections,
			// not 4).
			continue
		}
		if _, alreadyBreak := prev.(*ast.ThematicBreak); alreadyBreak {
			// Avoid a double break if a break (synthetic or authored)
			// already immediately precedes this heading.
			continue
		}

		doc.InsertBefore(doc, c, ast.NewThematicBreak())
	}
}

// resolveHeadingDividerLevels reads and type-asserts HeadingDividerKey out
// of pc, returning nil if absent, nil, or not a []int (e.g. a bare `false`
// sentinel -- headingDivider disabled).
func resolveHeadingDividerLevels(pc parser.Context) []int {
	v := pc.Get(HeadingDividerKey)
	if v == nil {
		return nil
	}
	levels, ok := v.([]int)
	if !ok {
		return nil
	}
	return levels
}

// levelQualifies reports whether level is present in levels.
func levelQualifies(levels []int, level int) bool {
	for _, l := range levels {
		if l == level {
			return true
		}
	}
	return false
}
