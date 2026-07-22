---
objective: 08-math-autofit
trd: "06"
subsystem: press-html-autofit
tags: [go, cli, html, auto-fit, js-removal, marp-core, flutter, criterion-4]

# Dependency graph
requires:
  - objective: 08-math-autofit
    provides: "08-05's NOTICE (Marp Core entry, STIX Two Math WOFF2 clause) — serialization-only dependency; this TRD edits the same NOTICE entry's browser-fit.js clause, functionally independent of the font work"
provides:
  - "Plain HTML/PDF output that NEVER ships viewer-side auto-fit JavaScript — the web half of criterion 4's resolved decision (auto-fit is Flutter-only)"
  - "TestAssembleHTMLNoViewerSideAutoFitJS (cmd/eden-press/htmldoc_test.go) — the permanent acceptance gate: default assembleHTML output has zero <script>, while both CORE-09 markers (data-auto-scaling=\"fit\", marp-fit-shrink) remain present"
  - "press/autofit.go's CORE-09 marker emission, confirmed UNCHANGED and still green — the exact input surface 08-07's Flutter TextPainter fit consumes"
affects:
  - "08-07 (Flutter-native auto-fit) — its sole input (the <!--fit-->/shrink markers) is proven still emitted and byte-for-byte unchanged by this TRD"
  - "cmd/eden-press-export/main.go and AGENTS.md — doc-comment-only accuracy fixes (see Deviations) so no doc anywhere describes a flag/script that no longer exists"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mechanical flag/splice removal ordered to keep the build green at every commit: callers (Task 1) removed BEFORE the accessor (Task 2), so no commit ever has a dangling press.BrowserFitJS/themes.BrowserFitJS reference"
    - "Zero-JS acceptance gate scoped to the DEFAULT path only: TestAssembleHTMLNoViewerSideAutoFitJS asserts on htmlDocOptions{} (no InjectScripts), leaving the separate, legitimate watch/serve SSE-reload script untouched and unasserted-against"
    - "Marker-emission trigger-shape substitution in tests: Objective 8's math converter (08-01/08-03, out of scope here) now converts $...$ into real <math> MathML BEFORE press/autofit.go's paragraph-text-shape check runs, so a fenced code block (not a $...$ block) is the reliable shrink-marker trigger for a test written after that converter landed"

key-files:
  created: []
  modified:
    - cmd/eden-press/flags.go
    - cmd/eden-press/htmldoc.go
    - cmd/eden-press/htmldoc_test.go
    - cmd/eden-press/convert_test.go
    - cmd/eden-press/preview.go
    - cmd/eden-press/serve.go
    - cmd/eden-press/watch.go
    - cmd/eden-press/format.go
    - press/themecss_test.go
    - press/themes/themes.go
    - press/themes/themes_test.go
    - press/autofit.go
    - cmd/eden-press-export/main.go
    - themes/embed.go
    - NOTICE
    - AGENTS.md
  deleted:
    - press/browserjs.go
    - themes/browser-fit.js

key-decisions:
  - "press/autofit.go and cmd/eden-press-export/main.go each carried a stale doc-comment-only reference to the removed script/flag; both were rephrased (zero functional/logic change) rather than left inaccurate, since the TRD's own repo-wide grep gate (`! grep -rn \"BrowserFitJS|browser-fit.js\" --include=*.go .`) would otherwise fail on press/autofit.go's leftover comment text, and leaving a stale --auto-fit-script mention in cmd/eden-press-export/main.go would contradict the TRD's own success criteria. See Deviations."
  - "The $...$-math shrink-marker test case was swapped for a fenced code block after diagnosis via a disposable debug test: Objective 8's math converter now renders $...$ as real MathML before press/autofit.go's text-shape check runs, so a fenced code block is the converter-unaffected trigger shape. This is an observation about a sibling TRD's effect, not a regression in this TRD's scope."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: true
  blockers: none

# Metrics
duration: ~25min
completed: 2026-07-22
---

