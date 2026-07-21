---
objective: 07-dart-binding
job: "05"
subsystem: dart-binding
tags: [dart-ffi, flutter, plugin_ffi, flutter_math_fork, flutter_highlighting, js_interop, wasm]

# Dependency graph
requires:
  - objective: 07-dart-binding
    provides: "07-01's C ABI (bind/capi PressRender/PressFree, cgo-exported, memory-ownership contract), 07-02's wasm build (bind/wasm main.go registering globalThis.pressRender), 07-03's native build scripts (build-android.sh .so-per-ABI / build-ios.sh EdenPress.xcframework)"
  - objective: 06-convert-pptx
    provides: "chase/model schema-v2 (SchemaVersion=eden-press.model/v2): Section.Blocks -- ordered Block{Kind,Text,Level,Language,Display,Ordered,Items} carrying raw math TeX (Text+Display) and raw code source+language (Text+Language) per block, serialized for free via press.Output.Model"
provides:
  - "bind/dart: a Flutter plugin_ffi package (name eden_press) -- dart:ffi bindings to PressRender/PressFree with a correct arena-scoped memory round-trip, Android/iOS/web platform loaders, Dart model classes mirroring the eden-press.capi/v1 + eden-press.model/v2 wire shapes, and EdenPressView -- a JS-free widget that renders math via flutter_math_fork and code via flutter_highlighting directly from Output.Model's blocks"
  - "Widget-tested proof (4/4 cases) that Math + HighlightView widgets render from a hand-built JSON model fixture, and that pubspec.yaml carries zero html/dom/js/webview dependencies"
affects: [08-math-fidelity, 09-mobile-app-shell]

# Tech tracking
tech-stack:
  added:
    - "ffi 2.1.4 (dart:ffi native bindings)"
    - "web 1.1.1 (declared for web platform interop; web_loader.dart itself uses dart:js_interop's @JS()/globalContext directly)"
    - "flutter_math_fork 0.7.4 (Math.tex TeX rendering)"
    - "highlighting + flutter_highlighting 0.9.0+11.8.0 (HighlightView code rendering, githubTheme default)"
    - "yaml 3.1.3 (dev-only: structural pubspec.yaml dependency-set assertion in the widget test, Test-list case 4)"
  patterns:
    - "Arena/`using`-scoped FFI round trip: toNativeUtf8(allocator: arena) for the Dart-owned input, PressRender, toDartString() to copy the Go/C-heap output out, then PressFree(responsePtr) -- never calloc.free on a Go-returned pointer"
    - "Conditional import (`native_loader.dart` if (dart.library.js_interop) `web_loader.dart`) to select the FFI vs js_interop backend at compile time behind one `String renderJson(String)` interface"
    - "Lazy top-level DynamicLibrary/bindings initialization (Dart top-level finals init on first access, not at import time) -- lets tests import eden_press.dart's barrel file without ever touching the native loader, since widget tests build Output fixtures directly and never call render()"
    - "JS-free render surface driven entirely by Output.Model.sections[*].blocks (schema-v2), never by DOM-parsing Output.html -- math via Math.tex(block.text, mathStyle: block.display ? .display : .text), code via HighlightView(block.text, languageId: block.language)"

key-files:
  created:
    - bind/dart/pubspec.yaml
    - bind/dart/lib/eden_press.dart
    - bind/dart/lib/src/ffi_bindings.dart
    - bind/dart/lib/src/native_loader.dart
    - bind/dart/lib/src/web_loader.dart
    - bind/dart/lib/src/model.dart
    - bind/dart/lib/src/render_surface.dart
    - bind/dart/test/render_surface_test.dart
  modified:
    - .gitignore

key-decisions:
  - "press.Output wire keys are Go-default CAPITALIZED (HTML/CSS/Model/Meta/Comments) because press.Output itself carries no json tags -- confirmed by reading press/options.go directly, matching 07-02-SUMMARY's own documented wire-key correction. Nested model.Document/Section/Block/ListItem DO have lowercase json tags (schemaVersion/sections/blocks/kind/text/level/language/display/ordered/items/sectionId) -- confirmed by reading chase/model/document.go directly. model.dart binds to both castings exactly as delivered; never guessed."
  - "06-01's delivered schema-v2 is a flat, UNIFIED Block{Kind,Text,Level,Language,Display,Ordered,Items} -- not the TRD's illustrative nested code{source,language}/math{tex,display} pseudo-shape. Block.Text serves double duty as raw TeX (math) or raw source (code) depending on Kind. render_surface.dart binds to the real, delivered field name (block.text) for both cases, per the TRD's own instruction to bind to the delivered shape rather than the illustrative example."
  - "bind/dart is declared a plugin_ffi package (pubspec.yaml flutter.plugin.platforms.android/ios: ffiPlugin:true) WITHOUT hand-built android/ios native directories (Gradle module, Podspec) -- empirically verified in a throwaway scratch project that flutter pub get / dart analyze pass cleanly with only the platform declaration present. The android/jniLibs vendoring of 07-03's .so and the iOS podspec vendored_libraries entry for 07-03's .xcframework are follow-up integration work, out of this TRD's files_modified scope (pubspec.yaml + lib/ + test/ only)."
  - "pubspec.lock is gitignored, not committed -- bind/dart is a leaf/library package (a Flutter plugin, not a standalone application), and Dart/pub convention is that packages omit the lockfile while applications commit it."

