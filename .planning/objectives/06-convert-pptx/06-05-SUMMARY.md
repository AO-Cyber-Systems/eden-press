---
objective: 06-convert-pptx
trd: "05"
subsystem: convert/pptx
tags: [ooxml, drawingml, pptx, speaker-notes, acceptance-gate, ecma-376, tdd, exp-03]

# Dependency graph
requires:
  - "06-02: convert/pptx EMU utility (Inches/Points/Centipoints, SlideSize16x9/4x3, IdentityGroupTransform)"
  - "06-03: convert/pptx deterministic OPC packager (buildZip) + static part scaffold (parts_static.go/contenttypes.go) + structural asserter (assertStructurallyOpenable/assertSlideSize)"
  - "06-04: ToPPTX(doc, opts) public API + buildSlide/shapes.go (Block -> DrawingML mapping, escapeXML, shapeIDGen, IdentityGroupTransform usage)"
provides:
  - "notes.go: Section.Notes -> ppt/notesSlides/notesSlideN.xml (conditional, one <a:p> per note string, <p:ph type=\"body\">), ppt/notesMasters/notesMaster1.xml (once-per-deck, iff any notes), and the full 4-way rels wiring (slideN.xml.rels->notesSlideN, notesSlideN.xml.rels->slideN+notesMaster1, presentation.xml.rels->notesMaster1, [Content_Types].xml Overrides)"
  - "verify_test.go: the objective's acceptance gate -- a realistic titles+paragraphs+nested-list+notes model.Document through the public ToPPTX at BOTH SlideSize16x9 and SlideSize4x3, asserting structural openability, editable <a:t> run content, notes-slide body text, and EMU shape positions independently recomputed from the writer's own layout helpers"
  - "presentationRelsXML(slides []slideRef, extra ...relationship) -- backward-compatible variadic extension point for injecting the notesMaster relationship into presentation.xml.rels without touching its fixed 5-relationship singleton list"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Conditional OPC part emission: notesSlideN.xml + its 2-relationship .rels are emitted ONLY for a section with len(Notes) > 0; notesMaster1.xml (+ rels + Content-Types Override + presentation.xml.rels entry) is emitted AT MOST ONCE, iff any section in the whole deck has notes -- both gated by a single hasNotes bool threaded through ToPPTX's assembly loop, in fixed slide order (never map-ranged)"
    - "4-way notes relationship-graph wiring closed exactly per 06-RESEARCH Pattern 1: slideN.xml.rels gains a notesSlide entry (via buildSlideRelsWithNotes, which OVERRIDES buildSlide's returned rels rather than merging into it); notesSlideN.xml.rels declares -> slideN + notesMaster1; notesMaster1.xml.rels declares -> theme1; presentation.xml.rels gains the notesMaster rId via a new variadic `extra ...relationship` parameter on presentationRelsXML -- no <p:notesMasterIdLst> element needed in presentation.xml's own content (the rels-file relationship is the only wiring the OOXML schema/06-RESEARCH pattern requires; slideMaster1.xml.rels's own always-declared-but-never-r:id-referenced theme relationship is the existing in-repo precedent for this asymmetry)"
    - "Position acceptance without hand-guessed pixels: verify_test.go's expectedShapePositions() calls the EXACT SAME helpers buildSlide uses internally (sectionTitle, skipTitleHeading, buildBodyShapes, newShapeIDGen) to independently recompute the writer's OWN intended EMU off/ext sequence, then a regex (extractOffExtPairs) pulls the actual emitted <a:off>/<a:ext> pairs from the real slide XML in document order for a structural reflect.DeepEqual -- proving the generated XML matches the writer's declared layout contract, not a redundant re-implementation of buildSlide"
    - "One shared structural asserter (assertStructurallyOpenable, from 06-03) is proven at BOTH SlideSize16x9 and SlideSize4x3 from a SINGLE realistic fixture (acceptanceDoc) via one assertAcceptanceDeck(t, size) helper parameterized only by size -- both-aspect-ratio coverage is the literal test structure, not a written-twice duplication"

