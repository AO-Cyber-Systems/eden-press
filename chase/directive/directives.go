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
func CoerceGlobal(key string, raw RawValue, themeExists ThemeExists) (value any, isKnown bool) {
	return nil, false
}

// CoerceLocal normalizes a single local directive's raw value, mirroring
// directives.js's locals{} table. Same (value, isKnown) contract as
// CoerceGlobal.
func CoerceLocal(key string, raw RawValue) (value any, isKnown bool) {
	return nil, false
}

// SpotKey reports whether key is a spot-prefixed directive name (an
// underscore-prefixed local directive, e.g. "_class") and returns the base
// local-directive name with the prefix stripped ("class"). ok is true only
// if key starts with "_" and has a non-empty remainder -- callers must still
// confirm the base name is a recognized local directive via CoerceLocal
// (mirrors directives/parse.js: "if (key.startsWith('_'))" then
// "if (directives.locals[spotKey])").
func SpotKey(key string) (base string, ok bool) {
	return "", false
}
