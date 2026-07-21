---
objective: 06-convert-pptx
trd: "02"
subsystem: convert/pptx
tags: [emu, ooxml, drawingml, ecma-376, pure-math, tdd]

# Dependency graph
requires: []
provides:
  - "convert/pptx package skeleton (doc.go) -- a new top-level export tree never imported by press/chase/profiles"
  - "Inches/Points/Centimeters/Millimeters(v float64) int64 -- exact-for-whole-units, round-to-nearest-for-fractional EMU conversions, fixed to the ECMA-376 constants (914400/12700/360000/36000)"
  - "Centipoints(pt float64) int -- the DrawingML a:rPr/@sz unit (hundredths of a point), explicitly NOT EMU"
  - "SlideSize{CX,CY,Type} authoritative constants: SlideSize16x9, SlideSize4x3, NotesSize"
  - "GroupTransform{Off,Ext,ChOff,ChExt} + IdentityGroupTransform(off,ext) + MapChild(off,ext) -- the chOff/chExt child-to-slide coordinate mapping, proven identity + non-identity"
affects: [06-03-opc-packager, 06-04-topptx-writer, 06-05-notes-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single round(v float64) int64 helper (math.Round, ties away from zero) shared by every EMU conversion and by the group-transform's scale math, so there is exactly one rounding rule in the whole package, not one per function"
    - "Fixed EMU constants encoded as independent literals (emuPerInch, emuPerPoint, emuPerCentimeter, emuPerMillimeter), never derived from one another, so a typo in one can't silently corrupt another -- the 72pt==1in relationship is asserted as its own test, not relied on as a derivation"
    - "chOff/chExt transform implemented as subtract-then-scale-then-add (ChOff subtracted from child offset FIRST, then scaled, then Off added) -- the ECMA-376 CT_GroupTransform2D order, called out in the TRD as the classic bug (scaling before subtracting chOff)"

key-files:
  created:
    - convert/pptx/doc.go
    - convert/pptx/emu.go
    - convert/pptx/emu_test.go
  modified: []

key-decisions:
  - "SlideSize is a plain exported struct value set (SlideSize16x9/SlideSize4x3/NotesSize vars), not a func or map, matching the TRD's error_recovery guidance ('the Test list locks the signature; keep it minimal') and giving 06-03/06-04 a zero-argument authoritative source to read cx/cy/type from directly."
  - "GroupTransform takes/returns small Point{X,Y}/Extent{CX,CY} struct values (not four bare int64 scalars) -- readable at 06-04's call site and matches the TRD's error_recovery suggestion; MapChild is a value-receiver method on GroupTransform (transforms are pure math, no need for a pointer receiver)."
  - "No new dependency: emu.go imports only stdlib math (for math.Round). go.mod/go.sum are untouched (verified via git diff against the pre-TRD commit)."

patterns-established:
  - "Pure-math package with zero I/O, zero XML, zero deps as the locked Wave-1 foundation other convert/pptx TRDs build atop -- 06-03 sizes <p:sldSz> from SlideSize16x9/4x3/NotesSize; 06-04 calls Inches/Points/Centipoints for shape placement/font-size and IdentityGroupTransform for its first grouped shape."

requirements-completed: [EXP-03]

# Verification evidence
verification:
  gates_defined: 5
  gates_passed: 5
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 4min
completed: 2026-07-21
---

# Objective 6 TRD 02: convert/pptx EMU Conversion Utility Summary

**Independently unit-tested EMU-conversion utility (inch/point/cm/mm -> int64 EMU), 16:9/4:3/notes slide-size constants, and a chOff/chExt grouped-shape coordinate transform -- proven identity + non-identity, zero new dependencies, zero XML/I/O.**

## Performance

- **Duration:** ~4 min (TRD assignment 15:50:51Z -> Task 2 commit 15:54:16Z UTC; both commits below)
- **Started:** 2026-07-21T15:50:51Z
- **Completed:** 2026-07-21T15:54:16Z
- **Tasks:** 2/2 complete
- **Files modified:** 3 (all newly created)

## Accomplishments
- `convert/pptx/doc.go`: establishes the new top-level `pptx` package with a doc comment stating its hand-rolled, stdlib-only, no-chromedp, never-imported-by-press/chase/profiles mandate.
- `Inches`/`Points`/`Centimeters`/`Millimeters(v float64) int64`: exact-for-whole-unit, round-to-nearest-for-fractional EMU conversions against the fixed ECMA-376 constants (914400/12700/360000/36000 EMU per inch/point/cm/mm), each proven by an exact-value table test.
- `Centipoints(pt float64) int`: the DrawingML `a:rPr/@sz` unit (hundredths of a point) -- guarded by its own test (`Centipoints(44) == 4400`) documenting it is NOT EMU.
- `SlideSize{CX, CY int64; Type string}` + `SlideSize16x9`/`SlideSize4x3`/`NotesSize`: the single authoritative source for `<p:sldSz>`/`<p:notesSz>` cx/cy/type, cross-checked against `Inches(...)` (e.g. `SlideSize16x9.CX == Inches(40.0/3.0)`, `SlideSize16x9.CY == Inches(7.5)`).
- `GroupTransform{Off, Ext, ChOff, ChExt}` + `IdentityGroupTransform(off, ext)` + `(t GroupTransform) MapChild(off, ext) (Point, Extent)`: implements ECMA-376 `CT_GroupTransform2D`'s chOff/chExt child-to-slide mapping, proven for both the identity case (chOff==off, chExt==ext -> child coords unchanged, scale 1/translate 0 -- criterion 3's grouped-shape case) and a non-identity 0.5-scale case (exact integer values, isolating the formula from rounding).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Package skeleton + EMU conversions + slide-size constants | `gofmt -l convert/pptx/*.go && go build ./... && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Inches\|Points\|Centi\|Milli\|Centipoint\|SlideSize' -v && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: chOff/chExt group-transform (identity + general) | `gofmt -l convert/pptx/emu.go && go vet ./convert/pptx/... && go test ./convert/pptx/ -run 'Transform\|Group\|ChOff\|Identity' -v && go test ./convert/pptx/ && bash scripts/check-no-chromedp.sh` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Package skeleton + EMU conversions + slide-size constants** - `c5eb144` (feat)
2. **Task 2: chOff/chExt group-transform utility (identity + general)** - `6de5619` (feat)

_Note: both tasks are `tdd="true"`; RED (compile failure against undefined `Inches`/`Points`/.../`SlideSize` for Task 1, and undefined `Point`/`Extent`/`GroupTransform`/`IdentityGroupTransform` for Task 2) confirmed before each GREEN implementation -- see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./convert/pptx/...` (and `go vet ./...`) | 0 | PASS |
| test | `go test ./convert/pptx/...` (and `go test ./...`, full repo) | 0 | PASS |
| gofmt | `gofmt -l convert/pptx/*.go` | 0 (no output) | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| isolation | `go list -deps ./press/... ./chase/... ./profiles/... \| grep -c convert/pptx` | 0 matches | PASS |
| addlicense | `addlicense -check convert/pptx/*.go` | 0 | PASS |
| no new deps | `git diff HEAD~2 -- go.mod go.sum` | (empty diff) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./convert/pptx/... -v` | 1 (compile failure: undefined Inches/Points/Centimeters/Millimeters/Centipoints/SlideSize/SlideSize16x9) | FAIL (correct) |
| GREEN (Task 1) | `go test ./convert/pptx/ -run 'Inches\|Points\|Centi\|Milli\|Centipoint\|SlideSize' -v` | 0 (10 subtests across 7 top-level tests, all PASS) | PASS (correct) |
| RED (Task 2) | `go test ./convert/pptx/ -run 'Transform\|Group\|ChOff\|Identity' -v` | 1 (compile failure: undefined Point/Extent/GroupTransform/IdentityGroupTransform) | FAIL (correct) |
| GREEN (Task 2) | `go test ./convert/pptx/ -run 'Transform\|Group\|ChOff\|Identity' -v` | 0 (2/2 PASS) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 -- no deviations; the design converged on the first implementation of both tasks and every test passed on the first GREEN run.
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 06-02-TRD.md frontmatter: independently-unit-tested EMU utility with exact-value tests; authoritative 16:9/4:3/notes slide-size constants; tested chOff/chExt identity + non-identity transform; centipoint font-size helper documented distinct from EMU; convert/pptx isolated from press/chase/profiles at 0 references).
- **Gate failures:** None.

## Files Created/Modified
- `convert/pptx/doc.go` - MIT header + package doc comment establishing `package pptx`'s hand-rolled/stdlib-only/no-chromedp/never-imported mandate.
- `convert/pptx/emu.go` - `emuPerInch`/`emuPerPoint`/`emuPerCentimeter`/`emuPerMillimeter` constants; `round` helper; `Inches`/`Points`/`Centimeters`/`Millimeters`/`Centipoints` conversion funcs; `SlideSize` type + `SlideSize16x9`/`SlideSize4x3`/`NotesSize` vars; `Point`/`Extent` types; `GroupTransform` type + `IdentityGroupTransform` constructor + `MapChild` method.
- `convert/pptx/emu_test.go` - Test-list cases 1-7: `TestInches`, `TestPoints` (incl. 72pt=1in cross-check), `TestCentimeters`, `TestMillimeters`, `TestCentipoints`, `TestSlideSizeConstants`, `TestSlideSizeCrossCheckAgainstInches`, `TestGroupTransformIdentity`, `TestGroupTransformNonIdentityScale`.

## Decisions Made
- `SlideSize`/`Point`/`Extent` are small exported struct value types (not scalar quadruples or maps) -- matches the TRD's own error_recovery guidance and keeps 06-03/06-04's call sites readable.
- `round()` is one shared helper (math.Round, ties away from zero) used by every conversion AND by `MapChild`'s scale math -- a single deterministic rounding rule for the whole package, rather than one per function.
- The 72pt==1in and slide-size-vs-Inches relationships are asserted as their own tests, never relied upon as a derivation between constants (per the TRD's anti_patterns guidance) -- each EMU constant (`emuPerInch`, `emuPerPoint`, etc.) is an independent literal.

## Deviations from Plan

None - 06-02-TRD.md executed exactly as written. Both tasks' RED phases failed as expected (compile errors against undefined identifiers) and both GREEN phases passed on the first implementation attempt with no auto-fix cycles needed.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required; pure stdlib, zero new dependencies.

## Next Objective Readiness
- `convert/pptx.Inches`/`Points`/`Centipoints` and `SlideSize16x9`/`SlideSize4x3`/`NotesSize` are ready for 06-03 (OPC zip packager) to size `<p:sldSz>`/`<p:notesSz>` from directly.
- `convert/pptx.IdentityGroupTransform`/`MapChild` are ready for 06-04 (`ToPPTX`) to place shape positions/extents and emit its first grouped shape via the proven identity case; the non-identity formula is locked for whenever a future TRD needs true nested-group scaling.
- This TRD ran in parallel (wave 1) with 06-01 (chase/model + press/math) -- no file overlap; `convert/pptx` is a brand-new package tree, confirmed at 0 references from `press/`, `chase/`, `profiles/`.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: convert/pptx/doc.go
- FOUND: convert/pptx/emu.go
- FOUND: convert/pptx/emu_test.go
- FOUND: .planning/objectives/06-convert-pptx/06-02-SUMMARY.md
- FOUND commit: c5eb144 (Task 1)
- FOUND commit: 6de5619 (Task 2)

---
*Objective: 06-convert-pptx*
*Completed: 2026-07-21*
