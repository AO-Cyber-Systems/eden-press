---
objective: 01-chase-framework
job: "04"
subsystem: theme
tags: [css, tdewolff, marpit, theme-pipeline, css-nesting, selector-scoping, cssdiff]

# Dependency graph
requires:
  - objective: 01-chase-framework (01-01)
    provides: chase/theme/selector — SplitList, Walk, String, JoinList, Prepend, Replace, MarkRoot, IncreasingSpecificity, InlineSVGContainerChain, NonSVGContainerChain, SlideChain
  - objective: 01-chase-framework (01-03)
    provides: chase/theme — Stylesheet{Meta,Rules,Atoms}, Parse, ParseMeta, ParseTheme
provides:
  - Two-tier ordered CSS scoping pipeline (Tier-1 Theme.Load at add-time, Tier-2 ThemeSet.Pack at render-time)
  - Pass{Name,Run} + RunPasses(sheet, passes...) shared pipeline-runner abstraction with PassError wrapping
  - CSS Nesting Level 1 down-leveling (implicit &-child, &-compound fusion, comma-list expansion, one-level :is()/:where() passthrough)
  - :root remap/specificity trick (rootMarkPass reused at both tiers, specificityPass strictly after scoping)
  - Selector-scoping via chase/theme/selector (container + slide chain fusion, scopePass/scopeSelector)
  - @import-theme recursive resolution with per-branch cycle detection (resolveImportTheme)
  - Scaffold CSS prepend + advanced-background CSS injection + pagination content neutralization
  - ThemeSet{Add,Get,Pack} render-time API with embedded ScaffoldCSS/AdvancedBackgroundCSS constants
  - cssdiff.Equal-verified full-pipeline fixtures for a synthetic stress theme and the scaffold theme
affects: [chase/render, chase/directive, any objective consuming packed CSS for slide rendering]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pass{Name,Run} + RunPasses(...) shared pipeline-runner reused across both Tier-1 (Load) and Tier-2 (Pack)"
    - "rootMarkPass is the SAME Pass value invoked at both tiers (Tier-1 on raw theme CSS, Tier-2 on freshly-injected scaffold/advanced-bg CSS)"
    - "Own serializers (renderPacked/renderRule/selectorText/declarationText) instead of reusing Stylesheet.String()/Rule.String()/Declaration.String() — needed for combinator/comma whitespace repadding lost by tdewolff's tokenizer"
    - "Cycle detection via a per-branch COPIED visited map (never mutated across sibling branches) in resolveImportTheme"
    - "mustLoadPlainRules/loadPlain bring meta-less static CSS (scaffold, advanced-bg) into the same Rule shape as real themes via Parse + RunPasses(passNesting, rootMarkPass)"

key-files:
  created:
    - chase/theme/pass.go
    - chase/theme/theme.go
    - chase/theme/pass_nesting.go
    - chase/theme/pass_root.go
    - chase/theme/scaffold.go
    - chase/theme/pass_import.go
    - chase/theme/pass_scaffold.go
    - chase/theme/pass_advancedbg.go
    - chase/theme/pass_pagination.go
    - chase/theme/pack.go
    - chase/theme/pack_test.go
  modified: []

key-decisions:
  - "pass_root.go built during Task 1 (one task earlier than the TRD's file_tree literally assigned) because theme.go's Tier-1 Load needs rootMarkPass to exist immediately — same total file set, no scope creep, just compile-order necessity"
  - "Added declarationText() in pack.go to repad whitespace after css.CommaToken in declaration VALUES — tdewolff's tokenizer drops whitespace adjacent to function-argument commas in values just as it does in selectors (already handled by selector.String()); needed for RESEARCH-verbatim/cssdiff fixture matching"
  - "Only @import-theme atoms are recursively resolved; plain @import atoms are deliberately left as recorded, unresolved AtRules — no filesystem/network layer exists in this TRD's scope"
  - "The AdvancedBackgroundCSS rule using the :marpit-container placeholder ALONE (not paired with :marpit-slide) is left un-substituted in Pack output — 01-01's locked selector.Replace/findPlaceholder only matches the exact 5-token :marpit-container > :marpit-slide sequence; modifying selector.go was out of scope. Confirmed present (as expected) in the frozen stress-theme fixture."
  - "Real Marpit ::backdrop retargeting logic is deferred (documented open question from RESEARCH); ::backdrop is scoped like any other selector (space-prepended), not specially retargeted"

