# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 — press/ Batteries + Public API — COMPLETE (9/9 TRDs; press.Render public API shipped, CI-enforced zero-chromedp boundary); Objective 5 (convert/pdf + convert/png) underway on this worktree (1/5 TRDs); Objective 4 (CLI) / Objective 6 (PPTX) / Objective 7 (Dart binding) also planned/parallel workstreams

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete; Objective 5 (convert/pdf + convert/png) 1/5 TRDs complete on this worktree, pending orchestrator reconcile at merge
Job: 05-01 of 5 complete on this worktree — wave-1 convert/ bootstrap (chromedp provisioned additively + no-chromedp gate proven green, EXP-04 Chrome discovery chain, one-browser-many-tabs Session pool). Pending orchestrator reconcile at merge.
Status: Objective 5 IN PROGRESS (1/5 TRDs, this worktree) — 05-01 executed: chromedp v0.16.0 provisioned additively into go.mod/go.sum (convert/ is now the module's sole chromedp-touching package, scripts/check-no-chromedp.sh stays green); convert/doc.go + convert/convert.go establish the boundary + shared ImageFormat/Options vocabulary; convert/chrome.Discover implements the pure/DI-tested EXP-04 four-tier fallback chain; convert/chrome.Session delivers a one-browser-many-tabs pool (anchor-tab technique, verified against chromedp's own ExampleNewContext_manyTabs). Objective 3 (press/ Batteries + Public API) remains COMPLETE and unaffected — 03-09 capstone: press.Render one-parse-two-sinks composition wiring all six batteries, sanitize-last, embedded-theme pack, Comments aggregation; no-chromedp CI gate (0 chromedp in press/); capstone proves all 3 themes + every battery via a press-only consumer. chase.go untouched. Objective 7 (Dart binding) may begin independently.
Last activity: 2026-07-21 — 05-01-TRD executed (Objective 5, wave 1): chromedp@v0.16.0 (+cdproto/sysutil/go-json-experiment/gobwas transitives) added via `go get` (never `go mod tidy`) — fully additive diff, go directive bumped 1.25.0→1.26 as chromedp's own mandated minimum (mechanical side effect, not a manual edit); convert/chrome.Discover(DiscoverOptions) resolves BrowserPath>CHROME_PATH env>PATH auto-detect(empty path, delegates to chromedp)>ErrChromeNotFound(documents headless-shell/Chrome-for-Testing remedy), pure and DI-tested (Getenv/LookPath fields), 5 hand-built table-driven tests all pass; convert/chrome.Session.New() builds one chromedp.NewExecAllocator with NoSandbox/unique-UserDataDir/disable-dev-shm-usage/force-device-scale-factor=1/lang=en-US/TZ=UTC baked in as DEFAULTS, NewTab() derives from an internal eagerly-Run anchor tab context (rootCtx) rather than the raw allocator context directly — RECORDED deviation: the TRD's simplified sketch would have silently spawned a new Chrome process per NewTab() call, caught pre-commit by cross-checking chromedp v0.16.0's real source; multi-tab smoke test is Chrome-presence-gated and skips cleanly (no system Chrome in this sandbox — the exact EXP-04/05-05 no-system-Chrome case). 3 task commits (0efde08, 405c4b6, 9d5722b); whole-repo build/vet/test (21 packages), gofmt, addlicense (convert/), no-chromedp (0 in press/chase/profiles) all green. EXP-04 intentionally left Pending in REQUIREMENTS.md — also spans 05-02 (font provisioning) and 05-05 (CI hardening).

Progress: [██████████] 100% (27/27 TRDs across fully-merged objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4, Objective 3: 9/9); Objective 5 in progress on this worktree: 1/5 TRDs (05-01 complete), pending orchestrator reconcile at merge

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

Last session: 2026-07-21 (05-01-TRD execution — Objective 5 wave-1: convert/ bootstrap, Chrome discovery + Session pool)
Stopped at: Completed 05-01-TRD.md (EXP-04, discovery half): chromedp v0.16.0 provisioned additively into go.mod/go.sum via `go get` (go directive 1.25.0→1.26, chromedp's own mandated minimum — mechanical, not hand-edited); convert/doc.go + convert/convert.go establish convert/ as the module's ONLY chromedp-touching package (ImageFormat PNG/JPEG + Options{BrowserPath} shared vocabulary for 05-03/05-04); convert/chrome.Discover(DiscoverOptions) implements the pure, DI-tested (Getenv/LookPath fields) four-tier EXP-04 fallback chain — BrowserPath > CHROME_PATH env > PATH auto-detect (empty path, delegates to chromedp) > ErrChromeNotFound (documents headless-shell/Chrome-for-Testing remedy) — 5 hand-built table-driven tests pass; convert/chrome.Session.New() builds one chromedp.NewExecAllocator with NoSandbox/unique-UserDataDir/disable-dev-shm-usage/force-device-scale-factor=1/lang=en-US/TZ=UTC baked in as DEFAULTS (not opt-ins), NewTab() hands out tabs on the SAME browser via an internal eagerly-Run anchor tab context (rootCtx) — RECORDED Rule-2 deviation: the TRD's simplified sketch (NewTab deriving straight from the raw allocator context) would have silently spawned a new Chrome process per call; caught pre-commit against chromedp v0.16.0's real source (its own ExampleNewContext_manyTabs), fixed in the initial design before any code was committed. Multi-tab smoke test is Chrome-presence-gated and skips cleanly (no system Chrome in this sandbox). 3 task commits (0efde08, 405c4b6, 9d5722b) on worktree branch; all gates green (gofmt/build/vet/test-21-packages/no-chromedp-0/addlicense). Objective 5 at 1/5 (this worktree) — pending orchestrator reconcile at merge. EXP-04 left Pending in REQUIREMENTS.md (also spans 05-02, 05-05). Next up on this workstream: 05-02 (shared determinism recipe + STIX Two Math font provisioning).
Resume file: None
