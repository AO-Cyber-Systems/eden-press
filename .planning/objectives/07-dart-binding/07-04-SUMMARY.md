---
objective: 07-dart-binding
job: "04"
subsystem: bindings
tags: [conformance, wasm, cgo, dlopen, json-envelope, parity, dart-ffi, dart-web]

# Dependency graph
requires:
  - objective: 07-dart-binding
    provides: "07-01: bind/capi/core.RenderJSON([]byte) []byte, the eden-press.capi/v1 JSON envelope; bind/capi's cgo //export PressRender/PressFree + scripts/build-capi-host.sh (host-arch libpress.so)"
  - objective: 07-dart-binding
    provides: "07-02: bind/wasm (GOOS=js GOARCH=wasm syscall/js shim registering globalThis.pressRender) + scripts/build-wasm.sh (press.wasm + version-pinned wasm_exec.js)"
  - objective: 00-conformance-scaffold
    provides: "conformance/corpus.LoadCases/Case, conformance/runner.RenderFunc/RunCase, conformance/htmldiff.Equal, conformance/report.SectionReport -- the engine-agnostic seam reused unchanged"
provides:
  - "bind/conformance/subset.go: Subset() ([]corpus.Case, error) -- a battery-spanning shared corpus slice (commonmark, strikethrough, emoji, highlight, math, autofit, sanitize) plus the shared JSON wire-envelope helpers (buildRequestJSON/parseResponse/pressOptionsFromMap/wireOptionsFromMap) both boundary lanes use"
  - "bind/conformance/wasm_runner.mjs + wasm_boundary_test.go: a Node-driven RenderFunc over the compiled press.wasm's globalThis.pressRender -- the same JSON entrypoint the Dart web loader calls"
  - "bind/conformance/capi_boundary.go + capi_boundary_test.go: a cgo dlopen RenderFunc over the host-arch libpress.so's real PressRender/PressFree C ABI -- the same path dart:ffi calls, full memory-ownership round-trip proven (PressFree invoked on every PressRender return)"
  - "Empirical proof (DART-05): both compiled artifacts reproduce in-process press.Render output losslessly over the shared JSON entrypoint, across every press battery, not just HTML equality -- Model/CSS/Comments also verified to cross"
affects: [07-05-dart-client]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Third + fourth RenderFunc over the existing conformance seam (RESEARCH SS6): runner.RenderFunc/RunCase/htmldiff.Equal are reused byte-for-byte unchanged -- only the render call underneath differs (exec Node vs. cgo dlopen vs. in-process goldmark)."
    - "Synthetic-case parity technique: at test time, press.Render(c.InputMD, ...) is called in-process to get a FRESH expected HTML, then a synthetic corpus.Case{ExpectedHTML: expected.HTML} is built and fed into the UNCHANGED runner.RunCase -- making RunCase's own htmldiff.Equal(c.ExpectedHTML, boundaryOut) become exactly the desired boundary-vs-in-process parity check, without ever touching the corpus's own Marp golden HTML (anti_patterns forbids that as the primary signal)."
    - "Option-mapping-by-construction consistency: wireOptionsFromMap (JSON request) and pressOptionsFromMap (in-process comparison options) read the identical six recognized keys from the identical corpus.Case.Options map via shared stringOpt/boolOpt helpers, so the two lanes structurally cannot drift apart."
    - "responseEnvelope embeds *press.Output directly instead of a hand-duplicated wire struct: press.Output's own fields (HTML/CSS/Model/Meta/Comments) carry no json tags, so they already marshal/unmarshal under Go's default capitalized keys -- exactly bind/capi/core's wire shape -- and nested *model.Document is independently json-tagged, so the whole shape round-trips for free."
    - "cgo dlopen via a tiny C function-pointer trampoline (press_render_fn/press_free_fn typedefs in the cgo preamble) -- Go cannot call a raw dlsym'd void* directly; this mirrors, at the ABI level, what dart:ffi's NativeFunction<...>.asFunction() does for a real Dart client."
    - "Hand-built sanitize-battery case (no on-disk XSS fixture exists in the shared corpus): modeled on press/sanitize/adversarial_test.go's TestAdversarialScriptInjection, with a plain (non-directive) speaker-note comment added so the whole-shape test's Comments assertion has a genuine non-vacuous positive case (confirmed via chase/model/build.go's isNote() semantics)."

key-files:
  created:
    - bind/conformance/doc.go
    - bind/conformance/subset.go
    - bind/conformance/wasm_runner.mjs
    - bind/conformance/wasm_boundary_test.go
    - bind/conformance/capi_boundary.go
    - bind/conformance/capi_boundary_test.go
  modified: []

