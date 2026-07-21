---
objective: 03-press-batteries-api
job: "09"
subsystem: public-api
tags: [press, one-parse-two-sinks, batteries, sanitize, themes, no-chromedp, capstone, API-01, API-02]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "chase/markdown.ParseWithEngine/NewEngine seam (03-01), embedded press/themes ThemeSet (03-02), strikethrough (03-03), emoji (03-04), highlight+remapHLJS (03-05), press/math.Option (03-06), autofit (03-07), press/sanitize.Sanitize (03-08), frozen press.Options/Output types (03-01)"
provides:
  - "press.Render(md string, opts Options) (Output, error) — the PUBLIC API-01 surface: one-parse-two-sinks composition wiring ALL six battery options into a single battery-laden engine, sanitizing last, packing embedded themes, aggregating Comments"
  - "flattenNotes(*model.Document) []string — pure aggregation of Model.Sections[*].Notes into Output.Comments (no second AST walk)"
  - "scripts/check-no-chromedp.sh + make check-no-chromedp + CI step — API-02 mechanical enforcement that press/chase/profiles stay chromedp-free"
  - "Capstone proof: a consumer importing ONLY press/ renders a complete deck under all 3 themes + every battery (the Objective-7-begin gate)"
affects: [objective-4-cli, objective-5-exporters, objective-7-dart-binding]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SIBLING composition (not a wrapper): press.Render builds its OWN battery-laden engine via markdown.NewEngine(pressExtraOpts...) and drives markdown.ParseWithEngine directly — chase/chase.go is neither called nor modified, preserving the 'existing callers unaffected' invariant and avoiding a second parse"
    - "One-parse-two-sinks with batteries: exactly ONE ParseWithEngine call, then the SAME finalized *ast.Document forks to engine.Renderer().Render (sink 1, HTML) + model.Build (sink 2, Model) — proven at runtime by a counting wrapper around a package-level parseWithEngine seam variable"
    - "Sanitize LAST over the composed HTML string only (never CSS/Model): nil opts.Sanitize selects the full sanitize.Sanitize pipeline (Policy().Sanitize + SVG foreignObject/viewBox case restoration); a non-nil *bluemonday.Policy override replaces the built-in wholesale"
    - "Theme resolution chain opts.Theme -> Meta.Directives[\"theme\"] (front matter) -> \"default\" (the bundled Marp default deck theme, NOT the bare scaffold — press deviates from chase.go's scaffold-only default to match Marp Core)"
    - "CSS inline-SVG mode derived from pc (svgEnabled) OR opts.InlineSVG, mirroring chase.go's packCSS: since ParseWithEngine unconditionally enables inline-SVG, the HTML always carries the <svg><foreignObject> wrapper and the packed CSS container chain must match"
    - "no-chromedp CI gate: go list -deps ./press/... ./chase/... ./profiles/... | grep -i chromedp -> fail if non-empty; wired beside addlicense in CI"

key-files:
  created:
    - press/press.go
    - press/press_test.go
    - press/comments.go
    - press/capstone_test.go
    - scripts/check-no-chromedp.sh
    - Makefile
  modified:
    - .github/workflows/ci.yml

key-decisions:
  - "Built-in sanitize path uses sanitize.Sanitize(html), NOT the codebase_examples' bare Policy().Sanitize() — the latter would lowercase foreignObject/viewBox (bluemonday's tokenizer has no case hook) and silently break the inline-SVG <svg><foreignObject><section> container chain. sanitize.Sanitize wraps Policy().Sanitize + the required case restoration (the sanitize package's own doc mandates callers use it). A non-nil opts.Sanitize override still calls the caller's policy verbatim (they own any restoration). Documented as a Rule-1 correction to the illustrative sketch."
  - "Added a package-level `var parseWithEngine = markdown.ParseWithEngine` seam so the load-bearing one-parse invariant is runtime-verifiable via a counting wrapper (TestOneParseInvariant). Production behavior is byte-identical to calling the function directly; the indirection is the idiomatic Go way to count a package-function call without a mock framework — justified by the TRD must_have 'calling ParseWithEngine EXACTLY ONCE' + error_recovery's explicit 'counting wrapper' suggestion."
  - "capstone_test.go is the EXTERNAL test package `press_test` importing ONLY press/ (theme names hardcoded as []string{default,gaia,uncover} rather than importing press/themes) — the import list itself IS the Objective-7-begin gate proof that a downstream consumer needs nothing but press/."

