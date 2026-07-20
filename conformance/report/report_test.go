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

package report

import (
	"strings"
	"testing"
)

// Render() must show a per-section pass/total breakdown (not just an aggregate).
func TestSectionReport_RenderPerSection(t *testing.T) {
	r := New()
	for i := 0; i < 12; i++ {
		r.Add("Lists", true)
	}
	r.Add("Lists", false) // one failure -> Lists: 12/13
	for i := 0; i < 3; i++ {
		r.Add("Headings", true) // Headings: 3/3
	}

	out := r.Render()
	if !strings.Contains(out, "Lists: 12/13") {
		t.Errorf("Render() missing per-section Lists line; got:\n%s", out)
	}
	if !strings.Contains(out, "Headings: 3/3") {
		t.Errorf("Render() missing per-section Headings line; got:\n%s", out)
	}

	// Sections must render in a deterministic (sorted) order: Headings before Lists.
	if strings.Index(out, "Headings") > strings.Index(out, "Lists") {
		t.Errorf("Render() sections not sorted; got:\n%s", out)
	}
}

// Summary() returns aggregate pass/total AND the count of sections with any failure.
func TestSectionReport_Summary(t *testing.T) {
	r := New()
	r.Add("A", true)
	r.Add("A", false) // A: 1/2 (failing)
	r.Add("B", true)
	r.Add("B", true) // B: 2/2 (clean)

	pass, total, failing := r.Summary()
	if pass != 3 {
		t.Errorf("pass = %d, want 3", pass)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if failing != 1 {
		t.Errorf("failingSections = %d, want 1 (only A failed)", failing)
	}
}

// AddPending records deferred cases (engine not yet built): they must NOT count as
// a failure and must NOT inflate pass/total, but should be surfaced in Render().
func TestSectionReport_Pending(t *testing.T) {
	r := New()
	r.Add("Marp", true)
	r.AddPending("Marp")
	r.AddPending("Marp")

	pass, total, failing := r.Summary()
	if pass != 1 || total != 1 {
		t.Errorf("pending must not affect pass/total; got pass=%d total=%d, want 1/1", pass, total)
	}
	if failing != 0 {
		t.Errorf("a section with only pass+pending must not be 'failing'; got %d", failing)
	}

	out := r.Render()
	if !strings.Contains(out, "Marp: 1/1") {
		t.Errorf("Render() missing Marp pass/total; got:\n%s", out)
	}
	if !strings.Contains(out, "pending") {
		t.Errorf("Render() should surface pending count; got:\n%s", out)
	}
}

// An empty report renders (and summarizes) without panicking.
func TestSectionReport_EmptyDoesNotPanic(t *testing.T) {
	r := New()
	_ = r.Render() // must not panic
	pass, total, failing := r.Summary()
	if pass != 0 || total != 0 || failing != 0 {
		t.Errorf("empty report Summary = (%d,%d,%d), want all zero", pass, total, failing)
	}
}
