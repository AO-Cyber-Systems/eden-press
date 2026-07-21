---
objective: 01-chase-framework
job: "08"
subsystem: markdown-rendering
tags: [goldmark, marpit, conformance, htmldiff, cssdiff, two-phase-parse]

# Dependency graph
requires:
  - objective: 01-chase-framework (01-04)
    provides: chase/theme Pack pipeline (scaffold prepend, selector scoping, cssdiff-verified fixtures)
  - objective: 01-chase-framework (01-06)
    provides: chase/markdown directive carry-forward + apply (data-attribute/style materialization)
  - objective: 01-chase-framework (01-07)
    provides: chase/markdown background images + inline-SVG advanced-background wrap
provides:
  - "chase/markdown.NewEngine/Parse/Render/RenderFunc: the canonical two-phase Parser().Parse() + Renderer().Render() seam (PARSE-01), never Convert()"
  - "conformance/runner/chase_corpus_test.go: NEW corpus runner driving the chase engine, PASS/BLOCKED/FAIL categorization against the 18-case Marp golden corpus"
  - "chase/theme/pack_conformance_test.go: this integration TRD's own cssdiff.Equal acceptance-gate attestation for the stress + scaffold themes"
  - "chase/markdown/testdata/inline_svg_sample.md + a human-verified (auto-approved, autonomous run) rasterization checkpoint"
affects: [02-docmodel, 03-marp-core-batteries, 05-rasterization]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-phase Parse()/Render() as the ONLY sanctioned chase-engine entrypoint (Convert() banned) — Objective 2's docmodel hook reuses Parse's returned (*ast.Document, parser.Context) pair"
    - "Corpus skip-map: battery-blocked cases recorded via report.AddPending keyed to the exact Objective-3 CORE-* requirement that closes them, never silently hidden as failures"

key-files:
  created:
    - chase/markdown/seam.go
    - chase/markdown/seam_test.go
    - conformance/runner/chase_corpus_test.go
    - chase/theme/pack_conformance_test.go
    - chase/markdown/testdata/inline_svg_sample.md
  modified:
    - chase/markdown/apply.go

key-decisions:
  - "marp-gfm-table PASSes outright through the chase engine (goldmark's stock GFM table extension, wired via extension.GFM) -- no Objective-3 battery needed; asserted explicitly in chase_corpus_test.go as a decision record, not left to fall out of a generic zero-failures check"
  - "headingDivider materialization was a genuine Objective-1 bug (Rule 1), not a skip-map candidate: apply.go's generic data-attribute loop stringified CoerceGlobal's expanded []int{1,2} range as \"1,2\", but Marp/Marpit materializes the author-facing scalar \"2\" -- fixed via a narrow headingDividerDisplayValue helper that leaves the expanded-range contract (headingdivider.go's synthetic-break transformer) untouched"
  - "Rasterization proof for this TRD is a human-verify checkpoint (browser screenshot), auto-approved per the autonomous-run instruction; the deterministic/CI headless-Chrome pixel-diff is explicitly deferred to Objective 5, where chromedp lives and the Chrome-free boundary (API-02) no longer applies"

patterns-established:
  - "runner.RenderFunc adapter: an engine package (chase/markdown) exposes an unnamed func literal with RenderFunc's exact shape via its own RenderFunc() constructor, so the corpus harness (conformance/runner) can drive it WITHOUT chase/markdown importing conformance/runner -- avoids any import-direction risk while keeping the harness engine-agnostic"

requirements-completed: [PARSE-01]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~45min
completed: 2026-07-21
---

# Objective 01 TRD 08: Integration — Corpus + CSSDiff Gates + Raster Check (PARSE-01) Summary

**Formalized goldmark's two-phase `Parser().Parse()`/`Renderer().Render()` seam as `chase/markdown/seam.go`, drove the 18-case Marp golden corpus through the chase engine (10 PASS including all 9 Marpit-mechanic cases + `marp-gfm-table`, 8 honestly BLOCKED on Objective-3 batteries, 0 unexplained failures), passed the theme `cssdiff.Equal` CSS-AST gate for the stress + scaffold themes, and visually confirmed inline-SVG rasterization in a real browser.**