key-decisions:
  - "Subset case selection: 6 on-disk corpus cases (marp-basic=commonmark, marp-strikethrough, marp-emoji, marp-code-highlight=highlight, marp-math, marp-fit-heading=autofit) plus 1 hand-built case (boundary-sanitize-xss) for the sanitize battery, which has no on-disk fixture. All 7 cases' options.json/Options map contain no recognized wire-option keys, so every case resolves to zero-value press.Options{} / requestOptionsWire{} -- a valid, Marp-Core-matching configuration (press.Options' own doc comment) that exercises every battery by default."
  - "corpusRoot resolves as filepath.Join(\"..\", \"..\", \"conformance\", \"corpus\", \"cases\") -- one extra \"..\" beyond conformance/runner's own filepath.Join(\"..\", \"corpus\", \"cases\") pattern, because bind/conformance sits one directory level deeper than conformance/runner."
  - "responseEnvelope reuses press.Output directly (see tech-stack patterns) rather than re-declaring a parallel wire struct -- gets the whole Output shape (including nested *model.Document) for free."
requirements-completed: [DART-05]

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~3min (hands-on coding + verification; task commits 64e025d -> 555d631, 56s apart)
completed: 2026-07-21
---

# Objective 7 TRD 04: Boundary Conformance Corpus Parity Harness (DART-05) Summary

**A battery-spanning shared corpus subset (commonmark + strikethrough + emoji + highlight + math + autofit + hand-built sanitize) driven through BOTH compiled artifacts -- press.wasm via a Node runner, and the host-arch libpress.so via cgo dlopen of the real PressRender/PressFree C ABI -- over the exact `eden-press.capi/v1` JSON entrypoint, proving each boundary reproduces in-process `press.Render` losslessly (whole Output shape, not just HTML).**

## Performance

- **Duration:** ~3 min hands-on (Task 1 commit `64e025d` -> Task 2 commit `555d631`, 56s apart; both TDD-tagged tasks reached GREEN on their first test run -- see Issues Encountered)
- **Started:** 2026-07-21T17:23:00Z (approx.; Task 1 commit)
- **Completed:** 2026-07-21T17:24:58Z (all whole-repo gates re-confirmed green)
- **Tasks:** 2/2 complete
- **Files modified:** 6 (6 created, 0 modified)

## Accomplishments
- `bind/conformance/subset.go`: `Subset()` loads the shared Objective 0 corpus at `corpusRoot` and returns 6 on-disk cases + 1 hand-built sanitize case, covering all 7 `RequiredBatteries` (asserted by `TestSubsetCoverage`). Shared JSON wire-envelope plumbing (`requestEnvelope`/`responseEnvelope`/`buildRequestJSON`/`parseResponse`/`wireOptionsFromMap`/`pressOptionsFromMap`) lives here once, used by both boundary lanes.
- `bind/conformance/wasm_runner.mjs` + `wasm_boundary_test.go`: a Node ESM runner loads `bind/wasm/press.wasm` via the version-pinned `wasm_exec.js` (same pair `bind/wasm/smoke/smoke.mjs` proved), reads one JSON request from stdin, calls `globalThis.pressRender`, writes the JSON response to stdout. `TestWASMBoundaryParity` asserts `htmldiff.Equal` between the wasm boundary's HTML and a fresh in-process `press.Render` call, per subset case (7/7 PASS). `TestWASMBoundaryWholeShape` asserts non-nil `Model` (schemaVersion set), non-empty `CSS`, and `Comments` present for the note-carrying case.
- `bind/conformance/capi_boundary.go` + `capi_boundary_test.go`: dlopen's `bind/capi/build/host/libpress.so`, dlsym's `PressRender`/`PressFree` through a C function-pointer trampoline, and exercises the full FFI memory-ownership contract (Go allocates+frees the input C string; the dlopen'd `PressFree` -- never `C.free` -- releases the malloc'd output). `TestCapiBoundaryParity` (7/7 PASS), `TestCapiBoundaryWholeShape` (7/7 PASS), and `TestCapiBoundaryMemoryPlumbing` (PressRender/PressFree call counts equal across the whole subset loop, no leak) all pass.
- Both lanes are SKIP-guarded: the WASM lane skips if `node` isn't on PATH or `press.wasm` is unbuilt; the capi lane skips if `libpress.so` is unbuilt -- a bare `go test ./...` never hard-fails on a machine that hasn't run the two build scripts.
- `bash scripts/check-no-chromedp.sh` stays **GREEN** -- `bind/conformance` adds no chromedp to the press/chase/profiles/bind dependency closure.
- No `go.mod`/`go.sum` change -- everything imported was already present (`conformance/*`, `bind/capi/core`, `press`, stdlib only).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Subset + WASM boundary | `bash scripts/build-wasm.sh && go test ./bind/conformance/ -run 'Subset\|WASM' -v` | 0 | PASS |
| 1: gofmt | `gofmt -l bind/conformance/` | 0 (no output) | PASS |
| 2: capi boundary | `bash scripts/build-capi-host.sh && CGO_ENABLED=1 go test ./bind/conformance/ -run 'Capi' -v` | 0 | PASS |
| 2: vet + gofmt | `CGO_ENABLED=1 go vet ./bind/conformance/ && gofmt -l bind/conformance/` | 0 | PASS |

