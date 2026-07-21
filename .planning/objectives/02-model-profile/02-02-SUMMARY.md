---
objective: 02-model-profile
job: "02"
subsystem: profile
tags: [go, interface-design, registry, concurrency, decision-gate, architecture]

# Dependency graph
requires:
  - objective: 01-chase-framework
    provides: chase/theme.Size value type (chase/theme/stylesheet.go) — reused by SizeTable, one-way import edge
provides:
  - "chase/profile package: Profile interface (ID/UnitElement/Container/Sizes/Pagination/Scaffold) — bottom-up, every method mapped to a named hardcoded call-site in chase/theme"
  - "Package-level registry: Register/Get/Default, sync.RWMutex-guarded, race-tested, deterministic first-registered Default()"
  - "Resolved objective decision gate: chase/* is EXPORTED (no internal/ prefix), documented in chase/profile/doc.go"
affects: [02-03, 02-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Bottom-up interface design: every Profile method cites the specific hardcoded call-site in chase/theme it will replace (scope.go chains, meta.go size fallback, pass_pagination.go literals, scaffold.go CSS constants) — no speculative method added"
    - "Package-level registry with unexported resetForTest() test-isolation hook (not part of public surface) for deterministic tests against shared mutable state"
    - "Deferred-method documentation pattern: doc.go records WHY Boundary()/Directives() were intentionally left out, not just that they were"

key-files:
  created:
    - chase/profile/profile.go
    - chase/profile/registry.go
    - chase/profile/registry_test.go
    - chase/profile/doc.go
  modified: []

key-decisions:
  - "chase/* stays EXPORTED (no internal/ prefix) — resolves the objective's decision gate, documented authoritatively in chase/profile/doc.go per PROJECT.md differentiator #2 (library-first & server-native)"
  - "Duplicate-ID Register() is last-wins REPLACE (not reject-with-error) — re-registration is a legitimate test/hot-reload case, not a programmer error; re-registering an existing ID never changes which ID is Default()"
  - "Default() returns nil (not panic) when nothing is registered — simplest option, no untested crash path introduced beyond the required test-list cases"
  - "Boundary()/Directives() deliberately deferred (not added) — no consumer in this objective; documented in doc.go per the TRD's locked anti-pattern against a speculative superset"

patterns-established:
  - "chase/profile may import chase/theme for shared value types (Size); chase/theme MUST NOT import chase/profile back — one-way edge, verified by successful ./chase/... build with no cycle"

requirements-completed: [MODEL-03]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 8min
completed: 2026-07-21
---

# Objective 02 TRD 02: chase/profile Interface + Registry Summary

**`chase/profile` ships a lean, bottom-up `Profile` interface (ID/UnitElement/Container/Sizes/Pagination/Scaffold) plus a race-safe package registry (Register/Get/Default), resolving the objective's exported-vs-internal decision gate as EXPORTED with the rationale recorded in doc.go.**

## Performance

- **Duration:** 8 min (first RED commit to last commit: 01:21:35 -> 01:23:36 local)
- **Started:** 2026-07-21T05:21:35Z
- **Completed:** 2026-07-21T05:23:36Z
- **Tasks:** 3
- **Files modified:** 4 (all newly created)

## Accomplishments
- `Profile` interface with exactly 6 methods, each traced in its doc comment to the specific hardcoded call-site in `chase/theme` it de-hardcodes for 02-03/02-04 (no speculative method)
- `SizeTable`/`PaginationRule` shared value types; `SizeTable` reuses `chase/theme.Size` directly (one-way import edge — `chase/theme` never imports `chase/profile`)
- Package-level registry (`Register`/`Get`/`Default`) guarded by `sync.RWMutex`, verified race-free under `go test -race`, with deterministic first-registered-ID `Default()` and documented last-wins duplicate-ID replace semantics
- `doc.go` resolves the objective's `chase/*` internal-vs-exported decision gate (EXPORTED) and records why `Boundary()`/`Directives()` were deliberately deferred

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Define Profile interface + shared value types | `go build ./chase/profile/ && go test ./chase/profile/ -run TestInterface -v` | 0 | PASS |
| 2: Implement profile registry (Register/Get/Default) | `go test ./chase/profile/... -v && go vet ./chase/profile/...` | 0 | PASS |
| 3: Record EXPORTED-vs-internal decision in doc.go | `go build ./chase/... && test -z "$(find chase -type d -name internal)" && grep -qi 'exported' chase/profile/doc.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (raw `git commit` never used):

1. **Task 1 RED: failing Profile interface satisfaction test** - `400ba71` (test)
2. **Task 1 GREEN: Profile interface + shared value types** - `283451b` (feat)
3. **Task 2 RED: failing registry behavior tests** - `9940725` (test)
4. **Task 2 GREEN: profile registry (Register/Get/Default)** - `5a8acd3` (feat)
5. **Task 3: EXPORTED-vs-internal decision record (doc.go)** - `c2c6e51` (docs; TDD-EXCEPTION, documentation-only)

**Plan metadata:** `5d059f4` (docs: create objective TRDs) — pre-existing, not part of this TRD's execution.

_Note: Tasks 1 and 2 are TDD (RED->GREEN two-commit pairs); Task 3 is a documentation-only file with no testable behavior, executed under the TDD-EXCEPTION mechanism (see TDD Exceptions below) as a single commit._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./chase/profile/...` | 0 | PASS |
| vet | `go vet ./chase/profile/...` | 0 | PASS |
| test | `go test ./chase/profile/...` | 0 | PASS |

Additional evidence gathered beyond the TRD's 3 named gates:
- `go build ./chase/... && go vet ./chase/... && go test ./chase/...` (whole `chase/` tree) — exit 0, all 5 packages pass (`chase/directive`, `chase/markdown`, `chase/profile`, `chase/theme`, `chase/theme/selector`)
- `go build ./... && go test ./...` (whole repo) — exit 0, no regressions in `conformance/*`
- `go test -race ./chase/profile/...` — exit 0, 6 test functions pass including a 50-goroutine concurrent `Get`/`Default` read test
- `addlicense -l mit -s -c "AO Cyber Systems" -check chase/profile/` — exit 0, all 4 new files carry the Eden MIT header ending in `// SPDX-License-Identifier: MIT`
- `gofmt -l chase/profile/` — empty output (no formatting diffs)
- `go doc ./chase/profile` — renders the package doc cleanly, confirming `doc.go`'s comment (not a stray file-level comment) is recognized as the package doc

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1, commit `400ba71`) | `go test ./chase/profile/ -run TestInterface -v` | 1 (`undefined: Profile`/`SizeTable`/`PaginationRule`) | FAIL (correct) |
| GREEN (Task 1, commit `283451b`) | `go test ./chase/profile/ -run TestInterface -v` | 0 | PASS (correct) |
| RED (Task 2, commit `9940725`) | `go test ./chase/profile/...` | 1 (`undefined: Register`/`Get`/`Default`/`resetForTest`) | FAIL (correct) |
| GREEN (Task 2, commit `5a8acd3`) | `go test ./chase/profile/... -v` | 0 | PASS (correct) |

