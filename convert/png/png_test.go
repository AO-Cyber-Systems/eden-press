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

package png

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
	"github.com/AO-Cyber-Systems/eden-press/convert"
	"github.com/AO-Cyber-Systems/eden-press/convert/chrome"
	"github.com/AO-Cyber-Systems/eden-press/press"

	// Blank imports: populate the chase/profile registry resolveProfileSize
	// looks up out.Profile in. png.go itself imports NO profiles package --
	// that is the point of this change (it used to reach profile.Default(),
	// whose answer depended on whoever happened to register first).
	_ "github.com/AO-Cyber-Systems/eden-press/profiles/paged"
	_ "github.com/AO-Cyber-Systems/eden-press/profiles/slides"
)

// newTestSession is the Chrome-presence gate every test in this file shares:
// t.Skip cleanly (never fail) when no Chrome/Chromium is discoverable --
// matching every other Chrome-gated test in convert/chrome (session_test.go,
// load_test.go). This sandbox has no system Chrome, so these tests exercise
// only the skip path here; they run live once 05-05 provisions Chrome in CI.
func newTestSession(t *testing.T) *chrome.Session {
	t.Helper()

	if _, _, err := chrome.Discover(chrome.DiscoverOptions{}); err != nil {
		t.Skipf("no Chrome discovered, skipping live-Chrome png smoke: %v", err)
	}

	sess, err := chrome.New(convert.Options{})
	if err != nil {
		t.Skipf("could not start a Chrome session, skipping live-Chrome png smoke: %v", err)
	}
	return sess
}

// loadPlainFixture builds a fake press.Output from the hand-built plain-mode
// fixture: testdata/deck.html (a div.marpit container with 3 <section>
// slides, each carrying a unique solid background color) + testdata/deck.css
// (a minimal margin/padding reset). Model.Sections carries one entry per
// slide (schema-accurate: chase/model.Build always populates one Section per
// slide), which is all ToImages reads off Model.
func loadPlainFixture(t *testing.T) press.Output {
	t.Helper()

	htmlBytes, err := os.ReadFile("testdata/deck.html")
	if err != nil {
		t.Fatalf("reading testdata/deck.html: %v", err)
	}
	cssBytes, err := os.ReadFile("testdata/deck.css")
	if err != nil {
		t.Fatalf("reading testdata/deck.css: %v", err)
	}

	return press.Output{
		HTML: string(htmlBytes),
		CSS:  string(cssBytes),
		Model: &model.Document{
			Sections: []model.Section{{ID: 1}, {ID: 2}, {ID: 3}},
		},
		// A hand-built Output must record a Profile like a real press.Render
		// Output does -- ToImages errors rather than guessing when it is
		// absent. The fixture's 1280x720 sample points pin it to slides.
		Profile: "slides",
	}
}

// slideColors is the ordered list of each plain-fixture slide's unique solid
// background color, matching testdata/deck.html verbatim -- used by the
// document-order assertion (Test-list case 2).
var slideColors = []color.NRGBA{
	{R: 0xff, G: 0x00, B: 0x00, A: 0xff}, // slide 1: red
	{R: 0x00, G: 0xff, B: 0x00, A: 0xff}, // slide 2: green
	{R: 0x00, G: 0x00, B: 0xff, A: 0xff}, // slide 3: blue
}

// samplePoint is a coordinate inside every fixture slide's body, deliberately
// near the bottom-right corner -- away from the "Slide N" text, which
// default browser layout renders flush at the section's top-left.
const sampleX, sampleY = 1270, 710

