---
objective: 03-press-batteries-api
trd: "03"
subsystem: markdown-render
tags: [goldmark, gfm, strikethrough, node-renderer, heading-slug, chase-markdown]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "03-01: chase/markdown.NewEngine(extra ...goldmark.Option) extra-opts hook + chase/markdown.NewEngine's already-baked-in extension.GFM/ghtml.WithHardWraps/parser.WithAutoHeadingID; press package skeleton (options.go)"
provides:
  - "press.strikethroughOption() -- a self-contained goldmark.Option overriding GFM strikethrough to render <s>...</s> instead of goldmark's default <del>...</del>, via a renderer.NodeRenderer registered at priority 100 (< goldmark's own StrikethroughHTMLRenderer priority 500)"
  - "Regression-locked proof that CORE-03's tables/hard-breaks and CORE-04's heading-ID slugs (h1-h6, with dedup) already hold in chase/markdown.NewEngine with zero new wiring"
affects: [03-09-press-render-compose]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Priority-ordered renderer.NodeRenderer override: goldmark's renderer.Render sorts NodeRenderers ascending by priority then dispatches in REVERSE, so the LOWEST priority number for a given ast.NodeKind always registers last and wins ('last-write-wins'). A priority < 500 registration for extast.KindStrikethrough beats goldmark's own extension.StrikethroughHTMLRenderer (priority 500) without touching chase/markdown's own NodeRenderer (which never registers KindStrikethrough)."
    - "Verify-only regression test file (gfm_verify_test.go) with zero production code, added purely to lock already-shipped upstream behavior (extension.GFM tables, ghtml.WithHardWraps, parser.WithAutoHeadingID slugs) against silent future regression in chase/markdown.NewEngine."

key-files:
  created:
    - press/strikethrough.go
    - press/strikethrough_test.go
    - press/gfm_verify_test.go
  modified: []

key-decisions:
  - "strikethroughPriority = 100 (matches the STACK.md reference pattern and chase/markdown's own nodeRenderer convention of priority 0) -- any value < 500 would work; 100 was chosen for readability/convention, not because a narrower value was required."
  - "Verify-only tests (Task 2) build the engine WITH strikethroughOption() applied (markdown.NewEngine(strikethroughOption())), not a bare NewEngine() -- since 03-09 will always fold this option into press.Render's engine, testing the exact composed configuration is the more faithful regression target, and the TRD's own action text allows either ('these features predate the override')."

patterns-established:
  - "goldmark NodeRenderer override via util.Prioritized + renderer.WithNodeRenderers: the reusable shape for any future press battery that needs to override a single AST-node's HTML rendering without forking or wrapping chase/markdown's own Extender."

