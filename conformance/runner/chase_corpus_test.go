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

package runner

// chase_corpus_test.go is a NEW, separate acceptance-gate test -- it does
// NOT edit corpus_test.go's TestMarpCorpus, which Objective 3 legitimately
// flips from PENDING once its own batteries land (01-08-TRD.md
// anti_patterns).
//
// It drives the FULL Objective-0 Marp golden corpus through the chase
// engine (chase/markdown.RenderFunc) and categorizes every marp-core case:
//
//   - PASS:    htmldiff.Equal(expected.html, chase output) is true.
//   - BLOCKED: the case is in chaseSkipMap, keyed to the Objective-3
//     requirement that will close it -- NOT counted as a failure.
//   - FAIL:    an un-mapped htmldiff mismatch -- a genuine Objective-1
//     defect (01-08-TRD.md error_recovery: "fix it, do not hide it in the
//     skip-map").

import (
	"path/filepath"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus"
	"github.com/AO-Cyber-Systems/eden-press/conformance/report"
)

// chaseSkipMap maps a corpus case ID still blocked on an Objective-3
// "battery" to the requirement that will close it (01-08-TRD.md
// anti_patterns: "heading-slug parity is CORE-04, emoji CORE-06, code
// CORE-07, math CORE-08, strikethrough CORE-03, fit CORE-09, theme-gaia/
// uncover CORE-01 embedded themes, size directive CORE-02").
var chaseSkipMap = map[string]string{
	"marp-emoji":          "CORE-06",
	"marp-code-highlight": "CORE-07",
	"marp-math":           "CORE-08",
	"marp-strikethrough":  "CORE-03",
	"marp-fit-heading":    "CORE-09",
	"marp-theme-gaia":     "CORE-01",
	"marp-theme-uncover":  "CORE-01",
	"marp-size-4-3":       "CORE-02",
}

// marpitMechanicCases is 01-08's own PASS-required set (Test-list case 2):
// the 9 corpus cases exercising ONLY Objective-1 mechanics (slide-split,
// directive carry-forward, backgrounds/inline-SVG) -- none needs an
// Objective-3 "battery".
//
// marp-gfm-table is deliberately NOT in this set (it is not a Marpit
// mechanic -- it exercises goldmark's stock GFM table extension, wired via
// extension.GFM in seam.go's NewEngine) nor in chaseSkipMap. Its fate was
// decided empirically: rendering it through the chase engine and
// htmldiff.Equal-ing against expected.html PASSES outright -- no Objective-3
// battery is needed, since table rendering has zero dependency on marp-core.
// It is asserted explicitly below (not left to fall out of "zero failed")
// so a future regression here is reported by name, not just as an
// unexplained corpus failure. See 01-08-SUMMARY.md for this decision
// record.
var marpitMechanicCases = map[string]bool{
	"marp-basic":           true,
	"marp-slide-split":     true,
	"marp-class-spot":      true,
	"marp-heading-divider": true,
	"marp-paginate":        true,
	"marp-header-footer":   true,
	"marp-bg-color":        true,
	"marp-bg-image":        true,
	"marp-bg-split":        true,
}

// TestChaseCorpus is the wave-5 corpus acceptance gate: it asserts every
// Marpit-mechanic case PASSes, every skip-mapped case is BLOCKED (not
// failed), and ZERO cases fail for an unexplained (Objective-1) reason.
func TestChaseCorpus(t *testing.T) {
	root := filepath.Join("..", "corpus", "cases")
	cases, err := corpus.LoadCases(root)
	if err != nil {
		t.Fatalf("load corpus %q: %v", root, err)
	}
	if len(cases) == 0 {
		t.Fatal("Marp golden corpus is empty")
	}

	rf := RenderFunc(markdown.RenderFunc())
	rep := report.New()

	var passed, blocked, failed []string

	for _, c := range cases {
		if c.RequiresEngine != "marp-core" {
			// Out of this gate's scope -- corpus_test.go's TestMarpCorpus
			// covers commonmark/"" cases against the goldmark baseline.
			continue
		}

		if reqID, skip := chaseSkipMap[c.ID]; skip {
			rep.AddPending("chase/" + c.ID + " (" + reqID + ")")
			blocked = append(blocked, c.ID)
			continue
		}

		pass, diff := RunCase(rf, c, "chase/"+c.ID, rep)
		if pass {
			passed = append(passed, c.ID)
		} else {
			failed = append(failed, c.ID)
			t.Errorf("case %q: htmldiff mismatch (not in the skip-map -- a genuine Objective-1 defect):\n%s", c.ID, diff)
		}
	}

	t.Logf("chase corpus: %d passed %v, %d blocked (skip-map) %v, %d failed %v\nper-case report:\n%s",
		len(passed), passed, len(blocked), blocked, len(failed), failed, rep.Render())

	for id := range marpitMechanicCases {
		found := false
		for _, p := range passed {
			if p == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Marpit-mechanic case %q did not PASS (passed=%v, failed=%v)", id, passed, failed)
		}
	}

	// Gate: zero cases fail for an Objective-1 reason -- every marp-core
	// case must be either PASS or explicitly BLOCKED.
	if len(failed) != 0 {
		t.Errorf("ZERO cases should FAIL for an Objective-1 reason; got %d: %v", len(failed), failed)
	}

	// Decision record (see marpitMechanicCases doc comment above):
	// marp-gfm-table PASSes outright -- goldmark's stock GFM table
	// extension needs no marp-core battery. Asserted explicitly so a
	// regression here is reported by name.
	gfmTablePassed := false
	for _, p := range passed {
		if p == "marp-gfm-table" {
			gfmTablePassed = true
			break
		}
	}
	if !gfmTablePassed {
		t.Errorf("marp-gfm-table: expected PASS (GFM tables need no marp-core battery), got passed=%v failed=%v", passed, failed)
	}
}
