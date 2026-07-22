# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objectives 0–7 COMPLETE and verified (all VERIFICATION.md = passed); Objective 4.1 (CLI Agent Interface, AI-usability) COMPLETE. Only Objective 8 (Math-Fidelity Hardening + Auto-Fit) remains.

## Current Position

Objective: 0–7 of 9 COMPLETE (23 TRDs across Obj 4/5/6/7 built in parallel workstreams + Obj 4.1 insertion). Objective 8 IN PROGRESS — 08-01 (fork+vendor latex2mathml) + 08-05 (STIX Two Math WOFF2) + 08-06 (remove viewer-side JS auto-fit, web half of criterion 4) + 08-07 (Flutter-native auto-fit, Flutter half of criterion 4) COMPLETE; 08-02 (converter patches A: big-operator limit stacking via munderover tag-switch [Open Q1 resolved] + \sqrt[n] radicand fix, structural MathML-DOM regression tests — 4/8 spike cases at KaTeX-parity) COMPLETE, 5/7 TRDs; commits 113ce04/c77264e/063acb7/391bfe6/66477ec/53d8238/6e65a95. 08-03 (converter patches B) edits the same files strictly-sequentially next.
Job: Objective 4.1 (CLI Agent Interface) — 2/2 TRDs complete: `--format json` structured Output envelope + machine-readable errors/exit-codes (04.1-01); Chrome-free `--format pptx` via convert/pptx.ToPPTX + no-chromedp gate extended to ./cmd/... + AGENTS.md (04.1-02).
Status: eden-press CLI is a complete agent interface — `--format html|json|pptx`, JSON envelope of the full press.Output (model blocks/outline/comments), stable exit codes, AGENTS.md; CLI stays chromedp-free (CI-enforced across press/chase/profiles/bind/cmd). PNG/PDF intentionally out (separate export path). preview untouched (default OS browser).

Progress: [██████████] Objectives 0–7 complete + Obj 4.1 complete; Objective 8 pending.

## Accumulated Context

> Decisions and performance metrics are logged in STATE_ARCHIVE.md.

### Pending Todos

None.

### Blockers/Concerns

- Objective 8 decision gates (resolve at planning): concrete MathML fallback-trigger rule; final auto-fit mechanism (Flutter TextPainter vs CSS cqw/SVG-text vs drop).
- Follow-up (optional, not blocking): wire PNG/PDF export into a separate Chrome-permitting path/binary (kept out of the chromedp-free core CLI by design).

## Session Continuity

Last session: 2026-07-22 — Objective 4.1 (CLI Agent Interface) built + merged; formal verifiers run for Objectives 4/5/6/7 (all passed).
Stopped at: Objective 4.1 complete on main. Objective 8 is the only remaining roadmap objective.
Resume file: None

08-07-TRD.md (Flutter-native shrink-only auto-fit, bind/dart — criterion 4 Flutter half) complete on this worktree branch: `computeFitFontSize` + `FitText` widget wired into `EdenPressView`'s heading case, zero JS, TDD unit + widget tests green; see 08-07-SUMMARY.md. Awaits merge/reconciliation with sibling Objective-8 wave TRDs.

08-05-TRD.md (STIX Two Math WOFF2 companion + Chrome-gated MATH-table pixel-check smoke — criterion 3) complete on this worktree branch: official stipub v2.13 WOFF2 embedded + `FontFaceDataURIWoff2()` additive accessor, MATH-table survival verified two ways, `TestStixMathTableSmoke` Chrome-gated CI smoke (SKIPs cleanly without Chrome, PASSES for real with it), NOTICE extended; commits 41f224f/69e8941; see 08-05-SUMMARY.md.

08-06-TRD.md (remove viewer-side JS auto-fit from the HTML/CLI path — criterion 4 web half) complete on this worktree branch: `--auto-fit-script` flag + `AutoFitScript` splice + all four caller reads removed, `press.BrowserFitJS`/`themes/browser-fit.js` surface fully deleted, `TestAssembleHTMLNoViewerSideAutoFitJS` proves zero `<script>` in default HTML while CORE-09 `<!--fit-->`/shrink markers stay emitted unchanged (feeding 08-07), NOTICE + AGENTS.md updated; commits 391bfe6/66477ec/53d8238/6e65a95; see 08-06-SUMMARY.md.