requirements-completed: [API-01, API-02]

# Verification evidence
verification:
  gates_defined: 8
  gates_passed: 8
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 10min
completed: 2026-07-21
---

# Objective 3 TRD 09: press.Render Public API + no-chromedp Gate + Capstone Summary

**The public `press.Render(md, opts) -> {HTML, CSS, Model, Meta, Comments}` composes ALL six batteries (strikethrough/emoji/highlight/math/autofit + sanitize + embedded themes) into ONE battery-laden goldmark engine, parses EXACTLY once, forks that single doc to HTML + Model sinks, sanitizes last, packs the opts->front-matter->"default" theme, and aggregates speaker Comments — a SIBLING to chase.Render that never touches chase.go — with a CI-enforced zero-chromedp boundary (API-02) and a capstone proving all 3 themes + every battery through a press-only consumer.**

## Performance

- **Duration:** ~10 min (start 2026-07-21T14:33:45Z -> SUMMARY 2026-07-21T14:43:54Z)
- **Started:** 2026-07-21T14:33:45Z
- **Completed:** 2026-07-21T14:43:54Z
- **Tasks:** 3/3 complete
- **Files:** 6 created, 1 modified

## Accomplishments

- **API-01 — `press.Render`:** one-parse-two-sinks composition in `press/press.go`. `pressExtraOpts(opts)` bundles the six battery options (strikethrough always-on, emoji always-on, highlight unless `NoHighlight`, `pmath.Option(opts.MathMode)`, autofit always-on) into `markdown.NewEngine(...)`; `parseWithEngine(md, engine)` is THE ONE parse; sink 1 renders HTML via `engine.Renderer().Render(...)` + `remapHLJS` (chroma short-class -> `.hljs-*`); sink 2 is `model.Build(doc, source, pc)` on the SAME doc; sanitize runs LAST over the composed HTML string; CSS is packed from the embedded `press/themes.ThemeSet` with the theme resolved `opts.Theme -> front-matter theme: -> "default"`.
- **Comments aggregation:** `comments.go`'s `flattenNotes` flattens `Model.Sections[*].Notes` in document order — a pure aggregation of data `model.Build` already populated, never a second `ast.Walk`.
- **API-02 — no-chromedp gate:** `scripts/check-no-chromedp.sh` runs `go list -deps ./press/... ./chase/... ./profiles/...` and fails if any line matches `chromedp`; wired as `make check-no-chromedp` and a CI step beside the existing `addlicense` check (which was preserved, not clobbered). Current chromedp count in press/: **0**.
- **Capstone:** `capstone_test.go` (external `press_test`, press-only import) renders a deck exercising every battery under all 3 bundled themes, asserting distinct non-empty CSS per theme, every battery's markup present, the XSS `<script>` neutralized while battery markup (emoji `<img>`, `<s>`, MathML, `.hljs-*`, fit marker, `<foreignObject>`/`viewBox`) survives, and Comments/Meta correct — plus a `TestCapstonePressOnlyConsumer` proving a press-only consumer renders a complete deck.
- **One-parse invariant proven at runtime:** `TestOneParseInvariant` wraps the `parseWithEngine` seam with a call counter and asserts Render invokes it exactly once.
- **chase.go untouched:** `git diff main -- chase/chase.go` is empty; press.Render is a sibling composition, not a wrapper.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: press.Render composition + Comments + theme resolution | `go test ./press/ -run 'TestRender\|TestOneParse\|TestThemeResolution\|TestOptions\|TestSanitizeLast\|TestComments' -v && gofmt -l press/press.go press/comments.go && go vet ./press/...` | 0 | PASS |
| 2: no-chromedp CI gate (script + Makefile + CI step) | `bash scripts/check-no-chromedp.sh && grep -q 'check-no-chromedp' .github/workflows/ci.yml Makefile` | 0 | PASS |
| 3: capstone — all 3 themes + every battery vs standing gates | `go test ./press/... ./conformance/... ./chase/... && grep-gate == 0` | 0 | PASS |

