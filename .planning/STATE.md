# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 2 — chase/model + chase/profile + profiles/slides (Marpit-in-Go) — COMPLETE; Objective 3 (press/ Batteries + Public API) planning next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — IN PROGRESS; Objectives 0-2 complete
Job: 3 of 9 complete (03-01 API-03 seam/Options/Output/deps + emoji-compat spike; 03-02 CORE-01 embedded themes; 03-04 CORE-06 emoji shortcode+unicode → twemoji — 03-03 wave-2 still pending)
Status: Objective 3 in progress — 03-04 (emoji battery) done via goldmark-emoji reuse (shortcode half) + a bespoke unicode-literal InlineParser (unicode half, same east.Emoji node/renderer); remaining wave-2 battery TRDs (03-03, 03-05..03-08) unblocked and independent
Last activity: 2026-07-21 — 03-04-TRD executed (CORE-06 native emoji, no JS): press/emoji.go wires github.com/yuin/goldmark-emoji's emoji.New(WithRenderingMethod(Twemoji)) as a goldmark.Extender (reused verbatim, configurable TwemojiOptions{Base,Ext} per Marp's base/ext contract); press/emoji_unicode.go adds ONLY the bespoke piece goldmark-emoji lacks — a rune-sequence→*definition.Emoji reverse index (seeded from definition.Github(), which exposes no enumeration method) feeding a halfspace-triggered InlineParser that emits the SAME east.Emoji AST node goldmark-emoji's own renderer already handles (no second NodeRenderer); 2 task commits (6f81008, e052a0b); all 5 TDD test-list cases pass; whole-repo build/vet/test, gofmt, addlicense, Obj-1 corpus/cssdiff, Obj-2 grep-gate, and no-chromedp invariant all green

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

Last session: 2026-07-21 (03-04-TRD execution — emoji battery, Objective 3 wave 2)
Stopped at: Completed 03-04-TRD.md (CORE-06): goldmark-emoji Twemoji shortcode reuse + bespoke unicode-literal InlineParser (same east.Emoji node/renderer, no second NodeRenderer); SUMMARY committed on worktree branch. Objective 3 at 3/9 (03-01, 03-02, 03-04 done; 03-03, 03-05..03-09 remain).
Resume file: None
