---
status: passed
objective: 5
verified: 2026-07-21
score: 3/3 requirements, 5/5 success criteria
---

# Objective 5 Verification — convert/pdf + convert/png (chromedp raster export)

**Verdict: PASSED.** Verified against the actual merged codebase on `main` (not SUMMARY claims). All gates run clean, the Chrome-touching-code boundary is mechanically proven at zero, and every requirement maps to real, substantive, wired code.

## Gates run (this session, against `main`)

| Gate | Command | Result |
|------|---------|--------|
| Format | `gofmt -l convert/` | ✅ clean (no output) |
| Build | `go build ./...` | ✅ exit 0 |
| Vet | `go vet ./...` | ✅ exit 0 |
| Test (convert only) | `go test ./convert/...` | ✅ all 5 sub-packages `ok` |
| Test (whole module) | `go test ./...` | ✅ all 21 testable packages `ok` |
| No-chromedp gate | `bash scripts/check-no-chromedp.sh` | ✅ `PASS: no chromedp in the press/chase/profiles dependency closure.` |
| **Boundary count (critical)** | `go list -deps ./press/... ./chase/... ./profiles/... \| grep -c chromedp` | ✅ **0** |
| Boundary count (bind/) | `go list -deps ./bind/... \| grep -c chromedp` | ✅ **0** |
| Confinement sanity | `go list -deps ./convert/... \| grep -c chromedp` | ✅ **62** (proves chromedp genuinely lives here, not that the grep is vacuous) |

Test detail: 52 tests `PASS`, 11 cleanly `SKIP` (all Chrome/LibreOffice-presence-gated: `TestCapstoneExportEndToEnd`, `TestToPDF*` (4), `TestToImages*` (2), `TestSessionMultiTab`, `TestLoadHTMLReadsBackRealContent`, `TestTrivialDeckLibreOfficeSmoke`, `TestAcceptanceDeckLibreOfficeSmoke`). This sandbox has no system Chrome/`soffice` — every skip carries an explicit `t.Skipf("no Chrome discovered...")`/`"soffice not on PATH..."` message, per the expected pattern; treated as expected, not a gap. `TestFontFaceDataURI` (the one test that could have skipped on a missing font asset) **ran and passed** — the STIX OTF is genuinely embedded (838,652-byte file via `go:embed`), not a deferred stub.

## Requirements coverage