patterns-established:
  - "Ordered Pass slice construction in Pack(): scaffoldPass -> [advancedBackgroundPass if InlineSVG] -> paginationPass -> rootMarkPass -> scopePass -> specificityPass"
  - "Frozen cssdiff fixtures reviewed via a throwaway debug-print test (deleted after review), not hand-typed blind"

requirements-completed: [THEME-03]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: true

# Metrics
duration: 22min
completed: 2026-07-20
---

# Objective 1 TRD 04: Two-Tier CSS Scoping Pipeline Summary

**Ordered two-tier CSS pipeline (Tier-1 Theme.Load, Tier-2 ThemeSet.Pack) wiring nesting down-level, :root remap, chase/theme/selector-based scoping, @import-theme resolution, scaffold/advanced-background injection, and pagination neutralization, verified against frozen cssdiff.Equal fixtures for a synthetic :is()/:where()/nesting stress theme and the Marpit scaffold theme.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-07-20T22:49:45-04:00 (first task commit)
- **Completed:** 2026-07-20T22:58:12-04:00 (last task commit)
- **Tasks:** 3
- **Files modified:** 11 (all new)

## Accomplishments
- Shared `Pass`/`RunPasses` pipeline-runner abstraction reused identically across both Tier-1 (`Theme.Load`) and Tier-2 (`ThemeSet.Pack`)
- Hand-rolled CSS Nesting Level 1 down-leveler (implicit `&`-child, `&`-compound fusion, comma-list expansion, one-level `:is()`/`:where()` passthrough), with an explicit, documented error for 2+ nesting depth
- `:root` remap/specificity trick correctly ordered: `rootMarkPass` at Tier-1 on raw theme CSS AND again at Tier-2 on freshly-injected scaffold/advanced-bg CSS, with `specificityPass` verified to run strictly after `scopePass`
- Full render-time `ThemeSet.Pack` pipeline: `@import-theme` recursive resolution (cycle-safe), scaffold CSS prepend, advanced-background CSS injection, pagination content neutralization, selector scoping via the locked `chase/theme/selector` package, specificity rewrite, and a dedicated repadding serializer (`renderPacked`/`renderRule`/`selectorText`/`declarationText`)
- Two frozen, hand-reviewed `cssdiff.Equal`-verified fixtures (stress theme with `:is()/:where()`/nesting + scaffold theme) exercising the entire ordered pipeline end-to-end

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Tier-1 Theme.Load, nesting down-level, scaffold embed | `go test ./chase/theme/ -run 'TestNesting\|TestThemeLoad\|TestScaffoldEmbedded'` | 0 | PASS |
| 2: Tier-2 Pack — import, scaffold, advanced-bg, pagination, scope, specificity | `go test ./chase/theme/ -run 'TestPack\|TestRoot\|TestImportTheme\|TestAdvancedBg\|TestPagination'` | 0 | PASS |
| 3: cssdiff.Equal full-pipeline acceptance gate | `go test ./chase/theme/ -run 'TestPackFullPipeline'` | 0 | PASS |

## Task Commits

Each task was committed atomically:

1. **Task 1: Tier-1 Theme.Load + nesting down-level + scaffold embed** - `b8d5103` (feat)
2. **Task 2: Tier-2 Pack — import resolve, scaffold, advanced-bg, pagination, scope, specificity** - `42d3dd3` (feat)
3. **Task 3: cssdiff.Equal acceptance gate for Pack(stress) + Pack(scaffold)** - `62cd9e8` (test)

