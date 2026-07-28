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

	// The scaffold's paged-specific CSS reaches the packed stylesheet,
	// INCLUDING its @media print block — the rules that enforce one section
	// per physical page, which the packer used to discard silently.
	for _, want := range []string{
		"edenpress-paged",
		"counter-increment: edenpress-page",
		"@media print {",
		"page-break-after: always",
	} {
		if !strings.Contains(out.CSS, want) {
			t.Errorf("packed CSS missing %q", want)
		}
	}

	// Rules are scoped under THIS profile's container, not the slides chain.
	if strings.Contains(out.CSS, "div.marpit") {
		t.Errorf("paged CSS is scoped under the slides container chain: %s", firstLines(out.CSS, 3))
	}
	if !strings.Contains(out.CSS, "div.edenpress-paged") {
		t.Errorf("paged CSS is not scoped under its own container: %s", firstLines(out.CSS, 3))
	}

	// Both sections survive, and the model is profile-agnostic as designed.
	if got := len(out.Model.Sections); got != 2 {
		t.Errorf("sections = %d, want 2", got)
	}
}

// TestPagedPipelineGapsClosed guards the two chase/theme packing defects that
// EPD-R2 uncovered and fixed, so a regression is caught here rather than
// showing up as silently wrong CSS.
//
// Gap 1 — block at-rule contents were discarded. chase/theme/parse.go recorded
// only an at-rule's OPENING into Stylesheet.Atoms and dropped its body, so
// `@media print { ... }` packed to the literal, invalid `@media print;`. It hit
// a shipped theme: themes/uncover.css's @media print block lost its
// print-specific pagination styling in every packed stylesheet. It survived
// because profiles/slides' scaffold contains zero at-rules, so nothing
// exercised the path until profiles/paged needed print rules.
//
// Gap 2 — Profile.Container() was never wired into the packing pipeline. Its
// own doc claimed it "de-hardcodes chase/theme/selector/scope.go's
// inlineSVGChain / nonSVGChain", but press never passed it, so those package
// vars stayed the real source of truth and paged's rules were scoped under
// "div.marpit > svg > foreignObject" — CSS that could never match its own
// markup. Closed by ThemeSet.SetContainerChains.
func TestPagedPipelineGapsClosed(t *testing.T) {
	// Gap 1, on the shipped theme that was actually affected.
	uncover, err := press.Render(pagedDoc, press.Options{Theme: "uncover"})
	if err != nil {
		t.Fatalf("Render(uncover): %v", err)
	}
	if strings.Contains(uncover.CSS, "@media print;") {
		t.Error("GAP 1 REGRESSION: at-rule block emitted in the invalid statement form")
	}
	if !strings.Contains(uncover.CSS, "@media print {") {
		t.Error("GAP 1 REGRESSION: uncover's @media print block lost its body again")
	}

	// Gap 2: each profile's rules scope under its OWN container chain.
	slides, err := press.Render(pagedDoc, press.Options{Profile: "slides"})
	if err != nil {
		t.Fatalf("Render(slides): %v", err)
	}
	if !strings.Contains(slides.CSS, "div.marpit") {
		t.Error("GAP 2 REGRESSION: slides no longer scopes under the marpit chain")
	}

	paged, err := press.Render(pagedDoc, press.Options{Profile: "paged"})
	if err != nil {
		t.Fatalf("Render(paged): %v", err)
	}
	if strings.Contains(paged.CSS, "div.marpit") {
		t.Error("GAP 2 REGRESSION: paged is scoped under the slides container chain")
	}
}

// firstLines returns at most n lines of s, for readable failure output.
func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
