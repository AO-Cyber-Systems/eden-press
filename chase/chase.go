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

// Package chase is Eden Press's internal one-parse-two-sinks entrypoint
// (ARCHITECTURE.md Pattern 1/Pattern 3, adapted for this objective -- no
// press/ batteries yet, profile resolved from the registry): Render calls
// chase/markdown.Parse EXACTLY ONCE, then forks that SAME finalized
// *ast.Document to two independent sinks -- chase/markdown.RenderDoc
// (HTML) and chase/model.Build (the JSON-serializable Document) -- and
// packs CSS via chase/theme.Pack, parameterized by the active
// chase/profile.Profile (profiles/slides, the only one registered today).
//
// This is NOT the public API (MODEL-02, ARCHITECTURE.md line 87: "chase.go
// -- internal Parse+Render entrypoint, NOT the public API"). Objective 3's
// press.Render wraps this with batteries (emoji/chroma/math/sanitize) and
// go:embed themes; chase.go stays the thin internal composer.
package chase

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/parser"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
	"github.com/AO-Cyber-Systems/eden-press/chase/model"
	"github.com/AO-Cyber-Systems/eden-press/chase/profile"
	"github.com/AO-Cyber-Systems/eden-press/chase/theme"

	// Blank import: registers profiles/slides (the only Profile impl) via
	// its init() side-effect, so profile.Default() resolves even for a
	// caller that only imports this "chase" package and never imports
	// profiles/slides itself (error_recovery: "ensure profiles/slides is
	// imported for its init() side-effect").
	_ "github.com/AO-Cyber-Systems/eden-press/profiles/slides"
)

// Output is the one-parse-two-sinks result a single chase.Render call
// returns: the rendered HTML, its packed CSS, the JSON-serializable
// Model, and Meta -- a convenience alias for Model.Meta, surfaced as its
// own top-level field so a caller wanting only deck metadata need not
// reach through Model.
type Output struct {
	HTML  string
	CSS   string
	Model *model.Document
	Meta  model.Meta
}

// Render is the internal one-parse-two-sinks entrypoint (MODEL-02): ONE
// call to chase/markdown.Parse produces a single finalized *ast.Document,
// which forks to sink 1 (chase/markdown.RenderDoc, HTML) and sink 2
// (chase/model.Build, the Document) -- the SAME tree, never a second
// parse and never two separate render calls. CSS is packed via packCSS,
// parameterized by the registry's default Profile (profiles/slides
// today).
//
// CRITICAL: exactly one markdown.Parse call happens in this function.
func Render(md string) (Output, error) {
	p := profile.Default()
	if p == nil {
		return Output{}, fmt.Errorf("chase: Render: no profile registered (import a profiles/* package for its init side-effect)")
	}

	source := []byte(md)
	doc, pc := markdown.Parse(md) // ★ the ONE parse

	html, err := markdown.RenderDoc(doc, source) // sink 1: HTML, same doc
	if err != nil {
		return Output{}, err
	}

	m := model.Build(doc, source, pc) // sink 2: Model, the SAME doc

	css, err := packCSS(p, pc)
	if err != nil {
		return Output{}, err
	}

	return Output{HTML: html, CSS: css, Model: m, Meta: m.Meta}, nil
}

// packCSS bridges the active Profile's tables (unit element + scaffold/
// advanced-background CSS) into chase/theme.Pack's parameters -- the
// profile->primitives mapping that keeps chase/theme free of any
// chase/profile import (chase/profile/doc.go: chase/theme must never
// import chase/profile; the edge is one-way, Profile -> theme).
//
// inlineSVG mirrors the SAME markdown.SvgOptionsKey value
// chase/markdown.Parse set on pc (seam.go's Parse always enables it
// today), rather than hardcoding true, so packCSS stays correct if
// inline-SVG mode is ever made conditional.
//
// Only the reserved scaffold theme identity (theme.ScaffoldThemeName) is
// packed here -- this internal entrypoint selects no user-authored/
// embedded named theme (no @theme front-matter lookup, no go:embed
// themes); that battery belongs to Objective 3's press.Render, per this
// TRD's anti_patterns.
func packCSS(p profile.Profile, pc parser.Context) (string, error) {
	inlineSVG := svgEnabled(pc)

	scaffoldCSS := p.Scaffold(false)
	var advancedBackgroundCSS string
	if inlineSVG {
		// Profile.Scaffold(true) returns the base scaffold CSS with the
		// advanced-background CSS APPENDED (profiles/slides.Scaffold);
		// theme.NewThemeSet wants the two as SEPARATE strings (its third
		// param is only spliced in by Pack when PackOptions.InlineSVG is
		// set) -- recover the suffix rather than adding a second Profile
		// method with no other consumer (chase/profile/doc.go's
		// speculative-superset stance: a method needs a named call-site).
		advancedBackgroundCSS = strings.TrimPrefix(p.Scaffold(true), scaffoldCSS)
	}

	ts := theme.NewThemeSet(p.UnitElement(), scaffoldCSS, advancedBackgroundCSS)
	return ts.Pack(theme.ScaffoldThemeName, theme.PackOptions{InlineSVG: inlineSVG})
}

// svgEnabled reports whether pc carries markdown.SvgOptionsKey with
// Enabled=true -- the exact parser.Context value
// chase/markdown/inlinesvg.go's svgTransformer reads to decide whether to
// wrap Sections in Svg/ForeignObject. packCSS mirrors this so its
// container-chain / advanced-background choice always matches how doc was
// actually rendered.
func svgEnabled(pc parser.Context) bool {
	v, ok := pc.Get(markdown.SvgOptionsKey).(*markdown.SvgOptions)
	return ok && v != nil && v.Enabled
}
