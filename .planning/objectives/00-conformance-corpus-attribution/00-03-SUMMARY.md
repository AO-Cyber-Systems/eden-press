---
objective: 00-conformance-corpus-attribution
job: "03"
subsystem: testing
tags: [css-ast, tdewolff, cssdiff, spike, grammar-stream, go]

# Dependency graph
requires:
  - objective: 00-01
    provides: "Authoritative go.mod/go.sum with tdewolff/parse/v2 v2.8.13 pinned; conformance/cssdiff/ directory skeleton"
provides:
  - "conformance/cssdiff package: normalized, order-preserving Stylesheet/Rule/Declaration model"
  - "cssdiff.Parse(css string) (Stylesheet, error) — a tdewolff/parse/v2/css grammar-TOKEN-stream builder (there is no p.AST() to consume)"
  - "Proven detectability exit criteria: a single changed declaration value, and a reordered pair of same-property declarations, both make two Stylesheet models NOT reflect.DeepEqual"
  - "Within-node normalization: hex color lowercasing (scoped to values, not ID selectors), whitespace collapse, comment stripping, canonical double-quote strings"
affects: [00-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "css.Parser Next()/Values() grammar-stream walk (BeginRulesetGrammar/DeclarationGrammar/EndRulesetGrammar/AtRuleGrammar/BeginAtRuleGrammar/EndAtRuleGrammar/ErrorGrammar) materializing a hand-rolled model, since tdewolff/parse/v2/css exposes tokens, not a navigable AST"
    - "ruleStack (stack of []Rule indices, with a discardedRule=-1 sentinel) tracks the currently-open Rule so declarations attach correctly even under native CSS nesting or at-rule-block discarding"
    - "At-rule contents captured only as a raw, opaque Rule.AtRule prelude string (e.g. \"@media(max-width:400px)\"); nested rules inside an at-rule block are deliberately NOT modeled (spike scope)"
    - "Hex-color lowering is scoped to declaration VALUES only — a HashToken in a selector is a case-sensitive ID selector, not a color, so selectors are never hex-lowered"

key-files:
  created:
    - conformance/cssdiff/model.go
    - conformance/cssdiff/build.go
    - conformance/cssdiff/spike_test.go
  modified: []

key-decisions:
  - "css.QualifiedRuleGrammar is handled identically to BeginRulesetGrammar defensively (per TRD hard_constraint #5), even though tracing tdewolff v2.8.13's parseQualifiedRule confirms it is never actually returned in stylesheet mode (isInline=false) — only BeginRulesetGrammar/ErrorGrammar are reachable there"
  - "Nested CSS rulesets (native nesting, e.g. `a { &:hover { ... } }`) are supported via a ruleStack so declarations always attach to the correct innermost open Rule, even though the Test List didn't require it — the flat Rule model has no parent/child field, so nested selectors surface as additional flat sibling Rules in document order (correctness safety net, not scope creep: ~15 lines)"
  - "!important detection operates at the token level (trailing DelimToken \"!\" + IdentToken \"important\"), relying on tdewolff's own whitespace-collapsing around \"!\" (verified against its upstream test suite) rather than string-suffix matching on the joined value"
  - "Quote canonicalization targets double-quotes as the canonical style; single-quoted content is re-escaped only for a literal embedded double-quote — sufficient for the spike's hand-built fixtures (no_llm_test_data), not a full CSS-string-escaping engine"

requirements-completed: [CONF-03]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 5min
completed: 2026-07-20
---

# Objective 00 TRD 03: CSS-AST Diff Spike — Normalized Model + tdewolff Grammar-Stream Builder Summary

**`cssdiff.Parse()` materializes an order-preserving Stylesheet/Rule/Declaration model from `tdewolff/parse/v2/css`'s grammar-token stream (no `p.AST()` exists), proven via `reflect.DeepEqual` to make both a changed declaration value and a reordered same-property declaration pair detectable — de-risking CONF-03 before the 00-06 comparator is built.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-20T23:07:09Z
- **Completed:** 2026-07-20T23:12:00Z
- **Tasks:** 2
- **Files modified:** 3 created

## Accomplishments
- Defined the exact `Stylesheet{Rules []Rule}` / `Rule{Selector, Declarations, AtRule}` / `Declaration{Property, Value, Important}` model from the TRD's codebase_examples, with rule/declaration order preserved throughout — nothing sorts or set-compares.
- Built `Parse(css string) (Stylesheet, error)` on top of `css.NewParser(...).Next()`/`.Values()`, walking the real v2.8.13 `css.GrammarType` constants (confirmed via `go doc github.com/tdewolff/parse/v2/css` and by reading `parse.go`/`parse_test.go` in the pinned module — `QualifiedRuleGrammar` exists as a constant but is never actually returned in stylesheet mode; only `BeginRulesetGrammar`/`ErrorGrammar` are reachable there, so it's handled defensively per hard_constraint #5 without being reachable in practice).
- Implemented within-node normalization only: hex colors lowercased in declaration values (never in selectors, to protect case-sensitive ID selectors), whitespace collapse (largely free from tdewolff's own lexer, which already canonicalizes whitespace runs to a single token), comment stripping (also largely free — tdewolff drops comments inside selectors/declarations at the lexer level; top-level between-rule comments are explicitly skipped), and canonical double-quote string normalization.
- Proved the spike's two exit criteria with `reflect.DeepEqual`: a changed declaration value, and a reordered pair of same-property declarations, both make two parsed `Stylesheet` values NOT equal — plus a positive control (identical CSS parsed twice IS equal) proving the comparison itself is meaningful.
- Ad hoc verified (not committed as a test) that `@import url(...)`, `@media (...) { ... }` block discarding, and rules following an at-rule block all behave correctly and preserve document order.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Normalized CSS model + grammar-stream builder | `go build ./conformance/cssdiff/` ; `go doc ./conformance/cssdiff` | 0 | PASS |
| 2: Spike test — model behavior + detectability exit criteria | `go test ./conformance/cssdiff/ -v` | 0 | PASS |

`go doc ./conformance/cssdiff` output (Parse/Stylesheet/Rule/Declaration all present):
```
package cssdiff // import "."
...
type Declaration struct{ ... }
type Rule struct{ ... }
type Stylesheet struct{ ... }
    func Parse(cssText string) (Stylesheet, error)
```

`go test ./conformance/cssdiff/ -v` (GREEN, full run):
```
=== RUN   TestParse_SingleRule_SingleDeclaration
--- PASS: TestParse_SingleRule_SingleDeclaration (0.00s)
=== RUN   TestParse_MultipleDeclarations_PreserveOrder
--- PASS: TestParse_MultipleDeclarations_PreserveOrder (0.00s)
=== RUN   TestParse_Important_CapturedAndStripped
--- PASS: TestParse_Important_CapturedAndStripped (0.00s)
=== RUN   TestParse_MultipleRules_PreserveOrder
--- PASS: TestParse_MultipleRules_PreserveOrder (0.00s)
=== RUN   TestParse_NormalizeHexColor
=== RUN   TestParse_NormalizeHexColor/three-digit
=== RUN   TestParse_NormalizeHexColor/six-digit
=== RUN   TestParse_NormalizeHexColor/mixed-case
--- PASS: TestParse_NormalizeHexColor (0.00s)
    --- PASS: TestParse_NormalizeHexColor/three-digit (0.00s)
    --- PASS: TestParse_NormalizeHexColor/six-digit (0.00s)
    --- PASS: TestParse_NormalizeHexColor/mixed-case (0.00s)
=== RUN   TestParse_CollapseWhitespace
--- PASS: TestParse_CollapseWhitespace (0.00s)
=== RUN   TestParse_StripComments
--- PASS: TestParse_StripComments (0.00s)
=== RUN   TestParse_NormalizeQuoteStyle
--- PASS: TestParse_NormalizeQuoteStyle (0.00s)
=== RUN   TestDetectability_IdenticalCSS_DeepEqual
--- PASS: TestDetectability_IdenticalCSS_DeepEqual (0.00s)
=== RUN   TestDetectability_ChangedValue_NotDeepEqual
--- PASS: TestDetectability_ChangedValue_NotDeepEqual (0.00s)
=== RUN   TestDetectability_ReorderedDeclarations_NotDeepEqual
--- PASS: TestDetectability_ReorderedDeclarations_NotDeepEqual (0.00s)
PASS
ok  	github.com/AO-Cyber-Systems/eden-press/conformance/cssdiff	0.231s
```

## Task Commits

Each task was committed atomically:

1. **Task 1: Normalized CSS model + grammar-stream builder** - `e9bd7d1` (feat)
2. **Task 2: Spike test — model behavior + mutation-detectability exit criterion** - `6d1bcd7` (test)

**Plan metadata:** this SUMMARY.md committed separately (docs).

_TDD note: RED was driven with a stub `Parse` returning `Stylesheet{}` against the full `spike_test.go`, confirmed failing on real assertions (not compile errors, no panics) with exit code 1; GREEN is the real `build.go` implementation shown above. Because the TRD splits model+builder (Task 1) and the test suite (Task 2) into separate declared file sets, the RED/GREEN cycle was driven during development against both files together and the resulting commits were then split along the TRD's task/file boundaries — see TDD Evidence below for the actual RED/GREEN command output._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| test | `go test ./conformance/cssdiff/` | 0 | PASS |
| lint | `go vet ./conformance/cssdiff/` | 0 | PASS |

Additional checks run: `gofmt -l conformance/cssdiff/*.go` → empty (clean); `addlicense -l mit -s -c "AO Cyber Systems" -check conformance/cssdiff/` → exit 0 (all 3 new files carry the Eden MIT header, byte-identical block to `doc.go`).

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED | `go test ./conformance/cssdiff/` (stub `Parse` returning `Stylesheet{}`) | 1 | FAIL (correct) — 11 of 12 tests failed on real assertion mismatches (e.g. `Parse() = cssdiff.Stylesheet{Rules:[]cssdiff.Rule(nil)}, want ...`); only the identical-CSS positive control incidentally passed since two empty stubs are trivially equal |
| GREEN | `go test ./conformance/cssdiff/ -v` (real `build.go`) | 0 | PASS (correct) — all 12 top-level tests / 3 subtests pass, including both detectability exit-criterion assertions |

REFACTOR phase was not needed — no changes were required after GREEN; `go vet` was clean on the first pass.

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 3/3 (order-preserving model built from the grammar-token stream; within-node-only normalization with order preserved; changed-value + reorder both detectable via `reflect.DeepEqual`)
- **Gate failures:** None

## Files Created/Modified
- `conformance/cssdiff/model.go` — `Stylesheet{Rules []Rule}` / `Rule{Selector, Declarations, AtRule}` / `Declaration{Property, Value, Important}`, exactly matching the TRD's codebase_examples shape
- `conformance/cssdiff/build.go` — `Parse(css string) (Stylesheet, error)`: walks `css.NewParser(...).Next()`/`.Values()` over `AtRuleGrammar`/`BeginAtRuleGrammar`/`EndAtRuleGrammar`/`BeginRulesetGrammar`/`QualifiedRuleGrammar`/`DeclarationGrammar`/`EndRulesetGrammar`/`TokenGrammar`/`ErrorGrammar`; applies within-node normalization (hex lowering scoped to values, quote canonicalization, `!important` extraction) while preserving order via a `ruleStack`
- `conformance/cssdiff/spike_test.go` — full Test List coverage (happy-path build/order/`!important`/multi-rule cases; normalization edge cases for hex/whitespace/comments/quotes; the two detectability exit-criterion assertions plus a positive-control identical-CSS assertion)

## Decisions Made
- Hex-color lowering is applied only when building declaration **values** (never selectors), because `css.HashToken` represents both hex colors in values AND ID selectors in selectors, and ID selectors are case-sensitive — lowering them would be a correctness bug, not a normalization.
- Added minimal, non-speculative at-rule capture: `AtRuleGrammar`/`BeginAtRuleGrammar` produce one `Rule{AtRule: "<raw prelude>"}` entry (e.g. `"@media(max-width:400px)"`), and any content nested inside a `BeginAtRuleGrammar`/`EndAtRuleGrammar` block is walked-but-discarded (not modeled) via an `atDepth` counter — this satisfies "capture a raw AtRule string only" (hard_constraint #6) without building `@media`/`@keyframes` semantics.
- Added a `ruleStack` (rather than always appending declarations to the last `Rule` in the slice) so declarations attach to the correct rule even under native CSS nesting or at-rule-block discarding — verified ad hoc (not committed as a test, since not in the Test List) that a trailing sibling rule after a discarded `@media` block still gets its own correct declarations.

## Deviations from Plan

None - TRD executed exactly as written. No Rule 1-4 deviations were needed: the tdewolff v2.8.13 grammar constants matched the TRD's codebase_examples (with the one already-anticipated caveat that `QualifiedRuleGrammar` is a dead branch in stylesheet mode, per hard_constraint #5's own error-recovery guidance), and no go.mod/go.sum edits were required beyond the anticipated `go mod download` (see Issues Encountered).

## Issues Encountered
- `go doc github.com/tdewolff/parse/v2/css` initially failed with "missing go.sum entry for go.mod file: github.com/tdewolff/test@v1.0.12" (a transitive go.mod-graph dependency of the pinned tdewolff/parse/v2, not an import of this package). Ran `go mod download github.com/tdewolff/test` per hard_constraint #2/#3's explicit remedy. This appended 2 lines to `go.sum` but the Go toolchain also incidentally rewrote `go.mod`'s `go 1.25` directive to `go 1.25.0`; that unwanted `go.mod` edit was reverted with `git checkout -- go.mod` before proceeding, confirmed unchanged (`git diff --stat go.mod` empty) for the remainder of the job. **The resulting `go.sum` delta (2 added lines for `github.com/tdewolff/test v1.0.12`) is intentionally left uncommitted** per instruction from a coordinating session, to be reconciled by the orchestrator on the integrated tree — `go build`/`go vet`/`go test` all pass locally with this local-only `go.sum` addition in place.
- A stray Maestro-style panic surfaced during RED-phase development (index-out-of-range in one subtest reading `Rules[0]` against an empty stub result), which aborted the RED test run before later tests could execute. Fixed by adding a length guard in that one test case (`TestParse_NormalizeQuoteStyle`) so RED evidence is a clean, complete run of real assertion failures instead of a partial run truncated by a panic.

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- CONF-03 is de-risked: the model shape and grammar-stream-walk mechanics are proven, including at-rule handling (raw-capture, no semantics) and native-nesting-safe declaration attachment.
- 00-06 (CSS-AST diff comparator + theme negative/order tests) can build directly on `cssdiff.Parse` → `Stylesheet` without re-deriving the tdewolff API; the `ruleStack`/at-rule-discarding pattern in `build.go` is available as a reference if 00-06 needs to extend at-rule modeling.
- Per a coordinating session's explicit instruction, this executor did **not** update `.planning/STATE.md`, `.planning/ROADMAP.md`, or `.planning/REQUIREMENTS.md` in this pass — that bookkeeping is owned by the orchestrator this wave to avoid parallel-edit conflicts with the sibling 00-02 branch. The orchestrator should mark CONF-03 / 00-03 complete and reconcile the `go.sum` delta noted above.

## Self-Check: PASSED

All 3 claimed files exist on disk (`conformance/cssdiff/model.go`, `build.go`, `spike_test.go`); both task commits (`e9bd7d1`, `6d1bcd7`) exist in git history (`git log --oneline` confirms both, most recent first). No missing items.

---
*Objective: 00-conformance-corpus-attribution*
*Completed: 2026-07-20*
