---
objective: 06-convert-pptx
trd: "04"
subsystem: convert/pptx
tags: [ooxml, drawingml, pptx, editable-shapes, ecma-376, tdd, exp-03]

# Dependency graph
requires:
  - "06-01: chase/model schema-v2 (Section.Blocks: paragraph/list/heading/code/math, ordered) + Outline"
  - "06-02: convert/pptx EMU utility (Inches/Points/Centipoints, SlideSize16x9/4x3, IdentityGroupTransform)"
  - "06-03: convert/pptx deterministic OPC packager (buildZip) + static part scaffold (parts_static.go/contenttypes.go) + structural asserter"
provides:
  - "ToPPTX(doc *model.Document, opts Options) ([]byte, error) -- the PUBLIC API of the whole convert/pptx objective (EXP-03): an editable-text-box .pptx built DIRECTLY from the docmodel, no HTML, no chromedp"
  - "Options{SlideSize} -- zero value defaults to 16:9; SlideSize4x3 selects 4:3 (threaded into <p:sldSz>)"
  - "buildSlide(section, outline, size) ([]byte,[]byte) -- one Section -> one ppt/slides/slideN.xml + rels (title from lowest-Level Outline, body from Section.Blocks in order)"
  - "shapes.go: the SINGLE Block-kind -> DrawingML mapping place -- buildTextBox (<p:sp>), buildParagraph (<a:p> plain/bullet/ordered), buildGroupShape (<p:grpSp> identity), shapeIDGen (per-slide unique cNvPr ids), escapeXML"