## Task Commits

Each task committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: press.Render one-parse-two-sinks composition + Comments + theme resolution** — `fb4fe44` (feat)
2. **Task 2: no-chromedp CI gate (API-02) — script + Makefile target + CI step** — `3c35e8a` (chore)
3. **Task 3: capstone integration test — all 3 themes + every battery, press-only consumer** — `9a49d47` (test)

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l press/ scripts/` | 0 (no output) | PASS |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` | 0 | PASS |
| Obj-1 corpus/cssdiff | `go test ./conformance/... ./chase/theme/...` | 0 | PASS |
| Obj-2 grep-gate | `grep -rnE '\bSlide\b\|"section"\|16:9\|1280\|720' chase/model chase/theme --include='*.go' \| grep -v _test.go \| grep -c .` | 0 matches | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -check .` | 0 | PASS |
| no-chromedp (API-02) | `bash scripts/check-no-chromedp.sh` | 0 (0 chromedp) | PASS |

## TDD Evidence

Task 1 is `tdd="true"`. The composition was driven test-first; the one-parse invariant and sanitize-last were proven by dedicated tests.

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./press/ -run 'TestRender...'` (before implementation) | non-zero (undefined Render/Output composition) | FAIL (correct) |
| GREEN (Task 1) | `go test ./press/ -run 'TestRender\|TestOneParse\|TestThemeResolution\|TestOptions\|TestSanitizeLast\|TestComments' -v` | 0 (9 tests) | PASS (correct) |
| One-parse proof | `go test ./press/ -run TestOneParseInvariant -v` | 0 (counter == 1) | PASS (correct) |
| Press-only consumer | `go test ./press/ -run TestCapstonePressOnlyConsumer -v` | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (two minor test-assertion corrections during the RED->GREEN loop, before each commit — see Deviations; neither was a production-code auto-fix cycle)
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 03-09-TRD.md frontmatter)
- **Gate failures:** None remaining
- **API-01:** press.Render returns fully-populated {HTML, CSS, Model, Meta, Comments}; ParseWithEngine called exactly once; sanitize last; chase.go untouched — VERIFIED
- **API-02:** `go list -deps ./press/...` has 0 chromedp; `make check-no-chromedp` wired into CI beside addlicense — VERIFIED

## Files Created/Modified

- `press/press.go` — `Render` (one-parse-two-sinks composition), `pressExtraOpts`, `resolveProfile`, `packThemeCSS`, `resolveThemeName`, `svgEnabled`, `parseWithEngine` seam var, `defaultThemeName` const
- `press/comments.go` — `flattenNotes` (Model.Sections[*].Notes -> Output.Comments, pure aggregation)
- `press/press_test.go` — Test-list cases 1-6 (compose-every-battery, zero-value Options, one-parse invariant, theme resolution, options honored, sanitize-last, Comments aggregation)
- `press/capstone_test.go` — external `press_test`, press-only import: all-3-themes/every-battery, press-only consumer, default-fallback
- `scripts/check-no-chromedp.sh` — API-02 gate script
- `Makefile` — `check-no-chromedp` (+ convenience build/vet/test) targets
- `.github/workflows/ci.yml` — added the `Check no chromedp (API-02)` step beside addlicense (existing steps preserved)

## Decisions Made

- **Built-in sanitize uses `sanitize.Sanitize(html)`**, not the sketch's bare `Policy().Sanitize()` — the sanitize package's own doc requires the wrapper because bluemonday's tokenizer lowercases `foreignObject`/`viewBox`; the wrapper restores their case and preserves the inline-SVG container chain. A non-nil `opts.Sanitize` override calls the caller's policy verbatim.
- **`parseWithEngine` package-level seam var** for a runtime one-parse proof (counting wrapper); production behavior identical.
- **CSS inline-SVG mode from `svgEnabled(pc) || opts.InlineSVG`**, mirroring chase.go — ParseWithEngine always enables inline-SVG, so the packed CSS container chain must match the always-SVG-wrapped HTML.
- **Capstone as external `press_test`** with hardcoded theme names to keep the import list press-only (the Objective-7-begin gate proof).