# Objective 08 TRD 06: Remove Viewer-Side JS Auto-Fit from the HTML/CLI Path Summary

**The `--auto-fit-script` flag, its `AutoFitScript` splice, and the entire vendored `press.BrowserFitJS`/`themes/browser-fit.js` surface are gone — plain HTML/PDF output now NEVER ships viewer-side auto-fit JavaScript, proven by a permanent gate test asserting zero `<script>` in the default assembled document while both `<!--fit-->`/shrink CORE-09 markers remain present and unchanged, ready to feed 08-07's Flutter-native TextPainter fit.**

## What was built

### Task 1 — Remove the flag + splice + four caller reads (commit 391bfe6)
- `cmd/eden-press/flags.go`: deleted the `--auto-fit-script` `f.Bool(...)` registration and its doc comment.
- `cmd/eden-press/htmldoc.go`: deleted the `AutoFitScript bool` field from `htmlDocOptions` and the `if opts.AutoFitScript { ...press.BrowserFitJS()... }` splice branch from `assembleHTML`. `InjectScripts []string` (the watch/serve SSE live-reload seam) and the `press` import (still needed for `press.Output`) were both kept intact.
- Removed the `AutoFitScript: cfg.Bool("auto-fit-script")` read from `preview.go`, `format.go`, `serve.go`, `watch.go` — `serve.go`/`watch.go` retain their `InjectScripts` assignment for the reload client.
- Removed `TestAssembleHTMLAutoFitScript` (`htmldoc_test.go`) and `TestRunConvertAutoFitScript` (`convert_test.go`).

### Task 2 — Delete the BrowserFitJS surface + add the no-JS-ships gate + NOTICE (commits 66477ec, 53d8238)
- Deleted `press/browserjs.go` (the `press.BrowserFitJS` re-export) and `themes/browser-fit.js` (the vendored Marp `lib/browser.js`).
- `themes/embed.go`: removed the `//go:embed browser-fit.js` directive, its `var`, and `BrowserFitJS()`.
- `press/themes/themes.go`: removed the `BrowserFitJS()` accessor.
- Removed the two now-dangling tests: `TestBrowserFitJSReexport` (`press/themecss_test.go`, plus its now-unused `press/themes` import) and `TestBrowserFitJS` (`press/themes/themes_test.go`).
- `NOTICE`: dropped the browser-fit.js clause from the Marp Core entry, kept the default/gaia/uncover.css verbatim attribution untouched, and added a bullet documenting the removal + that auto-fit is now Flutter-only.
- Added `TestAssembleHTMLNoViewerSideAutoFitJS` (`cmd/eden-press/htmldoc_test.go`): renders a deck with a fitting header (`# <!--fit-->`) and a fenced code block through the default `assembleHTML(out, htmlDocOptions{})` path, asserting zero `<script>` while `data-auto-scaling="fit"` and `marp-fit-shrink` both remain present in `out.HTML`.
- Doc-comment-only fixes (see Deviations): `press/autofit.go`'s top-of-file comment (stale "browser-fit.js" mentions) and `cmd/eden-press-export/main.go`'s `registerExportFlags` doc comment (stale "--auto-fit-script" mention) — zero change to any executable code in either file.

