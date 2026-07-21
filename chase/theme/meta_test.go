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

import "testing"

// Task 3: Metadata parse — @theme/@size/@auto-scaling + size table (THEME-02).
//
// Covers Test-list cases 5-8, plus an end-to-end ParseTheme integration
// test against the RESEARCH stress theme fixture (stressThemeCSS, defined
// in stylesheet_test.go).

// TestParseMetaThemeName covers Test-list case 5: `@theme stress` present.
func TestParseMetaThemeName(t *testing.T) {
	css := "/**\n * @theme stress\n */\nsection { color: red; }"
	m, err := ParseMeta(css)
	if err != nil {
		t.Fatalf("ParseMeta returned error: %v", err)
	}
	if m.Name != "stress" {
		t.Fatalf("Name = %q, want %q", m.Name, "stress")
	}
}

// TestParseMetaMissingThemeErrors covers Test-list case 6: a leading
// metadata comment with no @theme line is an ERROR, never silently
// defaulted (THEME-02's acceptance point).
func TestParseMetaMissingThemeErrors(t *testing.T) {
	css := "/**\n * @size 16:9 1280px 720px\n */\nsection {}"
	if _, err := ParseMeta(css); err == nil {
		t.Fatal("ParseMeta with no @theme = nil error, want an error")
	}
}

// TestParseMetaNoCommentAtAllErrors covers the same acceptance point for a
// CSS string with no leading comment whatsoever.
func TestParseMetaNoCommentAtAllErrors(t *testing.T) {
	if _, err := ParseMeta(`section { color: red; }`); err == nil {
		t.Fatal("ParseMeta with no leading comment = nil error, want an error")
	}
}

// TestParseMetaSizeTable covers Test-list case 7: two repeated @size lines
// build a named size table, and ResolveSize's 1280x720 default still holds
// for unmatched/empty names.
func TestParseMetaSizeTable(t *testing.T) {
	css := "/**\n" +
		" * @theme stress\n" +
		" * @size 16:9 1280px 720px\n" +
		" * @size 4:3 960px 720px\n" +
		" */\n"
	m, err := ParseMeta(css)
	if err != nil {
		t.Fatalf("ParseMeta returned error: %v", err)
	}
	if len(m.Sizes) != 2 {
		t.Fatalf("len(Sizes) = %d, want 2", len(m.Sizes))
	}
	if got := m.Sizes["16:9"]; got != (Size{Name: "16:9", WidthPx: 1280, HeightPx: 720}) {
		t.Fatalf("Sizes[16:9] = %+v, want {16:9 1280 720}", got)
	}
	if got := m.Sizes["4:3"]; got != (Size{Name: "4:3", WidthPx: 960, HeightPx: 720}) {
		t.Fatalf("Sizes[4:3] = %+v, want {4:3 960 720}", got)
	}
	if w, h := m.ResolveSize(""); w != 1280 || h != 720 {
		t.Fatalf(`ResolveSize("") = (%d,%d), want (1280,720)`, w, h)
	}
	if w, h := m.ResolveSize("4:3"); w != 960 || h != 720 {
		t.Fatalf(`ResolveSize("4:3") = (%d,%d), want (960,720)`, w, h)
	}
	if w, h := m.ResolveSize("missing"); w != 1280 || h != 720 {
		t.Fatalf(`ResolveSize("missing") = (%d,%d), want default (1280,720)`, w, h)
	}
}

// TestParseMetaAutoScaling covers Test-list case 8: `@auto-scaling true`.
func TestParseMetaAutoScaling(t *testing.T) {
	css := "/**\n * @theme stress\n * @auto-scaling true\n */\n"
	m, err := ParseMeta(css)
	if err != nil {
		t.Fatalf("ParseMeta returned error: %v", err)
	}
	if m.AutoScaling != "true" {
		t.Fatalf("AutoScaling = %q, want %q", m.AutoScaling, "true")
	}
}

// TestParseMetaStressThemeBareSize exercises the stress theme's bare
// `@size 4:3` form (no explicit pixel dimensions), which must still resolve
// via the well-known-keyword fallback table (see the TRD's Task 3 recovery
// note on tolerating the comment styles the corpus themes use).
func TestParseMetaStressThemeBareSize(t *testing.T) {
	m, err := ParseMeta(stressThemeCSS)
	if err != nil {
		t.Fatalf("ParseMeta returned error: %v", err)
	}
	if m.Name != "stress" {
		t.Fatalf("Name = %q, want %q", m.Name, "stress")
	}
	if got := m.Sizes["4:3"]; got != (Size{Name: "4:3", WidthPx: 960, HeightPx: 720}) {
		t.Fatalf("Sizes[4:3] = %+v, want {4:3 960 720}", got)
	}
}

// TestParseThemeIntegration is an end-to-end check that ParseTheme composes
// Parse (structural Rules/Atoms) and ParseMeta (identity Meta) into one
// fully-populated Stylesheet.
func TestParseThemeIntegration(t *testing.T) {
	sheet, err := ParseTheme(stressThemeCSS)
	if err != nil {
		t.Fatalf("ParseTheme returned error: %v", err)
	}
	if sheet.Meta.Name != "stress" {
		t.Fatalf("Meta.Name = %q, want %q", sheet.Meta.Name, "stress")
	}
	if len(sheet.Rules) != 5 {
		t.Fatalf("len(Rules) = %d, want 5", len(sheet.Rules))
	}
}

// TestParseThemeMissingMetaErrors confirms ParseTheme surfaces ParseMeta's
// required-@theme error even when the structural Parse itself would
// otherwise succeed cleanly (plain, theme-less CSS).
func TestParseThemeMissingMetaErrors(t *testing.T) {
	if _, err := ParseTheme(`section { color: red; }`); err == nil {
		t.Fatal("ParseTheme with no @theme = nil error, want an error")
	}
}
