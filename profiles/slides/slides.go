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

// Package slides implements chase/profile.Profile for Marp slide decks --
// the only Profile implementation this repo registers (TRD 02-03,
// MODEL-04): every value chase/theme used to hardcode (the unit element,
// the size table, the slide-reset scaffold CSS, and the pagination rule)
// originates HERE, threaded into chase/theme's profile-agnostic scoping
// passes as caller-supplied parameters. A future non-slide profile (a
// different unit element, a different size table entirely) would live
// alongside this one as its own package, registering under its own ID --
// chase/theme itself would need no change to support it.
package slides

import (
	"github.com/AO-Cyber-Systems/eden-press/chase/profile"
	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
)

// slides implements profile.Profile for Marp slide decks.
type slides struct{}

// New returns the slides Profile implementation.
func New() profile.Profile { return slides{} }

// init registers the slides profile at import time -- any binary that
// imports profiles/slides gets it in the profile.Get/profile.Default
// registry with no further wiring (chase/profile's Register/Get/Default
// doc comment).
func init() { profile.Register(New()) }

// ID is the profile's registry key.
func (slides) ID() string { return "slides" }

// UnitElement is the element chase/theme's scoping passes scope theme
// rules onto for Marp slide decks.
func (slides) UnitElement() string { return "section" }

// Container returns the physical container-chain selector text a slide's
// unit element is wrapped in for rendering: Marpit's inline-SVG render
// mode wraps it in an SVG foreignObject; the non-SVG mode wraps it in a
// plain div.
func (slides) Container(inlineSVG bool) string {
	if inlineSVG {
		return "div.marpit > svg > foreignObject"
	}
	return "div.marpit"
}

// ContainerClass is the class the rendered run's wrapping <div> carries.
// "marpit" is load-bearing for Marp compatibility: the conformance corpus,
// the three bundled themes' scoped selectors, and every existing consumer
// depend on this exact value.
func (slides) ContainerClass() string { return "marpit" }

// Sizes returns the slide size table: a 16:9 default, plus a 4:3
// alternative -- the two named sizes a slide deck's @size metadata can
// select between (see chase/theme's Meta.ResolveSize).
func (slides) Sizes() profile.SizeTable {
	sixteenNine := theme.Size{Name: "16:9", WidthPx: 1280, HeightPx: 720}
	return profile.SizeTable{
		ByName: map[string]theme.Size{
			"16:9": sixteenNine,
			"4:3":  {Name: "4:3", WidthPx: 960, HeightPx: 720},
		},
		Default: sixteenNine,
	}
}

// Pagination describes the render-time pagination CSS shape Marp slides
// want injected: the page-number counter attaches to the slide's own
// ::after pseudo-element, defaulting to Marpit's own pagination
// attribute.
func (slides) Pagination() profile.PaginationRule {
	return profile.PaginationRule{
		Selector:    "section::after",
		ContentExpr: "attr(data-marpit-pagination)",
	}
}

// Scaffold returns the base CSS text chase/theme prepends before a packed
// theme's own rules: the slide-reset scaffold CSS, plus the
// advanced-background support CSS appended when inlineSVG is true.
func (slides) Scaffold(inlineSVG bool) string {
	if inlineSVG {
		return ScaffoldCSS + AdvancedBackgroundCSS
	}
	return ScaffoldCSS
}
