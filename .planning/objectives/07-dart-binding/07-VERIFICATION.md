---
status: passed
objective: 7
verified: 2026-07-21
score: 5/5 requirements, 4/4 success criteria
---

# Objective 7 Verification — Dart/Flutter Binding (bind/capi + bind/dart)

**Verdict: PASSED.** Verified against the actual merged codebase on `main` (all 5 TRD merge commits present: `1d807b6`/`e665eab`/`5923046`/`fda3b65`/`8362afc`). Every gate was re-run live in this verification pass, not just read from SUMMARYs — including actually building the wasm module, the host c-shared library, and (since Xcode is present on this host) the real iOS device+simulator c-archive → xcframework pipeline, then running the conformance corpus subset through both compiled artifacts and the Flutter widget tests through the actual Dart/Flutter SDK.

## Success criteria / requirements

| REQ | Criterion | Evidence | Status |
|-----|-----------|----------|--------|
| DART-01 | Single Go C-ABI core, JSON-in/JSON-out (`PressRender`/`PressFree`), not mirrored structs | `bind/capi/core/render.go` (`RenderJSON([]byte) []byte`, pure Go, no cgo — `go list -f CgoFiles ./bind/capi/core` → `[]`); `bind/capi/capi.go` (`//export PressRender`/`//export PressFree`, cgo shim, `CgoFiles=[capi.go]`); envelope is JSON (`eden-press.capi/v1`), never mirrored Dart/Go structs. `go test ./bind/capi/core/...` → PASS | ✅ |
| DART-02 | Android `-buildmode=c-shared`, iOS `-buildmode=c-archive`; NO gomobile bind | `scripts/build-android.sh` (NDK per-ABI clang, c-shared, jniLibs layout) — ran live, correctly fails fast with remediation text since `ANDROID_NDK_HOME` is unset on this host (NDK absent by design; CI provisions it). `scripts/build-ios.sh` — **ran live end-to-end on this Xcode 26.6 host**: built device-arm64 + sim-arm64 + sim-x86_64 c-archives, `lipo`-merged the simulator slices (`lipo -info` confirms `x86_64 arm64`), packaged `EdenPress.xcframework` (verified `ios-arm64` + `ios-arm64_x86_64-simulator` slice dirs + `Info.plist` present). `.github/workflows/dart-native.yml` — two isolated jobs (`android-build` ubuntu-latest pinned NDK r27c; `ios-build` macos-15 pinned Xcode 16.4 with NDK co-present/unused, the SC#2 case), each runs its own script + `check-no-chromedp.sh`; YAML valid (`yaml.safe_load`) and `actionlint` clean. `grep -rn gomobile` across `scripts/`/`.github/` → only comment mentions of "NO gomobile", zero actual usage; `go.mod`/`go.sum` have no `golang.org/x/mobile`. | ✅ |
| DART-03 | Web `GOOS=js GOARCH=wasm` + `wasm_exec.js` loader (standard Go, not TinyGo) | `bind/wasm/main.go` (`//go:build js && wasm`, registers `globalThis.pressRender` over `core.RenderJSON`, `select{}`-blocked). Built live: `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./bind/wasm` → 0; `scripts/build-wasm.sh` run live → PASS, produced `press.wasm` + copied toolchain-matched `wasm_exec.js`; `scripts/check-wasm-exec-version.sh` → PASS (matched). `node bind/wasm/smoke/smoke.mjs` run live → PASS (`# Hi`→`<h1`, `~~struck~~`→`<s>` not `<del>`). No TinyGo anywhere (`grep -i tinygo` across `.go`/`.sh`/`.yml`/`go.mod`/`go.sum` → zero actual usage, only comments documenting the standard-Go decision). | ✅ |
| DART-04 | JS-free Dart rendering surface — math via `flutter_math_fork`, code via `highlighting`/`flutter_highlighting`, from the JSON Model | `bind/dart/lib/src/render_surface.dart` walks `Output.Model.sections[*].blocks` (schema-v2), dispatching `math`→`Math.tex(...)` (flutter_math_fork), `code`→`HighlightView(...)` (flutter_highlighting); `web_loader.dart` uses `dart:js_interop` only (not `dart:html`/`package:js`). Ran live on this host's Flutter 3.41.4/Dart 3.11.1 SDK: `flutter pub get` → 46 deps resolved clean; `dart analyze` → "No issues found!"; `flutter test` → **4/4 widget-test cases PASS** (Math+HighlightView render from a hand-built JSON model fixture; display-mode mapping; language/source passthrough; pubspec dependency-set structurally asserts no html/js/webview package). `pubspec.yaml` deps confirmed: `ffi`, `web` (js_interop only), `flutter_math_fork`, `highlighting`/`flutter_highlighting` — no `webview`/`js`/`html` package. | ✅ |
| DART-05 | Shared conformance-corpus subset runs THROUGH the compiled capi/wasm artifact via the JSON entrypoint | `bind/conformance/subset.go` (`Subset()`, 7 cases spanning commonmark/strikethrough/emoji/highlight/math/autofit/sanitize, `TestSubsetCoverage` PASS). Both boundary lanes are SKIP-guarded when artifacts are absent (confirmed: bare `go test ./...` SKIPs both); **re-ran live with real artifacts present** (built `press.wasm` via `build-wasm.sh` and `libpress.so` via `build-capi-host.sh` in this session): `CGO_ENABLED=1 go test ./bind/conformance/... -v` → **all 8 tests PASS, 7/7 cases each**: `TestWASMBoundaryParity`, `TestWASMBoundaryWholeShape`, `TestCapiBoundaryParity`, `TestCapiBoundaryWholeShape`, `TestCapiBoundaryMemoryPlumbing` (PressRender==PressFree call counts, no leak). This proves the boundary through the actual compiled artifacts, not just the Go-native in-process test. | ✅ |

**Success Criteria (ROADMAP.md, 4 stated):** all four map 1:1 onto DART-01/02 (SC#1+SC#2), DART-04 (SC#3), DART-05 (SC#4) above — all ✅.

## Gates re-run live on `main` (this verification pass)

| Gate | Command | Result |
|---|---|---|
| gofmt | `gofmt -l bind/` | empty (clean) |
| build | `go build ./...` | 0 |
| vet | `go vet ./...` | 0 |
| test (whole repo) | `go test ./...` | all `ok` (bind/capi/core, bind/conformance included) |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0, scans `./bind/...` (confirmed in `TREES` array) |
| pure-Go core | `go list -f '{{.CgoFiles}}' ./bind/capi/core` | `[]` (no cgo — compiles under wasm) |
| wasm build | `GOOS=js GOARCH=wasm go build ./bind/wasm/... ./bind/capi/core/...` | 0 both |
| wasm build script | `bash scripts/build-wasm.sh` | PASS, `press.wasm` + version-matched `wasm_exec.js` |
| wasm smoke | `node bind/wasm/smoke/smoke.mjs` | PASS (`<h1`, `<s>` battery) |
| capi host build | `bash scripts/build-capi-host.sh` | PASS, `libpress.so`+`libpress.h` |
| conformance (both boundaries, artifacts built) | `CGO_ENABLED=1 go test ./bind/conformance/... -v` | 8/8 tests PASS, 7/7 cases each lane |
| Android build script | `bash scripts/build-android.sh` (NDK absent, by design) | correct fail-fast with remediation text (exit 1) |
| iOS build script | `bash scripts/build-ios.sh` (real Xcode host) | **PASS live** — 3 slices + lipo + xcframework |
| CI workflow YAML | `python3 -c 'yaml.safe_load(...)'` + `actionlint .github/workflows/dart-native.yml` | 0 both |
| shellcheck | `shellcheck scripts/build-android.sh scripts/build-ios.sh` | 0 (no findings) |
| Dart analyze | `cd bind/dart && dart analyze` | "No issues found!" |
| Dart tests | `cd bind/dart && flutter test` | 4/4 PASS |
| license headers | `addlicense ... -check .` | clean (only finding: a generated, gitignored `.h` build artifact — expected, not committed) |

All build artifacts produced during this verification pass (`bind/wasm/press.wasm`, `bind/capi/build/`, `bind/dart/.dart_tool/`, `bind/dart/pubspec.lock`) were removed afterward; `git status` confirms a clean tree (only pre-existing unrelated untracked scratch files remain).

## Dependency check (06-01 schema-v2 gate)

07-05 (DART-04) was declared blocked on Objective 6's 06-01 landing `chase/model` schema-v2. Confirmed on `main`: `chase/model.SchemaVersion == "eden-press.model/v2"` (`chase/model/document.go:56`); `Block{Kind,Text,Level,Language,Display,Ordered,Items}` carries raw math TeX via `Text`+`Display` and raw code source+language via `Text`+`Language` (`chase/model/document.go:87-102`). `bind/dart/lib/src/model.dart` and `render_surface.dart` bind to exactly this delivered shape (confirmed by direct read, not guessed, per 07-05-SUMMARY's own key-decision). Dependency satisfied.

## Requirements traceability note (non-blocking documentation drift)

`.planning/REQUIREMENTS.md`'s per-requirement checklist (lines 77-81) shows `DART-01`, `DART-02`, `DART-04` as unchecked `[ ]` while its own traceability table (lines 177-181) and `.planning/ROADMAP.md`'s top-level Objective-7 bullet (line 32, `- [ ]`) also lag — even though the section-level TRD list (ROADMAP lines 182-186) correctly shows all 5 TRDs `[x]` complete, and `.planning/STATE.md`'s "Blockers/Concerns" still lists the standard-Go-vs-TinyGo decision gate as "open" (STATE.md:35) despite it being resolved in 07-02 (`bind/wasm/main.go` doc-comment + `07-02-SUMMARY.md` key-decisions: standard Go chosen, no TinyGo anywhere in the codebase, confirmed by this verification's own grep). This is the same reconciliation-commit-lag pattern observed in prior objectives in this repo (checkbox/tracking-doc updates are best-effort per-job and occasionally skipped even when the underlying work is fully merged and tested) — **not a gap in the objective**, since the code, tests, and CI gates are all green and verified live above. Recommend a follow-up `docs` commit to flip `REQUIREMENTS.md`/`ROADMAP.md`/`STATE.md` to match reality.

