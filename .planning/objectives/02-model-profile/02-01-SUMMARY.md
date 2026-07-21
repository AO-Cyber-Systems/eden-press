---
objective: 02-model-profile
job: "01"
subsystem: docmodel
tags: [goldmark, ast-walk, json-schema, marpit, docmodel]

# Dependency graph
requires:
  - objective: 01-chase-framework
    provides: "chase/markdown's two-phase Parse()/Render() seam (seam.go), *markdown.Section/CommentNode/CommentInline AST nodes, chase/directive's CoerceGlobal/CoerceLocal/SpotKey recognition functions"
provides:
  - "chase/model package: Document{SchemaVersion,Meta,Sections,Outline} JSON-serializable schema"
  - "Build(doc, source, pc) *Document — single-pass, read-only ast.Walk deriving the docmodel from the SAME finalized AST chase/markdown renders HTML from"
  - "Empirically-verified contract: chase/markdown always yields >=1 *markdown.Section, even for empty source (slideSplitTransformer's unconditional single-run append) — Sections are never nil for a real parsed doc"
affects: [02-02-profile, 02-03-grep-gate, 02-04-capstone-single-parse]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Full-tree ast.Walk (not shallow FirstChild/NextSibling scan) to find *markdown.Section nodes, because inlinesvg.go's svgTransformer (priority 400) always wraps every Section in <Svg><ForeignObject> when seam.go's Parse unconditionally enables SvgOptionsKey"
    - "Index-based (not pointer-based) tracking of the 'current Section' during a single-pass Walk, to avoid slice-reallocation invalidating a previously taken &d.Sections[i] pointer"
    - "Re-derive directive-comment KV from CommentNode/CommentInline.Raw via chase/directive.ParseComment (never trust the lossy .KV map[string]string) — mirrors chase/markdown/directive.go's own buildEventStream pattern"
    - "Deterministic structural-signature walk (node Kind + mutation-relevant fields, in fixed code order) as the non-mutation proof mechanism, instead of ast.Node.Dump text comparison — Dump's own DumpHelper renders extra fields from a Go map, which has randomized iteration order and produces flaky false-positive 'mutations' even with zero actual AST changes"

key-files:
  created:
    - chase/model/document.go
    - chase/model/document_test.go
    - chase/model/build.go
    - chase/model/build_test.go
  modified: []

key-decisions:
  - "Empty document (source=\"\") produces Document.Sections = [{ID:1}] (one empty Section), NOT nil — verified empirically against chase/markdown's actual AST output, which always contains exactly one *markdown.Section even for zero-length input. The TRD's error_recovery text assumed 'no Section children' for an empty deck; that premise does not hold against the real, already-shipped chase/markdown dependency. Faithfully mirroring the real AST (MODEL-01's core mandate) takes priority over the TRD's incorrect illustrative assumption. Document.Outline remains nil (no headings)."
  - "A block-level *markdown.CommentNode always carries a nested *markdown.CommentInline child with an identical Raw value (chase/markdown's inline-parsing pass re-detects the same comment span inside the block's own content). Build skips a CommentInline whose immediate parent is its own CommentNode, to avoid double-counting the same note/directive text."

patterns-established:
  - "Non-mutation proof for docmodel builders: render HTML from the parsed doc, run Build, render HTML again, assert byte-identical — plus an independent deterministic structural-signature walk (not stdlib/third-party Dump text) as a second check."

requirements-completed: [MODEL-01]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 19min
completed: 2026-07-21
---

# Objective 2 TRD 01: chase/model Document/Build Summary

**chase/model package: a versioned, JSON-serializable Document{Meta,Sections,Outline} built by a single read-only ast.Walk over chase/markdown's own finalized post-transform AST — proven non-mutating via byte-identical HTML render before/after Build.**

## Performance

- **Duration:** 19 min (TRD assignment 01:16:11 -> Task 2 commit 01:35:10, local time; both commits below)
- **Started:** 2026-07-21T05:16:11Z
- **Completed:** 2026-07-21T05:35:53Z
- **Tasks:** 2/2 complete
- **Files modified:** 4 (all newly created)

