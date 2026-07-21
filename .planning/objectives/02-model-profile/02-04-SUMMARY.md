---
objective: 02-model-profile
job: "04"
subsystem: chase-entrypoint
tags: [go, goldmark, one-parse-two-sinks, tdd, capstone, corpus-smoke]

# Dependency graph
requires:
  - objective: 02-model-profile
    provides: "02-01's chase/model.Build (non-mutating Document builder), 02-02's chase/profile.Profile interface + registry, 02-03's profiles/slides (only Profile impl) + de-hardcoded chase/theme.Pack"
provides:
  - "chase/markdown.RenderDoc: render an ALREADY-parsed *ast.Document via defaultEngine, with zero internal re-parse — the seam that makes a single-parse guarantee possible"
  - "chase.Render(md string) (Output, error): the internal one-parse-two-sinks entrypoint — ONE markdown.Parse call forks to markdown.RenderDoc (HTML sink) and model.Build (Model sink) on the SAME finalized *ast.Document, plus profile-parameterized theme.Pack for CSS"
  - "chase.Output{HTML, CSS, Model, Meta} — the composed, JSON-serializable result MODEL-02 requires from a single parse pass"
  - "packCSS: the profile -> theme.Pack primitives bridge (unit element, scaffold/advanced-background CSS via Scaffold(bool) TrimPrefix recovery), keeping chase/theme free of any chase/profile import"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One parse, two sinks (ARCHITECTURE.md Pattern 1/Pattern 3): a single finalized *ast.Document forks to two independent consumers (RenderDoc, model.Build) instead of two Parse/Render passes — proven by a byte-identical HTML-before/after-Build assertion, not by instrumenting the parser"
    - "Profile.Scaffold(true) TrimPrefix recovery: profiles/slides.Scaffold(true) concatenates base scaffold CSS + advanced-background CSS into one string; packCSS recovers the advanced-background suffix via strings.TrimPrefix(p.Scaffold(true), scaffoldCSS) rather than adding a second speculative Profile method with no other call site (chase/profile/doc.go's no-speculative-superset stance)"
    - "svgEnabled(pc) mirrors the SAME markdown.SvgOptionsKey parser.Context value chase/markdown/inlinesvg.go's own transformer reads, instead of hardcoding InlineSVG:true, so packCSS's container-chain choice always matches how doc was actually rendered"
    - "Corpus smoke test as dual-proof: chase.Render on 4 real Objective-1 corpus decks (marp-basic, marp-slide-split, marp-paginate, marp-header-footer) both (a) htmldiff.Equal-matches the existing golden expected.html (no regression from adding the model/CSS sinks) AND (b) yields non-empty CSS + populated Model from the SAME call — proving one-parse-two-sinks holds for real decks, not just the hand-built fixture"

key-files:
  created:
    - chase/markdown/renderdoc.go
    - chase/markdown/renderdoc_test.go
    - chase/chase.go
    - chase/chase_test.go
  modified: []

key-decisions:
  - "RenderDoc's non-mutation/no-reparse proof uses a MATCHING doc/source pair (Section-count before/after Build + idempotent double-RenderDoc call), not a deliberately mismatched source — an earlier draft that passed a shorter/mismatched source panicked inside goldmark's text.Segment.Value (AST byte-offsets are source-slice-relative); the safer test achieves the identical TDD acceptance intent without violating RenderDoc's own documented 'same source' contract"
  - "packCSS needs only UnitElement()+Scaffold(bool) from Profile — Pagination() and Sizes()/sizeFallback have no consumer in chase/theme's actual Pack/NewThemeSet signatures for this internal entrypoint (Sizes() only feeds Theme.Load/ParseTheme's named-theme-loading path, which chase.Render never exercises — no go:embed themes, that is Objective 3, per this TRD's anti_patterns)"
  - "Only the reserved theme.ScaffoldThemeName identity is packed by chase.Render — no user-authored/embedded theme selection is in scope for this internal composer"

requirements-completed: [MODEL-02]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~14min
completed: 2026-07-21
---

# Objective 02 TRD 04: chase.go — one-parse-two-sinks entrypoint (HTML+CSS+Model) Summary

**`chase.Render(md)` now returns `{HTML, CSS, Model, Meta}` from a SINGLE `markdown.Parse` call — the finalized AST forks to `markdown.RenderDoc` (HTML) and `model.Build` (Model) on the exact same tree, CSS is packed via `theme.Pack` parameterized by `profiles/slides`, and the proof holds across both a hand-built multi-slide fixture and 4 real Objective-1 corpus decks with zero HTML regression.**

## Performance

