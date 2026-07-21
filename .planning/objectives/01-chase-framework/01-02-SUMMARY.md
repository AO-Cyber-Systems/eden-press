---
objective: 01-chase-framework
job: "02"
subsystem: parsing
tags: [goldmark, marpit, directives, yaml, carry-forward, go]

# Dependency graph
requires: []
provides:
  - "chase/directive package: pure, goldmark-free HTML-comment + YAML front-matter detection (PARSE-03)"
  - "chase/directive.CoerceGlobal/CoerceLocal: exact Marpit directive value-coercion tables (PARSE-02)"
  - "chase/directive.Resolve: global/local/spot carry-forward cursor state machine over an ordered event stream (PARSE-02/PARSE-07)"
affects: [01-05-chase-markdown, 01-06-chase-markdown]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Detection vs recognition split: DetectComment/DetectFrontMatter only extract raw key/value pairs; CoerceGlobal/CoerceLocal decide directive recognition separately (mirrors Marpit's comment.js vs directives/parse.js split)"
    - "Ordered KV/Event slices (never map[string]string) wherever declaration/merge order is semantically significant"
    - "Injected ThemeExists predicate to keep chase/directive free of a chase/theme import (zero cross-package dependency)"

key-files:
  created:
    - chase/directive/comment.go
    - chase/directive/frontmatter.go
    - chase/directive/yaml.go
    - chase/directive/directives.go
    - chase/directive/carryforward.go
    - chase/directive/directive_test.go
  modified: []

key-decisions:
  - "yaml.js reveals js-yaml + FAILSAFE_SCHEMA (string/sequence/mapping only, no bool/int auto-coercion) -- a minimal hand-rolled scalar/flow-list parser is a faithful port; no YAML library dependency added to go.mod"
  - "headingDivider coercion folds Marpit's two-stage int expansion (directives.js scalar passthrough + heading_divider.js's later int-to-[1..N] expansion) into one step in CoerceGlobal, per TRD 01-02's explicit task text and Test-list case 5"
  - "backgroundSplit excluded from CoerceLocal's known-key table -- verified NOT a real text-directive key in directives.js; it is derived from the ![bg left:30%] image-syntax grammar (background_image/parse.js), out of scope for chase/directive"
  - "Front-matter-derived directive candidates are modeled as ordinary DirectiveCommentEvents occurring before the first SlideOpen, so the SAME Resolve() carry-forward loop handles both front-matter and in-body comments without a separate code path"

patterns-established:
  - "Pattern: DetectX(...) (raw detection) + ParseX(...) (raw kv extraction) + CoerceGlobal/CoerceLocal (recognition + normalization) as three separable stages"

requirements-completed: [PARSE-02, PARSE-03]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 15min
completed: 2026-07-21
---

# Objective 1 TRD 02: chase/directive Carry-Forward + Directive Syntaxes Summary

**Pure, goldmark-free `chase/directive` package: HTML-comment/front-matter detection, exact Marpit directive value-coercion tables, and a global/local/spot carry-forward cursor state machine, ported test-first from Marpit's `comment.js`/`directives/*.js` source.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-21T02:00Z (approx.)
- **Completed:** 2026-07-21T02:03:37Z
- **Tasks:** 3
- **Files modified:** 6 (all newly created)

## Accomplishments
- `chase/directive/comment.go` + `frontmatter.go` + `yaml.go`: string-level HTML-comment span detection (multi-line aware, mirrors `comment.js`'s `/<!--+\s*([\s\S]*?)\s*--+>/`), leading `---`-fenced YAML front-matter detection, and a minimal YAML-ish scalar/flow-list parser -- detection is strictly separated from directive recognition (RESEARCH Pitfall 4).
- `chase/directive/directives.go`: `CoerceGlobal`/`CoerceLocal` port every built-in Marpit directive (`theme`, `headingDivider`, `style`, `lang` globally; `paginate`, `class`, `footer`, `header`, `color`, `backgroundColor/Image/Position/Repeat/Size` locally) plus the `_`-prefixed spot rule (`SpotKey`), with an injected `ThemeExists` predicate keeping the package free of a `chase/theme` import.
- `chase/directive/carryforward.go`: a pure `Resolve(events []Event, themeExists ThemeExists) []map[string]any` cursor state machine ported verbatim from `directives/parse.js`'s local-directive loop, correctly carrying `local` forward, resetting `spot` on every `SlideClose`, and stamping resolved `globals` onto every slide identically at the end.
- Verified zero goldmark import: `go list -deps ./chase/directive | grep -c goldmark` == 0.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Comment detection + front-matter + YAML-ish parser (PARSE-03) | `go test ./chase/directive/ -run 'Comment\|FrontMatter\|Yaml'` | 0 | PASS |
| 2: Directive tables + coercion + spot rule (PARSE-02 tables) | `go test ./chase/directive/ -run 'Coerce\|Global\|Local\|Spot'` | 0 | PASS |
| 3: Carry-forward cursor state machine (PARSE-02/07) | `go test ./chase/directive/` (15/15 tests) | 0 | PASS |

## Task Commits

Each task was executed test-first with a RED (failing test) commit followed by a GREEN (implementation) commit:

1. **Task 1: Comment detection + front-matter + YAML-ish parser**
   - `6597192` test(01-02): RED - failing tests for comment/front-matter/YAML-ish detection
   - `dcd1ac4` feat(01-02): GREEN - HTML-comment + front-matter detection + YAML-ish parser
2. **Task 2: Directive tables + coercion + spot rule**
   - `72f42f9` test(01-02): RED - failing tests for directive coercion tables + spot rule
   - `671d5d8` feat(01-02): GREEN - global/local directive coercion tables + spot rule
3. **Task 3: Carry-forward cursor state machine**
   - `87e298a` test(01-02): RED - failing tests for carry-forward cursor state machine
   - `89a72af` feat(01-02): GREEN - carry-forward cursor state machine

_No REFACTOR commits were needed -- each GREEN implementation was already the intended final shape._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/directive/ \| (! grep .) && go vet ./chase/directive/` | 0 | PASS |
| test | `go test ./chase/directive/` | 0 | PASS (15/15) |
| build | `go build ./...` | 0 | PASS |

## TDD Evidence

| Phase | Task | Command | Exit Code | Expected |
|---|---|---|---|---|
| RED | 1 | `go test ./chase/directive/ -run 'Comment\|FrontMatter\|Yaml' -v` | 1 | FAIL (correct — 5/5 new tests failed against stubs) |
| GREEN | 1 | `go test ./chase/directive/ -run 'Comment\|FrontMatter\|Yaml' -v` | 0 | PASS (correct — 5/5) |
| RED | 2 | `go test ./chase/directive/ -run 'Coerce\|Global\|Local\|Spot' -v` | 1 | FAIL (correct — 3/3 new tests failed against stubs) |
| GREEN | 2 | `go test ./chase/directive/ -run 'Coerce\|Global\|Local\|Spot' -v` | 0 | PASS (correct — 3/3) |
| RED | 3 | `go test ./chase/directive/ -v` | 1 | FAIL (correct — 6/7 new tests failed against stubs) |
| GREEN | 3 | `go test ./chase/directive/ -v` | 0 | PASS (correct — 15/15 total) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 5/5 truths, 6/6 artifacts, 2/2 key-links
- **Gate failures:** None
- **Zero-goldmark-import check:** `go list -deps ./chase/directive | grep -c goldmark` → `0` (confirmed)
- **addlicense:** `addlicense -check chase/directive/*.go` → clean (all 6 files carry the Eden MIT header)

## Files Created/Modified
- `chase/directive/comment.go` (56 lines) - `DetectComment`/`ParseComment`: multi-line-aware HTML-comment span detection + raw kv extraction, zero recognition logic
- `chase/directive/frontmatter.go` (77 lines) - `DetectFrontMatter`/`ParseFrontMatter`: leading `---`-fenced YAML block detection + raw kv extraction
- `chase/directive/yaml.go` (136 lines) - `ParseYAMLish`: minimal scalar/flow-list value parser (bare/quoted strings, `[a, b]` flow lists) faithfully mirroring js-yaml's FAILSAFE_SCHEMA restriction to string/sequence/mapping types
- `chase/directive/directives.go` (172 lines) - `CoerceGlobal`/`CoerceLocal`/`SpotKey`: exact Marpit globals{}/locals{} coercion tables + `_`-prefix spot rule
- `chase/directive/carryforward.go` (135 lines) - `Event`/`EventKind`/`EventsFromKV`/`Resolve`: the pure cursor{slide, local, spot} carry-forward state machine
- `chase/directive/directive_test.go` (334 lines) - 15 table-driven tests covering Test-list cases 1-9 plus corpus-mirroring coverage (marp-paginate, marp-header-footer front-matter shapes) and `EventsFromKV` ordering

## Decisions Made
- **YAML dialect resolved without a new dependency:** read `directives/yaml.js` first (per recovery instructions) and confirmed it uses js-yaml's `FAILSAFE_SCHEMA`, which never auto-resolves scalars to bool/int/null (everything stays a string; only sequences/mappings get structure). This means Marpit's OWN directive-coercion functions do the bool/int comparison at the string level (`v === 'true'`, `Number.parseInt(v, 10)`) — a minimal hand-rolled scalar/flow-list parser is a faithful, sufficient port. No YAML library was added to go.mod.
- **headingDivider's int-to-range expansion folded into CoerceGlobal:** the literal `directives.js` source returns a bare int for a scalar `headingDivider` value (the `[1..N]` array expansion actually happens later, in `heading_divider.js`'s core rule). TRD 01-02's task action text and Test-list case 5 both explicitly specify `headingDivider: 2 → [1,2]` as this package's own coercion output, so the expansion was folded into `CoerceGlobal` here (documented in code) rather than deferred to a later chase/markdown pass — a deliberate simplification per the TRD's own written spec, not a re-interpretation of Marpit.
- **`backgroundSplit` omitted from the locals table:** verified against the real `directives.js` locals{} object that no such text-comment directive key exists; it is derived from the `![bg left:30%]` image-syntax option grammar (`background_image/parse.js`), which is `chase/markdown`'s PARSE-05 territory. Including it here would have fabricated a directive Marpit itself doesn't recognize via comments/front-matter.
- **Front-matter events unified with in-body comment events:** rather than a separate front-matter code path in `Resolve`, front-matter-derived keys are simply emitted as `DirectiveCommentEvent`s occurring before the first `SlideOpen` — this mirrors Marpit's own `if (frontMatterObject.yaml) applyDirectives(...)` call appearing in BOTH the global-parse and local-parse core rules, and keeps the carry-forward state machine a single, pure loop.

## Deviations from Plan

None — TRD executed exactly as written. The three decisions above are all resolutions of the TRD's own explicitly flagged "Open Question 1" and gotchas (read-source-first instructions), not scope changes; no Rule-4 architectural stop was needed, and go.mod was not touched.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- `chase/directive` is a complete, standalone, zero-goldmark-import package: `CoerceGlobal`/`CoerceLocal`/`SpotKey`/`Resolve`/`EventsFromKV` plus the `Detect*`/`Parse*` front-end are all exported and ready for `chase/markdown` (TRD 01-05/06) to drive from a real goldmark `BlockParser`/`InlineParser`/`ASTTransformer` pipeline.
- No blockers. `chase/theme` (parallel sibling package, TRD 01-01/03/04) remains independently buildable with zero cross-import, as designed in 01-RESEARCH.md.

---
*Objective: 01-chase-framework*
*Completed: 2026-07-21*
