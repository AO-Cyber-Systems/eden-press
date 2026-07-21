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

package press

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/press/themes"
)

// goldenDeck is the fixed input Test-list case 1 (additive/non-breaking)
// renders through Options{} -- deliberately the SAME minimal deck
// TestRenderZeroValueOptions (press_test.go) already exercises, so the golden
// captured here is anchored to a tiny, well-understood input.
const goldenDeck = "# Hi\n"

// The golden values below were captured via `press.Render(goldenDeck,
// press.Options{})` on main, IMMEDIATELY BEFORE this TRD's edits (i.e. before
// Options.ThemeCSS existed and before packThemeCSS grew the custom-theme
// registration loop). HTML is small enough to pin verbatim; CSS is the full
// packed "default" theme (tens of KB) so it is pinned by length + SHA-256
// digest instead of an inline text blob -- both are exact, byte-level
// identity checks over the ENTIRE output, just without bloating this file
// with a multi-KB literal.
const (
	goldenHTML      = "<div class=\"marpit\"><svg data-marpit-svg=\"\" viewBox=\"0 0 1280 720\"><foreignObject width=\"1280\" height=\"720\"><section id=\"1\"><h1 id=\"hi\">Hi</h1>\n</section></foreignObject></svg></div>"
	goldenCSSLen    = 47104
	goldenCSSSHA256 = "c1b666d1383e1641404a4d752e51a944057314544c196ab1d4b0f195c1a9eb74"
)

// TestThemeCSSAdditive is Test-list case 1: press.Render(goldenDeck,
// press.Options{}) -- i.e. a nil Options.ThemeCSS -- produces byte-identical
// HTML and CSS to the golden captured before this TRD's changes. Proves the
// new ThemeCSS field (and the packThemeCSS registration loop that consumes
// it) is a true no-op at the zero value: `range nil` never executes, so
// today's behavior is completely unchanged.
func TestThemeCSSAdditive(t *testing.T) {
	out, err := Render(goldenDeck, Options{})
	if err != nil {
		t.Fatalf("Render(goldenDeck, Options{}): unexpected error: %v", err)
	}

	if out.HTML != goldenHTML {
		t.Errorf("Options{} HTML diverged from pre-TRD golden:\n got:  %q\n want: %q", out.HTML, goldenHTML)
	}

	if len(out.CSS) != goldenCSSLen {
		t.Errorf("Options{} CSS length diverged from pre-TRD golden: got %d, want %d", len(out.CSS), goldenCSSLen)
	}
	if gotSum := fmt.Sprintf("%x", sha256.Sum256([]byte(out.CSS))); gotSum != goldenCSSSHA256 {
		t.Errorf("Options{} CSS SHA-256 diverged from pre-TRD golden: got %s, want %s", gotSum, goldenCSSSHA256)
	}
}

// TestBrowserFitJSReexport is Test-list case 2: press.BrowserFitJS() returns
// a non-empty string equal to press/themes.BrowserFitJS() -- proving the
// press-root re-export lets a press-only consumer (the CLI) reach the same
// verbatim fit script without importing press/themes itself. themes is
// imported ONLY inside this test to assert the equality; press/browserjs.go
// (production code) is the sole non-test importer.
func TestBrowserFitJSReexport(t *testing.T) {
	got := BrowserFitJS()
	want := themes.BrowserFitJS()

	if got == "" {
		t.Fatal("BrowserFitJS() returned an empty string")
	}
	if got != want {
		t.Errorf("press.BrowserFitJS() != press/themes.BrowserFitJS(): re-export diverged from source")
	}
	if !strings.Contains(got, "function") {
		t.Errorf("BrowserFitJS() does not look like JavaScript (no %q substring found)", "function")
	}
}
