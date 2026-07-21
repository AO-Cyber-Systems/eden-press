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
type nodeRenderer struct{}

// NewNodeRenderer returns chase/markdown's renderer.NodeRenderer.
func NewNodeRenderer() renderer.NodeRenderer {
	return &nodeRenderer{}
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
}

// renderDocument wraps the entire rendered slide run in
// <div class="marpit">...</div>.
func (r *nodeRenderer) renderDocument(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="marpit">`)
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
