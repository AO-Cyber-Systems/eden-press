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

// Package themes builds a name-keyed *chase/theme.ThemeSet from the three
// official Marp themes bundled verbatim via go:embed (CORE-01). It is the
// consumer press.Render (03-09) will use to resolve opts.Theme -> front-matter
// -> "default" against a fully-populated set.
//
// The embed strings themselves live in the repo-root `themes` package (go:embed
// must be co-located with the asset files — patterns cannot contain ".."); this
// package imports them (aliased `assets`) and wires each through the exact
// intake path chase/chase.go's packCSS uses at runtime:
//
//	ts := theme.NewThemeSet(unit, scaffoldCSS, advancedBackgroundCSS)
//	th, _ := theme.Load(css, unit, sizeFallback) // requires a leading @theme comment
//	ts.Add(th)                                    // registered under th.Name (=@theme name)
//
// It stays profile-AGNOSTIC on purpose: the unit element, scaffold /
// advanced-background CSS, and the bare-@size fallback table are all
// caller-supplied (from the active chase/profile.Profile — profiles/slides
// today), so press/themes never imports a profile and never forms an import
// cycle with press.
package themes

import (
	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
	assets "github.com/AO-Cyber-Systems/eden-press/themes"
)

// embedded is the fixed set of verbatim-bundled Marp themes, in a stable
// registration order (default first, so it is the natural fallback). Each entry
// pairs the @theme name (for diagnostics / the errorf below) with its compiled
// CSS text; theme.Load derives the authoritative Name from the CSS's own
// @theme metadata when it is Add'd.
var embedded = []struct {
	name string
	css  string
}{
	{"default", assets.DefaultCSS},
	{"gaia", assets.GaiaCSS},
	{"uncover", assets.UncoverCSS},
}

// ThemeSet builds a *theme.ThemeSet with the three embedded Marp themes
// registered under their @theme names (default/gaia/uncover), plus the
// caller-supplied scaffold / advanced-background / unit-element (from the active
// profile — mirror of chase/chase.go's packCSS). Every consumer (press.Render)
// resolves opts.Theme against the returned set, then calls set.Pack(name, …).
//
// unit is the unit-element ident theme rules are scoped onto ("section" for
// slides); scaffoldCSS is prepended before every packed theme's rules;
// advancedBackgroundCSS is spliced in on inline-SVG Pack calls (pass "" to omit);
// sizeFallback resolves a bare `@size <name>` metadata line (the profile's size
// table's ByName map). It returns an error if any embedded theme fails to Load
// (which would mean a bundled CSS lost its leading @theme comment — a Task-1
// regression), never a partially-populated set.
func ThemeSet(unit, scaffoldCSS, advancedBackgroundCSS string, sizeFallback map[string]theme.Size) (*theme.ThemeSet, error) {
	ts := theme.NewThemeSet(unit, scaffoldCSS, advancedBackgroundCSS)
	// Add ALL three before any caller Pack so cross-theme @import-theme
	// references resolve (theme.ThemeSet.Pack's resolveImportTheme walks the
	// fully-populated set) — see pack.go.
	for _, e := range embedded {
		th, err := theme.Load(e.css, unit, sizeFallback)
		if err != nil {
			return nil, err
		}
		ts.Add(th)
	}
	return ts, nil
}

// Names returns the @theme names of the bundled themes in registration order
// (default, gaia, uncover) — the set of values a caller's opts.Theme /
// front-matter `theme:` may select. "default" is first, the intended fallback.
func Names() []string {
	out := make([]string, len(embedded))
	for i, e := range embedded {
		out[i] = e.name
	}
	return out
}
