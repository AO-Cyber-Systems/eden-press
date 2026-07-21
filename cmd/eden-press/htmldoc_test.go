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
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// fixtureOutput is a small, fixed press.Output used by every assembleHTML
// test -- pinned so the golden in TestAssembleHTMLZeroJSGolden stays
// byte-stable.
func fixtureOutput() press.Output {
	return press.Output{
		HTML: `<div class="marpit"><svg><foreignObject><section>hello</section></foreignObject></svg></div>`,
		CSS:  `.marpit{color:red}`,
	}
}

// TestAssembleHTMLZeroJSGolden is test-list case 1: assembleHTML's default
// (htmlDocOptions{}) output is a byte-stable, complete standalone document
// with NO <script> anywhere -- CLI-01's zero-JS-by-default requirement.
func TestAssembleHTMLZeroJSGolden(t *testing.T) {
	out := fixtureOutput()

	got := assembleHTML(out, htmlDocOptions{})

	want := "<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n" +
		"<style>\n" + out.CSS + "\n</style>\n</head>\n<body>\n" +
		out.HTML +
		"\n</body>\n</html>\n"

	if got != want {
		t.Errorf("assembleHTML golden mismatch:\n got:  %q\n want: %q", got, want)
	}

	if strings.Contains(got, "<script") {
		t.Errorf("default assembleHTML output contains <script>, want zero-JS: %q", got)
	}
	if !strings.Contains(got, "<!doctype html>") {
		t.Error("missing <!doctype html>")
	}
	if !strings.Contains(got, "<style>\n"+out.CSS) {
		t.Error("missing <style> wrapping out.CSS")
	}
	if !strings.Contains(got, "<body>\n"+out.HTML) {
		t.Error("missing out.HTML inside <body>")
	}
}

// TestAssembleHTMLAutoFitScript is test-list case 2: AutoFitScript:true
// splices exactly one <script> carrying press.BrowserFitJS(), placed AFTER
// out.HTML.
func TestAssembleHTMLAutoFitScript(t *testing.T) {
	out := fixtureOutput()

	got := assembleHTML(out, htmlDocOptions{AutoFitScript: true})

	if n := strings.Count(got, "<script"); n != 1 {
		t.Fatalf("<script> count = %d, want exactly 1: %q", n, got)
	}

	bodyIdx := strings.Index(got, out.HTML)
	scriptIdx := strings.Index(got, "<script")
	if bodyIdx == -1 {
		t.Fatal("out.HTML not found in assembled document")
	}
	if scriptIdx < bodyIdx {
		t.Errorf("<script> at %d appears BEFORE out.HTML at %d, want after", scriptIdx, bodyIdx)
	}

	if !strings.Contains(got, press.BrowserFitJS()) {
		t.Error("assembled document missing press.BrowserFitJS() body")
	}
}

// TestAssembleHTMLInjectScriptsSeam is test-list case 3: InjectScripts
// splices each script after out.HTML -- the seam watch/serve (04-06/07)
// reuse for the SSE reload client.
func TestAssembleHTMLInjectScriptsSeam(t *testing.T) {
	out := fixtureOutput()

	got := assembleHTML(out, htmlDocOptions{InjectScripts: []string{"/*reload*/"}})

	if !strings.Contains(got, "<script>\n/*reload*/\n</script>") {
		t.Errorf("assembled document missing injected script block: %q", got)
	}

	bodyIdx := strings.Index(got, out.HTML)
	scriptIdx := strings.Index(got, "/*reload*/")
	if bodyIdx == -1 || scriptIdx == -1 {
		t.Fatal("out.HTML or injected script not found in assembled document")
	}
	if scriptIdx < bodyIdx {
		t.Errorf("injected script at %d appears BEFORE out.HTML at %d, want after", scriptIdx, bodyIdx)
	}
}

// TestAssembleHTMLTitleEscaped proves opts.Title is rendered (and
// HTML-escaped) while out.HTML/out.CSS are passed through verbatim
// (trusted engine output, not re-escaped).
func TestAssembleHTMLTitleEscaped(t *testing.T) {
	out := fixtureOutput()

	got := assembleHTML(out, htmlDocOptions{Title: `<Deck & "Title">`})

	if !strings.Contains(got, "<title>&lt;Deck &amp; &#34;Title&#34;&gt;</title>") {
		t.Errorf("title not escaped as expected: %q", got)
	}
}

// TestAssembleHTMLNoTitleOmitsElement proves the zero-value Title omits the
// <title> element entirely rather than emitting an empty one.
func TestAssembleHTMLNoTitleOmitsElement(t *testing.T) {
	out := fixtureOutput()

	got := assembleHTML(out, htmlDocOptions{})

	if strings.Contains(got, "<title>") {
		t.Errorf("assembleHTML with no Title emitted a <title> element: %q", got)
	}
}
