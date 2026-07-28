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

package press_test

import (
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/press"

	// Blank import registers profiles/paged. press itself only blank-imports
	// profiles/slides, so a consumer opting into a second profile opts in by
	// importing it -- the same registry contract slides uses.
	_ "github.com/AO-Cyber-Systems/eden-press/profiles/paged"
)

const pagedDoc = `# Q3 Report

Body text.

---

## Appendix

| Metric | Q3 |
|---|---|
| p95 | 550ms |
`

// TestPagedProfileEndToEnd is the payoff for the EPD-R2 container-class seam:
// selecting a non-slides profile must produce markup AND CSS that agree. It
// would have passed vacuously before the seam existed, because the container
// class was a "marpit" literal regardless of profile.
func TestPagedProfileEndToEnd(t *testing.T) {
	out, err := press.Render(pagedDoc, press.Options{Profile: "paged"})
	if err != nil {
		t.Fatalf("Render(paged): %v", err)
	}

	if !strings.Contains(out.HTML, `<div class="edenpress-paged">`) {
		t.Errorf("paged container class not emitted: %s", out.HTML)
	}
	if strings.Contains(out.HTML, `class="marpit"`) {
		t.Errorf("marpit container leaked into paged output: %s", out.HTML)
	}

	// The scaffold's paged-specific CSS reaches the packed stylesheet.
	for _, want := range []string{"edenpress-paged", "counter-increment: edenpress-page"} {
		if !strings.Contains(out.CSS, want) {
			t.Errorf("packed CSS missing %q", want)
		}
	}

	// Both sections survive, and the model is profile-agnostic as designed.
	if got := len(out.Model.Sections); got != 2 {
		t.Errorf("sections = %d, want 2", got)
	}
}

// TestPagedPipelineGaps is a CHARACTERIZATION test: it pins two behaviors that
// are currently WRONG, so that fixing either one fails here loudly instead of
// changing output nobody is watching. Both are gaps in chase/theme's packing
// pipeline, not in profiles/paged, whose own Scaffold()/Container() are correct
// and unit-tested in that package.
//
// When you fix one of these, INVERT the corresponding assertion here.
//
// Gap 1 — block at-rule contents are discarded from scaffold CSS.
// chase/theme/parse.go records only an at-rule's opening into Stylesheet.Atoms
// and, by its own doc comment, "its block contents (including any nested
// rulesets) are never modeled". So paged's `@media print { ... }` block — the
// rules that enforce one section per physical page — vanishes from the packed
// output. It never mattered before because profiles/slides' scaffold contains
// zero at-rules (grep-verified), so this path was never exercised. Note the
// inner rules are DROPPED, not leaked unwrapped, so print-only declarations at
// least do not wrongly apply on screen.
//
// Gap 2 — Profile.Container() is not wired into the packing pipeline.
// press calls themes.ThemeSet(p.UnitElement(), ...) and never passes the
// container chain, so chase/theme/selector/scope.go's hardcoded slides chains
// remain the source of truth and paged's rules are scoped under
// "div.marpit > svg > foreignObject > section". Container()'s own doc claims it
// "de-hardcodes chase/theme/selector/scope.go's inlineSVGChain / nonSVGChain",
// but its only non-test caller is convert/png/png.go, whose comment reads
// `// always "div.marpit"`. The method's documented purpose is unfulfilled.
func TestPagedPipelineGaps(t *testing.T) {
	out, err := press.Render(pagedDoc, press.Options{Profile: "paged"})
	if err != nil {
		t.Fatalf("Render(paged): %v", err)
	}

	if strings.Contains(out.CSS, "@media") {
		t.Error("GAP 1 FIXED: at-rules now survive scaffold packing — " +
			"invert this assertion and restore the @media print check in TestPagedProfileEndToEnd")
	}
	if !strings.Contains(out.CSS, "div.marpit > svg > foreignObject") {
		t.Error("GAP 2 FIXED: the packing pipeline no longer hardcodes the slides container chain — " +
			"invert this assertion and assert paged's own chain instead")
	}
}

// TestSlidesProfileUnchanged is the regression guard for the seam: the default
// profile must still emit exactly the Marp container every conformance case
// and bundled theme depends on.
func TestSlidesProfileUnchanged(t *testing.T) {
	for _, name := range []string{"", "slides"} {
		out, err := press.Render(pagedDoc, press.Options{Profile: name})
		if err != nil {
			t.Fatalf("Render(%q): %v", name, err)
		}
		if !strings.Contains(out.HTML, `<div class="marpit">`) {
			t.Errorf("profile %q: marpit container missing: %s", name, out.HTML)
		}
		if strings.Contains(out.HTML, "edenpress-paged") {
			t.Errorf("profile %q: paged container leaked into slides output", name)
		}
	}
}
