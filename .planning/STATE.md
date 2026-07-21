# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 COMPLETE (9/9 TRDs); Objectives 4/5/6/7 unblocked and executing in parallel worktreams — this worktree executed Objective 6 (convert/pptx) TRD 06-02

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete. Objective 6 (convert/pptx) now 1/5 TRDs complete (this worktree: 06-02) — parallel workstream, pending orchestrator reconcile at merge.
Job: Objective 6, TRD 02 of 5 (wave 1, parallel with 06-01 chase/model schema-v2 in a separate worktree — no file overlap). This worktree's task: 06-02 EMU-conversion utility + 16:9/4:3 slide-size constants + chOff/chExt group-transform (EXP-03, partial). Pending orchestrator reconcile at merge.
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

Last session: 2026-07-21 (06-02-TRD execution — Objective 6 wave 1: convert/pptx EMU-conversion utility + slide-size constants + chOff/chExt group-transform)
Stopped at: Completed 06-02-TRD.md (EXP-03, partial): built convert/pptx's pure-math numeric foundation test-first. `convert/pptx/doc.go` establishes the new package (never imported by press/chase/profiles). `emu.go`: `Inches`/`Points`/`Centimeters`/`Millimeters(float64) int64` against fixed ECMA-376 constants (914400/12700/360000/36000 EMU), exact for whole-unit inputs via a single shared `round()` (math.Round) helper; `Centipoints(float64) int` for DrawingML `a:rPr/@sz` (hundredths of a point, explicitly NOT EMU); `SlideSize{CX,CY,Type}` + `SlideSize16x9`/`SlideSize4x3`/`NotesSize` authoritative constants cross-checked against `Inches`; `GroupTransform{Off,Ext,ChOff,ChExt}` + `IdentityGroupTransform` + `MapChild` implementing ECMA-376 CT_GroupTransform2D's chOff/chExt child-to-slide mapping (subtract-then-scale-then-add order), proven both for the identity case (06-04's grouped-shape v1 simplification) and a non-identity 0.5-scale case. 2 task commits (c5eb144, 6de5619), both TDD (RED confirmed via compile failure, then GREEN on first implementation, 0 auto-fix cycles). Gates: gofmt, go build ./..., go vet (package + whole-repo), go test (package + whole-repo, all green), check-no-chromedp.sh PASS, addlicense PASS, isolation gate (`go list -deps ./press/... ./chase/... ./profiles/...` contains 0 convert/pptx references), go.mod/go.sum diff empty (zero new deps). This worktree ran parallel to 06-01 (chase/model schema-v2, separate worktree, no file overlap) as Objective 6's wave 1. 06-03 (OPC zip packager) and 06-04 (ToPPTX writer) are unblocked to consume this foundation. Objective 6 now 1/5 TRDs complete (this worktree) — pending orchestrator reconcile at merge.
Resume file: None
