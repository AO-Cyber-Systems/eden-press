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

// CONF-04: the full CommonMark + GFM spec sweep. Every vendored spec example is
// rendered through the SHARED goldmark engines (conformance/runner/engine.go) and
// DOM-diffed via conformance/htmldiff, with pass/fail tracked PER SECTION (never a
// bare aggregate — a 97% aggregate can hide a 0%-passing section like tabs or
// loose lists). In Objective 0 the engine-under-test is goldmark itself, so this
// records the BASELINE and proves the harness reports correctly; later objectives
// swap Eden Press's engine in behind the same runner. The gate is "the sweep runs
// and reports per section with the baseline recorded", NOT "100% green".

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/conformance/htmldiff"
	"github.com/AO-Cyber-Systems/eden-press/conformance/report"
	"github.com/AO-Cyber-Systems/eden-press/conformance/spec"
)

// specExample mirrors the shape emitted by CommonMark's / cmark-gfm's official
// test/spec_tests.py --dump-tests (also the shape goldmark's own tests use).
type specExample struct {
	Markdown  string `json:"markdown"`
	HTML      string `json:"html"`
	Example   int    `json:"example"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Section   string `json:"section"`
}

func loadSuite(t *testing.T, data []byte, name string) []specExample {
	t.Helper()
	var ex []specExample
	if err := json.Unmarshal(data, &ex); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
	if len(ex) == 0 {
		t.Fatalf("%s: suite is empty (truncated/corrupt vendor?)", name)
	}
	return ex
}

// unifyStrike rewrites goldmark's <del> strikethrough to markdown-it/Marp's <s>
// so the two spellings compare equal.
func unifyStrike(s string) string {
	s = strings.ReplaceAll(s, "<del>", "<s>")
	s = strings.ReplaceAll(s, "</del>", "</s>")
	return s
}

// isKnownDivergence reports whether an expected/actual mismatch is SOLELY the
// allow-listed strikethrough <s>/<del> cosmetic delta (PROPOSAL §12): the two
// become DOM-equal once the tags are unified. A genuine content mismatch (which
// stays unequal after unifying) returns false, so real regressions are never
// silently excused.
func isKnownDivergence(expected, actual string) bool {
	if eq, _ := htmldiff.Equal(expected, actual); eq {
		return false // already equal — not a divergence at all
	}
	eq, _ := htmldiff.Equal(unifyStrike(expected), unifyStrike(actual))
	return eq
}

func TestSpecSweep(t *testing.T) {
	cm := loadSuite(t, spec.CommonMark, "commonmark/spec.json")
	gfm := loadSuite(t, spec.GFM, "gfm/spec.json")
	ext := loadSuite(t, spec.GFMExtensions, "gfm/extensions.json")

	// Count guard vs VERSIONS.txt — a re-vendor that changes counts must update
	// VERSIONS.txt + NOTICE in the same change, and this catches a truncated file.
	for _, c := range []struct {
		name      string
		got, want int
	}{
		{"commonmark", len(cm), 652},
		{"gfm spec", len(gfm), 670},
		{"gfm extensions", len(ext), 30},
	} {
		if c.got != c.want {
			t.Errorf("%s count = %d, want %d (VERSIONS.txt drift — update VERSIONS.txt+NOTICE if intentional)", c.name, c.got, c.want)
		}
	}

	// Shared engines from 00-02 (engine.go) — no local newGM() here (00-05 shares
	// this package; a second definition would duplicate-symbol on merge).
	renderCM := GoldmarkRenderFunc(NewGoldmark())
	renderGFM := GoldmarkRenderFunc(NewGoldmarkGFM())

	rep := report.New()
	allowlisted := 0

	sweep := func(exs []specExample, render RenderFunc, prefix string) {
		for _, ex := range exs {
			section := strings.TrimSpace(ex.Section)
			if section == "" {
				section = "(unsectioned)"
			}
			section = prefix + section

			pass := false
			if actual, err := render(ex.Markdown, nil); err == nil {
				if eq, _ := htmldiff.Equal(ex.HTML, actual); eq {
					pass = true
				} else if isKnownDivergence(ex.HTML, actual) {
					pass = true
					allowlisted++
				}
			}
			rep.Add(section, pass)
		}
	}

	sweep(cm, renderCM, "CM/")
	sweep(gfm, renderGFM, "GFM/")
	sweep(ext, renderGFM, "EXT/")

	// The allow-list must be genuinely consulted for the known delta...
	if !isKnownDivergence("<p><del>x</del></p>", "<p><s>x</s></p>") {
		t.Error("allow-list not consulting the strikethrough <del>/<s> delta")
	}
	// ...and must NOT excuse a real content mismatch.
	if isKnownDivergence("<p>alpha</p>", "<p>beta</p>") {
		t.Error("allow-list wrongly excused a real content mismatch")
	}

	pass, total, failing := rep.Summary()
	t.Logf("per-section report:\n%s", rep.Render())
	t.Logf("BASELINE (goldmark base parser): %d/%d examples pass (%.1f%%); %d failing sections; %d strikethrough deltas allow-listed",
		pass, total, 100*float64(pass)/float64(max(total, 1)), failing, allowlisted)

	// Gate: every example is counted and results are tracked PER SECTION — not
	// that goldmark passes 100% (it is only the baseline engine in Objective 0).
	if wantTotal := len(cm) + len(gfm) + len(ext); total != wantTotal {
		t.Errorf("swept %d examples, want %d (every example must be counted once)", total, wantTotal)
	}
	if sections := strings.Count(strings.TrimRight(rep.Render(), "\n"), "\n") + 1; sections < 20 {
		t.Errorf("per-section report has only %d sections — results may be collapsed to an aggregate", sections)
	}
}
