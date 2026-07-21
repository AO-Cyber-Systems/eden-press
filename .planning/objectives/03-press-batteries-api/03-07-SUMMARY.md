---
objective: 03-press-batteries-api
trd: "07"
subsystem: press-batteries
tags: [goldmark, directive, marker, auto-fit, chase-directive, ast-transformer]

# Dependency graph
requires:
  - objective: 01-chase-framework
    provides: "chase/directive.CoerceGlobal's existing theme/headingDivider/style/lang table, chase/markdown.CommentInline (comment detection), chase/model.isRecognizedDirectiveKey/isNote classification"
  - objective: 03-press-batteries-api
    trd: "01"
    provides: "chase/markdown.NewEngine(extra ...goldmark.Option) extensibility hook, chase/markdown.ParseWithEngine engine-parameterized seam"
provides:
  - "chase/directive.CoerceGlobal recognizes \"size\"/\"math\" as GLOBAL directives (two passthrough cases mirroring style/lang) — comment-form <!-- size: 4:3 --> / <!-- math: mathml --> now classifies as a directive, not a presenter note"
  - "press.autofitOption() goldmark.Option — emits a data-auto-scaling=\"fit\" attribute on a fitting header (# <!--fit-->, marker comment consumed) and wraps fenced-code/$$...$$ math-shaped blocks in <div class=\"marp-fit-shrink\">"
affects: [03-08-sanitize, 03-09-press-render-compose]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "goldmark.Extender wrapper (not a raw goldmark.Option closure) is the correct shape when a single call needs to register BOTH a parser.ASTTransformer and a renderer.NodeRenderer — goldmark.Option is an unexported func(*markdown) type external packages cannot construct directly; goldmark.WithExtensions(Extender) is the one public constructor whose Extend method can touch both Parser().AddOptions and Renderer().AddOptions"
    - "Hand-rolled parent-child recursive walk (capture NextSibling BEFORE mutating, mirroring chase/markdown/headingdivider.go's headingDividerTransformer) instead of ast.Walk, whenever an ASTTransformer re-parents a node — ast.Walk's own walkHelper reads NextSibling AFTER visiting a node's children, so re-parenting mid-traversal silently truncates the ORIGINAL parent's remaining sibling chain"
    - "Wrap-don't-attribute for node kinds whose default HTML renderer ignores node.Attributes() (e.g. *ast.FencedCodeBlock) — goldmark's *ast.Heading renderer DOES write out Attributes() unconditionally (no parser.WithAttribute() dependency), but FencedCodeBlock's renderer never checks it at all, so a synthetic wrapper block is the sanitize-survivable, renderer-non-invasive way to carry a class around it"

key-files:
  created:
    - press/autofit.go
    - press/autofit_test.go
  modified:
    - chase/directive/directives.go
    - chase/directive/directive_test.go
    - chase/model/build_test.go

key-decisions:
  - "size/math passthrough added ONLY to CoerceGlobal (never CoerceLocal) — they are Marp-Core-layer GLOBAL directives (customDirectives.global), not Marpit local directives; front-matter-form size/math already reached Output.Meta today via buildMeta's unconditional materialization, so this TRD's entire CORE-02 scope is the two comment-form classification cases, nothing in chase/model or chase/theme."
  - "CORE-09 marker baseline: data-auto-scaling=\"fit\" attribute (heading) + marp-fit-shrink wrapper class (code/math) — chosen as the sanitize-survivable shape (an attribute/class on a real element, never a bare comment), per the TRD's own 03-08 co-design note. No math AST node/extension exists yet (CORE-08 is a separate, not-yet-built battery), so math-block detection uses a source-level heuristic: a Paragraph with exactly one Text child whose raw segment both starts and ends with the literal \"$$\" delimiter."

patterns-established:
  - "goldmark.Extender-wrapper shape for any future press/ battery that needs both a parser.ASTTransformer and a renderer.NodeRenderer registered from one goldmark.Option-returning function."

requirements-completed: [CORE-02, CORE-09]

# Verification evidence
verification:
  gates_defined: 2
  gates_passed: 2
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~25min
completed: 2026-07-21
---

# Objective 3 TRD 07: size/math Global Directives + Auto-Fit Markers Summary

**CoerceGlobal gains two passthrough cases (size/math) closing the comment-form classification gap for CORE-02; press.autofitOption() emits the CORE-09 fit/shrink MARKERS (data-auto-scaling="fit" on a fitting header, marp-fit-shrink wrapper on code/math blocks) for the already-vendored themes/browser-fit.js viewer helper — no runtime JS, @auto-scaling stays theme-CSS-only.**

## Performance

