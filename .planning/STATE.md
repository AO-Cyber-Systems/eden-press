# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — IN PROGRESS; Objectives 0-2 complete
Job: 3 of 9 complete (03-01 wave-1 foundation; 03-02 wave-1 embedded themes; 03-07 wave-2 size/math global directives + auto-fit markers (CORE-02, CORE-09)) — wave-2 battery TRDs (03-03..03-06, 03-08) executing in parallel worktrees, not yet reconciled here
Status: Objective 3 in progress — 03-01/03-02 (wave-1) done; 03-07 (wave-2) done from this worktree's perspective; remaining wave-2 battery TRDs (03-03, 03-04, 03-05, 03-06, 03-08) and wave-3 capstone (03-09) unblocked/in-flight
Last activity: 2026-07-21 — 03-07-TRD executed (wave-2): CORE-02 closed via two new chase/directive.CoerceGlobal passthrough cases ("size"/"math", mirroring style/lang) so comment-form `<!-- size: 4:3 -->`/`<!-- math: mathml -->` classifies as a directive instead of a presenter note (front-matter form already reached Output.Meta via buildMeta); CORE-09 closed via new press/autofit.go — a goldmark.Extender-wrapped autofitOption() emitting a data-auto-scaling="fit" attribute on a `# <!--fit-->` heading and a marp-fit-shrink wrapper class on fenced-code/math-shaped blocks (MARKERS ONLY, no runtime JS; @auto-scaling remains theme-CSS-only); 2 task commits (d5108c1, ff37ef9), SUMMARY commit (4b79d5a); whole-repo build/vet/test, gofmt, addlicense, Obj-1 corpus/cssdiff gate, Obj-2 grep-gate, and no-chromedp invariant all green; 0 deviations, both TDD cycles RED→GREEN on first attempt

Progress: [████████░░] 78% (21/27 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4, Objective 3: 3/9)

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

Last session: 2026-07-21 (03-07-TRD execution — wave-2 of Objective 3, CORE-02 + CORE-09)
Stopped at: Completed 03-07-TRD.md (CORE-02, CORE-09): chase/directive.CoerceGlobal size/math passthrough + press/autofit.go marker emitter; SUMMARY committed (4b79d5a) on worktree branch agent-a73bc4aee8e9eaa9a. Executed in parallel with other wave-2 battery TRDs (03-03..03-06, 03-08) in sibling worktrees — not yet merged/reconciled to main. Objective 3 at 3/9 from this worktree's perspective.
Resume file: None
