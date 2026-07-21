---
objective: 06-convert-pptx
trd: "01"
subsystem: docmodel
tags: [goldmark, ast-walk, json-schema, schema-v2, docmodel, math, duck-typing]

# Dependency graph
requires:
  - objective: 02-model-profile
    provides: "chase/model.Build single read-only ast.Walk deriving Document{SchemaVersion,Meta,Sections,Outline} from chase/markdown's finalized AST"
  - objective: 03-batteries-api
    provides: "press/math battery — bespoke $/$$ InlineParser + custom (unexported) mathNode{Raw,Block} + KindMath; press.Render one-parse-two-sinks engine (markdown.NewEngine(pmath.Option(...)))"
provides:
  - "chase/model schema v2: Section.Blocks []Block — ordered per-section body content, union {paragraph|list|code|math|heading}, materialized by the SAME single read-only Build walk"
  - "Block/BlockKind/ListItem exported types: code carries {Text=raw source, Language}; math carries {Text=raw TeX, Display}; list carries {Ordered, Items[]{Text,Level}}; heading carries {Level,Text}"
  - "SchemaVersion bumped eden-press.model/v1 -> /v2 (strict additive superset; every new field omitempty -> block-less JSON byte-identical to v1 except the version string)"
  - "press/math additive exported getters MathRaw() string / MathDisplay() bool on *mathNode (pure getters; zero parse/render/HTML change)"
  - "duck-typed rawMath interface seam in chase/model — reaches raw TeX with NO press/math import (cycle-free, no-chromedp closure intact)"
