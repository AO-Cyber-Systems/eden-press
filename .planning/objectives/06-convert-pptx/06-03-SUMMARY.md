---
objective: 06-convert-pptx
trd: "03"
subsystem: convert/pptx
tags: [opc, ooxml, zip, determinism, content-types, clrmap, fmtscheme, openability, tdd]

# Dependency graph
requires:
  - objective: 06-convert-pptx
    trd: "02"
    provides: "SlideSize{CX,CY,Type} / SlideSize16x9 / SlideSize4x3 / NotesSize constants consumed by presentationXML's <p:sldSz>/<p:notesSz>"
provides:
  - "package.go: deterministic OPC zip assembler -- part{name,content}, explicit ordered []part (never a map), fixed FileHeader.Modified (1980-01-01), zip.Store -- buildZip(parts) []byte, []error"
  - "contenttypes.go: [Content_Types].xml Default(rels,xml)+Override manifest builder (buildContentTypesXML) and a parser/coverage-checker (parseContentTypesXML/.covers) reused by the openability asserter"
  - "parts_static.go: every invariant OPC part builder in the minimal graph -- _rels/.rels, docProps/core+app, presentation.xml(+rels), presProps/viewProps/tableStyles, theme1 (12-attr clrMap consumer via slideMaster1XML, 3-entry-per-list fmtScheme in theme1XML), slideMaster1(+rels), slideLayout1(+rels) -- plus the N-slide-ready slideRef/presentationXML/presentationRelsXML plumbing"
  - "openable_test.go: the reusable structural openability asserter (assertStructurallyOpenable) -- content-types coverage + full .rels Target-resolution + r:id-resolution closure -- proven on a trivial hardcoded title-box deck at BOTH 16:9 and 4:3"
