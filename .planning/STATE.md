# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 2 of 9 (chase/model + chase/profile + profiles/slides) — COMPLETE; Objectives 0-1 also complete
Job: 4 of 4 complete (02-01 chase/model docmodel builder (MODEL-01), 02-02 chase/profile interface + registry (MODEL-03), 02-03 profiles/slides + de-hardcode chase/theme + grep-gate (MODEL-04), 02-04 chase.go one-parse-two-sinks entrypoint (MODEL-02) — wave 3/3 done)
Status: Objective 2 complete — ready for Objective 3 planning (press/ Batteries + Public API)
Last activity: 2026-07-21 — 02-04-TRD executed (capstone): chase/markdown.RenderDoc added (render an already-parsed doc, no re-parse); chase.Render(md) implemented as the internal one-parse-two-sinks entrypoint returning Output{HTML, CSS, Model, Meta} — ONE markdown.Parse call forks to RenderDoc (HTML) + model.Build (Model) on the SAME *ast.Document, CSS packed via profile-parameterized theme.Pack; MODEL-02 proven structurally (byte-identical HTML before/after Build) plus a 4-case Objective-1 corpus smoke test (marp-basic/slide-split/paginate/header-footer) with zero HTML regression; 3 task commits (8c10088, 9f0d142, 1e2629d) + 1 docs commit (1b2a0f7); whole-repo build/vet/test, gofmt, addlicense, Obj-1 cssdiff/corpus gates, and Obj-2 grep-gate all green throughout

Progress: [██████████] 100% (18/18 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4; Objective 3 not yet planned)

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

Last session: 2026-07-21 (02-04-TRD execution — final/capstone TRD of Objective 2)
Stopped at: Completed 02-model-profile-04-TRD.md (MODEL-02); Objective 2 now fully complete (4/4). SUMMARY committed 1b2a0f7. Objective 3 (press/ Batteries + Public API) planning next.
Resume file: None
