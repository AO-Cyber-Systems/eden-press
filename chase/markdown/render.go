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
	"strconv"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// nodeRenderer is chase/markdown's renderer.NodeRenderer. It injects the
// ".marpit" container at RENDER time -- NOT parse time: the AST holds only
// *Section children (the seam TRD 01-06/objective-2's docmodel reuses); no
// container node ever exists in the tree (must_haves: "the container
// <div class=\"marpit\"> is injected at render time, not parse time").
//
// It is registered at a priority (0) LOWER than goldmark's default
// html.NewRenderer() (renderer.DefaultRenderer = 1000), so its
// Document/Section/Comment funcs win; every other NodeKind (Heading,
// Paragraph, Text, ...) that this renderer never registers a func for
// still falls through to the default HTML renderer untouched.
type nodeRenderer struct {
	// containerClass is the class attribute renderDocument wraps the whole
	// rendered run in. Empty means DefaultContainerClass.
	containerClass string
}

// NewNodeRenderer returns chase/markdown's renderer.NodeRenderer.
func NewNodeRenderer() renderer.NodeRenderer {
	return &nodeRenderer{}
}

// DefaultContainerClass is the container class used when no profile supplies
// one. It stays "marpit" for Marp compatibility: every conformance-corpus
// case, every bundled theme's scoped selectors, and every existing consumer
// depend on it.
const DefaultContainerClass = "marpit"

// optContainerClass is the renderer.OptionName WithContainerClass sets.
const optContainerClass renderer.OptionName = "edenPressContainerClass"

// containerClassOption is a renderer.Option carrying the container class.
type containerClassOption struct{ class string }

func (o *containerClassOption) SetConfig(c *renderer.Config) {
	c.Options[optContainerClass] = o.class
}

// WithContainerClass returns a renderer.Option setting the class attribute of
// the <div> the whole rendered document is wrapped in.
//
// This exists because chase/profile.Profile.Container() supplies only the CSS
// SELECTOR a unit is scoped under, while the DOM class was a "marpit" literal
// in renderDocument. A second profile (profiles/paged) would otherwise emit
// markup its own scoped CSS could never match -- the profile abstraction
// describing a container the renderer does not produce.
//
// Pass it through goldmark.WithRendererOptions, e.g.
//
//	markdown.NewEngine(goldmark.WithRendererOptions(
//	    markdown.WithContainerClass(p.ContainerClass())))
//
// An empty class falls back to DefaultContainerClass: emitting
// <div class=""> would silently detach every scoped rule in the stylesheet,
// which is a far worse failure than ignoring the option.
func WithContainerClass(class string) renderer.Option {
	return &containerClassOption{class: class}
}

// SetOption implements renderer.SetOptioner, which goldmark's renderer calls
// for every registered option before the first Render.
func (r *nodeRenderer) SetOption(name renderer.OptionName, value any) {
	if name == optContainerClass {
		if s, ok := value.(string); ok {
			r.containerClass = s
		}
	}
}

// RegisterFuncs implements renderer.NodeRenderer. Kept as a single
// switch-by-kind registration point so TRD 01-06 (directive attrs) and
// 01-07/08 (svg/foreignObject/background kinds) can add further Register
// calls here without restructuring anything.
func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(KindSection, r.renderSection)
	reg.Register(KindCommentNode, r.renderComment)
	reg.Register(KindCommentInline, r.renderComment)
	reg.Register(KindHeaderElement, r.renderHeader)
	reg.Register(KindFooterElement, r.renderFooter)
	reg.Register(KindSvg, r.renderSvg)
	reg.Register(KindForeignObject, r.renderForeignObject)
	reg.Register(KindBackgroundLayer, r.renderBackgroundLayer)
	reg.Register(KindPseudoLayer, r.renderPseudoLayer)
}

