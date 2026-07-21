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
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Test-list case 1: bg -> background flag; bg fit -> size=contain (fit
// alias); bg left:40% -> split left @40%; bg vertical -> direction
// vertical; bg 50% -> size 50%.
func TestBgOptionParseKeywords(t *testing.T) {
	t.Run("bg alone sets Background", func(t *testing.T) {
		opts := ParseBgOptions("bg")
		if !opts.Background {
			t.Fatalf("Background = false, want true")
		}
	})

	t.Run("fit aliases contain", func(t *testing.T) {
		opts := ParseBgOptions("bg fit")
		if !opts.Background {
			t.Fatalf("Background = false, want true")
		}
		if opts.SizeKeyword != "contain" {
			t.Fatalf("SizeKeyword = %q, want %q (fit is an alias for contain)", opts.SizeKeyword, "contain")
		}
	})

	t.Run("split left with percentage", func(t *testing.T) {
		opts := ParseBgOptions("bg left:40%")
		if opts.SplitSide != "left" {
			t.Fatalf("SplitSide = %q, want %q", opts.SplitSide, "left")
		}
		if opts.SplitSize != "40%" {
			t.Fatalf("SplitSize = %q, want %q", opts.SplitSize, "40%")
		}
	})

	t.Run("vertical direction", func(t *testing.T) {
		opts := ParseBgOptions("bg vertical")
		if opts.Direction != "vertical" {
			t.Fatalf("Direction = %q, want %q", opts.Direction, "vertical")
		}
	})

	t.Run("bare percentage resize", func(t *testing.T) {
		opts := ParseBgOptions("bg 50%")
		if opts.ResizeSize != "50%" {
			t.Fatalf("ResizeSize = %q, want %q", opts.ResizeSize, "50%")
		}
	})
}

// Test-list case 2: bg w:300px h:50% -> width/height with units; bare
// number -> px.
func TestBgOptionParseDimensions(t *testing.T) {
	opts := ParseBgOptions("bg w:300px h:50%")
	if opts.Width != "300px" {
		t.Fatalf("Width = %q, want %q", opts.Width, "300px")
	}
	if opts.Height != "50%" {
		t.Fatalf("Height = %q, want %q", opts.Height, "50%")
	}

	bare := ParseBgOptions("bg w:300 h:200")
	if bare.Width != "300px" {
		t.Fatalf("bare Width = %q, want %q (bare number -> px)", bare.Width, "300px")
	}
	if bare.Height != "200px" {
		t.Fatalf("bare Height = %q, want %q (bare number -> px)", bare.Height, "200px")
	}
}

// Test-list case 3: bg blur -> filter blur(10px) default; bg drop-shadow ->
// default '0 5px 10px rgba(0,0,0,.4)'.
func TestBgOptionParseFilters(t *testing.T) {
	blur := ParseBgOptions("bg blur")
	if got := blur.FilterCSS(); got != "blur(10px)" {
		t.Fatalf("FilterCSS() = %q, want %q", got, "blur(10px)")
	}

	shadow := ParseBgOptions("bg drop-shadow")
	if got := shadow.FilterCSS(); got != "drop-shadow(0 5px 10px rgba(0,0,0,.4))" {
		t.Fatalf("FilterCSS() = %q, want %q", got, "drop-shadow(0 5px 10px rgba(0,0,0,.4))")
	}
}

// Test-list case 4 (Task 2): with inline-SVG mode explicitly enabled (no
// background images on the slide), svgTransformer wraps the untouched
// *Section in <svg data-marpit-svg viewBox="0 0 1280 720">
// <foreignObject width="1280" height="720">...</foreignObject></svg>.
func TestInlineSvgWrapsPlainSlide(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	pc := parser.NewContext()
	pc.Set(SvgOptionsKey, &SvgOptions{Enabled: true})
	source := []byte("# A\n")

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader, parser.WithContext(pc))

	svg := doc.FirstChild()
	if svg == nil || svg.Kind() != KindSvg {
		t.Fatalf("expected doc's first child to be *Svg, got %T", svg)
	}
	fo := svg.FirstChild()
	if fo == nil || fo.Kind() != KindForeignObject {
		t.Fatalf("expected *Svg's first child to be *ForeignObject, got %T", fo)
	}
	sec := fo.FirstChild()
	if sec == nil || sec.Kind() != KindSection {
		t.Fatalf("expected *ForeignObject's first child to be the original *Section, got %T", sec)
	}

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		t.Fatalf("Render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, `<svg data-marpit-svg="" viewBox="0 0 1280 720">`) {
		t.Fatalf("missing svg viewBox wrap, got: %s", html)
	}
	if !strings.Contains(html, `<foreignObject width="1280" height="720">`) {
		t.Fatalf("missing foreignObject wrap, got: %s", html)
	}
	if !strings.Contains(html, `<section id="1">`) {
		t.Fatalf("expected section id=1 preserved inside foreignObject, got: %s", html)
	}
}