## Notes

- No `gomobile bind` anywhere in the repository — confirmed by grep across all shell scripts, Go files, and CI YAML; every mention of "gomobile" is a comment documenting its deliberate absence.
- Three-front-doors-one-core architecture is real and verified structurally: `bind/capi/core.RenderJSON` is the single pure-Go marshalling seam that both the cgo shim (`bind/capi/capi.go`) and the wasm shim (`bind/wasm/main.go`) call — confirmed by reading both call sites, not inferred.
- DART-05's boundary tests are legitimately SKIP-guarded in a bare `go test ./...` (artifacts must be built first) — this verification pass built both artifacts live and re-ran the tests to prove the SKIP-guard is not masking a broken boundary.
- NDK-dependent Android build is correctly unavailable on this sandbox (no Android NDK installed) and fails fast with a precise message rather than a cryptic clang error; the authoritative Android proof is the CI job (`dart-native.yml`), which is present, valid, and pinned (NDK r27c). The iOS leg, unusually, *was* fully exercisable on this host (Xcode present) and was run for real, not just YAML-validated.

**5/5 TRDs complete with SUMMARYs + self-checks, all merged to `main`. Objective 7 achieves its goal: `press.Render()` is exposed to Flutter via one Go C-ABI JSON core, built three real ways, with zero `gomobile bind` and zero JavaScript in the Dart render path.**
