# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — IN PROGRESS; Objectives 0-2 complete
Job: 3 of 9 complete (wave-1: 03-01 ParseWithEngine seam + API-03 surface + deps, 03-02 embedded themes; wave-2: 03-06 CORE-08 BASELINE math — this branch; other wave-2 TRDs execute in parallel worktrees, reconciled at merge)
Status: Objective 3 in progress — 03-06 (CORE-08 math battery) done on this worktree; parallel wave-2 TRDs (03-03/04/05/07/08) in flight; 03-09 compose is wave 3
Last activity: 2026-07-21 — 03-06-TRD executed (wave-2 riskiest battery): press/math subpackage built from scratch (no reusable goldmark-math library) — bespoke $/$$ InlineParser + custom mathNode + routing NodeRenderer; test-first construct-detection predicate needsFallback (\tag|\label|\begin{aligned|align|alignat|cases|array}) pre-scans RAW source; common $…$/$$…$$ → native MathML via vendored latex2mathml; heavy constructs → PNG-ONLY base64 data-URI <img> via go-latex/latex drawtex/drawimg (drawtex has NO SVG canvas — framing corrected to PNG); Pandoc currency guard; MathMode "off" disables math; RECORDED surprise: go-latex/mtex PANICS on superscripts + all \begin{…} envs → fallback wraps recover() and degrades to alt-only <img> stub (real PNG path proven with \frac), so aligned-family currently renders the graceful stub — Objective 8 owns raster quality + final fallback rule; recover guard also on latex2mathml (8 known converter bugs → Obj 8); 2 task commits (76610d3, 6116618); whole-repo build/vet/test, gofmt, Go-source addlicense, Obj-1 corpus/cssdiff, Obj-2 grep-gate, no-chromedp (count 0) all green; go.mod/go.sum additive-only (gg/freetype/x/image/x/text transitive; go-latex+latex2mathml promoted to direct)

Progress: [█████████░] 96% (21/27 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4, Objective 3: 3/9)

## Accumulated Context

> Decisions and performance metrics are logged in STATE_ARCHIVE.md.

### Pending Todos

None yet.

### Blockers/Concerns

- Decision gate open: `chase/*` internal vs. exported Go (resolve during Objective 2 planning).
- Decision gate open: hand-rolled OOXML vs. any newly-emerged permissive Go PPTX lib (re-confirm at Objective 6 planning — `unioffice`/forks are AGPLv3, rejected per research).
- Decision gate open: standard Go vs. TinyGo for the WASM target (resolve before Objective 7's WASM-specific code is written — functional risk on reflection-heavy JSON/YAML paths, not just size).
- Decision gate open: concrete MathML fallback-trigger rule + final auto-fit mechanism (resolve during Objective 8 planning).
- CSS-AST diff tooling (Objective 0) is genuinely new/unproven engineering — no spike precedent exists; budget accordingly at planning time.

## Session Continuity

Last session: 2026-07-21 (03-06-TRD execution — wave-2 CORE-08 BASELINE math battery)
Stopped at: Completed 03-06-TRD.md (CORE-08): press/math from-scratch goldmark battery — $/$$→MathML via latex2mathml + test-first construct-detection predicate + PNG-only go-latex fallback; SUMMARY committed on worktree branch. Objective 3 at 3/9 (this worktree). NOTE for reconcile: parallel wave-2 worktrees (03-03/04/05/07/08) also advance Objective-3 tracking — the orchestrator reconciles the true count at merge. 03-08 sanitize MUST allow <math>+children and the fallback <img> (see 03-06-SUMMARY "For 03-08"); 03-09 wires press/math.Option(opts.MathMode) into the compose engine.
Resume file: None
