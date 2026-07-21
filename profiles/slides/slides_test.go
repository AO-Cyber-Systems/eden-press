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

package slides

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/profile"
	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
)

// slides_test.go covers this TRD's Test-list cases 1-4: profiles/slides
// is the only registered Profile, and it reproduces the exact slide
// values chase/theme used to hardcode.

// Test-list case 1: registration.
func TestSlidesIDAndRegistration(t *testing.T) {
	if got, want := (slides{}).ID(), "slides"; got != want {
		t.Fatalf("ID() = %q, want %q", got, want)
	}

	p, found := profile.Get("slides")
	if !found {
		t.Fatal(`profile.Get("slides") found = false, want true`)
	}
	if p.ID() != "slides" {
		t.Fatalf("profile.Get(\"slides\").ID() = %q, want %q", p.ID(), "slides")
	}

	def := profile.Default()
	if def == nil || def.ID() != "slides" {
		t.Fatalf("profile.Default() = %v, want the slides profile", def)
	}
}

// Test-list case 2: unit element + container chains.
func TestSlidesUnitElementAndContainer(t *testing.T) {
	p := slides{}

	if got, want := p.UnitElement(), "section"; got != want {
		t.Fatalf("UnitElement() = %q, want %q", got, want)
	}
	if got, want := p.Container(false), "div.marpit"; got != want {
		t.Fatalf("Container(false) = %q, want %q", got, want)
	}
	if got, want := p.Container(true), "div.marpit > svg > foreignObject"; got != want {
		t.Fatalf("Container(true) = %q, want %q", got, want)
	}
}

// Test-list case 3: size table.
func TestSlidesSizes(t *testing.T) {
	sizes := (slides{}).Sizes()

	want169 := theme.Size{Name: "16:9", WidthPx: 1280, HeightPx: 720}
	want43 := theme.Size{Name: "4:3", WidthPx: 960, HeightPx: 720}

	if sizes.Default != want169 {
		t.Fatalf("Default = %+v, want %+v", sizes.Default, want169)
	}
	if got := sizes.ByName["16:9"]; got != want169 {
		t.Fatalf(`ByName["16:9"] = %+v, want %+v`, got, want169)
	}
	if got := sizes.ByName["4:3"]; got != want43 {
		t.Fatalf(`ByName["4:3"] = %+v, want %+v`, got, want43)
	}
}

// Test-list case 4: pagination + scaffold.
func TestSlidesPaginationAndScaffold(t *testing.T) {
	p := slides{}

	pag := p.Pagination()
	if pag.Selector != "section::after" {
		t.Fatalf("Pagination().Selector = %q, want %q", pag.Selector, "section::after")
	}
	if pag.ContentExpr != "attr(data-marpit-pagination)" {
		t.Fatalf("Pagination().ContentExpr = %q, want %q", pag.ContentExpr, "attr(data-marpit-pagination)")
	}

	if got, want := p.Scaffold(false), ScaffoldCSS; got != want {
		t.Fatalf("Scaffold(false) did not return ScaffoldCSS verbatim (len got=%d want=%d)", len(got), len(want))
	}
	if got, want := p.Scaffold(true), ScaffoldCSS+AdvancedBackgroundCSS; got != want {
		t.Fatalf("Scaffold(true) did not return ScaffoldCSS+AdvancedBackgroundCSS verbatim (len got=%d want=%d)", len(got), len(want))
	}
}

// ---------------------------------------------------------------------
// Task 3 — grep-gate: profiles/slides is the ONLY place the
// profile-varying values may live; chase/model and chase/theme's
// PRODUCTION code (non-_test.go) must never re-hardcode them.
//
// The three patterns below are the TRD's exact, literal grep-gate
// commands (MODEL-04), reproduced here as a real, CI-enforceable test
// rather than a one-off manual grep: any Slide-family exported
// identifier, the quoted "section" unit-element literal, or any of the
// 16:9/4:3/1280/720/960 size-table numbers appearing in
// chase/model or chase/theme's non-test .go files is a grep-gate
// regression.
// ---------------------------------------------------------------------

// repoRootForGrepGate locates the repository root from this package's own
// working directory (profiles/slides is two levels below the repo root:
// profiles/slides -> profiles -> root).
func repoRootForGrepGate(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(wd, "..", "..")
}

// assertGrepGateClean runs `grep -r[E] pattern chase/model chase/theme
// --include='*.go'`, then filters out any _test.go match (mirroring the
// TRD's own `| grep -v _test.go` pipe stage), failing the test if any
// production-code match remains. extended selects -E (extended regex,
// patterns 1 and 3) vs plain grep (pattern 2's literal quoted string).
func assertGrepGateClean(t *testing.T, repoRoot, pattern string, extended bool) {
	t.Helper()

	args := []string{"-rn"}
	if extended {
		args = append(args, "-E")
	}
	args = append(args, pattern, "chase/model", "chase/theme", "--include=*.go")

	cmd := exec.Command("grep", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// grep exit code 1 = no matches found at all -- the gate
			// passes trivially.
			return
		}
		t.Fatalf("grep -rn%s %q chase/model chase/theme: %v", map[bool]string{true: "E", false: ""}[extended], pattern, err)
	}

	var violations []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" || strings.Contains(line, "_test.go") {
			continue
		}
		violations = append(violations, line)
	}
	if len(violations) > 0 {
		t.Fatalf("grep-gate violation for pattern %q (MODEL-04, TRD 02-03) in chase/model or chase/theme production code:\n%s",
			pattern, strings.Join(violations, "\n"))
	}
}

// TestGrepGate is Task 3's CI-style assertion (TRD 02-03's grep-gate
// proof): chase/model and chase/theme's production (.go, non-test) files
// must carry NONE of the profile-varying literals that were relocated
// into profiles/slides -- no Slide-family exported identifier, no quoted
// "section" unit-element literal, and no 16:9/4:3/1280/720/960 size-table
// number.
func TestGrepGate(t *testing.T) {
	repoRoot := repoRootForGrepGate(t)

	// Pattern 1: no Slide-family identifier (case-sensitive, capital S
	// required -- lowercase "slide" in prose/variable names, e.g.
	// Marpit's own container/slide placeholder terminology, is exempt).
	assertGrepGateClean(t, repoRoot, `\bSlide[A-Za-z]*\b`, true)

	// Pattern 2: the literal 10-char quoted "section" substring must not
	// appear anywhere, including comments. Bare, unquoted "section" in
	// prose is safe.
	assertGrepGateClean(t, repoRoot, `"section"`, false)

	// Pattern 3: no hardcoded size-table numbers, including in
	// comments/prose.
	assertGrepGateClean(t, repoRoot, `16:9|4:3|1280|720|960`, true)
}
