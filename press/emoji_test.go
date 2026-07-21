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