affects: [06-04, 07-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive schema versioning: every new field carries omitempty so a v(N) document that uses none of the new fields serializes byte-identically to v(N-1) apart from the SchemaVersion string — proven with a frozen-v1-golden normalize-and-compare test (TestBlockOmitemptyAdditive)"
    - "Duck-typed cross-package accessor seam: chase/model recovers press/math's UNEXPORTED mathNode's raw TeX via a locally-declared `rawMath interface{ MathRaw() string; MathDisplay() bool }` type-assertion — no import of press/math, so no dependency-closure coupling and no import cycle, while press/math exposes only two additive pure getters"
    - "Structural math-only-paragraph skip: goldmark wraps a standalone `$$…$$`/`$…$` line in a Paragraph whose ONLY child is the math node, yet Paragraph.Text reconstructs the RAW `$$…$$` source (the math node does not strip it) — so a naive text check double-emits raw TeX as prose; isMathOnlyParagraph checks children structurally (any math child + all other children whitespace) and skips the text Block, letting the walk's rawMath case emit the math Block alone (document order + display flag preserved)"

key-files:
  created:
    - .planning/objectives/06-convert-pptx/06-01-SUMMARY.md
  modified:
    - chase/model/document.go
    - chase/model/document_test.go
    - chase/model/build.go
    - chase/model/build_test.go
    - press/math/math.go
    - press/math/math_test.go

key-decisions:
  - "Math-only paragraph yields a STANDALONE math Block, NOT an empty paragraph Block + a math Block (per TRD error_recovery's recommended option) — preserves document order + the display flag and avoids double-emitting raw TeX. Implemented via isMathOnlyParagraph, a structural (not text-based) check, because Paragraph.Text returns the literal `$$…$$` source even when the sole child is a math node (empirically confirmed against goldmark v1.8.4)."
  - "Math block extraction lives in the walk's `default:` branch guarded by the rawMath type-assertion, NOT a concrete-type case — chase/model cannot name press/math's unexported *mathNode, and the duck-typed interface deliberately captures ANY node exposing MathRaw()/MathDisplay(), keeping the two packages decoupled."
  - "List extraction returns ast.WalkSkipChildren after collecting all items (including nested, each at its 0-based Level) so list-item TextBlocks/Paragraphs are never ALSO emitted as loose paragraph Blocks; nested lists are flattened entirely by collectListItems, never re-encountered by the outer walk."
  - "Code block raw source is reconstructed from node.Lines() segments (pre-chroma), and Language from FencedCodeBlock.Language(source) — the lossless JSON-native surface Objective 7's flutter_highlighting consumes; indented *ast.CodeBlock yields empty Language."

patterns-established:
  - "A schema-vN->v(N+1) bump is a strict additive superset when validated by BOTH a round-trip-stability test (every new field populated) AND a frozen-golden normalize test (no new field populated => bytes differ only by the version string)."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 12min
completed: 2026-07-21
---

# Objective 6 TRD 01: chase/model schema-v2 Blocks + press/math raw-TeX accessor Summary

**The ONE shared schema-v2 join point: `chase/model.Section.Blocks` — an additive, document-ordered union of paragraph/list/code(source+language)/math(rawTeX+display)/heading blocks materialized by the SAME single read-only `Build` ast.Walk — plus press/math's additive `MathRaw()`/`MathDisplay()` getters reached through a duck-typed seam so chase/model never imports press/math. Unblocks Objective 7 (DART-04) and feeds Objective 6's PPTX writer (06-04).**

## 🔓 Unblocks Objective 7 (DART-04)

This TRD is the `shared_prerequisite` join point Objective 7 waits on. `Section.Blocks` now delivers the EXACT union contract 07-05/DART-04 serializes verbatim:
- **code** blocks carry `{Text: <raw source, pre-chroma>, Language: <info-string>}` → `flutter_highlighting`
- **math** blocks carry `{Text: <raw TeX>, Display: <$$ vs $>}` → `flutter_math_fork`
- **paragraph / list / heading** blocks carry editable plain text → Objective 6's PPTX `<p:sp>` text-box shapes

Neither the raw code source nor the raw math TeX is recoverable from `Output.HTML` (chroma-classed spans / presentation-MathML with no `<annotation>`); `Section.Blocks` is the lossless, JSON-native surface. **Objective 7 (DART-04) is now unblocked.**

## Performance

- **Duration:** ~12 min (Task 1 commit 11:54 → Task 3 commit 12:05, local time)
- **Started:** 2026-07-21 (worktree branched from main @ 1b35c1b)
- **Completed:** 2026-07-21
- **Tasks:** 3/3 complete
- **Files modified:** 6 (2 non-test source, 1 non-test source + 3 test files)

## Accomplishments
- **Schema v2 types** (`document.go`): `Block{Kind, Text, Level, Language, Display, Ordered, Items}` + `BlockKind` enum (`paragraph|list|code|math|heading`) + `ListItem{Text, Level}`; `Section.Blocks []Block` appended after `Notes`; `SchemaVersion` bumped `eden-press.model/v1` → `/v2`. Every new field `omitempty` → a block-less document's JSON is byte-identical to v1 except the version string.
- **Block extraction** folded into `Build`'s EXISTING single `ast.Walk` (`build.go`) — paragraph, heading (alongside the unchanged Outline entry), list (nested items at 0-based Level, Ordered flag, `WalkSkipChildren` to avoid double-emit), fenced + indented code (raw source via `Lines()`, Language via `Language(source)`). No second parse, no doc mutation — `TestBuildNonMutation` stays green.
- **press/math additive getters** (`math.go`): `MathRaw() string` / `MathDisplay() bool` pure getters on `*mathNode` — zero change to the `$`-parser, NodeRenderer, latex2mathml/go-latex routing, or emitted HTML.
- **Duck-typed seam** (`build.go`): local `rawMath interface{ MathRaw() string; MathDisplay() bool }` + a `default:`-branch type-assertion emits a math Block from ANY node satisfying it — `chase/model` NEVER imports `press/math` (`go list -deps ./chase/model/... | grep -c press/math` = 0; cycle-free; no-chromedp closure intact).
- **Math-only-paragraph skip**: `isMathOnlyParagraph` structurally detects a Paragraph whose only content is math node(s) and skips the text Block so the raw `$$…$$` source is not double-emitted as prose (Paragraph.Text reconstructs the literal source even when the sole child is a math node).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: schema-v2 Block types + Section.Blocks + v2 bump | `go test ./chase/model/ -run 'SchemaVersion\|RoundTrip\|Omitempty\|Block' -v` | 0 | PASS |
| 2: paragraph/heading/list/code extraction in Build's walk | `go test ./chase/model/ -run 'Blocks\|Paragraph\|List\|Code\|Heading\|NonMutation' -v` | 0 | PASS |
| 3: math Block via press/math getters + duck-typed seam | `go test ./chase/model/ -run Math -v && go test ./press/math/ -run 'MathRaw\|MathDisplay\|Accessor' -v` | 0 | PASS |

## Task Commits

Each task committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: schema-v2 Block types + Section.Blocks (additive, omitempty)** — `5db9bb0` (feat)
2. **Task 2: extract paragraph/heading/list/code Blocks in Build's single read-only walk** — `ab20165` (feat)
3. **Task 3: math Block extraction via additive press/math getters + duck-typed rawMath seam** — `8ad7242` (feat)

_All three tasks are `tdd="true"`; RED confirmed before each GREEN — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (Obj-0 corpus/cssdiff/htmldiff, Obj-1, Obj-2 model, Obj-3 press, profiles) | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| gofmt | `gofmt -l chase/model/ press/math/` | 0 (no output) | PASS |
| import-cycle | `go list -deps ./chase/model/... \| grep -c press/math` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./chase/model/ -run 'SchemaVersion\|RoundTrip\|Omitempty\|Block'` | 1 (compile: undefined Block/BlockKind/ListItem, unknown field Blocks) | FAIL (correct) |
| GREEN (Task 1) | `go test ./chase/model/ -run 'SchemaVersion\|RoundTrip\|Omitempty\|Block' -v` | 0 | PASS (correct) |
| RED (Task 2) | `go test ./chase/model/ -run 'Blocks\|Paragraph\|List\|Code\|Heading'` | 1 (assertions: `blocks = 0, want N`) | FAIL (correct) |
| GREEN (Task 2) | `go test ./chase/model/ -run 'Blocks\|Paragraph\|List\|Code\|Heading\|NonMutation' -v` | 0 | PASS (correct) |
| RED (Task 3 accessor) | `go test ./press/math/ -run 'MathRaw\|MathDisplay\|Accessor'` | 1 (compile: MathRaw/MathDisplay undefined) | FAIL (correct) |
| RED (Task 3 build) | `go test ./chase/model/ -run TestBuildMathBlocks` | 1 (assertion: `math blocks = 0, want 2`) | FAIL (correct) |
| GREEN (Task 3) | `go test ./chase/model/ -run Math -v && go test ./press/math/ -run 'MathRaw\|MathDisplay\|Accessor' -v` | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 06-01-TRD.md frontmatter — additive Section.Blocks in one walk; v2 bump with omitempty byte-identity; raw code source+language & raw math TeX+display; editable paragraph/list/heading text; press/math additive getters reached without a press/math import)
- **Gate failures:** None
- **Additive invariant:** `git diff` on `document.go` shows only ADDED types/field + the SchemaVersion string change (removed lines are doc-comment rewrites + the old v1 constant); existing `Section{ID,Attrs,Notes}`/`Document`/`OutlineEntry` fields, JSON tags, and order untouched.

## Files Created/Modified
- `chase/model/document.go` — `BlockKind` enum, `Block`/`ListItem` types, `Section.Blocks` field, `SchemaVersion` → v2
- `chase/model/document_test.go` — v2 constant/round-trip updates + `TestBlockOmitemptyAdditive` (case 1, frozen-golden normalize) + `TestSchemaV2RoundTrip` (case 2, every-kind byte-stable round-trip)
- `chase/model/build.go` — `rawMath` interface, walk cases (paragraph/list/code/heading-block + `default` math), `collectListItems`/`rawLinesText`/`isMathOnlyParagraph` helpers
- `chase/model/build_test.go` — cases 3–7 (paragraph/list/code/heading/math via battery engine) + Task-2 output-preservation regression
- `press/math/math.go` — additive `MathRaw()`/`MathDisplay()` pure getters on `*mathNode`
- `press/math/math_test.go` — `TestMathRawDisplayAccessors` (case 8: getters + duck-typed interface assertion)

## Decisions Made
- Math-only paragraph → standalone math Block (not empty paragraph + math), via a structural `isMathOnlyParagraph` check (Paragraph.Text returns literal `$$…$$` source even for a math-only paragraph — empirically confirmed).
- Math extraction in the walk's `default:` branch via the `rawMath` type-assertion (chase/model cannot name press/math's unexported `*mathNode`; the interface deliberately captures any node with the two getters).
- List `WalkSkipChildren` after `collectListItems` to prevent double-emitting item text; nested items flattened at 0-based Level.

## Deviations from Plan

None — the TRD executed exactly as written across all three tasks; zero auto-fix cycles.

**Clarification (not a deviation):** During Task 3 an apparent contradiction surfaced — the battery-engine parse produced a `Math` AST node, yet `Build`'s paragraph case captured the literal `$$E=mc^2$$` text. Root cause (confirmed via a disposable debug walk, since removed): goldmark v1.8.4's `Paragraph.Text` reconstructs the RAW `$$…$$` source even when the sole child is the (text-less) math node. The TRD's own `error_recovery` already anticipated this ("ensure paragraph text extraction does NOT double-emit the math node's raw TeX"); the `isMathOnlyParagraph` structural skip is exactly that mandated handling, implemented as specified — no scope change.

## Issues Encountered
None beyond the Task-3 clarification above, resolved within the TDD cycle before commit.

## User Setup Required
None — no external service configuration; `go.mod` untouched (stdlib + existing goldmark only).

## Next Objective Readiness
- **Objective 7 / DART-04 (07-05):** UNBLOCKED. `Section.Blocks` delivers the exact union contract — code(source+language) + math(rawTeX+display) as lossless JSON — DART-04 serializes verbatim. This was the `shared_prerequisite` join point.
- **Objective 6 / 06-04 (PPTX writer):** ready to consume paragraph/list/heading Blocks (+ Section.Notes for 06-05) as editable `<p:sp>` text-box shapes from `press.Output.Model`.
- The v2 schema is a strict additive superset of v1: any existing v1 consumer is unaffected apart from branching on the `schemaVersion` string.

## Self-Check: PASSED

All claimed files confirmed present on disk; all three task commit hashes confirmed in `git log`.

- FOUND: chase/model/document.go
- FOUND: chase/model/document_test.go
- FOUND: chase/model/build.go
- FOUND: chase/model/build_test.go
- FOUND: press/math/math.go
- FOUND: press/math/math_test.go
- FOUND: .planning/objectives/06-convert-pptx/06-01-SUMMARY.md
- FOUND commit: 5db9bb0 (Task 1)
- FOUND commit: ab20165 (Task 2)
- FOUND commit: 8ad7242 (Task 3)

---
*Objective: 06-convert-pptx*
*Completed: 2026-07-21*
