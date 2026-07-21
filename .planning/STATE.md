# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — IN PROGRESS; Objectives 0-2 complete
Job: 3 of 9 complete (03-01 API-03 seam/types/deps, 03-02 CORE-01 embedded themes, 03-03 CORE-03/CORE-04 strikethrough override + GFM/slug verify — wave 1/2 of 3 waves done)
Status: Objective 3 in progress — 03-01/03-02 (wave 1) and 03-03 (wave 2) done; remaining wave-2 battery TRDs (03-04..03-08) and the wave-3 capstone (03-09) unblocked/pending
Last activity: 2026-07-21 — 03-03-TRD executed (wave 2): press/strikethrough.go adds strikethroughOption(), a self-contained goldmark.Option registering a custom renderer.NodeRenderer for extast.KindStrikethrough at priority 100 (< goldmark's own StrikethroughHTMLRenderer priority 500), overriding GFM strikethrough to render <s>...</s> instead of goldmark's default <del>...</del> (Marp parity) — proven both directions (option renders <s>, bare NewEngine() still renders <del>) plus a mixed-fixture non-collision proof (table + <br> + <s> together); press/gfm_verify_test.go locks CORE-03's tables/hard-breaks and CORE-04's heading-ID slugs (h1 + h6 + dedup) as pure regression tests over already-baked-in chase/markdown.NewEngine features, zero new wiring; 2 task commits (f658442, be6774d); whole-repo build/vet/test, gofmt, Go-source addlicense, Obj-1 corpus/cssdiff gates, Obj-2 grep-gate, and no-chromedp invariant all green

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

Last session: 2026-07-21 (03-03-TRD execution — strikethrough <s> override + GFM/slug verify, wave 2 of Objective 3)
Stopped at: Completed 03-03-TRD.md (CORE-03, CORE-04) in an isolated worktree. SUMMARY committed alongside 2 task commits (f658442, be6774d). press/strikethrough.go + press/strikethrough_test.go + press/gfm_verify_test.go added. Remaining wave-2 battery TRDs (03-04 emoji, 03-05 chroma highlight, 03-06 math, 03-07 size/math directives + auto-fit, 03-08 sanitize) and the wave-3 capstone (03-09) pending — orchestrator merges this worktree then advances the wave.
Resume file: None
