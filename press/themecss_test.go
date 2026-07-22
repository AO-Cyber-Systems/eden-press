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

// brandxCSS is a minimal, self-naming custom theme block used by Test-list
// cases 3 and 4: it self-names via its own leading `/* @theme brandx */`
// comment (theme.Load's requirement -- see chase/theme/parse.go's
// ParseTheme), and carries one rule whose color survives Pack's scoping
// pass, so a passing test proves the custom theme was actually packed (not
// merely registered without effect).
const brandxCSS = "/* @theme brandx */\nsection { color: #d4a853; }"

// TestCustomThemeByFrontMatter is Test-list case 3: a deck whose front
// matter selects `theme: brandx` packs the caller-supplied ThemeCSS block's
// scoped rule into Output.CSS, with no error -- proving
// opts.ThemeCSS -> theme.Load -> ts.Add makes a custom theme resolvable
// through the SAME front-matter `theme:` chain the 3 embedded themes use.
func TestCustomThemeByFrontMatter(t *testing.T) {
	deck := "---\ntheme: brandx\n---\n# Hi\n"
	out, err := Render(deck, Options{ThemeCSS: []string{brandxCSS}})
	if err != nil {
		t.Fatalf("Render with front-matter custom theme: unexpected error: %v", err)
	}
	if !strings.Contains(out.CSS, "#d4a853") {
		t.Errorf("Output.CSS does not contain the custom theme's scoped rule (#d4a853): CSS=%d bytes", len(out.CSS))
	}
}

// TestCustomThemeByOptsOverride is Test-list case 4: opts.Theme="brandx"
// selects the custom theme with NO front-matter `theme:` directive present
// -- proving opts.Theme overrides (here: supplies, absent any front matter)
// the theme name exactly as it does for the 3 embedded themes
// (TestThemeResolution in press_test.go).
func TestCustomThemeByOptsOverride(t *testing.T) {
	out, err := Render("# Hi\n", Options{Theme: "brandx", ThemeCSS: []string{brandxCSS}})
	if err != nil {
		t.Fatalf("Render with opts.Theme override to custom theme: unexpected error: %v", err)
	}
	if !strings.Contains(out.CSS, "#d4a853") {
		t.Errorf("Output.CSS does not contain the custom theme's scoped rule (#d4a853): CSS=%d bytes", len(out.CSS))
	}
}

// TestMalformedThemeCSSErrors is Test-list case 5: a ThemeCSS entry lacking
// the required leading `/* @theme name */` header makes Render return a
// wrapped "load custom theme CSS" error -- never a panic -- so a consumer
// like CLI-05's `--theme-set` gets a clean, surfaced failure for a bad file.
func TestMalformedThemeCSSErrors(t *testing.T) {
	malformed := "section { color: red; }" // no leading @theme comment
	_, err := Render("# Hi\n", Options{ThemeCSS: []string{malformed}})
	if err == nil {
		t.Fatal("Render with a malformed ThemeCSS block returned no error, want a wrapped load error")
	}
	if !strings.Contains(err.Error(), "load custom theme CSS") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "load custom theme CSS")
	}
}
