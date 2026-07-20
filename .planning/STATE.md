# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 0 — Conformance Corpus, Acceptance Gate & Attribution Bootstrap

## Current Position

Objective: 0 of 9 (Conformance Corpus, Acceptance Gate & Attribution Bootstrap)
Job: Not yet planned
Status: Ready to plan
Last activity: 2026-07-20 — ROADMAP.md and STATE.md created; REQUIREMENTS.md traceability filled (50/50 v1 requirements mapped, 100% coverage)

Progress: [░░░░░░░░░░] 0%

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

Last session: 2026-07-20 (roadmap creation)
Stopped at: ROADMAP.md + STATE.md written, REQUIREMENTS.md traceability updated; awaiting user approval of roadmap draft.
Resume file: None