key-files:
  created:
    - convert/pptx/notes.go
    - convert/pptx/notes_test.go
    - convert/pptx/verify_test.go
  modified:
    - convert/pptx/pptx.go
    - convert/pptx/parts_static.go

key-decisions:
  - "presentationRelsXML's signature extended to a variadic `presentationRelsXML(slides []slideRef, extra ...relationship) []byte` (in parts_static.go, NOT in the TRD's declared files_modified list) rather than adding a new sibling function or a <p:notesMasterIdLst> element -- backward-compatible (every existing 1-arg call site still compiles unchanged), and keeps the single presentation-rels builder as the one place that assembles the fixed-plus-conditional relationship list, avoiding a second, drifting copy of the same 5 singleton relationships."
  - "buildSlideRelsWithNotes(slideNum) OVERRIDES (not merges into) buildSlide's returned relsXML when a section has notes, rather than modifying slide.go/buildSlide itself (also outside files_modified) -- both relationship lists are only 2 entries (slideLayout1 + notesSlideN when notes exist), so a full override is simpler and less error-prone than a merge/patch step, and slide.go stays untouched."
  - "No <p:notesMasterIdLst> element added to presentation.xml's own content -- only the ppt/_rels/presentation.xml.rels entry -- per the TRD's own 06-RESEARCH Pattern 1 wiring diagram (which lists presentation.xml's only change as the already-emitted <p:notesSz>) and consistent with the existing slideMaster1.xml.rels precedent (a declared-but-inline-unreferenced theme relationship is valid and already asserted-allowed by assertStructurallyOpenable)."
  - "verify_test.go proves shape positions by calling buildSlide's OWN internal helpers (sectionTitle/skipTitleHeading/buildBodyShapes) to recompute expected EMU values, rather than hand-coding a second independent layout calculation -- per the TRD's error_recovery guidance ('assert the EMU values the writer INTENDS'), this is a structural cross-check of the writer's declared contract, not a tautological no-op, because it still parses the ACTUAL emitted XML bytes via a strict adjacency regex and compares in document order."

patterns-established:
  - "notes.go is the one place Section.Notes -> notesSlide/notesMaster OOXML lives; any future notes-adjacent feature (e.g. notes formatting) extends this file rather than duplicating the rels-wiring or <p:notes> shape logic."
  - "verify_test.go is the objective's standing acceptance gate: any future convert/pptx TRD that changes slide/notes layout should keep this test green (or deliberately update its expectedShapePositions/wantAcceptanceSlideRuns fixtures), since it is the concrete evidence OBJECTIVE.md's verification step reads for criteria 1, 2, and 4."

requirements-completed: [EXP-03]

