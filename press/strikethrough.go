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

// CORE-03's one genuinely new piece: goldmark GFM's extension.Strikethrough
// renders "~~x~~" as "<del>x</del>" (extension/strikethrough.go, registered
// at renderer priority 500), but Marp Core renders GFM strikethrough as
// "<s>x</s>". strikethrough.go closes that gap with a self-contained
// goldmark.Option a caller folds into chase/markdown.NewEngine's extra-opts
// hook -- it never re-implements or touches GFM tables, hard breaks, or
// heading-ID slugs, all three of which are already baked into NewEngine
// (extension.GFM, ghtml.WithHardWraps, parser.WithAutoHeadingID) and are
// verified, not re-wired, by gfm_verify_test.go.
package press

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// sRenderer is a renderer.NodeRenderer that renders extast.KindStrikethrough
// nodes as "<s>"/"</s>" instead of goldmark GFM's default "<del>"/"</del>".
type sRenderer struct{}

// RegisterFuncs registers renderStrike against extast.KindStrikethrough --
// the SAME ast.NodeKind goldmark's own extension.StrikethroughHTMLRenderer
// registers (extension/strikethrough.go), so whichever registration for that
// kind is applied LAST wins (goldmark's renderer.Render sorts NodeRenderers
// ascending by priority, then iterates in REVERSE -- see strikethroughOption
// below for the priority this must be registered under to win that race).
func (r *sRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extast.KindStrikethrough, r.renderStrike)
}

// renderStrike emits "<s>" on entering a Strikethrough node and "</s>" on
// exit, mirroring goldmark's own StrikethroughHTMLRenderer.renderStrikethrough
// shape but with the "<s>"/"</s>" tag Marp Core expects instead of "<del>".
func (r *sRenderer) renderStrike(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<s>")
	} else {
		_, _ = w.WriteString("</s>")
	}
	return ast.WalkContinue, nil
}

// strikethroughPriority is registered BELOW goldmark's own
// extension.StrikethroughHTMLRenderer priority (500, source-verified against
// goldmark@v1.8.4's extension/strikethrough.go). goldmark's renderer.Render
// sorts registered NodeRenderers ascending by priority, then walks the sorted
// list in REVERSE when dispatching -- so the LOWEST-priority registration for
// a given ast.NodeKind is applied LAST and wins the "last write" for that
// kind. 100 mirrors the same convention chase/markdown's own nodeRenderer
// uses (priority 0) to beat goldmark's DefaultRenderer, and sits safely below
// 500 without needing to be the lowest possible value.
const strikethroughPriority = 100

// strikethroughOption returns a self-contained goldmark.Option that overrides
// GFM strikethrough rendering to emit "<s>...</s>" instead of goldmark's
// default "<del>...</del>", matching Marp Core. It is composable: a caller
// folds it into chase/markdown.NewEngine(extra ...goldmark.Option)'s
// extensibility hook (e.g. press.Render's own pressExtraOpts, TRD 03-09)
// alongside any other battery options, without re-registering or colliding
// with chase/markdown's own NodeRenderer (New(), which never registers
// extast.KindStrikethrough) or with any other battery's renderers.
//
// strikethroughOption is an OPTION, not a render call: it configures a
// goldmark.Markdown instance's renderer registration and never itself
// parses or renders anything.
func strikethroughOption() goldmark.Option {
	return goldmark.WithRendererOptions(
		renderer.WithNodeRenderers(util.Prioritized(&sRenderer{}, strikethroughPriority)),
	)
}
