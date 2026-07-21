---
objective: 01-chase-framework
job: 03
subsystem: theme
tags: [css, tdewolff, parser, css-nesting, marpit, theme-metadata]

# Dependency graph
requires:
  - objective: 00-conformance-corpus
    provides: "conformance/cssdiff — the proven tdewolff/parse/v2/css grammar-walk pattern this package extends (not reuses)"
provides:
  - "chase/theme.Stylesheet{Meta, Rules, Atoms} — owned, token-preserving CSS model (SelectorTokens/Value stay []css.Token, never joined strings)"
  - "chase/theme.Parse(cssText) (Stylesheet, error) — purely structural css-token-stream builder; nesting preserved as Rule.Children with NestingDepth, at-rules RECORDED (not resolved) as Atoms"
  - "chase/theme.ParseMeta(cssText) (Meta, error) — @theme (required)/@size (repeatable, named table)/@auto-scaling extraction from the leading comment block, mirroring Marpit's postcss/meta.js regex"
  - "chase/theme.ParseTheme(cssText) (Stylesheet, error) — Parse + ParseMeta composed into one fully-populated Stylesheet"
  - "Meta.ResolveSize(name) (widthPx, heightPx int) — named-size lookup with 1280x720 default fallback"
affects: [01-04-scoping-pipeline, 01-07-inline-svg-mode]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Grammar-walk model builder over tdewolff/parse/v2/css (p.Next()/p.Values()), extending conformance/cssdiff's proven approach but keeping selector/value token LISTS instead of joined strings"
    - "Deep-clone (cloneTokens) every token slice stored beyond the current loop iteration — p.Values()/Token.Data are backed by the parser's reused internal buffers"
    - "Pointer-into-slice ancestor stack ([]*Rule) for nesting-aware tree building without flattening"
    - "Metadata extraction scoped to the leading comment only (not postcss's whole-document walkComments), regex-ported from Marpit's postcss/meta.js"

key-files:
  created:
    - chase/theme/stylesheet.go
    - chase/theme/stylesheet_test.go
    - chase/theme/parse.go
    - chase/theme/meta.go
    - chase/theme/meta_test.go
  modified: []

key-decisions:
  - "Parse() is purely structural and never requires @theme; @theme's required-ness (THEME-02) is enforced one layer up by ParseMeta/ParseTheme — needed to reconcile the TRD's plain, theme-less test-list CSS fixtures (cases 1-4) with the must-haves' '@theme is required' rule."
  - "@size supports both the full form (<name> <W>px <H>px>, Test-list case 7) and a small built-in fallback table for the two well-known bare keywords (4:3, 16:9) used by the RESEARCH stress-theme fixture (@size 4:3 alone)."
  - "ParseMeta scans only the CSS's leading comment block, not every comment in the document (Marpit's actual postcss/meta.js uses css.walkComments across the whole file) — matches the TRD's explicit Task 3 scope ('Extract metadata from the theme's leading comment block') and every corpus/stress-theme fixture's single-leading-block-comment convention."
  - "CustomPropertyGrammar (tdewolff's separate grammar event for --custom-property: value; declarations, confirmed via the pinned v2.8.13 source) is modeled as an ordinary Declaration alongside DeclarationGrammar — chase/theme has no need to distinguish the two at this layer."

patterns-established:
  - "Any token slice retained beyond a single grammar-walk loop iteration MUST go through cloneTokens first (aliasing-bug guard, verified via direct reproduction against the pinned parser)."

requirements-completed: [THEME-01, THEME-02]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~13min (commit-to-commit span; total session incl. source verification longer)
completed: 2026-07-20
---

# Objective 1 TRD 03: Stylesheet Model + Theme Metadata Parse Summary

**Token-preserving `chase/theme` CSS model over `tdewolff/parse/v2/css` — `Parse` builds nesting-aware `Stylesheet{Meta,Rules,Atoms}` from the grammar-token stream, `ParseMeta`/`ParseTheme` extract Marpit-style `@theme`/`@size`/`@auto-scaling` identity metadata from the leading comment, zero goldmark dependency.**

## Performance

