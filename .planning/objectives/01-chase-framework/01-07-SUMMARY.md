---
objective: 01-chase-framework
trd: "07"
subsystem: parsing
tags: [goldmark, marpit, background-image, inline-svg, advanced-background, ast-transformer, go]

# Dependency graph
requires: ["01-05", "01-06"]
provides:
  - "chase/markdown/image.go: ParseBgOptions — full `![bg …]` alt-text option-grammar parser (size keywords, split, direction, w:/h: dimensions, 9 CSS filters + drop-shadow, each with documented default amounts)"
  - "chase/markdown/background.go: extractBackgroundImages (two-phase walk + sweep-style empty-parent cleanup) and applyNonSVGBackground (reuses 01-06's applyDirectivesToSection directly for the non-SVG backgroundImage/backgroundSize path); shared Attr-slice helpers (cloneAttrs/overrideAttr/mergeStyleDecl/seedInlineStyle)"
  - "chase/markdown/inlinesvg.go: SvgOptionsKey/SvgOptions (opt-in parser.Context key, default disabled/1280x720) + svgTransformer (priority 400, last) + *Svg/*ForeignObject node kinds + wrapBaseSvg"
  - "chase/markdown/advancedbg.go: wrapAdvancedBackgroundSvg + buildAdvancedBackground — the 3-layer background/content/pseudo <foreignObject> structure, byte-exact against marp-bg-image/marp-bg-split"
  - "chase/markdown/render.go: render funcs for KindSvg/KindForeignObject/KindBackgroundLayer/KindPseudoLayer, plus shared writeAttrs/figureStyle helpers"
