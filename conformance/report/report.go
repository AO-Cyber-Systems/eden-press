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

// Package report provides SectionReport, a per-section pass/fail/pending
// aggregator for the Eden Press conformance harness. It renders a per-section
// breakdown (e.g. "Lists: 12/13") rather than only an aggregate percentage, so a
// regression can be traced to the exact spec section that broke.
//
// Pending results (AddPending) record cases deferred because the engine they need
// is not yet built. Pending is neither a pass nor a failure: it does not inflate
// pass/total and a section with only passes and pendings is NOT counted as
// failing. Both downstream runners (00-04 spec sweep, 00-05 Marp corpus) share
// this tracker.
package report

import (
	"fmt"
	"sort"
	"strings"
)

// sectionStat holds the running tally for one section.
type sectionStat struct {
	pass    int
	total   int // evaluated cases: pass + fail (pending excluded)
	pending int
}

// SectionReport accumulates results keyed by section name.
type SectionReport struct {
	stats map[string]*sectionStat
}

// New returns an empty SectionReport ready to accept results.
func New() *SectionReport {
	return &SectionReport{stats: make(map[string]*sectionStat)}
}

// stat returns (creating if needed) the tally for section.
func (r *SectionReport) stat(section string) *sectionStat {
	s, ok := r.stats[section]
	if !ok {
		s = &sectionStat{}
		r.stats[section] = s
	}
	return s
}

// Add records a single evaluated case under section: it increments the section's
// total and, when pass is true, its pass count.
func (r *SectionReport) Add(section string, pass bool) {
	s := r.stat(section)
	s.total++
	if pass {
		s.pass++
	}
}

// AddPending records a case that could not be evaluated because it requires an
// engine that is not yet built. It counts toward neither pass nor total, and does
// not make the section "failing".
func (r *SectionReport) AddPending(section string) {
	r.stat(section).pending++
}

// Render returns a per-section breakdown, one section per line, sorted by section
// name for deterministic output. Each line reads "<section>: <pass>/<total>",
// with a "(<n> pending)" suffix when the section has pending cases. An empty
// report renders a single "(no results)" line and never panics.
func (r *SectionReport) Render() string {
	if len(r.stats) == 0 {
		return "(no results)\n"
	}
	names := make([]string, 0, len(r.stats))
	for name := range r.stats {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		s := r.stats[name]
		fmt.Fprintf(&b, "%s: %d/%d", name, s.pass, s.total)
		if s.pending > 0 {
			fmt.Fprintf(&b, " (%d pending)", s.pending)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Summary returns the aggregate pass and total (evaluated) counts across all
// sections, plus failingSections — the number of sections with at least one
// failing case (total > pass). Pending cases never make a section failing.
func (r *SectionReport) Summary() (pass, total, failingSections int) {
	for _, s := range r.stats {
		pass += s.pass
		total += s.total
		if s.total > s.pass {
			failingSections++
		}
	}
	return pass, total, failingSections
}
