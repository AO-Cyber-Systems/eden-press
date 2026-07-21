# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — IN PROGRESS; Objectives 0-2 complete
Job: 1 of 9 complete (03-02 compiled-CSS theme extraction + go:embed + name-keyed ThemeSet (CORE-01) — wave 1; other TRDs pending/parallel)
Status: Objective 3 in progress — 03-02 (CORE-01) executed in an isolated worktree; awaiting orchestrator merge/reconciliation
Last activity: 2026-07-21 — 03-02-TRD executed: extended the tools/corpus-gen npm oracle with extract-themes.mjs to pull marp-core v4.4.0's OWN fully-compiled per-theme CSS (marp.themeSet.get(name).css) VERBATIM into themes/{default,gaia,uncover}.css (leading /*! @theme */ block hoisted so chase/theme.Load parses it), vendored lib/browser.js as themes/browser-fit.js (Marp MIT header), go:embed'd them (themes/embed.go) and built press/themes.ThemeSet (name-keyed, mirrors chase.go packCSS's NewThemeSet+Load+Add) — 7 press/themes tests green incl. a format-insensitive corpus shared-rule gate (gaia 64/84 rules+20/21 hljs, uncover 52/68+15/15); NOTICE credits the 3 themes + github-markdown-css (4th asset) + browser-fit.js, CI addlicense -ignore 'themes/**'; 3 task commits (b623195, 0b29b86, 2289ca1); whole-repo build/vet/test, gofmt, addlicense, Obj-1 cssdiff/corpus, Obj-2 grep-gate, and no-chromedp check all green

Progress: [█████████░] 95% (19/20 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4, Objective 3: 1/9 [03-02 done])

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

Last session: 2026-07-21 (03-02-TRD execution — bundled Marp themes / CORE-01, wave 1 of Objective 3)
Stopped at: Completed 03-02-TRD.md (CORE-01) in an isolated worktree. SUMMARY committed alongside 3 task commits (b623195, 0b29b86, 2289ca1). themes/*.css + browser-fit.js embedded; press/themes.ThemeSet wired to chase/theme.Pack. Remaining Objective-3 TRDs (03-01, 03-03..03-09) pending — orchestrator merges this worktree then advances the wave.
Resume file: None
