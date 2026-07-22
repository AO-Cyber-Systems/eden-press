---
objective: 08-math-autofit
trd: "05"
subsystem: convert-chrome-fonts
tags: [go, fonts, woff2, opentype-math, chromedp, embed, ci-smoke, additive-seam]

# Dependency graph
requires:
  - objective: 08-math-autofit
    provides: "NOTICE's shared structure (08-01 owns the latex2mathml section; this TRD only ever touches the STIX Two Math section, serialized by depends_on [08-01])"
provides:
  - "convert/chrome/fonts.go: FontFaceDataURIWoff2() — an ADDITIVE accessor emitting a format('woff2') @font-face rule from the newly go:embed'd STIX Two Math WOFF2 companion; FontFaceDataURI() (OTF) is byte-for-byte unchanged, so every existing caller/test keeps working"
  - "convert/chrome/fonts/STIXTwoMath-Regular.woff2 — the OFFICIAL stipub/stixfonts v2.13 WOFF2 build (not a local conversion), fetched verbatim, MATH-table survival independently verified two ways (fonttools decompression diff + a dependency-free Go WOFF2-table-directory decoder)"
  - "convert/chrome/fonts_test.go: TestWoff2MathTableSurvivesConversion (dependency-free WOFF2 table-directory MATH-tag proof) + TestFontFaceDataURIWoff2 (shape/regression proof) + TestStixMathTableSmoke (Chrome-gated CI render pixel-check that catches a stripped/missing MATH table via a stretchy-construct height threshold, empirically derived: ~152px MATH-intact vs ~85px MATH-stripped)"
  - "NOTICE's STIX Two Math entry extended with the WOFF2 companion's source, verbatim/non-subsetting claim, and MATH-survival verification, matching the existing OTF entry's attribution shape"
affects:
  - "Any future serve/preview HTTP path that wants the smaller WOFF2 payload instead of the 838KB base64-inflated OTF can call FontFaceDataURIWoff2() directly — no other convert/ file was touched (ComposeCSS/determinism.go untouched, out of this TRD's declared scope)"
  - "CI now has a mechanical guard (TestStixMathTableSmoke) against a future careless font swap (e.g. a CDN-subsetted copy) silently reintroducing tofu — the exact 05-RESEARCH Pitfall 6 regression the verbatim-font bundling decision exists to prevent"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive accessor over in-place modification: FontFaceDataURIWoff2() is a NEW function beside the unchanged FontFaceDataURI() (OTF) — determinism_test.go's exact-prefix TestFontFaceDataURI assertion never had to change"
    - "Dependency-free WOFF2 table-directory decoding (Go stdlib only): parses just the WOFF2 header + table directory per the W3C WOFF2 spec (UIntBase128 varints, 63-entry Known-Table-Tags array, glyf/loca/hmtx-only transform-length special case) to prove MATH-table presence without needing brotli decompression or a third-party WOFF2 library"
    - "Stretchy-construct pixel-check, not lone-glyph ink-check: a parenthesized tall fraction (not an isolated stretchy operator) is required to discriminate a MATH-intact render from a MATH-stripped one — Chrome's MathML Core fallback renders an isolated operator identically either way, so ink-presence alone on a solo glyph cannot detect a stripped MATH table (empirically proven against a real fonttools-built negative-control fixture, never shipped)"
    - "Chrome-gated smoke, cleanly skippable: mirrors the existing convert/chrome test posture (Discover(DiscoverOptions{}) -> t.Skip on ErrChromeNotFound), so `go test ./...` stays green in a browserless sandbox/CI leg while the CI job with the pinned headless-shell 151.0.7922.34 actually exercises the render"

key-files:
  created:
    - convert/chrome/fonts/STIXTwoMath-Regular.woff2
    - convert/chrome/fonts_test.go
  modified:
    - convert/chrome/fonts.go
    - NOTICE

key-decisions:
  - "Research Open Q4 resolved YES: an official stipub/stixfonts WOFF2 build exists at the exact v2.13 tag (fonts/static_otf_woff2/STIXTwoMath-Regular.woff2), fetched verbatim via curl from raw.githubusercontent.com — no local OTF->WOFF2 conversion tool was needed, eliminating any risk of an accidental subsetting conversion"
  - "MATH-table survival was independently verified TWO ways, not assumed: (1) fonttools ttLib.woff2 decompress on the fetched WOFF2 shows the identical 6760-glyph count and the same table set as the bundled OTF (minus DSIG, which stipub's own WOFF2 build conventionally drops as non-rendering-relevant); (2) a from-scratch, dependency-free Go WOFF2 table-directory decoder (TestWoff2MathTableSurvivesConversion) asserts a MATH tag entry directly against the shipped go:embed'd bytes, so both the on-disk asset and the embed wiring are proven, not just the upstream source file"
  - "The CI smoke's pixel-check construct is a parenthesized fraction, not an isolated operator: an early lone-operator (solo Sigma) design was empirically disproven during this TRD — Chrome's MathML Core silently falls back and renders it identically with or without a working MATH table. A construct that forces genuine cross-element stretch discriminates cleanly (~152px MATH-intact vs ~85px MATH-stripped, same markup/font-size), validated against a real fonttools-built negative-control fixture (MATH table surgically removed) that was never shipped"
  - "FontFaceDataURI (OTF) was left completely untouched; the WOFF2 capability is exposed only via the new FontFaceDataURIWoff2() accessor, per the TRD's error_recovery guidance to add rather than replace-in-place"

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 9
  gates_passed: 9
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: true
  blockers: none

