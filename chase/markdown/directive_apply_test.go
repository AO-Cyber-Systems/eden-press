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
	"os"
	"path/filepath"
	"strconv"
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

// sectionTag returns the opening `<section id="N" ...>` tag substring for
// the given 1-based slide id, or "" if not found. Assumes no directive
// value in the fixtures under test contains a literal '>' character.
func sectionTag(html string, id int) string {
	needle := `<section id="` + strconv.Itoa(id) + `"`
	idx := strings.Index(html, needle)
	if idx < 0 {
		return ""
	}
	end := strings.Index(html[idx:], ">")
	if end < 0 {
		return ""
	}
	return html[idx : idx+end+1]
}

// Test-list case 7: paginate drives a running page-number counter that
// does NOT increment on skip/hold, becomes page 1 retroactively when first
// set, and data-marpit-pagination-total is stamped (two-pass) on every
// slide that carries a data-marpit-pagination attribute -- including
// `hold` slides (which freeze the displayed number but still count as
// "paginating"), but NOT `skip` slides (which are excluded entirely).
func TestDirectivePaginationTwoPass(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	src := "<!-- paginate: true -->\n\n# One\n\n---\n\n# Two\n\n---\n\n<!-- paginate: skip -->\n\n# Three\n\n---\n\n<!-- paginate: true -->\n\n# Four\n\n---\n\n<!-- paginate: hold -->\n\n# Five\n"
	out := renderDoc(md, src)

	cases := []struct {
		id        int
		wantPage  string // "" means no data-marpit-pagination attr expected
		wantTotal string // "" means no data-marpit-pagination-total attr expected
	}{
		{1, "1", "3"},
		{2, "2", "3"},
		{3, "", ""}, // skip: no increment, no pagination attr, no total
		{4, "3", "3"},
		{5, "3", "3"}, // hold: no increment (stays 3), but still paginates
	}

	for _, c := range cases {
		tag := sectionTag(out, c.id)
		if tag == "" {
			t.Fatalf("slide %d: missing <section id=%q> in:\n%s", c.id, strconv.Itoa(c.id), out)
		}
		if c.wantPage == "" {
			if strings.Contains(tag, "data-marpit-pagination=") {
				t.Fatalf("slide %d: expected NO data-marpit-pagination attr, got tag:\n%s", c.id, tag)
			}
		} else {
			want := `data-marpit-pagination="` + c.wantPage + `"`
			if !strings.Contains(tag, want) {
				t.Fatalf("slide %d: expected %s, got tag:\n%s", c.id, want, tag)
			}
		}
		if c.wantTotal == "" {
			if strings.Contains(tag, "data-marpit-pagination-total=") {
				t.Fatalf("slide %d: expected NO data-marpit-pagination-total attr, got tag:\n%s", c.id, tag)
			}
		} else {
			want := `data-marpit-pagination-total="` + c.wantTotal + `"`
			if !strings.Contains(tag, want) {
				t.Fatalf("slide %d: expected %s, got tag:\n%s", c.id, want, tag)
			}
		}
	}
}

// readFixture reads a golden-corpus fixture file relative to this package
// directory. Fails the test if the fixture is missing.
func readFixture(t *testing.T, caseName, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "conformance", "corpus", "cases", caseName, file))
	if err != nil {
		t.Fatalf("reading fixture %s/%s: %v", caseName, file, err)
	}
	return string(b)
}

// Test-list case 9 (structural): marp-class-spot renders the correct
// spot-class directive output (ignoring Objective-3 batteries like heading
// slugs and the svg/foreignObject scaffold, which are TRD 01-07/01-08 +
// Objective 3 concerns per this TRD's own verification wording).
func TestCorpusMarpClassSpotStructural(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, readFixture(t, "marp-class-spot", "input.md"))

	tag1 := sectionTag(out, 1)
	for _, want := range []string{`data-class="lead"`, ` class="lead"`, `style="--class:lead;"`} {
		if !strings.Contains(tag1, want) {
			t.Fatalf("slide 1: expected %s, got tag:\n%s", want, tag1)
		}
	}

	tag2 := sectionTag(out, 2)
	if tag2 != `<section id="2">` {
		t.Fatalf(`slide 2: expected bare <section id="2"> (spot does not carry forward), got: %s`, tag2)
	}
}

// Test-list case 9 (structural): marp-paginate renders the running counter
// + two-pass total correctly for a real corpus fixture.
func TestCorpusMarpPaginateStructural(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, readFixture(t, "marp-paginate", "input.md"))

	tag1 := sectionTag(out, 1)
	for _, want := range []string{`data-paginate="true"`, `data-marpit-pagination="1"`, `style="--paginate:true;"`, `data-marpit-pagination-total="2"`} {
		if !strings.Contains(tag1, want) {
			t.Fatalf("slide 1: expected %s, got tag:\n%s", want, tag1)
		}
	}

	tag2 := sectionTag(out, 2)
	for _, want := range []string{`data-paginate="true"`, `data-marpit-pagination="2"`, `style="--paginate:true;"`, `data-marpit-pagination-total="2"`} {
		if !strings.Contains(tag2, want) {
			t.Fatalf("slide 2: expected %s, got tag:\n%s", want, tag2)
		}
	}
}

// Test-list case 9 (structural): marp-header-footer renders both the
// materialized attrs/style AND the actual <header>/<footer> elements
// (first/last child of the section) for a real corpus fixture.
func TestCorpusMarpHeaderFooterStructural(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDoc(md, readFixture(t, "marp-header-footer", "input.md"))

	tag1 := sectionTag(out, 1)
	for _, want := range []string{`data-header="Eden Press"`, `data-footer="CONFIDENTIAL"`, `--header:Eden Press`, `--footer:CONFIDENTIAL`} {
		if !strings.Contains(tag1, want) {
			t.Fatalf("slide 1: expected %s, got tag:\n%s", want, tag1)
		}
	}

	if !strings.Contains(out, "<header>Eden Press</header>") {
		t.Fatalf("expected a rendered <header>Eden Press</header> element, got:\n%s", out)
	}
	if !strings.Contains(out, "<footer>CONFIDENTIAL</footer>") {
		t.Fatalf("expected a rendered <footer>CONFIDENTIAL</footer> element, got:\n%s", out)
	}
	// header must precede the slide's own heading content; footer must
	// follow it (first/last child ordering).
	headerIdx := strings.Index(out, "<header>Eden Press</header>")
	footerIdx := strings.Index(out, "<footer>CONFIDENTIAL</footer>")
	headingIdx := strings.Index(out, "Slide</h1>")
	if !(headerIdx < headingIdx && headingIdx < footerIdx) {
		t.Fatalf("expected header before heading before footer, got:\n%s", out)
	}
}