### Task 3 — Update AGENTS.md (commit 6e65a95)
- Rewrote the `html` format table row: it now states plain HTML is ALWAYS zero-`<script>` (no auto-fit splice option exists), that `<!--fit-->`/shrink markers are still emitted but inert on the web path, and that auto-fit is Flutter-only via the Dart binding's native TextPainter fit (08-07).
- Confirmed (no code change) `press/autofit.go` still emits both markers and `press/autofit_test.go` is green.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: flag + splice removal | `go build ./... && go vet ./... && go test ./cmd/eden-press/... -v && ! grep -rn "auto-fit-script\|AutoFitScript" cmd/eden-press/*.go && gofmt -l cmd/eden-press/*.go` | 0 | PASS |
| 2: BrowserFitJS removal + no-JS gate | `go build ./... && go vet ./... && go test ./... && go test ./conformance/... && ! grep -rn "BrowserFitJS\|browser-fit.js" --include=*.go . && test ! -f themes/browser-fit.js && test ! -f press/browserjs.go && ! grep -qi "browser-fit.js" NOTICE && bash scripts/check-no-chromedp.sh && bash scripts/check-cli-imports.sh && go test ./profiles/slides/ -run TestGrepGate && addlicense ... -check .` | 0 | PASS |
| 3: AGENTS.md update | `! grep -n "auto-fit-script" AGENTS.md && go test ./press/ -run 'TestAutofit\|TestAutoFit' -v && grep -qi "flutter" AGENTS.md` | 0 | PASS |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok, incl. press/, press/themes/, cmd/eden-press/) |
| test (conformance) | `go test ./conformance/...` | 0 | PASS (corpus, cssdiff, htmldiff, runner ok) |
| gofmt | `gofmt -l .` | 0 | PASS (empty output) |
| repo grep gate (no BrowserFitJS/browser-fit.js/auto-fit-script in *.go) | `grep -rn "BrowserFitJS\|browser-fit.js\|auto-fit-script" . --include=*.go` | 1 (no matches) | PASS |
| deleted files stay deleted | `test ! -f themes/browser-fit.js && test ! -f press/browserjs.go` | 0 | PASS |
| NOTICE clean | `! grep -qi "browser-fit.js" NOTICE` | 0 | PASS |
| check-no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| check-cli-imports | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -ignore 'convert/chrome/fonts/**' -ignore 'internal/latex2mathml/**' -check .` | 0 | PASS |
| zero-JS acceptance gate | `go test ./cmd/eden-press/ -run TestAssembleHTMLNoViewerSideAutoFitJS -v` | 0 | PASS |
| marker preservation (press/autofit.go untouched functionally) | `go test ./press/ -run 'TestAutofit\|TestAutoFit' -v` | 0 | PASS (4/4: FitHeaderMarker, NormalHeadingCarriesNoFitMarker, ShrinkMarkersOnCodeAndMathBlocks, OptionIsComposableGoldmarkOption) |
| AGENTS.md accuracy | `! grep -n "auto-fit-script" AGENTS.md && grep -qi "flutter" AGENTS.md` | 0 | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - doc accuracy] Fixed stale "browser-fit.js" doc-comment mentions in press/autofit.go**
- **Found during:** Task 2's repo-wide grep gate (`! grep -rn "BrowserFitJS\|browser-fit.js" --include=*.go .`), which failed on two literal "browser-fit.js" mentions in this file's top-of-file package doc comment.
- **Issue:** The TRD's anti-pattern explicitly says "do NOT touch press/autofit.go" (protecting the marker-emission LOGIC that feeds 08-07). However its own comment text described the now-deleted browser-fit.js as the marker consumer, which is both stale and would fail the TRD's own mandatory gate.
- **Fix:** Rephrased the doc comment ONLY — zero change to package declaration, imports, constants, or any function/struct body — to state that the markers originally fed a now-removed viewer-side JS helper, and now serve as the Flutter binding's fit signal instead (inert on the plain-HTML web path). `press/autofit_test.go` was not touched at all (it had zero references to begin with) and stays green, confirming the marker-emission logic itself is byte-for-byte unchanged.
- **Files modified:** `press/autofit.go` (comment only)
- **Commit:** `66477ec`

**2. [Rule 2 - doc accuracy] Fixed stale "--auto-fit-script" doc-comment mention in cmd/eden-press-export/main.go**
- **Found during:** The same repo-wide grep gate, which found one remaining hit outside the TRD's `files_modified` list: `registerExportFlags`'s doc comment claimed "There is no ... --auto-fit-script here."
- **Issue:** This binary was never in the TRD's file list, but the literal substring would fail the mandatory zero-hits grep gate, and the comment's premise (comparing itself to a flag that no longer exists anywhere) was now inaccurate.
- **Fix:** Rephrased to drop the literal flag name while stating the equivalent, now-accurate fact: this exporter has no theme-set/config flags, and auto-fit was never one of its flags either way (it is Flutter-only, 08-06/08-07). Zero functional change.
- **Files modified:** `cmd/eden-press-export/main.go` (comment only)
- **Commit:** `66477ec`

### Observations (not deviations, no fix applied)

- While writing `TestAssembleHTMLNoViewerSideAutoFitJS`, a `$x^2$`-style math block did NOT trigger the `marp-fit-shrink` wrapper as expected, because Objective 8's math converter (08-01/08-03, out of scope for this TRD) now converts `$...$` into real `<math>` MathML BEFORE `press/autofit.go`'s paragraph-text-shape check runs. Diagnosed via a disposable debug test (created and removed, never committed). The test was written against a fenced code block instead — the other, converter-unaffected shrink-trigger shape `press/autofit.go` already supports — and passes cleanly. No code in this TRD's scope changed as a result.
- The repo-level `CLAUDE.md` (Hard Rule #2) still lists "the browser-fit script" alongside the 3 vendored Marp theme CSS files as carrying Marp's original copyright/license. This is now stale (the script no longer ships) but `CLAUDE.md` is outside this TRD's `files_modified` list, isn't scanned by any of this TRD's grep gates, and editing it wasn't requested by the coordinator. Left untouched; flagged here as an optional follow-up for a future docs pass.

No other deviations — Tasks 1–3 landed exactly as the TRD specified.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0 (the two doc-comment fixes were Rule-2 auto-fixes, not fix-attempt cycles against a failing gate)
- Must-haves verified: 4/4 — (1) flag/field/splice/four caller reads removed; (2) `<!--fit-->`/shrink markers unchanged in `press/autofit.go`, confirmed via green `TestAutofit*`/`TestAutoFit*`; (3) BrowserFitJS surface fully removed (both files deleted, both accessors removed, both dangling tests removed) and the zero-`<script>` gate test passes; (4) NOTICE + AGENTS.md updated accordingly.
- Gate failures: None
- Blockers: None

## Commits

- `391bfe6` feat(08-06): remove --auto-fit-script flag + AutoFitScript splice + all four caller reads
- `66477ec` feat(08-06): remove BrowserFitJS + browser-fit.js (no JS ships in HTML)
- `53d8238` test(08-06): assert default HTML output is script-free
- `6e65a95` docs(08-06): AGENTS.md -- html format is always zero-JS, auto-fit is Flutter-only

## Self-Check: PASSED

- Files verified on disk: `cmd/eden-press/flags.go`, `cmd/eden-press/htmldoc.go`, `cmd/eden-press/htmldoc_test.go`, `cmd/eden-press/convert_test.go`, `cmd/eden-press/preview.go`, `cmd/eden-press/serve.go`, `cmd/eden-press/watch.go`, `cmd/eden-press/format.go`, `press/themecss_test.go`, `press/themes/themes.go`, `press/themes/themes_test.go`, `press/autofit.go`, `cmd/eden-press-export/main.go`, `themes/embed.go`, `NOTICE`, `AGENTS.md` — all FOUND; `press/browserjs.go` and `themes/browser-fit.js` confirmed ABSENT.
- Commits verified in `git log` (4/4): `391bfe6`, `66477ec`, `53d8238`, `6e65a95` — all FOUND.
- All 3 TRD `<verify>` gates PASS; whole-repo `go build`/`go vet`/`go test`/`go test ./conformance/...`/`gofmt -l .` clean; repo-wide grep gate (BrowserFitJS/browser-fit.js/auto-fit-script) zero hits across all `*.go`; `check-no-chromedp`/`check-cli-imports` PASS; Obj-2 grep-gate PASS; addlicense clean; zero-`<script>` acceptance gate PASS; CORE-09 marker tests green (press/autofit.go confirmed functionally unchanged).
- No BLOCKER: all gates green, no auth gates encountered.
