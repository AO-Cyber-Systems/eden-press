# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** ROADMAP COMPLETE. All 9 objectives (0–8) + 2 insertions (4.1 CLI Agent Interface, 5.1 Export Binary) done and verified (every VERIFICATION.md = passed).

## Current Position

Objective: ALL COMPLETE — 0,1,2,3,4,4.1,5,5.1,6,7,8. Nothing pending on the roadmap.
Job: Objective 8 (final) — 7/7 TRDs complete: latex2mathml vendored fork (internal/latex2mathml via go.mod replace) + all 8 math spike cases at KaTeX-parity (TestSpikeCorpus, structural MathML-DOM asserts), fallback-trigger rule finalized to the permanent Chromium MathML-Core structural ceiling (\tag/\label/numbered-align → PNG), STIX Two Math OTF+WOFF2 (stipub v2.13) + MATH-table survival/smoke tests, and Flutter-only auto-fit (native TextPainter fit in bind/dart; --auto-fit-script/BrowserFitJS/browser-fit.js removed → HTML ships zero JS, markers still emitted).
Status: Full pipeline shipped & verified — press.Render (chrome-free HTML/CSS/Model) + batteries; eden-press CLI (convert/watch/serve/preview + --format json|pptx agent interface, chromedp-free, CI-enforced); eden-press-export (separate chrome-permitting pdf/png binary); convert/pptx (native OOXML); Dart binding (capi/wasm/native + JS-free Flutter surface); math at KaTeX-parity with a tested fallback. Pushed to github.com/AO-Cyber-Systems/eden-press (private).

Progress: [##########] 100% — all roadmap objectives + insertions complete and verified.

## Accumulated Context

> Decisions and performance metrics are logged in STATE_ARCHIVE.md.

### Pending Todos

None — roadmap complete.

### Blockers/Concerns

- All decision gates resolved (chase exported; hand-rolled OOXML; standard Go for WASM; MathML fallback = structural-ceiling rule; auto-fit = Flutter-only). Owned follow-up: the vendored internal/latex2mathml fork is now a maintenance surface.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 1 | disable-gpu and surface Chrome stderr so browser launch cannot hang silently | 2026-08-17 | 010a848 | [1-disable-gpu-and-surface-chrome-stderr-so](./quick/1-disable-gpu-and-surface-chrome-stderr-so/) |

## Session Continuity

Last activity: 2026-08-17 - Completed quick task 1: disable-gpu and surface Chrome stderr so browser launch cannot hang silently
Last session: 2026-07-22 — Objective 8 (Math-Fidelity Hardening + Auto-Fit) built, merged, verified (passed); roadmap complete.
Stopped at: Quick task 1 landed on branch `quick/chrome-disable-gpu` (3 commits: 83fc6d1, 21a544e, 010a848) — NOT merged, NOT tagged, NOT pushed. Consumer wiring (aodex export-sidecar passing Options.BrowserLog) is deliberately out of scope and still pending.
Resume file: None
