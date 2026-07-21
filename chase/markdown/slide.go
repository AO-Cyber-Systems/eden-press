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

// SlideMetaKey is the parser.Context key the slide-split transformer
// stashes the resolved slide count under, for the future carry-forward
// pass (TRD 01-06) to read (index, total) against.
var SlideMetaKey = parser.NewContextKey()

// SlideMeta is the value stored at SlideMetaKey.
type SlideMeta struct {
	// Total is the number of *Section slides the document was split into.
	Total int
}

// slideSplitTransformer splits the document's top-level children into
// *Section runs on every *ast.ThematicBreak sibling (dropping the break
// itself), wrapping each run in a <section> AST node.
//
// It runs at priority 200, AFTER headingDividerTransformer (priority 100),
// so it observes synthetic thematic breaks inserted for headingDivider
// alongside any author-written "---" breaks -- and consumes ALL of them
// uniformly, with no special-case code (must_haves: "headingDivider ...
// then the slide-splitter consumes ALL breaks uniformly").
//
// It requires NO setext-H2 special-casing: goldmark's own block-parser
// priority order (SetextHeadingParser=100 runs before
// ThematicBreakParser=200) means a genuine "Title\n---" is ALREADY
// consumed into an *ast.Heading before this transformer ever runs -- every
// remaining *ast.ThematicBreak sibling is guaranteed to be a real slide
// separator (01-RESEARCH.md "setext-H2 trap"; must_haves: "the setext-H2
// trap is resolved correctly ... goldmark's block-parser precedence
// pre-resolves this before the transformer runs").
type slideSplitTransformer struct{}

// newSlideSplitTransformer returns chase/markdown's slide-split
// parser.ASTTransformer.
func newSlideSplitTransformer() parser.ASTTransformer {
	return &slideSplitTransformer{}
}

// Transform implements parser.ASTTransformer.
func (t *slideSplitTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	// Collect top-level children into slide "runs", splitting on
	// *ast.ThematicBreak (dropping the break node itself -- it carries no
	// content of its own once it has done its job of marking a boundary).
	var runs [][]ast.Node
	var current []ast.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if _, isBreak := c.(*ast.ThematicBreak); isBreak {
			runs = append(runs, current)
			current = nil
			continue
		}
		current = append(current, c)
	}
	runs = append(runs, current)

	// Detach every original child from doc -- ast.go's ensureIsolated (used
	// by AppendChild below) only re-parents a node whose Parent() != nil,
	// and RemoveChildren nils Parent/PreviousSibling/NextSibling for every
	// removed child, so re-appending each detached node under a new
	// *Section parent below is safe.
	doc.RemoveChildren(doc)

	total := len(runs)
	for i, run := range runs {
		section := NewSection(i + 1)
		for _, n := range run {
			section.AppendChild(section, n)
		}
		doc.AppendChild(doc, section)
	}

	pc.Set(SlideMetaKey, &SlideMeta{Total: total})
}
