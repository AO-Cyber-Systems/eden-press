---
objective: 01-chase-framework
job: 01
subsystem: theme
tags: [css, selector-rewriter, tdewolff-parse, marpit-scoping, tdd]

# Dependency graph
requires:
  - objective: 01-chase-framework
    provides: 01-RESEARCH.md's Marpit scoping semantics (two-step placeholder + :root specificity trick), conformance/cssdiff's tdewolff/parse/v2/css usage pattern
provides:
  - Standalone, zero-parent-import CSS selector-token model (SplitList/Walk/String/JoinList) over tdewolff/parse/v2/css tokens
  - Two-step Marpit-style placeholder scoping (Prepend/Replace) for inline-SVG and non-SVG container chains
  - :root -> :marpit-root sentinel + post-scoping specificity-trick rewrite (MarkRoot/IncreasingSpecificity)
  - Dedicated regression suite proving the primitives against real marp-theme-gaia corpus selector shapes
affects: [01-02, 01-03, 01-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Token-level CSS rewriting (never flatten to string mid-pipeline) using tdewolff/parse/v2/css"
    - "Two-step placeholder scoping (fixed sentinel token sequence -> Replace with real chain) so :root rewriting can run strictly after scope-prefixing"
    - "Depth-aware flat-token Walk (FunctionToken/paren/bracket increment depth) for finding markers nested inside :is()/:where() arguments"
    - "Canonical re-serialization padding (String/JoinList) reinserts corpus-matching whitespace tdewolff's tokenizer drops around combinators and function-arg commas"

key-files:
  created:
    - chase/theme/selector/selector.go
    - chase/theme/selector/scope.go
    - chase/theme/selector/root.go
    - chase/theme/selector/selector_test.go
  modified: []

key-decisions:
  - "Test-list case 9 ('already-scoped/empty selector is a no-op') spans both Task 1 (SplitList/String empty + round-trip) and Task 2 (Prepend idempotency) — covered in both places since the TRD assigned it only to Task 1's numbered list but it conceptually depends on Task 2's Prepend"
  - "Task 3 (gaia regression suite) required zero new production code — it is a pure integration/regression proof that Task 1/2 primitives already generalize to real corpus selector shapes, not a TDD violation"
  - "Prepend's fused (same-element) branch triggers ONLY on a literal Ident('section') first token, matching Marpit's real prepend.js; MarkRoot emits literal text 'section:marpit-root' specifically so a :root rule fuses through the identical branch"
  - "String()/JoinList() reinsert canonical corpus spacing (single-space-padded combinators, space-padded function-arg commas, unpadded top-level list commas) at serialization time, since tdewolff's tokenizer drops the whitespace tokens themselves in those exact positions"

patterns-established:
  - "Selector-rewriter subsystem (chase/theme/selector) is importable and unit-testable with zero dependency on chase/theme's markdown/goldmark pipeline"

requirements-completed: [THEME-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 14min
completed: 2026-07-20
---

# Objective 01 TRD 01: Chase Selector-Rewriter Summary

**Standalone, token-level CSS selector-rewriter (chase/theme/selector) implementing Marpit's two-step placeholder scoping and :root specificity-trick rewrite over tdewolff/parse/v2/css tokens, with its own 14-test regression suite validated against real marp-theme-gaia corpus selectors.**

## Performance

- **Duration:** 14 min (first RED commit to last regression commit: 22:10:24 -> 22:24:15 local)
- **Started:** 2026-07-21T02:10:24Z
- **Completed:** 2026-07-21T02:24:15Z
- **Tasks:** 3
- **Files modified:** 4 (all newly created)

## Accomplishments
- Token-level selector-list model (`SplitList`, `Walk`, `String`, `JoinList`) that splits/rejoins only on top-level commas and never descends into `FunctionToken` arguments unless explicitly walked
- Two-step Marpit-style scoping (`Prepend` + `Replace`) reproducing the real `prepend.js`/`replace.js` fused-vs-spaced branching for inline-SVG and non-SVG container chains
- `:root` sentinel + specificity-trick rewrite (`MarkRoot` + `IncreasingSpecificity`) that finds `:root` even nested inside `:is()`/`:where()` arguments and rewrites it to `:where(section):not([\20 root])` strictly after scope-prefixing
- Dedicated regression suite (`TestGaiaRegression_ScopedSelectors`, 8 subtests) proving the primitives reproduce real `marp-theme-gaia/expected.css` selector output byte-for-byte

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Token-level Selector AST (comma split + compound segmentation) | `go test ./chase/theme/selector/ -run 'Split\|Segment\|List'` | 0 | PASS |
| 2: Two-step placeholder scoping + :root sentinel/specificity rewrite | `go test ./chase/theme/selector/` | 0 | PASS |
| 3: Standalone regression suite against gaia-shaped selectors | `go test ./chase/theme/selector/ -v` + `grep -L 'chase/theme"' chase/theme/selector/selector_test.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (raw `git commit` never used):

1. **Task 1 RED: failing tests for comma-split + segmentation** - `d7a392b` (test)
2. **Task 1 GREEN: selector token model (SplitList/Walk/String/JoinList)** - `78755ee` (feat)
3. **Task 1 addendum: already-scoped round-trip (case 9 continued)** - `ee313ea` (test)
4. **Task 2 RED: failing tests for scope-prepend + :root specificity rewrite** - `8dacdba` (test)
5. **Task 2 GREEN: two-step scope-prepend + :root specificity rewrite** - `9b544d0` (feat)
6. **Task 3: standalone gaia-shaped regression suite** - `dd04502` (test)

**Plan metadata:** `839af4a` (docs: create objective TRDs), `145140c` (docs: objective metadata) — pre-existing, not part of this TRD's execution.

_Note: Tasks 1 and 2 are TDD (RED->GREEN two-commit pairs); Task 3 is test-only by design (no production file), so it is a single commit that ran GREEN immediately, serving as regression proof rather than new-feature TDD._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/theme/selector/ \| (! grep .) ; go vet ./chase/theme/selector/` | 0 | PASS |
| test | `go test ./chase/theme/selector/` | 0 | PASS |
| build | `go build ./...` | 0 | PASS |

Additional evidence gathered beyond the TRD's 3 named gates:
- `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -check chase/theme/selector/` — exit 0 (all 4 new files carry the Eden MIT header)
- `go test ./...` (whole repo) — exit 0, all packages pass (`chase/theme/selector`, `conformance/corpus`, `conformance/cssdiff`, `conformance/htmldiff`, `conformance/report`, `conformance/runner`)
- `go test ./chase/theme/selector/ -v` — 14 top-level `--- PASS` test functions (several with table-driven subtests), 0 failures

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1, commit `d7a392b`) | `go test ./chase/theme/selector/ -run 'Split\|Segment\|List'` | 1 (compile/assert failure — no `selector.go` yet) | FAIL (correct) |
| GREEN (Task 1, commit `78755ee`) | `go test ./chase/theme/selector/ -run 'Split\|Segment\|List'` | 0 | PASS (correct) |
| RED (Task 2, commit `8dacdba`) | `go test ./chase/theme/selector/` | 1 (compile/assert failure — no `scope.go`/`root.go` yet) | FAIL (correct) |
| GREEN (Task 2, commit `9b544d0`) | `go test ./chase/theme/selector/` | 0 | PASS (correct) |

_REFACTOR: one intermediate `gofmt -w chase/theme/selector/scope.go` reformat (var-block alignment) applied after Task 2 GREEN; re-ran `go test ./chase/theme/selector/` afterward, exit 0 — no behavior change, folded into commit `9b544d0`'s working tree before commit._

_Task 3 (`dd04502`) is intentionally test-only (no companion production commit) — it exercises existing Task 1/2 primitives against real corpus-shaped input and passed GREEN on first run, confirming the primitives already generalize correctly rather than needing new code._

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 9/9 (all 9 test-list cases from the TRD covered: comma-split, function-arg-comma opacity, nested-comma segmentation, empty/already-scoped no-op, fused vs. spaced prepend, bare/fused `:root` sentinel, post-scope specificity rewrite, nested-marker gaia case)
- **Gate failures:** None

## Files Created/Modified
- `chase/theme/selector/selector.go` (265 lines) - Token-level primitives: `ParseSelectorTokens`, `SplitList`, `Walk`, `String`, `JoinList`, and the `mustParseTokens` dogfooding helper used by scope.go/root.go's fixed literals
- `chase/theme/selector/scope.go` (146 lines) - `Prepend`/`Replace` two-step placeholder scoping for inline-SVG and non-SVG container chains
- `chase/theme/selector/root.go` (206 lines) - `MarkRoot`/`IncreasingSpecificity` implementing the 4-step `:root` -> `:where(section):not([\20 root])` specificity trick
- `chase/theme/selector/selector_test.go` (462 lines) - Standalone test suite (imports only `testing` + `github.com/tdewolff/parse/v2/css` — no parent `chase/theme` import), 14 top-level test functions covering all 9 TRD test-list cases plus supplementary idempotency/ordering/depth checks and an 8-case gaia regression table

## Decisions Made
- Assigned test-list case 9 to both Task 1 (SplitList/String empty-input + already-scoped round-trip) and Task 2 (`Prepend` idempotency) since the TRD's numbered list placed it under Task 1 but its "already scoped" half is properly a Task 2 (`Prepend`) concern — documented rather than silently picked one interpretation.
- Confirmed via passing tests that `Prepend`'s fused branch must key off a literal `Ident("section")` first token (not any bare ident), and that `MarkRoot` deliberately emits literal text `"section:marpit-root"` (not a bare `:marpit-root"`) so a `:root` rule fuses through that exact same branch as a plain `section` rule.
- `String()`/`JoinList()` reinsert canonical corpus spacing (space-padded combinators and function-arg commas, unpadded top-level list commas) at serialization time rather than relying on tdewolff's preserved whitespace, since the tokenizer drops whitespace immediately adjacent to combinators and function-arg commas regardless of source spacing (confirmed empirically via scratch tokenizer dumps).

## Deviations from Plan

**1. [Rule 1 - Bug] Fixed wrong tdewolff TokenType constant name**
- **Found during:** Task 2 (writing `scope.go`/`root.go`)
- **Issue:** Used non-existent `css.Colon` instead of the real constant `css.ColonToken`, causing a compile failure across `root.go`, `scope.go`, and `selector_test.go`
- **Fix:** Grepped the `tdewolff/parse/v2@v2.8.13/css/lex.go` `TokenType = iota` block to confirm the exact name, then renamed all occurrences to `css.ColonToken`
- **Files modified:** `chase/theme/selector/scope.go`, `chase/theme/selector/root.go`, `chase/theme/selector/selector_test.go`
- **Verification:** `go build ./...` and `go test ./chase/theme/selector/` both passed immediately after the rename
- **Committed in:** `9b544d0` (part of Task 2 GREEN commit — fixed before commit, not a separate commit)

**2. [Rule 1 - Bug] Naive `String()` serialization didn't match corpus spacing**
- **Found during:** Task 1 (writing `selector.go`)
- **Issue:** First `String()` implementation wrote raw token `Data` byte-for-byte, producing `":is(h3,h4)+p"` instead of the corpus's `":is(h3, h4) + p"` — tdewolff's tokenizer drops whitespace around explicit combinators and function-arg commas
- **Fix:** Added combinator/comma-aware padding (`isCombinatorDelim` check + trailing space after `CommaToken`), confirmed against real spacing conventions in `conformance/corpus/cases/marp-theme-gaia/expected.css`
- **Files modified:** `chase/theme/selector/selector.go`
- **Verification:** `go test ./chase/theme/selector/ -run 'Split\|Segment\|List'` passed after the fix
- **Committed in:** `78755ee` (part of Task 1 GREEN commit)

---

**Total deviations:** 2 auto-fixed (2 bugs, both Rule 1)
**Impact on plan:** Both fixes were necessary for correctness (compile error; serialization fidelity against the real corpus). No scope creep — no new files, no architectural changes.

## Issues Encountered
- `css.Colon` vs `css.ColonToken` naming mismatch (see Deviations #1) — resolved by checking the actual dependency source, not blocking.
- tdewolff's whitespace-dropping tokenization around combinators/function-arg commas required empirical verification via a scratch tokenizer-dump program before `String()`/`JoinList()` could be written correctly (see Deviations #2) — resolved, not blocking.
- Earlier corpus-substring extraction (pre-existing session state) had produced a few gaia fixture strings missing their true leading prefix; re-extracted with a corrected anchor pattern before writing `TestGaiaRegression_ScopedSelectors` — resolved before any test was committed, not a runtime issue.

## User Setup Required
None - no external service configuration required. Pure Go package, zero new dependencies (uses the already-pinned `github.com/tdewolff/parse/v2`).

## Next Objective Readiness
- `chase/theme/selector` is fully unit-tested and ready to be consumed by the render pipeline (01-04), which owns steps (2)/(3) of the `:root` handling (a render-time second pass and the scope-prefix mechanics invocation) around this package's `MarkRoot`/`Prepend`/`Replace`/`IncreasingSpecificity` primitives.
- No blockers. `SplitList`/`JoinList` round-trip and the full inline-SVG/non-SVG scoping + `:root` rewrite pipeline are verified against real `marp-theme-gaia` corpus selector shapes, not just synthetic examples.

## Self-Check: PASSED

All created-file claims and commit-hash claims verified against disk/git before finalizing this summary:

- FOUND: `chase/theme/selector/selector.go`
- FOUND: `chase/theme/selector/scope.go`
- FOUND: `chase/theme/selector/root.go`
- FOUND: `chase/theme/selector/selector_test.go`
- FOUND: `.planning/objectives/01-chase-framework/01-01-SUMMARY.md`
- FOUND: `d7a392b`, `78755ee`, `ee313ea`, `8dacdba`, `9b544d0`, `dd04502` (all 6 in `git log --oneline --all`)

---
*Objective: 01-chase-framework*
*Completed: 2026-07-20*
