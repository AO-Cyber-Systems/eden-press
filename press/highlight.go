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
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// highlightOption returns the goldmark.Option press.Render (TRD 03-09) folds
// into markdown.NewEngine(pressExtraOpts...) to satisfy CORE-07: syntax
// highlighting for fenced code blocks (```go ... ```).
//
// It REUSES github.com/yuin/goldmark-highlighting/v2's NewHighlighting as
// the fenced-code-block NodeRenderer -- language lookup (lexers.Get /
// lexers.Analyse), chroma.Coalesce, per-block attribute overrides
// (linenos/hl_lines/style/nohl), and the CSSWriter hook are all already
// built there (research Don't Hand-Roll row 3: no bespoke fenced-code
// NodeRenderer is written by this package). goldmark-highlighting registers
// its renderer at NodeRenderer priority 200 (Extend, highlighting.go),
// ahead of goldmark's own default FencedCodeBlock renderer, so it wins
// without disturbing chase/markdown.NewEngine's other renderer options
// (WithUnsafe/WithHardWraps) or its own Extender.
//
// This function's only job is wiring chromahtml.WithClasses(true) through
// WithFormatOptions -- WithClasses itself is NOT a top-level
// goldmark-highlighting option; it is a chroma HTML-formatter option that
// must be injected via the FormatOptions hook (research error_recovery: "if
// goldmark-highlighting emits inline styles instead of classes,
// WithClasses(true) wasn't passed through WithFormatOptions"). With classes
// on, chroma emits `<span class="kd">`, `<span class="s2">`, etc -- its own
// Pygments-style short token classes -- rather than inline `style="..."`
// attributes. remapHLJS (highlight_remap.go) rewrites those short classes
// to the .hljs-* names the bundled themes' CSS (themes/*.css, TRD 03-02)
// actually targets, as a SEPARATE bounded post-format string pass over the
// already-rendered HTML -- kept out of this wiring so the one-parse
// invariant (chase.Render/03-04 "one-parse-two-sinks") is never touched:
// this function only configures the parse-time renderer, never re-parses.
//
// style selects the chroma style used to tokenize/classify code
// (press.Options.HighlightStyle). "" omits WithStyle entirely, leaving
// goldmark-highlighting's own NewConfig() default ("github") in effect --
// deliberately reusing the library's default rather than inventing a
// press-local one. Note the chosen style has NO effect on which .hljs-*
// class a token ends up remapped to: chroma's HTML formatter resolves a
// span's class purely from the TOKEN TYPE (chroma.StandardTypes), never
// from the style, so any registered chroma style name is safe to pass here
// under WithClasses(true) -- the style would only matter for chroma's own
// generated colors, which this package never emits (the bundled themes'
// CSS supplies the colors instead, via the .hljs-* remap).
func highlightOption(style string) goldmark.Option {
	var hlOpts []highlighting.Option
	if style != "" {
		hlOpts = append(hlOpts, highlighting.WithStyle(style))
	}
	hlOpts = append(hlOpts, highlighting.WithFormatOptions(chromahtml.WithClasses(true)))

	return goldmark.WithExtensions(highlighting.NewHighlighting(hlOpts...))
}
