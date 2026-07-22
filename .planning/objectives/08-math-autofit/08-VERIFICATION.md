---
status: passed
objective: 8
verified: 2026-07-22
score: 4/4 success criteria
---

# Objective 8 Verification — Math-Fidelity Hardening + Auto-Fit Resolution

**Verdict: PASSED.** Verified against the actual merged codebase on `main` (not SUMMARY claims). All whole-repo CI gates green; all 4 success criteria independently confirmed with structural/executable evidence — MathML-DOM assertions for criterion 1, a corpus-tested trigger regex for criterion 2, a verified font provenance + MATH-table smoke for criterion 3, and a repo-wide grep proving zero viewer-side JS plus a genuine native Flutter `TextPainter` fit for criterion 4. This is the FINAL roadmap objective; no new v1 requirement IDs are owned here (hardens CORE-08/CORE-09).

## Whole-repo CI gates

| Gate | Command | Result |
|------|---------|--------|
| gofmt | `gofmt -l .` | 0 files (clean) |
| build | `go build ./...` | PASS |
| vet | `go vet ./...` | PASS |
| unit tests | `go test ./...` | PASS (all 20 packages, 3 no-test-file) |
| no-chromedp boundary | `bash scripts/check-no-chromedp.sh` | PASS — 0 chromedp in press/chase/profiles/bind/cmd/eden-press; eden-press-export is sole chromedp cmd |
| CLI import boundary | `bash scripts/check-cli-imports.sh` | PASS — cmd/eden-press imports only press/ |
| conformance | `go test ./conformance/...` | PASS (TestMarpCorpus, TestSpecSweep 1258/1352) |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -ignore 'convert/chrome/fonts/**' -ignore 'internal/latex2mathml/**' -check .` | PASS (exit 0), matches `.github/workflows/ci.yml` exactly |

## Success criteria — per-criterion evidence

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | All 8 math-spike cases (limit stacking, binom/pmatrix fence, sqrt[n], aligned→mtable, mathvariant→Unicode) render at KaTeX-parity AND are a permanent regression set | ✅ PASS | `go test ./press/math/ -run TestSpikeCorpus -v` → 8/8 subtests PASS. Each assertion is genuine structural MathML-DOM inspection (parsed XML tree, not string match): case 1/2/3 assert `<munderover>`/`<munder>` presence and the absence of the wrong side-by-side `<msubsup>`/`<msub>`; case 4 asserts `<mroot>` has exactly 2 children (radicand="x", index="3"); cases 5/6 assert a matched, content-sized fence pair + (for pmatrix) an `<mtable>` body; case 7 asserts `<mtable>` with 2 `<mtr>` rows, right/left `columnalign` split, and no literal `<mi>&</mi>` survivor; case 8 asserts real Unicode codepoints (ℝ U+211D, 𝐯 U+1D42F, ℒ U+2112) with **zero** `mathvariant` attribute emitted (Chromium MathML-Core ignores it). Fork vendored at `internal/latex2mathml/` via `go.mod` `replace git.sr.ht/~mekyt/latex2mathml => ./internal/latex2mathml`; upstream module path/MIT license (mekyt, 2023) preserved verbatim in the vendored files and documented in `NOTICE`; `addlicense -ignore 'internal/latex2mathml/**'` correctly excludes it from the AO-Cyber header stamp. |
| 2 | Concrete, testable fallback-trigger detector auto-routes the permanent structural-ceiling constructs to the PNG path, corpus-tested | ✅ PASS | `press/math/detect.go`'s `fallbackRE` = `` \tag\b|\label\b|\begin{(?:align|alignat\*?)} `` — exactly `\tag`/`\label`/numbered-`align`-family, NOT the old over-broad set (confirmed `aligned`, `align*`, `cases`, `array` are absent from the regex and asserted as negatives). `go test ./press/math/ -run 'TestNeedsFallback|TestFallbackRouting' -v` → both PASS. `TestNeedsFallback`: hand-built positive/negative table incl. word-boundary guards (`\tagged`, `\labelled`, `\begingroup` must NOT trip it). `TestFallbackRouting`: 18-case corpus table asserting `\tag`/`\label`/`align`/`alignat`/`alignat*` → fallback=true, and `cases`/`aligned`/`align*`/`array` + all 8 spike-corpus raw strings → fallback=false — a genuine corpus test, not manual inspection. |
| 3 | STIX Two Math bundled from STIX-project's own OTF/WOFF2, CI smoke pixel-checks MATH-table presence | ✅ PASS | `convert/chrome/fonts/STIXTwoMath-Regular.otf` (838,652 bytes) + `STIXTwoMath-Regular.woff2` (552,168 bytes) present. `NOTICE` documents both sourced verbatim from `stipub/stixfonts` tag v2.13 (OTF from `fonts/static_otf/`, WOFF2 from `fonts/static_otf_woff2/` — the official sibling build, not a re-conversion), never a Google Fonts CDN mirror. `go test ./convert/chrome/ -run TestWoff2MathTableSurvivesConversion -v` → PASS (Go-stdlib-only WOFF2 table-directory parse confirms a MATH entry is present — no Chrome needed for this half). `go test ./convert/chrome/ -run TestStixMathTableSmoke -v` → SKIP ("no Chrome discovered") — this is the Chrome-driven pixel-render smoke; it SKIPs cleanly in this headless sandbox per the verification instructions' explicit "treat as expected, not a gap." Test source confirms the skip path is intentional (`t.Skipf("no Chrome discovered...")`), matching the pattern used elsewhere in the same file/package. |
| 4 | Auto-fit resolved with no silent viewer-side JS: Flutter-only native fit, JS removed from HTML path | ✅ PASS | Repo-wide grep across all `*.go`: `--auto-fit-script`, `BrowserFitJS`, `browser-fit.js` → **zero hits** in production code. `themes/browser-fit.js` and `press/browserjs.go` confirmed absent from disk. `go test ./cmd/eden-press/ -run 'TestAssembleHTMLZeroJSGolden|TestAssembleHTMLNoViewerSideAutoFitJS|TestAssembleHTMLInjectScriptsSeam' -v` → all 3 PASS: default `assembleHTML` output carries **zero** `<script>` tags even for a deck with a `# <!--fit-->` heading + fenced code block, while the CORE-09 markers (`data-auto-scaling="fit"`, `marp-fit-shrink`) still flow into `out.HTML` (inert on the web path, confirming `press/autofit.go` markers are unaffected). On the Flutter side, `bind/dart/lib/src/fit_text.dart` implements `computeFitFontSize` — a real `TextPainter`-measure + binary-search (12 iterations) shrink-only fit — and a `FitText` widget wired via `LayoutBuilder`; imports nothing beyond `package:flutter/material.dart` (no JS interop). `dart analyze` (bind/dart) → "No issues found!". `flutter test` (bind/dart) → 9/9 PASS, including `fit_text_test.dart`'s explicit "JS-free assertion" case and `render_surface_test.dart`'s "declares no html/dom-parsing/js/webview dependency" case. |

