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

package theme

// pass.go defines the small pipeline-runner abstraction shared by both
// Tier-1 (Theme.Load, add-time, see theme.go) and Tier-2 (ThemeSet.Pack,
// render-time, see pack.go): a Pass is a named, ordered mutation over a
// Stylesheet, and Run applies a sequence of them in order, stopping and
// returning the first error.
//
// This exists specifically so BOTH tiers are built from the SAME small
// abstraction (rather than each tier hand-rolling its own bespoke
// sequential-if-err-return-err chain) — the ordering itself is the
// load-bearing part of this TRD (01-RESEARCH.md Pitfall 1: passes are NOT
// a simplifiable linear pipeline; :root is rewritten twice, and the
// specificity trick must run strictly after selector-scoping), so making
// "a Pass runs, in order, over the current Stylesheet" a first-class,
// testable concept is deliberate.
type Pass struct {
	// Name identifies the pass for error-wrapping / diagnostics (e.g.
	// "nesting", "root-mark", "scope", "increasing-specificity").
	Name string
	// Run mutates sheet in place (or reassigns its fields) and returns an
	// error if the pass cannot proceed (e.g. unsupported nesting depth,
	// circular @import-theme).
	Run func(sheet *Stylesheet) error
}

// RunPasses applies passes to sheet in order, stopping at (and returning)
// the first error, wrapped with the failing pass's Name for diagnostics.
func RunPasses(sheet *Stylesheet, passes ...Pass) error {
	for _, p := range passes {
		if err := p.Run(sheet); err != nil {
			return &PassError{Pass: p.Name, Err: err}
		}
	}
	return nil
}

// PassError wraps an error raised by a named Pass.
type PassError struct {
	Pass string
	Err  error
}

func (e *PassError) Error() string {
	return "theme: pass " + e.Pass + ": " + e.Err.Error()
}

func (e *PassError) Unwrap() error { return e.Err }