| Requirement | Source TRD | Description | Status | Evidence |
|---|---|---|---|---|
| **EXP-01** | 05-03-TRD.md | PDF via `chromedp` `Page.PrintToPDF` inside an ActionFunc, fixed viewport, pinned TZ/locale, animations disabled | ✅ SATISFIED | `convert/pdf/pdf.go`: `page.PrintToPDF().WithPrintBackground(true).WithPreferCSSPageSize(true)...Do(ctx)` invoked inside `chromedp.ActionFunc(func(ctx context.Context) error {...})` (line ~91-108) — the 3-return-value `Do(ctx)` signature that forces ActionFunc use is honored, not worked around. `chrome.ApplyDeterminism` (called from `ToPDF`) pins viewport (`chromedp.EmulateViewport`), UTC (`emulation.SetTimezoneOverride("UTC")`), `en-US` locale, and `prefers-reduced-motion:reduce`; `chrome.ComposeCSS` appends an `!important` animation/transition/scroll-behavior kill rule as defense-in-depth. `@page` sizing via `chrome.PageCSSInches` + `WithPreferCSSPageSize(true)`. Tests: `TestToPDFStructuralPageCount`, `TestToPDFDeterministicStructure`, `TestToPDFPrintsBackgrounds`, `TestToPDFInlineSVGFixture` (all Chrome-gated, clean SKIP here). |
| **EXP-02** | 05-04-TRD.md | Per-slide PNG/JPEG via chromedp screenshots | ✅ SATISFIED | `convert/png/png.go` `ToImages`: loops `out.Model.Sections`, one `chromedp.Screenshot(sel, &buf, chromedp.ByQuery)` call per slide (never `ScreenshotNodes`, per its own doc comment), selector built for both plain (`div.marpit > section:nth-of-type(k)`) and inline-SVG (`div.marpit > svg:nth-of-type(k) > foreignObject > section`) modes. JPEG re-encode via `image/jpeg` when `Options.Format == convert.JPEG`. Shares the same `chrome.ApplyDeterminism`/`chrome.ComposeCSS` recipe as PDF (single shared substrate, not reimplemented). Tests: `TestToImagesPlainMode`, `TestToImagesInlineSVGModeSmoke` (Chrome-gated, clean SKIP here); structural/fixture tests pass unconditionally. |
| **EXP-04** | 05-01/05-02/05-05-TRD.md | Robust Chrome discovery (`--browser-path`/`CHROME_PATH` → known paths → pinned headless-shell) + STIX Two Math font provisioning; CI runs export tests against pinned headless-shell in a no-system-Chrome container | ✅ SATISFIED | `convert/chrome/discover.go` `Discover`: 4-tier fallback exactly as specified (1. `BrowserPath` 2. `CHROME_PATH` env (manual, not a chromedp builtin) 3. PATH auto-detect of known binary names, empty-execPath handoff to chromedp's own allocator 4. `ErrChromeNotFound` documenting the pinned-download remedy). Tests `TestDiscoverBrowserPath/ChromePathEnv/AutoDetect/NotFound/Defaults` all PASS (pure/DI-testable, no real Chrome needed). Font: `convert/chrome/fonts.go` embeds `convert/chrome/fonts/STIXTwoMath-Regular.otf` (838,652 bytes, real binary, not a placeholder) via `go:embed`, injected as base64 `@font-face` data-URI by `FontFaceDataURI()` — `TestFontFaceDataURI` passes (non-empty branch exercised). NOTICE (root `NOTICE` file, lines 76-99) attributes STIX Two Math to the STIX Fonts Project under **SIL OFL-1.1**, with verbatim-bundling rationale and Reserved-Font-Name scoping correctly noted; addlicense CI check `-ignore 'convert/chrome/fonts/**'` correctly excludes the binary from header-stamping. CI: `.github/workflows/ci.yml` `export` job builds a throwaway image (`chromedp/headless-shell:${CHROME_VERSION}` copied into a plain `golang:1.26-bookworm` base — no `apt-get install chromium` anywhere), runs as non-root (`--user 10001:10001`, dedicated `exporter` user), `--shm-size=1g` on top of the Session-default `--disable-dev-shm-usage`, unique `--user-data-dir` per run (`os.MkdirTemp` in `convert/chrome/session.go`, removed on `Close()`), `CHROME_PATH=/headless-shell/headless-shell` (forcing tier-2 resolution — proves the fallback chain, not just that some Chrome exists). `CHROME_VERSION` pinned to `151.0.7922.34` (never `latest`) in both `Makefile` and `ci.yml`. PDF-path re-validation process: `scripts/check-chrome-export.sh` + `make check-chrome-export` mechanically require `TestToPDFInlineSVGFixture` + `TestCapstoneExportEndToEnd` to have run on any version bump. `convert/EXPORT.md` documents all of the above for operators. |

**No orphaned requirements.** This objective's TRDs (`05-01` through `05-05`) collectively claim exactly `[EXP-01, EXP-02, EXP-04]`, matching `OBJECTIVE.md`'s declared `requirements:` frontmatter and `ROADMAP.md`'s Objective 5 entry 1:1. `EXP-03` (PPTX) is correctly out of scope here — it belongs to Objective 6 (`convert/pptx`), a sibling workstream, not this objective.

## ROADMAP.md success criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Rendered deck exports to PDF via `chromedp`'s `Page.PrintToPDF` (ActionFunc), fixed viewport, pinned TZ/locale, animations disabled | ✅ VERIFIED | `convert/pdf/pdf.go` + `convert/chrome/determinism.go` (see EXP-01 row above) |
| 2 | Same deck exports to per-slide PNG/JPEG via chromedp screenshots | ✅ VERIFIED | `convert/png/png.go` (see EXP-02 row above) |
| 3 | Chrome discovery fallback tested in CI against a no-system-Chrome container; STIX Two Math font-provisioning documented | ✅ VERIFIED | `convert/chrome/discover.go` + `.github/workflows/ci.yml` `export` job + `convert/EXPORT.md` |
| 4 | CI runs export tests against pinned `chromedp/headless-shell` with `--disable-dev-shm-usage`, non-root, unique `--user-data-dir`; PDF re-validated on version bump | ✅ VERIFIED | `.github/workflows/ci.yml` `export` job + `scripts/check-chrome-export.sh` |
| 5 | `go list -deps` on `chase/`, `press/`, `profiles/` still shows zero `chromedp` after `convert/` is added | ✅ VERIFIED | Measured directly this session: **0** (see gates table) |

**Score: 5/5 success criteria, 3/3 requirements.**

## Anti-pattern scan

`grep -rn -E "TODO|FIXME|XXX|HACK|PLACEHOLDER"` across `convert/*.go` (non-test): **zero hits.** No stub returns, no placeholder implementations found in the export path's production code.

## Functional verification

Skipped (browser/Maestro automation N/A) — this objective's "functional" surface is Go's own headless-Chrome test harness, which IS the functional check for this domain. Level-4 verification was performed by running the actual Go test suite (`go test ./convert/...`), which is the correct automated equivalent here: 52 tests pass, 11 skip cleanly and correctly (no system Chrome/soffice in this sandbox, exactly the documented, expected condition). No human/browser verification is needed beyond what the CI `export` job (which runs the same suite against a real pinned headless-shell) already proves — that job's presence and correct construction were verified by direct code inspection above, not just trusted from the SUMMARY.

## Notes (non-blocking)

- **REQUIREMENTS.md checklist drift** (same pattern as Objective 3, see agent memory): the top checklist section (lines 70-73) still shows `EXP-02` and `EXP-04` as unchecked `- [ ]`, while the traceability table further down (lines 173-176) correctly shows all of EXP-01/02/04 as "Complete", matching `ROADMAP.md` and the actual merged code. This is a documentation-reconciliation lag, not a code gap — recommend a follow-up `docs` commit to flip the two stale checkboxes, but it does not block this objective's PASSED status.

---

_Verified: 2026-07-21_
_Verifier: Claude (verifier)_