## Performance

- **Duration:** ~45 min (committed-task span 16fcc47 → dd68671: 00:38:12 → 00:49:57 local, plus pre-commit research/design)
- **Started:** 2026-07-21T04:38:12Z
- **Completed:** 2026-07-21T04:50:34Z
- **Tasks:** 3/3 complete
- **Files modified:** 6 (5 created, 1 modified)

## Accomplishments

- `chase/markdown/seam.go`: canonical `NewEngine`/`Parse`/`Render`/`RenderFunc` — the AST + `parser.Context` are proven inspectable BETWEEN `Parser().Parse()` and `Renderer().Render()` (criterion 1), `Convert()` is never called.
- `conformance/runner/chase_corpus_test.go` (NEW, `corpus_test.go` untouched): drives all 18 Marp golden corpus cases through the chase engine. 9 Marpit-mechanic cases PASS via `htmldiff.Equal`; `marp-gfm-table` PASSes too (decision recorded below); 8 marp-core "battery" cases are BLOCKED via an explicit skip-map keyed to their Objective-3 requirement; zero un-mapped failures.
- `chase/theme/pack_conformance_test.go` (NEW): formal `cssdiff.Equal` acceptance-gate attestation for `Pack("stress", InlineSVG:true)` and `Pack(scaffold, InlineSVG:false)`, reusing 01-04's `expectedStressPackedCSS`/`expectedScaffoldPackedCSS` fixtures verbatim.
- `chase/markdown/testdata/inline_svg_sample.md` + a rendered, real-scaffold-CSS-packed, standalone HTML artifact: visually confirmed in a browser that inline-SVG slides rasterize (headings/paragraphs paint, the bg-split advanced-background layer paints beside content, and pagination numbers 1/2/3 show bottom-right).
- `[Rule 1 - Bug]` fix: `chase/markdown/apply.go` now materializes `headingDivider` as the author-facing scalar (`data-heading-divider="2"`), matching real Marp/Marpit output, instead of the internally-expanded range.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Two-phase seam (PARSE-01) + criterion-1 AST hook test | `go test ./chase/markdown/ -run 'Seam\|TwoPhase\|Render' -v && go vet ./chase/markdown/` | 0 | PASS |
| 2: Chase corpus runner (PASS/BLOCKED/FAIL) + theme cssdiff gate | `go test ./conformance/runner/ ./chase/theme/ -v` | 0 | PASS |
| 3: Rasterization checkpoint — inline-SVG renders in a browser | Rendered artifact served on :8091, visually confirmed via browser screenshot (auto-approved, autonomous run) | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (raw `git commit` is hook-blocked and was never used):

1. **Task 1 (RED): failing criterion-1 seam test** - `16fcc47` (test)
2. **Task 1 (GREEN): two-phase seam (PARSE-01) + RenderFunc adapter** - `902e93d` (feat)
3. **Task 2: chase corpus runner + theme cssdiff gate (bundles the Rule-1 apply.go fix)** - `e9adebf` (test)
4. **Task 3: rasterization checkpoint sample deck (auto-approved)** - `dd68671` (docs)

**Plan metadata:** *pending — this SUMMARY.md + STATE files, committed next.*