// colorAt reads img's pixel at (x, y) as a direct 0-255 color.NRGBA (RGBA()
// itself returns premultiplied-alpha, 16-bit-per-channel values; converting
// through color.NRGBAModel gives the plain 8-bit values a hand-built fixture
// color is easiest to compare against).
func colorAt(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

// closeEnough allows a small per-channel tolerance for solid-fill color
// comparisons -- Chrome renders untouched solid CSS background-color fills
// exactly, but a tolerance guards against incidental compositor/anti-aliasing
// noise at the sampled pixel.
func closeEnough(got, want color.NRGBA, tol int) bool {
	diff := func(a, b uint8) bool {
		d := int(a) - int(b)
		if d < 0 {
			d = -d
		}
		return d <= tol
	}
	return diff(got.R, want.R) && diff(got.G, want.G) && diff(got.B, want.B)
}

// TestToImagesPlainMode covers Test-list cases 1-3: image count == slide
// count with each buffer decoding at the pinned viewport dimensions (case 1),
// document order proven via each slide's unique per-pixel background color
// (case 2), and JPEG opt-in decoding via image/jpeg (case 3). All three
// share one Session and one fixture load.
func TestToImagesPlainMode(t *testing.T) {
	sess := newTestSession(t)
	defer sess.Close()

	out := loadPlainFixture(t)

	t.Run("count, decode, dimensions (Test-list case 1)", func(t *testing.T) {
		images, err := ToImages(sess, out, Options{})
		if err != nil {
			t.Fatalf("ToImages: %v", err)
		}
		if len(images) != 3 {
			t.Fatalf("len(images) = %d, want 3 (one per slide)", len(images))
		}
		for i, buf := range images {
			img, err := png.Decode(bytes.NewReader(buf))
			if err != nil {
				t.Fatalf("slide %d: decoding PNG buffer: %v", i+1, err)
			}
			if got := img.Bounds(); !got.Eq(image.Rect(0, 0, 1280, 720)) {
				t.Fatalf("slide %d: bounds = %v, want 1280x720 (the pinned viewport)", i+1, got)
			}
		}
	})

	t.Run("document order via per-slide pixel (Test-list case 2)", func(t *testing.T) {
		images, err := ToImages(sess, out, Options{})
		if err != nil {
			t.Fatalf("ToImages: %v", err)
		}
		if len(images) != len(slideColors) {
			t.Fatalf("len(images) = %d, want %d", len(images), len(slideColors))
		}
		for i, buf := range images {
			img, err := png.Decode(bytes.NewReader(buf))
			if err != nil {
				t.Fatalf("slide %d: decoding PNG buffer: %v", i+1, err)
			}
			got := colorAt(img, sampleX, sampleY)
			want := slideColors[i]
			if !closeEnough(got, want, 4) {
				t.Fatalf("slide %d: pixel(%d,%d) = %+v, want %+v (document order broken, or wrong slide captured)", i+1, sampleX, sampleY, got, want)
			}
		}
	})

	t.Run("JPEG format (Test-list case 3)", func(t *testing.T) {
		images, err := ToImages(sess, out, Options{Format: convert.JPEG})
		if err != nil {
			t.Fatalf("ToImages: %v", err)
		}
		if len(images) != 3 {
			t.Fatalf("len(images) = %d, want 3", len(images))
		}
		for i, buf := range images {
			img, err := jpeg.Decode(bytes.NewReader(buf))
			if err != nil {
				t.Fatalf("slide %d: decoding JPEG buffer: %v", i+1, err)
			}
			if got := img.Bounds(); !got.Eq(image.Rect(0, 0, 1280, 720)) {
				t.Fatalf("slide %d: JPEG bounds = %v, want 1280x720", i+1, got)
			}
		}
	})
}

// inlineSVGDeckHTML is a HAND-BUILT 2-slide fixture matching the REAL DOM
// shape chase/markdown/inlinesvg.go's wrapBaseSvg produces in inline-SVG
// mode: each slide's <section> wrapped in its OWN
// <svg data-marpit-svg viewBox="0 0 W H"><foreignObject width height>
// sibling, directly under div.marpit -- confirmed against
// chase/markdown/background_test.go's own asserted HTML shape.
const inlineSVGDeckHTML = `<div class="marpit">` +
	`<svg data-marpit-svg="" viewBox="0 0 1280 720"><foreignObject width="1280" height="720">` +
	`<section id="1" style="display:block;width:1280px;height:720px;margin:0;padding:0;box-sizing:border-box;background-color:#ff0000;">Slide One</section>` +
	`</foreignObject></svg>` +
	`<svg data-marpit-svg="" viewBox="0 0 1280 720"><foreignObject width="1280" height="720">` +
	`<section id="2" style="display:block;width:1280px;height:720px;margin:0;padding:0;box-sizing:border-box;background-color:#00ff00;">Slide Two</section>` +
	`</foreignObject></svg>` +
	`</div>`

// inlineSVGDeckCSS sizes the <svg data-marpit-svg> element itself at the
// deck's pinned 1280x720 dimensions: without an explicit width/height, an
// inline <svg> with a viewBox but no width/height ATTRIBUTE (exactly what
// chase/markdown's renderSvg emits) falls back to the browser's default
// replaced-element size (300x150), which would scale everything inside the
// foreignObject down with it. Real Marpit CSS handles this the same way (a
// caller-supplied rule, not a browser default) -- this hand-built fixture
// supplies its own minimal equivalent rather than depending on
// profiles/slides' full scaffold pipeline (which this fixture deliberately
// bypasses, per the TRD's "fake press.Output" test shape).
const inlineSVGDeckCSS = `html, body { margin: 0; padding: 0; }
.marpit { margin: 0; padding: 0; }
svg[data-marpit-svg] { display: block; width: 1280px; height: 720px; }
`

// TestToImagesInlineSVGModeSmoke is Test-list case 4 -- the 05-RESEARCH Open
// Question #3 / Pitfall 4 smoke: proving Chrome resolves a per-slide
// chromedp.Screenshot of a <section> nested three levels deep inside
// svg>foreignObject to the CORRECT, non-empty, correctly-dimensioned,
// correctly-ordered bounding box -- not just the plain div>section case.
func TestToImagesInlineSVGModeSmoke(t *testing.T) {
	sess := newTestSession(t)
	defer sess.Close()

	out := press.Output{
		HTML: inlineSVGDeckHTML,
		CSS:  inlineSVGDeckCSS,
		Model: &model.Document{
			Sections: []model.Section{{ID: 1}, {ID: 2}},
		},
		Profile: "slides",
	}

	images, err := ToImages(sess, out, Options{InlineSVG: true})
	if err != nil {
		// error_recovery: if the foreignObject-nested selector resolves to
		// zero nodes (Open Question #3 realized in its harshest form), this
		// is where it would surface -- documented as a known inline-SVG-mode
		// screenshot caveat in 05-04-SUMMARY.md rather than silently masked.
		t.Fatalf("ToImages (InlineSVG): %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("len(images) = %d, want 2", len(images))
	}

	wantColors := []color.NRGBA{
		{R: 0xff, G: 0x00, B: 0x00, A: 0xff}, // slide 1: red
		{R: 0x00, G: 0xff, B: 0x00, A: 0xff}, // slide 2: green
	}

	for i, buf := range images {
		if len(buf) == 0 {
			t.Fatalf("slide %d: empty screenshot buffer (Open Question #3: foreignObject bounding-box resolution failed)", i+1)
		}
		img, err := png.Decode(bytes.NewReader(buf))
		if err != nil {
			t.Fatalf("slide %d: decoding PNG buffer: %v", i+1, err)
		}
		if got := img.Bounds(); got.Dx() == 0 || got.Dy() == 0 {
			t.Fatalf("slide %d: zero-dimension image %v (Open Question #3 realized)", i+1, got)
		}
		if got := img.Bounds(); !got.Eq(image.Rect(0, 0, 1280, 720)) {
			t.Fatalf("slide %d: bounds = %v, want 1280x720", i+1, got)
		}
		if got := colorAt(img, sampleX, sampleY); !closeEnough(got, wantColors[i], 4) {
			t.Fatalf("slide %d: pixel(%d,%d) = %+v, want %+v (inline-SVG document order broken)", i+1, sampleX, sampleY, got, wantColors[i])
		}
	}
}

// --- resolveProfileSize: the Chrome-free proof --------------------------

// TestResolveProfileSize proves this exporter's capture geometry follows the
// profile that actually produced the Output, mirroring convert/pdf's
// TestResolveSize. It is a pure function, so unlike every other test in this
// file it needs NO Chrome and never skips -- which matters, because the
// profile.Default() call it replaces was import-graph dependent, and an
// import-graph bug is exactly the kind that a skipped test hides.
func TestResolveProfileSize(t *testing.T) {
	out := func(profileID, sizeDirective string) press.Output {
		o := press.Output{Profile: profileID, Model: &model.Document{}}
		if sizeDirective != "" {
			o.Model.Meta = model.Meta{Directives: map[string]string{"size": sizeDirective}}
		}
		return o
	}

	cases := []struct {
		name             string
		out              press.Output
		wantW, wantH     int
		wantName         string
		wantContainer    string
		wantErrSubstring string
	}{
		{name: "slides, no directive -> 16:9", out: out("slides", ""), wantW: 1280, wantH: 720, wantName: "16:9", wantContainer: "div.marpit"},
		{name: "slides, 4:3 directive", out: out("slides", "4:3"), wantW: 960, wantH: 720, wantName: "4:3", wantContainer: "div.marpit"},
		{name: "paged, no directive -> A4", out: out("paged", ""), wantW: 794, wantH: 1123, wantName: "a4", wantContainer: "div.edenpress-paged"},
		{name: "paged, letter directive", out: out("paged", "letter"), wantW: 816, wantH: 1056, wantName: "letter", wantContainer: "div.edenpress-paged"},
		{name: "paged, a5 directive", out: out("paged", "a5"), wantW: 559, wantH: 794, wantName: "a5", wantContainer: "div.edenpress-paged"},
		{name: "paged, unknown directive -> PAGED's default, not slides'", out: out("paged", "nonsense"), wantW: 794, wantH: 1123, wantName: "a4", wantContainer: "div.edenpress-paged"},
		{name: "no recorded profile -> named error, never a silent fallback", out: out("", "16:9"), wantErrSubstring: "carries no Profile"},
		{name: "unregistered profile -> named error", out: out("epub", "16:9"), wantErrSubstring: `unknown profile "epub"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, size, err := resolveProfileSize(tc.out)

			if tc.wantErrSubstring != "" {
				if err == nil {
					t.Fatalf("resolveProfileSize(%+v) = %v/%+v, want an error containing %q", tc.out, p, size, tc.wantErrSubstring)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstring) {
					t.Errorf("error = %q, want it to contain %q", err, tc.wantErrSubstring)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveProfileSize(%+v): %v", tc.out, err)
			}
			if size.WidthPx != tc.wantW || size.HeightPx != tc.wantH {
				t.Errorf("size = %dx%d (%q), want %dx%d (%q)",
					size.WidthPx, size.HeightPx, size.Name, tc.wantW, tc.wantH, tc.wantName)
			}
			if size.Name != tc.wantName {
				t.Errorf("size name = %q, want %q", size.Name, tc.wantName)
			}
			// The selector must come from the SAME profile the geometry did,
			// or ToImages addresses a container the packed CSS never emitted.
			if got := p.Container(false); got != tc.wantContainer {
				t.Errorf("profile.Container(false) = %q, want %q", got, tc.wantContainer)
			}
		})
	}
}