- **Duration:** ~13 min (first commit `c998487` 22:02:40 → last commit `4547926` 22:15:03, 2026-07-20)
- **Tasks:** 3 (all TDD RED→GREEN)
- **Files created:** 5 (`stylesheet.go`, `stylesheet_test.go`, `parse.go`, `meta.go`, `meta_test.go`)

## Accomplishments

- Owned `Declaration`/`Rule`/`Size`/`Meta`/`AtRule`/`Stylesheet` model: selectors and declaration values kept as `[]css.Token` (never joined strings), so THEME-04's selector-rewriter can hand-walk `:is()`/`:where()` arguments; nested rulesets preserved as `Rule.Children` + `NestingDepth` (not flattened — TRD 01-04's job).
- `Parse(cssText) (Stylesheet, error)`: purely structural css-token-stream → model builder, handling simple rules, multi-selector comma-preserving selector lists, one-level (and deeper) CSS Nesting, `@import`/`@import-theme`/block at-rules (recorded only, never resolved), and `--custom-property` declarations.
- `ParseMeta(cssText) (Meta, error)`: extracts `@theme` (required — errors if absent), `@size` (repeatable, named table, full `<name> <W>px <H>px` form plus a bare-keyword fallback for `4:3`/`16:9`), and `@auto-scaling`, via a regex ported verbatim from the vendored `@marp-team/marpit/lib/postcss/meta.js` (`/^[*!\s]*@([\w-]+)\s+(.+)$/gim`).
- `ParseTheme(cssText) (Stylesheet, error)`: composes `Parse` + `ParseMeta` into one fully-populated `Stylesheet`.
- Verified via the actual pinned `tdewolff/parse/v2@v2.8.13` source (not recall) for every non-trivial grammar-event scenario, including the `CustomPropertyGrammar` vs `DeclarationGrammar` split and the `p.Values()`/`Token.Data` buffer-reuse aliasing hazard (reproduced directly, then guarded via a mandatory `cloneTokens` deep-clone helper).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Stylesheet model | `go test ./chase/theme/ -run 'Stylesheet\|Model\|Nesting' -v` | 0 | PASS |
| 2: Parse (css token stream → Stylesheet) | `go test ./chase/theme/ -run 'Parse\|Import\|AtRule' -v` | 0 | PASS |
| 3: Metadata parse (@theme/@size/@auto-scaling) | `go test ./chase/theme/ -v` (full package, 20 tests) | 0 | PASS |

## Task Commits

1. **Task 1: Stylesheet model** — RED `c998487` (test), GREEN `a9afcdd` (feat)
2. **Task 2: Parse — css token stream → Stylesheet (THEME-01)** — RED `13504b3` (test), GREEN `99358dc` (feat)
3. **Task 3: Metadata parse — @theme/@size/@auto-scaling + size table (THEME-02)** — RED `2e8ceae` (test), GREEN `4547926` (feat)

