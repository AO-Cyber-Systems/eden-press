---
objective: 00-conformance-corpus-attribution
job: "01"
subsystem: infra
tags: [go-module, licensing, attribution, addlicense, github-actions, ci, upstream-drift, marp]

# Dependency graph
requires: []
provides:
  - "Buildable Go module github.com/AO-Cyber-Systems/eden-press (go 1.25) — the root every other TRD sits on"
  - "Complete, hand-authored, version-pinned go.mod + go.sum (goldmark v1.8.4, golang.org/x/net v0.57.0, tdewolff/parse/v2 v2.8.13) — single source of truth for the whole objective; downstream is additive-only (go mod download, NEVER go mod tidy)"
  - "Conformance/tools/themes directory skeleton (conformance/{corpus,spec,htmldiff,cssdiff,runner}, tools/corpus-gen, themes)"
  - "Day-one licensing: LICENSE (MIT, AO Cyber Systems 2026), NOTICE (Marp x3 + Go deps + spec sources), README not-affiliated disclaimer"
  - "CI-enforced per-file MIT header mechanism via google/addlicense (both Eden-2026 and verbatim-Marp-2018 templates documented)"
  - "ci.yml acceptance gate (go build/vet/test + addlicense -check) that later conformance packages plug into"
  - "Scheduled upstream-drift.yml mechanism filing a deduplicated GitHub issue when a Marp repo publishes a newer release"
  - "Per-PR NOTICE-update checklist (PULL_REQUEST_TEMPLATE.md + CONTRIBUTING.md)"
affects: [00-02, 00-03, 00-04, 00-05, 00-06, "all engine objectives (import conformance/*, share go.mod)"]

# Tech tracking
tech-stack:
  added:
    - "github.com/yuin/goldmark v1.8.4 (declared, pinned)"
    - "golang.org/x/net v0.57.0 (declared, pinned)"
    - "github.com/tdewolff/parse/v2 v2.8.13 (declared, pinned)"
    - "github.com/google/addlicense v1.2.0 (dev tool, CI-installed)"
  patterns:
    - "Authoritative-manifest: go.mod/go.sum owned by 00-01, downstream additive-only (go mod download, never go mod tidy) to avoid parallel-worktree require pruning"
    - "addlicense -check acceptance gate; two header templates (Eden MIT 2026 vs preserved Marp 2018)"
    - "Mechanical upstream-drift (scheduled Action + dedup issue) instead of a prose reminder"

key-files:
  created:
    - go.mod
    - go.sum
    - doc.go
    - .gitignore
    - LICENSE
    - NOTICE
    - README.md
    - CONTRIBUTING.md
    - UPSTREAM-VERSIONS.txt
    - themes/README.md
    - .github/workflows/ci.yml
    - .github/workflows/upstream-drift.yml
    - .github/PULL_REQUEST_TEMPLATE.md
    - conformance/corpus/.gitkeep
    - conformance/spec/.gitkeep
    - conformance/htmldiff/.gitkeep
    - conformance/cssdiff/.gitkeep
    - conformance/runner/.gitkeep
    - tools/corpus-gen/.gitkeep
  modified: []

key-decisions:
  - "go.mod/go.sum hand-authored complete and pinned here as the objective's single source of truth; NO go mod tidy anywhere downstream (would prune sibling-branch requires and break the merge / 00-01 CI gate)"
  - "Copyright holder = 'AO Cyber Systems' for LICENSE + all Eden per-file headers; Marp verbatim assets keep 'Copyright (c) 2018 Marp team (marp-team@marp.app)' (year 2018 NOT relabeled 2026)"
  - "Workflow YAML files stamped with the Eden MIT header so the ci.yml step 'addlicense -check .' is green repo-wide (addlicense recognizes .yml)"
  - "Marp pins re-confirmed 2026-07-20 (marpit v3.2.2, marp-core v4.4.0, marp-cli v4.5.0) — matched latest, no update needed"

patterns-established:
  - "Authoritative-manifest / additive-only module ownership for parallel waves"
  - "addlicense two-template header enforcement (Eden-original vs verbatim-Marp)"
  - "Scheduled, deduplicated upstream-drift issue mechanism"

