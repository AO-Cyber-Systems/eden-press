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

import (
	"sync"
	"testing"
)

// ---------------------------------------------------------------------
// Task 1 — Test-list case 5. Written FIRST, before profile.go exists:
// this file intentionally fails to compile until the Profile interface
// (and its SizeTable/PaginationRule value types) are defined.
//
// Task 2 — Test-list cases 1-4, below. Written FIRST, before
// registry.go exists: these tests intentionally fail to compile until
// Register/Get/Default/resetForTest exist.
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

// ---------------------------------------------------------------------
// Task 2 — registry behavior (Register/Get/Default). The registry is
// package-level mutable state (registry.go), so every test below calls
// resetForTest() first for isolation — an unexported test-only hook
// (see registry.go's doc), never part of the public surface.
// ---------------------------------------------------------------------

// Test-list case 1: Register(fakeProfile{id:"x"}) then Get("x") ->
// returns that profile, found==true.
func TestRegisterAndGet(t *testing.T) {
	resetForTest()
	Register(fakeProfile{id: "x"})

	got, found := Get("x")
	if !found {
		t.Fatalf("Get(%q) found = false, want true", "x")
	}
	if got.ID() != "x" {
		t.Fatalf("Get(%q).ID() = %q, want %q", "x", got.ID(), "x")
	}
}

// Test-list case 2: Get("nope") -> found==false, nil profile.
func TestGetUnknownIDNotFound(t *testing.T) {
	resetForTest()
	Register(fakeProfile{id: "x"})

	got, found := Get("nope")
	if found {
		t.Fatalf("Get(%q) found = true, want false", "nope")
	}
	if got != nil {
		t.Fatalf("Get(%q) profile = %#v, want nil", "nope", got)
	}
}

// Test-list case 3: Default() returns the registered default (the
// first distinct ID ever registered) deterministically — not
// map-iteration-order dependent.
func TestDefaultIsDeterministicFirstRegistered(t *testing.T) {
	resetForTest()
	Register(fakeProfile{id: "first"})
	Register(fakeProfile{id: "second"})

	if got := Default(); got == nil || got.ID() != "first" {
		t.Fatalf("Default().ID() = %v, want %q", got, "first")
	}
}

// Test-list case 4: duplicate-ID register is documented behavior:
// last-wins REPLACE (chosen over reject-with-error — see registry.go's
// Register doc for rationale). Re-registering the SAME ID must not
// change which ID is Default.
func TestDuplicateIDRegisterReplaces(t *testing.T) {
	resetForTest()
	Register(fakeProfile{id: "x", unitElement: "section"})
	Register(fakeProfile{id: "x", unitElement: "page"})

	got, found := Get("x")
	if !found {
		t.Fatalf("Get(%q) found = false, want true", "x")
	}
	if got.UnitElement() != "page" {
		t.Fatalf("Get(%q).UnitElement() = %q, want %q (last-wins replace)", "x", got.UnitElement(), "page")
	}
	if def := Default(); def.ID() != "x" {
		t.Fatalf("Default().ID() = %q, want %q (re-register must not change default)", def.ID(), "x")
	}
}

// Concurrent Get/Default access must be race-free — press.Render is
// expected to be called concurrently across goroutines
// (ARCHITECTURE.md's scaling note), so registry reads must be safe
// under `go test -race`. Registration itself is expected once at
// process/package-init time (profiles register themselves in
// init/New), not exercised concurrently here.
func TestRegistryConcurrentReadsAreRaceFree(t *testing.T) {
	resetForTest()
	Register(fakeProfile{id: "concurrent"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = Get("concurrent")
			_ = Default()
		}()
	}
	wg.Wait()
}
