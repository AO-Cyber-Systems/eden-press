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

package themes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
	"github.com/AO-Cyber-Systems/eden-press/conformance/cssdiff"
	"github.com/AO-Cyber-Systems/eden-press/profiles/slides"
	assets "github.com/AO-Cyber-Systems/eden-press/themes"
)

// slideParams pulls the profile primitives ThemeSet needs from profiles/slides
// (the same wiring chase/chase.go's packCSS uses at runtime): the unit element,
// the base scaffold CSS, the advanced-background CSS recovered as the suffix of
// Scaffold(true), and the bare-@size fallback table.
func slideParams() (unit, scaffold, adv string, sizes map[string]theme.Size) {
	p := slides.New()
	unit = p.UnitElement()
	scaffold = p.Scaffold(false)
	adv = strings.TrimPrefix(p.Scaffold(true), scaffold)
	sizes = p.Sizes().ByName
	return
}

// wantThemes is the fixed set of bundled Marp themes this TRD vendors.
var wantThemes = []string{"default", "gaia", "uncover"}

// TestEmbeddedThemesLoad proves every embedded theme's compiled CSS carries its
// /*! @theme <name> … */ block as the LEADING comment, so chase/theme.Load
// (whose meta.go reads only the FIRST comment) parses it — the load-bearing
// result of Task 1's leading-comment hoist. This is the "each theme Loads"
// half of must_haves truth 2.
func TestEmbeddedThemesLoad(t *testing.T) {
	_, _, _, sizes := slideParams()
	for i, css := range []string{assets.DefaultCSS, assets.GaiaCSS, assets.UncoverCSS} {
		name := wantThemes[i]
		if !strings.HasPrefix(css, "/*!") {
			t.Errorf("%s: embedded CSS does not lead with a /*! comment (first 20: %q)", name, css[:20])
		}
		th, err := theme.Load(css, "section", sizes)
		if err != nil {
			t.Fatalf("%s: theme.Load: %v", name, err)
		}
		if th.Name != name {
			t.Errorf("%s: theme.Load .Name = %q, want %q (@theme metadata mismatch)", name, th.Name, name)
		}
	}
}

// TestLeadingCommentGate is the negative control for the gate above: if any
// comment precedes the /*! @theme */ block, meta.go's leadingComment() captures
// the wrong block and Load MUST fail with "missing required @theme metadata".
// This proves the leading-comment placement (never addlicense-stamping the .css)
// is genuinely load-bearing, not incidental.
func TestLeadingCommentGate(t *testing.T) {
	_, _, _, sizes := slideParams()
	displaced := "/* an Eden header would go here */\n" + assets.GaiaCSS
	if _, err := theme.Load(displaced, "section", sizes); err == nil {
		t.Fatal("expected Load to FAIL when a stray comment displaces the leading @theme block, got nil error")
	}
}

// TestThemeSetRegistersAllByName proves ThemeSet returns a set with all three
// themes registered under their @theme names (plus the reserved scaffold
// identity) — the "registered under their @theme names, keyed for Pack-by-name"
// half of must_haves truth 2.
func TestThemeSetRegistersAllByName(t *testing.T) {
	unit, scaffold, adv, sizes := slideParams()
	ts, err := ThemeSet(unit, scaffold, adv, sizes)
	if err != nil {
		t.Fatalf("ThemeSet: %v", err)
	}
	for _, name := range wantThemes {
		th, ok := ts.Get(name)
		if !ok {
			t.Errorf("ThemeSet missing theme %q", name)
			continue
		}
		if th.Name != name {
			t.Errorf("theme %q registered under wrong Name %q", name, th.Name)
		}
	}
	if _, ok := ts.Get(theme.ScaffoldThemeName); !ok {
		t.Errorf("ThemeSet missing reserved scaffold identity %q", theme.ScaffoldThemeName)
	}
}

// TestEveryThemePacksNonEmpty proves each embedded theme Packs to non-empty,
// fully-scoped CSS via ThemeSet.Pack(name, …) — the container chain
// (div.marpit > svg > foreignObject > section) confirms the selector-scope pass
// ran. This is must_haves truth 3's "Every embedded theme Packs to non-empty
// scoped CSS via ThemeSet.Pack(name, …)".
func TestEveryThemePacksNonEmpty(t *testing.T) {
	unit, scaffold, adv, sizes := slideParams()
	ts, err := ThemeSet(unit, scaffold, adv, sizes)
	if err != nil {
		t.Fatalf("ThemeSet: %v", err)
	}
	for _, name := range wantThemes {
		packed, err := ts.Pack(name, theme.PackOptions{InlineSVG: true})
		if err != nil {
			t.Fatalf("Pack(%s): %v", name, err)
		}
		if strings.TrimSpace(packed) == "" {
			t.Fatalf("Pack(%s) returned empty CSS", name)
		}
		if !strings.Contains(packed, "div.marpit > svg > foreignObject > section") {
			t.Errorf("Pack(%s) output not scoped to the inline-SVG container chain", name)
		}
		// The packed CSS must itself be well-formed under the CONF-03 comparator
		// (no parse error) — it is the CSS press.Render will emit.
		if _, err := cssdiff.Parse(packed); err != nil {
			t.Errorf("Pack(%s) output is not CONF-03-parseable: %v", name, err)
		}
	}
}