// writeAttrs writes each Attr in attrs as ` name="escaped-value"`, in
// order -- the same escaping/ordering renderSection already applies to
// *Section's own Attrs, shared here for the two advanced-background layer
// node kinds (neither of which is a *Section, so neither goes through
// renderSection itself).
func writeAttrs(w util.BufWriter, attrs []Attr) {
	for _, a := range attrs {
		_, _ = w.WriteString(` `)
		_, _ = w.WriteString(a.Name)
		_, _ = w.WriteString(`="`)
		_, _ = w.Write(util.EscapeHTML([]byte(a.Value)))
		_, _ = w.WriteString(`"`)
	}
}

// figureStyle builds one background image's <figure style="..."> value:
// background-image (always), background-size (if EffectiveSize resolved to
// a non-empty value), filter (if any CSS filter functions were parsed).
func figureStyle(img bgImage) string {
	style := NewInlineStyle()
	style.Set("background-image", `url("`+img.URL+`")`)
	if img.Size != "" {
		style.Set("background-size", img.Size)
	}
	if img.Filter != "" {
		style.Set("filter", img.Filter)
	}
	return style.String()
}

// renderDocument wraps the entire rendered run in
// <div class="CLASS">...</div>, where CLASS is the profile-supplied container
// class (WithContainerClass) and defaults to DefaultContainerClass ("marpit").
// The class is HTML-escaped: it reaches an attribute value, and a caller-
// supplied string must not be able to break out of it.
func (r *nodeRenderer) renderDocument(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		class := r.containerClass
		if class == "" {
			class = DefaultContainerClass
		}
		_, _ = w.WriteString(`<div class="`)
		_, _ = w.Write(util.EscapeHTML([]byte(class)))
		_, _ = w.WriteString(`">`)
	} else {
		_, _ = w.WriteString(`</div>`)
	}
	return ast.WalkContinue, nil
}

// renderSection renders a *Section as <section id="N" ...attrs>...</section>.
// Attrs is the clean hook TRD 01-06 populates (dataset/style directive
// attrs); this TRD always renders it empty.
func (r *nodeRenderer) renderSection(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	s := n.(*Section)
	if entering {
		_, _ = w.WriteString(`<section id="`)
		_, _ = w.WriteString(strconv.Itoa(s.ID))
		_, _ = w.WriteString(`"`)
		for _, a := range s.Attrs {
			_, _ = w.WriteString(` `)
			_, _ = w.WriteString(a.Name)
			_, _ = w.WriteString(`="`)
			_, _ = w.Write(util.EscapeHTML([]byte(a.Value)))
			_, _ = w.WriteString(`"`)
		}
		_, _ = w.WriteString(`>`)
	} else {
		_, _ = w.WriteString(`</section>`)
	}
	return ast.WalkContinue, nil
}

// renderComment renders a hidden CommentNode/CommentInline as nothing --
// per RESEARCH Pitfall 4, detection is separate from directive
// recognition (TRD 01-06), but EITHER way a detected comment must never
// leak into visible HTML output.
func (r *nodeRenderer) renderComment(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkSkipChildren, nil
}

// renderHeader renders a *HeaderElement (materialized by the header local
// directive, apply.go's prependHeaderElement) as <header>ESCAPED</header>.
func (r *nodeRenderer) renderHeader(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		h := n.(*HeaderElement)
		_, _ = w.WriteString(`<header>`)
		_, _ = w.Write(util.EscapeHTML([]byte(h.Content)))
	} else {
		_, _ = w.WriteString(`</header>`)
	}
	return ast.WalkContinue, nil
}

// renderFooter renders a *FooterElement (materialized by the footer local
// directive, apply.go's appendFooterElement) as <footer>ESCAPED</footer>.
func (r *nodeRenderer) renderFooter(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		f := n.(*FooterElement)
		_, _ = w.WriteString(`<footer>`)
		_, _ = w.Write(util.EscapeHTML([]byte(f.Content)))
	} else {
		_, _ = w.WriteString(`</footer>`)
	}
	return ast.WalkContinue, nil
}