_All 6 commits made exclusively via `df-tools.cjs commit` (never raw `git commit`, per this worktree's commit-hook constraint)._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint (gofmt) | `gofmt -l chase/theme/*.go` | 0 (no output) | PASS |
| lint (vet) | `go vet ./chase/theme/` | 0 | PASS |
| test | `go test ./chase/theme/` (20 tests) | 0 | PASS |
| build | `go build ./...` | 0 | PASS |
| license header | `addlicense -check chase/theme/*.go` | 0 | PASS |
| goldmark-independence | `go list -deps ./chase/theme/... \| grep -i goldmark` | 1 (no match — confirmed absent) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./chase/theme/...` (stylesheet_test.go only, no stylesheet.go) | 1 | FAIL — `undefined: Rule` etc. (correct) |
| GREEN (Task 1) | `go test ./chase/theme/ -run 'Stylesheet\|Model\|Nesting' -v` | 0 | PASS (correct) |
| RED (Task 2) | `go test ./chase/theme/...` (Parse-based tests added, parse.go absent) | 1 | FAIL — `undefined: Parse` (correct) |
| GREEN (Task 2) | `go test ./chase/theme/ -run 'Parse\|Import\|AtRule' -v` | 0 | PASS (correct) |
| RED (Task 3) | `go test ./chase/theme/...` (meta_test.go added, meta.go absent) | 1 | FAIL — `undefined: ParseMeta`/`ParseTheme` (correct) |
| GREEN (Task 3) | `go test ./chase/theme/ -v` (full package) | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 3/3 (`@theme` required — errors when absent; `@size` full-form + bare-keyword resolves to a named table with 1280×720 default; `@auto-scaling` captured verbatim)
- **Gate failures:** None

## Files Created/Modified

- `chase/theme/stylesheet.go` — owned model: `Declaration`, `Rule` (token-preserving selector + nesting-aware `Children`/`NestingDepth`), `Size`, `Meta` (+ `ResolveSize`), `AtRule`, `Stylesheet`, plus `String()` serializers and the `tokensText` join helper.
- `chase/theme/stylesheet_test.go` — Task 1 model-literal tests (cases 1-3 hand-built) + Task 2 `Parse`-based tests (cases 1-4, plus scaffold-theme and stress-theme rule-count/nesting-shape checks).
- `chase/theme/parse.go` — `Parse`, `newAtRule`, `cloneTokens`, `extractImportant`: the grammar-walk builder.
- `chase/theme/meta.go` — `ParseMeta`, `ParseTheme`, `leadingComment`, `parseSizeValue`, `metaLineRE`/`sizeLineRE`/`bareSizeFallback`.
- `chase/theme/meta_test.go` — Task 3 tests (cases 5-8, bare-`@size` fallback, end-to-end `ParseTheme` integration).

## Decisions Made

See `key-decisions` in frontmatter: (1) `Parse` stays theme-metadata-agnostic, `ParseMeta`/`ParseTheme` own the required-`@theme` check; (2) `@size` supports both the full pixel form and a bare-keyword fallback table; (3) `ParseMeta` scans only the leading comment, not the whole document; (4) `CustomPropertyGrammar` is modeled as an ordinary `Declaration`.

## Deviations from Plan

None — TRD executed exactly as written. The four items above are interpretive design decisions needed to resolve genuine ambiguity in the TRD's own text (e.g., "@theme is required" vs. plain theme-less test fixtures in cases 1-4; Marpit's real `postcss/meta.js` scans every comment in the document but the TRD's Task 3 action explicitly scopes to "the theme's leading comment block"), not bugs or missing functionality — no Rule 1/2/3 auto-fixes were needed.

## Issues Encountered

None. All grammar-event behaviors (including the `CustomPropertyGrammar`/`DeclarationGrammar` split and the `p.Values()`/`Token.Data` buffer-reuse hazard) were verified directly against the pinned `tdewolff/parse/v2@v2.8.13` module-cache source before implementation, avoiding any trial-and-error against `go test` failures.

## Next Objective Readiness

- `chase/theme`'s `Stylesheet{Meta, Rules, Atoms}` model, `Parse`, and `ParseTheme` are ready as the input model for TRD 01-04's scoping pipeline (nesting down-leveling, `:root` specificity rewrite, section-size detection, `@import` resolution) and TRD 01-07's inline-SVG viewBox sizing (via `Meta.ResolveSize`).
- No blockers. `chase/theme/selector/` (TRD 01-01's territory) was not touched.

## Self-Check: PASSED

- FOUND: chase/theme/stylesheet.go
- FOUND: chase/theme/stylesheet_test.go
- FOUND: chase/theme/parse.go
- FOUND: chase/theme/meta.go
- FOUND: chase/theme/meta_test.go
- FOUND: .planning/objectives/01-chase-framework/01-03-SUMMARY.md
- FOUND commit: c998487 (test — Stylesheet model RED)
- FOUND commit: a9afcdd (feat — Stylesheet model GREEN)
- FOUND commit: 13504b3 (test — Parse RED)
- FOUND commit: 99358dc (feat — Parse GREEN)
- FOUND commit: 2e8ceae (test — metadata parse RED)
- FOUND commit: 4547926 (feat — metadata parse GREEN)

All claims in this SUMMARY verified against the actual filesystem and `git log --oneline --all`, not asserted from memory.

---
*Objective: 01-chase-framework*
*Completed: 2026-07-20*