_Note: Task 1 used the full TDD RED→GREEN cycle; Task 2 was written test-first (corpus/cssdiff assertions) and passed on first run once the Rule-1 fix was folded in._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/ conformance/runner/chase_corpus_test.go \| (! grep .) ; go vet ./chase/... ./conformance/...` | 0 | PASS |
| test | `go test ./chase/... ./conformance/...` | 0 | PASS |
| build | `go build ./...` | 0 | PASS |

Whole-repo (beyond the TRD's own gates, per the coordinator's finalize request):

| Check | Command | Result |
|---|---|---|
| Whole-repo build | `go build ./...` | Clean |
| Whole-repo test | `go test ./...` | All packages `ok` |
| Whole-repo vet | `go vet ./...` | Clean |
| Whole-repo gofmt | `gofmt -l $(find . -name '*.go')` | Clean (no output) |
| License headers | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -check .` | exit 0 |
| Chrome-free boundary | `go list -deps ./chase/... \| grep -c chromedp` | 0 |
| Existing PENDING gate untouched | `git diff HEAD -- conformance/runner/corpus_test.go` | No diff |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1, seam criterion-1) | `go test ./chase/markdown/ -run TestSeamCriterion1ASTAndContextInspectableBetweenPhases` (pre-seam.go) | 1 (build failure: seam.go incomplete) | FAIL (correct) |
| GREEN (Task 1) | `go test ./chase/markdown/ -run 'Seam\|TwoPhase\|Render' -v` | 0 | PASS (correct) |
| GREEN (Task 2, corpus + cssdiff gates, written test-first) | `go test ./conformance/runner/ -run TestChaseCorpus -v` && `go test ./chase/theme/ -run TestObjective1ThemeCSSDiffGate -v` | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (the Rule-1 headingDivider fix was applied proactively during Task 2 authoring, not as a post-verify recovery cycle; it verified GREEN on first `chase_corpus_test.go` run)
- **Must-haves verified:** 5/5 (all 5 `must_haves.truths` from the TRD frontmatter — two-phase seam inspectable, Marpit-mechanic corpus PASS, skip-map categorization with zero unexplained failures, theme cssdiff gate, human-verified inline-SVG rasterization)
- **Gate failures:** None

## Final Corpus Table (18 cases)

| Case | Category | Requirement | Notes |
|---|---|---|---|
| marp-basic | PASS | PARSE-01 | |
| marp-slide-split | PASS | PARSE-01 | |
| marp-class-spot | PASS | PARSE-01 | |
| marp-heading-divider | PASS | PARSE-01 | Required the `[Rule 1 - Bug]` apply.go fix (scalar display value) |
| marp-paginate | PASS | PARSE-01 | |
| marp-header-footer | PASS | PARSE-01 | |
| marp-bg-color | PASS | PARSE-01 | |
| marp-bg-image | PASS | PARSE-01 | |
| marp-bg-split | PASS | PARSE-01 | |
| marp-gfm-table | PASS | PARSE-01 | Decision: passes outright via goldmark's stock GFM extension; no battery needed (see key-decisions) |
| marp-emoji | BLOCKED | CORE-06 | Objective-3 battery |
| marp-code-highlight | BLOCKED | CORE-07 | Objective-3 battery |
| marp-math | BLOCKED | CORE-08 | Objective-3 battery |
| marp-strikethrough | BLOCKED | CORE-03 | Objective-3 battery |
| marp-fit-heading | BLOCKED | CORE-09 | Objective-3 battery |
| marp-theme-gaia | BLOCKED | CORE-01 | Embedded theme CSS — Objective-3 battery |
| marp-theme-uncover | BLOCKED | CORE-01 | Embedded theme CSS — Objective-3 battery |
| marp-size-4-3 | BLOCKED | CORE-02 | Objective-3 battery |

**Totals: 10 PASS, 8 BLOCKED, 0 FAIL.** All 9 required Marpit-mechanic cases pass; `marp-gfm-table` passes as a bonus tenth. Every blocked case carries an explicit CORE-* requirement id via `conformance/report.SectionReport.AddPending` — none is hidden as a failure, and the existing `conformance/runner/corpus_test.go::TestMarpCorpus` PENDING gate is untouched (Objective 3 still flips it).

## Rasterization Checkpoint (Task 3, autonomous run — auto-approved)

