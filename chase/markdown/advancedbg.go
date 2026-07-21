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

// Advanced-background 3-layer structure (PARSE-06/PARSE-07, inline-SVG mode
// only): when a slide carries one or more `![bg ...]` images AND inline-SVG
// mode is enabled, the slide's single <svg> wraps THREE <foreignObject>
// layers in strict DOM order -- background, content, pseudo -- mirroring
// markdown/background_image/advanced.js's own
// `md.core.ruler.after('marpit_directives_apply', 'marpit_advanced_background', ...)`
// pass, byte-for-byte verified against conformance/corpus/cases/marp-bg-image
// and marp-bg-split's expected.html fixtures.
package markdown

import (
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// KindBackgroundLayer is the ast.NodeKind of a *BackgroundLayer node.
var KindBackgroundLayer = ast.NewNodeKind("MarpitAdvancedBackgroundLayer")

// BackgroundLayer is the advanced-background structure's FIRST
// <foreignObject> layer: a synthetic `<section>` (never carrying an `id`)
// containing a `<div data-marpit-advanced-background-container
// data-marpit-advanced-background-direction="...">` with one `<figure>` per
// background image.
type BackgroundLayer struct {
	ast.BaseBlock

	SectionAttrs []Attr
	Direction    string
	Images       []bgImage
}

func newBackgroundLayer(attrs []Attr, direction string, images []bgImage) *BackgroundLayer {
	return &BackgroundLayer{SectionAttrs: attrs, Direction: direction, Images: images}
}

// Kind implements ast.Node.
func (n *BackgroundLayer) Kind() ast.NodeKind { return KindBackgroundLayer }

// Dump implements ast.Node.
func (n *BackgroundLayer) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Direction": n.Direction}, nil)
}

// KindPseudoLayer is the ast.NodeKind of a *PseudoLayer node.
var KindPseudoLayer = ast.NewNodeKind("MarpitAdvancedPseudoLayer")

// PseudoLayer is the advanced-background structure's THIRD <foreignObject>
// layer: an empty synthetic `<section>` (never carrying an `id`) that
// stands in for Marpit's `section::after` pseudo-element in inline-SVG
// mode, since CSS pseudo-elements cannot target a real <section> hidden
// inside a <foreignObject> the same way.
type PseudoLayer struct {
	ast.BaseBlock

	SectionAttrs []Attr
}

func newPseudoLayer(attrs []Attr) *PseudoLayer {
	return &PseudoLayer{SectionAttrs: attrs}
}

// Kind implements ast.Node.
func (n *PseudoLayer) Kind() ast.NodeKind { return KindPseudoLayer }

// Dump implements ast.Node.
func (n *PseudoLayer) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{}, nil)
}

// wrapAdvancedBackgroundSvg replaces sec (a direct child of doc) with the
// full 3-layer advanced-background structure:
//
//	<svg viewBox="0 0 W H">
//	  <foreignObject width=W height=H>            <BackgroundLayer/>  </foreignObject>
//	  <foreignObject width=... height=H [x=...]>   sec (mutated)       </foreignObject>
//	  <foreignObject width=W height=H data-marpit-advanced-background="pseudo">
//	    <PseudoLayer/>
//	  </foreignObject>
//	</svg>
func wrapAdvancedBackgroundSvg(doc *ast.Document, sec *Section, data backgroundSlideData, width, height int) {
	w := strconv.Itoa(width)
	h := strconv.Itoa(height)

	svg := NewSvg(width, height)
	doc.InsertBefore(doc, sec, svg)

	bgFO, contentFO, pseudoFO := buildAdvancedBackground(sec, data, w, h)

	// Reparent sec (still carrying its real id + real content) under the
	// content layer -- AppendChild's ensureIsolated detaches sec from doc
	// automatically.
	contentFO.AppendChild(contentFO, sec)

	svg.AppendChild(svg, bgFO)
	svg.AppendChild(svg, contentFO)
	svg.AppendChild(svg, pseudoFO)
}

// buildAdvancedBackground mutates sec's own Attrs in place (content-layer
// attrs: data-marpit-advanced-background="content", plus the split attr and
// merged --marpit-advanced-background-split style declaration when the
// slide's background is split), then derives the background/pseudo layers'
// Attrs as independent snapshots of that SAME mutated set, with ONLY
// data-marpit-advanced-background overridden per layer (background/pseudo)
// -- everything else (split attr, style) carried through unchanged. This
// spread-and-override shape mirrors advanced.js's own
// `{...attrs, 'data-marpit-advanced-background': 'background'}` construction
// verbatim.
func buildAdvancedBackground(sec *Section, data backgroundSlideData, w, h string) (bgFO, contentFO, pseudoFO *ForeignObject) {
	sec.Attrs = append(sec.Attrs, Attr{Name: "data-marpit-advanced-background", Value: "content"})

	contentWidth := w
	contentX := ""
	if data.SplitSide != "" {
		size := data.SplitSize
		if size == "" {
			size = "50%"
		}
		sec.Attrs = append(sec.Attrs, Attr{Name: "data-marpit-advanced-background-split", Value: data.SplitSide})
		sec.Attrs = mergeStyleDecl(sec.Attrs, "--marpit-advanced-background-split", size)

		contentWidth = reducedPercent(size)
		if data.SplitSide == "left" {
			contentX = size
		}
	}

	base := sec.Attrs

	direction := data.Direction
	if direction == "" {
		direction = "horizontal"
	}
	bgAttrs := overrideAttr(cloneAttrs(base), "data-marpit-advanced-background", "background")
	bg := newBackgroundLayer(bgAttrs, direction, data.Images)
	bgFO = NewForeignObject(w, h)
	bgFO.AppendChild(bgFO, bg)

	contentFO = NewForeignObject(contentWidth, h)
	contentFO.X = contentX

	pseudoAttrs := overrideAttr(cloneAttrs(base), "data-marpit-advanced-background", "pseudo")
	pseudoAttrs = overrideAttr(pseudoAttrs, "style", pseudoColorStyle(sec))
	pseudo := newPseudoLayer(pseudoAttrs)
	pseudoFO = NewForeignObject(w, h)
	pseudoFO.DataAttr = "pseudo"
	pseudoFO.AppendChild(pseudoFO, pseudo)

	return bgFO, contentFO, pseudoFO
}

// reducedPercent returns "100 - size" for a bare "NN%"/"NN.N%" string (the
// advanced-background content layer's split width, e.g. split @40% ->
// content width 60%), falling back to size unchanged if it cannot be
// parsed as a percentage.
func reducedPercent(size string) string {
	numStr := strings.TrimSuffix(size, "%")
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return size
	}
	return strconv.FormatFloat(100-f, 'f', -1, 64) + "%"
}

// pseudoColorStyle extracts the "color" declaration (if any) from sec's
// current style attribute and returns it re-serialized as "color:VALUE;",
// or "" if no color declaration is present -- the pseudo layer's style
// attribute mirrors ONLY the color directive's effect (Marpit's
// section::after pseudo-element inherits color, nothing else), and is
// ALWAYS rendered (possibly as style="") per advanced.js/expected.html.
func pseudoColorStyle(sec *Section) string {
	for _, a := range sec.Attrs {
		if a.Name != "style" {
			continue
		}
		for _, part := range strings.Split(a.Value, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 && kv[0] == "color" {
				return "color:" + kv[1] + ";"
			}
		}
	}
	return ""
}
