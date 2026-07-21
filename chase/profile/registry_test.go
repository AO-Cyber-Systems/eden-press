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

package profile

import "testing"

// ---------------------------------------------------------------------
// Task 1 — Test-list case 5. Written FIRST, before profile.go exists:
// this file intentionally fails to compile until the Profile interface
// (and its SizeTable/PaginationRule value types) are defined.
//
// Task 2 (registry.go) will extend this file with Register/Get/Default
// behavior tests (Test-list cases 1-4); those are added alongside
// registry.go itself, once this interface-satisfaction test is green.
// ---------------------------------------------------------------------

// fakeProfile is a hand-built, minimal Profile implementation used only
// by this package's own tests — never a generated mock. Its zero value
// already satisfies every method (each returns its zero value), so
// `fakeProfile{}` alone is enough for the compile-time assertion below;
// fields are populated per-test only where a test cares about a
// specific value.
type fakeProfile struct {
	id          string
	unitElement string
	container   func(inlineSVG bool) string
	sizes       SizeTable
	pagination  PaginationRule
	scaffold    func(inlineSVG bool) string
}

func (f fakeProfile) ID() string          { return f.id }
func (f fakeProfile) UnitElement() string { return f.unitElement }

func (f fakeProfile) Container(inlineSVG bool) string {
	if f.container != nil {
		return f.container(inlineSVG)
	}
	return ""
}

func (f fakeProfile) Sizes() SizeTable           { return f.sizes }
func (f fakeProfile) Pagination() PaginationRule { return f.pagination }

func (f fakeProfile) Scaffold(inlineSVG bool) string {
	if f.scaffold != nil {
		return f.scaffold(inlineSVG)
	}
	return ""
}

// Test-list case 5: the Profile interface is satisfiable by a minimal
// fake implementing every method — a compile-time assertion, not a
// runtime check.
var _ Profile = fakeProfile{}

// TestInterfaceSatisfaction exercises the compile-time assertion above
// through an actual call, so `go test -run TestInterface` has a real
// test (not just a var declaration) to report PASS/FAIL for.
func TestInterfaceSatisfaction(t *testing.T) {
	var p Profile = fakeProfile{id: "fake"}

	if got, want := p.ID(), "fake"; got != want {
		t.Fatalf("ID() = %q, want %q", got, want)
	}
}