affects: [06-05-notes-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "String-built (strings.Builder + fmt) prefixed DrawingML (p:/a:) with ALL user text funneled through a single escapeXML (xml.EscapeText) seam -- matches parts_static.go's established convention (struct-marshal only the prefix-free .rels/content-types; string-build the namespaced parts) while guaranteeing < > & \" can never corrupt the OOXML"
    - "One shapeIDGen threaded through every builder on a slide (grouped + ungrouped) so cNvPr ids are globally unique within the slide -- the spTree group is id 1, shapes start at 2, a group's id is allocated before its children"
    - "Heading de-duplication: every markdown heading is materialized by chase/model into BOTH Outline AND a BlockHeading; buildSlide derives the title from the lowest-Level Outline entry and skips the FIRST matching heading Block (by Level+Text) so the title never double-renders, while all OTHER headings survive as body heading shapes"
    - "Body content wrapped in one identity <p:grpSp> per content-bearing slide (chOff==off, chExt==ext via 06-02's IdentityGroupTransform) -- satisfies criterion 3's grouped-shape case on every real slide with zero coordinate-scaling math"

key-files:
  created:
    - convert/pptx/shapes.go
    - convert/pptx/shapes_test.go
    - convert/pptx/slide.go
    - convert/pptx/slide_test.go
    - convert/pptx/pptx.go
    - convert/pptx/pptx_test.go
  modified: []

key-decisions:
  - "String-building (not encoding/xml struct marshaling) for the heavily prefixed DrawingML shapes, following parts_static.go's existing split; escaping correctness is guaranteed by routing 100% of user text through one escapeXML(xml.EscapeText) helper -- proven by a round-trip parse test (emitted XML decodes back to the exact original < > & \" input)."
  - "Body shapes are grouped in a single identity <p:grpSp> per slide (rather than wrapping the title, or emitting a synthetic empty group) -- semantically meaningful (the body content group), keeps text individually editable, and guarantees the grouped-shape case appears on every content-bearing slide."
  - "code/math blocks render their RAW Text as a plain body run (lossless text, no highlighting/typesetting) so no Section.Blocks content is silently dropped; syntax highlighting + math typesetting remain Objective 7's / a future TRD's scope. paragraph/list/heading are mapped explicitly per the must_haves."
  - "Title derived from the lowest-Level Outline entry (per the must_haves) with the matching heading Block skipped in the body loop -- honoring both 'title from Outline' and 'do not silently drop extra headings' without double-rendering."

patterns-established:
  - "shapes.go is the one place model.Block -> DrawingML shape-kind mapping lives; 06-05 (speaker notes) and any future block kind extend this single mapping rather than duplicating <p:sp>/<a:p> XML."

requirements-completed: [EXP-03]

# Verification evidence
verification:
  gates_defined: 6
  gates_passed: 6
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 12min
completed: 2026-07-21
---

# Objective 6 TRD 04: ToPPTX — Editable-Text-Box PPTX from the Docmodel Summary

**The public `ToPPTX(doc *model.Document, Options) ([]byte, error)`: each `chase/model.Section` becomes one `ppt/slides/slideN.xml` whose title (lowest-Level Outline heading) and body (`Section.Blocks`: paragraphs, lists, headings) render as REAL, editable `<p:sp>` text-box shapes with `<a:t>` runs — EMU-placed via 06-02, wrapped in an identity `<p:grpSp>` grouped-shape case, built directly from the docmodel with zero Chrome/HTML and deterministic, XML-safe output. This is EXP-03's core promise delivered.**

## Performance

- **Duration:** ~12 min (TRD assignment 17:08Z → Task 3 commit 17:19Z UTC)
- **Started:** 2026-07-21T17:08:44Z
- **Completed:** 2026-07-21T17:20:05Z
- **Tasks:** 3/3 complete
- **Files created:** 6 (all new; go.mod/go.sum untouched)

## Accomplishments

- **`shapes.go` — the single Block→DrawingML mapping place:**
  - `buildTextBox` emits an editable `<p:sp>` text box: `nvSpPr` (`cNvPr` id/name, `cNvSpPr txBox="1"`), `spPr` with an EMU `<a:xfrm>` (06-02 Inches/Points) + rect `prstGeom`, and a `txBody` of real `<a:p>/<a:r>/<a:t>` runs. **No image.**
  - `buildParagraph` emits `<a:p>` with an optional list `<a:pPr lvl marL indent>` — `<a:buChar char="•"/>` (unordered) or `<a:buAutoNum type="arabicPeriod"/>` (ordered) — then one `<a:r>` run; `marL` scales per nesting level (06-02 `Inches(0.5)*(level+1)`), `sz` in **centipoints** (06-02 `Centipoints`), never EMU.
  - `buildGroupShape` emits `<p:grpSp>` with `<a:xfrm>` off/ext/chOff/chExt from 06-02's `IdentityGroupTransform` (chOff==off, chExt==ext) wrapping child text boxes whose off/ext are literal slide EMU.
  - `shapeIDGen` hands out per-slide-unique `cNvPr` ids (spTree group = 1, shapes start at 2) threaded through grouped + ungrouped builders alike.
  - `escapeXML` routes every user string (title/paragraph/list-item text + shape name) through `xml.EscapeText`.
- **`slide.go` — `buildSlide(section, outline, size)`:** composes the slide's `<p:cSld><p:spTree>` (own group id 1), a title `<p:sp>` from the Section's lowest-Level Outline entry (skipped when none — untitled edge, no fabricated title), and body shapes from `Section.Blocks` in order (paragraph→run, list→bulleted/numbered `<a:p>`, heading→heading box, code/math→raw-text run) wrapped in one identity `<p:grpSp>`. `skipTitleHeading` prevents the title heading (which chase/model materializes into BOTH Outline and a BlockHeading) from double-rendering, while additional headings survive as body shapes. Emits `slideN.xml.rels → slideLayout1`.
- **`pptx.go` — public `ToPPTX` + `Options`:** defaults `SlideSize` to 16:9 when zero, builds one `slideN.xml` per `doc.Sections`, and performs the 3-fold per-slide wiring — `<p:sldIdLst>` entry + `presentation.xml.rels` rId (rId6..) + `[Content_Types].xml` `ctSlide` Override — then assembles via 06-03's deterministic `buildZip`. Reuses 06-03's packager + static parts unchanged; only the N-slide loop is new.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: `<p:sp>` + `<p:grpSp>` builders | `gofmt -l convert/pptx/shapes.go && go build ./... && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Shape\|TextBox\|Bullet\|Ordered\|Group\|Escap' -v && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: Section → slideN.xml | `gofmt -l convert/pptx/slide.go && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Slide\|Section\|Untitled' -v && go test ./convert/pptx/ && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 3: public ToPPTX + N-slide wiring | `gofmt -l convert/pptx/pptx.go && go build ./... && go vet ./... && go test ./convert/pptx/ -run 'ToPPTX\|EndToEnd\|Deterministic\|Aspect' -v && go test ./... && bash scripts/check-no-chromedp.sh && go list -deps ./convert/pptx/... \| grep -c chromedp (==0) && go list -deps ./press/... ./chase/... ./profiles/... \| grep -c convert/pptx (==0)` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: `<p:sp>` text-box + `<p:grpSp>` group-shape builders (Block → DrawingML)** — `0be1ad4` (feat)
2. **Task 2: Section → slideN.xml (title from Outline, body from Blocks)** — `e32427f` (feat)
3. **Task 3: public ToPPTX + Options; wire N model-driven slides into the deterministic package** — `635e0bb` (feat)

_All three tasks are `tdd="true"`; each RED (compile failure against the undefined builders/types) was confirmed before its GREEN implementation — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l convert/pptx/*.go` | 0 (no output) | PASS |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (whole repo, incl. convert/pptx) | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| writer browser-free | `go list -deps ./convert/pptx/... \| grep -c chromedp` | 0 matches | PASS |
| isolation invariant | `go list -deps ./press/... ./chase/... ./profiles/... \| grep -c convert/pptx` | 0 matches | PASS |
| addlicense | `addlicense -check convert/pptx/{shapes,slide,pptx}{,_test}.go` | 0 | PASS |
| no new deps | `git diff HEAD -- go.mod go.sum` | (empty) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./convert/pptx/ -run 'Shape\|TextBox\|Bullet\|Ordered\|Group\|Escap'` | 1 (compile: undefined buildTextBox/textBox/paragraph/bulletChar/...) | FAIL (correct) |
| GREEN (Task 1) | same | 0 (7 tests PASS, incl. renamed case-2 `TestShapeParagraphBodyRun`) | PASS (correct) |
| RED (Task 2) | `go test ./convert/pptx/ -run 'Slide\|Section\|Untitled'` | 1 (compile: undefined buildSlide) | FAIL (correct) |
| GREEN (Task 2) | same | 0 (4 tests PASS) | PASS (correct) |
| RED (Task 3) | `go test ./convert/pptx/ -run 'ToPPTX\|EndToEnd\|Deterministic\|Aspect'` | 1 (compile: undefined ToPPTX/Options) | FAIL (correct) |
| GREEN (Task 3) | same | 0 (5 tests PASS) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 — no deviations; every RED→GREEN converged on the first implementation. (One non-deviation refinement: the case-2 test was renamed `TestParagraphBodyRun`→`TestShapeParagraphBodyRun` so the TRD's own `-run 'Shape|...'` filter actually exercises it — a test-naming fix, not a code change, applied within Task 1 before its commit.)
- **Must-haves verified:** 5/5 (all `must_haves.truths`):
  1. `ToPPTX(doc *model.Document, opts Options) ([]byte, error)` maps one Section → one slideN.xml consuming the docmodel directly — **0 chromedp** in `convert/pptx` deps, **0** HTML parsing.
  2. Text renders as REAL editable `<p:sp>` `<a:t>` runs — asserted by decoding each slide's runs back to the section's title + block text; title from lowest-Level Outline, body from Blocks (paragraph/list bullet+ordered/heading); **no `<p:pic>`/`<a:blip>` anywhere**.
  3. Every shape EMU-placed (title xfrm asserted exact: `off x=457200 y=274320`, `ext cx=8229600 cy=1097280`) with ≥1 grouped `<p:grpSp>` identity case (chOff==off, chExt==ext EMU-asserted) on every content slide.
  4. `Options{SlideSize}` threads 16:9 (default) / 4:3 into `<p:sldSz>`; output deterministic (byte-identical rebuild proven).
  5. User text XML-escaped (round-trip parse of `< > & "` through the full ToPPTX pipeline yields back the exact input).
- **Gate failures:** None.

## Files Created/Modified

- `convert/pptx/shapes.go` — `shapeIDGen`; `bulletKind` (none/char/autonum); `paragraph`/`textBox`/`groupShape` types; `escapeXML`, `listMarL`, `listIndent`; `buildParagraph`/`buildTextBox`/`buildGroupShape`.
- `convert/pptx/shapes_test.go` — Test-list cases 1–5: `TestTextBoxShapeTitle`, `TestShapeParagraphBodyRun`, `TestListShapeBulletAndOrdered`, `TestGroupShapeIdentity`, `TestXMLEscaping` (+ `decodeRunTexts`/`assertNoImage` helpers).
- `convert/pptx/slide.go` — layout geometry (EMU) + font-size (pt→centipoint) constants; `sectionTitle`, `skipTitleHeading`, `bodyRunText`, `buildBodyShapes`, `buildSlide`.
- `convert/pptx/slide_test.go` — Test-list cases 6–7 + locks: `TestSlidePerSectionCarriesTitleAndBody`, `TestSlideBetaListIsBulleted`, `TestUntitledSectionEmitsBodyOnly`, `TestSlideMultiHeadingKeepsExtraHeadings` (+ `threeSectionDoc` inline fixture).
- `convert/pptx/pptx.go` — `Options{SlideSize}` + public `ToPPTX(doc, opts)`; per-slide 3-fold wiring + deterministic assembly.
- `convert/pptx/pptx_test.go` — Test-list case 8: `TestToPPTXEndToEnd16x9`, `TestToPPTXDefaultsTo16x9`, `TestToPPTXAspect4x3`, `TestToPPTXDeterministic`, `TestToPPTXEscapesUserText` (+ `wickedDoc` escaping fixture).

## Decisions Made

- String-built prefixed DrawingML + one `escapeXML` seam (matches parts_static.go convention); escaping proven by round-trip parse, not by asserting exact `&quot;` bytes (xml.EscapeText emits `&#34;` for `"`, still well-formed and round-trips correctly).
- Body content grouped in a single identity `<p:grpSp>` per content slide (criterion 3 on every real slide, text stays individually editable).
- code/math blocks render raw Text as body runs (no content dropped); paragraph/list/heading mapped explicitly per the must_haves.
- Title from lowest-Level Outline + `skipTitleHeading` de-dup (honors "title from Outline" AND "don't drop extra headings" without double-render).

## Deviations from Plan

None — 06-04-TRD.md executed exactly as written. All three tasks' RED phases failed as expected (compile errors against undefined identifiers) and all GREEN phases passed on the first implementation attempt with zero auto-fix cycles. The only mid-task adjustment was a test rename (`TestParagraphBodyRun`→`TestShapeParagraphBodyRun`) so the TRD's prescribed `-run` filter matches the case-2 test — a naming fix within Task 1, not a plan deviation.

## Issues Encountered

None. Baseline (`go build`, `go test ./convert/pptx/`) was green before starting; every gate stayed green throughout.

## User Setup Required

None — pure stdlib (archive/zip, encoding/xml), zero new dependencies, no external service configuration.

## Next Objective Readiness

- `ToPPTX` is the objective's public API; **06-05** layers speaker notes (`notesSlideN.xml` from `Section.Notes`) + the full openability/position verification on top of this entry point, extending shapes.go's single Block→shape mapping rather than duplicating it.
- The `<p:grpSp>` identity case is proven; a future TRD can introduce non-identity `chOff/chExt` scaling via 06-02's already-tested `GroupTransform.MapChild` once a real nested-group need appears.
- This TRD ran in an isolated worktree (wave 3); `convert/pptx` remains at **0 references** from press/chase/profiles and **0 chromedp** in its own dep closure — pending orchestrator reconcile at merge.

## Self-Check: PASSED

All claimed files confirmed present on disk; all three task commit hashes confirmed present in `git log`.

- FOUND: convert/pptx/shapes.go
- FOUND: convert/pptx/shapes_test.go
- FOUND: convert/pptx/slide.go
- FOUND: convert/pptx/slide_test.go
- FOUND: convert/pptx/pptx.go
- FOUND: convert/pptx/pptx_test.go
- FOUND: .planning/objectives/06-convert-pptx/06-04-SUMMARY.md
- FOUND commit: 0be1ad4 (Task 1)
- FOUND commit: e32427f (Task 2)
- FOUND commit: 635e0bb (Task 3)

---
*Objective: 06-convert-pptx*
*Completed: 2026-07-21*