**Plan metadata:** (this commit, below)

_Note: no TDD RED/GREEN split was used — tests were authored alongside implementation per task and verified passing before commit._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/theme/*.go` (empty output) | 0 | PASS |
| vet | `go vet ./chase/theme/` and `go vet ./chase/...` | 0 | PASS |
| test | `go test ./chase/theme/` (34 subtests, 0 failures) and `go test ./chase/...` | 0 | PASS |
| build | `go build ./...` | 0 | PASS |
| license | `addlicense -check` on all 11 new files | 0 | PASS |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (one self-corrected implementation detail — see Deviations — discovered and fixed via test failure before task commit, not a post-hoc Rule 1-4 cycle)
- **Must-haves verified:** 6/6 (ordered pipeline wired end-to-end; nesting down-level; :root remap/specificity ordering; selector-scoping reuse of locked chase/theme/selector; @import-theme recursive resolution; cssdiff-verified synthetic stress-theme fixture)
- **Gate failures:** None

## Files Created/Modified
- `chase/theme/pass.go` - `Pass{Name,Run}` + `RunPasses` shared pipeline-runner, `PassError` wrapping with `Unwrap()`
- `chase/theme/theme.go` - `Theme{Name,Sheet}`, Tier-1 `Load` (ParseTheme + passNesting + rootMarkPass), `loadPlain` for meta-less static CSS
- `chase/theme/pass_nesting.go` - CSS Nesting Level 1 down-leveler: `flattenNesting`, `flattenRule`, `expandNestedSelector`, `substituteAmpersand`, `joinCompounds`
- `chase/theme/pass_root.go` - `rootMarkPass`/`specificityPass` wrapping `selector.MarkRoot`/`selector.IncreasingSpecificity`, recursive `Children` handling
- `chase/theme/scaffold.go` - Embedded `ScaffoldCSS`, `AdvancedBackgroundCSS` constants (verbatim from RESEARCH), `ScaffoldThemeName`
- `chase/theme/pass_import.go` - `resolveImportTheme` recursive `@import-theme` resolution with per-branch cycle detection, `unquoteImportName`
- `chase/theme/pass_scaffold.go` - `scaffoldPass`/`prependScaffold` — prepends scaffold rules unless packing the scaffold theme itself
- `chase/theme/pass_advancedbg.go` - `advancedBackgroundPass`, `mustLoadPlainRules`; documents the `:marpit-container`-alone gap
- `chase/theme/pass_pagination.go` - `paginationPass`, neutralizes non-default `::after` pagination content
- `chase/theme/pack.go` - `PackOptions`, `ThemeSet{Add,Get,Pack}`, `NewThemeSet` (auto-registers scaffold identity), `scopePass`/`scopeSelector` (Prepend+Replace fusion), own serializers (`renderPacked`/`renderRule`/`selectorText`/`declarationText`)
- `chase/theme/pack_test.go` - 34 subtests covering nesting, Tier-1 Load, Tier-2 Pack, import resolution, scaffold/advanced-bg/pagination passes, and the two frozen cssdiff.Equal full-pipeline fixtures

## Decisions Made
- `pass_root.go` was built during Task 1 instead of its TRD-file_tree-listed Task 2 slot, purely for compile-order necessity (`theme.go`'s `Load` needs `rootMarkPass` immediately) — no scope change, same total files
- `@import` (plain) atoms are left unresolved/recorded-only; only `@import-theme` is recursively resolved, since no filesystem/network resource layer exists in this TRD's scope
- The documented `:marpit-container`-alone AdvancedBackgroundCSS gap rule is deliberately left un-scoped rather than modifying the locked `chase/theme/selector` package — confirmed present in the frozen stress-theme fixture as expected
- Real Marpit `::backdrop` retargeting is deferred (per RESEARCH's own open question); `::backdrop` is scoped like any ordinary selector

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Declaration-value comma whitespace lost by tokenizer**
- **Found during:** Task 2 (`TestAdvancedBgInjectedMatchesResearchVerbatim`)
- **Issue:** tdewolff's tokenizer drops whitespace immediately adjacent to a function-argument comma in declaration VALUES (not just selectors), and `stylesheet.go`'s `Declaration.String()`/`tokensText` do raw concatenation with no repadding — actual Pack output was `var(--marpit-advanced-background-split,50%)` instead of the expected `var(--marpit-advanced-background-split, 50%)`
- **Fix:** Added `declarationText(d Declaration) string` in `pack.go` that repads a trailing space after every `css.CommaToken` in the value, mirroring `selector.String()`'s existing repadding convention; switched `renderRule` to call it instead of `d.String()`
- **Files modified:** chase/theme/pack.go
- **Verification:** `TestAdvancedBgInjectedMatchesResearchVerbatim` and all other Task 2 tests pass after the fix
- **Committed in:** `42d3dd3` (part of Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, self-discovered/self-corrected before task commit)
**Impact on plan:** Necessary for correctness of rendered CSS values and for RESEARCH-verbatim/cssdiff fixture matching. No scope creep — mirrors an existing convention already used for selectors.

## Issues Encountered
None beyond the declaration-value whitespace issue documented above, which was resolved within Task 2 before commit.

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- `ThemeSet.Pack` is a complete, tested render-time API ready for consumption by `chase/render` (or an equivalent rendering subsystem) to produce final packed CSS per slide theme
- The documented `:marpit-container`-alone gap and the plain-`@import`-unresolved scope-narrowing should be revisited if a future objective needs full Marpit-parity advanced-background pseudo-element scoping or filesystem-backed `@import`
- No blockers for downstream objectives consuming `Theme.Load`/`ThemeSet.Pack`

## Self-Check: PASSED

**Files verified on disk:**
- FOUND: chase/theme/pass.go (70 lines)
- FOUND: chase/theme/theme.go (83 lines)
- FOUND: chase/theme/pass_nesting.go (181 lines)
- FOUND: chase/theme/pass_root.go (92 lines)
- FOUND: chase/theme/scaffold.go (139 lines)
- FOUND: chase/theme/pass_import.go (106 lines)
- FOUND: chase/theme/pass_scaffold.go (56 lines)
- FOUND: chase/theme/pass_advancedbg.go (76 lines)
- FOUND: chase/theme/pass_pagination.go (86 lines)
- FOUND: chase/theme/pack.go (263 lines)
- FOUND: chase/theme/pack_test.go (507 lines)

**Commits verified via `git log --oneline --all`:**
- FOUND: b8d5103 feat(01-04): Tier-1 Theme.Load + nesting down-level + scaffold embed
- FOUND: 42d3dd3 feat(01-04): Tier-2 Pack — import resolve, scaffold, advanced-bg, pagination, scope, specificity
- FOUND: 62cd9e8 test(01-04): cssdiff.Equal acceptance gate for Pack(stress) + Pack(scaffold)

**Gates re-verified at Self-Check time:**
- FOUND: `gofmt -l chase/theme/*.go` — no output (clean)
- FOUND: `go vet ./chase/theme/` and `go vet ./chase/...` — exit 0
- FOUND: `go test ./chase/theme/` — ok, 34 subtests, 0 failures
- FOUND: `go test ./chase/...` — ok (directive, theme, theme/selector)
- FOUND: `go build ./...` — exit 0
- FOUND: `addlicense -check` on all 11 files — exit 0 (all carry the Eden MIT header)

All claims in this SUMMARY verified against the actual filesystem and `git log --oneline --all`, not asserted from memory.

---
*Objective: 01-chase-framework*
*Completed: 2026-07-20*