# Metrics
duration: ~55min
completed: 2026-07-22
---

# Objective 08 TRD 05: STIX Two Math WOFF2 Companion + CI MATH-Table Smoke Summary

**The STIX Two Math OTF now has an official (not converted) WOFF2 sibling with independently verified MATH-table survival, exposed via an additive `FontFaceDataURIWoff2()` accessor that leaves the existing OTF path byte-for-byte unchanged; and CI now carries a Chrome-gated pixel-check smoke that renders a parenthesized fraction and would catch a future stripped/subsetted MATH table via a stretchy-height threshold empirically derived against a real negative-control fixture.**

## What was built

### Task 1 — STIX Two Math WOFF2 companion (lossless, MATH-table verified) + NOTICE (commit 41f224f)
- Research Open Q4 resolved empirically first: queried the stipub/stixfonts repo at its exact `v2.13` tag and confirmed `fonts/static_otf_woff2/STIXTwoMath-Regular.woff2` is an OFFICIAL build published by the project itself — fetched verbatim via `curl` (sha256 `e623fcdb808e9a8a27833781f77d5a5ed5c7d5f34830aaa8bcc9fcf8ec32b9c9`, 552168 bytes), avoiding any local conversion tooling and its subsetting risk entirely.
- `convert/chrome/fonts.go`: added `//go:embed fonts/STIXTwoMath-Regular.woff2` (`stixTwoMathWOFF2 []byte`), a sibling `fontFaceCSSWoff2` template (`format('woff2')`), and `FontFaceDataURIWoff2() string` — additive only; `FontFaceDataURI()` (OTF) untouched.
- `convert/chrome/fonts_test.go` (Task 1 portion): `woff2TableTags`/`parseUintBase128` — a dependency-free (Go stdlib only) WOFF2 header + table-directory decoder (W3C WOFF2 spec sections 4/5, 63-entry Known-Table-Tags array, MATH = index 31) — plus `TestWoff2MathTableSurvivesConversion` (asserts a MATH tag entry against the actual embedded bytes) and `TestFontFaceDataURIWoff2` (shape/regression proof, confirms OTF accessor unaffected).
- `NOTICE`: extended the existing STIX Two Math entry with the WOFF2 companion's exact source (official stipub v2.13 tag, not a CDN/re-conversion), the verbatim/non-subsetting claim, and the MATH-survival verification detail (identical 6760-glyph count and table set minus the non-rendering-relevant DSIG), referencing the standing test guards.

