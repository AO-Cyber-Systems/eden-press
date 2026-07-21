---
objective: 01-chase-framework
trd: "05"
subsystem: parsing
tags: [goldmark, marpit, ast-transformer, node-renderer, two-phase-seam, go]

# Dependency graph
requires: ["01-02"]
provides:
  - "chase/markdown package: goldmark Extender (New()/Extend()) wiring comment detection, slide-split, headingDivider, and .marpit/<section> render-time wrapping (PARSE-05)"
  - "chase/markdown.Section/CommentNode/CommentInline ast.Node kinds -- the AST shape TRD 01-06 (directive recognition) and objective 2 (docmodel) build on"
  - "Proven two-phase Parse()/Render() seam: the finalized, slide-split AST is inspectable between phases, before any HTML is emitted"
affects: [01-06-chase-markdown]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-phase seam only: every call site uses md.Parser().Parse() then, in a separate step, md.Renderer().Render() -- never md.Convert() -- so the finalized AST is inspectable/mutable between phases"
    - "ASTTransformer ordering as the mechanism for headingDivider + slide-split cooperation: headingDivider (priority 100) inserts synthetic *ast.ThematicBreak nodes; slide-split (priority 200) then consumes ALL breaks (synthetic + authored) uniformly with zero special-case code"
    - "setext-H2 trap resolved for free by goldmark's own block-parser priority order (SetextHeadingParser=100 before ThematicBreakParser=200) -- no trap-detection code needed in slide.go"
    - "Render-time (not parse-time) container injection: .marpit wraps output only inside the ast.KindDocument NodeRenderer override; no container node exists in the AST"
    - "Detection vs recognition split preserved into chase/markdown: comment Block/InlineParser only extract raw KV via chase/directive.DetectComment/ParseComment; deciding which keys are real directives is deferred entirely to TRD 01-06"
    - "Priority-0 registration wins the parser trigger race ('<!--' vs goldmark's HTMLBlockParser@900/RawHTMLParser@400) and the renderer override race (vs html.NewRenderer()@1000)"

key-files:
  created:
    - chase/markdown/section.go
    - chase/markdown/comment.go
    - chase/markdown/markdown.go
    - chase/markdown/headingdivider.go
    - chase/markdown/slide.go
    - chase/markdown/render.go
    - chase/markdown/markdown_test.go
  modified: []

key-decisions:
  - "headingDivider value is injected via a parser.Context key (HeadingDividerKey), not a New() constructor Option, mirroring the two-phase seam's own parser.WithContext(pc) pattern and letting the value vary per-parse (needed once TRD 01-06 resolves it per-document from front-matter/carry-forward)"
  - "slide-split's child-reparenting (doc.RemoveChildren(doc) then AppendChild under new *Section parents) verified safe by reading goldmark's ensureIsolated: it only detaches a node from a non-nil Parent(), which RemoveChildren has already nil-ed for every removed child"
  - "headingDivider never inserts a break before the document's very first child, even if it qualifies -- verified against the marp-heading-divider corpus fixture (3 expected sections, not 4)"
  - "CommentNode/CommentInline render as ast.WalkSkipChildren + no output -- detection (this TRD) and recognition (TRD 01-06) are cleanly separated, but a detected comment must never leak into HTML regardless of which TRD lands first"

patterns-established:
  - "Pattern: ASTTransformer priority ordering as an explicit contract between transformers (documented priority numbers + comments explaining why lower runs first), reusable for TRD 01-06/07/08's additional transformers"

requirements-completed: [PARSE-05]

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 35min
completed: 2026-07-21
---

# Objective 1 TRD 05: Two-Phase Render Seam + Slide-Splitting + Container Wrapping Summary

**`chase/markdown`'s goldmark Extender: the two-phase Parser().Parse()/Renderer().Render() seam, `---`/headingDivider slide-splitting (setext-H2-safe), HTML-comment detection-only, and render-time `.marpit`/`<section>` container wrapping.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 3
- **Files created:** 7 (all new)

## Accomplishments