**Score: 4/4 success criteria verified.**

## Notes (non-blocking)

- **Chrome-gated / Flutter-SDK-gated tests skipping cleanly is expected, not a gap** (per verification instructions): `TestStixMathTableSmoke` SKIPs with no Chrome in this sandbox; the Flutter SDK WAS available here, so `dart analyze` + `flutter test` for `bind/dart` were run for real (not skipped) and both are clean.
- **Minor doc staleness (non-functional, does not affect code/tests):**
  - `.planning/ROADMAP.md`'s Objective 8 TRD checklist shows `08-04-TRD.md` as `[ ]` unchecked, even though the objective-level summary line on the same file states "7/7 TRDs complete" and git history shows it merged (`1d2dbd7 merge(08-04): finalize fallback-trigger rule to the permanent structural ceiling — closes Objective 8`), and `TestNeedsFallback`/`TestFallbackRouting` both pass against the finalized `fallbackRE`. Recommend reconciling the single checkbox.
  - `.planning/STATE.md`'s narrative body (Current Position / Progress bar) is stale — reads as if Objective 8 were still in progress after 08-03, even though its own title line and `ROADMAP.md`'s objective entry both mark Objective 8 complete (2026-07-22). Tracking-doc drift only; the merged code and tests are the source of truth and were verified directly.
  - `CONTRIBUTING.md:59` still cites `themes/browser-fit.js` in an example `addlicense` invocation from the Objective-3 era; the file no longer exists (confirmed absent from disk) and no production code references it — a harmless doc example, not a functional gap.

## Gaps

None.

---

*Verified: 2026-07-22*
*Verifier: Claude (verifier)*
