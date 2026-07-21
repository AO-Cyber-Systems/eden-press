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

// CORE-09 (03-07): auto-fit + shrink MARKER emission -- a
// marker/attribute-materialization battery, NOT a rendering battery. This
// file emits two stable, sanitize-survivable markers a viewer-side helper
// (themes/browser-fit.js, vendored by TRD 03-02) reads at DOM-load time:
//
//  1. A fitting header ("# <!--fit-->") gets its marker comment stripped and
//     a `data-auto-scaling="fit"` attribute added directly to the <hN>
//     element.
//  2. A fenced-code block or a "$$...$$" math-shaped paragraph gets wrapped
//     in a `<div class="marp-fit-shrink">...</div>`.
//
// `@auto-scaling` (the theme front-matter directive) is NOT handled here at
// all -- it is already parsed into theme metadata by chase/theme/meta.go
// (THEME-02) and lives entirely in theme CSS. This file never emits runtime
// JavaScript and never implements a layout pass; it only materializes the
// two markers browser-fit.js's own (already-vendored) JS looks for. The
// exact marker shape is a documented BASELINE -- Objective 8 owns any
// hardening.
package press

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

const (
	// autofitFitAttrName / autofitFitAttrValue are the stable attribute
	// name/value pair a fitting header ("# <!--fit-->") gets, once its own
	// marker comment is stripped. This is the BASELINE marker shape
	// (attribute on a real element, not a bare comment) chosen specifically
	// because it is the shape most likely to survive the 03-08 sanitize
	// allow-list -- co-design note for that TRD.
	autofitFitAttrName  = "data-auto-scaling"
	autofitFitAttrValue = "fit"

	// autofitShrinkClass is the stable class wrapping a fenced-code or
	// math-shaped block flagged for shrink. Like the fit attribute above, a
	// class on a real wrapping element is the sanitize-survivable shape
	// (03-08 co-design note).
	autofitShrinkClass = "marp-fit-shrink"
)

// autofitOption returns a goldmark.Option that registers the CORE-09
// auto-fit + shrink marker emitter. It is composed as a goldmark.Extender
// (mirroring chase/markdown.New()'s own Extend shape) purely because
// goldmark.Option is an unexported func type this package cannot construct
// directly when both a parser.ASTTransformer AND a renderer.NodeRenderer
// need registering from one call -- goldmark.WithExtensions(Extender) is
// the one public constructor that lets a single Extend method touch both
// Parser().AddOptions and Renderer().AddOptions.
//
// 03-09 folds this directly into press.Render's own
// markdown.NewEngine(pressExtraOpts...) call; it is never used standalone
// against a bare goldmark.New(), since it depends on chase/markdown's own
// comment-detection extension (markdown.CommentInline) already having run.
func autofitOption() goldmark.Option {
	return goldmark.WithExtensions(newAutofitExtender())
}

// autofitExtender is the goldmark.Extender autofitOption() wraps.
type autofitExtender struct{}

func newAutofitExtender() goldmark.Extender {
	return &autofitExtender{}
}

// Extend implements goldmark.Extender.
func (e *autofitExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		// Priority 500: strictly AFTER chase/markdown's own baked-in
		// transformers (headingDivider 100, slideSplit 200, directiveApply
		// 300, svgTransformer 400, seam.go/markdown.go) so this transformer
		// always sees the FINAL tree shape (Section wrapping, inline-SVG
		// foreignObject wrapping already applied) -- it never needs to care
		// where in that structure a Heading/FencedCodeBlock ends up, only
		// that it eventually finds them via a full-tree descent.
		parser.WithASTTransformers(
			util.Prioritized(newAutofitTransformer(), 500),
		),
	)
	m.Renderer().AddOptions(
		// Priority 0: lower than goldmark's default html.NewRenderer()
		// (renderer.DefaultRenderer, 1000), matching chase/markdown's own
		// NewNodeRenderer() registration pattern (render.go) -- this
		// package's one new NodeKind (the shrink wrapper) is the only kind
		// it ever registers a func for; every other kind still falls
		// through untouched.
		renderer.WithNodeRenderers(
			util.Prioritized(newAutofitNodeRenderer(), 0),
		),
	)
}

// autofitTransformer implements parser.ASTTransformer: it walks the
// finalized tree looking for a fitting header and shrink-candidate blocks.
type autofitTransformer struct{}

func newAutofitTransformer() parser.ASTTransformer {
	return &autofitTransformer{}
}

// Transform implements parser.ASTTransformer. It deliberately does NOT use
// ast.Walk for the mutating descent: ast.Walk's helper captures
// n.NextSibling() only AFTER a node's own children have been visited
// (ast/ast.go walkHelper), so re-parenting a node (the shrink wrap below)
// WHILE ast.Walk is mid-traversal of that node's OWN parent would corrupt
// the parent's remaining sibling chain. Mirroring
// headingDividerTransformer's own pattern (chase/markdown/headingdivider.go
// -- capture NextSibling BEFORE mutating), autofitWalk below captures next
// itself and recurses depth-first by hand, so a wrap-in-place mutation
// never disturbs traversal of the node's ORIGINAL siblings.
func (t *autofitTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	autofitWalk(doc, source)
}

