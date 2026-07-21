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
	"testing"

	"github.com/microcosm-cc/bluemonday"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// TestOptionsZeroValueIsMarpDefault covers TRD 03-01 Test-list case 4: every
// field of press.Options{} sits at the zero value that MEANS "Marp Core's own
// default behavior", so press.Render(md, press.Options{}) just works. This
// table both documents each field's zero-value contract and fails loudly if a
// default is ever silently changed.
func TestOptionsZeroValueIsMarpDefault(t *testing.T) {
	var o Options

	// String fields: "" is the sentinel meaning "resolve the Marp default".
	if o.Theme != "" {
		t.Errorf(`Options{}.Theme = %q, want "" (front-matter theme: directive -> "default")`, o.Theme)
	}
	if o.Profile != "" {
		t.Errorf(`Options{}.Profile = %q, want "" (profile.Default(), today "slides")`, o.Profile)
	}
	if o.MathMode != "" {
		t.Errorf(`Options{}.MathMode = %q, want "" (-> "mathml")`, o.MathMode)
	}
	if o.HighlightStyle != "" {
		t.Errorf(`Options{}.HighlightStyle = %q, want "" (default chroma style)`, o.HighlightStyle)
	}

	// InlineSVG: baseline-on is applied in 03-09; the zero value of the flag
	// itself is false and documented as such (no speculative inversion).
	if o.InlineSVG != false {
		t.Errorf("Options{}.InlineSVG = %v, want false", o.InlineSVG)
	}

	// NoHighlight is deliberately INVERTED so the zero value is the safe,
	// Marp-matching default: false => highlighting ON.
	if o.NoHighlight != false {
		t.Errorf("Options{}.NoHighlight = %v, want false (zero value => highlighting ON)", o.NoHighlight)
	}

	// Sanitize is a *bluemonday.Policy: nil => the built-in always-on policy
	// (CORE-05 is NOT optional). nil does NOT mean "no sanitize".
	if o.Sanitize != nil {
		t.Errorf("Options{}.Sanitize = %v, want nil (built-in always-on policy applies)", o.Sanitize)
	}
}

// TestOutputZeroValueFields covers TRD 03-01 Test-list case 5: press.Output
// carries HTML/CSS (string), Model (*model.Document), Meta (model.Meta), and
// Comments ([]string), and the zero value of every field is its type's zero.
func TestOutputZeroValueFields(t *testing.T) {
	var out Output

	if out.HTML != "" {
		t.Errorf(`Output{}.HTML = %q, want ""`, out.HTML)
	}
	if out.CSS != "" {
		t.Errorf(`Output{}.CSS = %q, want ""`, out.CSS)
	}
	if out.Model != nil {
		t.Errorf("Output{}.Model = %v, want nil (*model.Document)", out.Model)
	}
	if out.Comments != nil {
		t.Errorf("Output{}.Comments = %v, want nil ([]string)", out.Comments)
	}

	// Comments must be exactly []string (the flattened model.Section.Notes
	// contract Objective 7's Dart binding serializes). Assigning a []string
	// compile-fences the element type.
	out.Comments = []string{"a note"}
	if got := len(out.Comments); got != 1 {
		t.Errorf("Output.Comments length = %d, want 1", got)
	}
	if out.Comments[0] != "a note" {
		t.Errorf("Output.Comments[0] = %q, want %q", out.Comments[0], "a note")
	}
}

// TestOptionsOutputCompileFence locks the API-03 surface: it constructs both
// structs with EVERY field named as a keyed composite literal, so a later
// rename or removal of any field fails the build (protecting the stability
// contract Objective 7's Dart binding depends on). It also proves each field's
// declared type by assigning a value of that type.
func TestOptionsOutputCompileFence(t *testing.T) {
	// Every Options field named + typed. bluemonday.UGCPolicy() proves the
	// Sanitize field's *bluemonday.Policy type (nil is its documented default,
	// but the field must accept a real policy too).
	opt := Options{
		Theme:          "custom",
		Profile:        "slides",
		InlineSVG:      true,
		MathMode:       "mathml",
		NoHighlight:    true,
		HighlightStyle: "github",
		Sanitize:       bluemonday.UGCPolicy(),
	}
	if opt.Theme != "custom" || opt.Profile != "slides" || !opt.InlineSVG ||
		opt.MathMode != "mathml" || !opt.NoHighlight || opt.HighlightStyle != "github" ||
		opt.Sanitize == nil {
		t.Fatalf("Options keyed-literal fence did not round-trip: %#v", opt)
	}

	// Every Output field named + typed.
	out := Output{
		HTML:     "<section></section>",
		CSS:      "section{}",
		Model:    &model.Document{SchemaVersion: model.SchemaVersion},
		Meta:     model.Meta{Directives: map[string]string{"theme": "default"}},
		Comments: []string{"note one", "note two"},
	}
	if out.HTML == "" || out.CSS == "" || out.Model == nil ||
		out.Meta.Directives["theme"] != "default" || len(out.Comments) != 2 {
		t.Fatalf("Output keyed-literal fence did not round-trip: %#v", out)
	}
}
