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
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

// renderDoc runs the full two-phase seam (Parse then Render, NEVER
// md.Convert()) and returns the rendered HTML string. Shared helper for the
// directive-materialization tests below.
func renderDoc(md goldmark.Markdown, src string) string {
	source := []byte(src)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		panic(err)
	}
	return buf.String()
}

// Test-list case 8: InlineStyle order -- setting the same property twice
// keeps the LAST value at the FIRST-seen position; distinct properties keep
// insertion order. Mirrors helpers/inline_style.js's decls object semantics
// (a bare Go map would flake this -- 01-RESEARCH.md "Don't Hand-Roll").
func TestInlineStyleOrder(t *testing.T) {
	s := NewInlineStyle()
	s.Set("color", "red")
	s.Set("background-color", "blue")
	s.Set("color", "green") // overwrite: keeps position 0, updates value

	got := s.String()
	want := "color:green;background-color:blue;"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestInlineStyleEmpty(t *testing.T) {
	s := NewInlineStyle()
	if !s.Empty() {
		t.Fatalf("Empty() = false, want true for a fresh InlineStyle")
	}
	if got := s.String(); got != "" {
		t.Fatalf("String() = %q, want empty string", got)
	}
	s.Set("color", "red")
	if s.Empty() {
		t.Fatalf("Empty() = true after Set, want false")
	}
}

// Test-list case 1: `<!-- class: lead -->` (non-spot, local) on slide 1
// carries forward onto slide 2 as well.
func TestDirectiveClassCarriesForward(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, "<!-- class: lead -->\n\n# A\n\n---\n\n# B\n")

	// Leading space distinguishes the standalone `class="lead"` attribute
	// from the `data-class="lead"` attribute (which also contains the
	// substring `class="lead"`).
	if strings.Count(out, ` class="lead"`) != 2 {
		t.Fatalf(`expected class="lead" on BOTH slides (carry-forward), got:\n%s`, out)
	}
}

// Test-list case 2: `<!-- _class: lead -->` (spot) applies to the current
// slide ONLY; the next slide has no class at all.
func TestDirectiveSpotClassAppliesOnlyToCurrentSlide(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, "<!-- _class: lead -->\n\n# A\n\n---\n\n# B\n")

	if strings.Count(out, ` class="lead"`) != 1 {
		t.Fatalf(`expected class="lead" on slide 1 ONLY (spot), got:\n%s`, out)
	}
	// The second section must carry NO class/data-class/style at all.
	secondIdx := strings.Index(out, `<section id="2"`)
	if secondIdx < 0 {
		t.Fatalf("missing <section id=\"2\">, got:\n%s", out)
	}
	if !strings.HasPrefix(out[secondIdx:], `<section id="2">`) {
		t.Fatalf(`expected bare <section id="2"> with no directive attrs, got:\n%s`, out[secondIdx:secondIdx+80])
	}
}

// Test-list case 3: `<!-- color: red -->` -> style has BOTH the generic
// `--color:red` CSS var AND the special `color:red` CSS override, in that
// exact order (generic loop runs before the fixed-order specials block --
// ported verbatim from directives/apply.js).
func TestDirectiveColorSpecialOverride(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, "<!-- color: red -->\n\n# A\n")

	want := `style="--color:red;color:red;"`
	if !strings.Contains(out, want) {
		t.Fatalf("expected %s in output, got:\n%s", want, out)
	}
}

// Test-list case 4: backgroundColor sets background-color AND clears any
// inherited background-image (background-image:none), per apply.js's
// `.set('background-color', v).set('background-image', 'none')` chain.
func TestDirectiveBackgroundColorSpecialOverride(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, "<!-- backgroundColor: #fff -->\n\n# A\n")

	want := `style="--background-color:#fff;background-color:#fff;background-image:none;"`
	if !strings.Contains(out, want) {
		t.Fatalf("expected %s in output, got:\n%s", want, out)
	}
}

// Test-list case 5: backgroundImage sets background-image plus
// position:center / repeat:no-repeat / size:cover defaults.
func TestDirectiveBackgroundImageSpecialOverride(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, "<!-- backgroundImage: url(x) -->\n\n# A\n")

	want := `style="--background-image:url(x);background-image:url(x);background-position:center;background-repeat:no-repeat;background-size:cover;"`
	if !strings.Contains(out, want) {
		t.Fatalf("expected %s in output, got:\n%s", want, out)
	}
}

// Test-list case 6: a global directive (`theme: gaia` in front-matter) is
// stamped on EVERY slide's section identically.
func TestDirectiveGlobalThemeStampedOnEverySlide(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, "---\ntheme: gaia\n---\n\n# A\n\n---\n\n# B\n")

	if strings.Count(out, `data-theme="gaia"`) != 2 {
		t.Fatalf(`expected data-theme="gaia" stamped on BOTH slides, got:\n%s`, out)
	}
	if strings.Count(out, `--theme:gaia`) != 2 {
		t.Fatalf(`expected --theme:gaia stamped on BOTH slides, got:\n%s`, out)
	}
}
