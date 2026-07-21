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

package theme

// theme.go implements Tier-1 of 01-RESEARCH.md's two-tier design: work
// done ONCE per theme, at add-time, before it is ever packed/rendered.
//
// Tier-1's documented pass order is [nesting?, meta, root/replace,
// section_size, import/parse (record-only)]. This package realizes it as:
// meta + structural parse (theme.ParseTheme, already implemented by
// 01-03/meta.go — @theme's required-ness is enforced there), then
// RunPasses(passNesting, rootMarkPass(unit)) — nesting down-leveling
// (pass_nesting.go) followed by the add-time ":root" -> "<unit>:marpit-
// root" sentinel rewrite (pass_root.go, wrapping chase/theme/selector's
// locked MarkRoot). unit and sizeFallback are caller-supplied (TRD 02-03,
// MODEL-04's de-hardcoding move — see profiles/slides, which supplies
// both today); chase/theme has no default of its own.
//
// "section_size" has no Tier-1-specific MUTATION of its own: Meta's
// @size table is already fully resolved (read-only, via Meta.ResolveSize)
// by 01-03 — there is nothing further to transform at Load time, only at
// render time, which is out of this TRD's scope (see the TRD's
// must_haves — render-time concerns are pagination/scaffold/advanced-
// background injection, all Tier-2/pack.go). "import/parse" is
// deliberately RECORD-ONLY here too: Parse (01-03) already records
// @import/@import-theme atoms without resolving them; ACTUAL recursive
// resolution needs the full ThemeSet (to look up an @import-theme target
// by name), which a single Theme.Load call does not have — that's
// pass_import.go's Tier-2 concern (see that file's doc).
type Theme struct {
	Name  string
	Sheet Stylesheet
}

// Load runs Tier-1 over a theme's raw CSS text: ParseTheme (requires and
// records @theme/@size/@auto-scaling metadata — THEME-02, resolving a
// bare @size keyword against sizeFallback) followed by the
// nesting-down-level and add-time-root-mark (scoped to unit) passes.
func Load(cssText, unit string, sizeFallback map[string]Size) (*Theme, error) {
	sheet, err := ParseTheme(cssText, sizeFallback)
	if err != nil {
		return nil, err
	}
	if err := RunPasses(&sheet, passNesting, rootMarkPass(unit)); err != nil {
		return nil, err
	}
	return &Theme{Name: sheet.Meta.Name, Sheet: sheet}, nil
}

// loadPlain runs the same nesting-down-level + add-time-root-mark passes
// as Load, but over plain, meta-less CSS text via the structural-only
// Parse (not ParseTheme) — used by pack.go to bring a caller-supplied
// scaffold/advanced-background CSS block into the same Rule/Stylesheet
// shape as any real theme, without requiring a bogus "@theme" header on
// static, non-theme CSS that was never authored as one (see scaffold.go's
// doc).
func loadPlain(cssText, unit string) (Stylesheet, error) {
	sheet, err := Parse(cssText)
	if err != nil {
		return Stylesheet{}, err
	}
	if err := RunPasses(&sheet, passNesting, rootMarkPass(unit)); err != nil {
		return Stylesheet{}, err
	}
	return sheet, nil
}
