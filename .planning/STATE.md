# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 — press/ Batteries + Public API — COMPLETE (9/9 TRDs); Objective 4 (CLI) — IN PROGRESS (1/8 TRDs; 04-01 Wave-0 enabler landed); Objective 7 (Dart binding) also open

## Current Position

Objective: 4 of 9 (CLI (cmd/eden-press)) — IN PROGRESS (1/8 TRDs); Objectives 0-3 complete
Job: 1 of 8 complete — wave-1 (04-01 press.Options.ThemeCSS additive extension + BrowserFitJS re-export, Wave-0 enabler for CLI-05 — this worktree). Pending orchestrator reconcile at merge.
Status: Objective 4 IN PROGRESS — 04-01-TRD executed on this worktree: press.Options gains an additive `ThemeCSS []string` field (raw custom-theme CSS text; press/ never touches the filesystem, keeping Render a pure function of (md, opts)); press.Render's packThemeCSS registers each entry via the SAME chase/theme.Load + ThemeSet.Add path the 3 embedded themes use, so a custom theme resolves through the existing opts.Theme / front-matter `theme:` chain; press.BrowserFitJS() re-exports press/themes.BrowserFitJS() at the package root so the CLI can splice the auto-fit script importing only press/. Purely additive: Options{} zero-value output proven byte-identical via a golden regression test (HTML pinned verbatim; CSS pinned by length+SHA-256, since the packed default theme is ~47KB). Obj-3 capstone + full press test suite + whole-repo build/vet/test + no-chromedp all green; 2 task commits (0307887, 5f5de94). Unblocks 04-05 (--theme-set) and 04-03 (--auto-fit-script).
Last activity: 2026-07-21 — 04-01-TRD executed (wave-1 Wave-0 enabler): see Status above.

Progress: [████████░░] 80% (28/35 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4, Objective 3: 9/9, Objective 4: 1/8)

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

Last session: 2026-07-21 (04-01-TRD execution — wave-1 Wave-0 enabler: press.Options.ThemeCSS + BrowserFitJS re-export)
Stopped at: Completed 04-01-TRD.md: Task 1 added the additive `press.Options.ThemeCSS []string` field (after `Sanitize`, no reordering) plus `press/browserjs.go`'s `BrowserFitJS()` re-export of `press/themes.BrowserFitJS()`, proven via a golden additive-parity test (`Render("# Hi\n", Options{})` HTML pinned verbatim, CSS — the ~47KB packed default theme — pinned by length+SHA-256) and a re-export identity test. Task 2 threaded `opts.ThemeCSS` through `packThemeCSS`: each entry registered via `theme.Load(css, p.UnitElement(), p.Sizes().ByName)` + `ts.Add(th)`, inserted after `themes.ThemeSet(...)` and before `resolveThemeName(...)` — the exact intake the 3 embedded themes use — with a malformed block (missing leading `/* @theme name */`) surfacing as a wrapped `press: Render: load custom theme CSS` error. RED/GREEN reconstructed and verified for all 5 new tests (temporarily reverting/removing the just-added code, confirming the expected failure, then restoring). 2 task commits (0307887, 5f5de94) on worktree branch; all 6 gates green (gofmt/build/vet/whole-repo test incl. Obj-3 capstone/no-chromedp/addlicense). `git diff press/options.go` confirms zero non-additive changes. Objective 4 at 1/8 (this worktree) — pending orchestrator reconcile at merge. Unblocks 04-05 (--theme-set) and 04-03 (--auto-fit-script).
Resume file: None