- **Duration:** ~14 min (Task 1 commit to Task 3 commit: 02:36:36 -> 02:42:54 local; merge base 02:29:08)
- **Started:** 2026-07-21T06:29:08Z
- **Completed:** 2026-07-21T06:42:54Z
- **Tasks:** 3
- **Files touched:** 4 (all created); +523 lines

## Accomplishments
- `chase/markdown.RenderDoc(doc *ast.Document, source []byte) (string, error)` renders an already-parsed doc via the SAME `defaultEngine` `markdown.Render` uses internally, with zero internal re-parse — the missing seam MODEL-02 needed.
- `chase.Render(md string) (Output, error)` composes `markdown.Parse` (ONE parse) -> `markdown.RenderDoc` (sink 1: HTML) + `model.Build` (sink 2: Model) on the SAME `*ast.Document`, plus `packCSS` (profile-parameterized `theme.Pack`) — returning `Output{HTML, CSS, Model, Meta}` from one call.
- MODEL-02's acceptance criterion is proven structurally, not by instrumentation: HTML rendered from the shared doc before `model.Build` runs is byte-identical to HTML rendered from the same doc after `model.Build` runs (`TestOneParseTwoSinksHTMLUnaffectedByModelBuild`), confirming the model sink doesn't perturb the HTML sink and both are fed by a single parse.
- CSS is profile-parameterized, not hardcoded: `packCSS` bridges `profile.Profile.UnitElement()`/`Scaffold(bool)` into `theme.NewThemeSet`/`Pack`, and the packed CSS contains the profile-scoped `div.marpit > svg > foreignObject > section {` rule (inline-SVG container chain), proving `profiles/slides` fed `chase/theme`, not a constant.
- A corpus smoke test (`TestRenderCorpusSmoke`) runs 4 representative Objective-1 Marpit-mechanic corpus decks (`marp-basic`, `marp-slide-split`, `marp-paginate`, `marp-header-footer`) through `chase.Render`: each call's HTML matches the corpus's golden `expected.html` (no regression) AND its CSS/Model are populated from that SAME call — proving one-parse-two-sinks holds for real decks, not just the synthetic fixture.
- Objective 2 closes: all 4 success criteria under ROADMAP "Objective 2" are now satisfied across TRDs 02-01 (`model.Build`), 02-02 (`profile.Profile` + registry), 02-03 (`profiles/slides` + de-hardcoded `chase/theme`), and this TRD (the composing entrypoint).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Add markdown.RenderDoc | `go test ./chase/markdown/ -run RenderDoc -v` | 0 | PASS |
| 2: Implement chase.Render (one parse, two sinks) | `go build ./... && go test ./chase/ -v` | 0 | PASS |
| 3: Validate against Marp corpus subset | `go test ./chase/ -run Corpus -v && go test ./...` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (raw `git commit` never used):

1. **Task 1: Add markdown.RenderDoc — render an already-parsed document without re-parsing** - `8c10088` (feat)
2. **Task 2: Implement chase.Render — one parse, two sinks (HTML + CSS + Model)** - `9f0d142` (feat)
3. **Task 3: Validate the entrypoint against the Marp corpus** - `1e2629d` (test)

