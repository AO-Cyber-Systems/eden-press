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

package xlsxtyping

import (
	"encoding/json"
	"os"
	"testing"
)

// TestCorpusJSONIsCurrent is the whole anti-drift mechanism: cases.json is a
// GENERATED projection of Cases, so a committed file that no longer matches
// the Go source is a second source of truth, which is exactly what this
// package exists to prevent. Same idea as an OpenAPI codegen-verify job.
//
// If this fails, do NOT edit cases.json by hand:
//
//	go generate ./conformance/corpus/xlsxtyping
func TestCorpusJSONIsCurrent(t *testing.T) {
	want, err := CasesJSON()
	if err != nil {
		t.Fatalf("CasesJSON: %v", err)
	}
	got, err := os.ReadFile("cases.json")
	if err != nil {
		t.Fatalf("read cases.json (regenerate with `go generate ./conformance/corpus/xlsxtyping`): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("cases.json is stale: %d bytes committed, %d bytes generated.\n"+
			"Regenerate with `go generate ./conformance/corpus/xlsxtyping` -- never hand-edit it.\n"+
			"--- committed ---\n%s\n--- generated ---\n%s",
			len(got), len(want), got, want)
	}
}

// TestCasesJSONIsDeterministic guards the mechanism itself: if CasesJSON were
// non-deterministic (map iteration, unstable order, variable indent), the
// currency test above would flap and the first response would be to relax it.
func TestCasesJSONIsDeterministic(t *testing.T) {
	a, err := CasesJSON()
	if err != nil {
		t.Fatalf("CasesJSON #1: %v", err)
	}
	for i := 0; i < 5; i++ {
		b, err := CasesJSON()
		if err != nil {
			t.Fatalf("CasesJSON #%d: %v", i+2, err)
		}
		if string(a) != string(b) {
			t.Fatalf("CasesJSON is not deterministic on call %d", i+2)
		}
	}
}

// TestCasesJSONRoundTrips: the projection a non-Go suite reads must decode
// back into the same cases, or the Dart side is asserting against something
// other than what Go asserts against.
func TestCasesJSONRoundTrips(t *testing.T) {
	b, err := CasesJSON()
	if err != nil {
		t.Fatalf("CasesJSON: %v", err)
	}
	var back Corpus
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Schema != SchemaID {
		t.Errorf("schema = %q, want %q", back.Schema, SchemaID)
	}
	if len(back.Cells) != len(Cases) {
		t.Fatalf("cells = %d, want %d", len(back.Cells), len(Cases))
	}
	for i, c := range back.Cells {
		if c != Cases[i] {
			t.Errorf("cell %d round-tripped as %+v, want %+v", i, c, Cases[i])
		}
	}
	if len(back.Structural) != len(StructuralCases) {
		t.Fatalf("structural = %d, want %d", len(back.Structural), len(StructuralCases))
	}
	for i, c := range back.Structural {
		if c != StructuralCases[i] {
			t.Errorf("structural %d round-tripped as %+v, want %+v", i, c, StructuralCases[i])
		}
	}
}

// TestCasesAreWellFormed: every case must carry a known class and no input may
// appear twice, since two entries for one string would let the two writers
// each pick the answer they already implement.
func TestCasesAreWellFormed(t *testing.T) {
	seen := make(map[string]int, len(Cases))
	for i, c := range Cases {
		switch c.Class {
		case ClassNumber, ClassText, ClassEmpty:
		default:
			t.Errorf("case %d (%q) has unknown class %q", i, c.Input, c.Class)
		}
		if j, dup := seen[c.Input]; dup {
			t.Errorf("input %q appears at both %d and %d; one string, one answer", c.Input, j, i)
		}
		seen[c.Input] = i
	}
	ids := make(map[string]bool, len(StructuralCases))
	for i, s := range StructuralCases {
		if s.ID == "" {
			t.Errorf("structural case %d has no ID", i)
		}
		if ids[s.ID] {
			t.Errorf("duplicate structural case ID %q", s.ID)
		}
		ids[s.ID] = true
		if s.Expect == "" {
			t.Errorf("structural case %q has no expected outcome", s.ID)
		}
		if s.Reason == "" {
			t.Errorf("structural case %q has no reason; a shape agreement nobody can justify gets deleted by the next reader", s.ID)
		}
	}
}

// TestHeadlineCasesArePresent pins the three cases this corpus was created
// for. Deleting one would make the suite green while the contract it encodes
// quietly disappeared.
func TestHeadlineCasesArePresent(t *testing.T) {
	want := map[string]Class{
		"007":   ClassText,   // the leading-zero corruption
		"+3":    ClassText,   // the case the two suites pinned in opposite directions
		"0.4":   ClassNumber, // the guard that must not be over-applied
		"0":     ClassNumber, // the length > 1 boundary
		"NaN":   ClassText,
		"1e3":   ClassNumber,
		"p95":   ClassText,
		"1,200": ClassText,
	}
	have := make(map[string]Class, len(Cases))
	for _, c := range Cases {
		have[c.Input] = c.Class
	}
	for in, class := range want {
		got, ok := have[in]
		if !ok {
			t.Errorf("case %q is missing from the corpus", in)
			continue
		}
		if got != class {
			t.Errorf("case %q classifies as %q, want %q", in, got, class)
		}
	}
}
