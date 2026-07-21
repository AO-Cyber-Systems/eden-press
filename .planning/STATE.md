# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 4 — CLI (cmd/eden-press) — IN PROGRESS (1/8 TRDs; 04-02 cobra/koanf skeleton shipped this worktree); Objective 3 COMPLETE (9/9); Objective 7 (Dart binding) also unblocked

## Current Position

Objective: 4 of 9 (CLI — cmd/eden-press) — IN PROGRESS (1/8 TRDs); Objectives 0-3 complete
Job: 1 of 8 complete (this worktree) — wave-1 (04-02 cobra skeleton + go.mod deps + flag→Options + stdin/file input; sibling wave-1 TRD 04-01 press.Options.ThemeCSS extension executed in a SEPARATE parallel worktree, not yet confirmed merged here). Pending orchestrator reconcile at merge.
Status: Objective 4 IN PROGRESS — 04-02-TRD executed on this worktree: cmd/eden-press root-as-default cobra command tree (convert/watch/serve/preview), koanf-backed buildOptions (posflag Pitfall-5 three-arg instance guard), stdin(-)/file resolveInput with observable source, compiling stub seams (runConvert/runWatch/runServe/runPreview/loadConfigSources/themeCSS) for every downstream Wave-2+ CLI TRD, go.mod provisioned ONCE additively (go get only, never tidy) with all four CLI dep groups (cobra/fsnotify/koanf+parsers+providers/pkg-browser). press.Options.ThemeCSS wiring deferred pending 04-01's merge (documented seam, not a defect). CLI confirmed to import ONLY press/ (no chase/profiles/chromedp). Objective 3 remains COMPLETE; Objective 7 (Dart binding) remains unblocked in parallel.
Last activity: 2026-07-21 — 04-02-TRD executed (wave-1 CLI skeleton, this worktree): cmd/eden-press cobra command tree + koanf posflag wiring + stdin/file input resolution + compiling stub seams for 04-03..04-08; go.mod/go.sum additive-only (cobra v1.10.2, fsnotify v1.10.1, koanf/v2 v2.3.5 + parsers/{yaml,json,toml} + providers/{file,posflag,env}, pkg/browser — all transitive deps landed `// indirect`, expected pending a post-merge `go mod tidy` reconciliation step); 3 task commits (14046ac, daa1c71, b9f5480); TDD RED→GREEN evidence captured for Tasks 2-3 (6 tests, all pass); whole-repo build/vet/test, gofmt, addlicense, no-chromedp (count 0) all green; CLI-imports-only-press/ boundary manually confirmed via `go list -f '{{join .Imports "\n"}}' ./cmd/eden-press/...`. Prior: 03-09-TRD (Objective 3 capstone) — press.Render one-parse-two-sinks composition + no-chromedp CI gate; Objective 3 finished 9/9.

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

Last session: 2026-07-21 (04-02-TRD execution — wave-1 CLI skeleton: cobra/koanf command tree + stub seams)
Stopped at: Completed 04-02-TRD.md (this worktree): cmd/eden-press root-as-default cobra tree (root RunE=runConvert; convert/watch/serve/preview explicit subcommands; Pitfall-1 collision documented in root Long help); full persistent+per-mode flag surface (--theme/--theme-set/--profile/--math/--no-highlight/--highlight-style/--inline-svg/--config/--auto-fit-script persistent; --output/-o, --port[8321]/--host local) registered via registerPersistentFlags/registerXxxFlags; buildOptions(cmd) koanf-backed single resolution point mapping cfg-resolved values onto press.Options verbatim (unset flags stay zero — press.Render owns fallbacks); loadConfigSources posflag-only baseline via the load-bearing three-arg posflag.Provider(cmd.Flags(), ".", k) instance form (Pitfall 5), proven by TestPosflagInstanceGuard; resolveInput/resolveInputFrom stdin(-)/file resolution exposing an observable inputSource (Pitfall 8) without enforcing rejection policy; compiling stub seams for 04-03(runConvert)/04-04(loadConfigSources body)/04-05(themeCSS)/04-06(runWatch)/04-07(runServe)/04-08(runPreview), each returning a clear TRD-numbered not-implemented error; go.mod/go.sum provisioned ONCE additively via go get (cobra v1.10.2, fsnotify v1.10.1, koanf/v2 v2.3.5+parsers+providers, pkg/browser — never go mod tidy). press.Options.ThemeCSS wiring deferred pending sibling Wave-1 TRD 04-01's merge (documented one-line follow-up in buildOptions, not a defect). 3 task commits (14046ac, daa1c71, b9f5480) on worktree branch; whole-repo build/vet/test, gofmt, addlicense, no-chromedp (count 0) all green; CLI-imports-only-press/ boundary manually confirmed (no chase/profiles/chromedp). Objective 4 at 1/8 (this worktree) — pending orchestrator reconcile at merge alongside sibling wave-1 TRD 04-01 and subsequent waves.
Resume file: None