affects: ["01-08", "objective-3"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-phase (read-then-mutate) ast.Walk for safe node removal: extractBackgroundImages collects (*ast.Image, BgOptions) matches in a read-only Walk, THEN a separate loop detaches matches and removes any parent block left childless — never mutates during Walk itself"
    - "Attr-slice snapshot-and-override for sibling-layer attribute derivation: cloneAttrs + overrideAttr produce three independent Attrs slices (content/background/pseudo) from one mutated base, each with ONLY data-marpit-advanced-background overridden — mirrors advanced.js's own `{...attrs, key: value}` spread-and-override shape verbatim"
    - "SvgOptionsKey is opt-in (Enabled defaults false), not opt-out: preserves every pre-existing 01-05/01-06 test's assumption that doc's top-level children are bare *Section nodes unless a caller explicitly requests the <svg><foreignObject> wrap"
    - "applyNonSVGBackground reuses apply.go's applyDirectivesToSection directly (same package, not reimplemented) by constructing a resolved map + keys slice exactly as real Marpit injects backgroundImage/backgroundSize into the SAME resolved-directives map before the generic per-key materialization pass runs"

key-files:
  created:
    - chase/markdown/image.go
    - chase/markdown/background.go
    - chase/markdown/inlinesvg.go
    - chase/markdown/advancedbg.go
  modified:
    - chase/markdown/markdown.go
    - chase/markdown/render.go
    - chase/markdown/background_test.go

key-decisions:
  - "SvgOptionsKey defaults to Enabled=false (not true): every existing 01-05/01-06 test parses with no parser.Context key set at all and asserts doc's direct children are bare *Section nodes; making inline-SVG mode opt-in via the context key was the only way to add the wrap without touching/breaking those tests, and the TRD's own must_haves explicitly describe a DISABLED mode as a first-class supported path"
  - "wrapAdvancedBackgroundSvg lives in advancedbg.go (Task 3), called directly from svgTransformer.Transform (inlinesvg.go, Task 2) via a one-line branch on len(data.Images) > 0 — a single transformer object handles both the base wrap and the advanced-bg wrap rather than two separately-registered ASTTransformers, since both must run at the exact same point (after directive-apply, per-Section) and share the same extractBackgroundImages/opts inputs"
  - "Filter has no dedicated branch in apply.go's fixed directive-materialization sequence (apply.go/directive.go are explicitly out of files_modified for this TRD); applyNonSVGBackground therefore merges the filter CSS onto the section's style attribute directly via mergeStyleDecl AFTER calling applyDirectivesToSection, rather than widening apply.go's branch set"
  - "Non-SVG mode's applyNonSVGBackground applies only the LAST bg-marked image on a slide (mirrors background_image/apply.js's own 'only the last image's backgroundImage/backgroundSize applies' precedent); split/direction on a slide are likewise taken from the last bg-marked image whose alt text carries them, for the SAME reason"

patterns-established:
  - "Pattern: opt-in parser.Context keys default to the CURRENTLY-SHIPPED behavior (disabled), not the eventually-intended default, when retrofitting a new always-on-in-real-Marpit feature onto an existing test suite that assumed its absence"
  - "Pattern: derive N sibling AST nodes' attribute sets via clone-then-override from one shared, already-mutated base, rather than independently constructing each from scratch — keeps 'same base, one field differs' invariants trivially correct by construction"

requirements-completed: [PARSE-06, PARSE-07]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~110min
completed: 2026-07-21
---

# Objective 1 TRD 07: Background Images + Inline-SVG Advanced Backgrounds Summary

**`![bg …]` alt-text option-grammar parser feeding both a non-SVG `backgroundImage` local-directive path (reusing 01-06's materialization) and an opt-in inline-SVG `<svg><foreignObject>` wrap with the full 3-layer background/content/pseudo advanced-background structure, verified byte-exact against the marp-bg-image/marp-bg-split corpus fixtures.**

## Performance

- **Duration:** ~110 min
- **Tasks:** 3
- **Files created:** 4 (all new) + 3 modified

## Accomplishments

- `chase/markdown/image.go`: `ParseBgOptions` — full alt-text option grammar ported from `background_image/parse.js` + `image/parse.js`: `bg` flag, size keywords (`auto`/`contain`/`cover`, `fit` aliasing `contain`), bare `NN%` resize, `w:`/`h:` dimensions with the full CSS unit set (bare number → px), `left|right[:NN%]` split, `vertical|horizontal` direction, and 9 CSS filter functions + `drop-shadow` each with their documented default amount.
- `chase/markdown/background.go`: `extractBackgroundImages` (two-phase read-then-mutate `ast.Walk`, sweep-style empty-parent cleanup) aggregates every bg-marked image on a slide into a `backgroundSlideData`; `applyNonSVGBackground` materializes the slide's LAST bg image as a `backgroundImage`(+`backgroundSize`) local directive by calling 01-06's `applyDirectivesToSection` directly (never reimplemented), then merges any filter CSS onto the style attribute separately. Shared `cloneAttrs`/`overrideAttr`/`mergeStyleDecl`/`seedInlineStyle` helpers back both this file and `advancedbg.go`.
- `chase/markdown/inlinesvg.go`: `SvgOptionsKey`/`SvgOptions` (opt-in `parser.Context` key, `Enabled` defaults `false`, `Width`/`Height` default 1280×720 — NEVER a `chase/theme` import); `*Svg`/`*ForeignObject` node kinds; `svgTransformer` (registered LAST at priority 400, strictly after directive-apply@300) that, per top-level `*Section`, extracts bg images, then either reuses the non-SVG path (disabled), wraps the bare section in `<svg><foreignObject>` (enabled, no bg images), or delegates to the advanced-background builder (enabled, bg images present).
- `chase/markdown/advancedbg.go`: `wrapAdvancedBackgroundSvg`/`buildAdvancedBackground` — the 3-layer background→content→pseudo `<foreignObject>` structure in strict DOM order, split width/x adjustment (`reducedPercent`), `--marpit-advanced-background-split` style merge, and `pseudoColorStyle` (pseudo layer mirrors only the content section's resolved `color`). `*BackgroundLayer`/`*PseudoLayer` node kinds.
- `chase/markdown/render.go`: render funcs for `KindSvg` (`<svg data-marpit-svg="" viewBox="0 0 W H">`), `KindForeignObject` (`width`/`height`/optional `x`/optional `data-marpit-advanced-background`), `KindBackgroundLayer` (`<section>` + direction container + one `<figure>` per image), `KindPseudoLayer` (empty `<section>`); shared `writeAttrs`/`figureStyle` helpers.
- `chase/markdown/markdown.go`: registered `svgTransformer` at priority 400 (last).
- 30/30 tests pass in `chase/markdown` (9 new for this TRD + 21 inherited from 01-05/01-06, all still green — zero regressions).
- `TestCorpusMarpBgImageStructural`/`TestCorpusMarpBgSplitStructural` confirm the rendered advanced-background HTML matches `conformance/corpus/cases/marp-bg-{image,split}/expected.html` byte-for-byte on every substring this TRD owns (svg/foreignObject/section/figure structure), reading the real `input.md` fixtures from disk.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: `![bg …]` option-grammar parser | `go test ./chase/markdown/ -run 'Bg\|Option\|Image\|Filter'` | 0 | PASS |
| 2: Inline-SVG base wrap + non-SVG bg path | `go test ./chase/markdown/ -run 'Svg\|Inline\|NonSvg\|Background'` | 0 | PASS |
| 3: Advanced-background 3-layer transformer | `go test ./chase/markdown/` (30/30) | 0 | PASS |

## Task Commits

Each task was executed test-first and committed atomically the moment it went GREEN:

1. **Task 1: `![bg …]` option-grammar parser** — `710a808` feat(01-07): bg option-grammar parser (Task 1)
2. **Task 2: Inline-SVG base wrap + non-SVG bg path** — `562e9fa` feat(01-07): inline-SVG base wrap + non-SVG bg path (Task 2)
3. **Task 3: Advanced-background 3-layer transformer** — `3db97cf` feat(01-07): advanced-background 3-layer transformer (Task 3)

_All commits made via `df-tools.cjs commit` per the CRITICAL commit-mechanism constraint — raw `git commit` never invoked._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/markdown/*.go` (empty) + `go vet ./chase/markdown/` | 0 | PASS |
| test | `go test ./chase/markdown/` (30/30) | 0 | PASS |
| build | `go build ./...` | 0 | PASS |
| license | `addlicense -check chase/markdown/*.go` | 0 | PASS (all files, including the 4 new) |
| full-repo test | `go test ./chase/...` | 0 | PASS |
| theme boundary | `go list -deps ./chase/markdown \| grep -c 'chase/theme'` | 0 | PASS (0 matches — zero cross-import preserved) |

## TDD Evidence

| Phase | Task | Command | Exit Code | Expected |
|---|---|---|---|---|
| RED | 1 | `go test ./chase/markdown/ -run 'TestBgOptionParse' -v` (image.go temporarily hidden) | 1 (build fail) | FAIL (correct — `undefined: ParseBgOptions` x9) |
| GREEN | 1 | `go test ./chase/markdown/ -run 'TestBgOptionParse' -v` (image.go restored) | 0 | PASS (correct — 9/9 subtests) |
| RED | 2 | `go test ./chase/markdown/` (inlinesvg.go added, background.go/markdown.go/render.go not yet updated) | 1 (build fail) | FAIL (correct — `undefined: extractBackgroundImages`/`applyNonSVGBackground`; coordinator flagged this build break, addressed before any further work) |
| GREEN | 2 | `go test ./chase/markdown/ -run 'InlineSvg\|NonSvg\|Background' -v` | 0 | PASS (correct — 3/3, plus full package 27/27) |
| RED | 3 | (advancedbg.go authored with a fabricated `newNodeKind`/`baseLayerNode` helper that doesn't exist in this codebase) `go build ./...` | 1 (build fail) | FAIL (correct — `undefined: newNodeKind`/`undefined: baseLayerNode`/`undefined: ast`); fixed inline to the established `ast.NewNodeKind`/`ast.BaseBlock` convention before tests were run |
| GREEN | 3 | `go test ./chase/markdown/ -run 'CorpusMarpBg\|AdvancedBg' -v` | 0 | PASS (correct — 2/2, plus full package 30/30) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (the coordinator-flagged build break from Task 2's `inlinesvg.go` landing before `background.go` existed — see Deviations below)
- **Must-haves verified:** 5/5 truths, 5/5 artifacts, 3/3 key_links
  - "`![bg …](url)` alt-text options parse: bg, size keywords (fit = contain alias), split..., vertical|horizontal, NN%, w:/h: with units, filters with default amounts" → `TestBgOptionParseKeywords`, `TestBgOptionParseDimensions`, `TestBgOptionParseFilters`
  - "Inline-SVG mode wraps EVERY slide as `<svg data-marpit-svg viewBox=\"0 0 W H\"><foreignObject width=W height=H><section…>`" → `TestInlineSvgWrapsPlainSlide`, `TestInlineSvgViewBoxOverride`
  - "With inline-SVG DISABLED, a single bg image becomes a backgroundImage LOCAL directive — no new HTML structure" → `TestNonSvgBackgroundImageDirective`
  - "With inline-SVG ENABLED, bg images trigger advanced-background mode: three foreignObjects... in DOM order, with figures, split width/x adjustment, and `--marpit-advanced-background-split`" → `TestCorpusMarpBgSplitStructural`
  - "marp-bg-color, marp-bg-image, marp-bg-split render byte-for-byte matching the corpus (modulo Objective-3 batteries)" → `TestCorpusMarpBgImageStructural`, `TestCorpusMarpBgSplitStructural`, plus pre-existing `TestDirectiveBackgroundColorSpecialOverride`/`TestDirectiveBackgroundImageSpecialOverride` (marp-bg-color, 01-06)
  - Artifacts: `image.go` (`func ParseBgOptions`), `background.go` (`func extractBackgroundImages`/`applyNonSVGBackground`), `inlinesvg.go` (`func newSvgTransformer`), `advancedbg.go` (`func wrapAdvancedBackgroundSvg`), `background_test.go` (9 new `func Test...`) — all present and non-empty.
  - Key links: inlinesvg.go→render.go (`KindSvg`/`KindForeignObject` registered + rendered with `viewBox`/`width`/`height`), advancedbg.go→render.go (`KindBackgroundLayer`/`KindPseudoLayer` rendered with `data-marpit-advanced-background`), background.go→apply.go (`applyNonSVGBackground` calls `applyDirectivesToSection` directly) — all confirmed via passing tests above.
- **Gate failures:** None (after the one auto-fix cycle below).
- **Package boundary check:** `go list -deps ./chase/markdown | grep -c 'chase/theme'` → `0`. W/H arrive exclusively via `SvgOptionsKey`, never a `chase/theme` import.
- **addlicense:** `addlicense -check chase/markdown/*.go` → clean, all files including the 4 new ones.
- **go.mod:** untouched throughout — `git diff --stat go.mod go.sum` empty; `go mod tidy` never run; no new dependencies.
- **Port constraint:** N/A — this TRD involved no server/network verification work; port 8080 was never referenced, port 8091 was never needed.

## Files Created

- `chase/markdown/image.go` (194 lines) — `BgOptions`, `ParseBgOptions`, filter-matcher table, CSS-length normalization, CSS-escape helper
- `chase/markdown/background.go` (206 lines) — `bgImage`/`backgroundSlideData`, `extractBackgroundImages`, `imageAltText`, `applyNonSVGBackground`, `cloneAttrs`/`overrideAttr`/`mergeStyleDecl`/`seedInlineStyle`
- `chase/markdown/inlinesvg.go` (163 lines) — `Svg`/`ForeignObject` node kinds, `SvgOptionsKey`/`SvgOptions`/`resolveSvgOptions`, `svgTransformer`, `wrapBaseSvg`
- `chase/markdown/advancedbg.go` (176 lines) — `BackgroundLayer`/`PseudoLayer` node kinds, `wrapAdvancedBackgroundSvg`, `buildAdvancedBackground`, `reducedPercent`, `pseudoColorStyle`

## Files Modified

- `chase/markdown/markdown.go` — registered `svgTransformer` at priority 400 (last, after directive-apply@300)
- `chase/markdown/render.go` — registered + implemented render funcs for `KindSvg`/`KindForeignObject`/`KindBackgroundLayer`/`KindPseudoLayer`; added shared `writeAttrs`/`figureStyle` helpers
- `chase/markdown/background_test.go` — added 9 tests (Test-list cases 1-8, plus the two-part case-1 keyword subtests): `TestBgOptionParseKeywords`, `TestBgOptionParseDimensions`, `TestBgOptionParseFilters`, `TestInlineSvgWrapsPlainSlide`, `TestInlineSvgViewBoxOverride`, `TestNonSvgBackgroundImageDirective`, `TestCorpusMarpBgImageStructural`, `TestCorpusMarpBgSplitStructural`

## Decisions Made

See `key-decisions` in frontmatter for the full list. Summarized:
- `SvgOptionsKey` defaults to disabled, not enabled — an opt-in design that keeps every 01-05/01-06 test passing unchanged while still fully implementing the wrap for callers that request it.
- One `svgTransformer` object handles both the base wrap and the advanced-bg delegation (a single one-line branch calling into `advancedbg.go`), rather than two separately-registered `ASTTransformer`s.
- Filter CSS for the non-SVG path is merged onto the section's style attribute directly in `background.go`, since `apply.go`'s fixed branch sequence is explicitly out of scope for structural edits in this TRD.
- Non-SVG mode applies only the LAST bg-marked image's `backgroundImage`/`backgroundSize`/split/direction, mirroring real Marpit's own "last image wins" precedent.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Build broken by `inlinesvg.go` landing before `background.go` existed**
- **Found during:** Between Task 1 and Task 2 — the coordinator flagged that an uncommitted `chase/markdown/inlinesvg.go` referenced `extractBackgroundImages`/`applyNonSVGBackground`, which did not yet exist (`background.go` had not been written), breaking `go build ./...`/`go test ./chase/...`.
- **Issue:** `inlinesvg.go`'s `svgTransformer.Transform` was written calling into functions intended for a not-yet-created `background.go`, and `markdown.go`/`render.go` had not yet been wired to register the new transformer/node kinds — the package did not compile.
- **Fix:** Wrote `chase/markdown/background.go` (extraction + non-SVG apply + shared Attr helpers), registered `newSvgTransformer()` in `markdown.go` at priority 400, and added `renderSvg`/`renderForeignObject` to `render.go`. Also discovered and fixed a second-order issue: the transformer's default `SvgOptions{Enabled: true, ...}` broke 6 pre-existing 01-05/01-06 tests that assumed no wrap by default — flipped the default to `Enabled: false` (see key-decisions).
- **Files modified:** `chase/markdown/background.go` (created), `chase/markdown/markdown.go`, `chase/markdown/render.go`
- **Verification:** `go build ./...` and `go test ./chase/...` both exit 0 immediately after the fix; re-ran full suite (30/30) before proceeding.
- **Committed in:** `562e9fa` (Task 2 commit — the fix and Task 2's own new work landed together, since the build-breaking file was still uncommitted at the time)

**2. [Rule 1 - Bug] Fabricated `newNodeKind`/`baseLayerNode` helpers in `advancedbg.go`'s first draft**
- **Found during:** Task 3 implementation, first `go build ./...` after writing `advancedbg.go`.
- **Issue:** The initial draft invented `newNodeKind(...)` and an embedded `baseLayerNode` type that do not exist anywhere in this codebase or goldmark — a drafting slip against the established per-file convention (`ast.NewNodeKind(...)` + embedded `ast.BaseBlock` + explicit `Dump` method, as used by every other node kind in `section.go`/`apply.go`/`inlinesvg.go`).
- **Fix:** Replaced with `ast.NewNodeKind("MarpitAdvancedBackgroundLayer"/"MarpitAdvancedPseudoLayer")`, embedded `ast.BaseBlock` in both `BackgroundLayer` and `PseudoLayer`, added the missing `github.com/yuin/goldmark/ast` import, and added `Dump` methods matching the rest of the package.
- **Files modified:** `chase/markdown/advancedbg.go`
- **Verification:** `go build ./...` exit 0 immediately after the fix, before any tests were run against it.
- **Committed in:** `3db97cf` (Task 3 commit — fixed before the file was ever committed in its broken form)

---

**Total deviations:** 2 auto-fixed (1 blocking build-break, 1 bug in a not-yet-committed draft)
**Impact on plan:** Both were caught and fixed before any broken code was committed (Task 2/3 commits both represent already-green states). No scope creep — the `Enabled: false` default and the shared-transformer design are documented above as deliberate, TRD-consistent decisions, not workarounds.

## Issues Encountered

None beyond the two auto-fixed items above — both caught by build/test failure, resolved inline, no blocking issues remain.

## User Setup Required

None — no external service configuration required.

## Next Objective Readiness

- `chase/markdown` now implements the full Marpit parse-side mechanic set: comments/front-matter (01-05/01-06), directive resolution + materialization (01-06), and background-image + inline-SVG advanced-background structure (01-07, PARSE-06/PARSE-07 both complete).
- `SvgOptionsKey`/`SvgOptions` is the ready-made injection point for 01-08's real `chase/theme`-backed width/height resolution (4:3 → 960×720, etc.) — chase/markdown still imports zero `chase/theme`.
- The advanced-background HTML structure is byte-exact against the corpus; 01-04's `pass_advancedbg.go` CSS pipeline and 01-08's cssdiff/htmldiff gate are the remaining pieces that meet this output.
- Heading-slug ids (e.g. `id="over-a-background"`) remain unimplemented, as explicitly scoped out of 01-06 and this TRD (an Objective-3/01-08 concern) — confirmed via direct probe against the real `marp-bg-image` fixture: every substring this TRD owns matches byte-for-byte; only the heading tag itself (id + surrounding whitespace) differs, which is out of scope here.
- No blockers. `go.mod` untouched throughout (`go mod tidy` never run, no new dependencies). Port 8080 never referenced; no server work was needed for this TRD.

---
*Objective: 01-chase-framework*
*Completed: 2026-07-21*

## Self-Check: PASSED
