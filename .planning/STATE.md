# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 1 — chase/markdown + chase/directive + chase/theme (Marpit-in-Go)

## Current Position

Objective: 1 of 9 (chase/markdown + chase/directive + chase/theme (Marpit-in-Go)) — COMPLETE; Objective 0 also complete
Job: 8 of 8 complete (01-01 selector-rewriter, 01-02 directive state machine, 01-03 Stylesheet model, 01-04 CSS scoping pipeline, 01-05 two-phase seam/slide-split/container, 01-06 directive apply, 01-07 backgrounds/inline-SVG, 01-08 integration — corpus + cssdiff gates + raster check (PARSE-01) all executed; wave 5/5 done)
Status: Objective 1 complete — ready for Objective 2 planning (chase/model + chase/profile + profiles/slides)
Last activity: 2026-07-21 — 01-08-TRD executed: two-phase Parse()/Render() seam (PARSE-01) formalized in chase/markdown/seam.go; NEW conformance/runner/chase_corpus_test.go drives the 18-case Marp corpus through the chase engine (10 PASS incl. all 9 Marpit-mechanic cases + marp-gfm-table, 8 BLOCKED on Objective-3 CORE-* batteries via explicit skip-map, 0 unexplained failures); chase/theme/pack_conformance_test.go cssdiff.Equal gate green for stress+scaffold; inline-SVG rasterization human-verified (auto-approved, autonomous run) via browser screenshot; [Rule 1] headingDivider display-value bug fixed in apply.go; 4 task commits (16fcc47, 902e93d, e9adebf, dd68671) + 1 docs commit (729a9ef)

Progress: [██████████] 100% (14/14 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8; Objective 2 not yet planned)

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

Last session: 2026-07-21 (01-08-TRD execution — final TRD of Objective 1)
Stopped at: Completed 01-chase-framework-08-TRD.md (PARSE-01); Objective 1 now fully complete (8/8). SUMMARY committed 729a9ef. Objective 2 (chase/model + chase/profile + profiles/slides) planning next.
Resume file: None
