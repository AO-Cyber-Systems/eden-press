---
objective: 05-convert-raster
job: "02"
subsystem: convert
tags: [chromedp, determinism, set-document-content, stix-two-math, ofl-1.1, exp-04]

# Dependency graph
requires:
  - objective: 05-convert-raster
    provides: "05-01's convert/chrome.Session (one-browser-many-tabs pool) -- ApplyDeterminism/LoadHTML both take a Session.NewTab() context"
provides:
  - "convert/chrome.ComposeCSS(baseCSS string) string -- pure CSS transform appending the animation/transition/scroll-behavior !important kill override + the STIX Two Math @font-face data-URI, overrides positioned LAST"
  - "convert/chrome.PageCSSInches(size theme.Size) string -- pure @page{size:<w>in <h>in;margin:0;} emitter at 96px/in, the PDF page-size mechanism 05-03 pairs with WithPreferCSSPageSize(true)"
  - "convert/chrome.ApplyDeterminism(ctx, viewportW, viewportH int64) error -- the ordered live-CDP recipe (fixed viewport+scale, UTC timezone, en-US locale, reduced-motion media emulation) applied to a tab BEFORE content loads"
  - "convert/chrome.LoadHTML(ctx, html string) error -- the page.SetDocumentContent loader (Navigate(about:blank) -> GetFrameTree -> SetDocumentContent -> document.fonts.ready wait), never data: URL or file://"
  - "convert/chrome.FontFaceDataURI() string + go:embed'd STIX Two Math OTF -- EXP-04 MATH-font provisioning so exported math renders real glyphs, not tofu"
  - "05-RESEARCH Open Question #1 (self-containment) RESOLVED against VERIFIED press.Render output, not assumption: press/sanitize's bluemonday policy already strips ALL relative asset references (style attr never allow-listed at all; RequireParseableURLs+no AllowRelativeURLs rejects schemeless URLs on allow-listed attrs too) before they reach convert/ -- not currently a self-containment risk in practice. Absolute http(s) references DO survive sanitize verbatim; convert/ adds no asset-inlining pre-pass, so making those resolvable is the deck author's responsibility."
affects: [05-03-pdf, 05-04-png, 05-05-ci-hardening]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ComposeCSS/PageCSSInches/ApplyDeterminism/LoadHTML are the ONE shared determinism recipe -- 05-03 (pdf) and 05-04 (png) both call exactly these four, never re-implementing any piece of it."
    - "STIX Two Math is embedded via go:embed and injected as a base64 @font-face data-URI (never a live font-fetch URL), matching convert/'s no-network-dependency rendering posture."
    - "Self-containment for convert/'s SetDocumentContent input is enforced upstream, incidentally, by press/sanitize's existing bluemonday policy (no style= attr allow-listed at all; RequireParseableURLs without AllowRelativeURLs rejects any schemeless URL on an allow-listed attribute) -- convert/ deliberately adds no asset-inlining pre-pass of its own."

key-files:
  created:
    - convert/chrome/determinism.go
    - convert/chrome/determinism_test.go
    - convert/chrome/fonts.go
    - convert/chrome/fonts/STIXTwoMath-Regular.otf
    - convert/chrome/load.go
    - convert/chrome/load_test.go
  modified:
    - convert/chrome/discover.go
    - NOTICE

