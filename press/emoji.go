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

// CORE-06: native emoji, no JavaScript. This file wires the SHORTCODE half
// (":smile:") entirely by REUSING github.com/yuin/goldmark-emoji v1.0.6 --
// its emoji.New(...) is a goldmark.Extender that already registers an
// InlineParser (triggers on ':', shortcode table from definition.Github())
// and a NodeRenderer (emoji.WithRenderingMethod(emoji.Twemoji) renders a
// static <img> tag). Nothing here hand-rolls a shortcode table or a ':'
// parser -- research's Don't-Hand-Roll row 1.
//
// The OTHER half of CORE-06 -- literal unicode emoji typed directly in prose
// -- is emoji_unicode.go's bespoke InlineParser. Task 2 folds its
// unicodeEmojiExtender into emojiOptionWithTwemoji below, bundling both
// halves into the ONE goldmark.Option press.Render (03-09) will fold into
// its engine. goldmark-emoji's own parser only triggers on ':'; it never
// looks for raw unicode runes, so that gap is the entire bespoke surface
// this TRD adds (research Don't-Hand-Roll row 2).
package press

import (
	"fmt"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
)

// defaultTwemojiBase and defaultTwemojiExt reproduce goldmark-emoji's own
// emoji.DefaultTwemojiTemplate (jsDelivr's twemoji CDN, 72x72 PNG assets) as
// separately overridable base/ext values -- Marp's own base/ext contract
// (FEATURES.md: "twemoji sub-options base (CDN/local path) and ext
// (svg/png)"), so a caller can self-host twemoji assets for air-gapped /
// untrusted-input deployments instead of depending on the public CDN.
const (
	defaultTwemojiBase = "https://cdn.jsdelivr.net/gh/twitter/twemoji@latest/assets/72x72/"
	defaultTwemojiExt  = ".png"
)

// TwemojiOptions configures the twemoji CDN/base + file extension used when
// rendering emoji as <img> tags -- the base/ext half of Marp's emoji
// contract (FEATURES.md). Base must end in "/"; Ext must include the leading
// dot (e.g. ".png" or ".svg").
type TwemojiOptions struct {
	// Base is the CDN or local base path emoji <img src> values are resolved
	// against, e.g. "https://cdn.jsdelivr.net/gh/twitter/twemoji@latest/assets/72x72/".
	Base string

	// Ext is the image file extension, including the leading dot, e.g.
	// ".png" or ".svg".
	Ext string
}

// DefaultTwemojiOptions returns the sensible default: jsDelivr's public
// twemoji CDN serving 72x72 PNG assets -- byte-identical to goldmark-emoji's
// own emoji.DefaultTwemojiTemplate, just expressed as overridable base/ext
// fields instead of a single opaque template string.
func DefaultTwemojiOptions() TwemojiOptions {
	return TwemojiOptions{Base: defaultTwemojiBase, Ext: defaultTwemojiExt}
}

// twemojiTemplate builds the printf template emoji.WithTwemojiTemplate
// expects, preserving goldmark-emoji's own three placeholders (%[1]s emoji
// name, %[2]s hex-codepoint file stem, %[3]s XHTML self-close marker) while
// substituting cfg's base/ext in place of the built-in CDN/extension.
func (cfg TwemojiOptions) twemojiTemplate() string {
	return fmt.Sprintf(`<img class="emoji" draggable="false" alt="%%[1]s" src="%s%%[2]s%s"%%[3]s>`, cfg.Base, cfg.Ext)
}

// emojiOption returns the goldmark.Option bundle press.Render (TRD 03-09)
// folds into NewEngine(pressExtraOpts...): goldmark-emoji's Twemoji
// shortcode rendering, reused verbatim, with the default CDN/ext. No
// JavaScript is involved anywhere -- twemoji <img> tags are static markup.
func emojiOption() goldmark.Option {
	return emojiOptionWithTwemoji(DefaultTwemojiOptions())
}

// emojiOptionWithTwemoji is emojiOption's configurable twin: cfg overrides
// the twemoji CDN/base + file extension (Marp's base/ext contract), e.g. for
// self-hosted or air-gapped twemoji assets.
func emojiOptionWithTwemoji(cfg TwemojiOptions) goldmark.Option {
	return goldmark.WithExtensions(
		emoji.New(
			emoji.WithRenderingMethod(emoji.Twemoji),
			emoji.WithTwemojiTemplate(cfg.twemojiTemplate()),
		),
	)
}
