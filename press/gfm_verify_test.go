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

// gfm_verify_test.go carries NO production code -- it is a pure regression
// suite over features chase/markdown.NewEngine (chase/markdown/seam.go)
// ALREADY bakes in: extension.GFM (tables, among other things) and
// ghtml.WithHardWraps() satisfy CORE-03's tables/hard-breaks;
// parser.WithAutoHeadingID() satisfies CORE-04's heading slugs. Nothing here
// wires anything new; these tests exist so a future NewEngine change can
// never silently drop tables, hard breaks, or heading-ID slugs without a red
// test catching it.
package press

import (
	"bytes"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

// TestGFMTableRenders covers TRD 03-03 Test-list case 4 (tables half): a
// standard GFM table fixture renders as an HTML <table> through the press
// engine (extension.GFM, already baked into chase/markdown.NewEngine -- no
// new wiring here).
func TestGFMTableRenders(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte("| a | b |\n|---|---|\n| 1 | 2 |\n"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, "<table>") {
		t.Errorf("GFM table fixture missing <table>; got %q", got)
	}
	if !strings.Contains(got, "<th>a</th>") || !strings.Contains(got, "<th>b</th>") {
		t.Errorf("GFM table fixture missing expected <th> header cells; got %q", got)
	}
	if !strings.Contains(got, "<td>1</td>") || !strings.Contains(got, "<td>2</td>") {
		t.Errorf("GFM table fixture missing expected <td> body cells; got %q", got)
	}
}

// TestHardWrapRendersBr covers TRD 03-03 Test-list case 4 (hard-breaks
// half): a two-line paragraph (a soft line break in the Markdown source)
// renders as <br> -- ghtml.WithHardWraps(), already baked into
// chase/markdown.NewEngine, promotes every soft break to a hard <br>.
func TestHardWrapRendersBr(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte("line one\nline two\n"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, "<br>") {
		t.Errorf("soft-broken paragraph missing <br> (WithHardWraps); got %q", got)
	}
}

// TestSlugHeadingID covers TRD 03-03 Test-list case 5 (h1 half): "# Hello
// World" renders an <h1> carrying a GitHub-compatible slug id attribute
// (id="hello-world") -- parser.WithAutoHeadingID(), already baked into
// chase/markdown.NewEngine.
func TestSlugHeadingID(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte("# Hello World\n"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, `<h1 id="hello-world">`) {
		t.Errorf(`"# Hello World" missing <h1 id="hello-world">; got %q`, got)
	}
}

// TestSlugH6 covers TRD 03-03 Test-list case 5 (h6 half): a level-6 heading
// ("######") also carries an auto-generated slug id -- WithAutoHeadingID
// applies uniformly across h1-h6, not just h1.
func TestSlugH6(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte("###### Deep\n"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, "<h6 id=\"deep\">") {
		t.Errorf(`"###### Deep" missing <h6 id="deep">; got %q`, got)
	}
}

// TestSlugDedup covers TRD 03-03 Test-list case 5 (dedup half): two headings
// sharing the same text get DIFFERENT, deduped slugs -- the second occurrence
// of "Hello" is suffixed "-1" (goldmark's parser.IDs.Generate: the first
// unused candidate wins, first collision appends "-1", next "-2", ...).
func TestSlugDedup(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte("# Hello\n\n# Hello\n"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, `<h1 id="hello">Hello</h1>`) {
		t.Errorf(`repeated-heading fixture missing first <h1 id="hello">Hello</h1>; got %q`, got)
	}
	if !strings.Contains(got, `<h1 id="hello-1">Hello</h1>`) {
		t.Errorf(`repeated-heading fixture missing deduped <h1 id="hello-1">Hello</h1>; got %q`, got)
	}
}
