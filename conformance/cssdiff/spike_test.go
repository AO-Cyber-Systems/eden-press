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

package cssdiff

import (
	"reflect"
	"testing"
)

// --- build -> model (happy) ---------------------------------------------

func TestParse_SingleRule_SingleDeclaration(t *testing.T) {
	sheet, err := Parse(`a { color: red; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := Stylesheet{
		Rules: []Rule{
			{
				Selector: "a",
				Declarations: []Declaration{
					{Property: "color", Value: "red"},
				},
			},
		},
	}
	if !reflect.DeepEqual(sheet, want) {
		t.Fatalf("Parse() = %#v, want %#v", sheet, want)
	}
}

func TestParse_MultipleDeclarations_PreserveOrder(t *testing.T) {
	sheet, err := Parse(`a { color: red; margin: 0; padding: 1px; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(sheet.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(sheet.Rules))
	}
	want := []Declaration{
		{Property: "color", Value: "red"},
		{Property: "margin", Value: "0"},
		{Property: "padding", Value: "1px"},
	}
	if !reflect.DeepEqual(sheet.Rules[0].Declarations, want) {
		t.Fatalf("Declarations = %#v, want %#v (source order must be preserved)", sheet.Rules[0].Declarations, want)
	}
}

func TestParse_Important_CapturedAndStripped(t *testing.T) {
	sheet, err := Parse(`a { color: red !important; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := Declaration{Property: "color", Value: "red", Important: true}
	if len(sheet.Rules) != 1 || len(sheet.Rules[0].Declarations) != 1 {
		t.Fatalf("unexpected shape: %#v", sheet)
	}
	got := sheet.Rules[0].Declarations[0]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Declaration = %#v, want %#v", got, want)
	}
}

func TestParse_MultipleRules_PreserveOrder(t *testing.T) {
	sheet, err := Parse(`a { color: red; } b { color: blue; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := Stylesheet{
		Rules: []Rule{
			{Selector: "a", Declarations: []Declaration{{Property: "color", Value: "red"}}},
			{Selector: "b", Declarations: []Declaration{{Property: "color", Value: "blue"}}},
		},
	}
	if !reflect.DeepEqual(sheet, want) {
		t.Fatalf("Parse() = %#v, want %#v (rule order must be preserved)", sheet, want)
	}
}

// --- within-node normalization (edge) ------------------------------------

func TestParse_NormalizeHexColor(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want string
	}{
		{"three-digit", `a { color: #FFF; }`, "#fff"},
		{"six-digit", `a { color: #FFFFFF; }`, "#ffffff"},
		{"mixed-case", `a { color: #AbC123; }`, "#abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet, err := Parse(tt.css)
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if len(sheet.Rules) != 1 || len(sheet.Rules[0].Declarations) != 1 {
				t.Fatalf("unexpected shape: %#v", sheet)
			}
			got := sheet.Rules[0].Declarations[0].Value
			if got != tt.want {
				t.Fatalf("Value = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse_CollapseWhitespace(t *testing.T) {
	sheet, err := Parse("a   b { color :   red  ; }")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := Stylesheet{
		Rules: []Rule{
			{
				Selector:     "a b",
				Declarations: []Declaration{{Property: "color", Value: "red"}},
			},
		},
	}
	if !reflect.DeepEqual(sheet, want) {
		t.Fatalf("Parse() = %#v, want %#v", sheet, want)
	}
}

func TestParse_StripComments(t *testing.T) {
	sheet, err := Parse(`
/* leading top-level comment */
a /* mid-selector */ {
  /* between rules */
  color: red; /* trailing */
}
`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	want := Stylesheet{
		Rules: []Rule{
			{
				Selector:     "a",
				Declarations: []Declaration{{Property: "color", Value: "red"}},
			},
		},
	}
	if !reflect.DeepEqual(sheet, want) {
		t.Fatalf("Parse() = %#v, want %#v (comments must be fully stripped)", sheet, want)
	}
}

func TestParse_NormalizeQuoteStyle(t *testing.T) {
	double, err := Parse(`a { content: "x"; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	single, err := Parse(`a { content: 'x'; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !reflect.DeepEqual(double, single) {
		t.Fatalf("single- and double-quoted equivalents did not normalize identically: %#v vs %#v", double, single)
	}
	if len(double.Rules) != 1 || len(double.Rules[0].Declarations) != 1 {
		t.Fatalf("unexpected shape: %#v", double)
	}
	want := `"x"`
	got := double.Rules[0].Declarations[0].Value
	if got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
}

// --- detectability (spike EXIT CRITERION) --------------------------------

// TestDetectability_IdenticalCSS_DeepEqual is the positive control for the
// two negative assertions below: it proves reflect.DeepEqual actually
// considers two independently-parsed-but-identical Stylesheets equal, so
// the NOT-equal assertions that follow are meaningful rather than
// incidental (e.g. because of a stray unexported field or pointer).
func TestDetectability_IdenticalCSS_DeepEqual(t *testing.T) {
	const src = `a { color: red; border: 1px solid #FFF; }`
	first, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	second, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical CSS parsed twice produced different models: %#v vs %#v", first, second)
	}
}

// TestDetectability_ChangedValue_NotDeepEqual is the spike's primary exit
// criterion: a single changed declaration value must be visible at the
// model level.
func TestDetectability_ChangedValue_NotDeepEqual(t *testing.T) {
	original, err := Parse(`a { color: red; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	mutated, err := Parse(`a { color: blue; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if reflect.DeepEqual(original, mutated) {
		t.Fatalf("mutated declaration value was NOT detected: both models equal %#v", original)
	}
}

// TestDetectability_ReorderedDeclarations_NotDeepEqual is the spike's
// second exit criterion: reordering two same-property declarations must
// also be visible at the model level, because CSS cascade order is
// significant. If this test fails, the model wrongly discarded order (do
// not relax the test — fix build.go).
func TestDetectability_ReorderedDeclarations_NotDeepEqual(t *testing.T) {
	original, err := Parse(`a { color: red; color: blue; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	reordered, err := Parse(`a { color: blue; color: red; }`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if reflect.DeepEqual(original, reordered) {
		t.Fatalf("reordered same-property declarations were NOT detected: both models equal %#v", original)
	}
}
