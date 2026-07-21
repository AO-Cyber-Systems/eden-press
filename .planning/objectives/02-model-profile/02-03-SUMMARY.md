---
objective: 02-model-profile
job: "03"
subsystem: profile
tags: [go, de-hardcoding, css-scoping, grep-gate, tdd, behavior-preserving]

# Dependency graph
requires:
  - objective: 02-model-profile
    provides: "02-02's chase/profile.Profile interface (ID/UnitElement/Container/Sizes/Pagination/Scaffold) + Register/Get/Default registry — implemented here"
provides:
  - "profiles/slides: the ONLY registered Profile implementation, owning every slide-specific value (unit element, size table, scaffold CSS, pagination rule)"
  - "chase/theme's scoping passes (selector/scope.go, pass_root.go, pack.go, meta.go, stylesheet.go) now take the unit-element ident, size-fallback table, scaffold CSS, and advanced-background CSS as CALLER-supplied parameters — zero slide-specific values remain in chase/theme"
  - "A CI-enforceable grep-gate test (profiles/slides/slides_test.go: TestGrepGate) that programmatically proves chase/model + chase/theme stay free of Slide-family identifiers, the quoted \"section\" literal, and 16:9/4:3/1280/720/960 size constants"
affects: [02-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Value relocation, not rewrite: scoping ALGORITHMS (Prepend/Replace/MarkRoot/IncreasingSpecificity, pass ordering) stay in chase/theme; only the profile-VARYING VALUES (unit ident, size table, scaffold/advanced-bg CSS text, pagination rule) moved to profiles/slides and are threaded back in as parameters"
    - "Local test-only fixtures instead of a test-only import cycle: chase/theme's own _test.go files define their own byte-identical copies of the unit ident / size-fallback table / scaffold CSS (testSizeFallback, testDefaultSize, testScaffoldCSS, testAdvancedBackgroundCSS) rather than importing profiles/slides — keeps chase/theme a true dependency leaf even in its test binary"
    - "Ordered, gate-verified commit sequence per the TRD's error_recovery: (1) profiles/slides supplying byte-identical values, GREEN; (2) chase/theme threading those values as params with the Objective-1 cssdiff/corpus gates re-verified GREEN before commit; (3) grep-gate test, GREEN"
    - "CI-style grep-gate as a real Go test (os/exec + grep), not a one-off manual command — regressions get caught automatically going forward"

key-files:
  created:
    - profiles/slides/slides.go
    - profiles/slides/scaffold.go
    - profiles/slides/slides_test.go
  modified:
    - chase/theme/pack.go
    - chase/theme/selector/scope.go
    - chase/theme/selector/root.go
    - chase/theme/pass_root.go
    - chase/theme/pass_advancedbg.go
    - chase/theme/pass_nesting.go
    - chase/theme/meta.go
    - chase/theme/stylesheet.go
    - chase/theme/scaffold.go
    - chase/theme/theme.go
    - chase/theme/meta_test.go
    - chase/theme/stylesheet_test.go
    - chase/theme/pack_test.go
    - chase/theme/pack_conformance_test.go
    - chase/theme/selector/selector_test.go

key-decisions:
  - "selector.SlideChain() renamed to UnitChain(unit string) and Prepend/MarkRoot/IncreasingSpecificity all take an explicit unit string param, compared against instead of the literal \"section\" — the fusion ALGORITHM is byte-for-byte identical, only the ident source changed"
  - "chase/theme.Load/loadPlain/ParseMeta/ParseTheme/NewThemeSet/Pack all gained caller-supplied params (unit, sizeFallback, scaffoldCSS, advancedBackgroundCSS) instead of reading package-level hardcoded constants — chase/theme now owns zero profile-specific values of its own"
  - "chase/theme's own test files do NOT import profiles/slides (even though Go permits it via the internal test package) — local byte-identical fixture constants (testScaffoldCSS, testAdvancedBackgroundCSS, testSizeFallback, testDefaultSize) are defined instead, preserving chase/theme as a true leaf package with no reverse dependency, even in tests"
  - "pass_nesting.go and chase/model needed no functional changes — pass_nesting.go got one comment-only reword (removed a quoted \"section\" example) purely to satisfy the grep-gate; chase/model was confirmed clean of all three patterns in production code from the start (all matches were in _test.go files)"

requirements-completed: [MODEL-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 15min
completed: 2026-07-21
---

# Objective 02 TRD 03: profiles/slides — the only Profile impl + de-hardcode chase/theme Summary

**`profiles/slides` now owns every Marp-slide-specific value (section unit element, 16:9/4:3 size table, scaffold/advanced-background CSS, pagination rule) byte-identically, and `chase/theme`'s scoping engine takes all of it as caller-supplied parameters — grep-proven clean of Slide/`"section"`/size-constant literals, with the Objective-1 cssdiff + chase-corpus gates staying green throughout.**

## Performance

- **Duration:** ~15 min (Task 1 commit to Task 3 commit: 02:11:15 -> 02:26:06 local)
- **Started:** 2026-07-21T06:11:15Z
- **Completed:** 2026-07-21T06:26:06Z
- **Tasks:** 3
- **Files touched:** 18 (3 created, 15 modified); +875/-404 lines

## Accomplishments
- `profiles/slides` implements `chase/profile.Profile` in full (`ID`, `UnitElement`, `Container`, `Sizes`, `Pagination`, `Scaffold`) and self-registers via `init()` as the sole profile — `profile.Get("slides")` and `profile.Default()` both resolve to it.
- The slide-reset scaffold CSS and advanced-background CSS were relocated **byte-for-byte** from `chase/theme/scaffold.go` into `profiles/slides/scaffold.go` (verified via fresh re-read before every edit), carrying forward the Marpit attribution/doc comments.
- `chase/theme`'s scoping engine — `selector.Prepend`/`MarkRoot`/`IncreasingSpecificity`/`UnitChain` (renamed from `SlideChain`), `pack.go`'s `NewThemeSet`/`Pack`, `meta.go`'s `ParseMeta`/`ParseTheme`, `stylesheet.go`'s `Meta.ResolveSize`, `theme.go`'s `Load`/`loadPlain` — now all take the unit-element ident, size-fallback table, scaffold CSS, and advanced-background CSS as **parameters**. The pass order and every scoping algorithm are unchanged.
- A real, CI-enforceable `TestGrepGate` (in `profiles/slides/slides_test.go`) programmatically runs the TRD's exact three grep patterns against `chase/model` + `chase/theme` production code and fails on any match — this is now a permanent regression guard, not a one-off manual check.
- The Objective-1 acceptance gates (`chase/theme/pack_conformance_test.go`'s `cssdiff.Equal` and `conformance/runner/chase_corpus_test.go`'s 10-passing-case corpus) both stayed green throughout, proving the relocation changed zero behavior.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Build profiles/slides | `go test ./profiles/slides/... -v && go vet ./profiles/slides/...` | 0 | PASS |
| 2: Thread params into chase/theme; re-green Objective-1 gates | `go build ./... && go test ./chase/theme/... ./conformance/... -v` | 0 | PASS |
| 3: Grep-proof gate | `go test ./profiles/slides/ -run TestGrepGate -v && bash -c 'grep -rnE "\"section\"\|16:9\|1280\|720\|960" chase/model chase/theme --include="*.go" \| grep -v _test.go; test $? -ne 0'` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (raw `git commit` never used):

1. **Task 1: profiles/slides — the only Profile impl** - `fec84a1` (feat)
2. **Task 2: thread profile-supplied unit/size/scaffold/pagination params into chase/theme** - `2dac72d` (refactor)
3. **Task 3: CI-enforceable grep-gate test** - `54f5171` (test)

_Note: Tasks 1-3 are `tdd="true"` per the TRD. Task 1 and Task 3 were developed test-first against already-defined interfaces/patterns (profiles/slides implementing 02-02's locked `Profile` interface; the grep-gate's three patterns were specified verbatim by the TRD itself) and landed as single coherent commits once green, matching the TRD's explicit ordered-commit recovery note ("Commit in this order so the build never breaks mid-move ... Each commit is GREEN before the next") rather than a literal separate RED-commit/GREEN-commit pair per task. Task 2 is a pure signature-threading refactor across 15 files verified continuously against the Objective-1 regression suite (chase/theme's own pre-existing tests + pack_conformance_test.go's cssdiff gate) at every step before commit — see TDD Evidence below for the specific RED/GREEN evidence this shape of task produces (compile-time `go vet` failures at stale call sites, standing in for RED, until every call site was updated)._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./profiles/... ./chase/theme/... ./conformance/...` | 0 | PASS |

Additional evidence gathered beyond the TRD's 3 named gates:
- `go test ./...` (whole repo, all packages) — exit 0, no regressions in `chase/directive`, `chase/markdown`, `chase/model`, `chase/profile`
- `go test ./chase/theme/... -run 'TestObjective1|TestPackFullPipeline' -v` — all 4 cssdiff-gate tests (`TestObjective1ThemeCSSDiffGateStress`, `TestObjective1ThemeCSSDiffGateScaffold`, `TestPackFullPipelineStressThemeMatchesFixtureViaCSSDiff`, `TestPackFullPipelineScaffoldThemeMatchesFixtureViaCSSDiff`) PASS — the golden fixtures (`expectedStressPackedCSS`/`expectedScaffoldPackedCSS`) were never edited
- `conformance/runner/chase_corpus_test.go`'s `TestChaseCorpus` — PASS, 10 passed / 8 blocked (skip-map, unrelated) / 0 failed, unchanged from pre-TRD state
- `gofmt -l .` — empty output, no formatting diffs
- `find . -name '*.go' | xargs addlicense -check -c "AO Cyber Systems"` — exit 0, every file (new and modified) carries the Eden MIT header ending in `// SPDX-License-Identifier: MIT`
- `go.mod`/`go.sum` untouched — no `go mod tidy` run (module stayed on the already-pinned `tdewolff/parse`/`goldmark` deps)

## Grep-Gate Proof (MODEL-04)

The TRD's exact three patterns, run against `chase/model chase/theme --include='*.go'` filtered to exclude `_test.go`:

```
$ grep -rnE '\bSlide[A-Za-z]*\b' chase/model chase/theme --include='*.go' | grep -v _test.go
(no output)

$ grep -rn '"section"' chase/model chase/theme --include='*.go' | grep -v _test.go
(no output)

$ grep -rnE '16:9|4:3|1280|720|960' chase/model chase/theme --include='*.go' | grep -v _test.go
(no output)
```

All three are empty — the gate passes. This was caught in-flight once: `chase/theme/selector/scope.go`'s `UnitChain` doc comment originally read "Replaces the old, fixed **SlideChain** helper", which itself tripped pattern 1 (case-sensitive `\bSlide[A-Za-z]*\b`); reworded to "Replaces the old, hardcoded fixed-ident helper" before the Task 3 commit.

`chase/model` required **zero production-code changes** — every pre-existing grep hit there was confined to `_test.go` files (`document_test.go`, `build_test.go`), confirmed via grep before Task 2 began.

Note: a broader, case-insensitive sweep (`grep -in -E 'Slide|"section"|1280|720|16:9|4:3'`) additionally surfaces lowercase `slide`/`profiles/slides` mentions in doc comments — e.g. Marpit's own `:marpit-container > :marpit-slide` placeholder terminology (predates this TRD, shipped in Objective 1's locked `selector` package) and legitimate cross-references to the `profiles/slides` package name in doc comments explaining where a value now originates. These are outside the TRD's literal, case-sensitive gate definition (which targets capitalized `Slide`-family identifiers and the quoted `"section"` literal specifically) and are expected/acceptable — the `TestGrepGate` test codifies the TRD's exact patterns, not the broader sweep.

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| Task 2 RED-equivalent (stale call sites after production signature changes) | `go vet ./...` | 1 (`chase/theme/selector/selector_test.go:191:35: not enough arguments in call to Prepend`; `chase/theme/meta_test.go:36:25: not enough arguments in call to ParseMeta`) | FAIL (correct — confirms production de-hardcoding took effect before test call sites were updated) |
| Task 2 GREEN | `go build ./... && go vet ./... && go test ./chase/theme/... ./conformance/...` | 0 | PASS (correct) |
| Task 3 (TestGrepGate authored against the TRD's pre-specified patterns) | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS on first write (patterns were fully satisfied by Task 2's completed relocation) |

_REFACTOR: none required beyond the single doc-comment fix documented above under Grep-Gate Proof._

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 — (1) profiles/slides is the ONLY registered Profile and reproduces 16:9/4:3/section/paginate with all values originating there; (2) chase/theme.Pack scopes using caller-supplied unit/size/scaffold, grep-proven clean; (3) Objective-1 cssdiff + chase-corpus gates stay green unchanged; (4) chase/theme imports nothing from chase/* except chase/theme/selector — no import cycle (verified via `go build ./...` succeeding and manual import-list review of pack.go/theme.go)
- **Gate failures:** None (one in-flight self-caught doc-comment grep trip, fixed before commit — see Grep-Gate Proof)

## Files Created/Modified

**Created:**
- `profiles/slides/slides.go` (102 lines) — `Profile` impl: `ID`, `UnitElement`, `Container`, `Sizes`, `Pagination`, `Scaffold`, `init(){ profile.Register(New()) }`
- `profiles/slides/scaffold.go` (135 lines) — `ScaffoldCSS`/`AdvancedBackgroundCSS`, byte-identical relocation from `chase/theme/scaffold.go`
- `profiles/slides/slides_test.go` (200 lines) — Test-list cases 1-4 (ID/registration, unit+container, sizes, pagination+scaffold) + Task 3's `TestGrepGate`

**Modified (chase/theme — de-hardcoding, Task 2):**
- `chase/theme/pack.go` — `NewThemeSet(unit, scaffoldCSS, advancedBackgroundCSS string)`, `Pack` uses `ts.unit`/`ts.advancedBackground`; `scopePass`/`scopeRulesAll`/`scopeSelector` take `unit string`
- `chase/theme/selector/scope.go` — `SlideChain()` removed, replaced by `UnitChain(unit string)`; `Prepend(compound []css.Token, unit string)` compares against the passed ident
- `chase/theme/selector/root.go` — `MarkRoot(tokens, unit string)`, `IncreasingSpecificity(tokens, unit string)`, `rootMarkerAt`/`marpitRootMarkerAt` take `unit string`
- `chase/theme/pass_root.go` — `rootMarkPass(unit string) Pass`, `specificityPass(unit string) Pass` (constructor funcs, not zero-arg package vars)
- `chase/theme/pass_advancedbg.go` — `mustLoadPlainRules(cssText, unit string)`, `advancedBackgroundPass(rules []Rule) Pass`
- `chase/theme/pass_nesting.go` — comment-only reword (removed a quoted `"section"` doc example)
- `chase/theme/meta.go` — `ParseMeta(cssText string, sizeFallback map[string]Size)`, `parseSizeValue(value string, sizeFallback map[string]Size)`, `ParseTheme(cssText string, sizeFallback map[string]Size)`
- `chase/theme/stylesheet.go` — `Meta.ResolveSize(name string, fallback Size)`; `defaultWidthPx`/`defaultHeightPx` consts removed
- `chase/theme/scaffold.go` — reduced to just the `ScaffoldThemeName` identity constant; `ScaffoldCSS`/`AdvancedBackgroundCSS` fully removed (relocated to profiles/slides)
- `chase/theme/theme.go` — `Load(cssText, unit string, sizeFallback map[string]Size)`, `loadPlain(cssText, unit string)`

**Modified (chase/theme tests — updated call sites, Task 2):**
- `chase/theme/meta_test.go`, `chase/theme/stylesheet_test.go`, `chase/theme/pack_test.go`, `chase/theme/pack_conformance_test.go`, `chase/theme/selector/selector_test.go` — every `Load`/`ParseMeta`/`ParseTheme`/`ResolveSize`/`NewThemeSet`/`Prepend`/`MarkRoot`/`IncreasingSpecificity` call site updated to the new signatures. `pack_test.go` and `stylesheet_test.go` gained local, byte-identical test-only fixtures (`testScaffoldCSS`, `testAdvancedBackgroundCSS`, `testSizeFallback`, `testDefaultSize`) standing in for a Profile's values — chosen over importing `profiles/slides` into `chase/theme`'s tests, to keep `chase/theme` a true dependency leaf even in its own test binary. `pack_conformance_test.go`'s golden fixtures (`expectedStressPackedCSS`/`expectedScaffoldPackedCSS`) were **not** touched.

## Decisions Made
- Kept `Replace`/`findPlaceholder`/`isContainerSentinel`/`InlineSVGContainerChain`/`NonSVGContainerChain` completely unchanged in `selector/scope.go` — lowercase `slide`/`container` parameter and variable names do not trip the case-sensitive `\bSlide[A-Za-z]*\b` grep pattern, so only the exported `SlideChain()` needed renaming, minimizing the diff.
- Chose local test-only fixture duplication over a test-only import of `profiles/slides` into `chase/theme`'s test files — even though Go's internal test-package mechanism would technically permit it, this keeps `chase/theme` verifiably a leaf package (zero reverse dependencies, in production code AND tests), removing any cycle-risk ambiguity for future readers.
- `pass_pagination.go` and all of `chase/model` required zero changes — both were confirmed clean of all three grep patterns before Task 2 began, and remain untouched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `chase/theme/pass_nesting.go` doc comment tripped the grep gate**
- **Found during:** Task 3 verification pass
- **Issue:** `expandNestedSelector`'s doc comment used a quoted `"section"` example, which is production (non-test) code matched by grep-gate pattern 2.
- **Fix:** Reworded the comment to describe the transform generically without a quoted literal example.
- **Files modified:** `chase/theme/pass_nesting.go` (not in the TRD's literal `files_modified` list, but necessary — de-hardcoding is grep-verified at the identifier/literal level across the whole `chase/theme` package, not just the files the TRD named)
- **Commit:** `2dac72d` (Task 2, since the fix was made before the Task 2 commit landed)

**2. [Rule 1 - Bug] `selector/scope.go`'s `UnitChain` doc comment self-referentially tripped pattern 1**
- **Found during:** first grep-gate proof pass, before Task 3's commit
- **Issue:** The doc comment read "Replaces the old, fixed **SlideChain** helper" — the word `SlideChain` itself is a `Slide`-family capitalized identifier match.
- **Fix:** Reworded to "Replaces the old, hardcoded fixed-ident helper".
- **Files modified:** `chase/theme/selector/scope.go`
- **Commit:** `2dac72d` (Task 2)

**3. [Scope expansion, not a Rule 1-4 deviation] Files touched beyond the TRD's literal `files_modified` list**
- The TRD named `chase/theme/pack.go`, `selector/scope.go`, `pass_root.go`, `meta.go`, `stylesheet.go`, `scaffold.go` as the files to modify. Achieving a fully compiling, fully grep-clean result also required touching `chase/theme/pass_advancedbg.go`, `chase/theme/theme.go`, `chase/theme/selector/root.go`, and `chase/theme/pass_nesting.go` (comment-only) — these hold the same category of caller-supplied-value threading (advanced-background CSS loading, Tier-1 `Load`, the `:root` sentinel/specificity trick) that the TRD's own `codebase_examples` and `error_recovery` sections describe as in-scope for "parameterizing the fused-element ident" and "the deep bake." Each was updated using the identical value-relocation-not-rewrite discipline as the explicitly named files, with the same regression gates run after every change.
- Five `_test.go` files (`meta_test.go`, `stylesheet_test.go`, `pack_test.go`, `pack_conformance_test.go`, `selector_test.go`) required call-site updates to match the new production signatures — an unavoidable consequence of the signature-threading refactor, not a scope or behavior change.

## Issues Encountered
None beyond the two self-caught, auto-fixed grep-gate comment trips documented above. No auth gates, no checkpoints, no blockers.

## User Setup Required
None — pure Go package/refactor work, zero new dependencies, `go.mod`/`go.sum` untouched (no `go mod tidy` run).

## Next Objective Readiness
- `profiles/slides` is fully implemented, self-registering, and the only `Profile` in the registry — ready for 02-04 (the chase entrypoint) to call `profile.Get`/`profile.Default` and pass the resolved `Profile`'s `UnitElement()`/`Sizes()`/`Scaffold()`/`Pagination()` into `theme.NewThemeSet`/`theme.Load`/`theme.Pack`.
- Adding a future non-slide profile (e.g. `profiles/paged` — `.page` container, A4/Letter sizes, no `section` reset) now requires **zero** `chase/theme` edits, per this TRD's success criterion — `chase/theme` reads every profile-varying value from its parameters.
- No blockers. The Objective-1 cssdiff (`chase/theme/pack_conformance_test.go`) and chase corpus (`conformance/runner/chase_corpus_test.go`) gates are confirmed green post-de-hardcoding and remain the standing regression guard for any future `chase/theme` change.

## Self-Check: PASSED

All created-file claims and commit-hash claims verified against disk/git before finalizing this summary:

- FOUND: `profiles/slides/slides.go`
- FOUND: `profiles/slides/scaffold.go`
- FOUND: `profiles/slides/slides_test.go`
- FOUND: `chase/theme/pack.go`, `chase/theme/selector/scope.go`, `chase/theme/selector/root.go`, `chase/theme/pass_root.go`, `chase/theme/pass_advancedbg.go`, `chase/theme/pass_nesting.go`, `chase/theme/meta.go`, `chase/theme/stylesheet.go`, `chase/theme/scaffold.go`, `chase/theme/theme.go`
- FOUND: `fec84a1`, `2dac72d`, `54f5171` (all 3 in `git log --oneline --all`)
- FOUND: `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l .`, `addlicense -check` all exit 0 (re-run at self-check time, not just earlier in the session)

---
*Objective: 02-model-profile*
*Completed: 2026-07-21*