## Accomplishments
- `Document`/`Section`/`Meta`/`OutlineEntry` schema: stable `SchemaVersion = "eden-press.model/v1"`, full JSON round-trip.
- `Build(doc ast.Node, source []byte, pc parser.Context) *Document`: single-pass `ast.Walk` deriving Sections (ID + directive attrs), a flat document-ordered Outline (heading level/text/AutoHeadingID slug grouped by owning Section), and per-Section speaker Notes (non-directive comments only) — all from the SAME finalized AST `chase/markdown.Parse` hands to the HTML renderer.
- MODEL-01 single-source proof: `TestBuildNonMutation` renders the parsed doc to HTML, runs `Build`, renders again -> byte-identical HTML; a deterministic structural signature of the tree (independent of `Build`'s own logic) is also compared before/after.
- Discovered and worked around a real structural fact about chase/markdown's finalized AST: `*markdown.Section` nodes are never direct children of `*ast.Document` (they're wrapped 2 levels deep in `<Svg><ForeignObject>` by `inlinesvg.go`'s always-on transformer) — `Build` uses a full-tree walk, not the TRD sketch's shallow `FirstChild`/`NextSibling` scan, so it actually finds Sections against the real, current AST shape.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Document/Section/Meta/Outline schema | `go test ./chase/model/ -run TestDocument -v` | 0 | PASS |
| 2: Build — read-only walk of finalized AST | `go test ./chase/model/... -v && go vet ./chase/model/...` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Document/Section/Meta/Outline schema** - `b9f1ec2` (test)
2. **Task 2: Build — read-only walk of finalized AST** - `1f1d844` (feat)

_Note: both tasks are `tdd="true"`; RED (compile failure against undefined types/`Build`) confirmed before each GREEN implementation — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./chase/model/...` (and `go build ./...`) | 0 | PASS |
| vet | `go vet ./chase/model/...` (and `go vet ./chase/...`) | 0 | PASS |
| test | `go test ./chase/model/...` (and `go test ./chase/...`) | 0 | PASS |
| gofmt | `gofmt -l chase/model/` | 0 (no output) | PASS |
| grep gate | `grep -rnE '\bSlide\b\|"section"\|16:9\|1280\|720' chase/model --include='*.go' \| grep -v _test.go` | (empty output) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./chase/model/ -run TestDocument -v` | 1 (compile failure: undefined SchemaVersion/Document/Meta/Section/OutlineEntry) | FAIL (correct) |
| GREEN (Task 1) | `go test ./chase/model/ -run TestDocument -v` | 0 | PASS (correct) |
| RED (Task 2) | `go test ./chase/model/... -v` | 1 (compile failure: undefined Build, 6 call sites) | FAIL (correct) |
| GREEN (Task 2) | `go test ./chase/model/... -v` | 0 (all 8 tests, after 2 auto-fix iterations — see Deviations) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (bundled 3 related fixes discovered during Task 2's own RED->GREEN loop, before the task commit — see Deviations)
- **Must-haves verified:** 6/6 (all `must_haves.truths` from 02-01-TRD.md frontmatter)
- **Gate failures:** None remaining (all fixed within Task 2's TDD cycle, before commit)

## Files Created/Modified
- `chase/model/document.go` - `SchemaVersion` constant, `Document`/`Section`/`Meta`/`OutlineEntry` JSON-tagged types
- `chase/model/document_test.go` - SchemaVersion + JSON round-trip tests (Test-list case 5)
- `chase/model/build.go` - `Build()` + `attrsToMap`/`headingSlug`/`buildMeta`/`stringifyRawValue`/`isNote`/`isRecognizedDirectiveKey` helpers
- `chase/model/build_test.go` - Test-list cases 1,2,3,4,6,7 + `structuralSignature` non-mutation helper

## Decisions Made
- Empty-document Sections is `[{ID:1}]`, not `nil` (see key-decisions above — matches the real, verified chase/markdown AST shape; documented as a deviation below).
- `Build` uses index-based Section tracking (`sectionIdx int`, not `current *Section`) during the walk, since `append` can reallocate `d.Sections`' backing array and silently invalidate a previously taken pointer.
- Non-directive-comment detection re-derives KV from `Raw` via `chase/directive.ParseComment` rather than trusting `CommentNode.KV`/`CommentInline.KV` (documented as lossy, unordered, non-re-parseable for array values) — mirrors the identical pattern `chase/markdown/directive.go`'s `collectSectionCommentEvents` already uses.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TRD code sketch's shallow FirstChild/NextSibling scan would find zero Sections against the real AST**
- **Found during:** Task 2, initial design/RED phase
- **Issue:** The TRD's embedded `codebase_examples` Build sketch walks `doc.FirstChild()`/`NextSibling()` assuming `*markdown.Section` nodes are direct children of `*ast.Document`. Empirically confirmed this is false: `chase/markdown/inlinesvg.go`'s `svgTransformer` (priority 400, always enabled by `seam.go`'s `Parse`) wraps every Section in `<Svg><ForeignObject>`, nesting it two levels deep.
- **Fix:** `Build` performs a full `ast.Walk` over the entire document, tracking an index into `d.Sections` for the Section currently being walked (set/cleared on Section enter/exit), rather than a shallow top-level scan.
- **Files modified:** chase/model/build.go
- **Verification:** `TestBuildThreeSlideDeck` passes (3 Sections found and correctly ID'd despite the Svg/ForeignObject wrapping).
- **Committed in:** 1f1d844 (Task 2 commit)

**2. [Rule 1 - Bug] Duplicate Notes entries from a block comment's own nested inline shadow node**
- **Found during:** Task 2, TDD GREEN phase (`TestBuildNotesVsDirectiveComment` initially failed: `Notes = [x x]`, duplicated)
- **Issue:** A block-level `<!-- ... -->` comment produces a `*markdown.CommentNode`, which ALSO contains a nested `*markdown.CommentInline` child carrying the identical `Raw` text (chase/markdown's inline-parsing pass re-detects the same comment span as an inline node inside the block's own content). Walking both unconditionally double-counted every plain note.
- **Fix:** `Build`'s `*markdown.CommentInline` case now skips a `CommentInline` whose immediate parent is a `*markdown.CommentNode` (the shadow copy already counted via the parent case); a genuine mid-paragraph inline comment (parented by e.g. a Paragraph) is still processed.
- **Files modified:** chase/model/build.go
- **Verification:** `TestBuildNotesVsDirectiveComment` passes (exactly one Notes entry).
- **Committed in:** 1f1d844 (Task 2 commit)

**3. [Rule 1 - Bug] Flaky non-mutation check via ast.Node.Dump text comparison**
- **Found during:** Task 2, TDD GREEN phase (`TestBuildNonMutation` initially failed on a spurious "AST dump changed" diff that was actually just reordered `Width`/`Height`/`X` fields)
- **Issue:** The originally-planned non-mutation proof captured `ast.Node.Dump`'s stdout output before/after `Build` and diffed the text. `ast.DumpHelper` (goldmark, used by `MarpitForeignObject.Dump` etc.) renders each node's "extra fields" from a `Go map[string]string`, whose iteration order is randomized per-process — empirically confirmed via a scratch script that calling `Dump` twice in a row on the exact same, never-mutated doc sometimes produces two DIFFERENT strings. This would make a Dump-text-diff a flaky false-positive detector unrelated to anything `Build` does.
- **Fix:** Replaced the Dump-text-diff with `structuralSignature(doc)`, a custom deterministic walk that appends each node's `Kind()` plus a small set of mutation-relevant fields (`Section.ID`/`Attrs`, `Heading.Level`, `Comment(Node|Inline).Raw`) in a fixed, literal code order — never a map range. Kept as a second, independent check alongside the primary (and TRD-mandated) proof: rendering the SAME parsed doc to HTML before and after `Build`, asserting byte-identical output.
- **Files modified:** chase/model/build_test.go
- **Verification:** `TestBuildNonMutation` passes deterministically (re-run 3x with no flakes); the HTML-render byte-identical assertion is the primary MODEL-01 proof and was unaffected by this fix.
- **Committed in:** 1f1d844 (Task 2 commit)

**4. [Rule 1 - Bug] TRD's error_recovery text ("empty deck -> nil Sections") does not match chase/markdown's real, verified AST output**
- **Found during:** Task 2, Test-list case 7 design
- **Issue:** 02-01-TRD.md's `<error_recovery>` states: "Empty deck (no `*Section` children): return a `*Document` with `Sections: nil`... never nil the whole Document." Empirically re-verified (twice, via disposable scratch Go programs) that `chase/markdown.Parse("")` produces exactly ONE `*markdown.Section{ID:1}` (wrapped in `Svg`/`ForeignObject`), never zero — a direct, unconditional consequence of `chase/markdown/slide.go`'s `slideSplitTransformer`, which always appends one run (even an empty one) after its loop. The TRD's premise of "no Section children" does not hold against the real, already-shipped Objective-1 dependency.
- **Fix:** `Build` performs no empty-input special-casing at all (consistent with MODEL-01's core mandate: faithfully mirror the SAME finalized AST, never fabricate a shape not actually present). `TestBuildEmptyDocument` asserts the real, verified behavior: `Sections = [{ID:1, Attrs:nil, Notes:nil}]`, `Outline = nil`, `SchemaVersion` populated, no panic.
- **Files modified:** chase/model/build_test.go (test expectation only; `build.go` requires no empty-case branch)
- **Verification:** `TestBuildEmptyDocument` passes; re-confirmed empirically via `go run` scratch script showing `markdown.Parse("")`'s full AST dump (one `MarpitSection{ID:1}` under `MarpitSvg`/`MarpitForeignObject`).
- **Committed in:** 1f1d844 (Task 2 commit)

---

**Total deviations:** 4 auto-fixed (all Rule 1 - Bug; all fixed within Task 2's own TDD RED->GREEN loop, before the task commit)
**Impact on plan:** All four are corrections to the TRD's own illustrative/error_recovery text so it matches the real, already-shipped chase/markdown behavior — none change MODEL-01's scope or the shipped schema. No scope creep.

## Issues Encountered
None beyond the four auto-fixed deviations above, all resolved within the TDD cycle before commit.

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- `chase/model.Build` is ready for 02-02 (chase/profile) to consume: it returns a stable, JSON-serializable `Document` derived from the exact same parsed AST a profile's HTML sink also renders.
- 02-04's capstone (single-parse, two-sink integration test) can now share one `markdown.Parse` call between an HTML render and a `Build` call, per this TRD's own non-mutation proof.
- 02-03's full profile-agnostic grep gate can build on the clean `grep -rnE '\bSlide\b|"section"|16:9|1280|720'` result already confirmed here for `chase/model`.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: chase/model/document.go
- FOUND: chase/model/document_test.go
- FOUND: chase/model/build.go
- FOUND: chase/model/build_test.go
- FOUND: .planning/objectives/02-model-profile/02-01-SUMMARY.md
- FOUND commit: b9f1ec2 (Task 1)
- FOUND commit: 1f1d844 (Task 2)

---
*Objective: 02-model-profile*
*Completed: 2026-07-21*
