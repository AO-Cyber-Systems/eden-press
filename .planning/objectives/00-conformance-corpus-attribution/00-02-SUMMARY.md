---
objective: 00-conformance-corpus-attribution
job: "02"
subsystem: testing
tags: [go, goldmark, x-net-html, dom-diff, conformance, corpus, commonmark, gfm, marp]

# Dependency graph
requires:
  - objective: 00-01
    provides: authoritative go.mod/go.sum (goldmark v1.8.4, x/net v0.57.0), doc.go MIT header, addlicense + drift CI
provides:
  - "htmldiff.Equal — DOM-normalized HTML diff with a narrow, named cosmetic allow-list (CONF-02)"
  - "corpus.Case + corpus.LoadCases — golden-corpus on-disk case format and loader (requires_engine + optional expected.css)"
  - "report.SectionReport — per-section pass/fail/pending aggregator (Add/AddPending/Render/Summary)"
  - "runner.RenderFunc + runner.RunCase — pluggable engine seam wiring corpus -> htmldiff -> report"
  - "runner.NewGoldmark/NewGoldmarkGFM/NewGoldmarkMarp + GoldmarkRenderFunc — the SINGLE shared goldmark constructor set"
affects: [00-04, 00-05, 00-06]

# Tech tracking
tech-stack:
  added: []  # go.mod owned by 00-01; this job is additive-only and adds NO new module requires
  patterns:
    - "Lift-and-harden: normalize()/walk() lifted verbatim from the md-spike prior art, then hardened + documented"
    - "Narrow named allow-list (not ignore-all-whitespace); <pre>/<code> compared verbatim; negative tests guard the boundary"
    - "Single shared engine constructor (engine.go) so parallel wave-3 jobs never duplicate-symbol a local newGM()"

key-files:
  created:
    - conformance/htmldiff/normalize.go
    - conformance/htmldiff/normalize_test.go
    - conformance/corpus/corpus.go
    - conformance/corpus/corpus_test.go
    - conformance/report/report.go
    - conformance/report/report_test.go
    - conformance/runner/runner.go
    - conformance/runner/engine.go
  modified: []

key-decisions:
  - "NewGoldmarkMarp uses extension.GFM (per TRD contract) rather than the spike's raw Table+Strikethrough — GFM is the superset markdown-it approximation the shared contract mandates"
  - "Pending results (AddPending) are excluded from pass/total and never mark a section failing — pending != failure"
  - "RunCase records a render error as a section failure and surfaces it through the diff string"

patterns-established:
  - "DOM-normalized comparison via golang.org/x/net/html canonical token stream (no hand-rolled tokenizer)"
  - "Corpus case = <id>/{input.md, options.json, expected.html[, expected.css]}; options.json carries optional requires_engine"

requirements-completed: [CONF-02]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 9min
completed: 2026-07-20
---

# Objective 0 TRD 02: Conformance-Harness Core Primitives Summary

**DOM-normalized HTML diff (htmldiff.Equal) with a proven-narrow cosmetic allow-list, a golden-corpus case format + loader, a per-section pass/fail/pending report, and the SINGLE shared goldmark engine constructor both wave-3 runners import — the reusable spine of the Eden Press conformance gate.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-07-20T22:59:37Z
- **Completed:** 2026-07-20T23:08:00Z
- **Tasks:** 3 (all TDD: RED -> GREEN)
- **Files created:** 8 (.go)