requirements-completed: [LIC-01, LIC-02, LIC-03, LIC-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: true

# Metrics
duration: 6min
completed: 2026-07-20
---

# Objective 00 TRD 01: Repository Scaffolding + Attribution Bootstrap Summary

**Buildable `github.com/AO-Cyber-Systems/eden-press` Go module (go 1.25) with a complete pinned go.mod/go.sum single-source-of-truth, day-one MIT licensing + full Marp/Go/spec attribution, addlicense-enforced per-file headers, a per-PR NOTICE checklist, and a scheduled Marp upstream-drift CI mechanism.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-20T22:47:30Z
- **Completed:** 2026-07-20T22:53:32Z
- **Tasks:** 3
- **Files modified:** 19 created

## Accomplishments
- Hand-authored a complete, version-pinned `go.mod` + full `go.sum` (goldmark v1.8.4, golang.org/x/net v0.57.0, tdewolff/parse/v2 v2.8.13) via `go mod download` — **no `go mod tidy`** — as the objective's single source of truth; a fresh `go build ./...` / `go vet ./...` / `go test ./...` all pass clean.
- Established the conformance/tools/themes directory skeleton (git-trackable via `.gitkeep`/README placeholders) so downstream TRDs 00-02..00-06 drop into a stable tree.
- Delivered day-one licensing/attribution: LICENSE (MIT, © 2026 AO Cyber Systems), NOTICE (Marpit/Marp Core/Marp CLI with verified 2018 copyright lines + goldmark/chroma/tdewolff/parse + latex2mathml/go-latex deferred + CommonMark/cmark-gfm spec sources with TODO markers for 00-04/00-05), and a README not-affiliated/endorsed disclaimer.
- Stood up the attribution **mechanism** (not just files): `addlicense -check` CI gate, a two-template header discipline in CONTRIBUTING (Eden 2026 vs preserved Marp 2018), a per-PR NOTICE-update checklist, and a weekly scheduled `upstream-drift.yml` that files a deduplicated `upstream-drift` issue on Marp release drift.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Go module + directory skeleton | `go build ./... && go vet ./...` ; `head -1 go.mod` ; `grep "^go 1.25" go.mod` ; `test -s go.sum` ; `grep` all 3 pinned deps | 0 | PASS |
| 2: LICENSE + NOTICE + README | `grep "AO Cyber Systems" LICENSE` ; `grep "MIT" LICENSE` ; `grep -i "not affiliated" NOTICE README.md` ; `grep "marp-team@marp.app" NOTICE` ; `grep -i Marpit/"Marp Core"/"Marp CLI"/goldmark NOTICE` | 0 | PASS |
| 3: Attribution enforcement + CI workflows | `addlicense -l mit -s -c "AO Cyber Systems" -check .` ; `grep addlicense/"go test ./..." ci.yml` ; `grep schedule:/"gh issue create" upstream-drift.yml` ; `grep -i NOTICE PR-template` ; `grep marp-team/marpit UPSTREAM-VERSIONS.txt` ; `actionlint` ; `yamllint` | 0 | PASS |

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize Go module + directory skeleton** — `e3be19d` (feat)
2. **Task 2: LICENSE + NOTICE + README (LIC-01, LIC-02, LIC-04)** — `43ddf5d` (feat)
3. **Task 3: Attribution enforcement mechanism + CI workflows (LIC-03, drift CI)** — `a6e784a` (feat)

**Plan metadata:** committed with SUMMARY.md + STATE/ROADMAP updates (docs commit).

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| lint | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (`[no test files]` — not a failure) | 0 | PASS |
| header-check | `addlicense -l mit -s -c "AO Cyber Systems" -check .` | 0 | PASS |
| actionlint | `actionlint .github/workflows/*.yml` | 0 | PASS |
| yamllint | `yamllint .github/workflows/*.yml` (relaxed line-length) | 0 | PASS |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 6/6 (buildable module; LICENSE+NOTICE+README exist with disclaimer; `addlicense -check` green on all Go files; scheduled dedup drift workflow; per-PR NOTICE checklist; go.mod+go.sum complete/pinned with no tidy)
- **Gate failures:** None

## TDD Exceptions

TRD carried a `TDD-EXCEPTION` marker: repo scaffolding, licensing paperwork, and CI/config with no business logic to unit-test. Verified via build/vet/test + grep + `addlicense -check` + `actionlint`/`yamllint` instead of RED/GREEN/REFACTOR. Appropriate — there is no behavioral unit to drive with a test.

## Files Created/Modified
- `go.mod` / `go.sum` — module path + go 1.25 floor + complete pinned require block (3 deps); go.sum populated by `go mod download` (authoritative, additive-only downstream)
- `doc.go` — `package edenpress` root doc, Eden MIT header + not-affiliated note
- `.gitignore` — build binaries, `*.wasm`, `/node_modules/`, coverage, OS cruft
- `LICENSE` — MIT, Copyright (c) 2026 AO Cyber Systems
- `NOTICE` — Marpit/Marp Core/Marp CLI (2018 lines) + goldmark/chroma/tdewolff/parse + latex2mathml/go-latex (deferred) + CommonMark/cmark-gfm spec sources (TODO markers for 00-04/00-05)
- `README.md` — project intro + `## Acknowledgments` with explicit not-affiliated/endorsed disclaimer
- `CONTRIBUTING.md` — two header templates (Eden 2026 / verbatim Marp 2018), additive-only "no `go mod tidy`" rule, new-vendored-asset checklist
- `UPSTREAM-VERSIONS.txt` — pinned marpit v3.2.2 / marp-core v4.4.0 / marp-cli v4.5.0
- `themes/README.md` — placeholder; Marp assets land in Objective 3
- `.github/workflows/ci.yml` — setup-go 1.25 + build/vet/test + `addlicense -check .`
- `.github/workflows/upstream-drift.yml` — weekly cron + `workflow_dispatch`, matrix over 3 Marp repos, deduplicated `upstream-drift` issue (permissions `issues: write`)
- `.github/PULL_REQUEST_TEMPLATE.md` — NOTICE-update + verbatim-header + no-tidy checkboxes
- `conformance/{corpus,spec,htmldiff,cssdiff,runner}/.gitkeep`, `tools/corpus-gen/.gitkeep` — skeleton placeholders

## Decisions Made
- **go.mod/go.sum ownership:** hand-authored complete + pinned here; downstream additive-only (`go mod download`, never `go mod tidy`). A `require` with no importing `.go` file yet is expected and left in place. Documented in CONTRIBUTING + PR template.
- **Copyright years:** Eden = 2026 "AO Cyber Systems"; Marp verbatim assets preserve 2018 "Marp team (marp-team@marp.app)" — not relabeled.
- **Marp pins:** re-confirmed against `gh release view` on 2026-07-20 — marpit v3.2.2 / marp-core v4.4.0 / marp-cli v4.5.0 all matched latest, so no pin update was needed (constraint #6 satisfied).

## Deviations from Plan

None that alter scope. Two implementation refinements worth recording (no user decision required):

1. **[Rule 3 - Blocking] `addlicense` false-positive on doc.go's package comment.** `addlicense`'s header detector treats any file containing the word "copyright" in its first 1000 bytes as already-licensed; `doc.go`'s package doc mentions "Marp copyright", so `addlicense` refused to stamp it. Fix: captured `addlicense`'s exact `-s` MIT output on a throwaway file and prepended that canonical header block above the package comment, then confirmed `addlicense -check doc.go` exits 0. The header text is the load-bearing artifact and is byte-identical to what the tool generates, so CI `-check` is genuinely green (not a detector false-positive). Committed in `e3be19d`.
2. **[Rule 3 - Blocking] Workflow YAML flagged by repo-wide `addlicense -check .`.** `addlicense` recognizes `.yml`, so the two workflow files failed the `ci.yml` step `addlicense -l mit -s -c "AO Cyber Systems" -check .`. Rather than narrow the check (which the TRD/constraint fixes as `.`), stamped both workflow files with the Eden MIT header (`#`-comment form) so the exact specified command is green repo-wide. Committed in `a6e784a`.

---

**Total deviations:** 2 auto-fixed (2 blocking). **Impact on plan:** none — both keep the TRD-specified verify commands (`addlicense -check .`) intact and passing; no scope change. `addlicense` **was** successfully installed (v1.2.0) and run, so no header hand-authoring deferral was needed for CI enforcement.

## Issues Encountered
- Both issues above were the `addlicense` header-detection quirks; both resolved in-task. No network/proxy issues (`go mod download` succeeded; go.sum hashes for goldmark + x/net match the prior spike, and tdewolff/parse resolved from the module cache).

## User Setup Required
None — no external service configuration required. Note: `upstream-drift.yml` needs an `upstream-drift` label to exist in the GitHub repo for `gh issue create --label` to succeed on first drift; the workflow will surface a clear error if the label is missing, and this is a one-line repo-settings step deferred to whenever the repo is pushed to GitHub (out of scope for this local worktree).

## Next Objective Readiness
- The module + acceptance-gate skeleton is ready; TRDs 00-02..00-06 can proceed in parallel, all sharing this authoritative go.mod/go.sum (additive-only). The `conformance/*` package tree exists for htmldiff/cssdiff/runner/corpus/spec to fill.
- NOTICE carries explicit `TODO(00-04)` / `TODO(00-05)` markers for exact spec tags + example counts + mined corpus counts — this deliberately dogfoods the per-PR NOTICE-update checklist established here.
- Reminder for downstream executors: **never run `go mod tidy`** in a parallel worktree; add modules with a pinned `require` + `go mod download` only.

## Self-Check: PASSED

All 20 claimed files exist on disk; all 3 task commits (`e3be19d`, `43ddf5d`, `a6e784a`) exist in git history. No missing items.

---
*Objective: 00-conformance-corpus-attribution*
*Completed: 2026-07-20*
