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

// Inline-SVG mode (PARSE-07): wraps every slide's *Section in
// <svg data-marpit-svg viewBox="0 0 W H"><foreignObject width=W height=H>,
// mirroring markdown/inline_svg.js. Runs LAST (registered at priority 400 in
// markdown.go, strictly after directiveApplyTransformer's priority 300) so
// every Section already carries its directive-derived Attrs before being
// wrapped (01-06 gotcha).
//
// W/H come from SvgOptionsKey (a parser.Context key, default 1280x720) --
// NEVER a chase/theme import (RESEARCH build-order constraint: chase/markdown
// must stay theme-agnostic at parse time).
package markdown

import (
	"strconv"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// KindSvg is the ast.NodeKind of a *Svg node.
var KindSvg = ast.NewNodeKind("MarpitSvg")

// Svg is the per-slide `<svg data-marpit-svg viewBox="0 0 W H">` wrapper
// inline-SVG mode wraps every Section in.
type Svg struct {
	ast.BaseBlock

	// ViewBoxWidth/ViewBoxHeight are the resolved theme size (default
	// 1280x720).
	ViewBoxWidth  int
	ViewBoxHeight int
}

// NewSvg returns a new *Svg node with the given viewBox width/height.
func NewSvg(width, height int) *Svg {
	return &Svg{ViewBoxWidth: width, ViewBoxHeight: height}
}

// Kind implements ast.Node.
func (n *Svg) Kind() ast.NodeKind { return KindSvg }

// Dump implements ast.Node.
func (n *Svg) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"ViewBoxWidth":  strconv.Itoa(n.ViewBoxWidth),
		"ViewBoxHeight": strconv.Itoa(n.ViewBoxHeight),
	}, nil)
}

// KindForeignObject is the ast.NodeKind of a *ForeignObject node.
var KindForeignObject = ast.NewNodeKind("MarpitForeignObject")

// ForeignObject is a `<foreignObject>` layer inside a *Svg. It is reused for
// all three advanced-background layers (background/content/pseudo) as well
// as the base (non-background) wrap -- X and DataAttr are optional, used
// only by the content layer (split width/x adjustment) and the pseudo layer
// (data-marpit-advanced-background="pseudo" on the foreignObject itself)
// respectively.
type ForeignObject struct {
	ast.BaseBlock

	Width  string
	Height string

	// X is the split content layer's horizontal offset (e.g. "50%"),
	// empty otherwise.
	X string

	// DataAttr, when non-empty, renders a
	// data-marpit-advanced-background="{DataAttr}" attribute on the
	// foreignObject element itself (the pseudo layer only -- confirmed
	// against marp-bg-image/expected.html and marp-bg-split/expected.html).
	DataAttr string
}

// NewForeignObject returns a new *ForeignObject with the given width/height.
func NewForeignObject(width, height string) *ForeignObject {
	return &ForeignObject{Width: width, Height: height}
}

// Kind implements ast.Node.
func (n *ForeignObject) Kind() ast.NodeKind { return KindForeignObject }

// Dump implements ast.Node.
func (n *ForeignObject) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Width":  n.Width,
		"Height": n.Height,
		"X":      n.X,
	}, nil)
}

// SvgOptionsKey is the parser.Context key inline-SVG mode reads its
// enabled/width/height configuration from -- a parser.Context key (not a
// New() constructor Option) so it can vary per-parse, mirroring
// HeadingDividerKey's own pattern (headingdivider.go).
//
// Absent entirely -> disabled, 1280x720. chase/markdown stays opt-in here:
// callers that want the <svg><foreignObject> wrap (or the advanced
// background layer) must explicitly set SvgOptionsKey with Enabled=true --
// otherwise every pre-existing 01-05/01-06 caller that never touches this
// key keeps getting bare *Section children directly off the *ast.Document,
// exactly as it always has.
//
// A non-nil *SvgOptions with Width/Height left at zero fills in the
// 1280x720 default for whichever dimension was not overridden.
var SvgOptionsKey = parser.NewContextKey()

// SvgOptions controls inline-SVG mode for one parse.
type SvgOptions struct {
	Enabled bool
	Width   int
	Height  int
}

// resolveSvgOptions reads SvgOptionsKey out of pc, applying the 1280x720
// default (entirely, or per-dimension when only one was overridden).
func resolveSvgOptions(pc parser.Context) SvgOptions {
	if v, ok := pc.Get(SvgOptionsKey).(*SvgOptions); ok && v != nil {
		opts := *v
		if opts.Width == 0 {
			opts.Width = 1280
		}
		if opts.Height == 0 {
			opts.Height = 720
		}
		return opts
	}
	return SvgOptions{Enabled: false, Width: 1280, Height: 720}
}

// svgTransformer wraps every top-level *Section in <svg><foreignObject> when
// inline-SVG mode is enabled (the base wrap, or -- once a slide carries
// background images -- the 3-layer advanced-background structure); when
// disabled, it materializes a single bg image as a backgroundImage local
// directive instead (background.go), with no new HTML structure.
//
// Registered LAST (priority 400) in markdown.go, after
// directiveApplyTransformer (300): every Section already carries its
// directive-derived Attrs before this transformer runs.
type svgTransformer struct{}

// newSvgTransformer returns chase/markdown's inline-SVG parser.ASTTransformer.
func newSvgTransformer() parser.ASTTransformer {
	return &svgTransformer{}
}

// Transform implements parser.ASTTransformer.
func (t *svgTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	opts := resolveSvgOptions(pc)
	source := reader.Source()

	// Snapshot the document's *Section children before mutating -- every
	// top-level child is a *Section by this point (slide-split, priority
	// 200, has already run), mirroring slideSplitTransformer's own
	// detach-then-rebuild pattern.
	var sections []*Section
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if sec, ok := c.(*Section); ok {
			sections = append(sections, sec)
		}
	}

	for _, sec := range sections {
		data := extractBackgroundImages(sec, source)

		if !opts.Enabled {
			applyNonSVGBackground(sec, data)
			continue
		}

		if len(data.Images) == 0 {
			wrapBaseSvg(doc, sec, opts.Width, opts.Height)
			continue
		}

		// TRD 01-07 Task 3 (advancedbg.go) replaces this with the real
		// 3-layer background/content/pseudo structure; Task 2 only
		// establishes extraction + the base wrap.
		wrapBaseSvg(doc, sec, opts.Width, opts.Height)
	}
}

// wrapBaseSvg replaces sec (a direct child of doc) with
// <svg><foreignObject>sec</foreignObject></svg>, reparenting sec unchanged.
func wrapBaseSvg(doc *ast.Document, sec *Section, width, height int) {
	w := strconv.Itoa(width)
	h := strconv.Itoa(height)

	svg := NewSvg(width, height)
	fo := NewForeignObject(w, h)

	// Insert svg immediately before sec (still doc's child at this point),
	// then reparent sec under fo -- AppendChild's ensureIsolated detaches
	// sec from doc automatically, leaving svg exactly where sec used to be.
	doc.InsertBefore(doc, sec, svg)
	fo.AppendChild(fo, sec)
	svg.AppendChild(svg, fo)
}
