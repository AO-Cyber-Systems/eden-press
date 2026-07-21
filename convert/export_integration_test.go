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

// export_integration_test.go is 05-05's end-to-end EXP-04 capstone: it drives
// the WHOLE export path through PUBLIC surfaces only -- press.Render (the
// Objective-3 public render entrypoint) piped straight into chrome.New +
// pdf.ToPDF + png.ToImages (the Objective-5 exporters) -- proving the
// exporters compose with a REAL press-rendered deck, not just the hand-built
// static HTML fixtures convert/pdf's and convert/png's own package tests use.
//
// It is deliberately the EXTERNAL test package `convert_test`, importing only
// press, convert, convert/chrome, convert/pdf, and convert/png (plus stdlib)
// -- mirroring press/capstone_test.go's press-only-consumer discipline one
// level up the stack. Chrome-gated: t.Skip (never fail) when no Chrome is
// discoverable, exactly like every other live-Chrome test in this module
// (convert/chrome/session_test.go, convert/pdf/pdf_test.go,
// convert/png/png_test.go) -- this sandbox has none. The pinned
// no-system-Chrome CI export job (scripts/check-chrome-export.sh,
// .github/workflows/ci.yml) is where this test runs live, proving the EXP-04
// discovery fallback chain resolves Chrome in a clean container.
package convert_test

import (
	"bytes"
	"image/png"
	"regexp"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/convert"
	"github.com/AO-Cyber-Systems/eden-press/convert/chrome"
	"github.com/AO-Cyber-Systems/eden-press/convert/pdf"
	cpng "github.com/AO-Cyber-Systems/eden-press/convert/png"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// capstoneDeck is a hand-built (no_llm_test_data), three-slide deck covering
// a plain text slide, a math slide (native MathML via press/math's default
// backend -- the no-tofu asset this capstone sanity-checks), and a closing
// slide. It deliberately carries NO relative asset URLs (05-02 Open Question
// #1's documented self-containment contract) so the exporters' self-contained
// HTML composition never has to resolve a filesystem/network reference.
const capstoneDeck = "---\n" +
	"marp: true\n" +
	"---\n\n" +
	"# Slide One\n\n" +
	"A plain text slide -- the capstone's non-math control.\n\n" +
	"---\n\n" +
	"# Slide Two -- Math\n\n" +
	"Inline $a^2 + b^2 = c^2$ and block:\n\n" +
	"$$\\frac{1}{2}\\pi r^2$$\n\n" +
	"---\n\n" +
	"# Slide Three\n\n" +
	"A closing slide.\n"

// capstoneViewport is the pinned default 16:9 viewport (profiles/slides'
// Default size) both pdf.ToPDF's resolveSize and png.ToImages resolve to when
// the deck carries no `size:` front-matter directive, exactly as this
// capstone deck does.
const capstoneViewportW, capstoneViewportH = 1280, 720

// pageTypeRE matches a PDF `/Type /Page` LEAF object (never the `/Type
// /Pages` tree root) -- the same byte-scan structural proxy convert/pdf's own
// package tests use (05-RESEARCH's documented sufficiency bar for
// small-fixture structural page counting); duplicated here (rather than
// exported from convert/pdf) because this capstone deliberately stays on
// PUBLIC surfaces only.
var pageTypeRE = regexp.MustCompile(`/Type\s*/Page\b`)

func countPDFPages(data []byte) int {
	return len(pageTypeRE.FindAll(data, -1))
}

// newCapstoneSession is the shared Chrome-presence gate: t.Skip cleanly
// (never fail) when no Chrome/Chromium is discoverable, matching every other
// live-Chrome test in this module.
func newCapstoneSession(t *testing.T) *chrome.Session {
	t.Helper()
	if _, _, err := chrome.Discover(chrome.DiscoverOptions{}); err != nil {
		t.Skipf("no Chrome discovered, skipping live-Chrome export capstone: %v", err)
	}
	sess, err := chrome.New(convert.Options{})
	if err != nil {
		t.Skipf("could not start a Chrome session, skipping live-Chrome export capstone: %v", err)
	}
	return sess
}

// TestCapstoneExportEndToEnd is the 05-05 EXP-04 capstone (test list case 1):
// press.Render -> pdf.ToPDF + png.ToImages, through public surfaces only,
// asserting a valid PDF (page count == slide count) AND N per-slide PNGs
// decoding at the pinned viewport size, both derived from ONE real
// press-rendered deck -- proving the exporters compose with real Objective-3
// press.Render output, not just hand-built fixtures.
func TestCapstoneExportEndToEnd(t *testing.T) {
	out, err := press.Render(capstoneDeck, press.Options{InlineSVG: true})
	if err != nil {
		t.Fatalf("press.Render: %v", err)
	}
	if out.Model == nil || len(out.Model.Sections) != 3 {
		t.Fatalf("press.Render produced %d sections, want 3 (capstoneDeck authoring bug)", len(out.Model.Sections))
	}
	wantPages := len(out.Model.Sections)

	sess := newCapstoneSession(t)
	defer sess.Close()

	// --- PDF path ---------------------------------------------------------
	pdfBytes, err := pdf.ToPDF(sess, out, pdf.Options{})
	if err != nil {
		t.Fatalf("pdf.ToPDF: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("pdf.ToPDF output does not start with the %%PDF- magic header: first bytes %q", pdfBytes[:min(16, len(pdfBytes))])
	}
	if got := countPDFPages(pdfBytes); got != wantPages {
		t.Fatalf("countPDFPages = %d, want %d (== len(out.Model.Sections))", got, wantPages)
	}

	// Math no-tofu sanity (test list case 1, step 6): a full tofu-blank
	// render (MathML glyphs failing to resolve to STIX Two Math and
	// collapsing to empty boxes) would print materially smaller than a
	// real 3-slide deck with actual glyph content on every slide. This is
	// a light byte-size guard, not a pixel check -- the real pixel-level
	// MATH-table smoke is Objective 8's, per this TRD's must_haves.
	const nonTrivialPDFSize = 3072
	if len(pdfBytes) < nonTrivialPDFSize {
		t.Fatalf("capstone PDF is only %d bytes -- suspiciously small for a 3-slide deck with a math slide (possible MathML tofu-blank render, Pitfall 6)", len(pdfBytes))
	}

	// --- PNG path -----------------------------------------------------------
	imgs, err := cpng.ToImages(sess, out, cpng.Options{InlineSVG: true})
	if err != nil {
		t.Fatalf("png.ToImages: %v", err)
	}
	if len(imgs) != len(out.Model.Sections) {
		t.Fatalf("len(imgs) = %d, want %d (== len(out.Model.Sections))", len(imgs), len(out.Model.Sections))
	}
	for i, buf := range imgs {
		img, err := png.Decode(bytes.NewReader(buf))
		if err != nil {
			t.Fatalf("slide %d: decoding PNG buffer: %v", i+1, err)
		}
		if got := img.Bounds(); got.Dx() != capstoneViewportW || got.Dy() != capstoneViewportH {
			t.Fatalf("slide %d: bounds = %v, want %dx%d (the pinned default viewport)", i+1, got, capstoneViewportW, capstoneViewportH)
		}
	}
}

// min is a tiny local helper -- avoids depending on Go 1.21's builtin min
// across every build tag this module targets (mirrors convert/pdf/pdf_test.go's
// identical local helper).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