// renderSvg renders a *Svg (inline-SVG mode's per-slide wrapper,
// inlinesvg.go) as <svg data-marpit-svg viewBox="0 0 W H">...</svg>.
func (r *nodeRenderer) renderSvg(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		s := n.(*Svg)
		_, _ = w.WriteString(`<svg data-marpit-svg="" viewBox="0 0 `)
		_, _ = w.WriteString(strconv.Itoa(s.ViewBoxWidth))
		_, _ = w.WriteString(` `)
		_, _ = w.WriteString(strconv.Itoa(s.ViewBoxHeight))
		_, _ = w.WriteString(`">`)
	} else {
		_, _ = w.WriteString(`</svg>`)
	}
	return ast.WalkContinue, nil
}

// renderForeignObject renders a *ForeignObject (the base wrap layer, or one
// of the three advanced-background layers -- inlinesvg.go/advancedbg.go) as
// <foreignObject width="W" height="H" [x="X"] [data-marpit-advanced-background="DataAttr"]>...</foreignObject>.
func (r *nodeRenderer) renderForeignObject(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	fo := n.(*ForeignObject)
	if entering {
		_, _ = w.WriteString(`<foreignObject width="`)
		_, _ = w.Write(util.EscapeHTML([]byte(fo.Width)))
		_, _ = w.WriteString(`" height="`)
		_, _ = w.Write(util.EscapeHTML([]byte(fo.Height)))
		_, _ = w.WriteString(`"`)
		if fo.X != "" {
			_, _ = w.WriteString(` x="`)
			_, _ = w.Write(util.EscapeHTML([]byte(fo.X)))
			_, _ = w.WriteString(`"`)
		}
		if fo.DataAttr != "" {
			_, _ = w.WriteString(` data-marpit-advanced-background="`)
			_, _ = w.Write(util.EscapeHTML([]byte(fo.DataAttr)))
			_, _ = w.WriteString(`"`)
		}
		_, _ = w.WriteString(`>`)
	} else {
		_, _ = w.WriteString(`</foreignObject>`)
	}
	return ast.WalkContinue, nil
}

// renderBackgroundLayer renders a *BackgroundLayer (the advanced-background
// structure's first layer, advancedbg.go) as a self-contained, one-shot
// (entering-only) emission:
//
//	<section ...SectionAttrs>
//	  <div data-marpit-advanced-background-container="true" data-marpit-advanced-background-direction="...">
//	    <figure style="...">[<figcaption>ESCAPED</figcaption>]</figure>*
//	  </div>
//	</section>
func (r *nodeRenderer) renderBackgroundLayer(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	bg := n.(*BackgroundLayer)

	_, _ = w.WriteString(`<section`)
	writeAttrs(w, bg.SectionAttrs)
	_, _ = w.WriteString(`><div data-marpit-advanced-background-container="true" data-marpit-advanced-background-direction="`)
	_, _ = w.Write(util.EscapeHTML([]byte(bg.Direction)))
	_, _ = w.WriteString(`">`)

	for _, img := range bg.Images {
		_, _ = w.WriteString(`<figure style="`)
		_, _ = w.Write(util.EscapeHTML([]byte(figureStyle(img))))
		_, _ = w.WriteString(`">`)
		if img.Alt != "" {
			_, _ = w.WriteString(`<figcaption>`)
			_, _ = w.Write(util.EscapeHTML([]byte(img.Alt)))
			_, _ = w.WriteString(`</figcaption>`)
		}
		_, _ = w.WriteString(`</figure>`)
	}

	_, _ = w.WriteString(`</div></section>`)
	return ast.WalkContinue, nil
}

// renderPseudoLayer renders a *PseudoLayer (the advanced-background
// structure's third layer, advancedbg.go) as a self-contained, one-shot
// (entering-only) emission: <section ...SectionAttrs></section> (always
// empty -- it stands in for Marpit's section::after pseudo-element).
func (r *nodeRenderer) renderPseudoLayer(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	p := n.(*PseudoLayer)

	_, _ = w.WriteString(`<section`)
	writeAttrs(w, p.SectionAttrs)
	_, _ = w.WriteString(`></section>`)
	return ast.WalkContinue, nil
}
