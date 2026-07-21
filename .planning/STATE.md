# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 1 — chase/markdown + chase/directive + chase/theme (Marpit-in-Go)

## Current Position

Objective: 1 of 9 (chase/markdown + chase/directive + chase/theme (Marpit-in-Go)) — Objective 0 complete
Job: 4 of 8 complete (01-01 selector-rewriter, 01-02 directive state machine, 01-03 Stylesheet model, 01-04 ordered two-tier CSS scoping pipeline all executed; wave 2 done)
Status: Executing — Wave 2 complete (01-04 delivered THEME-03); 01-05 (chase/markdown two-phase seam + slide-split + container, PARSE-05) next, wave 2/3
Last activity: 2026-07-20 — 01-04-TRD executed: Tier-1 Theme.Load (nesting down-level, :root mark) + Tier-2 ThemeSet.Pack (@import-theme resolve, scaffold prepend, advanced-bg injection, pagination neutralize, selector-scope via chase/theme/selector, specificity rewrite), cssdiff.Equal-verified stress+scaffold fixtures (THEME-03); 3 task commits + 1 docs commit

Progress: [███████░░░] ~71% (10/14 TRDs across planned objectives; Objective 1: 4/8)

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

Last session: 2026-07-20 (01-04-TRD execution)
Stopped at: Completed 01-chase-framework-04-TRD.md (THEME-03); two-tier CSS scoping pipeline (Theme.Load + ThemeSet.Pack) verified via cssdiff.Equal fixtures; SUMMARY committed 0ac4f74; 01-05-TRD next.
Resume file: None
