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
	"bytes"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

// TestStrikethroughOverrideRendersS covers TRD 03-03 Test-list case 1:
// markdown.NewEngine(strikethroughOption()) renders "~~gone~~" as
// "<s>gone</s>", NOT goldmark GFM's default "<del>gone</del>". This is the
// one genuinely new piece of CORE-03: a custom renderer.NodeRenderer
// registering extast.KindStrikethrough at a priority BELOW 500 (goldmark's
// own StrikethroughHTMLRenderer registers at priority 500 -- source-verified
// against goldmark@v1.8.4's extension/strikethrough.go), so the LAST
// RegisterFuncs call for that NodeKind wins (last-write-wins by ascending
// priority, reverse-iterated).
func TestStrikethroughOverrideRendersS(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte("~~gone~~"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, "<s>gone</s>") {
		t.Errorf("strikethroughOption() output missing <s>gone</s>; got %q", got)
	}
	if strings.Contains(got, "<del>") {
		t.Errorf("strikethroughOption() output still contains goldmark GFM's default <del>; got %q", got)
	}
}

// TestStrikethroughDefaultIsDel covers TRD 03-03 Test-list case 2: the
// priority-override proof. WITHOUT strikethroughOption(), the same chase
// engine (markdown.NewEngine()) renders "~~gone~~" as goldmark GFM's default
// "<del>gone</del>" -- documenting the baseline strikethroughOption() flips,
// and confirming the option itself (not some ambient engine behavior) is
// what changes the output in the sibling test above.
func TestStrikethroughDefaultIsDel(t *testing.T) {
	engine := markdown.NewEngine()

	var buf bytes.Buffer
	if err := engine.Convert([]byte("~~gone~~"), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, "<del>gone</del>") {
		t.Errorf("default chase engine (no strikethroughOption) = %q, want it to contain <del>gone</del> (documents the overridden baseline)", got)
	}
	if strings.Contains(got, "<s>gone</s>") {
		t.Errorf("default chase engine unexpectedly rendered <s>gone</s> without strikethroughOption(); got %q", got)
	}
}

// mixedGFMFixture is a hand-built fixture mixing a GFM table, a strikethrough
// span, and a soft-broken (two-line) paragraph -- Test-list case 3's proof
// that the <s> override does not disturb any other GFM output.
const mixedGFMFixture = "| a | b |\n|---|---|\n| 1 | 2 |\n\nline one\nline two\n\n~~gone~~\n"

// TestStrikethroughDoesNotDisturbOtherGFM covers TRD 03-03 Test-list case 3:
// a fixture mixing a table, a strikethrough span, and a soft-broken paragraph
// renders the table and the <br> (WithHardWraps, baked into NewEngine)
// normally, AND the <s> override -- the strikethroughOption() renderer
// registration composes cleanly alongside chase/markdown's own GFM table and
// hard-wrap rendering, with no collision.
func TestStrikethroughDoesNotDisturbOtherGFM(t *testing.T) {
	engine := markdown.NewEngine(strikethroughOption())

	var buf bytes.Buffer
	if err := engine.Convert([]byte(mixedGFMFixture), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	got := buf.String()

	if !strings.Contains(got, "<table>") {
		t.Errorf("mixed fixture missing <table>; got %q", got)
	}
	if !strings.Contains(got, "<br>") {
		t.Errorf("mixed fixture missing <br> (soft break, WithHardWraps); got %q", got)
	}
	if !strings.Contains(got, "<s>gone</s>") {
		t.Errorf("mixed fixture missing <s>gone</s>; got %q", got)
	}
	if strings.Contains(got, "<del>") {
		t.Errorf("mixed fixture unexpectedly contains <del>; got %q", got)
	}
}