requirements-completed: [CORE-03, CORE-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 12min
completed: 2026-07-21
---

# Objective 3 TRD 03: Strikethrough `<s>` Override + GFM/Slug Verify Summary

**A priority-100 `renderer.NodeRenderer` override makes GFM `~~strike~~` render as `<s>strike</s>` (Marp parity) instead of goldmark's default `<del>`, plus regression tests locking in CORE-03's tables/hard-breaks and CORE-04's heading-ID slugs (already baked into `chase/markdown.NewEngine`).**

## Performance

- **Duration:** 12 min (TRD read to Task 2 commit)
- **Started:** 2026-07-21T13:58:00Z
- **Completed:** 2026-07-21T14:00:28Z
- **Tasks:** 2/2 complete
- **Files modified:** 3 (all newly created)

## Accomplishments
- `press/strikethrough.go`: `strikethroughOption()` returns a `goldmark.Option` wrapping a custom `sRenderer` that registers `extast.KindStrikethrough` (from `github.com/yuin/goldmark/extension/ast`) at priority 100 via `renderer.WithNodeRenderers(util.Prioritized(&sRenderer{}, 100))` -- last-write-wins beats goldmark's own `StrikethroughHTMLRenderer` (priority 500, source-verified against `goldmark@v1.8.4/extension/strikethrough.go`).
- Proven both directions: `markdown.NewEngine(strikethroughOption())` renders `~~gone~~` as `<s>gone</s>` with no `<del>` present; the bare `markdown.NewEngine()` (no option) still renders the goldmark default `<del>gone</del>` -- confirming the option itself is what flips behavior, not ambient engine state.
- Proven the override composes cleanly: a fixture mixing a GFM table, a strikethrough span, and a soft-broken paragraph renders `<table>` + `<br>` + `<s>gone</s>` together, with no `<del>` leaking in.
- `press/gfm_verify_test.go`: 5 pure regression tests (zero new production code) lock CORE-03's tables (`<table>`/`<th>`/`<td>`) and hard-breaks (`<br>` via `ghtml.WithHardWraps()`), and CORE-04's heading slugs on both h1 (`# Hello World` -> `id="hello-world"`) and h6 (`###### Deep` -> `id="deep"`), plus slug dedup (`# Hello` twice -> `id="hello"` then `id="hello-1"`, goldmark's `parser.IDs.Generate` collision-suffix behavior, source-verified against `goldmark@v1.8.4/parser/parser.go`).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: strikethrough `<s>` override option | `go test ./press/ -run TestStrikethrough -v && gofmt -l press/strikethrough.go` | 0 | PASS |
| 2: CORE-03/CORE-04 verify-only regression tests | `go test ./press/ -run 'TestGFM\|TestSlug\|TestHardWrap' -v` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: strikethrough `<s>` override option (RED->GREEN)** - `f658442` (feat)
2. **Task 2: CORE-03/CORE-04 verify-only regression tests** - `be6774d` (test)

_Note: Task 1 is `tdd="true"` -- RED (compile failure: `undefined: strikethroughOption`, both call sites) confirmed before the GREEN implementation. Task 2 is also `tdd="true"` per TRD frontmatter, but per the task's own action text it adds ZERO new production code -- the features under test (`extension.GFM`, `ghtml.WithHardWraps`, `parser.WithAutoHeadingID`) already existed in `chase/markdown.NewEngine` before this TRD, so all 5 tests pass on first run with no RED phase possible or expected; see TDD Evidence below for the honest accounting of this._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build (TRD-scoped) | `go build ./press/...` | 0 | PASS |
| vet (TRD-scoped) | `go vet ./press/...` | 0 | PASS |
| test (TRD-scoped) | `go test ./press/...` | 0 | PASS |
| build (whole-repo) | `go build ./...` | 0 | PASS |
| vet (whole-repo) | `go vet ./...` | 0 | PASS |
| test (whole-repo) | `go test ./...` | 0 | PASS (15 packages, incl. chase/, conformance/, press/, profiles/slides/) |
| gofmt | `gofmt -l .` | 0 (no output) | PASS |
| addlicense (new files) | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -check press/strikethrough.go press/strikethrough_test.go press/gfm_verify_test.go` | 0 (no output) | PASS |
| Obj-1 corpus/cssdiff | `go test ./conformance/...` | 0 | PASS (corpus, cssdiff, htmldiff, report, runner all ok) |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS |
| no-chromedp invariant | `go list -deps ./... \| grep -c chromedp` | 0 (count) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./press/ -run TestStrikethrough -v` | 1 (compile failure: `undefined: strikethroughOption` at both call sites in strikethrough_test.go) | FAIL (correct) |
| GREEN (Task 1) | `go test ./press/ -run TestStrikethrough -v` | 0 (all 3 tests pass) | PASS (correct) |
| Task 2 (verify-only, no RED possible) | `go test ./press/ -run 'TestGFM\|TestSlug\|TestHardWrap' -v` | 0 (all 5 tests pass on first write) | PASS -- documented exception below |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 3/3 (all `must_haves.truths` from 03-03-TRD.md frontmatter: `<s>`-over-`<del>` priority override; the option is self-contained/collision-free; tables+hard-breaks+slugs verified against the chase engine)
- **Gate failures:** None

## TDD Exceptions

**Task 2 (`press/gfm_verify_test.go`) had no RED phase, by design.** The TRD's own task action text states this explicitly: "GOTCHA: no production code here -- these are pure regression tests over already-baked-in NewEngine features." `extension.GFM` (tables), `ghtml.WithHardWraps()` (hard breaks), and `parser.WithAutoHeadingID()` (heading slugs) were all already wired into `chase/markdown.NewEngine` before this TRD (`chase/markdown/seam.go:62`, shipped in Objective 1). There is no failing state to drive toward -- the tests exist purely as a regression lock, and all 5 passed immediately on first write. This matches the TRD's `tdd: true` frontmatter type in spirit (Test-list-first discipline was followed -- the Test list itself was written and read before any test code) while the individual task is intentionally a verify-only exception, exactly as the TRD instructs.

## Files Created/Modified
- `press/strikethrough.go` - `sRenderer` (renderer.NodeRenderer for `extast.KindStrikethrough` -> `<s>`/`</s>`) + `strikethroughOption()` (the composable `goldmark.Option`, priority 100)
- `press/strikethrough_test.go` - Test-list cases 1-3: override proof, default-baseline documentation, mixed-GFM-fixture non-collision proof
- `press/gfm_verify_test.go` - Test-list cases 4-5: GFM table render, hard-wrap `<br>`, h1/h6 heading slugs, slug dedup

## Decisions Made
- Priority 100 for the strikethrough override (not the bare minimum, e.g. 0) -- readability/convention match with the TRD's own reference pattern and STACK.md, not a functional requirement (any value < 500 satisfies the override).
- Task 2's verify tests build the engine WITH `strikethroughOption()` applied (`markdown.NewEngine(strikethroughOption())`) rather than a bare `markdown.NewEngine()`, since 03-09's `press.Render` will always compose the two together -- testing the actual composed configuration is the more faithful regression target for what ships.

## Deviations from Plan

None - TRD executed exactly as written. No Rule 1-4 deviations were triggered; both tasks matched their `<action>` specs on the first implementation pass.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- `press.strikethroughOption()` is ready for 03-09's `press.Render` to fold into `NewEngine(pressExtraOpts...)` alongside the other wave-2 battery options (emoji, chroma highlight, math, sanitize) -- it registers no `ast.NodeKind` any other planned battery is expected to touch, per the TRD's own collision-freedom must-have.
- CORE-03 (tables/hard-breaks/strikethrough) and CORE-04 (heading slugs) are both now regression-locked in `press/` itself, independent of `chase/markdown`'s own test suite -- a future `NewEngine` change that silently drops any of the three will fail `press/gfm_verify_test.go` immediately.
- No blockers for 03-04 (emoji) or any other wave-2 battery TRD; `go.mod` was not touched (per gotchas), and `chase/markdown`'s own files (`seam.go`, `render.go`) remain byte-for-byte untouched.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: press/strikethrough.go
- FOUND: press/strikethrough_test.go
- FOUND: press/gfm_verify_test.go
- FOUND commit: f658442 (Task 1)
- FOUND commit: be6774d (Task 2)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