# Verification evidence
verification:
  gates_defined: 7
  gates_passed: 7
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~5min (Task 1 commit -> Task 2 commit; investigation/design time preceding Task 1's commit not separately tracked across a context-compaction boundary)
completed: 2026-07-21
---

# Objective 6 TRD 05: Speaker Notes + Comprehensive Openability/Position Acceptance Summary

**`Section.Notes` now renders to fully-wired, editable `ppt/notesSlides/notesSlideN.xml` speaker-notes parts (conditional per section, `notesMaster1.xml` once per deck), and a comprehensive acceptance test proves — structurally, at BOTH 16:9 and 4:3, from one realistic titles+paragraphs+nested-list+notes fixture — that the deck opens correctly with every shape's EMU position matching the writer's own declared layout. This closes Objective 6: EXP-03 is fully satisfied.**

## Performance

- **Duration:** ~5 min from Task 1's commit to Task 2's commit (11dc9be -> d0a1766); investigation/design work preceding the first commit spanned a context-compaction boundary and is not separately timed.
- **Task 1 commit:** 2026-07-21T17:46:15Z
- **Task 2 commit:** 2026-07-21T17:50:38Z
- **Tasks:** 2/2 complete
- **Files created:** 3 (notes.go, notes_test.go, verify_test.go); **files modified:** 2 (pptx.go, parts_static.go — see Deviations)

## Accomplishments

- **`notes.go` — `Section.Notes` -> speaker-notes OPC parts:**
  - `buildNotesSlide(notes []string)` emits the `<p:notes>` body-placeholder shape (`<p:ph type="body" idx="1">`) with one escaped `<a:p>` per note string; returns `nil` for an empty/nil slice (no notes -> no part).
  - `buildNotesMaster()` + `buildNotesMasterRels()` emit the once-per-deck static `notesMaster1.xml` (+ rels -> `theme1.xml`, reusing 06-03's 12-attr `clrMapAttrs`).
  - `buildNotesSlideRels(slideNum)` emits `notesSlideN.xml.rels` -> `slideN.xml` + `notesMaster1.xml`; `buildSlideRelsWithNotes(slideNum)` emits the notes-aware `slideN.xml.rels` (-> `slideLayout1.xml` + `notesSlideN.xml`), overriding `buildSlide`'s plain rels only when a section has notes.
- **`pptx.go`'s `ToPPTX`** now, per section with `len(Notes) > 0`: swaps in the notes-aware slide rels, appends a `notesSlideN.xml` + rels part and its `ctNotesSlide` Content-Types Override, all in fixed slide order. If ANY section has notes, it appends `notesMaster1.xml` (+ rels + `ctNotesMaster` Override) exactly once and threads the `notesMaster` relationship into `presentation.xml.rels` via the new `presentationRelsXML(slides, extra...)` variadic.
- **`verify_test.go` — the objective's acceptance gate:** one hand-built `acceptanceDoc()` fixture (3 sections: title+paragraph, title+2-level ordered list with notes, title+paragraph) run through `ToPPTX` at both `SlideSize16x9` and `SlideSize4x3`, asserting: (a) `assertStructurallyOpenable` (06-03), (b) `assertSlideSize` (06-03), (c) exact per-slide editable run text, (d) EMU shape positions matching `expectedShapePositions` (independently recomputed via `buildSlide`'s own `sectionTitle`/`skipTitleHeading`/`buildBodyShapes` helpers), (e) the notes slide's body text, and (f) that notes-free sections emit NO `notesSlideN.xml` part at all. An optional, `soffice`-PATH-guarded LibreOffice-headless convert-to-pdf smoke covers both sizes with a unique `t.TempDir()`-scoped `UserInstallation` per subtest (SKIPped in this environment — `soffice` not installed).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Speaker-notes slides + notesMaster (conditional) | `gofmt -l convert/pptx/notes.go convert/pptx/pptx.go && go build ./... && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Notes\|NotesMaster\|Conditional' -v && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: Comprehensive openability + position verification (16:9 + 4:3) | `go test ./convert/pptx/ -run 'Verify\|Comprehensive\|Acceptance\|Position\|Smoke' -v && go build ./... && go vet ./... && go test ./... && bash scripts/check-no-chromedp.sh && go list -deps ./convert/pptx/... \| grep -c chromedp (==0) && go list -deps ./press/... ./chase/... ./profiles/... \| grep -c convert/pptx (==0)` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Section.Notes -> notesSlideN.xml + full 4-way notes rels wiring** — `11dc9be` (feat)
2. **Task 2: comprehensive openability + position verify (16:9 + 4:3)** — `d0a1766` (test)

_Task 1 is `tdd="true"`; its RED phase (compile failure against undefined `buildNotesSlide`/`notes.go` symbols) preceded its GREEN implementation — see TDD Evidence below. Task 2 is a plain `auto` task (the acceptance/integration gate itself), not TDD-paired._

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

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./convert/pptx/ -run 'Notes\|NotesMaster\|Conditional'` | 1 (compile: undefined buildNotesSlide/buildNotesMaster/buildSlideRelsWithNotes) | FAIL (correct) |
| GREEN (Task 1) | same | 0 (5 tests PASS: TestNotesSlideBodyRuns, TestNotesSlideEmptyReturnsNil, TestNotesSlideEscapesUserText, TestNotesWiringClosure, TestConditionalNotesEmission) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 — one test-design correction made before either task's commit (an EMU-position assertion in `verify_test.go` initially omitted the slide's own always-zero root `<p:grpSpPr><a:xfrm>` off/ext pair that `buildSlide` unconditionally emits before the title; `expectedShapePositions` was corrected to prepend that fixed `{0,0}/{0,0}` entry before either Task 2 commit was made — a pre-commit test-correctness fix, not a deviation from the TRD's plan).
- **Must-haves verified:** 5/5 (all `must_haves.truths`):
  1. `Section.Notes` -> `notesSlideN.xml` with one `<a:p>` per note in a `<p:ph type="body">` shape, wired `slideN.xml.rels -> notesSlideN + notesMaster1` (via `notesMaster1.xml.rels`), `notesMaster1.xml` (+ rels -> theme1) + `<p:notesSz>` (already emitted since 06-03); notes parts emitted ONLY for sections with notes — proven by `TestNotesSlideBodyRuns`/`TestConditionalNotesEmission`.
  2. Notes parts fully wired into the OPC graph: `[Content_Types].xml` gains `notesSlide`/`notesMaster` Overrides, `presentation.xml.rels` gains the `notesMaster` rId — proven by `TestNotesWiringClosure`'s explicit rels/Override assertions AND by `assertStructurallyOpenable` passing with notes present in both `TestNotesWiringClosure` and every `verify_test.go` acceptance run.
  3. A comprehensive openability + position verification runs on a real multi-slide fixture (titles + paragraphs + a nested list + notes) through `ToPPTX` at BOTH 16:9 and 4:3 — proven by `TestComprehensiveAcceptance16x9`/`TestComprehensiveAcceptance4x3` (structural asserter, editable runs, notes body text, EMU positions, `<p:sldSz>`).
  4. An independent OOXML consumer smoke (LibreOffice-headless, unique `UserInstallation`, SKIP-guarded) is present as `TestAcceptanceDeckLibreOfficeSmoke` — SKIPped in this environment (`soffice` not on PATH), exercising the SKIP-guard path itself rather than a false PASS.
  5. `go list -deps ./convert/pptx/... | grep -c chromedp` = 0; `go list -deps ./press/... ./chase/... ./profiles/... | grep -c convert/pptx` = 0; `bash scripts/check-no-chromedp.sh` PASS.
- **Gate failures:** None.

## Files Created/Modified

- `convert/pptx/notes.go` — `ctNotesSlide`/`ctNotesMaster` content types; `relTypeNotesSlide`/`relTypeNotesMaster`; fixed notes-relationship rIds; `buildNotesSlide`, `buildNotesSlideRels`, `buildSlideRelsWithNotes`, `buildNotesMaster`, `buildNotesMasterRels`.
- `convert/pptx/notes_test.go` — Test-list cases 1–3: `TestNotesSlideBodyRuns`, `TestNotesSlideEmptyReturnsNil`, `TestNotesSlideEscapesUserText`, `TestNotesWiringClosure`, `TestConditionalNotesEmission` (+ `notesDoc` inline fixture, `relsTargetPresent` helper).
- `convert/pptx/verify_test.go` — Test-list cases 4–6: `TestComprehensiveAcceptance16x9`, `TestComprehensiveAcceptance4x3`, `TestAcceptanceDeckLibreOfficeSmoke` (+ `acceptanceDoc` fixture, `offExt`/`extractOffExtPairs`/`expectedShapePositions`/`assertAcceptanceDeck` helpers, reusing `unzipParts`/`assertStructurallyOpenable`/`assertSlideSize`/`decodeSlideRunTexts` from 06-03/06-04 rather than duplicating them).
- `convert/pptx/pptx.go` (modified) — `ToPPTX` gained the conditional notes-parts assembly (per-section notesSlide + once-per-deck notesMaster), in fixed slide order.
- `convert/pptx/parts_static.go` (modified, deviation — see below) — `presentationRelsXML` signature extended to a variadic `presentationRelsXML(slides []slideRef, extra ...relationship) []byte`.

## Decisions Made

- `presentationRelsXML`'s signature was extended with a variadic `extra ...relationship` parameter (backward compatible) rather than introducing a second presentation-rels builder or duplicating its fixed 5-relationship list elsewhere.
- `buildSlideRelsWithNotes` fully overrides (rather than patches/merges into) `buildSlide`'s returned rels XML for a notes-bearing section, keeping `slide.go` itself untouched.
- No `<p:notesMasterIdLst>` element was added to `presentation.xml`'s own content — only the `presentation.xml.rels` entry — matching the TRD's own wiring diagram and the existing `slideMaster1.xml.rels` precedent for declared-but-inline-unreferenced relationships.
- `verify_test.go`'s position assertions call `buildSlide`'s own internal layout helpers to compute expected EMU values (never hand-guessed pixels), per the TRD's `error_recovery` guidance.

## Deviations from Plan

**1. [Rule 3 - Auto-fix blocking] `parts_static.go` modified beyond the declared `files_modified` list**
- **Found during:** Task 1 implementation.
- **Issue:** Wiring the `notesMaster` relationship into `ppt/_rels/presentation.xml.rels` required either modifying `presentationRelsXML` (in `parts_static.go`, not listed in the TRD's `files_modified`) or duplicating its entire fixed-relationship-list logic inside `notes.go`/`pptx.go`.
- **Fix:** Extended `presentationRelsXML`'s signature to `presentationRelsXML(slides []slideRef, extra ...relationship) []byte` — a backward-compatible variadic addition; every pre-existing 1-argument call site (including 06-03's `buildTrivialDeck` in `openable_test.go`) continues to compile and pass unchanged.
- **Files modified:** `convert/pptx/parts_static.go`.
- **Commit:** `11dc9be` (part of Task 1's commit).

No other deviations — every other file touched (`notes.go`, `notes_test.go`, `pptx.go`, `verify_test.go`) matches the TRD's declared `files_modified` list exactly. `convert/pptx/testdata/.gitkeep` already existed on disk from a prior TRD and required no changes (Task 2 used inline fixtures only, per the TRD's "no generated test data" instruction).

## Issues Encountered

None. Baseline (`go build ./...`, `go test ./...`) was green before starting; every gate stayed green throughout both tasks.

## User Setup Required

None — pure stdlib (`archive/zip`, `encoding/xml`, `regexp`, `strconv`), zero new dependencies (`go.mod`/`go.sum` untouched), no external service configuration. The optional LibreOffice smoke requires `soffice` on `PATH` to actually run (absent in this environment; it SKIPs cleanly rather than failing).

## Next Objective Readiness

- **Objective 6 (convert/pptx) is COMPLETE: 5/5 TRDs** (06-01 schema-v2, 06-02 EMU utility, 06-03 OPC packager, 06-04 ToPPTX writer, 06-05 this TRD). EXP-03 is fully satisfied: hand-rolled OOXML, docmodel-sourced, editable-text-box PPTX export, verified structurally (and optionally via LibreOffice) without PowerPoint, at both 16:9 and 4:3.
- `notes.go`'s conditional-emission + 4-way-rels-wiring pattern is the template any future notes-adjacent feature should extend.
- `verify_test.go` is now the standing acceptance gate for this package — any future convert/pptx change to slide/notes layout should keep it green or deliberately update its fixtures.
- This TRD ran in a shared, multi-objective worktree (this worktree also carries completed work for Objectives 4/5/7); `convert/pptx` remains at **0 references** from press/chase/profiles and **0 chromedp** in its own dependency closure — pending orchestrator reconcile at merge.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log`.

- FOUND: convert/pptx/notes.go
- FOUND: convert/pptx/notes_test.go
- FOUND: convert/pptx/verify_test.go
- FOUND: convert/pptx/pptx.go
- FOUND: convert/pptx/parts_static.go
- FOUND: .planning/objectives/06-convert-pptx/06-05-SUMMARY.md
- FOUND commit: 11dc9be (Task 1)
- FOUND commit: d0a1766 (Task 2)

---
*Objective: 06-convert-pptx*
*Completed: 2026-07-21*
