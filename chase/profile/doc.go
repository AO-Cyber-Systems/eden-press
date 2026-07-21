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

// <!-- TDD-EXCEPTION: Documentation-only change (package doc comment
// recording an architecture decision); no testable behavior. -->

// Package profile defines chase's output-profile abstraction
// (differentiator #1 — PROJECT.md; ARCHITECTURE.md Pattern 2): the
// Profile interface (profile.go) plus a small package-level registry
// (Register/Get/Default, registry.go) that lets an output kind
// ("slides" today; "paged"/"article"/"epub" later) supply exactly the
// tables/predicates chase's profile-agnostic parsing/directive/theme
// core needs — and nothing else.
//
// # Decision gate resolved: chase/* is EXPORTED, not internal
//
// ROADMAP.md's Objective-2 entry opened a decision gate: should
// chase/* (this package included) get an `internal/` prefix, forcing
// every consumer through press/, or stay independently importable?
//
// RESOLVED here: chase/* stays EXPORTED — no `internal/` prefix
// anywhere under chase/. Rationale (PROJECT.md's differentiator #2,
// "library-first & server-native"): profiles/paged, profiles/article,
// and profiles/epub (future objectives) — and any advanced external
// consumer building a custom output profile Eden Press itself never
// ships — must be able to `import "github.com/AO-Cyber-Systems/eden-press/chase/profile"`
// (and chase/theme, chase/model, chase/directive) directly, exactly
// the way profiles/slides (02-03) does. An `internal/` prefix would
// force every one of those consumers through press/, which is the
// batteries+facade layer, not the framework — that is the opposite of
// "pluggable output profiles" (differentiator #1) actually being
// pluggable by anyone outside this module.
//
// This is a one-way decision for the whole chase/ tree, not just this
// package: `find chase -type d -name internal` must always return
// nothing (enforced by this TRD's Task 3 verify command, and worth a
// standing CI check per ARCHITECTURE.md's Anti-Pattern-3-style
// `go list -deps` checks).
//
// # Deferred methods: Boundary() and Directives()
//
// ARCHITECTURE.md Pattern 2's sketch of a FUTURE, more-general Profile
// includes two methods this package deliberately does NOT define yet:
//
//   - Boundary(n ast.Node, pc parser.Context) bool — reports whether an
//     AST node starts a new Unit. Deferred because profiles/slides
//     (02-03) doesn't need it: chase/markdown's boundary transformer
//     already regroups siblings on thematic breaks independently of
//     any Profile method, and no consumer in this objective calls a
//     profile-supplied boundary predicate.
//   - Directives() []directive.Spec — lets a profile register
//     profile-only directives (e.g. a hypothetical paged profile's
//     `toc:`/`runningHeader:`) into chase/directive's schema. Deferred
//     because slides adds no profile-only directives in v1 — every
//     directive slides needs is already profile-agnostic.
//
// Adding either now — before a second profile exists to prove the
// shape is right — is exactly the SPECULATIVE SUPERSET anti-pattern
// this TRD's anti_patterns section locks against: "do NOT add
// Boundary()/Directives(), or any method that has NO consumer in this
// objective." When 02-03 or a later paged/article/epub objective has a
// concrete call-site for either method, add it then, bottom-up, the
// same way every method in profile.go's Profile interface was derived
// from a specific hardcoded call-site in chase/theme (see profile.go's
// per-method doc comments).
package profile
