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

import "sync"

// registry.go is chase/profile's package-level Profile registry:
// profiles register themselves by ID (profiles/slides calls Register
// from its init/New, wired in 02-03) and the chase entrypoint (02-04)
// resolves one by ID via Get, or falls back to Default.
//
// Guarded by a sync.RWMutex (not a bare map) since press.Render is
// expected to be safe to call concurrently across goroutines
// (ARCHITECTURE.md's scaling note, "embedded in a multi-tenant
// service"): registration happens once at process/package-init time,
// but Get/Default may be read from many goroutines concurrently
// during concurrent renders.

var (
	registryMu sync.RWMutex
	registry   = map[string]Profile{}
	// registrationOrder tracks distinct IDs in first-registered order,
	// so Default() is deterministic (insertion order) rather than
	// dependent on Go's randomized map iteration order.
	registrationOrder []string
)

// Register adds p to the registry under p.ID(). Registering a
// duplicate ID REPLACES the previous registration under that ID —
// the documented behavior for this TRD's decision point (see
// error_recovery: "pick ONE documented behavior ... and test it",
// Test-list case 4). Replace (not reject-with-error) is chosen because
// re-registering the same ID is a legitimate, expected occurrence
// (tests, hot-reload of a profile during development), not a
// programmer error worth failing on. The FIRST distinct ID ever
// registered becomes the deterministic Default() fallback; replacing
// an already-registered ID does not change which ID is default.
func Register(p Profile) {
	registryMu.Lock()
	defer registryMu.Unlock()

	id := p.ID()
	if _, exists := registry[id]; !exists {
		registrationOrder = append(registrationOrder, id)
	}
	registry[id] = p
}

// Get resolves a Profile by ID. found is false when no profile is
// registered under id — callers must check found rather than trusting
// a possibly-nil Profile (Test-list case 2).
func Get(id string) (p Profile, found bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	p, found = registry[id]
	return p, found
}

// Default returns the registry's default Profile: the first distinct
// ID ever registered (deterministic insertion order, not map iteration
// order — Test-list case 3), matching the common v1 case of exactly
// one registered profile ("slides"). Default returns nil if nothing
// has been registered yet.
func Default() Profile {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if len(registrationOrder) == 0 {
		return nil
	}
	return registry[registrationOrder[0]]
}

// resetForTest clears all registry state. It exists ONLY so this
// package's own tests (registry_test.go, same package) can isolate
// each test case from package-level mutable state left behind by a
// previous test — it is unexported and never called by production
// code (profiles/slides, the chase entrypoint) or by any other
// package.
func resetForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry = map[string]Profile{}
	registrationOrder = nil
}
