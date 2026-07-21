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

// Package markdown is chase's goldmark Extender (PARSE-05): the two-phase
// Parser().Parse() / Renderer().Render() seam, comment DETECTION (not
// recognition -- that's TRD 01-06), slide-splitting on thematic breaks +
// headingDivider, and render-time <section>/.marpit container wrapping.
//
// Callers MUST use the two-phase seam directly:
//
//	reader := text.NewReader(source)
//	doc := md.Parser().Parse(reader, parser.WithContext(pc))
//	// ... inspect / mutate doc here ...
//	md.Renderer().Render(&buf, source, doc)
//
// NEVER md.Convert() in production code paths -- Convert() collapses both
// phases into one call, leaving no seam for a caller to inspect the
// finalized AST between parse and render (01-RESEARCH.md "Two-phase
// Parse/Render seam").
package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// ext is chase/markdown's goldmark.Extender.
type ext struct{}

// New returns chase/markdown's goldmark.Extender. Register it via
// goldmark.WithExtensions(markdown.New()).
//
// headingDivider is NOT a constructor argument: it is injected per-parse via
// a parser.Context key (HeadingDividerKey, headingdivider.go) so it can vary
// per-document/per-render exactly like Marp's own front-matter-driven
// "headingDivider" global directive value does, without requiring a new
// goldmark.Markdown instance per value.
func New() goldmark.Extender {
	return &ext{}
}

// Extend implements goldmark.Extender.
func (e *ext) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		// Comment BlockParser/InlineParser are registered at priority 0 --
		// lower than goldmark's built-in HTMLBlockParser (900) and
		// RawHTMLParser (400) -- so ours wins the trigger race for "<!--"
		// (01-RESEARCH.md "HTML-Comment Directive Detection").
		parser.WithBlockParsers(
			util.Prioritized(newCommentBlockParser(), 0),
		),
		parser.WithInlineParsers(
			util.Prioritized(newCommentInlineParser(), 0),
		),
		// headingDivider (100) MUST run before slide-split (200): it
		// inserts synthetic *ast.ThematicBreak siblings that slide-split
		// then consumes uniformly alongside any author-written "---"
		// breaks (01-RESEARCH.md "headingDivider ordering").
		parser.WithASTTransformers(
			util.Prioritized(newHeadingDividerTransformer(), 100),
			util.Prioritized(newSlideSplitTransformer(), 200),
		),
	)
	m.Renderer().AddOptions(
		// Registered at priority 0 -- lower than goldmark's default
		// html.NewRenderer() (renderer.DefaultRenderer, 1000) -- so our
		// Document/Section/Comment funcs win; every other NodeKind we
		// never register a func for still falls through to the default
		// renderer.
		renderer.WithNodeRenderers(
			util.Prioritized(NewNodeRenderer(), 0),
		),
	)
}
