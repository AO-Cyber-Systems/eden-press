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
	"fmt"
	"image/jpeg"
	"image/png"
	"strconv"

	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/chase/profile"
	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
	"github.com/AO-Cyber-Systems/eden-press/convert"
	"github.com/AO-Cyber-Systems/eden-press/convert/chrome"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// Options configures ToImages.
type Options struct {
	// BrowserPath mirrors convert.Options.BrowserPath for surface parity
	// with the rest of the convert/ Options shapes. ToImages itself never
	// opens a Session (the caller supplies one, already built via
	// chrome.New) -- this field is presently inert inside ToImages, kept
	// only so a future convenience wrapper (or the CLI) has one consistent
	// Options shape to build a Session from before calling ToImages.
	BrowserPath string

	// Format selects PNG (the zero value, convert.PNG) or JPEG
	// (convert.JPEG). chromedp.Screenshot always captures PNG bytes
	// (chromedp hardcodes page.CaptureScreenshotFormatPng inside
	// ScreenshotNodes); when Format is convert.JPEG, ToImages decodes that
	// PNG buffer and re-encodes it via image/jpeg.
	Format convert.ImageFormat

	// InlineSVG selects the inline-<svg><foreignObject> per-slide selector
	// (Marpit's inline-SVG render mode) over the plain div.marpit>section
	// selector. It does not affect out.HTML's own shape (out already
	// carries whichever mode press.Render produced it in) -- it only tells
	// ToImages which selector shape to use when addressing each slide's
	// <section> for its Screenshot call.
	InlineSVG bool
}

// ToImages composes out into a self-contained HTML document (out.HTML +
// chrome.ComposeCSS(out.CSS)), opens a new tab on sess sized to the size-table
// entry resolved from the profile that ACTUALLY produced out (out.Profile --
// never profile.Default(), whose "first registered wins" rule made the
// captured dimensions depend on the final binary's import graph), folds in the
// 05-02 determinism recipe
// (chrome.ApplyDeterminism + chrome.LoadHTML), then loops over
// out.Model.Sections capturing each slide's <section> via ONE
// chromedp.Screenshot call each -- never chromedp.ScreenshotNodes (05-RESEARCH
// Open Question #2). It returns exactly len(out.Model.Sections) buffers, in
// document order.
func ToImages(sess *chrome.Session, out press.Output, opts Options) ([][]byte, error) {
	if out.Model == nil {
		return nil, fmt.Errorf("convert/png: ToImages: out.Model is nil (no docmodel to derive slide count/size from)")
	}

	n := len(out.Model.Sections)
	if n == 0 {
		return nil, nil
	}

	p, size, err := resolveProfileSize(out)
	if err != nil {
		return nil, err
	}

	doc := "<!doctype html><html><head><meta charset=\"utf-8\"><style>" +
		chrome.ComposeCSS(out.CSS) + "</style></head><body>" + out.HTML + "</body></html>"

	tab, cancel := sess.NewTab()
	defer cancel()

	if err := chrome.ApplyDeterminism(tab, int64(size.WidthPx), int64(size.HeightPx)); err != nil {
		return nil, fmt.Errorf("convert/png: ToImages: %w", err)
	}
	if err := chrome.LoadHTML(tab, doc); err != nil {
		return nil, fmt.Errorf("convert/png: ToImages: %w", err)
	}

	base := p.Container(false) // always "div.marpit" -- the one physical root both modes hang off of.
	unit := p.UnitElement()    // "section"

	images := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		k := i + 1 // CSS nth-of-type is 1-based; Model.Sections is 0-based.
		sel := slideSelector(base, unit, opts.InlineSVG, k)

		var buf []byte
		if err := chromedp.Run(tab, chromedp.Screenshot(sel, &buf, chromedp.ByQuery)); err != nil {
			return nil, fmt.Errorf("convert/png: ToImages: screenshotting slide %d (selector %q): %w", k, sel, err)
		}

		if opts.Format == convert.JPEG {
			jpegBuf, err := pngToJPEG(buf)
			if err != nil {
				return nil, fmt.Errorf("convert/png: ToImages: re-encoding slide %d as JPEG: %w", k, err)
			}
			buf = jpegBuf
		}

		images = append(images, buf)
	}

	return images, nil
}

// resolveProfileSize resolves BOTH the profile that ACTUALLY produced out
// (out.Profile, recorded by press.Render) and the size ToImages captures at:
// out.Model.Meta's "size" front-matter directive looked up in THAT profile's
// own size table, falling back to THAT table's Default when the directive is
// absent or names a size the profile does not define.
//
// It replaces a profile.Default() call. That function's "first registered
// wins" rule is deterministic only for a fixed registration order, and
// registration happens in package init() -- so which profile it returned
// depended on the final binary's import graph rather than on anything the
// caller decided. press.Render was fixed to avoid exactly that hazard (see
// press.go's defaultProfileName comment); this exporter had not read it.
//
// The profile is returned alongside the size because ToImages needs its
// Container()/UnitElement() for the per-slide selector too -- and those must
// come from the SAME profile the geometry did, or the selector addresses a
// container the CSS never generated.
//
// An Output with no recorded profile is an ERROR, never a silent fallback.
func resolveProfileSize(out press.Output) (profile.Profile, theme.Size, error) {
	if out.Profile == "" {
		return nil, theme.Size{}, fmt.Errorf(
			"convert/png: resolveProfileSize: Output carries no Profile; re-render with a press version that records it")
	}
	p, ok := profile.Get(out.Profile)
	if !ok {
		return nil, theme.Size{}, fmt.Errorf(
			"convert/png: resolveProfileSize: unknown profile %q (import the profiles/%s package for its init side-effect)",
			out.Profile, out.Profile)
	}

	table := p.Sizes()
	size := table.Default
	if out.Model != nil {
		if name := out.Model.Meta.Directives["size"]; name != "" {
			if sz, ok := table.ByName[name]; ok {
				size = sz
			}
		}
	}
	return p, size, nil
}

// slideSelector builds the k-th slide's chromedp.ByQuery (querySelector)
// selector under base ("div.marpit").
//
// Plain mode: base > unit:nth-of-type(k) -- e.g. "div.marpit >
// section:nth-of-type(3)".
//
// Inline-SVG mode: base > svg:nth-of-type(k) > foreignObject > unit. Every
// slide's <section> is wrapped in its OWN <svg><foreignObject> sibling
// directly under div.marpit (chase/markdown/inlinesvg.go's wrapBaseSvg), so
// the position discriminator attaches to "svg" -- NOT to "section" inside
// the compound selector, since each foreignObject has exactly one <section>
// child and nth-of-type on "section" would always resolve to position 1
// there, never selecting slide 2+ (see doc.go's package comment and 05-04's
// Open Question #3 smoke test in png_test.go).
func slideSelector(base, unit string, inlineSVG bool, k int) string {
	idx := strconv.Itoa(k)
	if inlineSVG {
		return base + " > svg:nth-of-type(" + idx + ") > foreignObject > " + unit
	}
	return base + " > " + unit + ":nth-of-type(" + idx + ")"
}

// pngToJPEG decodes a PNG buffer (chromedp.Screenshot's native output) and
// re-encodes it as JPEG at the standard library's default quality.
func pngToJPEG(pngBuf []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(pngBuf))
	if err != nil {
		return nil, fmt.Errorf("decoding screenshot PNG buffer: %w", err)
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpeg.DefaultQuality}); err != nil {
		return nil, fmt.Errorf("encoding JPEG: %w", err)
	}
	return out.Bytes(), nil
}