_REFACTOR: none required — both GREEN implementations passed on first write with no cleanup needed._

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 3/3 (all 3 TRD `must_haves.truths` confirmed: lean bottom-up interface with no speculative method; registry resolves/not-found/duplicate behavior all tested; exported-vs-internal decision resolved and documented in-package)
- **Gate failures:** None

## TDD Exceptions

- Task 3 (`chase/profile/doc.go`): documentation-only change (package doc comment recording an architecture decision) — no testable behavior. Marked `<!-- TDD-EXCEPTION: Documentation-only change (package doc comment recording an architecture decision); no testable behavior. -->` in the file itself. Verified instead via `go build`, `find chase -type d -name internal` (empty), and `grep -qi 'exported'` on the file content.

## Files Created/Modified
- `chase/profile/profile.go` (113 lines) - `Profile` interface (6 methods, each doc-linked to its de-hardcoding call-site) + `SizeTable`/`PaginationRule` value types
- `chase/profile/registry.go` (107 lines) - `Register`/`Get`/`Default` package-level registry, `sync.RWMutex`-guarded, plus the unexported `resetForTest` test-isolation hook
- `chase/profile/registry_test.go` (181 lines) - `fakeProfile` hand-built fake + `TestInterfaceSatisfaction` (case 5) + 4 registry-behavior tests (cases 1-4) + a 50-goroutine concurrent-read race test
- `chase/profile/doc.go` (85 lines) - Package doc comment resolving the EXPORTED-vs-internal decision gate and documenting the `Boundary()`/`Directives()` deferral

## Decisions Made
- Resolved ROADMAP.md's Objective-2 decision gate: `chase/*` stays EXPORTED (no `internal/` prefix), matching PROJECT.md differentiator #2 ("library-first & server-native") — advanced consumers and future `profiles/paged`/`article`/`epub` must be able to import `chase/*` directly, not be forced through `press/`.
- Chose last-wins REPLACE for duplicate-ID `Register()` (over reject-with-error) since re-registration is a legitimate test/hot-reload occurrence, not a programmer error; tested that replacing an existing ID never changes which ID `Default()` resolves to.
- `Default()` returns `nil` (not panic) when the registry is empty — simplest option that satisfies the required test-list cases without introducing an untested crash path.
- Deferred `Boundary()`/`Directives()` per the TRD's locked anti-pattern (no consumer in this objective) — documented the rationale in `doc.go` rather than silently omitting them with no record.

## Deviations from Plan

None - TRD executed exactly as written. All 6 Profile methods, the registry semantics, and the decision-gate documentation match the TRD's `codebase_examples` suggested interface and `anti_patterns` constraints with no structural changes.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. Pure Go package, zero new dependencies (uses only the already-pinned `chase/theme` internal import + stdlib `sync`/`testing`). `go.mod`/`go.sum` untouched — no `go mod tidy` run.

## Next Objective Readiness
- `chase/profile` is fully unit-tested and ready for 02-03 (`profiles/slides` implements `Profile` and calls `Register`) and 02-04 (the chase entrypoint calls `profile.Get`/`profile.Default`).
- No blockers. The one-way `chase/profile` -> `chase/theme` import edge is verified (whole-repo `go build ./...` succeeds with no cycle); `chase/theme` was not modified by this TRD.
- If 02-03's bottom-up validation reveals a missing or mis-shaped method, this is expected (per the TRD's error_recovery) and should be added there with its concrete call-site — not treated as a defect in this TRD.

## Self-Check: PASSED

All created-file claims and commit-hash claims verified against disk/git before finalizing this summary:

- FOUND: `chase/profile/profile.go`
- FOUND: `chase/profile/registry.go`
- FOUND: `chase/profile/registry_test.go`
- FOUND: `chase/profile/doc.go`
- FOUND: `400ba71`, `283451b`, `9940725`, `5a8acd3`, `c2c6e51` (all 5 in `git log --oneline --all`)

---
*Objective: 02-model-profile*
*Completed: 2026-07-21*