// autofitWalk recurses over parent's children, applying the fit-marker
// transform to headings and the shrink-wrap transform to fenced-code/math
// blocks. next is captured before any mutation touches the current child,
// exactly like headingDividerTransformer's own loop -- see Transform's doc
// comment above for why this matters.
func autofitWalk(parent ast.Node, source []byte) {
	var next ast.Node
	for c := parent.FirstChild(); c != nil; c = next {
		next = c.NextSibling()

		switch v := c.(type) {
		case *ast.Heading:
			applyFitMarker(v)
		case *ast.FencedCodeBlock:
			wrapWithShrinkMarker(parent, v)
		case *ast.Paragraph:
			if isMathBlockParagraph(v, source) {
				wrapWithShrinkMarker(parent, v)
			}
		}

		// Recurse into c's OWN children -- unaffected by the wrap above,
		// which only changes c's PARENT, never c's own child list.
		autofitWalk(c, source)
	}
}

// applyFitMarker detects a fitting header: a Heading whose inline content
// carries a *markdown.CommentInline node with the literal raw body "fit"
// (i.e. the source comment was exactly "<!--fit-->", chase/markdown's own
// comment detection already having produced this node -- see
// chase/markdown/comment.go). When found, the marker comment is REMOVED
// from the tree (never rendered, matching the "comment consumed" contract)
// and the stable fit attribute is set directly on the heading.
//
// A heading with no such marker is left completely untouched -- carries no
// fit attribute at all.
func applyFitMarker(h *ast.Heading) {
	for c := h.FirstChild(); c != nil; c = c.NextSibling() {
		ci, ok := c.(*markdown.CommentInline)
		if !ok || strings.TrimSpace(ci.Raw) != "fit" {
			continue
		}
		h.RemoveChild(h, ci)
		h.SetAttributeString(autofitFitAttrName, []byte(autofitFitAttrValue))
		return
	}
}

// mathBlockMarker is the literal "$$" delimiter a block-math paragraph
// starts and ends with. CORE-08 (a separate, not-yet-built battery) will own
// actual LaTeX-to-MathML conversion; this baseline only needs to RECOGNIZE
// the "$$...$$" block shape well enough to flag it for shrink, since no math
// AST node/extension exists yet in this codebase.
var mathBlockMarker = []byte("$$")

// isMathBlockParagraph reports whether p is a block-level "$$...$$" math
// paragraph: exactly one child, an *ast.Text node whose raw source segment
// both starts and ends with the literal "$$" delimiter (and is long enough
// that the opening/closing markers do not overlap).
func isMathBlockParagraph(p *ast.Paragraph, source []byte) bool {
	if p.ChildCount() != 1 {
		return false
	}
	t, ok := p.FirstChild().(*ast.Text)
	if !ok {
		return false
	}
	val := t.Segment.Value(source)
	return len(val) >= 4 &&
		bytes.HasPrefix(val, mathBlockMarker) &&
		bytes.HasSuffix(val, mathBlockMarker)
}

// wrapWithShrinkMarker replaces node (a child of parent) with a new
// autofitShrinkNode wrapper, then re-parents node as that wrapper's sole
// child -- node's own children/content are completely untouched, only its
// position in the tree changes.
func wrapWithShrinkMarker(parent ast.Node, node ast.Node) {
	wrapper := newAutofitShrinkNode()
	parent.ReplaceChild(parent, node, wrapper)
	wrapper.AppendChild(wrapper, node)
}

// kindAutofitShrink is the ast.NodeKind of the synthetic shrink-wrap node
// this package introduces.
var kindAutofitShrink = ast.NewNodeKind("PressAutofitShrink")

// autofitShrinkNode is a synthetic block-level wrapper introduced purely to
// carry the shrink marker class around an existing fenced-code or math
// block, without needing to reimplement that child's own rendering (e.g.
// *ast.FencedCodeBlock's default renderer does not consult
// node.Attributes() at all, unlike *ast.Heading's -- wrapping sidesteps that
// asymmetry entirely).
type autofitShrinkNode struct {
	ast.BaseBlock
}

func newAutofitShrinkNode() *autofitShrinkNode {
	return &autofitShrinkNode{}
}

// Kind implements ast.Node.
func (n *autofitShrinkNode) Kind() ast.NodeKind {
	return kindAutofitShrink
}

// Dump implements ast.Node.
func (n *autofitShrinkNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// autofitNodeRenderer is the renderer.NodeRenderer autofitOption()
// registers -- its one job is wrapping an already-shrink-marked child in
// `<div class="marp-fit-shrink">...</div>`; every other NodeKind (Heading
// included -- the fit attribute rides goldmark's own default renderer,
// which already writes out node.Attributes() unconditionally) falls through
// untouched.
type autofitNodeRenderer struct{}

func newAutofitNodeRenderer() renderer.NodeRenderer {
	return &autofitNodeRenderer{}
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *autofitNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindAutofitShrink, r.renderShrinkWrapper)
}

func (r *autofitNodeRenderer) renderShrinkWrapper(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="`)
		_, _ = w.WriteString(autofitShrinkClass)
		_, _ = w.WriteString(`">`)
	} else {
		_, _ = w.WriteString(`</div>`)
	}
	return ast.WalkContinue, nil
}