## Deviations from Plan

### Illustrative-sketch corrections (no scope change)

**1. [Rule 1 - Bug] Built-in sanitize path corrected from bare `Policy().Sanitize()` to `sanitize.Sanitize()`**
- **Found during:** Task 1 implementation (reading press/sanitize/policy.go's own doc)
- **Issue:** The TRD `codebase_examples` sketch sanitized via `policy := opts.Sanitize; if nil { policy = sanitize.Policy() }; html = policy.Sanitize(html)`. Bare `Policy().Sanitize()` lowercases `foreignObject`/`viewBox` (bluemonday has no tag-casing hook), which silently breaks the inline-SVG `<svg><foreignObject><section>` container chain — the sanitize package explicitly documents that callers MUST use `sanitize.Sanitize` (Policy().Sanitize + case restoration).
- **Fix:** nil `opts.Sanitize` -> `sanitize.Sanitize(html)`; non-nil -> `opts.Sanitize.Sanitize(html)` (caller owns restoration). `TestSanitizeLast` asserts `<foreignObject`/`viewBox=` survive with correct case.
- **Files:** press/press.go, press/press_test.go
- **Committed in:** fb4fe44 (Task 1)

**2. [Rule 1 - Test] Capstone task-list checkbox assertion dropped**
- **Found during:** Task 3 (initial capstone run)
- **Issue:** An initial capstone fingerprint asserted `type="checkbox"` survives. The 03-08 Marp-parity sanitize policy deliberately does not allowlist `<input>`, so the GFM task-list checkbox is stripped (a documented sanitize-policy limitation, like the `style` attr limitation in 03-08-SUMMARY) — not a battery failure.
- **Fix:** removed the checkbox fingerprint (the GFM table fingerprint still proves GFM); the task list remains in the deck to exercise the parser.
- **Files:** press/capstone_test.go
- **Committed in:** 9a49d47 (Task 3)

---

**Total deviations:** 2 (both corrections to illustrative sketch / initial test assertions, before their respective commits). No production-code auto-fix cycles; no scope creep.

## Issues Encountered

None beyond the two corrections above. All eight gates green on the final run.

## User Setup Required

None — no external service configuration required. The no-chromedp gate runs in CI automatically on push/PR.

## Next Objective Readiness

- **Objective 7 (Dart binding) may begin:** a consumer imports ONLY `press/` and calls `press.Render(md, opts)` to get a complete, JSON-serializable `{HTML, CSS, Model, Meta, Comments}` — proven by `TestCapstonePressOnlyConsumer` and the CI-enforced zero-chromedp boundary.
- **Objectives 4 (CLI) / 5 (exporters)** consume the same stable `press.Render` surface; `Output.Meta`/`Output.Comments` are top-level for callers that need only metadata/notes.
- **Objective 8** owns math raster quality + the final auto-fit/fallback rule; the heavy `\begin{aligned}` construct currently routes to the graceful `math-fallback` `<img>` stub (per 03-06), exercised (not over-asserted) by the capstone.

## Self-Check: PASSED

All claimed files confirmed present on disk; all three task commit hashes confirmed in `git log`.

- FOUND: press/press.go
- FOUND: press/comments.go
- FOUND: press/press_test.go
- FOUND: press/capstone_test.go
- FOUND: scripts/check-no-chromedp.sh
- FOUND: Makefile
- FOUND: .github/workflows/ci.yml (modified — no-chromedp step added)
- FOUND: .planning/objectives/03-press-batteries-api/03-09-SUMMARY.md
- FOUND commit: fb4fe44 (Task 1)
- FOUND commit: 3c35e8a (Task 2)
- FOUND commit: 9a49d47 (Task 3)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
