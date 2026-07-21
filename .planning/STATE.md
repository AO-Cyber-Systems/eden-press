# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 — press/ Batteries + Public API — COMPLETE (9/9 TRDs; press.Render public API shipped, CI-enforced zero-chromedp boundary); Objective 4 (CLI) / Objective 7 (Dart binding) next

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete
Job: 9 of 9 complete — wave-1 (03-01 ParseWithEngine seam + API-03, 03-02 embedded themes), wave-2 (03-03 strikethrough, 03-04 emoji, 03-05 highlight, 03-06 math, 03-07 size/math directives + autofit, 03-08 sanitize), wave-3 (03-09 press.Render compose API-01 + no-chromedp gate API-02 + capstone — this worktree). Pending orchestrator reconcile at merge.
Status: Objective 3 COMPLETE — 03-09 (capstone) executed on this worktree: press.Render one-parse-two-sinks composition wiring all six batteries, sanitize-last, embedded-theme pack, Comments aggregation; no-chromedp CI gate (0 chromedp in press/); capstone proves all 3 themes + every battery via a press-only consumer. chase.go untouched. Objective 7 (Dart binding) may begin.
Last activity: 2026-07-21 — 06-01-TRD executed on an isolated worktree (Obj-6 wave-1, SHARED PREREQUISITE unblocking Obj-7/DART-04): chase/model schema-v2 — additive Section.Blocks []Block union {paragraph|list|code(source+language)|math(rawTeX+display)|heading}, materialized in the SAME single read-only Build ast.Walk (no second parse; TestBuildNonMutation stays green), SchemaVersion bumped v1→v2 with every new field omitempty (block-less doc's JSON byte-identical to v1 except the version string); press/math gained additive MathRaw()/MathDisplay() pure getters (zero parse/render/HTML change) reached from chase/model via a duck-typed rawMath interface so chase/model NEVER imports press/math (go list -deps grep press/math = 0; cycle-free, no-chromedp closure intact); math-only paragraph structurally skipped so raw TeX is not double-emitted as prose; 3 task commits (5db9bb0, ab20165, 8ad7242); whole-repo build/vet/test (Obj-0 corpus/cssdiff/htmldiff, Obj-1, Obj-2 model, Obj-3 press all green), gofmt, no-chromedp (count 0) all green; go.mod untouched (stdlib + existing goldmark only). Pending orchestrator reconcile at merge. PRIOR: 03-06-TRD executed (wave-2 riskiest battery): press/math subpackage built from scratch (no reusable goldmark-math library) — bespoke $/$$ InlineParser + custom mathNode + routing NodeRenderer; test-first construct-detection predicate needsFallback (\tag|\label|\begin{aligned|align|alignat|cases|array}) pre-scans RAW source; common $…$/$$…$$ → native MathML via vendored latex2mathml; heavy constructs → PNG-ONLY base64 data-URI <img> via go-latex/latex drawtex/drawimg (drawtex has NO SVG canvas — framing corrected to PNG); Pandoc currency guard; MathMode "off" disables math; RECORDED surprise: go-latex/mtex PANICS on superscripts + all \begin{…} envs → fallback wraps recover() and degrades to alt-only <img> stub (real PNG path proven with \frac), so aligned-family currently renders the graceful stub — Objective 8 owns raster quality + final fallback rule; recover guard also on latex2mathml (8 known converter bugs → Obj 8); 2 task commits (76610d3, 6116618); whole-repo build/vet/test, gofmt, Go-source addlicense, Obj-1 corpus/cssdiff, Obj-2 grep-gate, no-chromedp (count 0) all green; go.mod/go.sum additive-only (gg/freetype/x/image/x/text transitive; go-latex+latex2mathml promoted to direct)

Progress: [██████████] 100% (27/27 TRDs across currently-planned objectives — Objective 0: 6/6, Objective 1: 8/8, Objective 2: 4/4, Objective 3: 9/9)

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

Last session: 2026-07-21 (03-09-TRD execution — wave-3 capstone: press.Render public API + no-chromedp gate)
Stopped at: Completed 03-09-TRD.md (API-01, API-02): press.Render(md, opts) one-parse-two-sinks composition — builds engine := markdown.NewEngine(pressExtraOpts...) bundling all six batteries, ParseWithEngine exactly ONCE (runtime-proven via a counting wrapper around a parseWithEngine seam var), forks to HTML render (sink 1) + model.Build (sink 2), remapHLJS, sanitize LAST via sanitize.Sanitize (case-restores foreignObject/viewBox — corrected the sketch's bare Policy().Sanitize which would break the inline-SVG chain), packs the opts→front-matter→"default" theme from the embedded ThemeSet, aggregates Comments from Model.Sections[*].Notes. no-chromedp gate: scripts/check-no-chromedp.sh + make check-no-chromedp + CI step beside addlicense (0 chromedp in press/chase/profiles). Capstone (external press_test, press-only import) proves all 3 themes + every battery. chase.go UNTOUCHED (empty diff vs main). 3 task commits (fb4fe44, 3c35e8a, 9a49d47) on worktree branch; all 8 gates green (gofmt/build/vet/test/Obj-1 corpus-cssdiff/Obj-2 grep-gate/addlicense/no-chromedp). Objective 3 at 9/9 (this worktree) — pending orchestrator reconcile at merge. Objective 7 (Dart binding) unblocked.
Resume file: None