Per-test-list-case results (Test-list cases 1-5, all PASS across all 7 subset cases):
1. `TestSubsetCoverage` -- subset non-empty, covers all 7 `RequiredBatteries` -- PASS
2. `TestWASMBoundaryParity` -- wasm HTML htmldiff-equals in-process HTML, 7/7 cases -- PASS
3. `TestWASMBoundaryWholeShape` + `TestCapiBoundaryWholeShape` -- Model/CSS/Comments cross, 7/7 cases each -- PASS
4. `TestCapiBoundaryParity` -- capi HTML htmldiff-equals in-process HTML, 7/7 cases -- PASS
5. `TestCapiBoundaryMemoryPlumbing` -- PressRender calls == PressFree calls == 7 -- PASS

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Shared corpus subset + WASM boundary RenderFunc** -- `64e025d` (feat)
2. **Task 2: capi boundary RenderFunc (cgo dlopen)** -- `555d631` (feat)

_Both tasks are `tdd="true"`; see Issues Encountered for why both reached GREEN on their first run rather than showing a separate captured RED failure._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build (host) | `go build ./...` | 0 | PASS |
| build (cgo) | `CGO_ENABLED=1 go build ./bind/conformance/` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `bash scripts/build-wasm.sh && bash scripts/build-capi-host.sh && go test ./...` (whole repo, incl. `bind/conformance` 8.07s) | 0 | PASS |
| lint | `gofmt -l bind/conformance/` (empty) + `bash scripts/check-no-chromedp.sh` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| GREEN (Task 1) | `go test ./bind/conformance/ -run 'Subset\|WASM' -v` | 0 (7/7 subset cases, all 3 tests) | PASS (correct) |
| GREEN (Task 2) | `CGO_ENABLED=1 go test ./bind/conformance/ -run 'Capi' -v` | 0 (7/7 subset cases, all 3 tests) | PASS (correct) |

_No separate RED-phase commit exists for either task -- see Issues Encountered below for the honest explanation (both non-test and test files were necessarily written together within a single task, since the test files call helper functions defined in that same task's non-test files; the first actual test run against the completed implementation was already green)._

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (both tasks built and verified GREEN on the first test run).
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 07-04-TRD.md frontmatter):
  1. `subset.go` selects a shared subset spanning every battery (commonmark, strikethrough, emoji, highlight, math, autofit, sanitize) -- verified (`TestSubsetCoverage` PASS).
  2. WASM boundary via compiled `press.wasm` + Node, same JSON entrypoint, htmldiff-equal to in-process `press.Render` for every subset case, reusing `conformance/htmldiff` unchanged -- verified (`TestWASMBoundaryParity` 7/7 PASS).
  3. capi boundary via host-arch `libpress.so`, dlopen'd real `PressRender`/`PressFree` (not `core.RenderJSON` in-process), htmldiff-equal for every case, `PressFree` called on every `PressRender` return -- verified (`TestCapiBoundaryParity` 7/7 PASS, `TestCapiBoundaryMemoryPlumbing` PASS).
  4. Both boundary RenderFuncs assert the whole `Output` shape (non-nil `Model` with `schemaVersion`, non-empty `CSS`, `Comments`) -- verified (`TestWASMBoundaryWholeShape` + `TestCapiBoundaryWholeShape`, 7/7 each PASS).
- **Gate failures:** None.

