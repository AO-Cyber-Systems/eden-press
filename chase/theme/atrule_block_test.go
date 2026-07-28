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

package theme

import (
	"strings"
	"testing"
)

// Block at-rule contents were never modeled: Parse recorded only the OPENING
// of an @media/@supports block into Stylesheet.Atoms and dropped everything
// inside it. Stylesheet.String then emitted the recorded opening followed by a
// semicolon, so a theme authored with
//
//	@media print { section { color: red } }
//
// packed to the literal, invalid text `@media print;` with its body gone.
//
// That is not hypothetical: themes/uncover.css ships an `@media print` block,
// so every packed uncover stylesheet has carried invalid CSS and lost its
// print-specific pagination styling. The gap survived because profiles/slides'
// own scaffold contains zero at-rules, so nothing exercised the path.
//
// Block at-rules are now modeled as a Rule carrying At, held in Rules (NOT
// Atoms) so their authored cascade position is preserved -- hoisting an
// @media override above the rules it overrides would silently invert it.

func TestParseBlockAtRuleKeepsContents(t *testing.T) {
	sheet, err := Parse("@media print { section { color: red; } }")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(sheet.Atoms) != 0 {
		t.Errorf("block at-rule leaked into Atoms: %+v", sheet.Atoms)
	}
	if len(sheet.Rules) != 1 {
		t.Fatalf("Rules = %d, want 1 (the @media block)", len(sheet.Rules))
	}
	r := sheet.Rules[0]
	if r.At == nil {
		t.Fatal("Rule.At is nil; the rule is not marked as an at-rule block")
	}
	if r.At.Name != "media" {
		t.Errorf("At.Name = %q, want media", r.At.Name)
	}
	if !strings.Contains(r.At.Prelude, "print") {
		t.Errorf("At.Prelude = %q, want it to contain print", r.At.Prelude)
	}
	if len(r.Children) != 1 {
		t.Fatalf("at-rule Children = %d, want 1", len(r.Children))
	}
	if got := len(r.Children[0].Declarations); got != 1 {
		t.Errorf("nested rule declarations = %d, want 1", got)
	}
}

func TestStatementAtRuleStillInAtoms(t *testing.T) {
	// pass_import.go consumes Atoms for @import / @import-theme resolution,
	// so statement at-rules must keep landing there.
	sheet, err := Parse(`@import "x.css";` + "\nsection { color: red; }")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(sheet.Atoms) != 1 {
		t.Fatalf("Atoms = %d, want 1 (@import)", len(sheet.Atoms))
	}
	if sheet.Atoms[0].Name != "import" {
		t.Errorf("Atoms[0].Name = %q, want import", sheet.Atoms[0].Name)
	}
}

func TestBlockAtRuleRoundTrips(t *testing.T) {
	sheet, err := Parse("@media print { section { color: red; } }")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := sheet.String()
	if !strings.Contains(got, "@media print {") {
		t.Errorf("rendered CSS lost the at-rule block opening: %q", got)
	}
	if !strings.Contains(got, "color: red") {
		t.Errorf("rendered CSS lost the at-rule body: %q", got)
	}
	if strings.Contains(got, "@media print;") {
		t.Errorf("rendered CSS still emits the invalid statement form: %q", got)
	}
}

func TestBlockAtRuleCascadePositionPreserved(t *testing.T) {
	// The @media block is authored BETWEEN two rules and must stay there.
	// Emitting it first (the old Atoms behavior) would invert an override.
	sheet, err := Parse(`a { color: red; }
@media print { b { color: green; } }
c { color: blue; }`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := sheet.String()
	ia := strings.Index(got, "a {")
	im := strings.Index(got, "@media")
	ic := strings.Index(got, "c {")
	if ia < 0 || im < 0 || ic < 0 {
		t.Fatalf("missing expected rules in output: %q", got)
	}
	if !(ia < im && im < ic) {
		t.Errorf("cascade order not preserved (want a < @media < c): %q", got)
	}
}

func TestNestedAtRuleSurvives(t *testing.T) {
	// uncover.css's real shape: @supports nested inside @media print.
	sheet, err := Parse(`@media print { @supports (x:y) { section:after { fill: red; } } }`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := sheet.String()
	for _, want := range []string{"@media print {", "@supports", "fill: red"} {
		if !strings.Contains(got, want) {
			t.Errorf("nested at-rule output missing %q: %s", want, got)
		}
	}
}

func TestEmptyBlockAtRuleDropped(t *testing.T) {
	// An at-rule block with nothing in it is noise; emitting `@media print {}`
	// is harmless but pointless, and dropping it keeps packed output tidy.
	sheet, err := Parse("@media print { }\nsection { color: red; }")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if strings.Contains(sheet.String(), "@media") {
		t.Errorf("empty at-rule block should be dropped: %q", sheet.String())
	}
}
