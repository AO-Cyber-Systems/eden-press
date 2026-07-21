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

// Port of Marpit's `![bg …]` alt-text option grammar (background_image/parse.js
// + image/parse.js), ported verbatim branch-by-branch (PARSE-06).
//
// Marpit parses an image's alt text as a space-separated list of options in
// TWO passes:
//
//  1. image/parse.js's generic matchers -- resize percentage, w:/h: with
//     units, and the 9 CSS filter functions -- run over EVERY image
//     (background or not).
//  2. background_image/parse.js's own pass -- ONLY when at least one
//     unconsumed option is the literal "bg" keyword -- recognizes the "bg"
//     keyword itself, the size keywords (auto|contain|cover|fit, fit
//     aliasing contain), the left/right split matcher, and the
//     vertical/horizontal direction keyword.
//
// Both passes mark each option "consumed" as they match it; whatever is left
// unconsumed becomes the figcaption alt text (Alt field).
package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// bgFilter is one parsed CSS filter function/amount pair, e.g. {"blur",
// "10px"}, in the order encountered in the alt text.
type bgFilter struct {
	Name   string
	Amount string
}

// BgOptions is the structured result of parsing an image's alt text as
// Marpit background-image options (mirrors marpitImage's merged meta shape).
type BgOptions struct {
	// Background reports whether the alt text contained the "bg" keyword.
	// All other fields are meaningless (zero) unless this is true.
	Background bool

	// SizeKeyword is one of "auto", "contain", "cover" -- "fit" is an ALIAS
	// for "contain", never a distinct value. Empty when no size keyword
	// option was present.
	SizeKeyword string

	// ResizeSize is a bare "NN%"/"NN.N%" resize option, empty if absent.
	ResizeSize string

	// Width/Height are the w:/h: option values, unit-normalized (a bare
	// number becomes "Npx"). Empty if absent.
	Width  string
	Height string

	// Filters holds every parsed CSS filter function, in alt-text order.
	Filters []bgFilter

	// SplitSide is "left" or "right" (empty if no split option).
	SplitSide string

	// SplitSize is the split's optional "NN%" (empty defaults to 50% --
	// applied by the caller, not here, mirroring Marpit's own
	// `splitSize || '50%'` deferral to the advanced-background builder).
	SplitSize string

	// Direction is "vertical" or "horizontal" (empty if unspecified --
	// defaults to "horizontal", applied by the caller).
	Direction string

	// Alt is the leftover (non-option) alt text, used as the figcaption
	// when non-empty.
	Alt string
}

// FilterCSS joins Filters into a single CSS `filter` property value, e.g.
// "blur(10px) brightness(1.5)". Returns "" if there are no filters.
func (o BgOptions) FilterCSS() string {
	if len(o.Filters) == 0 {
		return ""
	}
	parts := make([]string, len(o.Filters))
	for i, f := range o.Filters {
		parts[i] = f.Name + "(" + f.Amount + ")"
	}
	return strings.Join(parts, " ")
}

