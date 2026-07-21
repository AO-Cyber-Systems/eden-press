# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 COMPLETE (9/9 TRDs); global wave A merged (04-01/04-02/05-01/06-01/06-02/07-01) — this worktree executed Objective 4 (CLI) TRD 04-03, the CLI-01 capstone

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete. Objective 4 (CLI) now 3/8 TRDs complete (04-01, 04-02, 04-03) — parallel workstream, pending orchestrator reconcile at merge.
Job: Objective 4, TRD 03 of 8 (wave 2, depends on 04-02 merged). This worktree's task: 04-03 htmldoc.go standalone zero-JS document assembler + script-injection seam + filled runConvert end-to-end pipeline (CLI-01). Pending orchestrator reconcile at merge.
Status: 04-03-TRD executed on this worktree — cmd/eden-press/htmldoc.go's assembleHTML(out press.Output, opts htmlDocOptions) wraps press.Output into a complete standalone document (<!doctype html> + charset/viewport meta + <style>{out.CSS}</style> + <body>{out.HTML}</body>) with NO <script> by default; opts.AutoFitScript splices press.BrowserFitJS() and opts.InjectScripts splices arbitrary scripts, both spliced AFTER out.HTML (never re-entering press.Render's sanitize pass) — the seam 04-06 (watch)/04-07 (serve) will reuse for SSE reload. cmd/eden-press/convert.go's runConvert stub filled with the full pipeline: resolveInputFrom(arg, cmd.InOrStdin()) -> buildOptions(cmd) -> press.Render -> assembleHTML -> writeOutput(cmd, doc); writeOutput defensively looks up the --output flag (present only on the "convert" subcommand, not root's default action) and falls back to cmd.OutOrStdout() otherwise. CLI-01 satisfied end-to-end: `eden-press <in.md>`, `cat deck.md | eden-press -`, and `eden-press convert --output x.html` all verified manually plus via 9 new unit/integration tests. CLI imports ONLY press/ (no chase/profiles/chromedp).
Last activity: 2026-07-21 — 04-03-TRD executed (Objective 4, wave 2): Task 1 htmldoc.go + htmldoc_test.go (zero-JS golden + auto-fit + inject-scripts seam, commit 49a59a7), Task 2 convert.go runConvert fill + convert_test.go (file->stdout, stdin->stdout, -o file, auto-fit-script, commit a144f4d); all gates green (gofmt, go build ./..., go vet ./..., go test ./... whole-repo, check-no-chromedp.sh, addlicense -check).

Progress: [██████████] 100% (27/27 TRDs across Objectives 0-3, fully complete); Objective 4 (CLI): 3/8 TRDs complete on this worktree (04-01, 04-02, 04-03)

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