// Test-list case 5 (Task 2): the viewBox/foreignObject dimensions follow
// SvgOptionsKey's overridden Width/Height (e.g. a 4:3 theme resolving to
// 960x720), NOT the 1280x720 default -- and chase/markdown never imports
// chase/theme to get there (the value always arrives via the parser.Context
// key, per RESEARCH's zero-import boundary).
func TestInlineSvgViewBoxOverride(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	pc := parser.NewContext()
	pc.Set(SvgOptionsKey, &SvgOptions{Enabled: true, Width: 960, Height: 720})
	source := []byte("# A\n")

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader, parser.WithContext(pc))

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		t.Fatalf("Render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, `viewBox="0 0 960 720"`) {
		t.Fatalf("expected overridden viewBox 960x720, got: %s", html)
	}
	if !strings.Contains(html, `<foreignObject width="960" height="720">`) {
		t.Fatalf("expected overridden foreignObject 960x720, got: %s", html)
	}
}

// Test-list case 6 (Task 2): with inline-SVG mode left at its default
// (disabled), a single `![bg](...)` image materializes as a backgroundImage
// local directive on the slide's *Section (reusing apply.go's
// applyDirectivesToSection directly) -- no svg/foreignObject structure is
// emitted, and the image itself is removed from the visible content tree.
func TestNonSvgBackgroundImageDirective(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	source := []byte("![bg](https://example.com/bg.jpg)\n")

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	if got := doc.FirstChild(); got == nil || got.Kind() != KindSection {
		t.Fatalf("default (SVG disabled) must leave doc's first child as *Section, got %T", got)
	}
	if img := findNode(doc, ast.KindImage); img != nil {
		t.Fatalf("expected the bg-marked image to be removed from the content tree, still found: %v", img)
	}

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		t.Fatalf("Render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, `background-image:url(&quot;https://example.com/bg.jpg&quot;)`) {
		t.Fatalf("expected background-image style with escaped url, got: %s", html)
	}
	if strings.Contains(html, "<svg") || strings.Contains(html, "<foreignObject") {
		t.Fatalf("non-SVG mode must not emit svg/foreignObject wrap, got: %s", html)
	}
}

// renderDocWithSvg runs the two-phase seam with inline-SVG mode explicitly
// enabled (SvgOptionsKey), returning the rendered HTML.
func renderDocWithSvg(md goldmark.Markdown, src string) string {
	source := []byte(src)
	pc := parser.NewContext()
	pc.Set(SvgOptionsKey, &SvgOptions{Enabled: true})

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader, parser.WithContext(pc))

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		panic(err)
	}
	return buf.String()
}

// Test-list case 7 (Task 3): marp-bg-image's advanced-background structure,
// verified structurally (ignoring the heading-id slug/whitespace concerns
// TestCorpusMarpClassSpotStructural already documents as an
// Objective-3/TRD-01-08-only concern) against the byte-exact background/
// content/pseudo layer shape confirmed against
// conformance/corpus/cases/marp-bg-image/expected.html.
func TestCorpusMarpBgImageStructural(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDocWithSvg(md, readFixture(t, "marp-bg-image", "input.md"))

	for _, want := range []string{
		`<svg data-marpit-svg="" viewBox="0 0 1280 720">`,
		`<foreignObject width="1280" height="720"><section data-marpit-advanced-background="background"><div data-marpit-advanced-background-container="true" data-marpit-advanced-background-direction="horizontal"><figure style="background-image:url(&quot;https://example.com/bg.jpg&quot;);"></figure></div></section></foreignObject>`,
		`<foreignObject width="1280" height="720"><section id="1" data-marpit-advanced-background="content">`,
		`<foreignObject width="1280" height="720" data-marpit-advanced-background="pseudo"><section data-marpit-advanced-background="pseudo" style=""></section></foreignObject>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected substring:\n%s\nnot found in:\n%s", want, out)
		}
	}
}

// Test-list case 8 (Task 3): marp-bg-split's split=left advanced-background
// structure -- content layer width/x adjust to the split percentage,
// background/pseudo layers carry the split attr + merged
// --marpit-advanced-background-split style through unchanged -- verified
// against conformance/corpus/cases/marp-bg-split/expected.html.
func TestCorpusMarpBgSplitStructural(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	out := renderDocWithSvg(md, readFixture(t, "marp-bg-split", "input.md"))

	for _, want := range []string{
		`<foreignObject width="1280" height="720"><section data-marpit-advanced-background="background" data-marpit-advanced-background-split="left" style="--marpit-advanced-background-split:50%;"><div data-marpit-advanced-background-container="true" data-marpit-advanced-background-direction="horizontal"><figure style="background-image:url(&quot;https://example.com/l.jpg&quot;);"></figure></div></section></foreignObject>`,
		`<foreignObject width="50%" height="720" x="50%"><section id="1" data-marpit-advanced-background="content" data-marpit-advanced-background-split="left" style="--marpit-advanced-background-split:50%;">`,
		`<foreignObject width="1280" height="720" data-marpit-advanced-background="pseudo"><section data-marpit-advanced-background="pseudo" data-marpit-advanced-background-split="left" style=""></section></foreignObject>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected substring:\n%s\nnot found in:\n%s", want, out)
		}
	}
}