patterns-established:
  - "JS-free Flutter rendering surface for a JSON document model: one StatelessWidget walks ordered content blocks and dispatches per Kind to the fidelity-appropriate native widget (Math.tex / HighlightView / plain Text), never touching a pre-rendered HTML/DOM representation."

requirements-completed: [DART-04]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 21min
completed: 2026-07-21
---

# Objective 7 TRD 05: JS-Free Dart Rendering Surface Summary

**Flutter `plugin_ffi` package `bind/dart` (eden_press): dart:ffi bindings to PressRender/PressFree with an arena-scoped memory round-trip, Android/iOS/web platform loaders, and an `EdenPressView` widget rendering math via flutter_math_fork and code via flutter_highlighting directly from `Output.Model`'s schema-v2 blocks -- zero JavaScript, zero HTML/DOM parsing, in the Dart render path.**

## Performance

- **Duration:** 21 min (07-02 wasm merge / TRD start anchor 13:02:37 -> Task 2 commit 13:22:43, local time)
- **Started:** 2026-07-21T17:02:37Z
- **Completed:** 2026-07-21T17:22:43Z
- **Tasks:** 2/2 complete
- **Files modified:** 9 (8 created, 1 modified)

## Accomplishments
- `bind/dart` scaffolded as a genuine Flutter `plugin_ffi` package (`eden_press`) with pinned deps (`ffi 2.1.4`, `web 1.1.1`, `flutter_math_fork 0.7.4`, `highlighting`/`flutter_highlighting 0.9.0+11.8.0`) -- `flutter pub get` resolved cleanly against pub.dev (46 dependencies).
- `ffi_bindings.dart`: `NativeRenderBindings.renderJson` performs the full arena-scoped memory round trip -- `toNativeUtf8(allocator: arena)` for the Dart-owned input, `PressRender`, `toDartString()` to copy the Go/C-heap response out, then `PressFree(responsePtr)` inside a `finally` (never `calloc.free` on a Go-returned pointer).
- Platform loaders: `native_loader.dart` (`DynamicLibrary.open('libpress.so')` on Android, `DynamicLibrary.process()` on iOS/macOS for the podspec-static-linked archive) and `web_loader.dart` (`dart:js_interop`'s `@JS('pressRender')` static binding to the wasm-registered global -- no `dart:html`/`package:js`), selected via a compile-time conditional import behind one `renderJson(String)` interface.
- `model.dart`: Dart classes for the full `eden-press.capi/v1` + `eden-press.model/v2` wire shapes, with the capitalized-`Output`/lowercase-`Document` key casing verified directly against `press/options.go` and `chase/model/document.go` source (not guessed).
- `render_surface.dart`'s `EdenPressView` walks `output.model.sections[*].blocks` in order, dispatching `math` -> `Math.tex(block.text, mathStyle: ...)`, `code` -> `HighlightView(block.text, languageId: block.language, theme: githubTheme)`, other kinds -> plain `Text`.
- TDD widget test (4/4 cases): genuine RED (stubbed math/code cases to plain `Text`, confirmed 3 widget-assertion failures) -> genuine GREEN (restored implementation, all 4 pass) -- proves `Math` + `HighlightView` render from a hand-built JSON model fixture, the `display` flag maps to `MathStyle.display`/`.text`, `language`/raw `source` pass through unmodified, and `pubspec.yaml`'s dependency set (parsed via `package:yaml`, not naive whole-file grep) carries no html/dom/js/webview package.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: plugin_ffi scaffold + dart:ffi bindings + platform loaders + memory round-trip | `cd bind/dart && flutter pub get && dart analyze` | 0 | PASS |
| 2: JS-free rendering surface + widget test | `cd bind/dart && flutter test test/render_surface_test.dart && dart analyze` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: plugin_ffi scaffold + dart:ffi bindings + platform loaders** - `2f6e5ad` (feat)
2. **Task 2: JS-free rendering surface (flutter_math_fork + flutter_highlighting)** - `4a3aaa4` (feat)

_Note: Task 2 is `tdd="true"`; RED (3 widget-assertion failures against a stubbed implementation) confirmed before GREEN (restored implementation, all 4 pass) -- see TDD Evidence below. Both RED and GREEN were captured within Task 2's own cycle, before the single task commit (consistent with 02-01-SUMMARY's precedent of one commit per TDD task)._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint (dart) | `cd bind/dart && dart analyze` | 0 | PASS |
| format (dart) | `cd bind/dart && dart format --output=none --set-exit-if-changed .` | 0 (after one `dart format .` auto-fix pass) | PASS |
| test (dart) | `cd bind/dart && flutter test` | 0 (4/4) | PASS |
| build (dart) | `cd bind/dart && flutter pub get` | 0 | PASS |
| structural (no-scrape) | `grep -Eiv '^\s*#' pubspec.yaml \| grep -Ei 'html\|webview\|package:js\|dart:html'` (dependencies block only) | 1 (no match, i.e. clean) | PASS |
| gofmt (Go, unaffected) | `gofmt -l .` | 0 (no output) | PASS |
| go build (Go, unaffected) | `go build ./...` | 0 | PASS |
| go vet (Go, unaffected) | `go vet ./...` | 0 | PASS |
| go test (Go, unaffected) | `go test ./...` | 0 (all packages ok) | PASS |
| no-chromedp (Go, unaffected) | `bash scripts/check-no-chromedp.sh` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED | `flutter test test/render_surface_test.dart` (math/code block cases stubbed to plain `Text`) | 1 (Cases 1-3 failed: 0 `Math`/`HighlightView` widgets found; Case 4 passed independently) | FAIL (correct) |
| GREEN | `flutter test test/render_surface_test.dart` (real `Math.tex`/`HighlightView` implementation restored) | 0 (4/4 passed) | PASS (correct) |
| REFACTOR | `dart format .` (whitespace/line-wrap only) + `flutter test` re-run | 0 (4/4 passed) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (Rule 3 sequencing fix for the Task1/Task2 forward-reference, described below)
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 07-05-TRD.md frontmatter: plugin_ffi + memory round-trip; per-platform loaders; JS-free Model-driven rendering; widget-test proof)
- **Gate failures:** None remaining

