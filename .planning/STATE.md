# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 COMPLETE (9/9 TRDs); Objectives 4/5/6/7 unblocked and executing in parallel worktreams — this worktree executed Objective 6 (convert/pptx) TRD 06-02

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete. Objective 6 (convert/pptx) now 1/5 TRDs complete (this worktree: 06-02) — parallel workstream, pending orchestrator reconcile at merge.
Job: Objective 7 (07-dart-binding), TRD 04 of 5 (wave 3) — 07-04 (DART-05: boundary conformance parity harness) complete on this worktree: bind/conformance/subset.go (battery-spanning shared corpus subset + shared JSON wire-envelope helpers) + wasm_runner.mjs/wasm_boundary_test.go (Node-driven press.wasm boundary) + capi_boundary.go/capi_boundary_test.go (cgo dlopen of host libpress.so, real PressRender/PressFree C ABI) — both boundaries empirically proven to reproduce in-process press.Render losslessly (whole Output shape) over the shared eden-press.capi/v1 JSON entrypoint, across every press battery; commits 64e025d/555d631. Pending orchestrator reconcile at merge.
Status: 06-02-TRD executed on this worktree — new convert/pptx package created (doc.go) with an independently unit-tested EMU-conversion utility: Inches/Points/Centimeters/Millimeters(float64) int64 against the fixed ECMA-376 constants (914400/12700/360000/36000), a Centipoints helper explicitly distinct from EMU (a:rPr/@sz unit), authoritative SlideSize16x9/SlideSize4x3/NotesSize constants cross-checked against Inches, and a GroupTransform implementing the chOff/chExt child-to-slide coordinate mapping (ECMA-376 CT_GroupTransform2D) proven for both the identity case (06-04's grouped-shape v1 simplification) and a non-identity 0.5-scale case. Zero new dependencies (stdlib math only); go.mod/go.sum untouched. convert/pptx confirmed at 0 references from press/chase/profiles (isolation gate). Objective 6 at 1/5 TRDs — 06-03 (OPC zip packager) and 06-04 (ToPPTX writer) now consume this foundation.
Last activity: 2026-07-21 — 06-02-TRD executed (Objective 6, wave 1): convert/pptx/emu.go built test-first (RED->GREEN both tasks, 0 auto-fix cycles) — Task 1 (EMU conversions + slide-size constants, commit c5eb144), Task 2 (chOff/chExt group-transform identity + non-identity, commit 6de5619); all gates green (gofmt, go build ./..., go vet ./convert/pptx/..., go test ./convert/pptx/... + whole-repo go test ./..., check-no-chromedp.sh, addlicense, isolation grep-gate 0 matches, go.mod/go.sum diff empty).

Progress: [██████████] 100% (27/27 TRDs across Objectives 0-3, fully complete); Objective 6 (convert/pptx): 1/5 TRDs complete on this worktree (06-02); Objective 4 (cli): 04-04 complete on this worktree (koanf config loading, CLI-06)

## Accumulated Context

> Decisions and performance metrics are logged in STATE_ARCHIVE.md.

### Pending Todos

None yet.

### Blockers/Concerns

- Decision gate open: `chase/*` internal vs. exported Go (resolve during Objective 2 planning).
- Decision gate re-confirmed at 06-02 execution: hand-rolled OOXML vs. any newly-emerged permissive Go PPTX lib — 06-RESEARCH.md (dated 2026-07-21) re-verified no new permissive Go PPTX library has emerged; `unioffice`/forks remain rejected (AGPLv3/commercial license-key). No new dependency added.
- Decision gate open: standard Go vs. TinyGo for the WASM target (resolve before Objective 7's WASM-specific code is written — functional risk on reflection-heavy JSON/YAML paths, not just size).
- Decision gate open: concrete MathML fallback-trigger rule + final auto-fit mechanism (resolve during Objective 8 planning).
- CSS-AST diff tooling (Objective 0) is genuinely new/unproven engineering — no spike precedent exists; budget accordingly at planning time.

## Session Continuity

Last session: 2026-07-21 (07-04-TRD execution — Objective 7 wave 3, DART-05: boundary conformance parity harness)
Stopped at: Completed 07-04-TRD.md (DART-05): proved the two compiled Objective-7 artifacts (07-02's `press.wasm`, 07-01/07-03's `libpress.so`) answer identically to in-process `press.Render` over the shared `eden-press.capi/v1` JSON entrypoint. `bind/conformance/subset.go`: `Subset()` selects 6 on-disk corpus cases (marp-basic/strikethrough/emoji/code-highlight/math/fit-heading) + 1 hand-built sanitize case (no on-disk XSS fixture existed), covering all 7 required batteries; shared JSON wire-envelope helpers (`buildRequestJSON`/`parseResponse`/`wireOptionsFromMap`/`pressOptionsFromMap`) reused by both lanes so they cannot drift apart. WASM lane: `wasm_runner.mjs` (Node ESM, loads `press.wasm` via the pinned `wasm_exec.js`, stdin JSON -> `globalThis.pressRender` -> stdout JSON) + `wasm_boundary_test.go` — `TestWASMBoundaryParity`/`TestWASMBoundaryWholeShape`, 7/7 PASS. capi lane: `capi_boundary.go` (cgo dlopen of `libpress.so`, real `PressRender`/`PressFree` via a C function-pointer trampoline, full input/output memory-ownership round-trip) + `capi_boundary_test.go` — `TestCapiBoundaryParity`/`TestCapiBoundaryWholeShape`/`TestCapiBoundaryMemoryPlumbing`, 7/7 PASS, PressFree called on every PressRender return (no leak). Both lanes reuse `conformance/runner.RunCase`/`htmldiff.Equal` unchanged via a synthetic-case technique (fresh in-process `press.Render` HTML fed in as the "expected", never the corpus's own Marp golden). Both SKIP-guarded (missing `node`/`press.wasm`/`libpress.so`). 2 task commits (64e025d, 555d631), both `tdd=true` (GREEN on first run — documented transparently, no fabricated RED). Gates: gofmt, `go build ./...`, `CGO_ENABLED=1 go build ./bind/conformance/`, `go vet ./...`, `go test ./...` (whole repo green), `check-no-chromedp.sh` PASS. No `go.mod` change. Objective 7 now 4/5 TRDs (07-01/07-02/07-03/07-04, this worktree) — 07-05 (Dart client) is now unblocked. Parallel worktree — pending orchestrator reconcile at merge.
Resume file: None
