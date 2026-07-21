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

package pdf

import (
	"bytes"
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
	"github.com/AO-Cyber-Systems/eden-press/convert"
	"github.com/AO-Cyber-Systems/eden-press/convert/chrome"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// --- shared fixtures & helpers ------------------------------------------

const (
	deckHTMLPath = "testdata/deck.html"
	deckCSSPath  = "testdata/deck.css"

	bgRuleStart = "/* bg-rule-start */"
	bgRuleEnd   = "/* bg-rule-end */"
)

// newTestSession is the shared Chrome-presence gate every test in this file
// opens with: t.Skip cleanly (never fail) when no Chrome/Chromium is
// discoverable, matching convert/chrome's own live-Chrome test pattern
// (session_test.go's TestSessionMultiTab, load_test.go's
// TestLoadHTMLReadsBackRealContent) -- this sandbox has none.
func newTestSession(t *testing.T) *chrome.Session {
	t.Helper()
	if _, _, err := chrome.Discover(chrome.DiscoverOptions{}); err != nil {
		t.Skipf("no Chrome discovered, skipping live-Chrome pdf test: %v", err)
	}
	sess, err := chrome.New(convert.Options{})
	if err != nil {
		t.Skipf("could not start a Chrome session, skipping live-Chrome pdf test: %v", err)
	}
	return sess
}

// readFixture reads a testdata file, FAILING the test (not skipping) on
// error -- a missing/unreadable fixture is a test-authoring bug, not a
// live-Chrome environment gap.
func readFixture(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", path, err)
	}
	return string(b)
}

// fixtureOutput builds a hand-built press.Output from deck.html/deck.css,
// decoupled from press.Render (05-RESEARCH: "testable against hand-written
// static HTML fixtures"), plus a 3-section model.Document so every test's
// expected page count is pinned to len(out.Model.Sections) rather than a
// bare magic number.
func fixtureOutput(t *testing.T) press.Output {
	t.Helper()
	return press.Output{
		HTML: readFixture(t, deckHTMLPath),
		CSS:  readFixture(t, deckCSSPath),
		Model: &model.Document{
			SchemaVersion: model.SchemaVersion,
			Sections: []model.Section{
				{ID: 1}, {ID: 2}, {ID: 3},
			},
		},
	}
}

// stripBGRule deterministically removes the `.bg{...}` rule bracketed by
// deck.css's "bg-rule-start"/"bg-rule-end" markers -- a byte-slice, not a
// regeneration, so the "backgrounds stripped" variant stays hand-built and
// never drifts from deck.css's own text.
func stripBGRule(t *testing.T, css string) string {
	t.Helper()
	start := strings.Index(css, bgRuleStart)
	end := strings.Index(css, bgRuleEnd)
	if start == -1 || end == -1 {
		t.Fatalf("deck.css fixture: bg-rule markers not found (start=%d end=%d)", start, end)
	}
	return css[:start] + css[end+len(bgRuleEnd):]
}

// pageTypeRE matches a PDF `/Type /Page` LEAF object -- not `/Type /Pages`,
// the page-TREE root every PDF has exactly one of. The trailing \b asserts
// a non-word byte (or end of input) immediately after "Page": a word-word
// position is never a boundary, so the pattern never matches the "s" of
// "Pages" at all. This is a byte-scan structural proxy -- sufficient for
// these small fixtures (05-RESEARCH's own recommendation), not a
// general-purpose PDF parser.
var pageTypeRE = regexp.MustCompile(`/Type\s*/Page\b`)

func countPDFPages(data []byte) int {
	return len(pageTypeRE.FindAll(data, -1))
}

// mediaBoxRE captures a PDF `/MediaBox [x0 y0 x1 y1]` array's four numbers
// -- used to compare page dimensions STRUCTURALLY across two runs (Test
// list case 2), never against a fixed expected value here (that
// confirmation belongs to convert/chrome's own PageCSSInches unit tests).
var mediaBoxRE = regexp.MustCompile(`/MediaBox\s*\[\s*([\d.\-]+)\s+([\d.\-]+)\s+([\d.\-]+)\s+([\d.\-]+)\s*\]`)