_Note: all 3 tasks are `tdd="true"` per the TRD. Task 1 and Task 2 followed a real RED (build/compile failure — `undefined: RenderDoc`, then `undefined: Render`) -> GREEN cycle, landing as single coherent commits once green. Task 3 is additive test-only code against the already-GREEN `Render` API; it went RED -> GREEN in one pass (no code-under-test changes needed) and is documented in TDD Evidence below via its first-run PASS._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` | 0 | PASS |

Additional evidence gathered beyond the TRD's 3 named gates:
- `gofmt -l .` — empty output, no formatting diffs across the whole repo.
- `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -check .` — exit 0, all 4 new files carry the Eden MIT header ending in `// SPDX-License-Identifier: MIT`.
- `go test ./profiles/slides/ -run TestGrepGate -v` — PASS (Objective-2 grep-gate: `chase/model`/`chase/theme` production code stays free of Slide-family identifiers, the quoted `"section"` literal, and size constants).
- `go test ./conformance/... -v` — PASS: `TestChaseCorpus` (Objective-1 corpus gate), `TestMarpCorpus`, `TestSpecSweep`, `conformance/cssdiff` and `conformance/htmldiff` unit suites — all green, unchanged from pre-TRD state.
- `go test ./chase/theme/... -v` — PASS, including the `pack_conformance_test.go` cssdiff-gate tests (golden fixtures untouched).

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./chase/markdown/ -run RenderDoc -v` (before renderdoc.go existed) | 1 (build failed: `undefined: RenderDoc`) | FAIL (correct) |
| GREEN (Task 1) | `go test ./chase/markdown/ -run RenderDoc -v` (after renderdoc.go) | 0 | PASS (correct) |
| RED (Task 2) | `go test ./chase/ -v` (chase_test.go present, chase.go absent) | 1 (build failed: `undefined: Render` x3) | FAIL (correct) |
| GREEN (Task 2) | `go test ./chase/ -v` (after chase.go) | 0 (4/4 tests PASS) | PASS (correct) |
| GREEN (Task 3) | `go test ./chase/ -run Corpus -v` (corpus smoke test, additive) | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 (single-parse-single-Parse-call structure; byte-identical HTML before/after Build; RenderDoc no-reparse seam; profile-parameterized CSS)
- **Gate failures:** None

## Files Created/Modified
- `chase/markdown/renderdoc.go` - `RenderDoc(doc *ast.Document, source []byte) (string, error)`: renders a pre-parsed doc via `defaultEngine.Renderer().Render`, no internal `Parse`.
- `chase/markdown/renderdoc_test.go` - Fixture-driven equality (`RenderDoc` == `Render`) across 4 markdown fixtures, plus a non-mutation/idempotency proof (`TestRenderDocDoesNotMutateOrReparse`).
- `chase/chase.go` - `Output{HTML, CSS, Model, Meta}`, `Render(md string) (Output, error)`, `packCSS(p profile.Profile, pc parser.Context) (string, error)`, `svgEnabled(pc parser.Context) bool` — the internal one-parse-two-sinks composer.
- `chase/chase_test.go` - 5 tests: Test-list cases 1-4 (`TestRenderReturnsHTMLCSSAndModelFromOneCall`, `TestOneParseTwoSinksHTMLUnaffectedByModelBuild`, `TestRenderHTMLMatchesStandaloneMarkdownRender`, `TestRenderCSSIsProfileParameterizedAndScoped`) plus Test-list case 6 (`TestRenderCorpusSmoke`).

## Decisions Made
- Bridged `Profile.Scaffold(inlineSVG bool) string`'s single-string return (base+advanced-background concatenated when `true`) into `theme.NewThemeSet`'s two separate string params via `strings.TrimPrefix(p.Scaffold(true), scaffoldCSS)`, rather than adding a new Profile method — `profiles/slides.Scaffold(true)` literally does `ScaffoldCSS + AdvancedBackgroundCSS`, so the trim is byte-exact, and `chase/profile/doc.go` explicitly discourages adding a method with no other consumer.
- Scoped `packCSS` to only `UnitElement()` + `Scaffold(bool)` — verified `Pagination()` has no consumer in `chase/theme`'s `Pack`/`NewThemeSet` (still hardcoded in `pass_pagination.go`) and `Sizes()` only feeds the named-theme-loading path (`Theme.Load`/`ParseTheme`), which this internal entrypoint never exercises (no `go:embed` themes — that's Objective 3).
- Read `svgEnabled` dynamically from `pc.Get(markdown.SvgOptionsKey)` rather than hardcoding `InlineSVG: true`, mirroring how `chase/markdown/inlinesvg.go`'s own transformer reads the same context key, even though `seam.go`'s `Parse` unconditionally enables it today.

## Deviations from Plan

None - TRD executed exactly as written. (One test-design self-correction occurred during Task 1's RED-phase drafting — an initial version of the non-mutation test passed a deliberately mismatched/shorter `source` slice and panicked inside goldmark's `text.Segment.Value`, since AST byte-offsets are relative to the exact source they were parsed against; this was caught and replaced before any commit with a matching-source non-mutation + idempotency proof achieving the same acceptance intent. This is a test-authoring correction, not a deviation from the TRD's functional requirements, so no Rule 1-4 classification applies.)

## Issues Encountered
None beyond the self-caught test-design issue documented above.

## Next Objective Readiness
- Objective 2 (model-profile) is complete: `chase.Render` is the proven one-parse-two-sinks internal entrypoint MODEL-02 required, composing 02-01/02-02/02-03's deliverables.
- Objective 3's `press.Render` can wrap `chase.Render` directly, adding batteries (emoji/chroma/math/sanitize) and `go:embed` named-theme loading without touching this entrypoint's single-parse guarantee.
- No blockers.

## Self-Check: PASSED

- FOUND: `chase/markdown/renderdoc.go`, `chase/markdown/renderdoc_test.go`, `chase/chase.go`, `chase/chase_test.go`, `.planning/objectives/02-model-profile/02-04-SUMMARY.md` — all present on disk.
- FOUND: commit hashes `8c10088`, `9f0d142`, `1e2629d` — all present in `git log --oneline --all`.
- FOUND: `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .`, `addlicense -check` — all exit 0 (re-run at self-check time, not just earlier in the session).

---
*Objective: 02-model-profile*
*Completed: 2026-07-21*
