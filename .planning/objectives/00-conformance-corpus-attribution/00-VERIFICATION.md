---
objective: 00-conformance-corpus-attribution
verified: 2026-07-20T23:49:09Z
status: passed
score: 8/8 requirements verified, 5/5 success criteria verified
notes:
  - kind: documentation_drift
    path: .planning/REQUIREMENTS.md
    note: "Checkbox list (lines 12-15) and traceability table (lines 132-139) mark CONF-01/CONF-03/CONF-04 as unchecked/'Pending' even though the underlying code is complete and passing. 00-03-SUMMARY.md explicitly states STATE.md/ROADMAP.md/REQUIREMENTS.md bookkeeping was deliberately deferred to the orchestrator ('to avoid parallel-edit conflicts'); the orchestrator updated ROADMAP.md's top-level objective checkbox to [x] complete but did not reconcile REQUIREMENTS.md's per-requirement checkboxes/status column or the ROADMAP Plans sub-checklist (00-03..00-06 still show '[ ]'). Advisory only — does not affect the actual deliverable, which was independently verified against the codebase below. Recommend a follow-up edit to flip CONF-01/03/04 to [x]/Complete and check off 00-03..00-06 in ROADMAP.md before starting Objective 1.
  - kind: documentation_drift
    path: .planning/STATE.md
    note: "Reports 'Job 2 of 6 complete / ~33% progress', stale from mid-Wave-2. All 6 SUMMARY.md files exist and all 6 TRDs' tests pass. Advisory only."
---

# Objective 0: Conformance Corpus, Acceptance Gate & Attribution Bootstrap Verification Report

**Objective Goal:** Establish the acceptance gate every other objective is validated against, and get licensing/attribution right from day one — before any vendored engine/theme asset exists.
**Verified:** 2026-07-20T23:49:09Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Build / Test / Vet (whole-repo gate)