affects: [06-04-topptx-writer, 06-05-notes-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fixed r:id constants (rIDMaster1, rIDSlideLayout1, etc.) declared once and shared by BOTH a part's own content call site and its sibling .rels builder call site, so the '#1 unresolved r:id bug' the TRD warns about is structurally prevented at the Go level -- independently re-verified by parsing the emitted XML/rels bytes back out in tests, not just trusted from construction"
    - "slideRef{RelID,Target} slice-based plumbing in presentationXML/presentationRelsXML, exercised at N=1 in this TRD but requiring no signature change for 06-04's N model-driven slides"
    - "Trivial deck's slide1.xml/slide1.xml.rels/buildTrivialDeck kept TEST-ONLY (openable_test.go), never in parts_static.go -- keeps packaging proof isolated from any docmodel dependency, per the TRD's explicit anti-pattern"
    - "assertStructurallyOpenable reuses production parseContentTypesXML/.covers (contenttypes.go) alongside test-only parseRelsXML/extractRIDs (parts_static_test.go) -- the same reusable harness 06-05 extends to the full model-driven + notes deck"
    - "OPC relative-Target resolution (resolveRelTarget/relsOwnerDir) implemented per the OPC convention that a .rels file's Target paths are relative to the OWNING PART's directory, not the .rels file's own directory -- handles both '../'-relative and leading-'/'-absolute forms"

key-files:
  created:
    - convert/pptx/package.go
    - convert/pptx/package_test.go
    - convert/pptx/contenttypes.go
    - convert/pptx/parts_static.go
    - convert/pptx/parts_static_test.go
    - convert/pptx/openable_test.go
    - convert/pptx/testdata/.gitkeep
  modified: []

key-decisions:
  - "presentation.xml's <p:sldSz type=\"...\"> attribute is emitted conditionally (omitted entirely when SlideSize.Type == \"\", as for NotesSize, which has no Type) -- presentationXML's sldSzType string is built once and only applied to the requested slide size, never notesSz."
  - "The package-level _rels/.rels file is exempt from the owning-part r:id cross-check in assertStructurallyOpenable (it has no single 'owning part' with r:id attributes of its own) -- every OTHER .rels file IS cross-checked both for Target existence and r:id-declaration completeness against its owning part's actual content."
  - "clrMap's 12 required attributes and fmtScheme's 3-entries-per-list are emitted as static, verbatim boilerplate strings/blocks (not generated), matching the TRD's anti-pattern guidance against a partial clrMap or variable-length fmtScheme list."

patterns-established:
  - "Deterministic OPC zip assembly (fixed Modified + zip.Store + explicit ordered []part, never a map) as the locked foundation 06-04's real content mapping fills without touching packaging plumbing."
  - "Structural, no-PowerPoint-required openability proof (unzip + content-types coverage + r:id/.rels closure) as the CI gate, with an optional SKIP-guarded LibreOffice-headless smoke as an independent-consumer bonus check."

requirements-completed: [EXP-03]

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 9min
completed: 2026-07-21
---

# Objective 6 TRD 03: convert/pptx Deterministic OPC Packager Summary

**Hand-rolled, byte-deterministic OPC zip packager (fixed Modified + zip.Store + explicit part ordering) emitting the complete "boring but complete" minimal PresentationML part graph -- 12-attr clrMap and 3-entry fmtScheme done right -- proven structurally openable (no PowerPoint required) on a trivial title-box deck at both 16:9 and 4:3.**

## Performance

- **Duration:** ~9 min (Task 1 commit 12:23:30 -> gofmt-fix commit 12:32:20, local time; all commits below)
- **Started:** 2026-07-21T16:23:30Z
- **Completed:** 2026-07-21T16:32:20Z
- **Tasks:** 3/3 complete
- **Files modified:** 7 (all newly created)

## Accomplishments
- `package.go`: `part{name,content}` + `buildZip([]part) ([]byte, error)` -- every entry uses `zip.Store` and a fixed `fixedModified = 1980-01-01` timestamp, iterated over an explicit `[]part` slice (never a map), so two builds of the identical part set are byte-identical.
- `contenttypes.go`: `buildContentTypesXML([]contentTypeOverride) []byte` (fixed `rels`/`xml` Defaults + ordered Overrides) and `parseContentTypesXML`/`.covers` -- the coverage-checking reader reused by the openability asserter.
- `parts_static.go`: every invariant part in the minimal OPC graph -- `_rels/.rels`, `docProps/core.xml`, `docProps/app.xml`, `ppt/presentation.xml`(+rels), `ppt/presProps.xml`, `ppt/viewProps.xml`, `ppt/tableStyles.xml`, `ppt/theme/theme1.xml` (full 12-color clrScheme + fontScheme + exactly-3-entry-per-list fmtScheme), `ppt/slideMasters/slideMaster1.xml`(+rels, mandatory 12-attr `<p:clrMap>`), `ppt/slideLayouts/slideLayout1.xml`(+rels) -- with fixed, shared r:id constants preventing part/.rels drift.
- `presentationXML(size SlideSize, slides []slideRef)` / `presentationRelsXML(slides []slideRef)`: N-slide-ready plumbing (exercised at N=1 here) sizing `<p:sldSz>`/`<p:notesSz>` directly from 06-02's `SlideSize16x9`/`SlideSize4x3`/`NotesSize` constants.
- `openable_test.go`: `buildTrivialDeck(size SlideSize) ([]byte, error)` assembles the full part graph plus a test-only hardcoded title `<p:sp>` slide1.xml; `assertStructurallyOpenable` -- the reusable, no-PowerPoint-required CI gate -- asserts content-types coverage (no missing/orphan overrides) and full `.rels` Target-existence + r:id-resolution closure across every part; `TestTrivialDeckOpenable16x9`/`TestTrivialDeckOpenable4x3` prove the SAME build code path structurally openable at both aspect ratios (criterion 4); `TestTrivialDeckLibreOfficeSmoke` runs an optional, SKIP-guarded `soffice --headless --convert-to pdf` independent-consumer proof (skipped cleanly in this environment -- `soffice` not on PATH).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Deterministic OPC zip packager + Content-Types manifest | `gofmt -l convert/pptx/package.go convert/pptx/contenttypes.go && go build ./... && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Determinism\|ContentTypes\|Coverage' -v && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: Complete static boilerplate part set (clrMap + fmtScheme) | `gofmt -l convert/pptx/parts_static.go && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Rels\|ClrMap\|FmtScheme\|Resolution' -v && go test ./convert/pptx/ && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 3: Trivial static deck + structural openability asserter (16:9 + 4:3) | `go test ./convert/pptx/ -run 'Openable\|StaticDeck\|Aspect\|Smoke' -v && go build ./... && go vet ./... && go test ./... && bash scripts/check-no-chromedp.sh && test "$(go list -deps ./press/... ./chase/... ./profiles/... \| grep -c convert/pptx)" = "0"` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Deterministic OPC zip packager + Content-Types manifest** - `eb2ada9` (feat)
2. **Task 2: Complete static boilerplate part set (clrMap + fmtScheme done right)** - `5a98584` (feat)
3. **Task 3: Trivial static deck + structural openability asserter (16:9 + 4:3)** - `b209398` (feat)
4. **Post-task fix: gofmt-align parts_static_test.go builders map literal** - `95fe34c` (fix) -- see Deviations below

_Note: Tasks 1 and 2 are `tdd="true"`; RED (compile failure against undefined `part`/`buildZip`/`fixedModified`/`contentTypeOverride`/`buildContentTypesXML`/`parseContentTypesXML` for Task 1, and undefined `relationship`/`relsDocXML`/`slideRef`/`rIDSlide1`/`presentationXML`/`presentationRelsXML`/`slideMaster1XML`/`slideMaster1RelsXML` for Task 2) confirmed before each GREEN implementation -- see TDD Evidence below. Task 3 has no `tdd` attribute and the TRD's `type: standard` frontmatter means its effective TDD flag is FALSE (per the executor's effective-tdd-flag rules); it was executed and verified as a standard (non-TDD) task -- its `<verify>` command was run once, after implementation, and passed on the first attempt._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (full repo, all packages) | 0 | PASS |
| gofmt | `gofmt -l convert/pptx/*.go` | 0 (no output, after gofmt-fix commit) | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| isolation | `go list -deps ./press/... ./chase/... ./profiles/... \| grep -c convert/pptx` | 0 matches | PASS |
| addlicense | `addlicense -check convert/pptx/*.go` | 0 | PASS |
| no new deps | `git diff -- go.mod go.sum` (empty across all 4 commits) | (empty diff) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./convert/pptx/ -run 'Determinism\|ContentTypes\|Coverage' -v` | 1 (compile failure: undefined part/buildZip/fixedModified/contentTypeOverride/buildContentTypesXML/parseContentTypesXML) | FAIL (correct) |
| GREEN (Task 1) | `go test ./convert/pptx/ -run 'Determinism\|ContentTypes\|Coverage' -v` | 0 (4/4 PASS: TestBuildZipDeterminism, TestBuildZipDeterminismMetadata, TestContentTypesCoverage, TestContentTypesCoverageNegative) | PASS (correct) |
| RED (Task 2) | `go test ./convert/pptx/ -run 'Rels\|ClrMap\|FmtScheme\|Resolution' -v` | 1 (compile failure: undefined relationship/relsDocXML/slideRef/rIDSlide1/presentationXML/presentationRelsXML/slideMaster1XML/slideMaster1RelsXML) | FAIL (correct) |
| GREEN (Task 2) | `go test ./convert/pptx/ -run 'Rels\|ClrMap\|FmtScheme\|Resolution' -v` | 0 (5/5 PASS: TestRelsResolutionPresentation, TestRelsResolutionSlideMaster, TestRelsTargetsAreWellFormed, TestClrMap12Attrs, TestFmtSchemeThreeEntriesPerList) | PASS (correct) |
| Task 3 (non-TDD, standard verify) | `go test ./convert/pptx/ -run 'Openable\|StaticDeck\|Aspect\|Smoke' -v` | 0 (2 PASS + 1 SKIP: TestTrivialDeckOpenable16x9, TestTrivialDeckOpenable4x3, TestTrivialDeckLibreOfficeSmoke skipped -- soffice absent) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 -- a `gofmt`-alignment drift in `parts_static_test.go`'s `builders` map literal (introduced when the map was authored in the Task 2 commit) was caught during the final full-repo gate sweep after Task 3 landed, fixed with `gofmt -w`, re-verified (build/vet/test all still green), and committed separately (`95fe34c`) since Task 2's commit had already been made. See Deviations below.
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 06-03-TRD.md frontmatter: deterministic zip with fixed Modified/zip.Store/explicit ordering; the full boring-but-complete minimal OPC part graph emitted from day one; 12-attr clrMap + exactly-3-entry fmtScheme lists; trivial single-slide deck structurally openable at both 16:9 and 4:3; `<p:sldSz>`/`<p:notesSz>` driven by 06-02's SlideSize constants).
- **Gate failures:** None remaining (the one gofmt drift found was fixed and re-verified before this SUMMARY was written).

## Files Created/Modified
- `convert/pptx/package.go` - `part{name,content}`, `fixedModified` (1980-01-01), `buildZip([]part) ([]byte, error)` (zip.Store, explicit ordering).
- `convert/pptx/package_test.go` - `TestBuildZipDeterminism`, `TestBuildZipDeterminismMetadata`, `TestContentTypesCoverage`, `TestContentTypesCoverageNegative`.
- `convert/pptx/contenttypes.go` - `xmlDeclaration`, `marshalPart`, `ctDefault`/`ctOverride`/`ctTypes` XML structs, `buildContentTypesXML`, `parsedContentTypes`/`parseContentTypesXML`/`.covers`.
- `convert/pptx/parts_static.go` - namespace/relationship-type/content-type/r:id constants; `relationship`/`relsDocXML` + `buildRelsXML`; `rootRelsXML`/`docPropsCoreXML`/`docPropsAppXML`/`presPropsXML`/`viewPropsXML`/`tableStylesXML`/`theme1XML` (12-color clrScheme + 3-entry fmtScheme); `clrMapAttrs` (12-attr); `slideMaster1XML`/`slideMaster1RelsXML`/`slideLayout1XML`/`slideLayout1RelsXML`; `slideRef`/`presentationXML`/`presentationRelsXML`.
- `convert/pptx/parts_static_test.go` - `extractRIDs`/`parseRelsXML`/`assertRIDsResolve` helpers; `TestRelsResolutionPresentation`, `TestRelsResolutionSlideMaster`, `TestRelsTargetsAreWellFormed`, `TestClrMap12Attrs`, `TestFmtSchemeThreeEntriesPerList`, `TestStaticPartsProduceWellFormedXML`.
- `convert/pptx/openable_test.go` - `slide1XML`/`slide1RelsXML`/`buildTrivialDeck`; `unzipParts`/`resolveRelTarget`/`relsOwnerDir`/`assertStructurallyOpenable`; `assertSlideSize`/`assertSlide1HasTitleText`; `TestTrivialDeckOpenable16x9`, `TestTrivialDeckOpenable4x3`, `TestTrivialDeckLibreOfficeSmoke`.
- `convert/pptx/testdata/.gitkeep` - placeholder holding the (currently empty) golden-fixture directory for future golden-byte tests.

## Decisions Made
- `<p:sldSz>`'s `type` attribute is emitted only when `SlideSize.Type != ""` -- keeps `presentationXML` a single code path for both named slide sizes AND `NotesSize` (which carries no `Type`).
- The package-level `_rels/.rels` is exempt from the "owning-part r:id cross-check" step of `assertStructurallyOpenable` (it has no single owning part with its own r:id-bearing content) -- every other `.rels` file IS cross-checked both for Target-existence and r:id-declaration completeness.
- The trivial deck's `slide1.xml`/`slide1.xml.rels`/`buildTrivialDeck` helper live ONLY in `openable_test.go` (test-only), never in `parts_static.go`, per the TRD's anti-pattern guidance -- keeps the packaging proof isolated from any docmodel dependency; 06-04 replaces this hardcoded slide with N model-driven slides while reusing all of `parts_static.go`'s builders unchanged.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] gofmt-alignment drift in `parts_static_test.go`'s `builders` map literal**
- **Found during:** Final full-repo gate sweep, after Task 3's commit (`b209398`) landed
- **Issue:** `gofmt -l convert/pptx/*.go` flagged `parts_static_test.go` -- the `builders` map literal's value-column alignment (introduced when the map was originally authored in the Task 2 commit, `5a98584`) had drifted from what `gofmt -w` produces, so the already-committed file was not gofmt-clean.
- **Fix:** Ran `gofmt -w convert/pptx/parts_static_test.go`; re-verified `go build ./...`, `go vet ./...`, and `go test ./convert/pptx/...` all still pass (whitespace-only change, no behavior difference).
- **Files modified:** `convert/pptx/parts_static_test.go`
- **Verification:** `gofmt -l convert/pptx/*.go` now empty; full `go test ./...` still green.
- **Commit:** `95fe34c`

## Issues Encountered
None beyond the one auto-fixed gofmt-alignment deviation above, resolved and re-verified before this SUMMARY was written.

## User Setup Required
None - no external service configuration required. The optional LibreOffice-headless smoke test (`TestTrivialDeckLibreOfficeSmoke`) is fully SKIP-guarded and was confirmed to skip cleanly in this environment (`soffice` not on PATH); installing LibreOffice is optional, not required, for this TRD's gates to pass.

## Next Objective Readiness
- `convert/pptx`'s deterministic zip assembler (`package.go`) and the complete static part-builder set (`parts_static.go`) are ready for 06-04 (`ToPPTX`) to fill with N real, model-driven slides -- `slideRef`/`presentationXML`/`presentationRelsXML` already accept a slice of slides, so 06-04 extends this plumbing without changing it.
- `openable_test.go`'s `assertStructurallyOpenable` is ready for 06-05 to extend to the full model-driven + notes deck, exactly as the TRD's `key_links` anticipated.
- `convert/pptx` remains confirmed at 0 references from `press/`, `chase/`, `profiles/` (isolation gate), and `check-no-chromedp.sh` stays green; no new dependency was added across any of this TRD's 4 commits (`go.mod`/`go.sum` diff empty).

## Self-Check: PASSED

All claimed files confirmed present on disk; all four commit hashes confirmed present in `git log --oneline --all`.

- FOUND: convert/pptx/package.go
- FOUND: convert/pptx/package_test.go
- FOUND: convert/pptx/contenttypes.go
- FOUND: convert/pptx/parts_static.go
- FOUND: convert/pptx/parts_static_test.go
- FOUND: convert/pptx/openable_test.go
- FOUND: convert/pptx/testdata/.gitkeep
- FOUND: .planning/objectives/06-convert-pptx/06-03-SUMMARY.md
- FOUND commit: eb2ada9 (Task 1)
- FOUND commit: 5a98584 (Task 2)
- FOUND commit: b209398 (Task 3)
- FOUND commit: 95fe34c (post-task gofmt fix)

---
*Objective: 06-convert-pptx*
*Completed: 2026-07-21*