## Accomplishments
- CONF-02 satisfied: `htmldiff.Equal(expected, actual) (bool, diff)` treats ONLY the four named cosmetic classes (void-element syntax `<br>`/`<hr>`, attribute order, inter-block whitespace) as equal, and a NEGATIVE test suite (broken code-span `a<b` vs `a>b`, altered `<pre>` content, `<em>` vs `<strong>`, differing attr values) proves the allow-list does not mask real bugs.
- `<pre>`/`<code>` text compared VERBATIM via the `inPre` flag lifted from the spike — the whitespace-significant boundary the allow-list must not cross.
- Golden-corpus format defined and documented: `corpus.LoadCases` reads `<id>/{input.md, options.json, expected.html[, expected.css]}` into a typed `corpus.Case`, with the optional `requires_engine` field ("" | commonmark | marpit | marp-core) so runners can mark unbuildable cases pending.
- `report.SectionReport` renders a per-section breakdown (`Lists: 12/13`), not just an aggregate; `Summary()` returns aggregate pass/total plus the failing-section count; pending cases never inflate totals or mark a section failing.
- `conformance/runner` builds as a package (non-test `runner.go` + `engine.go`) exposing the pluggable `RenderFunc` seam, `RunCase`, and the ONE shared goldmark constructor set — so 00-04 and 00-05 import `NewGoldmark`/`NewGoldmarkGFM`/`NewGoldmarkMarp` instead of each defining `newGM()` (no duplicate-symbol on merge).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: DOM-normalized HTML diff (CONF-02) | `go test ./conformance/htmldiff/ -v` | 0 | PASS |
| 1: vet | `go vet ./conformance/htmldiff/` | 0 | PASS |
| 2: Corpus format + loader | `go test ./conformance/corpus/ -v` | 0 | PASS |
| 2: doc | `go doc ./conformance/corpus` (schema shown) | 0 | PASS |
| 3: Report + runner + engine | `go test ./conformance/report/ -v` | 0 | PASS |
| 3: runner build | `go build ./conformance/runner/` | 0 | PASS |
| 3: runner doc | `go doc ./conformance/runner` (NewGoldmark/NewGoldmarkGFM/NewGoldmarkMarp/GoldmarkRenderFunc/RenderFunc/RunCase) | 0 | PASS |
| 3: vet all | `go vet ./conformance/...` | 0 | PASS |

## Task Commits

Each task committed atomically (TDD RED then GREEN):

1. **Task 1: DOM-normalized HTML diff (CONF-02)** — `426ed85` (test, RED) -> `400d817` (feat, GREEN)
2. **Task 2: Golden-corpus case format + loader** — `d1ab580` (test, RED) -> `d3103c4` (feat, GREEN)
3. **Task 3: Report + runner seam + shared goldmark engine** — `8c89085` (test, RED) -> `df8399f` (feat, GREEN)

**Plan metadata:** committed with SUMMARY.md + STATE.md + ROADMAP.md.

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| test | `go test ./conformance/...` | 0 | PASS |
| lint | `go vet ./conformance/...` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (htmldiff) | `go test ./conformance/htmldiff/` | non-zero (build failed: `undefined: Equal`) | FAIL (correct) |
| GREEN (htmldiff) | `go test ./conformance/htmldiff/` | 0 | PASS (correct) |
| RED (corpus) | `go test ./conformance/corpus/` | non-zero (build failed: `undefined: LoadCases`) | FAIL (correct) |
| GREEN (corpus) | `go test ./conformance/corpus/` | 0 | PASS (correct) |
| RED (report) | `go test ./conformance/report/` | non-zero (build failed: `undefined: New`) | FAIL (correct) |
| GREEN (report) | `go test ./conformance/report/` | 0 | PASS (correct) |

CONF-02 negative sub-tests (part of the htmldiff GREEN run) all PASS — i.e. every deliberately-broken case is reported as a DIFF:
`code_span_inner_text_changed`, `pre_content_leading-space_altered`, `pre_content_line_changed`, `different_element_em_vs_strong`, `attribute_VALUE_differs`, `text_content_differs`.

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 6/6 (all `must_haves.truths` satisfied — narrow allow-list, verbatim pre/code, negative-test guard, corpus loader, per-section report, single shared engine)
- **Gate failures:** None