- `chase/markdown/section.go`: `Section` (ordered `Attrs []Attr` extension point for TRD 01-06), `CommentNode`/`CommentInline` -- detection-only AST node kinds, `Hidden: true` always set.
- `chase/markdown/comment.go`: a `BlockParser`/`InlineParser` pair triggering on `<`, wrapping `chase/directive.DetectComment`/`ParseComment` into `CommentNode`/`CommentInline`; registered at priority 0 to win the trigger race against goldmark's own `HTMLBlockParser`(900)/`RawHTMLParser`(400).
- `chase/markdown/headingdivider.go`: `HeadingDividerKey` (parser.Context key) + an `ASTTransformer` (priority 100) inserting synthetic `*ast.ThematicBreak` nodes before qualifying headings -- skipping the doc's first child and avoiding double-breaks.
- `chase/markdown/slide.go`: `SlideMetaKey`/`SlideMeta` + the slide-split `ASTTransformer` (priority 200, after headingDivider): splits on every `*ast.ThematicBreak`, wraps runs in `*Section` nodes (`id = index+1`), requires zero setext-H2 special-casing (goldmark's own `SetextHeadingParser`(100) already consumes `Title\n---` into an `*ast.Heading` before either transformer runs).
- `chase/markdown/render.go`: a `renderer.NodeRenderer` (priority 0, overriding `html.NewRenderer()`@1000) injecting `<div class="marpit">...</div>` around `ast.KindDocument` at RENDER time, `<section id="N">` for `*Section`, and skip-with-no-output for `CommentNode`/`CommentInline`.
- `chase/markdown/markdown.go`: `New()`/`Extend()` wiring all of the above via the goldmark-meta `Extend(m goldmark.Markdown)` pattern.
- Proved the two-phase seam is real: `TestTwoPhaseSeamSectionsExistBeforeRender` and `TestTwoPhaseSeamRendersMarpitContainerAndSections` inspect the finalized, already-slide-split AST between an explicit `md.Parser().Parse()` call and a separate `md.Renderer().Render()` call -- no `md.Convert()` used anywhere in `chase/markdown` or its tests.
- Verified package boundary: `go list -deps ./chase/markdown | grep -c 'chase/theme'` → `0`.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Comment detection (block+inline) + AST node kinds | `go test ./chase/markdown/ -run 'Comment'` | 0 | PASS |
| 2: Slide-split + headingDivider ASTTransformers (setext-H2 safe) | `go test ./chase/markdown/ -run 'Slide\|Heading\|Setext\|TwoPhaseSeamSections'` | 0 | PASS |
| 3: Render-time `.marpit` container + `<section>`/comment NodeRenderer | `go test ./chase/markdown/` (9/9 tests) | 0 | PASS |

## Task Commits

Each task was executed test-first with a RED (failing test) commit followed by a GREEN (implementation) commit:

1. **Task 1: Comment detection + AST node kinds**
   - `1e217fb` test(01-05): add failing comment-detection tests for two-phase seam
   - `b5bda9a` feat(01-05): two-phase seam comment detection (block+inline) + AST node kinds
2. **Task 2: Slide-split + headingDivider ASTTransformers**
   - `3fa5324` test(01-05): add failing slide-split, setext-H2 trap, and headingDivider tests
   - `93381c6` feat(01-05): slide-split + headingDivider ASTTransformers (setext-H2 safe)
3. **Task 3: Render-time `.marpit` container + NodeRenderer**
   - `d876a40` test(01-05): add failing two-phase render + comment-hiding tests
   - `816c4e9` feat(01-05): render-time .marpit container + section/comment NodeRenderer

_No REFACTOR commits were needed -- each GREEN implementation was already the intended final shape._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `gofmt -l chase/markdown/*.go \| (! grep .) && go vet ./chase/markdown/...` | 0 | PASS |
| test | `go test ./chase/markdown/...` | 0 | PASS (9/9) |
| build | `go build ./...` | 0 | PASS |
| license | `addlicense -check chase/markdown/*.go` | 0 | PASS (all 7 files) |

## TDD Evidence

| Phase | Task | Command | Exit Code | Expected |
|---|---|---|---|---|
| RED | 1 | `go test ./chase/markdown/...` (markdown.go absent) | 1 | FAIL (correct -- `undefined: New`) |
| GREEN | 1 | `go test ./chase/markdown/... -v` | 0 | PASS (correct -- 2/2) |
| RED | 2 | `go test ./chase/markdown/...` (slide.go/headingdivider.go absent) | 1 | FAIL (correct -- `undefined: newSlideSplitTransformer`, `undefined: newHeadingDividerTransformer`, `undefined: HeadingDividerKey`) |
| GREEN | 2 | `go test ./chase/markdown/... -v` | 0 | PASS (correct -- 7/7) |
| RED | 3 | `go test ./chase/markdown/...` (render.go absent, renderer not registered) | 2 (panic) | FAIL (correct -- default renderer panics: no NodeRendererFunc registered for the custom `KindSection` kind, proving render.go is required) |
| GREEN | 3 | `go test ./chase/markdown/... -v` | 0 | PASS (correct -- 9/9 total) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 5/5 truths, 7/7 artifacts
  - "A deck splits into slides on `---` ... `<section id>` inside `.marpit`" → `TestSlideSplitBasic`, `TestTwoPhaseSeamRendersMarpitContainerAndSections`
  - "setext-H2 trap resolved correctly" → `TestSlideSplitSetextH2TrapIsNotABreak`
  - "headingDivider inserts synthetic breaks, slide-splitter consumes uniformly" → `TestHeadingDividerInsertsSyntheticBreaks`
  - "HTML comments detected as hidden nodes, no directive recognition" → `TestCommentBlockDetection`, `TestCommentInlineDetection`, `TestCommentNeverLeaksIntoRenderedOutput`
  - "container injected at RENDER time, not parse time" → `render.go`'s `ast.KindDocument` override (no container AST node anywhere in section.go/slide.go)
- **Gate failures:** None
- **Package boundary check:** `go list -deps ./chase/markdown | grep -c 'chase/theme'` → `0` (confirmed -- no `chase/theme` import at parse time)
- **addlicense:** `addlicense -check chase/markdown/*.go` → clean (all 7 files carry the Eden MIT header)
- **Two-phase seam (never `md.Convert()`):** confirmed by inspection -- `chase/markdown/markdown_test.go` calls `md.Parser().Parse(...)` and `md.Renderer().Render(...)` as two separate statements in every test; `md.Convert()` does not appear anywhere in the package or its tests.

## Files Created

- `chase/markdown/section.go` (164 lines) -- `Section`/`Attr`, `CommentNode`, `CommentInline` custom `ast.Node` kinds
- `chase/markdown/comment.go` (~130 lines) -- comment `BlockParser`/`InlineParser` wrapping `chase/directive.DetectComment`/`ParseComment`
- `chase/markdown/markdown.go` (~90 lines) -- `New()`/`Extend()` goldmark Extender wiring parsers, transformers, and the node renderer
- `chase/markdown/headingdivider.go` (~115 lines) -- `HeadingDividerKey` + headingDivider `ASTTransformer` (priority 100)
- `chase/markdown/slide.go` (~90 lines) -- `SlideMetaKey`/`SlideMeta` + slide-split `ASTTransformer` (priority 200)
- `chase/markdown/render.go` (~95 lines) -- `NodeRenderer` for `.marpit` container, `<section>`, and hidden comments
- `chase/markdown/markdown_test.go` (~270 lines) -- 9 tests covering comment detection, slide-split, setext-H2 trap, headingDivider, the two-phase seam (pre- and post-render), and comment-hiding

## Decisions Made

- **`HeadingDividerKey` as a `parser.Context` key, not a constructor `Option`:** mirrors the seam's own `parser.WithContext(pc)` injection and lets the resolved `[]int` value vary per-parse -- required once TRD 01-06 starts resolving it per-document from front-matter/carry-forward instead of a fixed test-injected value.
- **`ensureIsolated` verified safe for slide-split's reparenting approach:** read goldmark's `ast.go` source directly; `ensureIsolated(v)` only detaches `v` from a non-nil `Parent()`, and `RemoveChildren` already nils every removed child's `Parent`/`PreviousSibling`/`NextSibling` -- so `doc.RemoveChildren(doc)` followed by `AppendChild`-ing each original child under new `*Section` parents is safe with no risk of a panic or silent corruption.
- **No setext-H2 special-case code in slide.go:** confirmed via a dedicated test (`TestSlideSplitSetextH2TrapIsNotABreak`) that goldmark's own `SetextHeadingParser`(priority 100)/`ThematicBreakParser`(priority 200) ordering already consumes `Title\n---` into an `*ast.Heading` before any `ASTTransformer` (both registered at priority ≥100) ever runs -- every `*ast.ThematicBreak` a transformer sees is guaranteed to be a real separator.
- **Comment rendering hard-hides regardless of recognition status:** `render.go`'s `renderComment` returns `ast.WalkSkipChildren` and writes nothing for both `CommentNode` and `CommentInline`, independent of whether TRD 01-06 has yet decided a given key is a real directive -- detection and recognition are cleanly separated, but "never leak into HTML" is guaranteed starting now.

## Deviations from Plan

None -- TRD executed exactly as written. The `RemoveChildren`/`ensureIsolated` safety check and the `HeadingDividerKey` context-vs-option design were both resolutions of open implementation questions already anticipated by 01-RESEARCH.md, not scope changes; no Rule-4 architectural stop was needed, and `go.mod` was not touched (`go mod tidy` never run).

## Issues Encountered

One inherited compile bug was fixed inline during Task 1 verification (Rule 1 -- auto-fix bug, not a TRD deviation): `comment.go`'s `Close()` method called `lines.At(i).Value(reader.Source())` directly on a non-addressable `text.Segment` return value, which does not compile against `text.Segment.Value`'s pointer receiver. Fixed by assigning to an intermediate `seg := lines.At(i)` variable before calling `.Value(...)`. No test changes were needed; caught by `go test` failing to build before any RED/GREEN evidence was captured.

## User Setup Required

None -- no external service configuration required.

## Next Objective Readiness

- `chase/markdown` is a complete, working goldmark Extender: comment detection, slide-split (setext-H2-safe), headingDivider, and `.marpit`/`<section>` render-time wrapping are all in place and covered by the two-phase seam.
- `Section.Attrs` (ordered `[]Attr`) is the ready-made extension point for TRD 01-06 to populate with directive-derived HTML/style attributes.
- `CommentNode`/`CommentInline`'s raw `KV`/`Raw` fields are ready for TRD 01-06's directive-recognition `ASTTransformer` to consume via `chase/directive`'s carry-forward machine.
- No blockers. `chase/theme` remains independently buildable with zero cross-import, as designed.

---
*Objective: 01-chase-framework*
*Completed: 2026-07-21*

## Self-Check: PASSED

All 7 created files confirmed present on disk (`chase/markdown/{section,comment,markdown,headingdivider,slide,render,markdown_test}.go`). All 6 referenced task commit hashes (`1e217fb`, `b5bda9a`, `3fa5324`, `93381c6`, `d876a40`, `816c4e9`) confirmed present via `git log --oneline --all`.