| Command | Result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./...` | exit 0 — all packages `ok` (corpus, cssdiff, htmldiff, report, runner) |
| `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -check .` | exit 0 (run locally, mirrors `.github/workflows/ci.yml`) |

### Success Criteria (from ROADMAP.md "Objective 0")

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | Golden corpus seeded from Marp's own MIT Jest fixtures + full CommonMark (~600) + GFM (~672) spec sweeps, pass/fail tracked per section | ✓ VERIFIED | `conformance/corpus/cases/` has 18 real cases generated from `@marp-team/marp-core@4.4.0` via `tools/corpus-gen/gen.mjs` (verified `node_modules/@marp-team/marp-core` installed, version 4.4.0; sample case `marp-basic` produces genuine Marpit SVG-wrapped output). `conformance/spec/commonmark/spec.json` = 652 examples, `conformance/spec/gfm/spec.json` = 670, `conformance/spec/gfm/extensions.json` = 30 (`jq length` confirmed). `go test ./conformance/runner/ -v` → `TestSpecSweep` logs a **per-section** table (CM/ATX headings, CM/Raw HTML, GFM/Tables (extension), etc. — 60+ distinct sections), not an aggregate-only number. |
| 2 | DOM-normalized HTML diff via a narrow, explicit cosmetic allow-list, proven not to mask bugs by a negative test (broken `<pre>`/code-span still caught) | ✓ VERIFIED | `conformance/htmldiff/normalize_test.go`: `TestEqual_CosmeticDifferencesCompareEqual` (br/hr variants, attr order, whitespace → equal) and `TestEqual_RealDifferencesReportedAsDiff` including subtests `"code span inner text changed (a<b vs a>b)"` and `"pre content leading-space altered"` / `"pre content line changed"` — both PASS, confirming the negative case is caught, not masked. |
| 3 | CSS-AST diff comparator (new tooling, not repurposed HTML diff) detects an intentionally-broken theme as a failing case | ✓ VERIFIED | `conformance/cssdiff/diff.go` (`Equal(expected, actual string) (bool, string)`) built on its own token-stream-derived `Stylesheet` model (`model.go`, `build.go` — from `tdewolff/parse/v2/css`), independent of `htmldiff`. `diff_test.go`: `TestEqualChangedValue`, `TestEqualDroppedImportant`, `TestEqualChangedSelector` (the 3 broken-theme cases named in the how-to-verify) all assert `want=false` and PASS. |
| 4 | `LICENSE`(MIT)/`NOTICE`(crediting Marpit/Marp Core/Marp CLI + goldmark/chroma/latex2mathml/go-latex/latex)/README "inspired by, not affiliated" language + standing per-PR NOTICE checklist | ✓ VERIFIED | `LICENSE` = MIT, "Copyright (c) 2026 AO Cyber Systems". `NOTICE` credits Marpit/Marp Core/Marp CLI (with copyright/license/URL), goldmark, chroma, tdewolff/parse, latex2mathml + go-latex/latex (license verification explicitly and honestly deferred to the math objective — documented, not silently skipped), CommonMark spec + cmark-gfm (with exact pinned tags/commits and measured example counts). `README.md` line 26-28: "Eden Press is inspired by and Markdown-compatible with Marp ... not affiliated with, endorsed by, or sponsored by the Marp team." `.github/PULL_REQUEST_TEMPLATE.md` + `CONTRIBUTING.md` both carry an "Attribution checklist" / "New vendored/verbatim asset checklist" requiring a NOTICE update per new asset. |
| 5 | Scheduled CI job checks drift against Marp's latest upstream tag (mechanism, not a manual reminder) | ✓ VERIFIED | `.github/workflows/upstream-drift.yml`: `on.schedule: cron '0 6 * * 1'` (weekly) + `workflow_dispatch`, matrix over `[marp-team/marpit, marp-team/marp-core, marp-team/marp-cli]`, compares `gh release view --json tagName` against `UPSTREAM-VERSIONS.txt`, files a deduplicated GitHub issue (`upstream-drift` label) on drift. `UPSTREAM-VERSIONS.txt` present with all 3 pins. |

**Score:** 5/5 success criteria verified

### Requirement Coverage (all 8 REQ-IDs)

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| CONF-01 | 00-05-TRD/SUMMARY | Golden corpus from Marp's own Jest fixtures (MIT) | ✓ SATISFIED | 18 cases in `conformance/corpus/cases/`, generated from real `@marp-team/marp-core` v4.4.0 via `tools/corpus-gen/gen.mjs`; `conformance/runner/corpus_test.go::TestMarpCorpus` loads all 18, marks them `AddPending` (engine doesn't exist until Obj 3) — asserted via `if pending == 0 { t.Error(...) }`. `go test ./conformance/runner/ -v` shows all 18 as pending, 0 evaluated-and-failed. |
| CONF-02 | 00-02-TRD/SUMMARY | DOM-normalized HTML diff runner | ✓ SATISFIED | `conformance/htmldiff/normalize.go` + `normalize_test.go`; negative test (broken `<pre>`/code-span) present and passing (see criterion 2 above). |
| CONF-03 | 00-03 (spike) + 00-06 (comparator)-TRD/SUMMARY | CSS-AST diff comparator, new tooling | ✓ SATISFIED | `conformance/cssdiff/{model.go,build.go,diff.go}` — independent `Stylesheet` model + `Equal()`. `diff_test.go` has broken-theme negatives (changed value/dropped `!important`/changed selector) AND cascade-order negatives (`TestEqualRuleReorder`, `TestEqualDeclReorderSameProperty`) — all correctly report `false` (not equal). `spike_test.go` covers the underlying model build/normalization separately. |
| CONF-04 | 00-04-TRD/SUMMARY | Full CommonMark+GFM spec sweep, per-section | ✓ SATISFIED | `conformance/spec/{commonmark,gfm}/*.json` vendored (652/670/30 via `jq length`); `conformance/runner/spec_test.go::TestSpecSweep` reports **per section** (60+ sections logged, e.g. `CM/Raw HTML: 7/20`, `GFM/Tables (extension): 7/8`) plus an explicit, self-checked strikethrough `<s>`/`<del>` allow-list (`t.Error` fires if the allow-list either fails to consult the known delta or excuses an unrelated mismatch). Baseline: 1258/1352 (93.0%) on the goldmark base parser — expected, since Eden Press's own directive/theme engine doesn't exist yet; this is the acceptance-gate baseline objective 1+ will improve against. |
| LIC-01 | 00-01-TRD/SUMMARY | LICENSE (MIT) | ✓ SATISFIED | `LICENSE` — MIT, "Copyright (c) 2026 AO Cyber Systems". |
| LIC-02 | 00-01-TRD/SUMMARY | NOTICE/CREDITS | ✓ SATISFIED | `NOTICE` — see criterion 4 above; all named deps credited with URL+license, deferred items explicitly flagged rather than glossed over. |
| LIC-03 | 00-01-TRD/SUMMARY (mechanism) + Obj 3 (per-file headers, deferred) | Per-file MIT headers on verbatim Marp assets | ✓ SATISFIED (mechanism; per REQUIREMENTS.md's own documented split) | `CONTRIBUTING.md` has an explicit "two header templates" section: original Eden files get the Eden MIT header, verbatim Marp assets (3 themes + browser-fit script) get a **preserved 2018 Marp-team copyright** header — with the exact `addlicense` invocation for each. `.github/workflows/ci.yml` runs `addlicense -check` (verified locally, exit 0) ignoring only `conformance/corpus/cases/**` (Marp-derived golden data, correctly excluded per its own NOTICE-recorded provenance) and `node_modules`. The 3 themes/script don't exist yet (they land with CORE-01 in Objective 3) — REQUIREMENTS.md itself documents this split is intentional, not a gap; confirmed by inspection, not just trusted. |
| LIC-04 | 00-01-TRD/SUMMARY | README acknowledgment | ✓ SATISFIED | `README.md` — see criterion 4 above. |

**Score:** 8/8 requirements satisfied. No orphaned requirements (all 8 map to a TRD's `requirements:` frontmatter; cross-referenced against `.planning/REQUIREMENTS.md` traceability table, which maps all 8 to Objective 0).

### Key Wiring Verification

| From | To | Via | Status |
|---|---|---|---|
| `conformance/runner/runner.go` (`RunCase`) | `conformance/corpus`, `conformance/htmldiff`, `conformance/report` | Go imports + direct calls (`corpus.Case`, `htmldiff.Equal`, `rep.Add`) | ✓ WIRED |
| `conformance/runner/corpus_test.go` (`TestMarpCorpus`) | `conformance/corpus.LoadCases`, `conformance/runner.GoldmarkRenderFunc`, `conformance/report` | Direct calls, executed by `go test` | ✓ WIRED |
| `conformance/runner/spec_test.go` (`TestSpecSweep`) | `conformance/spec` embedded JSON, `conformance/htmldiff` | `go:embed` + `htmldiff.Equal` calls | ✓ WIRED |
| `.github/workflows/ci.yml` | `addlicense -check` | Shell step, reproduced locally (exit 0) | ✓ WIRED |
| `.github/workflows/upstream-drift.yml` | `UPSTREAM-VERSIONS.txt` + `gh release view` | Shell step reading the pin file, matrixed over 3 repos | ✓ WIRED (workflow syntax valid; not executed live since it requires GitHub Actions runtime/secrets — schedule + logic verified by inspection) |
| `tools/corpus-gen/gen.mjs` | `conformance/corpus/cases/*` | Real `@marp-team/marp-core` render → file write (verified: package installed, sample output is genuine Marpit-shaped HTML/CSS) | ✓ WIRED |

### Anti-Patterns Found

None. Scanned all objective-relevant source (`conformance/`, `tools/corpus-gen/gen.mjs`, `.github/workflows/`, `LICENSE`, `NOTICE`, `README.md`, `CONTRIBUTING.md`, `go.mod`) for `TODO|FIXME|XXX|HACK|PLACEHOLDER` and `placeholder|coming soon|will be here|not yet implemented` — zero hits outside vendored `node_modules`/golden-case CSS data (false positives from legitimate `color-scheme:light/dark` CSS tokens).

Two **documentation-drift** items were found in planning bookkeeping (not in the shipped deliverable) — see `notes:` in frontmatter above: `.planning/REQUIREMENTS.md`'s per-requirement checkboxes/status column and `.planning/STATE.md`'s progress line were not reconciled after Wave 2/3 executors explicitly deferred that bookkeeping to the orchestrator. Advisory only; does not affect the pass verdict since it is a tracking-artifact staleness, not a code/deliverable gap — every claim in those stale rows was independently re-verified against the actual codebase above.

### Functional Verification

Not applicable — Objective 0 is a backend/tooling-only objective (Go test harness, CI workflows, license files). No UI/web/mobile surface exists to drive via Playwright/Maestro.

### Human Verification Required

None. All 5 success criteria and all 8 requirements were verified programmatically against the actual codebase (build/test/vet, direct test execution with verbose output, file content inspection, and a live local `addlicense -check` run).

### Gaps Summary

No gaps. All 8 requirement IDs (CONF-01..04, LIC-01..04) have real, substantive, wired implementations verified by directly running the test suites and reading the underlying source — not by trusting SUMMARY.md claims. The one deliberate scope split (LIC-03: enforcement mechanism now, per-file headers on the not-yet-existing theme assets deferred to Objective 3) is explicitly documented in `.planning/REQUIREMENTS.md`'s traceability section and confirmed intentional by cross-reading `CONTRIBUTING.md`'s two-template header system and the CI `-ignore` flags, which correctly account for assets that don't exist yet.

The only issue found is non-blocking documentation staleness in `.planning/REQUIREMENTS.md` (checkboxes/status column for CONF-01/03/04 still read "Pending"/unchecked) and `.planning/STATE.md` (stuck at "2/6 jobs"), both explicitly attributed in `00-03-SUMMARY.md` to bookkeeping deferred to the orchestrator and apparently missed when Objective 0 was marked complete in `ROADMAP.md`. Recommend reconciling both files before planning Objective 1 so downstream agents don't read stale "Pending" status and re-do completed work.

---

*Verified: 2026-07-20T23:49:09Z*
*Verifier: Claude (verifier)*
