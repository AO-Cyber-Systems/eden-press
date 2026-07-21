# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 COMPLETE (9/9 TRDs); Objectives 4/5/6/7 unblocked and executing in parallel worktreams — this worktree executed Objective 6 (convert/pptx) TRD 06-02

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete. Objective 6 (convert/pptx) now 1/5 TRDs complete (this worktree: 06-02) — parallel workstream, pending orchestrator reconcile at merge.
Job: Objective 7 (07-dart-binding), TRD 03 of 5 (wave 2) — 07-03 (DART-02: Android/iOS native builds) complete on this worktree: scripts/build-android.sh (c-shared .so per NDK ABI) + scripts/build-ios.sh (c-archive→lipo→EdenPress.xcframework, verified end-to-end on this host) + .github/workflows/dart-native.yml (two isolated Android/iOS CI jobs, NDK r27c + Xcode 16.4 pinned, SC#2 covered); no gomobile; commits b81a450/c307f5c/31326dc. Pending orchestrator reconcile at merge.
Status: 06-02-TRD executed on this worktree — new convert/pptx package created (doc.go) with an independently unit-tested EMU-conversion utility: Inches/Points/Centimeters/Millimeters(float64) int64 against the fixed ECMA-376 constants (914400/12700/360000/36000), a Centipoints helper explicitly distinct from EMU (a:rPr/@sz unit), authoritative SlideSize16x9/SlideSize4x3/NotesSize constants cross-checked against Inches, and a GroupTransform implementing the chOff/chExt child-to-slide coordinate mapping (ECMA-376 CT_GroupTransform2D) proven for both the identity case (06-04's grouped-shape v1 simplification) and a non-identity 0.5-scale case. Zero new dependencies (stdlib math only); go.mod/go.sum untouched. convert/pptx confirmed at 0 references from press/chase/profiles (isolation gate). Objective 6 at 1/5 TRDs — 06-03 (OPC zip packager) and 06-04 (ToPPTX writer) now consume this foundation.
Last activity: 2026-07-21 — 06-02-TRD executed (Objective 6, wave 1): convert/pptx/emu.go built test-first (RED->GREEN both tasks, 0 auto-fix cycles) — Task 1 (EMU conversions + slide-size constants, commit c5eb144), Task 2 (chOff/chExt group-transform identity + non-identity, commit 6de5619); all gates green (gofmt, go build ./..., go vet ./convert/pptx/..., go test ./convert/pptx/... + whole-repo go test ./..., check-no-chromedp.sh, addlicense, isolation grep-gate 0 matches, go.mod/go.sum diff empty).

Progress: [██████████] 100% (27/27 TRDs across Objectives 0-3, fully complete); Objective 6 (convert/pptx): 1/5 TRDs complete on this worktree (06-02); Objective 4 (cli): 04-04 and 04-06 complete on this worktree (koanf config loading CLI-06; watch mode fsnotify+debounce+SSE reload CLI-02 — commits da9ab7b/ec6df0a/99c912b/2d48136)

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

Last session: 2026-07-21 (07-02-TRD execution — Objective 7 wave 2, DART-03: web WASM binding)
Stopped at: Completed 07-02-TRD.md (DART-03): built the DART-03 web front door over 07-01's shared `bind/capi/core.RenderJSON` seam. `bind/wasm/main.go` (`package main`, `//go:build js && wasm`) registers `globalThis.pressRender` via `js.Global().Set`/`js.FuncOf`, reads the JSON request from `args[0]`, delegates to the SAME `core.RenderJSON` (no cgo, no second marshalling), and blocks `main()` on `select{}` so the export stays callable. The FULL press battery chain (goldmark GFM + chroma + latex2mathml + bluemonday) compiles clean under STANDARD-Go `GOOS=js GOARCH=wasm CGO_ENABLED=0` — the WASM-target decision gate is RESOLVED in favor of standard Go (TinyGo = future size-spike note only; no press-chain precedent). `scripts/build-wasm.sh` emits `bind/wasm/press.wasm` (gitignored build product) + copies `$(go env GOROOT)/lib/wasm/wasm_exec.js` (Go 1.24+ path). `scripts/check-wasm-exec-version.sh` diff-gates loader-vs-toolchain drift (RESEARCH Pitfall 2; proven both directions: stale→exit1, matched→exit0). `bind/wasm/smoke/smoke.mjs` loads press.wasm under Node (v24) via wasm_exec.js and round-trips `# Hi`→`<h1` and `~~struck~~`→press `<s>` (not goldmark `<del>`), asserting the real wire key `output.HTML` and envelope `eden-press.capi/v1`. 2 task commits (aced805, 235c54f), both `type=auto`, 0 auto-fix cycles. Gates: gofmt, host `go build ./...`, `GOOS=js GOARCH=wasm` build, go vet ./..., go test ./... (whole repo green), check-no-chromedp.sh PASS, addlicense PASS (recognizes wasm_exec.js's upstream BSD header). Objective 7 now 2/5 TRDs (07-01 + 07-02, this worktree) — 07-04 (conformance) and 07-05 (Dart web loader) consume press.wasm + the pinned wasm_exec.js. Parallel worktree — pending orchestrator reconcile at merge.
Resume file: None
