# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objectives 0–7 COMPLETE and verified (all VERIFICATION.md = passed); Objective 4.1 (CLI Agent Interface, AI-usability) COMPLETE. Only Objective 8 (Math-Fidelity Hardening + Auto-Fit) remains.

## Current Position

Objective: 0–7 of 9 COMPLETE (23 TRDs across Obj 4/5/6/7 built in parallel workstreams + Obj 4.1 insertion). Objective 8 not started (gated on Obj 3 + 7, both done → unblocked).
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