## Files Created/Modified
- `conformance/htmldiff/normalize.go` — lifted+hardened `normalize()`/`walk()` + exported `Equal(expected, actual) (bool, diff)` with human-readable canonical-stream diff
- `conformance/htmldiff/normalize_test.go` — cosmetic-equivalence happy cases + void-element named assertion + CONF-02 negative suite
- `conformance/corpus/corpus.go` — `Case` struct + `LoadCases`; on-disk schema documented as the corpus-format spec
- `conformance/corpus/corpus_test.go` — inline `t.TempDir()` fixtures (no_llm_test_data); populated-fields, optional CSS, requires_engine default, error paths
- `conformance/report/report.go` — `SectionReport` (`New`/`Add`/`AddPending`/`Render`/`Summary`), sorted deterministic output
- `conformance/report/report_test.go` — per-section render, summary counts, pending semantics, empty-report no-panic
- `conformance/runner/runner.go` — package doc + `RenderFunc` type + `RunCase` (non-test, so runner builds before wave-3 adds `_test.go`)
- `conformance/runner/engine.go` — `NewGoldmark`/`NewGoldmarkGFM`/`NewGoldmarkMarp` + `GoldmarkRenderFunc` (the ONE goldmark builder)

## Decisions Made
- `NewGoldmarkMarp` uses `extension.GFM` (+ `WithUnsafe` + `WithHardWraps`) per the TRD shared-engine contract, superseding the spike's raw `Table`+`Strikethrough` — GFM is the markdown-it superset the contract mandates and what 00-05 will import.
- Pending (`AddPending`) is a third outcome distinct from pass/fail: excluded from pass/total, never marks a section failing (a case awaiting an unbuilt engine is not a regression).
- Every new `.go` file carries the exact Eden Press MIT header copied from the committed `doc.go` (addlicense -check stays green).

## Deviations from Plan

None to the task scope — all 3 tasks executed exactly as written (lift-and-harden, corpus loader, report+runner+engine), each TDD RED then GREEN.

### Environment note (go.mod/go.sum — intentionally NOT committed)
- **Observed:** With the local Go 1.26.4 toolchain, building the module surfaced a missing `go.sum` entry for `github.com/tdewolff/test v1.0.12/go.mod` (a transitive go.mod of `tdewolff/parse/v2`, which is a `require` in 00-01's go.mod but is imported only by the sibling parallel job 00-03). The toolchain also wants to normalize the `go 1.25` directive to `go 1.25.0`.
- **Handling (per TRD error_recovery + hard constraint #2):** Ran `go mod download` ONLY (never `go mod tidy`, which would prune 00-03's requires) to make the local build/test resolve, and produced all verify evidence. Per constraint #2, `go.mod`/`go.sum` are owned by 00-01 and were kept OUT of every commit — verified with `git show --stat` on each commit (no `go.mod`/`go.sum`). The working-tree go.sum delta (one additive checksum line) is discarded and not merged; it belongs to 00-03/00-01, not this job.
- **Impact:** None on this job's deliverables. Flag for the orchestrator: 00-01's committed go.sum is missing the `tdewolff/test` go.mod checksum, so a fresh `-mod=readonly` build of the whole module needs `go mod download` until 00-03 (the tdewolff/parse consumer) lands its go.sum line.

## Issues Encountered
- The `-mod=readonly` build failure described above. Resolved with `go mod download` (additive, non-pruning) as the TRD explicitly directs; no `go mod tidy` was ever run.

## Next Objective Readiness
- **00-04 (CommonMark + GFM spec sweep):** import `runner.NewGoldmark`/`NewGoldmarkGFM`, `runner.RunCase`, `report.SectionReport` — do NOT define a local `newGM()`.
- **00-05 (Marp golden-corpus runner):** import `runner.NewGoldmarkMarp` + `runner.GoldmarkRenderFunc`, load cases via `corpus.LoadCases`, mark `RequiresEngine != ""` cases pending via `report.AddPending`.
- **00-06 (CSS-AST diff):** parallels htmldiff on the CSS side; `report.SectionReport` is reusable as-is.

## Self-Check: PASSED

- All 8 created `.go` files exist on disk (verified with `[ -f ]`).
- All 6 task commits exist in git history (426ed85, 400d817, d1ab580, d3103c4, 8c89085, df8399f).
- `go build ./...`, `go test ./conformance/...`, `go vet ./conformance/...` all exit 0.
- Every new `.go` file carries the Eden MIT header; `go.mod`/`go.sum` NOT in any commit; no `go mod tidy` run.

---
*Objective: 00-conformance-corpus-attribution*
*Completed: 2026-07-20*