- **Sample deck:** `chase/markdown/testdata/inline_svg_sample.md` (title slide + `![bg left](...)` split slide + paginated slide, `paginate: true` front matter).
- **Render artifact:** generated via a one-shot render helper (chase engine, inline-SVG on, packed with the real `theme.Pack(scaffold, InlineSVG:true)` CSS) to `/private/tmp/claude-501/-Users-justin-dev/ebb440f8-b11a-4131-ac02-7f07af0b933d/scratchpad/inline_svg_sample.html`. The helper itself (`zzz_gen_checkpoint_artifact_test.go`) was a temporary, uncommitted tool — deleted after use; it is not part of this TRD's declared file list.
- **Visual verification:** `file://` is sandboxed in this environment's browser tool, so the artifact was served briefly on **port 8091** (port 8080 never used, per the standing hard rule) and screenshotted:
  - Slide 1 (title): heading + paragraph paint inside `<svg><foreignObject><section>`; pagination "1" shows bottom-right.
  - Slide 2 (bg-split): left advanced-background `<figure>` layer (background-image) and right content `foreignObject` both paint side by side; pagination "2" shows.
  - Slide 3 (paginated): heading + paragraph paint; pagination "3" shows.
  - Screenshot evidence: `/Users/justin/dev/01-08-inline-svg-checkpoint.png`.
- **Auto-approval:** per the autonomous-run instruction, this `checkpoint:human-verify` was treated as auto-approved once the automation confirmed rasterization — execution continued without blocking for a live human response. The automated corpus (`htmldiff.Equal`) and theme (`cssdiff.Equal`) gates remain the real Objective-1 verification; this checkpoint satisfies ROADMAP criterion 3's "rasterized (not string-diff alone)" half.
- **Objective-5 deferral (documented):** the deterministic/CI headless-Chrome pixel-diff proof is deferred to Objective 5, where `chromedp` lives and the Chrome-free boundary (API-02) enforced on `chase/`/`press/` no longer applies. This TRD's `chase/` package tree confirmed to import zero `chromedp` transitive dependencies.

## Files Created/Modified

- `chase/markdown/seam.go` - Canonical two-phase `Parse`/`Render`/`RenderFunc`/`NewEngine` — the Objective-2 seam
- `chase/markdown/seam_test.go` - Criterion-1 test proving AST + `parser.Context` inspectable between phases
- `conformance/runner/chase_corpus_test.go` - NEW corpus runner: PASS/BLOCKED/FAIL categorization for the chase engine
- `chase/theme/pack_conformance_test.go` - This TRD's cssdiff.Equal acceptance-gate attestation (stress + scaffold)
- `chase/markdown/testdata/inline_svg_sample.md` - Sample deck for the rasterization checkpoint
- `chase/markdown/apply.go` - `[Rule 1 - Bug]` fix: `headingDividerDisplayValue` materializes the scalar display form

## Decisions Made