- **Duration:** ~25 min (session start -> Task 2 commit)
- **Completed:** 2026-07-21T14:09:12Z
- **Tasks:** 2/2 complete
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `chase/directive.CoerceGlobal` recognizes `"size"` and `"math"` as GLOBAL directives (two cases mirroring the existing `style`/`lang` passthrough: raw value returned unchanged, `isKnown=true`). `chase/model.isRecognizedDirectiveKey` (which calls `CoerceGlobal` to classify a detected HTML comment as directive-vs-note) picks the new cases up automatically — no `chase/model` or `chase/theme` change was needed.
- Verified via a new `chase/model` test (`TestBuildCommentFormSizeMathNotNotes`) that comment-form `<!-- size: 4:3 -->` / `<!-- math: mathml -->` no longer leak into `Section.Notes`, while a genuine free-form note in the same slide still does.
- `press/autofit.go`: `autofitOption() goldmark.Option`, a `goldmark.Extender` wrapper registering (a) a priority-500 `parser.ASTTransformer` that detects a fitting header (`# <!--fit-->`) — strips the `*markdown.CommentInline` marker and sets `data-auto-scaling="fit"` directly on the `<hN>` element — and wraps fenced-code / `$$...$$` math-shaped blocks in a synthetic node; and (b) a priority-0 `renderer.NodeRenderer` rendering that synthetic node as `<div class="marp-fit-shrink">...</div>`.
- Confirmed empirically (scratch AST dump) that goldmark's default `*ast.Heading` renderer writes `node.Attributes()` unconditionally (no `parser.WithAttribute()` dependency), but `*ast.FencedCodeBlock`'s default renderer ignores `Attributes()` entirely — driving the wrap-vs-attribute split documented above.
- All markers are MARKERS ONLY — no runtime JS is emitted anywhere in this file; `@auto-scaling` (theme front-matter) is untouched, already owned by `chase/theme/meta.go` (THEME-02).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: CoerceGlobal size/math (CORE-02) | `go test ./chase/directive/... ./chase/model/... -v && gofmt -l chase/directive/directives.go` | 0 | PASS |
| 2: autofitOption marker emitter (CORE-09) | `go test ./press/ -run 'TestAutofit\|TestFitMarker\|TestShrinkMarker' -v && go vet ./press/... && gofmt -l press/autofit.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: size/math GLOBAL directives (CORE-02)** - `d5108c1` (feat)
2. **Task 2: auto-fit + shrink marker emitter (CORE-09)** - `ff37ef9` (feat)

_Both tasks are `tdd="true"`; RED confirmed before each GREEN implementation — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build (whole repo) | `go build ./...` | 0 | PASS |
| vet (whole repo) | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS |
| gofmt | `gofmt -l .` | 0 (no output) | PASS |
| Obj-1 corpus/cssdiff | `go test ./conformance/...` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS |
| no-chromedp invariant | `go list -deps ./press/... \| grep -c chromedp` | 0 (count) | PASS |
| license headers (touched files) | `addlicense -l mit -s -c "AO Cyber Systems" -check <5 touched files>` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1, chase/directive) | `go test ./chase/directive/... -run 'TestCoerceGlobalSizeMathRecognized\|...' -v` | 1 | FAIL — `size`/`math` returned `(nil, false)` (correct) |
| RED (Task 1, chase/model) | `go test ./chase/model/... -run TestBuildCommentFormSizeMathNotNotes -v` | 1 | FAIL — `Notes = [size: 4:3, math: mathml, just a note]`, 3 not 1 (correct) |
| GREEN (Task 1) | `go test ./chase/directive/... ./chase/model/... -v` | 0 | PASS, all cases (correct) |
| RED (Task 2) | `go test ./press/ -run 'TestAutofit' -v` | build failure (`undefined: autofitOption`) | FAIL (correct) |
| GREEN (Task 2) | `go test ./press/ -run 'TestAutofit' -v` | 0 | PASS, all 4 cases (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 — both RED->GREEN cycles reached GREEN on the first implementation attempt; no bugs required a Rule 1-3 auto-fix.
- **Must-haves verified:** 3/3 (all `must_haves.truths` from `03-07-TRD.md` frontmatter — CoerceGlobal recognition, marker emission without runtime JS, markers emitted pre-sanitize for 03-08 co-design).
- **Gate failures:** None.

## Files Created/Modified

- `chase/directive/directives.go` - two new `CoerceGlobal` cases: `"size"`, `"math"` (passthrough, mirroring `style`/`lang`)
- `chase/directive/directive_test.go` - `TestCoerceGlobalSizeMathRecognized`, `TestCoerceGlobalStyleLangPassthroughAndUnknownKeyRegression`
- `chase/model/build_test.go` - `TestBuildCommentFormSizeMathNotNotes` (comment-form size/math no longer leak into Notes)
- `press/autofit.go` - `autofitOption()`, `autofitExtender`, `autofitTransformer`/`autofitWalk`, `applyFitMarker`, `isMathBlockParagraph`, `wrapWithShrinkMarker`, `autofitShrinkNode`, `autofitNodeRenderer`
- `press/autofit_test.go` - `TestAutofitFitHeaderMarker`, `TestAutofitNormalHeadingCarriesNoFitMarker`, `TestAutofitShrinkMarkersOnCodeAndMathBlocks`, `TestAutofitOptionIsComposableGoldmarkOption`

## Decisions Made

- `size`/`math` were added ONLY to `CoerceGlobal`, never `CoerceLocal` — they are Marp-Core-layer GLOBAL directives, not Marpit local ones (FEATURES.md). No `chase/theme`/`chase/model` change was made beyond the automatic pickup through `isRecognizedDirectiveKey`.
- `autofitOption()` is implemented as a `goldmark.Extender` wrapped by `goldmark.WithExtensions(...)`, not a hand-written `goldmark.Option` closure — `goldmark.Option` is `func(*markdown)` where `markdown` is an unexported concrete type, so an external package cannot construct one directly when it needs to register both parser- and renderer-side options from a single call. This mirrors `chase/markdown.New()`'s own `Extend` shape.
- The auto-fit `ASTTransformer` is registered at priority 500 — strictly after chase/markdown's own baked-in transformers (100/200/300/400) — so it always sees the final Section/ForeignObject-wrapped tree shape and never needs to special-case where in that structure a Heading/FencedCodeBlock ends up.
- Mutating traversal is a hand-rolled parent/child recursive walk (`autofitWalk`), not `ast.Walk`: `ast.Walk`'s own `walkHelper` reads `NextSibling()` only after visiting a node's children, so re-parenting a node (the shrink-wrap) while `ast.Walk` is mid-traversal of that node's ORIGINAL parent would silently truncate the parent's remaining sibling chain. `autofitWalk` captures `next` before any mutation, exactly like `headingDividerTransformer` already does for synthetic `ThematicBreak` insertion.
- Fenced-code/math blocks are WRAPPED in a synthetic node rather than given a `class` attribute directly, because goldmark's default `*ast.FencedCodeBlock` renderer never consults `node.Attributes()` at all (confirmed by reading `renderer/html/html.go`) — unlike `*ast.Heading`'s renderer, which does. Wrapping sidesteps re-implementing `<pre><code>` rendering just to add one class.
- Math-block detection is a source-level heuristic (`$$...$$`-delimited single-Text paragraph), since CORE-08 (actual math parsing/rendering) is a separate, not-yet-built battery — there is no math AST node to match against yet. This is documented as an explicit BASELINE in `press/autofit.go`'s doc comment; CORE-08 will introduce a real math AST node, which this heuristic can then be swapped for.

## Deviations from Plan

None — TRD executed exactly as written. The exact marker/wrap shape (attribute vs. wrapper-class) was chosen within the TRD's own explicitly-granted baseline latitude ("pick the simplest browser-fit.js-readable attribute/class... do not over-engineer"); see Decisions Made above for the reasoning.

## Issues Encountered

None. Both TDD cycles (RED confirmed, then GREEN on first implementation) passed without needing a second attempt.

## User Setup Required

None — no external service configuration required.

## Next Objective Readiness

- CORE-02 and CORE-09 are both complete; `chase/directive.CoerceGlobal`'s size/math cases and `press.autofitOption()` are ready for `03-09`'s `press.Render` to fold `autofitOption()` directly into its own `markdown.NewEngine(pressExtraOpts...)` call.
- The `data-auto-scaling="fit"` attribute and `marp-fit-shrink` class are documented as the exact markers `03-08`'s sanitize allow-list must preserve — flagged explicitly in `press/autofit.go`'s doc comments for that TRD's co-design.
- `themes/browser-fit.js` (already vendored by 03-02) is the viewer-side consumer of both markers; no further chase/markdown or chase/theme change is required for either battery.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: chase/directive/directives.go
- FOUND: chase/directive/directive_test.go
- FOUND: chase/model/build_test.go
- FOUND: press/autofit.go
- FOUND: press/autofit_test.go
- FOUND: .planning/objectives/03-press-batteries-api/03-07-SUMMARY.md
- FOUND commit: d5108c1 (Task 1)
- FOUND commit: ff37ef9 (Task 2)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