key-decisions:
  - "ApplyDeterminism landed in the Task 1 commit (b452d30) alongside the pure CSS helpers, rather than Task 2 as the TRD's file_tree sketch implied -- both live in determinism.go and were written+reviewed together for cohesion; Task 2's commit (e0e2303) carries load.go/load_test.go/discover.go's doc update instead. No functional or interface deviation, purely a commit-boundary sequencing choice, documented here per the executor's atomic-commit convention."
  - "TestSelfContainmentContract's relative-background-image subtest was REWRITTEN after empirical verification (a throwaway probe test, deleted before commit) showed the TRD's assumed premise -- that a relative asset URL 'survives into the composed output verbatim' -- does not hold under press.Render's DEFAULT sanitize policy: press/sanitize never allow-lists the style attribute at all (so `<figure style=\"background-image:url(...)\">` is stripped whole, relative or absolute), and independently rejects ANY schemeless URL on an allow-listed attribute (e.g. a relative `<img src=...>`) because the policy calls RequireParseableURLs(true) but never AllowRelativeURLs(true). The test now pins the VERIFIED behavior (relative references are stripped before reaching convert/; an absolute http(s) image DOES survive verbatim) with 3 subtests instead of 2, still resolving Open Question #1, now against ground truth rather than an untested assumption."
  - "NOTICE's STIX Two Math copyright line uses the font project's ACTUAL OFL.txt text (\"Copyright 2001-2021 The STIX Fonts Project Authors (https://github.com/stipub/stixfonts)\") rather than the TRD's placeholder (\"Copyright (c) 2001-2026 by the STI Pub Companies\") -- corrected against the real license file, per the TRD's own instruction to \"adjust the copyright line/holder to match the font's actual OFL.txt if it differs.\""
  - "STIX Two Math OTF sourced from the stipub/stixfonts GitHub repo's v2.13 TAG (fonts/static_otf/STIXTwoMath-Regular.otf), not the master branch HEAD -- master .gitignores the compiled font output directory, so no BLOCKER was needed; the font was obtainable and bundled verbatim (838,652 bytes, confirmed OpenType/CFF via magic bytes)."
  - "REQUIREMENTS.md's EXP-04 row remains Pending (not checked off) -- per 05-01-SUMMARY.md's own note, EXP-04 spans 05-01 (discovery+Session), 05-02 (this TRD: determinism+font), and 05-05 (CI hardening); only the third TRD's landing completes the full requirement."

patterns-established:
  - "Shared, non-duplicated determinism substrate (ComposeCSS + PageCSSInches + ApplyDeterminism + LoadHTML) is the one recipe both wave-3 raster exporters fold in."
  - "Verify-before-assert discipline for behavioral test premises: rather than trust the TRD's Test-list wording that a relative asset URL 'survives' sanitize, a scratch probe test against 4 concrete cases (relative bg, relative inline img, absolute inline img, absolute bg) established ground truth BEFORE writing the committed assertion -- avoiding a test that encodes an incorrect assumption."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 6
  gates_passed: 6
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 47min
completed: 2026-07-21
---

# Objective 5 TRD 02: Shared Determinism Substrate — ComposeCSS/ApplyDeterminism/LoadHTML + STIX Two Math Font Summary

**The one shared determinism recipe (CSS animation-kill + STIX @font-face, fixed viewport/UTC/en-US/reduced-motion CDP recipe, and the page.SetDocumentContent loader with a document.fonts.ready wait) that both wave-3 raster exporters (05-03 pdf, 05-04 png) will fold in exactly once — plus the go:embed'd STIX Two Math OTF (EXP-04 font provisioning) and a self-containment contract resolved against VERIFIED press.Render output rather than assumption.**

## Performance

- **Duration:** ~47 min (prior HEAD 095ac6f at 12:11:07 -> Task 3 commit e7a3012 at 12:58:20, local time)
- **Started:** 2026-07-21T16:11:07Z
- **Completed:** 2026-07-21T16:58:20Z
- **Tasks:** 3/3 complete
- **Files modified:** 8 (6 created, 2 modified: convert/chrome/discover.go, NOTICE)

