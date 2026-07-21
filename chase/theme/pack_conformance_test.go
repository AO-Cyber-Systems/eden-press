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

// pack_conformance_test.go is 01-08's own (Objective-1 integration TRD)
// formal attestation of the theme CSS-AST diff gate (Test-list case 5:
// "pass the theme CSS-AST diff gate via a NEW
// chase/theme/pack_conformance_test.go"). 01-04's pack_test.go already
// implements this exact gate for the "stress" and "scaffold" themes
// (TestPackFullPipelineStressThemeMatchesFixtureViaCSSDiff /
// ...ScaffoldThemeName..., committed under 62cd9e8) against
// expectedStressPackedCSS / expectedScaffoldPackedCSS -- both reused
// verbatim here (same package, no import, no redeclaration) rather than
// re-authoring a second copy of the same fixture, per the TRD's own
// instruction to reuse the 01-04 fixtures. This file's distinctly-named
// test functions are 01-08's own citable acceptance-gate record for the
// Objective-1 integration milestone, not a duplicate of 01-04's unit tests.

import (
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/conformance/cssdiff"
)

// TestObjective1ThemeCSSDiffGateStress is 01-08's acceptance-gate
// attestation for the "stress" synthetic theme: Pack("stress",
// PackOptions{InlineSVG: true}) must equal expectedStressPackedCSS
// (chase/theme/pack_test.go) via cssdiff.Equal's format-insensitive,
// order-sensitive CSS-AST diff.
func TestObjective1ThemeCSSDiffGateStress(t *testing.T) {
	th, err := Load(stressThemeCSS, "section", testSizeFallback)
	if err != nil {
		t.Fatalf("Load(stressThemeCSS): %v", err)
	}
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)
	ts.Add(th)

	out, err := ts.Pack("stress", PackOptions{InlineSVG: true})
	if err != nil {
		t.Fatalf("Pack(stress): %v", err)
	}

	if equal, diff := cssdiff.Equal(expectedStressPackedCSS, out); !equal {
		t.Fatalf("01-08 acceptance gate: Pack(stress) != expected fixture via cssdiff.Equal:\n%s", diff)
	}
}

// TestObjective1ThemeCSSDiffGateScaffold is 01-08's acceptance-gate
// attestation for the built-in Marpit-scaffold theme: Pack(ScaffoldThemeName,
// PackOptions{InlineSVG: false}) must equal expectedScaffoldPackedCSS
// (chase/theme/pack_test.go) via cssdiff.Equal.
func TestObjective1ThemeCSSDiffGateScaffold(t *testing.T) {
	ts := NewThemeSet("section", testScaffoldCSS, testAdvancedBackgroundCSS)

	out, err := ts.Pack(ScaffoldThemeName, PackOptions{InlineSVG: false})
	if err != nil {
		t.Fatalf("Pack(scaffold): %v", err)
	}

	if equal, diff := cssdiff.Equal(expectedScaffoldPackedCSS, out); !equal {
		t.Fatalf("01-08 acceptance gate: Pack(scaffold) != expected fixture via cssdiff.Equal:\n%s", diff)
	}
}
