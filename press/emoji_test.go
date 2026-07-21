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
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark-emoji/definition"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

// renderEmoji builds a fresh markdown.NewEngine(emojiOption()) and converts
// md through it, failing the test on error -- the shared helper for every
// default-config case in this file (Test-list cases 1, 2, 3).
func renderEmoji(t *testing.T, md string) string {
	t.Helper()
	engine := markdown.NewEngine(emojiOption())
	var buf bytes.Buffer
	if err := engine.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("engine.Convert(%q) failed: %v", md, err)
	}
	return buf.String()
}

// renderEmojiWithTwemoji is renderEmoji's twin for a custom TwemojiOptions
// engine (Test-list case 4).
func renderEmojiWithTwemoji(t *testing.T, cfg TwemojiOptions, md string) string {
	t.Helper()
	engine := markdown.NewEngine(emojiOptionWithTwemoji(cfg))
	var buf bytes.Buffer
	if err := engine.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("engine.Convert(%q) failed: %v", md, err)
	}
	return buf.String()
}

// extractImgTag returns the substring from the first "<img" in html up to
// (and including) the next ">" -- the whole <img ...> tag, for shape
// comparisons between shortcode-origin and unicode-origin emoji.
func extractImgTag(t *testing.T, html string) string {
	t.Helper()
	return extractImgTagFrom(t, html, 0)
}

// extractImgTagFrom is extractImgTag starting the search at byte offset from.
func extractImgTagFrom(t *testing.T, html string, from int) string {
	t.Helper()
	rel := strings.Index(html[from:], "<img")
	if rel < 0 {
		t.Fatalf("no <img tag found in %q (searching from byte %d)", html, from)
	}
	start := from + rel
	end := strings.Index(html[start:], ">")
	if end < 0 {
		t.Fatalf("unterminated <img tag in %q", html)
	}
	return html[start : start+end+1]
}

// TestEmojiShortcode is Test-list case 1: markdown.NewEngine(emojiOption())
// renders ":smile:" to a twemoji <img ...> (NOT the literal ":smile:"),
// entirely via goldmark-emoji's reused Twemoji rendering method.
func TestEmojiShortcode(t *testing.T) {
	html := renderEmoji(t, ":smile:")

	if !strings.Contains(html, "<img") {
		t.Fatalf(":smile: did not render an <img> tag; got %q", html)
	}
	if !strings.Contains(html, `class="emoji"`) {
		t.Fatalf(`twemoji <img> missing class="emoji"; got %q`, html)
	}
	if !strings.Contains(html, "twemoji") {
		t.Fatalf("twemoji <img> missing the twemoji CDN src; got %q", html)
	}
	if strings.Contains(html, ":smile:") {
		t.Fatalf("shortcode :smile: was left unrendered; got %q", html)
	}
}

// TestEmojiBaseExt is Test-list case 4: an emojiOption variant with a custom
// twemoji base/ext produces <img src> under that base -- proving the Marp
// base/ext contract (FEATURES.md) is honored, not hardcoded.
func TestEmojiBaseExt(t *testing.T) {
	cfg := TwemojiOptions{Base: "/assets/emoji/", Ext: ".svg"}
	html := renderEmojiWithTwemoji(t, cfg, ":smile:")

	if !strings.Contains(html, `src="/assets/emoji/1f604.svg"`) {
		t.Fatalf("custom base/ext not honored; want src under /assets/emoji/ with .svg, got %q", html)
	}
	if strings.Contains(html, "jsdelivr") || strings.Contains(html, "twemoji@latest") {
		t.Fatalf("default twemoji CDN leaked through despite custom base/ext; got %q", html)
	}
}

// TestEmojiUnicode is Test-list case 2: the same engine renders a LITERAL
// unicode emoji (U+1F604, the SAME rune :smile: resolves to) typed directly
// in prose to the SAME shape of twemoji <img> as the shortcode -- proving
// both origins share one render path (goldmark-emoji's emojiHTMLRenderer),
// not two.
func TestEmojiUnicode(t *testing.T) {
	shortcodeHTML := renderEmoji(t, ":smile:")
	unicodeHTML := renderEmoji(t, "Hello \U0001F604 world")

	shortcodeImg := extractImgTag(t, shortcodeHTML)
	unicodeImg := extractImgTag(t, unicodeHTML)

	if shortcodeImg != unicodeImg {
		t.Fatalf("shortcode and literal-unicode emoji rendered differently (two render paths, not one):\nshortcode: %s\nunicode:   %s", shortcodeImg, unicodeImg)
	}
	if !strings.Contains(unicodeHTML, "Hello") || !strings.Contains(unicodeHTML, "world") {
		t.Fatalf("surrounding prose text was not preserved around the literal emoji; got %q", unicodeHTML)
	}
}

// TestEmojiMixed is Test-list case 3: a mixed line renders BOTH a shortcode
// and a literal-unicode emoji as <img> tags and preserves the surrounding
// text -- ":tada:" and its literal unicode twin "\U0001F389" (party popper)
// must render byte-identically, since both resolve through the same
// *definition.Emoji entry and the same NodeRenderer.
func TestEmojiMixed(t *testing.T) {
	html := renderEmoji(t, "hi :tada: \U0001F389 there")

	if n := strings.Count(html, "<img"); n != 2 {
		t.Fatalf("expected exactly 2 <img> tags (shortcode tada + literal tada), got %d: %q", n, html)
	}

	hiIdx := strings.Index(html, "hi")
	firstImgIdx := strings.Index(html, "<img")
	thereIdx := strings.LastIndex(html, "there")
	secondImgIdx := strings.LastIndex(html, "<img")

	if hiIdx < 0 || firstImgIdx < 0 || thereIdx < 0 || secondImgIdx < 0 {
		t.Fatalf("expected \"hi\", \"there\" and two <img> tags all present; got %q", html)
	}
	if !(hiIdx < firstImgIdx && firstImgIdx < secondImgIdx && secondImgIdx < thereIdx) {
		t.Fatalf("surrounding text ordering not preserved (want hi < img1 < img2 < there); got %q", html)
	}

	firstImg := extractImgTagFrom(t, html, firstImgIdx)
	secondImg := extractImgTagFrom(t, html, secondImgIdx)
	if firstImg != secondImg {
		t.Fatalf(":tada: shortcode and literal party-popper emoji did not render identically:\n%s\n%s", firstImg, secondImg)
	}
}

// TestEmojiReverseIndex is Test-list case 5: a unit test isolated from
// rendering. The rune -> *definition.Emoji reverse index built from
// definition.Github() maps U+1F604's rune sequence to the SAME entry
// definition.Github().Get("smile") resolves to (lookup correctness).
func TestEmojiReverseIndex(t *testing.T) {
	index, maxRunes := buildUnicodeEmojiIndex()

	if maxRunes < 1 {
		t.Fatalf("buildUnicodeEmojiIndex reported maxRunes=%d, want >= 1", maxRunes)
	}

	key := string([]rune{0x1F604}) // the literal rune :smile: resolves to (😄)
	entry, ok := index[key]
	if !ok {
		t.Fatalf("reverse index has no entry for U+1F604 (😄)")
	}

	want, ok := definition.Github().Get("smile")
	if !ok {
		t.Fatalf("definition.Github() has no \"smile\" entry (test setup broken)")
	}
	if entry != want {
		t.Fatalf("reverse index entry for 😄 is not the same *definition.Emoji that Github().Get(\"smile\") resolves to")
	}
}
