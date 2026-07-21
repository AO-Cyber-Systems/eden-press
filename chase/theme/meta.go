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

package theme

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// metaLineRE mirrors Marpit's postcss/meta.js EXACTLY:
// `/^[*!\s]*@([\w-]+)\s+(.+)$/gim` (verified directly against the vendored
// @marp-team/marpit/lib/postcss/meta.js source). Applied per physical line
// (Go's `(?m)` multiline flag, matching JS's `m` flag) across a comment's
// raw inner text, case-insensitively (`(?i)`/`i`), it matches an
// `@key value` line optionally prefixed by block-comment decoration — the
// leading run of `*`, `!`, and/or whitespace that begins each line of a
// `/** ... */` block comment (e.g. " * ").
var metaLineRE = regexp.MustCompile(`(?mi)^[*!\s]*@([\w-]+)\s+(.+)$`)

// sizeLineRE parses one @size metadata value's full form: `<name> <W>px
// <H>px` (Test-list case 7). A bare `<name>` alone (no explicit pixel
// dimensions) does not match this and falls through to the caller-supplied
// sizeFallback table in parseSizeValue.
var sizeLineRE = regexp.MustCompile(`^(\S+)\s+(\d+(?:\.\d+)?)px\s+(\d+(?:\.\d+)?)px$`)

// ParseMeta extracts a theme's identity metadata from its LEADING CSS
// comment block: `@theme <name>` (REQUIRED — THEME-02's acceptance point,
// see Meta's doc), `@size <name> <W>px <H>px` (repeatable, building the
// named size table), and `@auto-scaling <value>` (captured verbatim).
// sizeFallback resolves a bare `@size <name>` line (no explicit pixel
// dimensions — e.g. a corpus/stress-theme fixture's shorthand keyword) —
// see parseSizeValue; the caller (ultimately, the active Profile — see
// chase/profile) supplies this table, since chase/theme has no
// profile-specific size keywords of its own (TRD 02-03, MODEL-04's
// de-hardcoding move).
//
// Scope note: postcss/meta.js's `css.walkComments` scans EVERY comment
// anywhere in the document; ParseMeta deliberately scans only the LEADING
// comment (per the TRD's Task 3 action: "Extract metadata from the theme's
// leading comment block") since every corpus theme and the RESEARCH stress
// theme fixture author identity metadata as a single leading block comment
// — this is a narrower, TRD-scoped mirror of meta.js's regex mechanism, not
// a full port of walkComments' whole-document behavior.
//
// Returns an error if no @theme name is present, or if an @size line's
// value can't be parsed — never silently defaulting or dropping either.
func ParseMeta(cssText string, sizeFallback map[string]Size) (Meta, error) {
	raw := leadingComment(cssText)

	m := Meta{Raw: raw, Sizes: map[string]Size{}}
	for _, match := range metaLineRE.FindAllStringSubmatch(raw, -1) {
		key := strings.ToLower(match[1])
		value := strings.TrimSpace(match[2])

		switch key {
		case "theme":
			m.Name = value
		case "size":
			sz, err := parseSizeValue(value, sizeFallback)
			if err != nil {
				return Meta{}, fmt.Errorf("theme: invalid @size %q: %w", value, err)
			}
			m.Sizes[sz.Name] = sz
		case "auto-scaling":
			m.AutoScaling = value
		}
	}

	if m.Name == "" {
		return Meta{}, fmt.Errorf("theme: missing required @theme metadata")
	}
	return m, nil
}

// leadingComment returns a theme CSS string's leading comment's inner text
// (the raw text between "/*" and "*/", exclusive — matching PostCSS's own
// `comment.text` node property), or "" if the stylesheet doesn't open with
// a comment at all.
//
// It walks the tdewolff/parse/v2/css grammar stream just far enough to
// capture the FIRST CommentGrammar event — a real css.Parser-recognized
// comment token (see parse.go's Parse, whose own CommentGrammar case
// defers to this function rather than modeling comments itself) — not a
// naive "/*"..."*/" text search, so it can't be fooled by comment-like
// byte sequences the lexer wouldn't itself treat as a comment.
func leadingComment(cssText string) string {
	p := css.NewParser(parse.NewInputString(cssText), false)
	gt, _, data := p.Next()
	if gt != css.CommentGrammar {
		return ""
	}
	text := string(data)
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	return text
}

// parseSizeValue parses one @size metadata value into a Size.
//
// The full form is `<name> <W>px <H>px` (Test-list case 7). A bare `<name>`
// alone — no explicit pixel dimensions, e.g. a stress-theme fixture's bare
// keyword shorthand — resolves via the caller-supplied sizeFallback table
// (see ParseMeta's doc).
func parseSizeValue(value string, sizeFallback map[string]Size) (Size, error) {
	if m := sizeLineRE.FindStringSubmatch(value); m != nil {
		w, wErr := strconv.ParseFloat(m[2], 64)
		h, hErr := strconv.ParseFloat(m[3], 64)
		if wErr != nil || hErr != nil {
			return Size{}, fmt.Errorf("non-numeric @size dimensions in %q", value)
		}
		return Size{Name: m[1], WidthPx: int(w), HeightPx: int(h)}, nil
	}

	name := strings.TrimSpace(value)
	if sz, ok := sizeFallback[name]; ok {
		return sz, nil
	}
	return Size{}, fmt.Errorf(
		`unrecognized @size format (want "<name> <W>px <H>px", or a name recognized by the active profile's size-fallback table)`,
	)
}

// ParseTheme parses a complete theme CSS string into a fully-populated
// Stylesheet: structural Rules/Atoms via Parse, plus identity Meta via
// ParseMeta. It returns an error if EITHER the structural parse fails OR
// @theme metadata is missing (see ParseMeta's doc) — a theme CSS string
// isn't a valid chase theme without both. sizeFallback is threaded
// through to ParseMeta (see its doc).
func ParseTheme(cssText string, sizeFallback map[string]Size) (Stylesheet, error) {
	sheet, err := Parse(cssText)
	if err != nil {
		return Stylesheet{}, err
	}

	meta, err := ParseMeta(cssText, sizeFallback)
	if err != nil {
		return Stylesheet{}, err
	}

	sheet.Meta = meta
	return sheet, nil
}
