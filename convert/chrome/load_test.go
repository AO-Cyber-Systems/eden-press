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

	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/convert"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// TestLoadHTMLReadsBackRealContent is Test-list case 4: LoadHTML +
// ApplyDeterminism on a real tab must load a hand-built <html> fixture via
// SetDocumentContent and, after document.fonts.ready resolves, a subsequent
// chromedp.Text read must return the fixture's actual content -- proving
// SetDocumentContent worked (as opposed to a blank data-URL truncation).
// Chrome-gated: skips cleanly when no Chrome/Chromium is discoverable (this
// sandbox has none), matching every other Chrome-presence-gated test in this
// package (session_test.go's TestSessionMultiTab).
func TestLoadHTMLReadsBackRealContent(t *testing.T) {
	if _, _, err := Discover(DiscoverOptions{}); err != nil {
		t.Skipf("no Chrome discovered, skipping live-Chrome load smoke: %v", err)
	}

	sess, err := New(convert.Options{})
	if err != nil {
		t.Skipf("could not start a Chrome session, skipping live-Chrome load smoke: %v", err)
	}
	defer sess.Close()

	tab, cancel := sess.NewTab()
	defer cancel()

	if err := ApplyDeterminism(tab, 1280, 720); err != nil {
		t.Fatalf("ApplyDeterminism: %v", err)
	}

	const fixture = `<html><body><p id="x">hello</p></body></html>`
	if err := LoadHTML(tab, fixture); err != nil {
		t.Fatalf("LoadHTML: %v", err)
	}

	var text string
	if err := chromedp.Run(tab, chromedp.Text("#x", &text)); err != nil {
		t.Fatalf("reading back loaded content via chromedp.Text: %v", err)
	}
	if got := strings.TrimSpace(text); got != "hello" {
		t.Fatalf("SetDocumentContent did not load the real fixture content: got %q, want %q", got, "hello")
	}
}

// TestSelfContainmentContract is Test-list case 5: it RESOLVES 05-RESEARCH
// Open Question #1, against VERIFIED (not assumed) press.Render output
// shape. Ground truth, confirmed by direct experiment against this
// checkout's press/sanitize policy (press/sanitize/policy.go):
//
//   - press/sanitize's bluemonday policy never allow-lists the "style"
//     attribute at all (a pre-existing, documented limitation tracked in
//     03-08-SUMMARY.md, orthogonal to convert/), so a Marpit background image
//     (`![bg](...)`, rendered as `<figure style="background-image:url(...)">`
//     by chase/markdown/render.go's figureStyle) is stripped ENTIRELY --
//     relative or absolute makes no difference, because the whole style
//     attribute is gone before a URL is ever considered.
//   - Independently, the policy calls RequireParseableURLs(true) but never
//     AllowRelativeURLs(true) (press/sanitize/policy.go), so bluemonday's
//     validURL (microcosm-cc/bluemonday sanitize.go) rejects ANY schemeless
//     (relative) URL on an allow-listed attribute too -- e.g. a plain
//     `![alt](relative.png)` inline image loses its `src` the same way.
//   - An ABSOLUTE http(s) URL on an allow-listed attribute (inline `<img
//     src="https://...">`) is the one shape that survives sanitize verbatim.
//
// Net effect: under press.Render's default (built-in) sanitize policy,
// relative asset references are ALREADY stripped before they ever reach
// convert/ -- they are not currently a self-containment risk in practice.
// convert/ still adds NO asset-inlining pre-pass of its own (v1 scope): if a
// deck author's markup produces a surviving ABSOLUTE remote URL (or a
// caller supplies a custom opts.Sanitize policy that allows relative ones),
// making that reference resolvable (e.g. via a data: URI) is the deck
// author's / caller's responsibility, not convert/'s.
func TestSelfContainmentContract(t *testing.T) {
	t.Run("image-free deck is self-contained", func(t *testing.T) {
		const deck = "---\nmarp: true\n---\n\n# Title\n\nNo images anywhere in this deck.\n"

		out, err := press.Render(deck, press.Options{})
		if err != nil {
			t.Fatalf("press.Render: %v", err)
		}

		doc := "<style>" + ComposeCSS(out.CSS) + "</style>" + out.HTML

		if strings.Contains(doc, "url(http") {
			t.Fatalf("composed HTML+CSS for an image-free deck contains an external url(http...) reference Chrome would fetch:\n%s", doc)
		}
		if strings.Contains(doc, `src="http`) || strings.Contains(doc, `src='http`) {
			t.Fatalf("composed HTML+CSS for an image-free deck contains an external src=\"http...\" reference Chrome would fetch:\n%s", doc)
		}
	})

	t.Run("relative background image is stripped by press/sanitize before reaching convert (pre-existing, not convert's concern)", func(t *testing.T) {
		const relPath = "relative-image.png"
		deck := "---\nmarp: true\n---\n\n![bg](" + relPath + ")\n\n# Slide with a relative background\n"

		out, err := press.Render(deck, press.Options{})
		if err != nil {
			t.Fatalf("press.Render: %v", err)
		}

		doc := "<style>" + ComposeCSS(out.CSS) + "</style>" + out.HTML

		// VERIFIED (see doc comment above): the whole `style` attribute --
		// and with it any background-image reference, relative or absolute
		// -- is stripped by press/sanitize's bluemonday policy. This
		// assertion pins that already-verified behavior so a future policy
		// change is caught here rather than silently reintroducing a
		// self-containment gap.
		if strings.Contains(doc, relPath) {
			t.Fatalf("expected press/sanitize to strip the relative background image reference %q entirely, but it survived into the composed output -- press/sanitize's policy may have changed, which would reopen a self-containment question for convert/:\n%s", relPath, doc)
		}
		if strings.Contains(doc, `style="`) {
			t.Fatalf("expected press/sanitize to strip the style attribute entirely (no style= survives sanitize today), but one was found -- press/sanitize's policy may have changed:\n%s", doc)
		}
	})

	t.Run("absolute remote image survives sanitize verbatim -- self-containment for it is the deck author's responsibility, not convert's", func(t *testing.T) {
		const remoteURL = "https://example.com/abs-image.png"
		deck := "---\nmarp: true\n---\n\n![alt text](" + remoteURL + ")\n\n# Slide with a remote image\n"

		out, err := press.Render(deck, press.Options{})
		if err != nil {
			t.Fatalf("press.Render: %v", err)
		}

		doc := "<style>" + ComposeCSS(out.CSS) + "</style>" + out.HTML

		// DOCUMENTED CONTRACT (05-RESEARCH Open Question #1, RESOLVED): an
		// absolute http(s) image URL a deck author writes survives sanitize
		// and lands in convert/'s input verbatim. convert/ does NOT add an
		// asset-inlining pre-pass in v1 -- if Chrome needs live network
		// access to fetch it, making the deck self-contained (e.g. by
		// pre-embedding it as a data: URI) is the deck author's
		// responsibility, not convert/'s.
		if !strings.Contains(doc, remoteURL) {
			t.Fatalf("expected the absolute remote image URL %q to survive sanitize into the composed output (documenting the author-responsibility contract), but it was not found -- the deck markup shape may have changed:\n%s", remoteURL, doc)
		}
	})
}
