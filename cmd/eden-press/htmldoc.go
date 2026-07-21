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

package main

import (
	"html"
	"strings"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// htmlDocOptions configures assembleHTML's standalone document wrapper.
// Every field defaults to the zero-JS baseline: an unset htmlDocOptions{}
// produces NO <script> anywhere in the document (CLI-01's literal "zero-JS
// static HTML"), matching Marp's empty `bare` block script.
type htmlDocOptions struct {
	// AutoFitScript splices press.BrowserFitJS() into the document, AFTER
	// out.HTML, when true. This is the CLI's own --auto-fit-script opt-in
	// (registered as a persistent flag in flags.go); it is never fed back
	// through press.Render, whose sanitize pass strips <script>
	// unconditionally (press/press_test.go asserts this).
	AutoFitScript bool

	// InjectScripts is the SEAM every OTHER viewer-side script reuses: each
	// entry is spliced, verbatim, into its own <script> block AFTER
	// out.HTML and after any AutoFitScript block. watch (04-06) and serve
	// (04-07) pass the SSE reload client through this same seam rather
	// than editing this file -- the zero-JS-vs-reload distinction lives in
	// ONE place (assembleHTML), not duplicated per mode.
	InjectScripts []string

	// Title sets the document's <title>. "" (the zero value) omits the
	// <title> element entirely; a caller that wants one derives it from
	// out.Meta (e.g. a `title` front-matter directive) before calling
	// assembleHTML -- assembleHTML itself never inspects out.Meta.
	Title string
}

// assembleHTML wraps a press.Output into a COMPLETE, standalone HTML
// document: <!doctype html> -> <head> (charset+viewport meta, optional
// <title>, <style>{out.CSS}</style>) -> <body>{out.HTML}</body>. By default
// (htmlDocOptions{}) the result carries NO <script> tag anywhere --
// CLI-01's zero-JS-by-default requirement.
//
// out.HTML is already sanitized by press.Render (its sanitize pass strips
// <script> unconditionally); assembleHTML does NOT re-sanitize it, and does
// NOT escape it -- it is trusted engine output, not user input reaching
// this function directly. The same is true of out.CSS. Only opts.Title
// (an operator-supplied string, not deck content) is HTML-escaped.
//
// Any requested script (opts.AutoFitScript / opts.InjectScripts) is
// spliced AFTER out.HTML -- outside any sanitize path -- so it is never at
// risk of being stripped by press.Render's policy, and never re-enters it
// either.
func assembleHTML(out press.Output, opts htmlDocOptions) string {
	var b strings.Builder

	b.WriteString("<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	if opts.Title != "" {
		b.WriteString("<title>" + html.EscapeString(opts.Title) + "</title>\n")
	}
	b.WriteString("<style>\n" + out.CSS + "\n</style>\n</head>\n<body>\n")
	b.WriteString(out.HTML)

	// Scripts go AFTER out.HTML only, and ONLY when requested -- these two
	// branches are the SOLE <script> emitters in this function. Default
	// (both zero) emits zero <script> tags: the zero-JS baseline.
	if opts.AutoFitScript {
		b.WriteString("\n<script>\n" + press.BrowserFitJS() + "\n</script>\n")
	}
	for _, s := range opts.InjectScripts {
		b.WriteString("\n<script>\n" + s + "\n</script>\n")
	}

	b.WriteString("\n</body>\n</html>\n")
	return b.String()
}
