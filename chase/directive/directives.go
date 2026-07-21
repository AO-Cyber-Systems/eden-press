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

package directive

import (
	"strconv"
	"strings"
)

// ThemeExists is an injected predicate for the "theme" global directive's
// membership check (`marpit.themeSet.has(v)` in directives.js). chase/directive
// does not own the theme set (chase/theme does) -- accepting a predicate
// here keeps `theme: gaia` resolution testable without importing chase/theme,
// preserving the zero-cross-import boundary between the two packages.
type ThemeExists func(name string) bool

// CoerceGlobal normalizes a single global directive's raw value, mirroring
// directives.js's globals{} table (theme/headingDivider/style/lang).
// isKnown reports whether key names a recognized global directive at all;
// value is nil when the key IS recognized but this particular raw value
// does not produce an applicable directive (e.g. an unrecognized theme name
// -- "silently dropped, not errored", per the must_haves truth).
//
// headingDivider deliberately folds Marpit's TWO-stage int expansion into a
// single step here: directives.js's own headingDivider() function returns a
// bare int for a scalar value (e.g. "2" -> 2), and it is only later, in
// heading_divider.js's ASTTransformer-equivalent core rule (line 29,
// `[...Array(target).keys()].map(i => i + 1)`), that a scalar int gets
// expanded into the [1..N] range. TRD 01-02's task action + Test-list case 5
// both specify "headingDivider: 2 -> [1,2]" as THIS package's coercion
// output, so the expansion is folded in here -- chase/markdown (01-05/06)
// then only has to handle "false" or an already-resolved []int, without
// re-deriving the int-to-range expansion itself.
func CoerceGlobal(key string, raw RawValue, themeExists ThemeExists) (value any, isKnown bool) {
	switch key {
	case "headingDivider":
		return coerceHeadingDivider(raw), true
	case "style":
		return raw, true
	case "theme":
		s, isStr := raw.(string)
		if !isStr || themeExists == nil || !themeExists(s) {
			return nil, true // known directive name, unknown/no theme -> silently dropped
		}
		return s, true
	case "lang":
		return raw, true
	default:
		return nil, false
	}
}

// coerceHeadingDivider ports directives.js's headingDivider() coercion,
// folding in heading_divider.js's scalar-to-range expansion (see CoerceGlobal
// doc comment above).
func coerceHeadingDivider(raw RawValue) any {
	switch v := raw.(type) {
	case []string:
		// Array input: filter the fixed [1..6] heading levels down to those
		// present in the given array (mirrors
		// `headings.filter(v => convertedArr.includes(v))`) -- no
		// scalar-to-range expansion applies to an already-explicit array.
		present := map[int]bool{}
		for _, s := range v {
			if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				present[n] = true
			}
		}
		filtered := []int{}
		for h := 1; h <= 6; h++ {
			if present[h] {
				filtered = append(filtered, h)
			}
		}
		return filtered
	case string:
		if v == "false" {
			return false
		}
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 1 && n <= 6 {
			out := make([]int, n)
			for i := 0; i < n; i++ {
				out[i] = i + 1
			}
			return out
		}
		return nil // out-of-range / non-numeric -> "return {}" in directives.js
	default:
		return nil
	}
}

// CoerceLocal normalizes a single local directive's raw value, mirroring
// directives.js's locals{} table. Same (value, isKnown) contract as
// CoerceGlobal.
//
// NOTE: "backgroundSplit" is intentionally NOT included here. It is not a
// real text-comment/front-matter local directive in the verified
// directives.js locals{} table -- it is derived from the `![bg left:30%]`
// background-image markdown-syntax option grammar
// (background_image/parse.js), which belongs to chase/markdown's PARSE-05
// scope, not this package. Fabricating it here would recognize a directive
// key Marpit itself does not support via comments/front-matter.
func CoerceLocal(key string, raw RawValue) (value any, isKnown bool) {
	switch key {
	case "backgroundColor", "backgroundImage", "backgroundPosition",
		"backgroundRepeat", "backgroundSize", "color":
		return raw, true
	case "class":
		switch v := raw.(type) {
		case []string:
			return strings.Join(v, " "), true
		case string:
			return v, true
		default:
			return nil, true
		}
	case "footer":
		if s, ok := raw.(string); ok {
			return s, true
		}
		return nil, true // guards against non-string values (RESEARCH coercion table)
	case "header":
		if s, ok := raw.(string); ok {
			return s, true
		}
		return nil, true
	case "paginate":
		s, _ := raw.(string)
		normalized := strings.ToLower(s)
		if normalized == "hold" || normalized == "skip" {
			return normalized, true
		}
		return normalized == "true", true
	default:
		return nil, false
	}
}

// SpotKey reports whether key is a spot-prefixed directive name (an
// underscore-prefixed local directive, e.g. "_class") and returns the base
// local-directive name with the prefix stripped ("class"). ok is true only
// if key starts with "_" and has a non-empty remainder -- callers must still
// confirm the base name is a recognized local directive via CoerceLocal
// (mirrors directives/parse.js: "if (key.startsWith('_'))" then
// "if (directives.locals[spotKey])").
func SpotKey(key string) (base string, ok bool) {
	if len(key) > 1 && key[0] == '_' {
		return key[1:], true
	}
	return "", false
}