### Task 2 — Chrome-gated MATH-table pixel-check smoke (tofu/stripped-table detector) (commit 69e8941)
- `convert/chrome/fonts_test.go` (Task 2 portion): `TestStixMathTableSmoke` — Chrome-presence-gated (`Discover(DiscoverOptions{})` -> `t.Skip` on absence, mirroring every other convert/chrome test), reuses the SAME `ApplyDeterminism` + `LoadHTML` determinism substrate every convert/ exporter uses, renders `<math><mrow><mo>(</mo><mfrac><mn>1</mn><mn>2</mn></mfrac><mo>)</mo></mrow></math>` at 80px with STIX Two Math as the sole font, screenshots the element (`chromedp.Screenshot`), and pixel-checks two things via a Go-stdlib-only `image`/`image/png` helper (`mathGlyphInkBBox`): ink presence (>= 3%, catches outright font-load failure) and stretchy bounding-box height (>= 110px, catches a stripped/missing MATH table).
- The 110px threshold was **empirically derived during this TRD**, not guessed: the identical markup/font-size renders ~152px tall with STIX Two Math's MATH table intact, and collapses to ~85px against a byte-for-byte copy of the SAME font with only its MATH table surgically removed (via `fonttools`, as a throwaway negative-control fixture — built and used for validation only, never shipped in the repo). An earlier lone-operator design was tried first and empirically disproven: Chrome's MathML Core fallback renders an isolated stretchy operator identically with or without a working MATH table, so only a genuine cross-element stretchy construct discriminates.
- Verified the test actually exercises the real logic, not just its skip path, by pointing `CHROME_PATH` at a local Chrome install for one manual run: `TestStixMathTableSmoke` PASSED for real (0.70s) against the shipped WOFF2. The standing/CI gate state (no `CHROME_PATH`, matching the sandbox) SKIPs cleanly, exactly as the TRD requires.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: WOFF2 companion + NOTICE | `go build ./... && go vet ./... && go test ./convert/chrome/... -run 'TestWoff2MathTableSurvivesConversion\|TestFontFaceDataURIWoff2\|TestFontFaceDataURI' -v` | 0 | PASS (3/3) |
| 2: Chrome-gated MATH smoke | `go test ./convert/chrome/... -run TestStixMathTableSmoke -v` (no CHROME_PATH) | 0 | PASS (SKIP, clean) |
| 2 (manual, real-Chrome confirmation) | `CHROME_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" go test ./convert/chrome/... -run TestStixMathTableSmoke -v` | 0 | PASS (0.70s, real render) |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l convert/chrome/fonts.go convert/chrome/fonts_test.go` | 0 | PASS (empty output) |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo, no CHROME_PATH) | `go test ./...` | 0 | PASS (all packages ok, smoke SKIPs cleanly) |
| conformance | `go test ./conformance/...` | 0 | PASS |
| no-chromedp invariant | `bash scripts/check-no-chromedp.sh` | 0 | PASS (press/chase/profiles/bind/cmd closure: 0; convert/ remains the sole chromedp-permitting tree) |
| CLI-import invariant | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -ignore 'convert/chrome/fonts/**' -ignore 'internal/latex2mathml/**' -check .` | 0 | PASS (fonts/** ignore already covers the new .woff2; fonts_test.go carries the MIT header) |

## Deviations from Plan

None — the TRD executed exactly as written. Research Open Q4 resolved in the "official build exists" branch (no local conversion tooling needed), and the pixel-check smoke's exact construct/thresholds were empirically derived during Task 2 as the TRD's `<embedded_context>` anticipated ("if the pixel-check is flaky... narrow the compared region"); no fallback/recovery path was actually triggered.

One process note (not a plan deviation): an early `df-tools.cjs commit --help` invocation (checking wrapper usage) accidentally created a stray commit sweeping in two non-source files (`.planning/.awareness-cache.json`, `.planning/.skill-active`). Caught immediately and undone via `git reset --soft HEAD~1` + unstage before either task commit landed — neither file is present in any commit on this branch.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0
- Must-haves verified: 4/4 (WOFF2 companion sourced+MATH-verified; FontFaceDataURI WOFF2 variant wired additively; Chrome-gated MATH-table CI smoke lands and both SKIPs cleanly without Chrome and PASSES for real with Chrome; NOTICE extended matching the OTF entry's attribution shape)
- Gate failures: None on the standing (no-CHROME_PATH) state. Note: pointing an UNPINNED local Chrome (150.0.7871.181) at the suite via `CHROME_PATH` during manual verification surfaces pre-existing, already-documented PDF page-count version-sensitivity failures in `convert`/`convert/pdf` (unrelated to this TRD — CLAUDE.md's Hard Rule 5 explains why `CHROME_VERSION` stays pinned at `151.0.7922.34` in CI). These are NOT regressions from this TRD; the standing no-CHROME_PATH gate run (the correct sandbox baseline) is fully green.
- Blockers: None

## Commits

- `41f224f` feat(08-05): STIX Two Math WOFF2 companion (lossless, MATH-table verified) + NOTICE
- `69e8941` test(08-05): Chrome-gated MATH-table pixel-check smoke (tofu/stripped-table detector)

## STATE / ROADMAP

Objective 08-math-autofit TRD 05 (STIX Two Math WOFF2 companion + CI MATH-table smoke) complete — 2/2 tasks committed, all standing gates green, NOTICE extended, no blockers.

## Self-Check: PASSED

- Files verified on disk (5/5): `convert/chrome/fonts.go`, `convert/chrome/fonts_test.go`, `convert/chrome/fonts/STIXTwoMath-Regular.woff2`, `NOTICE`, `.planning/objectives/08-math-autofit/08-05-SUMMARY.md` — all FOUND.
- Commits verified in `git log` (2/2): `41f224f`, `69e8941` — both FOUND.
- All 9 standing gates PASS on the no-CHROME_PATH sandbox baseline; `TestStixMathTableSmoke` additionally confirmed to PASS for real against a local Chrome (manual, not part of the standing gate).
- No BLOCKER: official stipub WOFF2 build existed at the exact v2.13 tag, so no local conversion tooling dependency was ever needed.