## Files Created/Modified
- `bind/dart/pubspec.yaml` - plugin_ffi package declaration, pinned deps, android/ios `ffiPlugin: true` platform metadata
- `bind/dart/lib/eden_press.dart` - public API: `Future<Output> render(String md, {EdenPressOptions opts})` + re-exports of `model.dart`/`render_surface.dart`
- `bind/dart/lib/src/ffi_bindings.dart` - `PressRender`/`PressFree` typedefs, lookup, and the arena-scoped `renderJson` round trip
- `bind/dart/lib/src/native_loader.dart` - Android `DynamicLibrary.open('libpress.so')` / iOS+macOS `DynamicLibrary.process()` loader, lazily bound
- `bind/dart/lib/src/web_loader.dart` - `dart:js_interop` `@JS('pressRender')` binding to the 07-02 wasm global
- `bind/dart/lib/src/model.dart` - `Output`/`Meta`/`ModelDocument`/`Section`/`Block`/`BlockKind`/`ListItem`/`OutlineEntry`/`RenderRequest`/`RenderResponse`/`EdenPressOptions` JSON mirrors
- `bind/dart/lib/src/render_surface.dart` - `EdenPressView`: Model-block-driven, JS-free widget tree (`Math.tex` / `HighlightView` / plain `Text`)
- `bind/dart/test/render_surface_test.dart` - Test-list cases 1-4, hand-built JSON fixtures, structural pubspec dependency assertion via `package:yaml`
- `.gitignore` - added Dart/Flutter build-tooling ignores (`.dart_tool/`, `pubspec.lock`, `.flutter-plugins*`, `build/`)

