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
	"regexp"
	"strings"
)

// hljsClassRemap maps chroma's Pygments-style short token classes (chroma's
// chroma.StandardTypes values -- e.g. "kd", "s2", "nv" -- emitted by
// chromahtml.WithClasses(true) in highlight.go) to the .hljs-* class names
// the three bundled Marp themes (themes/default.css, themes/gaia.css,
// themes/uncover.css -- TRD 03-02's compiled go:embed output) actually
// target.
//
// Every value here was ACQUIRED, not recalled from memory (research Open
// Question #3): TestRemapGrounded proves each one appears in
//
//	grep -ohE '\.hljs-[a-zA-Z_-]+' themes/*.css | sort -u
//
// -- the ground-truth set of .hljs-* selectors the acquired CSS actually
// defines (the acquired set, confirmed against themes.DefaultCSS/GaiaCSS/
// UncoverCSS): addition, attr, attribute, built_in, bullet, class, code,
// comment, deletion, doctag, emphasis, formula, keyword, link, literal,
// meta, name, number, operator, params, quote, regexp, section,
// selector-attr, selector-class, selector-id, selector-pseudo,
// selector-tag, string, strong, subst, symbol, tag, template-tag,
// template-variable, title, type, variable.
//
// Only chroma short codes that map onto that acquired set are listed here.
// Every chroma class NOT in this table -- "p" (Punctuation), "nx" (plain
// NameOther identifiers), "w" (whitespace), chroma's own structural wrapper
// classes ("chroma", "line", "cl", ...), and Generic/diff subtypes the
// bundled themes never style -- is intentionally left OUT: remapHLJS
// (below) leaves any class not present here exactly as chroma emitted it,
// rather than dropping the token's styling hook outright (research
// anti-pattern: never drop a class).
//
// chroma.StringInterpol ("si", embedded `${...}` expressions inside a
// string) maps to "hljs-subst" rather than "hljs-string": that is the same
// distinction real highlight.js draws between a string's own literal text
// and a substitution expression nested inside it, and "subst" is one of
// the acquired .hljs-* names (gaia.css's `.hljs-subst{...}` rule).
var hljsClassRemap = map[string]string{
	// Keywords.
	"k":  "hljs-keyword",
	"kc": "hljs-keyword",
	"kd": "hljs-keyword",
	"kn": "hljs-keyword",
	"kp": "hljs-keyword",
	"kr": "hljs-keyword",
	"kt": "hljs-type",

	// Names.
	"na": "hljs-attr",
	"nb": "hljs-built_in",
	"bp": "hljs-built_in",
	"nc": "hljs-title",
	"nd": "hljs-meta",
	"ne": "hljs-type",
	"nf": "hljs-title",
	"nn": "hljs-title",
	"no": "hljs-variable",
	"nt": "hljs-tag",
	"nv": "hljs-variable",
	"py": "hljs-attribute",
	"vc": "hljs-variable",
	"vg": "hljs-variable",
	"vi": "hljs-variable",
	"vm": "hljs-variable",

	// Literals / numbers.
	"l":  "hljs-literal",
	"ld": "hljs-literal",
	"m":  "hljs-number",
	"mb": "hljs-number",
	"mf": "hljs-number",
	"mh": "hljs-number",
	"mi": "hljs-number",
	"il": "hljs-number",
	"mo": "hljs-number",

	// Strings.
	"s":  "hljs-string",
	"sa": "hljs-string",
	"sb": "hljs-string",
	"sc": "hljs-string",
	"dl": "hljs-string",
	"sd": "hljs-string",
	"s2": "hljs-string",
	"se": "hljs-string",
	"sh": "hljs-string",
	"si": "hljs-subst",
	"sx": "hljs-string",
	"sr": "hljs-regexp",
	"s1": "hljs-string",
	"ss": "hljs-symbol",

	// Operators.
	"o":  "hljs-operator",
	"ow": "hljs-operator",
	"or": "hljs-operator",

	// Comments.
	"c":   "hljs-comment",
	"ch":  "hljs-comment",
	"cm":  "hljs-comment",
	"cp":  "hljs-comment",
	"cpf": "hljs-comment",
	"c1":  "hljs-comment",
	"cs":  "hljs-comment",

	// Generic (diff-style lexer output).
	"gd": "hljs-deletion",
	"gi": "hljs-addition",
	"gh": "hljs-section",
	"gu": "hljs-section",
	"ge": "hljs-emphasis",
	"gs": "hljs-strong",
	"gp": "hljs-meta",
}

// classAttrPattern matches an HTML class="..." attribute value -- the
// bounded surface remapHLJS rewrites. It captures the space-separated
// token list verbatim so a multi-class attribute (e.g. chroma's
// highlighted-line wrapper, class="line hl") is rewritten token-by-token
// rather than as one opaque unit.
var classAttrPattern = regexp.MustCompile(`class="([^"]*)"`)

// remapHLJS is CORE-07's one bespoke piece (research option (a)): a
// bounded post-format string pass over chromahtml's WithClasses(true)
// output (highlight.go) that rewrites every chroma short-code class token
// present in hljsClassRemap to its .hljs-* counterpart, leaving every other
// class token -- unmapped chroma codes and chroma's own structural wrapper
// classes ("chroma", "line", "cl", ...) alike -- untouched. It never
// re-parses the document: it is a pure string->string rewrite over
// already-rendered HTML, so the one-parse invariant (chase.Render/03-04
// "one-parse-two-sinks") is never disturbed -- this runs strictly AFTER
// rendering, as a second sink-side step, not a second parse.
func remapHLJS(html string) string {
	return classAttrPattern.ReplaceAllStringFunc(html, func(m string) string {
		sub := classAttrPattern.FindStringSubmatch(m)
		classes := strings.Fields(sub[1])
		for i, c := range classes {
			if mapped, ok := hljsClassRemap[c]; ok {
				classes[i] = mapped
			}
		}
		return `class="` + strings.Join(classes, " ") + `"`
	})
}
