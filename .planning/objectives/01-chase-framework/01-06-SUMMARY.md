---
objective: 01-chase-framework
trd: "06"
subsystem: parsing
tags: [goldmark, marpit, directive, ast-transformer, carry-forward, pagination, go]

# Dependency graph
requires: ["01-02", "01-05"]
provides:
  - "chase/markdown/inline_style.go: ordered, dedupe-by-property InlineStyle builder (never a Go map for style serialization)"
  - "chase/markdown/directive.go: directive-apply ASTTransformer (priority 300) driving chase/directive's carry-forward from an ordered Section/Comment event stream, plus a front-matter BlockParser stripping YAML front-matter and feeding chase/directive.ParseFrontMatter into the transformer"
  - "chase/markdown/apply.go: directive -> data-*/--css-var/style materialization (the 4 special overrides + header/footer element insertion + two-pass pagination counter)"
  - "chase/markdown/render.go: KindHeaderElement/KindFooterElement render funcs (<header>/<footer> emission)"
affects: ["01-07", "01-08", "objective-3"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "orderedKeys parallel tracker: chase/directive.Resolve() intentionally returns unordered Go maps (values only); a separate buildOrderedKeysPerSlide() replays the EXACT SAME SlideOpen/SlideClose/DirectiveCommentEvent control flow, calling the SAME directive.CoerceGlobal/CoerceLocal/SpotKey functions, but tracks insertion ORDER (not values) via a touch()-idempotent orderedKeys{keys []string; seen map[string]bool} type -- mirrors JS's {...a,...b} spread-merge semantics (re-assigned key keeps its original position; new keys append)"
    - "front-matter as a genuine goldmark BlockParser (not just an ASTTransformer), following the vendored goldmark-meta precedent: Open() reads reader.Source() whole, calls chase/directive.DetectFrontMatter (holistic byte-span detection), and Advance()s past the entire block in one shot; Close() removes the placeholder node from the tree"
    - "CommentNode.KV is deliberately bypassed: chase/markdown/directive.go re-derives an ordered, correctly-typed []directive.KV from CommentNode.Raw via chase/directive.ParseComment(node.Raw) directly, since 01-05's .KV map is a lossy stringified map[string]string"
    - "apply.js ported branch-by-branch, in its FIXED code order (generic key loop -> lang -> class -> color -> backgroundColor -> backgroundImage -> paginate -> header/footer -> style attrSet), confirmed against the marp-bg-color fixture that specials apply in fixed order regardless of directive declaration order in source"
    - "pagination two-pass: a running pageNumber counter + paginating []*Section accumulator threaded through applyDirectives' per-section loop (advancePageNumber then applyDirectivesToSection->applyPaginateAttr), with a SEPARATE trailing pass stamping data-marpit-pagination-total once the final page count is known -- never computed inline"
    - "zero chase/theme cross-import preserved: ThemeExistsKey is an OPTIONAL parser.Context injection point (directive.ThemeExists predicate), defaulting to a permissive always-true function when unset"

key-files:
  created:
    - chase/markdown/inline_style.go
    - chase/markdown/directive.go
    - chase/markdown/apply.go
    - chase/markdown/directive_apply_test.go
  modified:
    - chase/markdown/markdown.go
    - chase/markdown/render.go

key-decisions:
  - "orderedKeys parallel-tracker design: chase/directive.Resolve() returns unordered map[string]any per slide by design (01-02's scope); rather than modify 01-02's carry-forward machine to also track key order, chase/markdown/directive.go replays its exact event-driven control flow a second time, using the SAME coercion functions, purely to derive per-slide key ORDER -- keeps 01-02 untouched and the ordering concern scoped entirely to this TRD"
  - "CommentNode.Raw re-derivation bypasses the lossy .KV map: 01-05's CommentNode.KV is a map[string]string that stringifies array values via fmt.Sprintf and loses declaration order; chase/markdown/directive.go instead calls chase/directive.ParseComment(node.Raw) directly to get a properly ordered, correctly-typed []directive.KV"
  - "HeaderElement/FooterElement fields named Content, not Text: ast.Node requires a Text(source []byte) []byte method (BaseNode's default text-extraction), which a same-named struct field collides with at compile time -- confirmed via goldmark ast.go source, fixed by renaming the field"
  - "Structural substring/attribute assertions (not full htmldiff.Equal) against the 3 required corpus directive cases (marp-class-spot, marp-paginate, marp-header-footer): reading real fixture input.md files from disk and asserting the specific data-*/style/header/footer output this TRD owns. Full htmldiff.Equal against raw expected.html would fail all 3 regardless of directive correctness, because current chase/markdown render output lacks the svg/foreignObject wrapper and heading-slug ids the fixtures carry -- both explicitly out of scope here per the TRD's own verification text ('heading-slug/inline-SVG differences remain, closed by 01-07 + 01-08 + Objective 3') and 'structurally green (sans Objective-3 batteries)'"
  - "Front-matter handled via a genuine BlockParser (frontMatterBlockParser), not folded into the ASTTransformer: chase/directive.DetectFrontMatter needs the WHOLE remaining source to find the closing '---' fence, so it must run at parse time (Open()) via reader.Advance(), mirroring goldmark-meta's own precedent -- an ASTTransformer only sees the already-parsed tree, too late to consume raw front-matter bytes"

patterns-established:
  - "Pattern: replay-for-order-only alongside an existing unordered resolver, when the resolver's own return shape is intentionally value-only and modifying it would widen its contract beyond its owning TRD's scope"
  - "Pattern: two-pass counters (pagination) live entirely in the driver loop (applyDirectives), never inside the per-section materializer (applyDirectivesToSection) -- keeps the trailing total-stamping pass auditable in one place"

requirements-completed: [PARSE-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~70min
completed: 2026-07-20
---

# Objective 1 TRD 06: Directive Resolution Wired Into chase/markdown Summary

**An ASTTransformer that drives 01-02's carry-forward state machine from an ordered Section/Comment event stream and materializes resolved directives onto each `<section>` as `data-*` attributes + an order-preserving inline style, including the 4 special CSS overrides and the two-pass pagination counter (PARSE-04).**

## Performance

- **Duration:** ~70 min
- **Tasks:** 3
- **Files created:** 4 (all new) + 2 modified

## Accomplishments

- `chase/markdown/inline_style.go`: `InlineStyle` — ordered `[]string` keys + `map[string]string` values, `Set(prop, value)` dedupes by property (last-write-wins, first-seen position preserved), `String()` serializes as `prop:value;prop2:value2;` in first-seen order. Backs every style write in `apply.go` — never a bare Go map.
- `chase/markdown/directive.go`: a `frontMatterBlockParser` (priority 0, triggers on `-`, only at line 0) stripping YAML front-matter via `chase/directive.DetectFrontMatter`/`ParseFrontMatter` into `FrontMatterKey`; a `directiveApplyTransformer` (priority 300, after slide-split@200/headingDivider@100) building an ordered `[]directive.Event` stream (front-matter globals, then per-Section `SlideOpen`+comment events via `directive.ParseComment(node.Raw)`+`SlideClose`), calling `chase/directive.Resolve` for values and a parallel `buildOrderedKeysPerSlide` (replaying the same event flow with `CoerceGlobal`/`CoerceLocal`/`SpotKey`) for key order.
- `chase/markdown/apply.go`: `applyDirectives`/`applyDirectivesToSection` — the full apply.js branch sequence ported verbatim (generic `data-{kebab}`/`--{kebab}` loop → `lang` → `class` attrJoin → `color` → `backgroundColor`[+`background-image:none`] → `backgroundImage`[+position/repeat/size, each override-able] → `paginate` → `header`/`footer` element insertion → `style` attrSet); `advancePageNumber`/`applyPaginateAttr` implement the two-pass pagination counter (skip/hold freeze the counter, retroactive page-1, `data-marpit-pagination-total` stamped in a trailing pass over the `paginating []*Section` accumulator); `HeaderElement`/`FooterElement` new AST node kinds (field named `Content`, not `Text`, to avoid colliding with `ast.Node`'s required `Text(source []byte) []byte` method).
- `chase/markdown/render.go`: registered `KindHeaderElement`→`renderHeader`/`KindFooterElement`→`renderFooter` emitting escaped `<header>...</header>`/`<footer>...</footer>`; existing `renderSection`'s `s.Attrs` emission (from 01-05) required no changes — it already writes every `Attr` in order.
- `chase/markdown/markdown.go`: registered `frontMatterBlockParser` (priority 0, distinct trigger byte from the comment parser, no priority race) and `directiveApplyTransformer` (priority 300).
- 21/21 tests pass in `chase/markdown` (12 new for this TRD + 9 inherited from 01-05, all still green).
- Structural corpus assertions (reading real `conformance/corpus/cases/{marp-class-spot,marp-paginate,marp-header-footer}/input.md` from disk) confirm correct directive materialization for all 3 required cases.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Ordered InlineStyle builder | `go test ./chase/markdown/ -run 'InlineStyle\|Style'` | 0 | PASS |
| 2: Directive-apply transformer + materialization | `go test ./chase/markdown/ -run 'Apply\|Directive\|Carry\|Global\|Spot\|Background'` | 0 | PASS |
| 3: Pagination two-pass counter + render emission | `go test ./chase/markdown/` (21/21) | 0 | PASS |

## Task Commits

Each task was executed test-first with a RED (failing test) commit followed by a GREEN (implementation) commit:

1. **Task 1: Ordered InlineStyle builder**
   - `db4efff` test(01-06): add failing InlineStyle ordering test
   - `620dbfe` feat(01-06): ordered dedupe-by-property InlineStyle builder
2. **Task 2: Directive-apply transformer + materialization**
   - `1e2dfe6` test(01-06): add failing directive materialization tests (cases 1-6)
   - `9c6f97f` feat(01-06): directive-apply ASTTransformer + materialization (data-*, ordered style, 4 special overrides, header/footer, front-matter)
3. **Task 3: Pagination two-pass counter + render emission**
   - `e284417` test(01-06): add failing pagination two-pass + corpus structural tests (cases 7, 9)
   - `548d3d8` feat(01-06): pagination two-pass counter + header/footer render emission

_No REFACTOR commits were needed — each GREEN implementation was already the intended final shape._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/markdown/*.go \| (! grep .) ; go vet ./chase/markdown/` | 0 | PASS |
| test | `go test ./chase/markdown/` (21/21) | 0 | PASS |
| build | `go build ./...` | 0 | PASS |
| license | `addlicense -check chase/markdown/*.go` | 0 | PASS (all 6 files) |
| full-repo test | `go test ./...` | 0 | PASS |

## TDD Evidence

| Phase | Task | Command | Exit Code | Expected |
|---|---|---|---|---|
| RED | 1 | `go test ./chase/markdown/ -run 'InlineStyle'` (inline_style.go absent) | 1 (build fail) | FAIL (correct — `undefined: NewInlineStyle`) |
| GREEN | 1 | `go test ./chase/markdown/ -run 'InlineStyle' -v` | 0 | PASS (correct — 2/2) |
| RED | 2 | `go test ./chase/markdown/ -run 'Directive'` (directive.go/apply.go absent) | 1 (build fail) | FAIL (correct — `undefined: New` extension wiring, etc.) |
| GREEN | 2 | `go test ./chase/markdown/ -run 'Apply\|Directive\|Carry\|Global\|Spot\|Background' -v` | 0 | PASS (correct — 6/6) |
| RED | 3 | `go test ./chase/markdown/ -run 'Pagination\|Corpus' -v` (pagination attrs + header/footer render missing) | 1 | FAIL (correct — `TestDirectivePaginationTwoPass`/`TestCorpusMarpPaginateStructural`/`TestCorpusMarpHeaderFooterStructural` failed on missing `data-marpit-pagination` attr and un-rendered header/footer elements; `TestCorpusMarpClassSpotStructural` already passed from Task 2) |
| GREEN | 3 | `go test ./chase/markdown/ -run 'Pagination\|Corpus' -v` | 0 | PASS (correct — 4/4) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 5/5 truths, 4/4 artifacts, 3/3 key_links
  - "Every recognized directive materializes as `data-{kebab}` + `--{kebab}` in an order-preserving inline style" → `TestDirectiveClassCarriesForward`, `TestCorpusMarpClassSpotStructural`
  - "4 special directives override real CSS" → `TestDirectiveColorSpecialOverride`, `TestDirectiveBackgroundColorSpecialOverride`, `TestDirectiveBackgroundImageSpecialOverride`
  - "Global stamped on every slide; local carries forward; spot applies to current slide only" → `TestDirectiveGlobalThemeStampedOnEverySlide`, `TestDirectiveClassCarriesForward`, `TestDirectiveSpotClassAppliesOnlyToCurrentSlide`
  - "paginate two-pass counter (skip/hold freeze, retroactive page 1, trailing total)" → `TestDirectivePaginationTwoPass`, `TestCorpusMarpPaginateStructural`
  - "InlineStyle order stable, dedupe by property, never a Go map" → `TestInlineStyleOrder`, `TestInlineStyleEmpty`
- **Gate failures:** None
- **Package boundary check:** `directive.go`'s `ThemeExistsKey` remains an optional injection point; `chase/markdown` still imports zero `chase/theme` (unchanged from 01-05).
- **addlicense:** `addlicense -check chase/markdown/*.go` → clean (all 6 files, including the 2 modified, carry the Eden MIT header).
- **go.mod:** untouched — `go mod tidy` never run; no new dependencies added.
- **Corpus directive cases now structurally green:** `marp-class-spot` (class carry-forward + spot), `marp-paginate` (two-pass counter + total), `marp-header-footer` (header/footer element insertion + data-*/style) — all 3 verified via `TestCorpusMarp*Structural`, reading the real `input.md` fixtures from disk. `marp-bg-color` background-color/-image special overrides verified via synthetic tests (`TestDirectiveBackgroundColorSpecialOverride`/`TestDirectiveBackgroundImageSpecialOverride`), not a corpus-fixture read, since the TRD's Test-list only required cases 1/2/7/9 to read real fixtures — case 4/5 (bg overrides) were scoped as synthetic-input tests per the Test-list itself.
- **Full htmldiff.Equal scoping decision:** NOT attempted against raw `expected.html` — would fail all 3 required cases regardless of directive correctness (current render lacks the svg/foreignObject wrapper + heading-slug ids, both deferred to 01-07/01-08/Objective 3 per this TRD's own verification text). Structural substring/attribute assertions against real fixture `input.md` files were used instead; documented as a deliberate scoping decision, not a shortfall.

## Files Created

- `chase/markdown/inline_style.go` (81 lines) — `InlineStyle` ordered dedupe-by-property builder
- `chase/markdown/directive.go` (306 lines) — front-matter `BlockParser`, directive-apply `ASTTransformer`, ordered event-stream builder, `orderedKeys` parallel key-order tracker
- `chase/markdown/apply.go` (351 lines) — directive → attribute/style materialization, `HeaderElement`/`FooterElement` AST kinds, two-pass pagination counter
- `chase/markdown/directive_apply_test.go` (317 lines) — 12 tests covering Test-list cases 1-9 (InlineStyle ordering, class carry-forward/spot, color/backgroundColor/backgroundImage overrides, global theme, pagination two-pass, 3 corpus structural cases)

## Files Modified

- `chase/markdown/markdown.go` — registered `frontMatterBlockParser`@0 and `directiveApplyTransformer`@300
- `chase/markdown/render.go` — registered `KindHeaderElement`/`KindFooterElement` render funcs

## Decisions Made

See `key-decisions` in frontmatter for the full list. Summarized:
- `orderedKeys` parallel-tracker (order-only replay of `Resolve()`'s event flow) rather than widening 01-02's `Resolve()` contract.
- `CommentNode.Raw` re-derivation via `chase/directive.ParseComment` to bypass 01-05's lossy `.KV` map.
- `HeaderElement`/`FooterElement.Content` field naming (not `Text`) to avoid the `ast.Node.Text(source []byte) []byte` interface-method collision.
- Structural substring assertions (not full `htmldiff.Equal`) against the 3 required corpus cases, with the svg/foreignObject/heading-slug gap explicitly out of scope per the TRD's own text.
- Front-matter as a genuine `BlockParser` (not folded into the `ASTTransformer`), since `DetectFrontMatter` needs the whole remaining source at parse time.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `ast.Node.Text()` interface-method collision on `HeaderElement`/`FooterElement`**
- **Found during:** Task 3 implementation (initial compile of `apply.go`'s new AST node kinds)
- **Issue:** `cannot use n (variable of type *HeaderElement) as ast.Node value ... *HeaderElement.Text is a field, not a method` — `ast.Node` requires a `Text(source []byte) []byte` method, which a same-named struct field shadows.
- **Fix:** Renamed the field to `Content` in both `HeaderElement` and `FooterElement`, updated constructors and `Dump()` methods accordingly.
- **Files modified:** `chase/markdown/apply.go`
- **Commit:** `9c6f97f`

**2. [Rule 1 - Bug] Test-assertion substring collision (`class="lead"` matching inside `data-class="lead"`)**
- **Found during:** Task 2 test-writing (`TestDirectiveClassCarriesForward`/`TestDirectiveSpotClassAppliesOnlyToCurrentSlide`)
- **Issue:** `strings.Count(out, `class="lead"`)` over-counted because `data-class="lead"` also ends with that substring.
- **Fix:** Changed the needle to a leading-space-prefixed `` ` class="lead"` `` to isolate the standalone `class` attribute.
- **Files modified:** `chase/markdown/directive_apply_test.go`
- **Commit:** `1e2dfe6`

### Scoped, Not a Deviation

Full `htmldiff.Equal` against raw corpus `expected.html` fixtures was intentionally NOT attempted (see Decisions Made / Post-TRD Verification above) — this is a scoping decision explicitly anticipated by the TRD's own verification text, not an unplanned deviation.

## Issues Encountered

None beyond the two auto-fixed items above — both caught immediately by compile/test failure, resolved inline, no blocking issues.

## User Setup Required

None — no external service configuration required.

## Next Objective Readiness

- `chase/markdown` now fully resolves and materializes the Marpit directive set (PARSE-04 complete): theme/style/headingDivider/paginate/header/footer/class/color/backgroundColor/backgroundImage/backgroundPosition/backgroundRepeat/backgroundSize, plus front-matter globals and spot (`_`-prefixed) directives.
- `HeaderElement`/`FooterElement` AST kinds and their render funcs are the ready-made extension point for 01-07's svg/advanced-bg work (explicitly left untouched per this TRD's gotchas).
- `ThemeExistsKey` remains an optional, unset-by-default injection point — 01-08/theme-integration can wire a real `chase/theme`-backed predicate without chase/markdown ever importing `chase/theme` directly.
- No blockers. `go.mod` untouched throughout (`go mod tidy` never run, no new dependencies).

---
*Objective: 01-chase-framework*
*Completed: 2026-07-20*

## Self-Check: PASSED

All 4 created files confirmed present on disk (`chase/markdown/{inline_style,directive,apply,directive_apply_test}.go`), and both modified files (`chase/markdown/{markdown,render}.go`) confirmed changed. All 6 referenced task commit hashes (`db4efff`, `620dbfe`, `1e2dfe6`, `9c6f97f`, `e284417`, `548d3d8`) confirmed present via `git log --oneline --all`.