## Decisions Made
- Wire-key casing resolved by direct source read (not guessed): `press.Output` fields are Go-default capitalized (`HTML`/`CSS`/`Model`/`Meta`/`Comments`, no json tags on that struct); nested `model.Document`/`Section`/`Block`/`ListItem` use their own lowercase json tags. See key-decisions above for the full rationale.
- `render()`'s public signature matches the TRD's literal Task 1 text exactly: `Future<Output> render(String md, {EdenPressOptions opts})` -- implemented as `async` wrapping a synchronous FFI call, so the API is stable if isolate-offloaded execution is added later without a breaking signature change.
- `githubTheme` (from `flutter_highlighting/themes/github.dart`) chosen as `EdenPressView`'s default `highlightTheme` for neutral, high-contrast readability; overridable via the constructor parameter. Theme choice is a presentation decision, not a DART-04 correctness concern (per the TRD's own `<recovery>` note for Task 2).
- Added `yaml: 3.1.3` as a **dev-only** dependency (never imported by the render path) so Test-list case 4 parses only the `dependencies:`/`dev_dependencies:` block structurally, rather than grep-scanning the whole file -- a naive whole-file substring scan would have false-positived on this very pubspec.yaml's own descriptive comment text ("...HTML/DOM parsing, anywhere in the render path.").

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Task 1's `eden_press.dart` forward-referenced Task 2's not-yet-created `render_surface.dart`**
- **Found during:** Task 1, pre-verify review
- **Issue:** The TRD's Task 1 action text says `lib/eden_press.dart` should "export the EdenPressView widget (Task 2)" -- but Task 1's own `<files>` list excludes `render_surface.dart`, and Task 2 creates it. Literally including `import`/`export 'src/render_surface.dart'` in Task 1's `eden_press.dart` would make Task 1's own verify command (`flutter pub get && dart analyze`) fail on a missing file, since Task 2 hadn't run yet.
- **Fix:** Task 1's `eden_press.dart` omitted the render_surface import/export (kept only `model.dart` wiring + the `render()` function) so Task 1 verified and committed standalone; Task 2 added the one-line `export 'src/render_surface.dart';` once the file existed, completing the public API surface the TRD's Task 1 text anticipated.
- **Files modified:** bind/dart/lib/eden_press.dart (touched in both Task 1 and Task 2 commits)
- **Verification:** `dart analyze` clean after both Task 1 (without the export) and Task 2 (with it) commits.
- **Committed in:** 2f6e5ad (Task 1, without export), 4a3aaa4 (Task 2, adds export)

**2. [Rule 2 - Missing critical functionality] `.gitignore` did not exclude Dart/Flutter build tooling**
- **Found during:** Task 2, pre-commit `git status` review
- **Issue:** `flutter pub get` generates `bind/dart/.dart_tool/` and `bind/dart/pubspec.lock`, both of which appeared as untracked and would otherwise get swept into a commit.
- **Fix:** Added `**/.dart_tool/`, `**/pubspec.lock`, `**/.flutter-plugins`, `**/.flutter-plugins-dependencies`, `**/build/` to the root `.gitignore` (pubspec.lock is intentionally excluded per Dart's package/library convention -- see Decisions above).
- **Files modified:** .gitignore
- **Verification:** `git status --short` shows a clean tree after both task commits (no untracked build artifacts).
- **Committed in:** 4a3aaa4 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 3, 1 Rule 2)
**Impact on plan:** Both are sequencing/hygiene fixes required to make the TRD's own two-task structure and file scope internally consistent and keep the git tree clean; neither changes DART-04's scope or the shipped API/rendering behavior. No scope creep.

## Issues Encountered
None beyond the two auto-fixed deviations above, both resolved before their respective task commits.

## User Setup Required
None - the Flutter SDK was already present on this host (`/opt/homebrew/bin/flutter` 3.41.4, Dart 3.11.1) and all dependencies resolved from the public pub.dev registry with no additional credentials or services required.

## Next Objective Readiness
- `bind/dart` is a complete, JS-free, natively-rendering Flutter package ready for a consuming host app to add as a path/git dependency; the android/ios native-artifact vendoring (jniLibs `.so`, podspec `.xcframework`) from 07-03 is the remaining integration step for a real device/simulator build, out of this TRD's file scope.
- `EdenPressView`'s per-block dispatch (`BlockKind` switch) is the extension point for Objective 8's math-fidelity work and any future block kinds schema-v2 adds.
- The widget test's hand-built JSON fixtures are a reusable pattern for testing any future consumer of `Output`/`ModelDocument` without needing a live native/wasm render call.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: bind/dart/pubspec.yaml
- FOUND: bind/dart/lib/eden_press.dart
- FOUND: bind/dart/lib/src/ffi_bindings.dart
- FOUND: bind/dart/lib/src/native_loader.dart
- FOUND: bind/dart/lib/src/web_loader.dart
- FOUND: bind/dart/lib/src/model.dart
- FOUND: bind/dart/lib/src/render_surface.dart
- FOUND: bind/dart/test/render_surface_test.dart
- FOUND commit: 2f6e5ad (Task 1)
- FOUND commit: 4a3aaa4 (Task 2)

---
*Objective: 07-dart-binding*
*Completed: 2026-07-21*