- **marp-gfm-table decision:** categorized PASS, not skip-mapped. Empirically confirmed (via `TestChaseCorpus`) that goldmark's stock `extension.GFM` table rendering, wired into `chase/markdown.NewEngine`, matches `expected.html` via `htmldiff.Equal` with zero Objective-3 dependency. Locked in as an explicit assertion (`marp-gfm-table: expected PASS`) so a future regression is reported by name.
- **headingDivider fix scope:** fixed at the display-materialization layer only (`apply.go`), NOT at the `CoerceGlobal`/`coerceHeadingDivider` layer — the expanded `[]int` range is a locked-in 01-02 contract `headingdivider.go`'s synthetic-break transformer depends on; only the author-facing rendered attribute needed correcting.
- **Rasterization proof strategy:** human-verify checkpoint (screenshot) now, deferred deterministic pixel-diff to Objective 5 — keeps `chase/`/`press/` Chrome-free (API-02) through Objective 1-4 while still satisfying ROADMAP criterion 3 today.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed headingDivider scalar materialization**
- **Found during:** Task 2 (writing `conformance/runner/chase_corpus_test.go` against `marp-heading-divider`)
- **Issue:** `chase/directive.CoerceGlobal("headingDivider", "2", nil)` correctly expands the author's scalar `"2"` into `[]int{1,2}` (a locked-in 01-02 contract `headingdivider.go`'s synthetic-break transformer needs), but `chase/markdown/apply.go`'s generic directive-materialization loop stringified this expanded range via `directiveValueString` as `"1,2"` for BOTH the `data-heading-divider` attribute and the `--heading-divider` CSS custom property. Real Marp/Marpit materializes the author-facing scalar `"2"` back onto the slide (per `conformance/corpus/cases/marp-heading-divider/expected.html`), not the internal expansion.
- **Fix:** Added `headingDividerDisplayValue(v any) string` — collapses an exact contiguous `[1..N]` range back to `"N"` for display; falls back to the existing comma-joined `directiveValueString` for any non-contiguous case (not exercised by this corpus). Special-cased `k == "headingDivider"` in the generic materialization loop to use this helper.
- **Files modified:** `chase/markdown/apply.go`
- **Verification:** `TestChaseCorpus` — `marp-heading-divider` PASSes via `htmldiff.Equal`; `go test ./chase/markdown/` remains green (no regression to the 01-02-locked `TestCoerceGlobalHeadingDividerAndTheme` contract, which is untouched).
- **Committed in:** `e9adebf` (part of Task 2's commit)

---

**Total deviations:** 1 auto-fixed (1 Rule-1 bug)
**Impact on plan:** Necessary for the `marp-heading-divider` corpus case to htmldiff-pass — a genuine Objective-1 mechanic gap per the TRD's own `error_recovery` guidance ("fix the responsible upstream TRD's code... do not hide it in the skip-map"). No scope creep; no architectural change; the 01-02-locked expanded-range contract is untouched.

## Issues Encountered

- `file://` URLs are blocked by this environment's browser-automation sandbox — worked around by briefly serving the checkpoint artifact on port 8091 (never 8080, per the standing hard rule) instead of opening the file directly; the local Python server was torn down immediately after the screenshot was captured.
- A self-authored test bug in `seam_test.go` (using a direct-children-only Section counter against an AST where inline-SVG wrapping nests Sections as descendants, not direct children) was caught and fixed before the GREEN commit — not a defect in `seam.go`/`apply.go`.

## User Setup Required

None - no external service configuration required.

## Next Objective Readiness

- **PARSE-01 is closed.** Objective 2's docmodel work can reuse `chase/markdown.Parse` (returns `(*ast.Document, parser.Context)`) as its AST-inspection hook directly — no new seam needed.
- **Objective 3** has a precise, requirement-tagged punch list: CORE-01 (embedded gaia/uncover themes), CORE-02 (size directive), CORE-03 (strikethrough), CORE-06 (emoji), CORE-07 (code highlight), CORE-08 (math) — each keyed to exactly one corpus case in `conformance/runner/chase_corpus_test.go`'s `chaseSkipMap`, ready to flip from BLOCKED to PASS as batteries land.
- **Objective 5** has a documented, scoped deferral: the deterministic headless-Chrome pixel-diff proof for inline-SVG rasterization, with a concrete fixture (`testdata/inline_svg_sample.md`) and a known-good manual screenshot (`/Users/justin/dev/01-08-inline-svg-checkpoint.png`) to calibrate against.
- No blockers.

## Self-Check: PASSED

- All 6 declared files (5 created, 1 modified) verified present on disk.
- All 4 task commit hashes (`16fcc47`, `902e93d`, `e9adebf`, `dd68671`) verified present in `git log --all`.
- Rasterization checkpoint artifact (`/private/tmp/.../scratchpad/inline_svg_sample.html`) and its browser screenshot (`/Users/justin/dev/01-08-inline-svg-checkpoint.png`) verified present on disk.
- `go build ./...`, `go test ./...`, `go vet ./...` all green at time of writing; `gofmt`/`addlicense -check` clean; `conformance/runner/corpus_test.go` confirmed byte-identical to HEAD.

---
*Objective: 01-chase-framework*
*Completed: 2026-07-21*