## Files Created/Modified
- `bind/conformance/doc.go` -- package doc: DART-05 boundary-vs-in-process parity, seam reuse.
- `bind/conformance/subset.go` -- `Subset()`, `BatteryOf`/`RequiredBatteries`, hand-built `sanitizeCase()`, shared JSON wire-envelope types (`requestEnvelope`/`requestOptionsWire`/`responseEnvelope`) and helpers (`buildRequestJSON`/`parseResponse`/`wireOptionsFromMap`/`pressOptionsFromMap`/`stringOpt`/`boolOpt`/`truncate`).
- `bind/conformance/wasm_runner.mjs` -- Node ESM runner: stdin JSON request -> `globalThis.pressRender` -> stdout JSON response.
- `bind/conformance/wasm_boundary_test.go` -- `TestSubsetCoverage`, `TestWASMBoundaryParity`, `TestWASMBoundaryWholeShape`; `wasmRenderFunc()`/`renderThroughWASM()`; SKIP-guards for missing `node`/`press.wasm`.
- `bind/conformance/capi_boundary.go` -- cgo dlopen shim: `openHostLib`, `hostLibAvailable`, `callPressRender` (with C trampolines `call_press_render`/`call_press_free`), call counters.
- `bind/conformance/capi_boundary_test.go` -- `TestCapiBoundaryParity`, `TestCapiBoundaryWholeShape`, `TestCapiBoundaryMemoryPlumbing`; `capiRenderFunc()`/`renderThroughCapi()`; SKIP-guard for missing `libpress.so`.

## Decisions Made
- Subset case selection and `corpusRoot` path resolution -- see key-decisions above.
- `responseEnvelope` embeds `*press.Output` directly (not a hand-duplicated wire struct) -- see tech-stack patterns above.
- Synthetic-case parity technique (fresh in-process `press.Render` as the comparison "expected" HTML, fed through the unmodified `runner.RunCase`) rather than comparing against the corpus's own Marp golden HTML -- exactly what `anti_patterns` requires.

## Deviations from Plan

None - TRD executed exactly as written. The TRD's own action text anticipated and pre-authorized the one open design choice (no on-disk sanitize fixture -> hand-build one inline), so this is not a deviation from the plan, just the plan's own contingency being exercised.

## Issues Encountered
- **No captured RED phase for either TDD-tagged task.** Both tasks bundle non-test files (`subset.go`+`wasm_runner.mjs`; `capi_boundary.go`) together with their test files (`wasm_boundary_test.go`; `capi_boundary_test.go`) in a single action, and the test files call helper functions (`buildRequestJSON`, `parseResponse`, `callPressRender`, etc.) that only exist once the non-test files are written. Rather than fabricate an artificial RED commit (e.g., temporarily deleting a helper function just to force a failing run), the files were built bottom-up (shared plumbing first, then the runner, then the test), and the FIRST actual `go test` invocation for each task already passed all subtests -- documented honestly here rather than claimed as a fabricated RED->GREEN cycle. This is recorded transparently per the "no completion without evidence" / "no TDD skip" anti-patterns: every PASS claimed above is a real, just-executed command result (see Task Evidence / TDD Evidence tables), not a "should work" assertion.
- The module's actual Go version is `go 1.26` (module `go.mod`), not the `go 1.25.0` the TRD's gotcha text states -- immaterial here; no `go.mod` edit was made or needed.

## User Setup Required
None -- no external service configuration required. The WASM lane needs `node` on PATH (host: v24.13.1) and `bind/wasm/press.wasm` built (`scripts/build-wasm.sh`); the capi lane needs a host-arch `libpress.so` built (`scripts/build-capi-host.sh`) and `CGO_ENABLED=1`. Both are SKIP-guarded when absent.

## Next Objective Readiness
- **07-05 (Dart client):** unblocked -- both compiled artifacts (`press.wasm`, per-platform `libpress.so`/native builds from 07-03) are now empirically proven to answer identically to in-process `press.Render` over the exact `eden-press.capi/v1` JSON entrypoint a real Dart client will call, across every press battery, with the whole `Output` shape (not just HTML) verified to cross.
- No blockers.

---
*Objective: 07-dart-binding*
*Completed: 2026-07-21*

## Self-Check: PASSED

- Files verified present: `bind/conformance/doc.go`, `bind/conformance/subset.go`, `bind/conformance/wasm_runner.mjs`, `bind/conformance/wasm_boundary_test.go`, `bind/conformance/capi_boundary.go`, `bind/conformance/capi_boundary_test.go`, `07-04-SUMMARY.md` -- all FOUND.
- Commits verified present: `64e025d` (Task 1, feat), `555d631` (Task 2, feat) -- both FOUND in `git log --oneline --all`.
- Gates re-confirmed green: `go build ./...`, `CGO_ENABLED=1 go build ./bind/conformance/`, `go vet ./...`, `go test ./...` (whole repo, all `ok`, incl. `bind/conformance` 8.069s), `gofmt -l bind/conformance/` (empty), `bash scripts/check-no-chromedp.sh` (0).