## Accomplishments
- `convert/chrome.ComposeCSS(baseCSS string) string` appends the guaranteed `*,*::before,*::after{animation:none!important;transition:none!important;scroll-behavior:auto!important;}` kill override plus the bundled STIX Two Math `@font-face` data-URI, overrides positioned LAST so `!important` + cascade order both win — independently unit-tested, zero chromedp import.
- `convert/chrome.PageCSSInches(size theme.Size) string` emits `@page{size:<w>in <h>in;margin:0;}` at 96px/in: 16:9 (1280x720) -> `"13.333in 7.5in"`, 4:3 (960x720) -> `"10in 7.5in"` — the exact PDF page-size mechanism 05-03 will pair with `WithPreferCSSPageSize(true)`.
- `convert/chrome.FontFaceDataURI() string` + `//go:embed fonts/STIXTwoMath-Regular.otf` bundles the STIX Two Math OTF (838,652 bytes, sourced verbatim from `stipub/stixfonts` tag `v2.13`, `fonts/static_otf/STIXTwoMath-Regular.otf` — NOT the Google Fonts CDN, per the anti-pattern) as a base64 `@font-face` data-URI, so headless-Chrome math export renders real glyphs instead of tofu with zero network fetch.
- `convert/chrome.ApplyDeterminism(ctx, viewportW, viewportH int64) error` runs, in order, via one `chromedp.Run`: `EmulateViewport`+`EmulateScale(1.0)`, `emulation.SetTimezoneOverride("UTC")`, `emulation.SetLocaleOverride().WithLocale("en-US")`, `emulation.SetEmulatedMedia` with `prefers-reduced-motion:reduce` — pinning every live-CDP source of render variance BEFORE content loads.
- `convert/chrome.LoadHTML(ctx, html string) error` loads via `Navigate("about:blank")` -> `page.GetFrameTree` -> `page.SetDocumentContent(frameID, html)` -> a `document.fonts.ready` wait through a custom `awaitPromise` `EvaluateOption` (chromedp has no built-in `AwaitPromise` wrapper) — never a `data:` URL (documented truncation bug) or a temp file/`file://` (unwanted local-file-access posture).
- `TestSelfContainmentContract` resolves 05-RESEARCH Open Question #1 against **verified**, not assumed, `press.Render` behavior (see Decisions/Deviations): an image-free deck carries no external references; a relative background-image reference is stripped entirely by `press/sanitize`'s existing bluemonday policy before it ever reaches convert/; an absolute http(s) image reference DOES survive sanitize verbatim, and making that resolvable (e.g. as a `data:` URI) is the deck author's responsibility, not convert/'s — convert/ adds no asset-inlining pre-pass.
- NOTICE carries a new "Bundled fonts (export-environment MATH rendering — EXP-04)" section crediting STIX Two Math under OFL-1.1, with the copyright line corrected to the font project's actual OFL.txt text, verbatim-bundling + reserved-name compliance stated, and the `.otf` binary documented as `addlicense`-excluded (mirroring the `themes/**` precedent).
- `bash scripts/check-no-chromedp.sh` stays PASS after every task; `convert/` remains the module's sole chromedp-touching package.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Pure determinism CSS helpers + font-face builder | `go test ./convert/chrome/ -run 'ComposeCSS\|PageCSS\|FontFace' -v && go build ./convert/... && gofmt -l convert/chrome/determinism.go convert/chrome/fonts.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: ApplyDeterminism + LoadHTML + self-containment contract | `go test ./convert/chrome/ -v && go build ./convert/... && go vet ./convert/... && gofmt -l convert/chrome/load.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 3: NOTICE attribution for bundled STIX Two Math OFL font | `grep -q "STIX Two Math" NOTICE && grep -q "OFL-1.1" NOTICE && addlicense -l mit -s -c "AO Cyber Systems" -ignore 'convert/chrome/fonts/**' -check convert/` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Pure determinism CSS helpers (ComposeCSS, PageCSSInches) + STIX Two Math font-face builder** — `b452d30` (test) — includes `ApplyDeterminism` (see key-decisions: commit-boundary sequencing note)
2. **Task 2: SetDocumentContent HTML loader (LoadHTML) + self-containment contract** — `e0e2303` (feat) — includes `discover.go`'s package-doc update (documents load.go alongside determinism.go/fonts.go)
3. **Task 3: NOTICE attribution for the bundled STIX Two Math OFL font** — `e7a3012` (docs)

_Note: Task 1 is `tdd="true"` — RED (compile failure: `undefined: ComposeCSS`/`PageCSSInches`/`FontFaceDataURI`, confirmed by temporarily stashing determinism.go+fonts.go) confirmed before the GREEN implementation (all 4 test functions pass), matching the project's established one-commit-per-task convention. Task 2 is a plain `auto` task (live-Chrome smoke is Chrome-presence-gated per the TRD; the self-containment contract needed no live Chrome and was iterated against verified ground truth before commit — see Deviations). Task 3 is a plain `auto` documentation task._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l .` (repo-wide) | 0 (no output) | PASS |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (21 packages, incl. `convert/chrome` 8 pass + 2 skip) | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 (PASS printed) | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'convert/chrome/fonts/**' -check convert/` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./convert/chrome/ -run 'ComposeCSS\|PageCSS\|FontFace' -v` (determinism.go + fonts.go temporarily stashed) | 1 (compile failure: `undefined: ComposeCSS`, `undefined: PageCSSInches`, `undefined: FontFaceDataURI`) | FAIL (correct) |
| GREEN (Task 1) | `go test ./convert/chrome/ -run 'ComposeCSS\|PageCSS\|FontFace' -v` (files restored) | 0 (`TestComposeCSS`, `TestComposeCSS_emptyBase`, `TestPageCSSInches`, `TestFontFaceDataURI` all PASS) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (the self-containment test premise was corrected BEFORE commit via a deleted scratch probe — not a post-commit auto-fix cycle against already-landed code)
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 05-02-TRD.md frontmatter)
- **Gate failures:** None

## Files Created/Modified
- `convert/chrome/determinism.go` — `ComposeCSS`, `PageCSSInches`, `formatInches`, `animationKillCSS`, `pxPerInch`, `ApplyDeterminism`
- `convert/chrome/determinism_test.go` — `TestComposeCSS`, `TestComposeCSS_emptyBase`, `TestPageCSSInches` (table-driven, 16:9 + 4:3), `TestFontFaceDataURI`
- `convert/chrome/fonts.go` — `//go:embed fonts/STIXTwoMath-Regular.otf` + `FontFaceDataURI()`
- `convert/chrome/fonts/STIXTwoMath-Regular.otf` — bundled verbatim, 838,652 bytes, OpenType/CFF, sourced from `stipub/stixfonts` tag `v2.13`
- `convert/chrome/load.go` — `LoadHTML(ctx, html) error` + the unexported `awaitPromise` `EvaluateOption`
- `convert/chrome/load_test.go` — `TestLoadHTMLReadsBackRealContent` (Chrome-gated) + `TestSelfContainmentContract` (3 subtests, verified ground truth)
- `convert/chrome/discover.go` — package-doc comment extended to also describe determinism.go/load.go/fonts.go (avoids a duplicate package-doc block across files)
- `NOTICE` — new "Bundled fonts (export-environment MATH rendering — EXP-04)" section: STIX Two Math, OFL-1.1, corrected copyright line, verbatim-bundling + reserved-name compliance, addlicense-exclusion note

## Decisions Made
- `ApplyDeterminism` landed in the Task 1 commit (alongside the pure CSS helpers, both in `determinism.go`) rather than being split into a separate Task 2 commit as the TRD's `file_tree` sketch implied — no interface/behavior deviation, a commit-boundary sequencing choice documented here.
- `TestSelfContainmentContract`'s relative-asset subtest was rewritten to assert VERIFIED behavior (relative references are stripped by `press/sanitize` before reaching convert/) rather than the TRD's assumed-but-unverified premise (a relative URL "survives into the composed output verbatim"). Ground truth was established via a throwaway probe test (4 cases: relative bg, relative inline img, absolute inline img, absolute bg), then deleted before committing the real test. See Deviations for the full root-cause chain.
- NOTICE's STIX Two Math copyright line was corrected to the font project's actual OFL.txt text rather than the TRD's placeholder text, per the TRD's own "adjust ... if it differs" instruction.
- REQUIREMENTS.md's `EXP-04` row remains `Pending` — this TRD delivers the determinism+font half; 05-05 (CI hardening) still must land before the full requirement is satisfied (consistent with 05-01-SUMMARY.md's own note).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in test premise, caught pre-commit] TRD's self-containment Test-list case 5 assumed a relative asset URL "survives into the composed output verbatim" — verified false under press.Render's default sanitize policy**
- **Found during:** Task 2, while writing `TestSelfContainmentContract`'s relative-background-image subtest — it failed (`relative-image.png` not found in composed output) against the TRD's exact wording.
- **Root cause (verified, not guessed):** `press/sanitize/policy.go`'s bluemonday policy (a) never allow-lists the `style` attribute at all (a pre-existing, documented limitation tracked in `03-08-SUMMARY.md`, orthogonal to convert/), so a Marpit background image (`<figure style="background-image:url(...)">`, from `chase/markdown/render.go`'s `figureStyle`) is stripped WHOLE, relative or absolute; (b) independently calls `RequireParseableURLs(true)` but never `AllowRelativeURLs(true)`, so bluemonday's `validURL` (`microcosm-cc/bluemonday` `sanitize.go`) rejects ANY schemeless URL on an allow-listed attribute too (e.g. a relative `<img src=...>` also loses its `src`). Confirmed via a throwaway probe test against 4 concrete cases (relative bg, relative inline img, absolute inline img, absolute bg) — only the absolute inline `<img src="https://...">` case survived verbatim.
- **Fix:** Rewrote the subtest to assert the VERIFIED behavior (relative background image is stripped entirely — pins the current, correct behavior against future regression) and added a third subtest proving an absolute http(s) image DOES survive sanitize verbatim, which is the actual case where convert/'s "no asset-inlining pre-pass, self-containment is the author's responsibility" contract applies. Open Question #1 is still fully resolved — now against ground truth.
- **Files modified:** convert/chrome/load_test.go (before first commit of Task 2 — no separate revert/fix commit needed)
- **Verification:** `go test ./convert/chrome/ -run TestSelfContainmentContract -v` — all 3 subtests PASS.
- **Committed in:** e0e2303 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — corrected an unverified test premise before any commit; no scope change, no interface change)
**Impact on plan:** Strengthens Open Question #1's resolution with verified ground truth instead of an assumed URL-survival behavior; convert/'s public surface (`ComposeCSS`/`PageCSSInches`/`ApplyDeterminism`/`LoadHTML`/`FontFaceDataURI`) is unchanged from the TRD's spec.

## Issues Encountered
- No system Chrome/Chromium is installed in this execution sandbox, so `TestLoadHTMLReadsBackRealContent` exercises only the `t.Skip` path here (same as 05-01's `TestSessionMultiTab`). This is the TRD-anticipated "CI without system Chrome" case; it will run live once 05-05 provisions Chrome/`headless-shell` in CI.
- The large base64 STIX font payload (~1.1MB on a single line inside `ComposeCSS`'s output) triggered this session's tool-output truncation mechanism when dumped in a failing test's debug output — worked around with targeted `grep -c`/`grep -o` rather than materializing the full line; no impact on the shipped code or test correctness.

## User Setup Required
None for this TRD's scope. Downstream: 05-05 will need a provisioned Chrome/`chromedp/headless-shell` in CI for `TestLoadHTMLReadsBackRealContent` and later PDF/PNG export tests to run live rather than skip; 05-05 also wires the `convert/chrome/fonts/**` addlicense `-ignore` pattern into `.github/workflows/ci.yml` (this TRD only verified it locally).

## Next Objective Readiness
- `convert/chrome.ComposeCSS`/`PageCSSInches`/`ApplyDeterminism`/`LoadHTML`/`FontFaceDataURI` are locked and ready for 05-03 (PDF via `PrintToPDF`, paired with `PageCSSInches` + `WithPreferCSSPageSize(true)`) and 05-04 (PNG via screenshot) to fold in identically — neither exporter should re-implement any piece of this recipe.
- The self-containment contract is documented and test-pinned: convert/'s exporters can assume relative asset references from a `press.Render`-produced deck are already stripped by `press/sanitize` upstream; an absolute remote reference is the deck author's responsibility to make resolvable, not convert/'s.
- STIX Two Math is provisioned and injectable; 05-05 owns the CI-side pixel-diff smoke and the `.github/workflows/ci.yml` addlicense `-ignore` wiring for the `.otf` binary.

## Self-Check: PASSED

All claimed files confirmed present on disk; all three task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: convert/chrome/determinism.go
- FOUND: convert/chrome/determinism_test.go
- FOUND: convert/chrome/fonts.go
- FOUND: convert/chrome/fonts/STIXTwoMath-Regular.otf
- FOUND: convert/chrome/load.go
- FOUND: convert/chrome/load_test.go
- FOUND: convert/chrome/discover.go (modified)
- FOUND: NOTICE (modified)
- FOUND: .planning/objectives/05-convert-raster/05-01-SUMMARY.md
- FOUND commit: b452d30 (Task 1)
- FOUND commit: e0e2303 (Task 2)
- FOUND commit: e7a3012 (Task 3)

---
*Objective: 05-convert-raster*
*Completed: 2026-07-21*