// canonRuleSet parses css via the CONF-03 comparator and returns its rules as a
// set of format-insensitive selector+declarations keys (hex casing, whitespace,
// comment, and quote normalization applied by cssdiff.Parse). Bare at-rule
// statements (empty selector) are skipped.
func canonRuleSet(t *testing.T, css string) map[string]bool {
	t.Helper()
	sh, err := cssdiff.Parse(css)
	if err != nil {
		t.Fatalf("cssdiff.Parse: %v", err)
	}
	set := map[string]bool{}
	for _, r := range sh.Rules {
		if r.Selector == "" {
			continue
		}
		var b strings.Builder
		b.WriteString(r.Selector)
		b.WriteByte('|')
		for _, d := range r.Declarations {
			b.WriteString(d.Property)
			b.WriteByte(':')
			b.WriteString(d.Value)
			if d.Important {
				b.WriteByte('!')
			}
			b.WriteByte(';')
		}
		set[b.String()] = true
	}
	return set
}

// TestCorpusSharedRuleGate is the Objective-0 CSS-AST diff gate SHAPE
// (must_haves truth 3): where a marp-core-rendered corpus case exists
// (marp-theme-gaia, marp-theme-uncover), our packed theme output and marp-core's
// expected.css must SHARE the theme's own rules under the format-insensitive
// cssdiff model.
//
// NOT full-document cssdiff.Equal: expected.css is marp-core's FULL render
// output, which layers render-time passes press/themes does NOT apply at this
// objective (marp-h1 custom-element expansion of heading selectors, emoji-CSS
// injection, highlight-base injection, and a fuller scaffold) — those belong to
// press.Render / later batteries, not this verbatim-theme-bundling TRD. The
// theme's OWN palette is what must be byte-parity, and the .hljs-* highlight
// rules (the ground truth 03-05's chroma→hljs remap is derived from) are the
// sharpest signal: they carry no heading/emoji render-time rewrite, so they must
// appear near-identically in both.
func TestCorpusSharedRuleGate(t *testing.T) {
	unit, scaffold, adv, sizes := slideParams()
	ts, err := ThemeSet(unit, scaffold, adv, sizes)
	if err != nil {
		t.Fatalf("ThemeSet: %v", err)
	}
	for _, name := range []string{"gaia", "uncover"} {
		packed, err := ts.Pack(name, theme.PackOptions{InlineSVG: true})
		if err != nil {
			t.Fatalf("Pack(%s): %v", name, err)
		}
		expPath := filepath.Join("..", "..", "conformance", "corpus", "cases", "marp-theme-"+name, "expected.css")
		expBytes, err := os.ReadFile(expPath)
		if err != nil {
			t.Fatalf("%s: read corpus expected.css: %v", name, err)
		}
		pkgSet := canonRuleSet(t, packed)
		expSet := canonRuleSet(t, string(expBytes))

		var shared, total, hljsShared, hljsTotal int
		for k := range pkgSet {
			total++
			if expSet[k] {
				shared++
			}
			if strings.Contains(k, ".hljs") {
				hljsTotal++
				if expSet[k] {
					hljsShared++
				}
			}
		}
		t.Logf("%s: packedRules=%d sharedWithCorpus=%d | hljsPacked=%d hljsShared=%d", name, total, shared, hljsTotal, hljsShared)

		// Overall theme-layer overlap: a majority of our packed rules are
		// byte-identical (format-insensitive) to marp-core's render.
		if shared*2 < total {
			t.Errorf("%s: only %d/%d packed rules shared with marp-core expected.css (< 50%%)", name, shared, total)
		}
		// Highlight palette must be essentially byte-parity (>= 90% of packed
		// .hljs rules present in marp-core's output) — 03-05's ground truth.
		if hljsTotal == 0 {
			t.Errorf("%s: no .hljs rules in packed output (theme highlight palette missing)", name)
		} else if hljsShared*10 < hljsTotal*9 {
			t.Errorf("%s: only %d/%d packed .hljs rules match marp-core (< 90%%)", name, hljsShared, hljsTotal)
		}
	}
}

// TestNames pins the bundled name set + order (default first, the fallback).
func TestNames(t *testing.T) {
	got := Names()
	if strings.Join(got, ",") != strings.Join(wantThemes, ",") {
		t.Errorf("Names() = %v, want %v", got, wantThemes)
	}
}