func mediaBoxes(data []byte) []string {
	matches := mediaBoxRE.FindAllStringSubmatch(string(data), -1)
	boxes := make([]string, 0, len(matches))
	for _, m := range matches {
		boxes = append(boxes, strings.Join(m[1:], " "))
	}
	return boxes
}

// chromeProduct t.Logs the live Chrome/Chromium product string (e.g.
// "HeadlessChrome/120.0.6099.0") via CDP's Browser.getVersion -- feeds
// 05-05's version-pin + PDF-revalidation process (this TRD's error_recovery:
// "Record the Chrome version tested"). Returns "" (logging why) if the CDP
// call itself fails -- never fails the test on its own account.
func chromeProduct(t *testing.T, sess *chrome.Session) string {
	t.Helper()
	tab, cancel := sess.NewTab()
	defer cancel()

	var product string
	err := chromedp.Run(tab, chromedp.ActionFunc(func(ctx context.Context) error {
		_, p, _, _, _, versionErr := browser.GetVersion().Do(ctx)
		product = p
		return versionErr
	}))
	if err != nil {
		t.Logf("could not read Chrome version via Browser.getVersion: %v", err)
		return ""
	}
	return product
}

// --- Test list case 1 -----------------------------------------------------

// TestToPDFStructuralPageCount is Test-list case 1: ToPDF on the hand-built
// multi-slide fixture returns bytes starting with the PDF magic header and
// containing exactly one `/Type /Page` leaf object per fixture slide.
func TestToPDFStructuralPageCount(t *testing.T) {
	sess := newTestSession(t)
	defer sess.Close()

	out := fixtureOutput(t)
	data, err := ToPDF(sess, out, Options{})
	if err != nil {
		t.Fatalf("ToPDF: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("ToPDF output does not start with the %%PDF- magic header: first bytes %q", data[:min(16, len(data))])
	}

	want := len(out.Model.Sections)
	if got := countPDFPages(data); got != want {
		t.Fatalf("countPDFPages = %d, want %d (fixture slide count)", got, want)
	}
}

// --- Test list case 2 -----------------------------------------------------

// TestToPDFDeterministicStructure is Test-list case 2: two ToPDF runs of the
// identical fixture must produce PDFs with identical page count AND
// identical page dimensions (structural equality). The real acceptance bar
// this stands in for is pixel-diff-under-threshold in a full image pipeline
// (05-RESEARCH Pitfall C -- Chrome has acknowledged PRNG-based rendering
// non-determinism) -- this test deliberately does NOT compare the two PDFs'
// raw bytes for equality.
func TestToPDFDeterministicStructure(t *testing.T) {
	sess := newTestSession(t)
	defer sess.Close()

	out := fixtureOutput(t)

	data1, err := ToPDF(sess, out, Options{})
	if err != nil {
		t.Fatalf("ToPDF (run 1): %v", err)
	}
	data2, err := ToPDF(sess, out, Options{})
	if err != nil {
		t.Fatalf("ToPDF (run 2): %v", err)
	}

	if p1, p2 := countPDFPages(data1), countPDFPages(data2); p1 != p2 {
		t.Fatalf("page count differs across runs: run1=%d run2=%d", p1, p2)
	}

	boxes1, boxes2 := mediaBoxes(data1), mediaBoxes(data2)
	if len(boxes1) == 0 {
		t.Fatal("no /MediaBox entries found in run 1's PDF -- cannot assert dimension equality")
	}
	if len(boxes1) != len(boxes2) {
		t.Fatalf("MediaBox entry count differs across runs: run1=%d run2=%d", len(boxes1), len(boxes2))
	}
	for i := range boxes1 {
		if boxes1[i] != boxes2[i] {
			t.Fatalf("MediaBox[%d] differs across runs: run1=%q run2=%q", i, boxes1[i], boxes2[i])
		}
	}
}

// --- Test list case 3 -----------------------------------------------------

// TestToPDFPrintsBackgrounds is Test-list case 3: the fixture's solid slide
// background-color (deck.css's `.bg` rule) must survive into the printed
// PDF. ToPDF always sets WithPrintBackground(true), but CDP's own default is
// PrintBackground=false ("Print background graphics. Defaults to false."),
// so a regression that dropped that call would silently blank every theme
// background. Proven here by comparing the fixture's PDF byte size against
// the SAME fixture with the `.bg` rule removed -- with the background
// present, the PDF must come out materially larger.
func TestToPDFPrintsBackgrounds(t *testing.T) {
	sess := newTestSession(t)
	defer sess.Close()

	withBG := fixtureOutput(t)

	noBG := withBG
	noBG.CSS = stripBGRule(t, withBG.CSS)

	dataWithBG, err := ToPDF(sess, withBG, Options{})
	if err != nil {
		t.Fatalf("ToPDF (with background): %v", err)
	}
	dataNoBG, err := ToPDF(sess, noBG, Options{})
	if err != nil {
		t.Fatalf("ToPDF (backgrounds stripped): %v", err)
	}

	if len(dataWithBG) <= len(dataNoBG) {
		t.Fatalf("expected the with-background PDF to be larger than the backgrounds-stripped one (WithPrintBackground working): withBG=%d bytes, noBG=%d bytes", len(dataWithBG), len(dataNoBG))
	}
}

// --- Test list case 4 -----------------------------------------------------

// inlineSVGFixtureHTML is a hand-built (no_llm_test_data) Pitfall-4/A
// regression fixture: each of deck.html's three slides re-expressed inside
// Marpit's own inline-SVG container shape (chase/profile's Container(true):
// "div.marpit > svg > foreignObject") -- one <svg> per slide, each wrapping
// exactly one <foreignObject> with an explicit width/height and one
// namespaced <section>. Both the foreignObject's explicit width/height AND
// the inner xmlns are REQUIRED: their absence is the documented
// silent-non-render trap -- a foreignObject that renders empty in the PDF
// path while an HTML string comparison would still pass.
const inlineSVGFixtureHTML = `<svg viewBox="0 0 1280 720"><foreignObject width="1280" height="720"><section xmlns="http://www.w3.org/1999/xhtml">
<h1>Welcome</h1>
<p>A hand-built multi-slide fixture for convert/pdf's structural tests.</p>
</section></foreignObject></svg>
<svg viewBox="0 0 1280 720"><foreignObject width="1280" height="720"><section xmlns="http://www.w3.org/1999/xhtml" class="bg">
<h1>Background Slide</h1>
<p>This slide carries a solid background-color via the .bg class in deck.css.</p>
</section></foreignObject></svg>
<svg viewBox="0 0 1280 720"><foreignObject width="1280" height="720"><section xmlns="http://www.w3.org/1999/xhtml">
<h1>Closing</h1>
<p>Three slides total -- the structural page-count baseline every ToPDF test in this package pins against.</p>
</section></foreignObject></svg>
`

// TestToPDFInlineSVGFixture is Test-list case 4 -- the PDF-path-only
// regression smoke for Pitfall A/4 (Chrome >=108 SVG-in-PDF issues,
// foreignObject fragility): rasterizing inlineSVGFixtureHTML must still
// produce a PDF with one leaf page per slide and a non-trivial byte size
// (not a blank/cropped page). The Chrome product string is t.Logged to feed
// 05-05's version-pin + PDF-revalidation process.
func TestToPDFInlineSVGFixture(t *testing.T) {
	sess := newTestSession(t)
	defer sess.Close()

	if product := chromeProduct(t, sess); product != "" {
		t.Logf("inline-SVG PDF smoke tested against Chrome product: %s", product)
	}

	base := fixtureOutput(t)
	out := press.Output{
		HTML:  inlineSVGFixtureHTML,
		CSS:   base.CSS,
		Model: base.Model,
	}

	data, err := ToPDF(sess, out, Options{})
	if err != nil {
		t.Fatalf("ToPDF (inline-SVG fixture): %v", err)
	}

	want := len(out.Model.Sections)
	if got := countPDFPages(data); got != want {
		t.Fatalf("countPDFPages = %d, want %d (fixture slide count) -- a foreignObject may have rendered empty/collapsed (Pitfall 4/A)", got, want)
	}

	const nonTrivialSize = 1024
	if len(data) < nonTrivialSize {
		t.Fatalf("inline-SVG PDF is only %d bytes -- suspiciously small, likely a blank page (Pitfall 4/A foreignObject silent-non-render trap)", len(data))
	}
}