// EffectiveSize computes the figure/backgroundImage's effective
// `background-size` value, porting background_image/apply.js's IIFE
// verbatim:
//
//	s = size || backgroundSize || undefined
//	return !['contain','cover'].includes(s) && (width||height)
//	  ? `${width||s||'auto'} ${height||s||'auto'}`
//	  : s
func (o BgOptions) EffectiveSize() string {
	s := o.ResizeSize
	if s == "" {
		s = o.SizeKeyword
	}
	if s != "contain" && s != "cover" && (o.Width != "" || o.Height != "") {
		w := o.Width
		if w == "" {
			w = firstNonEmpty(s, "auto")
		}
		h := o.Height
		if h == "" {
			h = firstNonEmpty(s, "auto")
		}
		return w + " " + h
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// bgOption is one space-separated token of an image's alt text, tracking its
// leading whitespace run (needed to faithfully reconstruct leftover alt
// text) and whether a matcher has already consumed it.
type bgOption struct {
	content  string
	leading  string
	consumed bool
}

// splitOptions tokenizes alt into content/whitespace runs, mirroring
// image/parse.js's `token.content.split(/(\s+)/)` reduce.
func splitOptions(alt string) []*bgOption {
	var opts []*bgOption
	leading := ""
	for _, part := range reSplitToken.FindAllString(alt, -1) {
		if strings.TrimSpace(part) == "" {
			leading += part
			continue
		}
		opts = append(opts, &bgOption{content: part, leading: leading})
		leading = ""
	}
	return opts
}

var reSplitToken = regexp.MustCompile(`\s+|\S+`)

// Generic (non-bg-specific) option matchers, ported from image/parse.js.
var (
	rePercent = regexp.MustCompile(`^(\d*\.)?\d+%$`)
	reWidth   = regexp.MustCompile(`^w(?:idth)?:((?:\d*\.)?\d+(?:%|ch|cm|em|ex|in|mm|pc|pt|px)?|auto)$`)
	reHeight  = regexp.MustCompile(`^h(?:eight)?:((?:\d*\.)?\d+(?:%|ch|cm|em|ex|in|mm|pc|pt|px)?|auto)$`)
	reBareNum = regexp.MustCompile(`^(\d*\.)?\d+$`)
	reEscape  = regexp.MustCompile(`[\\;:()]`)
)

// normalizeLength appends "px" to a bare number, mirroring
// image/parse.js's normalizeLength.
func normalizeLength(v string) string {
	if reBareNum.MatchString(v) {
		return v + "px"
	}
	return v
}

// escape mirrors image/parse.js's CSS-escape helper: every `\;:()`
// character becomes `\{hex} ` (backslash, lowercase hex code point, space).
func escape(target string) string {
	return reEscape.ReplaceAllStringFunc(target, func(m string) string {
		return fmt.Sprintf(`\%x `, m[0])
	})
}

// filterMatcher is one of the 9 simple (single-optional-argument) CSS
// filter function matchers.
type filterMatcher struct {
	name string
	re   *regexp.Regexp
	dflt string
}

// filterMatchers ports image/parse.js's optionMatchers filter entries
// verbatim, in DECLARATION order (first match wins per option) with their
// documented default amounts (01-RESEARCH.md filter defaults table).
var filterMatchers = []filterMatcher{
	{"blur", regexp.MustCompile(`^blur(?::(.+))?$`), "10px"},
	{"brightness", regexp.MustCompile(`^brightness(?::(.+))?$`), "1.5"},
	{"contrast", regexp.MustCompile(`^contrast(?::(.+))?$`), "2"},
	{"grayscale", regexp.MustCompile(`^grayscale(?::(.+))?$`), "1"},
	{"hue-rotate", regexp.MustCompile(`^hue-rotate(?::(.+))?$`), "180deg"},
	{"invert", regexp.MustCompile(`^invert(?::(.+))?$`), "1"},
	{"opacity", regexp.MustCompile(`^opacity(?::(.+))?$`), ".5"},
	{"saturate", regexp.MustCompile(`^saturate(?::(.+))?$`), "2"},
	{"sepia", regexp.MustCompile(`^sepia(?::(.+))?$`), "1"},
}

// reDropShadow ports drop-shadow's multi-arg matcher verbatim: up to 4
// comma-separated arguments, all optional.
var reDropShadow = regexp.MustCompile(`^drop-shadow(?::(.+?),(.+?)(?:,(.+?))?(?:,(.+?))?)?$`)

// reColorFunc recognizes a CSS color function so drop-shadow's args can
// escape only the function's inner content, mirroring image/parse.js's
// colorFunc regex.
var reColorFunc = regexp.MustCompile(`^(rgba?|hsla?|hwb|(?:ok)?(?:lab|lch)|color)\((.*)\)$`)

// matchFilter tries every filter matcher (simple 9 + drop-shadow) against
// content, returning the first match's (name, amount).
func matchFilter(content string) (name, amount string, ok bool) {
	for _, fm := range filterMatchers {
		m := fm.re.FindStringSubmatch(content)
		if m == nil {
			continue
		}
		amt := fm.dflt
		if m[1] != "" {
			amt = m[1]
		}
		return fm.name, escape(amt), true
	}
	if m := reDropShadow.FindStringSubmatch(content); m != nil {
		var args []string
		for _, g := range m[1:] {
			if g == "" {
				continue
			}
			if cm := reColorFunc.FindStringSubmatch(g); cm != nil {
				args = append(args, cm[1]+"("+escape(cm[2])+")")
			} else {
				args = append(args, escape(g))
			}
		}
		joined := strings.Join(args, " ")
		if joined == "" {
			joined = "0 5px 10px rgba(0,0,0,.4)"
		}
		return "drop-shadow", joined, true
	}
	return "", "", false
}

// bgSizeKeywords ports background_image/parse.js's bgSizeKeywords table --
// "fit" is an ALIAS for "contain", never a distinct value.
var bgSizeKeywords = map[string]string{
	"auto":    "auto",
	"contain": "contain",
	"cover":   "cover",
	"fit":     "contain",
}

// reSplit ports background_image/parse.js's splitSizeMatcher verbatim.
var reSplit = regexp.MustCompile(`^(left|right)(?::((?:\d*\.)?\d+%))?$`)

// ParseBgOptions parses an image's alt text as Marpit background-image
// options, porting background_image/parse.js + image/parse.js's two-pass
// matcher application verbatim.
func ParseBgOptions(alt string) BgOptions {
	tokens := splitOptions(alt)
	var opts BgOptions

	// Pass 1 (image/parse.js): resize percentage, w:/h:, filters -- run
	// over every option regardless of whether this ends up being a bg
	// image at all.
	for _, t := range tokens {
		if t.consumed {
			continue
		}
		switch {
		case rePercent.MatchString(t.content):
			opts.ResizeSize = t.content
			t.consumed = true
		case reWidth.MatchString(t.content):
			opts.Width = normalizeLength(reWidth.FindStringSubmatch(t.content)[1])
			t.consumed = true
		case reHeight.MatchString(t.content):
			opts.Height = normalizeLength(reHeight.FindStringSubmatch(t.content)[1])
			t.consumed = true
		default:
			if name, amount, ok := matchFilter(t.content); ok {
				opts.Filters = append(opts.Filters, bgFilter{Name: name, Amount: amount})
				t.consumed = true
			}
		}
	}

	// Trigger check: background_image/parse.js's pass only runs when at
	// least one UNCONSUMED option is the literal "bg" keyword.
	hasBg := false
	for _, t := range tokens {
		if !t.consumed && t.content == "bg" {
			hasBg = true
			break
		}
	}
	if !hasBg {
		return opts
	}
	opts.Background = true

	// Pass 2 (background_image/parse.js): bg keyword, size keyword, split,
	// direction -- skips options already consumed by pass 1.
	for _, t := range tokens {
		if t.consumed {
			continue
		}
		consumed := false
		if t.content == "bg" {
			consumed = true
		}
		if kw, ok := bgSizeKeywords[t.content]; ok {
			opts.SizeKeyword = kw
			consumed = true
		}
		if m := reSplit.FindStringSubmatch(t.content); m != nil {
			opts.SplitSide = m[1]
			opts.SplitSize = m[2]
			consumed = true
		}
		if t.content == "vertical" || t.content == "horizontal" {
			opts.Direction = t.content
			consumed = true
		}
		if consumed {
			t.consumed = true
		}
	}

	var altBuf strings.Builder
	for _, t := range tokens {
		if !t.consumed {
			altBuf.WriteString(t.leading)
			altBuf.WriteString(t.content)
		}
	}
	opts.Alt = strings.TrimLeft(altBuf.String(), " \t\n\r\f\v")

	return opts
}
