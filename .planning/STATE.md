# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — IN PROGRESS; Objectives 0-2 complete
Job: 3 of 9 complete (03-01 API-03 ParseWithEngine seam + press.Options/Output + six battery deps — wave 1; 03-02 CORE-01 go:embed bundled Marp themes + name-keyed ThemeSet — wave 1; 03-05 CORE-07 chroma syntax highlighting via reused goldmark-highlighting/v2 + CSS-grounded chroma→.hljs remap — wave 2)
Status: Objective 3 in progress — 03-01 (wave-1 foundation) done; wave-1 03-02 and wave-2 battery TRDs unblocked (go.mod owned + provisioned, press types + ParseWithEngine seam available)
Last activity: 2026-07-21 — 03-01-TRD executed (wave-1 foundation): chase/markdown.ParseWithEngine added (ADDITIVE engine-parameterized twin of Parse — same SvgOptionsKey + resolved HeadingDividerKey pre-seed, parses via caller-supplied engine; seam.go/chase.go byte-for-byte untouched); press.Options + press.Output defined as the frozen API-03 surface (zero value = Marp-Core default, NoHighlight inverted, Sanitize nil = built-in policy); six battery deps provisioned into go.mod additively (chroma/v2 v2.27.0, goldmark-emoji v1.0.6, goldmark-highlighting/v2, bluemonday v1.0.27, latex2mathml, codeberg.org/go-latex/latex v0.3.0 — go-latex provisioned ONLINE, no defer/BLOCKER); goldmark-emoji↔goldmark v1.8.4 compat spike (research riskiest-item #3) closed with a passing test; 3 task commits (8af93ad, 9b74483, 60a12a4); whole-repo build/vet/test, gofmt, Go-source addlicense, Obj-1 corpus/cssdiff gates, Obj-2 grep-gate, and no-chromedp invariant all green

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

Last session: 2026-07-21 (03-05-TRD execution — CORE-07 chroma syntax highlighting, wave 2 of Objective 3)
Stopped at: Completed 03-05-TRD.md (CORE-07): reused goldmark-highlighting/v2 (chromahtml.WithClasses(true)) + a CSS-grounded chroma-short-class -> .hljs-* remap table derived from themes/{default,gaia,uncover}.css; SUMMARY committed on worktree branch. 4 task commits (79837f0, 2849ff9, eb0fee5, cf3df93). Objective 3 at 3/9 (03-01, 03-02, 03-05 done); 03-03/03-04/03-06/03-07/03-08 remain open (executed by separate parallel worktree agents).
Resume file: None
