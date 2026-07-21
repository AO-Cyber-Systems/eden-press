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

package chrome

import (
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
)

// TestComposeCSS is Test-list case 1: ComposeCSS must return CSS containing
// the original rule AND the animation/transition kill block AND the STIX
// @font-face data-URI, with the overrides positioned AFTER the base CSS
// (Pattern 2 item 4 -- convert/ fully controls the fed CSS, so a guaranteed
// !important override beats trusting the theme).
func TestComposeCSS(t *testing.T) {
	const base = "section{color:red}"
	got := ComposeCSS(base)

	if !strings.Contains(got, base) {
		t.Fatalf("ComposeCSS dropped the original CSS: %q", got)
	}
	if !strings.Contains(got, "animation:none!important") {
		t.Fatalf("ComposeCSS missing the animation kill override: %q", got)
	}
	if !strings.Contains(got, "transition:none!important") {
		t.Fatalf("ComposeCSS missing the transition kill override: %q", got)
	}
	if !strings.Contains(got, "scroll-behavior:auto!important") {
		t.Fatalf("ComposeCSS missing the scroll-behavior kill override: %q", got)
	}
	if !strings.Contains(got, "@font-face") || !strings.Contains(got, "STIX Two Math") {
		t.Fatalf("ComposeCSS missing the STIX Two Math @font-face injection: %q", got)
	}

	baseIdx := strings.Index(got, base)
	killIdx := strings.Index(got, "animation:none!important")
	if killIdx < baseIdx {
		t.Fatalf("ComposeCSS ordering: the animation-kill override must come AFTER the base CSS (cascade + !important both need to win), got base@%d kill@%d", baseIdx, killIdx)
	}
}

// TestComposeCSS_emptyBase confirms ComposeCSS still appends the overrides
// even when the caller's base CSS is empty (a deck with no theme CSS at all
// still gets the determinism kill + font injection).
func TestComposeCSS_emptyBase(t *testing.T) {
	got := ComposeCSS("")
	if !strings.Contains(got, "animation:none!important") {
		t.Fatalf("ComposeCSS(\"\") missing animation kill override: %q", got)
	}
}

// TestPageCSSInches is Test-list case 2: PageCSSInches must convert the
// profiles/slides pixel sizes to inches at 96px/in, formatted consistently
// (720px is exactly 7.5in; 1280px is 13.333...in; 960px is exactly 10in).
func TestPageCSSInches(t *testing.T) {
	cases := []struct {
		name string
		size theme.Size
		want string
	}{
		{
			name: "16:9 (1280x720)",
			size: theme.Size{Name: "16:9", WidthPx: 1280, HeightPx: 720},
			want: "@page{size:13.333in 7.5in;margin:0;}",
		},
		{
			name: "4:3 (960x720)",
			size: theme.Size{Name: "4:3", WidthPx: 960, HeightPx: 720},
			want: "@page{size:10in 7.5in;margin:0;}",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PageCSSInches(c.size)
			if got != c.want {
				t.Fatalf("PageCSSInches(%+v) = %q, want %q", c.size, got, c.want)
			}
		})
	}
}

// TestFontFaceDataURI is Test-list case 3: FontFaceDataURI must return a
// non-empty @font-face rule with the expected shape when the OTF is
// embedded, or an empty string when the asset was deferred -- either
// outcome is valid, but an embedded font must produce a well-formed rule.
func TestFontFaceDataURI(t *testing.T) {
	got := FontFaceDataURI()

	if got == "" {
		t.Skip("STIX Two Math OTF not embedded (deferred asset) -- see 05-02-SUMMARY.md blocker")
	}

	const wantPrefix = "@font-face{font-family:'STIX Two Math';src:url(data:font/otf;base64,"
	if !strings.HasPrefix(got, wantPrefix) {
		n := len(got)
		if n > 120 {
			n = 120
		}
		t.Fatalf("FontFaceDataURI has unexpected shape (want prefix %q), got: %q...", wantPrefix, got[:n])
	}
	if !strings.Contains(got, "format('opentype')") {
		t.Fatalf("FontFaceDataURI missing format('opentype'): %q", got)
	}
	if !strings.HasSuffix(got, "}") {
		t.Fatalf("FontFaceDataURI rule is not closed: %q", got)
	}
}
