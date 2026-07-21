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

// Package sanitize (this file) hand-filters the raw-HTML tags CommonMark's
// GFM extension deliberately does NOT neutralize.
package sanitize

// GFMDisallowedTags lists the nine raw-HTML tags CommonMark's GFM
// "disallowed raw HTML" extension names, which cmark-gfm's own tagfilter
// extension ESCAPES (renders as visible encoded text) rather than strips.
// goldmark's GFM extension -- this project's markdown engine -- provides NO
// automatic protection for these tags at all (research Pitfall 12): there is
// no goldmark-side tagfilter equivalent. Policy()'s blank-slate bluemonday
// allow-list is therefore the ONLY layer that neutralizes them.
//
// bluemonday's own hardcoded skip-content set already fully strips six of
// these nine outright (script, iframe, style, title, noembed, noframes); the
// remaining three (textarea, xmp, plaintext) are NOT in that hardcoded set,
// so left alone they would have their tag removed but their INNER TEXT
// CONTENT preserved (bluemonday's ordinary default for any non-allow-listed
// element). Policy() calls p.SkipElementsContent(GFMDisallowedTags...) with
// this full list so all nine are stripped uniformly and explicitly, rather
// than relying on a partial, easily-forgotten hardcoded overlap.
//
// DELIBERATE DEVIATION (documented, tested -- see TestStripVsEscape): this
// policy STRIPS these tags and their content. Marp's JS `xss` library
// instead ESCAPES disallowed tags. CORE-05's bar is behavioral XSS
// neutralization, not byte-parity with the JS library's strip-vs-escape
// choice.
var GFMDisallowedTags = []string{
	"script",
	"iframe",
	"style",
	"textarea",
	"title",
	"xmp",
	"noembed",
	"noframes",
	"plaintext",
}
