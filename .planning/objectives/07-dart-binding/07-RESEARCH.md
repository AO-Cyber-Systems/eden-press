# Objective 7: Dart/Flutter Binding (bind/capi + bind/dart) - Research

**Researched:** 2026-07-21
**Domain:** Go C-ABI (cgo `c-shared`/`c-archive`) + `GOOS=js/wasm` cross-compilation, consumed by Flutter/Dart via `dart:ffi` and `dart:js_interop`
**Confidence:** HIGH for cgo/build-mode mechanics and dart:ffi platform docs (official sources, dated 2026); MEDIUM-HIGH for the TinyGo-vs-Go decision (TinyGo's own current compatibility table checked directly, but goldmark/chroma/latex2mathml/tdewolff/bluemonday are third-party and untracked by that table — no substitute for a direct build spike); HIGH for the `chase/model` content gap (verified by reading the actual `press/`, `chase/model/`, `chase/directive/` source in this repo, not assumed).

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| DART-01 | Single Go C-ABI core, JSON-in/JSON-out boundary (not mirrored Dart/Go structs) — `PressRender`/`PressFree` | Architecture Patterns §1 (capi core shape, memory-ownership contract); Don't Hand-Roll row 1 |
| DART-02 | Native binding via `dart:ffi` — Android `-buildmode=c-shared`, iOS `-buildmode=c-archive`; no `gomobile bind` | Architecture Patterns §2 (build recipes), §4 (Dart loaders); Pitfall 1 (toolchain isolation, corrected gomobile-panic scope) |
| DART-03 | Web binding via `GOOS=js GOARCH=wasm` + `wasm_exec.js` loader | Architecture Patterns §3 (wasm build, why it is NOT the same cgo shim); Pitfall 2 |
| DART-04 | JS-free Dart rendering surface — math via `flutter_math_fork`, highlight via `highlighting`/`flutter_highlighting` | Architecture Patterns §5 (the `chase/model` content gap and the recommended fix); Open Question 1 — **this is the highest-risk item in the objective** |
| DART-05 | Bindings pass a shared subset of Objective 0's conformance corpus through the SAME JSON entrypoint the Dart code uses | Architecture Patterns §6 (boundary-runner harness shape, grounded in the actual `conformance/runner` code) |
</phase_requirements>

## Summary

This objective wraps the already-frozen `press.Render(md, opts) → press.Output{HTML, CSS, Model, Meta, Comments}` (Objective 3, merged) behind a JSON-in/JSON-out C ABI and compiles that one core three ways: Android `.so` (`-buildmode=c-shared` + NDK clang), iOS `.a` (`-buildmode=c-archive` + `lipo`), and a browser `.wasm` (`GOOS=js GOARCH=wasm`). Two prior research passes (STACK.md/ARCHITECTURE.md/PITFALLS.md) already did the stack-pressure-testing for this; this document extends that work with (a) exact, dated (2026) build/tooling specifics for cgo build modes, `dart:ffi`, and TinyGo, (b) two corrections grounded in this repo's actual source rather than assumption, and (c) an implementation-ready TRD sequencing.

**The single most important finding in this research pass is not in the original brief:** `chase/model.Document` (Objective 2, merged) was deliberately scoped to **only** outline + notes + metadata + attrs — it does **not** carry raw code text/language or raw math TeX source per block (verified by reading `chase/model/document.go`'s own doc comments and `Section`'s field list). `Output.HTML` has the *rendered* forms (chroma-classed `<code>` spans, presentation MathML with no TeX annotation — confirmed by reading `press/sanitize/policy.go`'s MathML allow-list, which has no `annotation`/`semantics` elements). `flutter_math_fork.Math.tex(...)` wants raw TeX, and `flutter_highlighting`'s `HighlightView` wants raw source + a language string — **neither is recoverable, without loss, from `Output.HTML` alone.** DART-04 as specified therefore has an unstated prerequisite: `chase/model` needs a small, additive, schema-versioned enrichment (raw math/code content per block, captured from the same finalized AST `chase/model.Build` already walks — see Architecture Patterns §5) before the Dart rendering surface can be built against JSON alone rather than an HTML/DOM scrape. This is flagged as Open Question 1 and should be resolved as an early Wave-1 decision, not discovered mid-implementation of the Dart package.

**Primary recommendation:** Split `bind/capi` into a pure-Go, no-cgo "core" (`RenderJSON(cmd []byte) ([]byte, error)`, or equivalent) plus two thin target-specific shims — a cgo file with `//export PressRender`/`//export PressFree` for Android/iOS, and a `syscall/js`-based file for the web target — because **cgo `//export` and `GOOS=js GOARCH=wasm` are mutually exclusive** (cgo is disabled on the js/wasm port; there is no C ABI to export into). Default the WASM target to **standard Go, not TinyGo**, for v1 — TinyGo's own current compatibility table (2026-04) shows `encoding/json` as "importable" but explicitly **not** verified to pass its own test suite, and none of goldmark's actual dependency chain here (chroma, latex2mathml, tdewolff/parse, bluemonday, goldmark-highlighting, goldmark-emoji) is tracked by that table at all — TinyGo remains an explicit, gated, opt-in size-optimization spike, never a default.

## Standard Stack

### Core (Go side — already decided in STACK.md, restated as the frozen toolchain for this objective)

| Tool | Version/Channel | Purpose | Why Standard |
|------|------------------|---------|---------------|
| `go build -buildmode=c-shared` | Go 1.25.x toolchain (matches `go.mod`) | Android `.so` per ABI | Core compiler build mode, not an `x/` sub-repo; actively maintained; what `dart:ffi`'s `DynamicLibrary.open()` actually consumes |
| `go build -buildmode=c-archive` | Go 1.25.x toolchain | iOS `.a` (static) | Same rationale; consumed via `DynamicLibrary.process()`/`.executable()` once linked into the app binary |
| `GOOS=js GOARCH=wasm go build` | Go 1.25.x toolchain (standard, **not** TinyGo, for v1 — see decision below) | Web `.wasm` | Confirmed-working pattern for goldmark specifically (goldmark's own `util` package ships a `js`-build-tagged safe fallback — see Architecture Patterns §3); TinyGo is unverified for this exact dependency chain |
| Android NDK (LLVM prebuilt clang, per-ABI) | Pin the exact NDK version in CI | `CC` for `CGO_ENABLED=1` cross-compiles | Required for cgo cross-compilation to Android; version drift between dev machines is a known source of "works on my machine" build failures |
| Xcode command-line tools + `lipo` | Pin the exact Xcode version in CI | iOS per-arch `.a` compile + universal-archive combine | `-buildmode=c-archive` produces one `.a`+`.h` per architecture; `lipo -create` merges device+simulator slices |

### Supporting (Dart/Flutter side — restated from STACK.md, current as of this research date)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `dart:ffi` (SDK-bundled) | Dart 3.12.x (current stable, bundled with Flutter 3.44.6) | Native call boundary | Android `.so` / iOS `.a` consumption |
| `package:ffi` (`calloc`, `Arena`, `toNativeUtf8`/`toDartString`) | pub, current | Scoped native allocation + UTF-8 string marshalling on the Dart side of the boundary | Every FFI call that crosses a string |
| `dart:js_interop` + `package:web` | Dart SDK / pub, current | Web boundary — calling the `syscall/js`-registered global function the compiled `.wasm` exposes | **Not** `dart:html`/`package:js` — those do not compile when the Flutter app itself targets `--wasm` (see State of the Art) |
| `flutter_math_fork` | 0.7.4, Apache-2.0 | Native Dart math rendering (KaTeX-parser port) | Consumes **raw TeX**, not MathML — see Open Question 1 |
| `highlighting` + `flutter_highlighting` | v0.9.0+11.8.0, MIT | Native Dart syntax highlighting (highlight.js port) | Consumes **raw source + language string**, not pre-rendered HTML — see Open Question 1 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Plain `go build -buildmode=c-shared/c-archive` | `golang.org/x/mobile/cmd/gomobile bind` | REJECTED (already decided in STACK.md): generates unneeded JNI/ObjC binding layers for a pure `dart:ffi` consumer; minimally maintained (no tagged v1 release ever) |
| Legacy `plugin_ffi` Flutter template | Newer `package_ffi` + build-hooks template | `package_ffi` is Flutter's new default (since 3.38) for **dynamic** linking, but Flutter's own docs still name `plugin_ffi` as the correct template specifically for **static linking on iOS/macOS** — since this objective needs static linking on iOS, use `plugin_ffi` for the whole `bind/dart` plugin (one template, both platforms) rather than splitting templates per platform |
| Standard Go for `GOOS=js/wasm` | TinyGo | Only if binary size becomes a measured, real product blocker — see decision below; not a v1 default |

**Installation (Dart side):**
```bash
flutter pub add ffi flutter_math_fork highlighting flutter_highlighting
```

## Architecture Patterns

### 1. The capi core: one JSON function, two very different front doors

`DART-01` asks for a single Go C-ABI core with `//export PressRender`/`PressFree`. That is correct for Android/iOS, but **cgo is not available on `GOOS=js GOARCH=wasm` at all** — cgo requires a C compiler and the js/wasm port doesn't support it; there is no `//export`-generated C ABI on that target ([go.dev/blog/wasmexport](https://go.dev/blog/wasmexport); confirmed via WebSearch cross-check of the cgo/wasm relationship, MEDIUM-HIGH — no single official "cgo is unsupported on js/wasm" doc sentence was found, but the wasmexport blog post and multiple `golang/go` issues are consistent on this point). Concretely, structure `bind/capi` as:

- **`bind/capi/core`** (plain Go, no `import "C"`, no build tags): one function, `RenderJSON(cmd []byte) ([]byte, error)` (name illustrative only — no code is prescribed here), that does exactly what the JSON-in/JSON-out contract requires: `json.Unmarshal` into a small request struct (`{Markdown string; Options press.Options}` — mapping directly onto the already-frozen `press.Options`/`press.Output` types in `press/options.go`), call `press.Render`, `json.Marshal` the `press.Output` result. This function is reusable, unit-testable with plain `go test`, and has zero platform-specific code.
- **`bind/capi/cshim`** (`import "C"`, build-tag-free or `//go:build cgo`): the cgo file with `//export PressRender(cmd *C.char) *C.char` and `//export PressFree(p *C.char)`, wrapping `core.RenderJSON` with `C.GoString`/`C.CString` conversions. This file is what `-buildmode=c-shared`/`-buildmode=c-archive` actually compile.
- **`bind/wasm`** (`//go:build js && wasm`): a `package main` using `syscall/js` to register a JS-global function (`js.Global().Set("pressRender", js.FuncOf(...))`) that calls the SAME `core.RenderJSON`, converting between `js.Value`/JS `Uint8Array`/string and `[]byte`. This is a **separate `main` package** from the cgo shim, built with `GOOS=js GOARCH=wasm go build` (CGO_ENABLED=0 required for this target).

This is the one correction the objective brief's framing needs: "the same cgo core builds three ways" is not literally true — the *marshalling logic* is shared (one core function), but the *front door* (cgo export vs. `syscall/js` registration) is necessarily target-specific. Get this split right in Wave 1 and the rest of the objective is mechanical.

**Memory-ownership contract (the load-bearing detail for DART-01/DART-02):**
- `C.CString` allocates on the **C heap via `malloc`**, not Go's GC-managed heap — "it is the caller's responsibility to arrange for it to be freed, such as by calling `C.free`" ([pkg.go.dev/cmd/cgo](https://pkg.go.dev/cmd/cgo), HIGH confidence, official doc, quoted verbatim). `PressRender`'s return value is therefore Go/C-heap memory that **outlives the call** and is invisible to both Go's GC and Dart's GC.
- Dart must copy the returned bytes into a Dart-owned string (`Pointer<Utf8>.toDartString()`, which copies) and then call the exported `PressFree(ptr)` — which must call `C.free(unsafe.Pointer(p))` on the Go side — to release it. **Do not** let Dart call its own `calloc.free()` directly on a Go-allocated pointer: even though both are ultimately libc `malloc`/`free ` in the common case, coupling Dart's allocator choice to Go's internal allocation strategy is exactly the kind of cross-boundary assumption that breaks silently on one platform (e.g., a debug/hardened allocator wrapper on one OS) and works by accident on another. Always round-trip through the explicit exported free function.
- The **input** direction is the mirror image: Dart allocates the input JSON as a native UTF-8 buffer (`toNativeUtf8(allocator: arena)` from `package:ffi`, scoped with `using(...)` / `Arena`) and is responsible for freeing *that* buffer — Go never allocates or frees the input.
- `C` code (including the code Dart's FFI runtime effectively is, from Go's perspective) "may not keep a copy of a Go pointer after the call returns" ([pkg.go.dev/cmd/cgo](https://pkg.go.dev/cmd/cgo), passing-pointers rules) — this is exactly why the C-ABI boundary must copy into malloc'd memory rather than trying to hand back a Go-native string/slice pointer.

### 2. Build recipes

**Android (`-buildmode=c-shared`), per ABI, via the NDK's own clang** (pattern confirmed against multiple independent cross-compilation write-ups; the exact NDK path/binary-name convention is MEDIUM-HIGH — verify the installed NDK's actual `toolchains/llvm/prebuilt/<host>/bin/<triple><api>-clang` path before pinning in CI, since it varies by NDK version):
```bash
export ANDROID_NDK_HOME=/path/to/ndk
export NDK_BIN="$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/<host-tag>/bin"
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
  CC="$NDK_BIN/aarch64-linux-android21-clang" \
  go build -buildmode=c-shared -o android/jniLibs/arm64-v8a/libpress.so ./bind/capi
# repeat per ABI: armeabi-v7a (armv7a-linux-androideabi21-clang, GOARCH=arm GOARM=7),
#                 x86_64 (x86_64-linux-android21-clang), x86 (i686-linux-android21-clang)
```

**iOS (`-buildmode=c-archive`), per-arch then `lipo`-combined** (device arm64 shown; add simulator arm64/x86_64 slices the same way with the simulator SDK/clang wrapper, then merge):
```bash
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC=$(go env GOROOT)/misc/ios/clangwrap.sh \
  CGO_CFLAGS="-fembed-bitcode" \
  go build -buildmode=c-archive -tags ios -o build/ios-arm64/libpress.a ./bind/capi
lipo -create -output libpress.a build/ios-arm64/libpress.a build/ios-sim-arm64/libpress.a build/ios-sim-x86_64/libpress.a
```
Package the merged `.a` + header into an `.xcframework` (or a plain static-library target) consumed by the `plugin_ffi` template's iOS podspec `source_files`/`vendored_libraries`.

**Web (`GOOS=js GOARCH=wasm`)**:
```bash
GOOS=js GOARCH=wasm go build -o bind/wasm/press.wasm ./bind/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" bind/wasm/wasm_exec.js
```
`wasm_exec.js` moved from `misc/wasm/` to `lib/wasm/` in current Go releases ([go.dev/wiki/WebAssembly](https://go.dev/wiki/WebAssembly), HIGH confidence, official). The major Go version of the compiler and of `wasm_exec.js` **must match** — pin both together in CI, not just the compiler.

**Toolchain-panic mitigation, corrected scope (important honest-reporting note):** the specific "confirmed toolchain-panic case" the objective brief references (`golang/go#47296`) is a bug in **`gomobile bind`'s own `env.go`** (`archNDK()` panics with `"panic: unsupported GOARCH: arm64"` on Apple-Silicon hosts, because gomobile's iOS build path unconditionally tries to set up Android NDK cross-compilation when `ANDROID_NDK_HOME` is present) — verified directly against the GitHub issue, which is closed and scoped entirely to gomobile's own code, not to plain `go build -buildmode=c-archive`. Since this objective already excludes `gomobile bind` (DART-02), this **exact** panic will not reproduce. That said, the underlying hygiene concern is still real and worth keeping as CI policy: don't assume a shared CI runner's leftover `CC`/`CGO_ENABLED`/NDK environment variables from an Android job won't leak into a subsequent iOS job on the same machine. **Recommendation:** run Android and iOS as two fully isolated CI pipelines (separate jobs/containers, no shared environment), which is good practice regardless of whether the specific gomobile bug applies.

### 3. Standard Go vs. TinyGo for the WASM target — the resolved decision gate

**Recommendation: standard Go for v1.** TinyGo is an explicit, later, opt-in size-optimization spike — not a default.

**Evidence, broken into what's actually verified vs. what remains genuinely unknown:**

- **goldmark itself is confirmed `js`-target-safe, directly from its vendored source in this repo's module cache** (`github.com/yuin/goldmark@v1.8.4/util/util_safe.go` vs. `util_unsafe_go121.go`): goldmark's zero-copy `unsafe.String`/`unsafe.Slice` string-conversion fast path is gated `!appengine && !js`, with an explicit `appengine || js`-tagged safe (copying) fallback. This means goldmark's own maintainers already anticipated and handled `GOOS=js` — regardless of standard Go or TinyGo, since TinyGo's browser wasm target **also** sets `GOOS=js GOARCH=wasm` ([tinygo.org/docs/guides/webassembly/wasm](https://tinygo.org/docs/guides/webassembly/wasm/): `GOOS=js GOARCH=wasm tinygo build`). HIGH confidence — read directly, not inferred.
- **Front-matter parsing is NOT a yaml.v3/reflection risk in this codebase — correcting the objective brief's stated concern.** `chase/directive/frontmatter.go` + `yaml.go` implement a hand-rolled, line-based "YAML-ish" scanner (`strings.Split`/`strings.IndexByte`, mirroring Marpit's own restricted front-matter subset) — there is **no** `gopkg.in/yaml.v3` import anywhere in `go.mod`'s direct or indirect requires that eden-press actually uses for this (the one `yaml.v3` line in `go.sum` is a stale transitive `go.mod`-only reference from an unrelated dependency graph, never compiled in). Zero reflection, zero third-party YAML library, in this specific code path. HIGH confidence, verified by reading the source.
- **The real reflection-dependent risk is `chase/model.Document`'s JSON marshalling**, and here the picture is genuinely mixed, not resolved: TinyGo's own current official compatibility table (dated 2026-04-20, "TinyGo 0.41.0/Go 1.26.2") lists `encoding/json` as **"Importable: yes" but "Passes tests: no"**, and `reflect` as both "yes"/"yes" ([tinygo.org/docs/reference/lang-support/stdlib](https://tinygo.org/docs/reference/lang-support/stdlib/), HIGH confidence — official, current, directly fetched). The page's own disclaimer: importability "does not mean that all functions and types in the program can be used." This is a **materially more nuanced and more current** picture than the older "TinyGo can't do JSON" folklore (2020-2023-era blog posts/HN threads), but it is not a green light either — it's an honest "unresolved, verify yourself" signal from TinyGo's own maintainers.
- **chroma, latex2mathml, `tdewolff/parse`, bluemonday, goldmark-highlighting, goldmark-emoji are all third-party and untracked by TinyGo's official table entirely** — there is no authoritative signal, positive or negative, for any of them. `press.Render` composes all of them into one engine; a TinyGo build must successfully compile and correctly execute the **entire** chain, not goldmark in isolation.
- **A real, working precedent exists for goldmark + standard-Go WASM in production** (a client-side Markdown editor, `milinddethe15/render-md`, using Goldmark + Go + WASM + highlight.js, MIT-licensed, found via WebSearch — MEDIUM confidence, a community project not an official reference, but concrete evidence the exact goldmark-to-standard-Go-wasm path is proven, unlike TinyGo-plus-the-full-press-chain which has no such precedent found).
- An older, closed TinyGo issue (#1848, 2021) reported a stack-overflow panic importing `goldmark/parser` under TinyGo-wasm; it is stale (TinyGo has changed substantially since) but underscores that goldmark-under-TinyGo specifically has not been a smooth, well-trodden path historically, unlike goldmark-under-standard-Go.

**If TinyGo is revisited later:** budget it as a direct build-and-run spike against the **actual `press` package** (not a toy program) — specifically, round-trip `json.Marshal`/`Unmarshal` of `press.Options`/`press.Output` (and, once Open Question 1 is resolved, the enriched `Model`) through a `tinygo build -target wasm` binary against a real conformance-corpus subset — before trusting it for anything beyond a size experiment. Pin TinyGo's own `wasm_exec.js` (`$(tinygo env TINYGOROOT)/targets/wasm_exec.js`) to the exact TinyGo version — it is "based on" but explicitly **not interchangeable with** standard Go's `wasm_exec.js` ([tinygo.org/docs/guides/webassembly/wasm](https://tinygo.org/docs/guides/webassembly/wasm/), HIGH confidence, official, current).

**Binary size, for context (standard Go, the chosen path):** a Go/Wasm binary floors around ~2MB+ uncompressed, compressing to roughly 500-660KB with gzip or ~2.4-3.4MB→496-660KB range depending on payload (Brotli consistently smallest) ([go.dev/wiki/WebAssembly](https://go.dev/wiki/WebAssembly), HIGH confidence, official, with concrete numbers). Serve the `.wasm` pre-compressed with Brotli; measure real first-load latency rather than assuming this is acceptable.

### 4. Dart side: loaders and the plugin-template choice

- **Android:** `DynamicLibrary.open('libpress.so')` — dynamic, one `.so` per ABI in `android/src/main/jniLibs/<abi>/`.
- **iOS:** `DynamicLibrary.process()` (or `.executable()`) — the `.a` is statically linked into the app binary at build time via the plugin's podspec, not loaded on demand.
- **Template choice:** Flutter's docs (fetched 2026-06-08, reflecting Flutter 3.44.0) recommend the newer `package_ffi` + build-hooks template as the general default *since Flutter 3.38*, but explicitly carve out an exception: "the legacy FFI plugin template (`plugin_ffi`) ... is still useful if you need to ... use static linking (on iOS and macOS)" ([docs.flutter.dev/platform-integration/ios/c-interop](https://docs.flutter.dev/platform-integration/ios/c-interop), HIGH confidence, official, dated). Because this objective needs static linking on iOS, **use `plugin_ffi` for the whole `bind/dart` package** — mixing templates per-platform for one logical plugin is avoidable complexity; `plugin_ffi` fully supports Android's dynamic `.so` case too, it's simply not Flutter's new *default* recommendation there. A concrete iOS release-build gotcha to carry into CI docs: Xcode strips symbols on release builds by default (Build Settings → Strip Style: change from "All Symbols" to "Non-Global Symbols"), or `DynamicLibrary.process()` symbol lookups fail at runtime with no build-time warning.
- **Web:** the compiled `.wasm` + `wasm_exec.js` are loaded as ordinary web assets; the Go side registers its function on the JS global object via `syscall/js` (see §1); the Dart side calls it via **`dart:js_interop`/`package:web`, not `dart:html`/`package:js`.** This is not just future-proofing: Flutter's own Wasm compilation target (`flutter build web --wasm`, via `dart2wasm`) is now **stable** (graduated from experimental) and Flutter's 2026 roadmap has it heading toward becoming the *default* web output ([docs.flutter.dev/platform-integration/web/wasm](https://docs.flutter.dev/platform-integration/web/wasm), MEDIUM-HIGH — official page, cross-checked via WebSearch summary). `dart:html`/`package:js` do not compile at all under `--wasm` output; `dart:js_interop`/`package:web` work under both `--wasm` and the still-default `dart2js` output, so it is the only choice that doesn't foreclose Flutter's own roadmap direction. Note this is a **separate** Wasm concern from the Go core's own `.wasm` — one is "what does the Flutter app itself compile to," the other is "what does our Go core compile to"; both interoperate through the same JS-global mechanism regardless of which the Flutter app chose.

### 5. The `chase/model` content gap — the resolution DART-04 actually needs

Verified directly against this repo's source (not assumed):

- `chase/model.Section` (in `chase/model/document.go`) carries only `ID`, `Attrs`, `Notes` — its own doc comment states "Blocks/HTML content are deliberately NOT part of this shape" (a correct, intentional scope decision for Objective 2/MODEL-01, which only needed outline+notes+meta+attrs).
- `press/math`'s custom AST node (`mathNode` in `press/math/math.go`) carries `Raw` (the original LaTeX string) and `Block` (display vs. inline) **before** any HTML/MathML rendering happens — this is exactly the value DART-04 needs, and it already exists in the parse tree `chase/model.Build` walks.
- `Output.HTML`'s MathML has no path back to that raw TeX: `press/sanitize/policy.go`'s MathML element allow-list (`math, mrow, mi, mn, mo, mtext, mspace, mstyle, ...`) contains no `annotation`/`semantics` wrapper, and `latex2mathml` (the vendored converter) does not appear to emit one — presentation MathML alone is lossy relative to the original TeX syntax.
- Code blocks are less lossy (goldmark's `ast.FencedCodeBlock` carries the raw source text and the language info-string directly, pre-chroma), but recovering them from `Output.HTML` would still require DOM-parsing chroma-classed `<span>` soup back into plain text — solvable, but exactly the kind of "reverse-engineer structure from rendered HTML" pattern `chase/model`'s own design doc explicitly rejects elsewhere ("outline/notes/metadata WITHOUT parsing rendered HTML back out of HTML").

**Recommendation:** extend `chase/model` with an additive, schema-versioned enrichment (bump `SchemaVersion` from `"eden-press.model/v1"`) that captures, per Section, an ordered list of raw content blocks (kind: text/code/math; for code: source + language; for math: raw TeX + display-mode) — captured directly from the same finalized AST during `Build()`, the same way `Section`/`Outline` already are. This is a natural continuation of the existing "one-parse-two-sinks" architecture (Model is sink 2, built from the AST, never from rendered HTML), not a new pattern. It should land as an early task in this objective (or an immediately-preceding fast-follow to Objective 2/3) — DART-04's Dart widgets have nothing correct to consume from `Model` alone until this exists, and building a parallel HTML-DOM-scraping path in Dart as a permanent solution would be exactly the "pure-Dart reimplementation as default strategy" anti-pattern ARCHITECTURE.md already warns against, just relocated to the JSON-consumption side instead of the parsing side.

### 6. DART-05: the boundary-runner harness, grounded in the actual conformance code

`conformance/runner/runner.go` already defines the exact seam this needs:
```go
type RenderFunc func(markdown string, opts map[string]any) (html string, err error)
```
`RunCase` (same file) renders a `corpus.Case` through any `RenderFunc` and DOM-normalizes the result via `htmldiff.Equal` — it is already engine-agnostic; Objective 0's spec sweep and Marp-corpus runner (`conformance/runner/engine.go`'s `NewGoldmark*`/`GoldmarkRenderFunc`) are just two existing implementations of this same seam.

**DART-05's job is a third (and fourth) `RenderFunc` implementation** that calls the compiled artifact through its JSON entrypoint instead of calling Go code in-process, reusing `RunCase`/`htmldiff`/`report` completely unchanged:
- **WASM boundary test — the easiest of the three to automate, start here.** Go's own toolchain can *execute* a `GOOS=js GOARCH=wasm`-built test binary directly via its `go_js_wasm_exec` wrapper (which shells out to Node), so a `RenderFunc` that spins up the compiled `press.wasm` under Node (or any WASM runtime with JS-glue support) and calls the registered `pressRender`-equivalent function is a same-CI, no-emulator, no-device path.
- **capi (native) boundary test — needs a host-arch rebuild trick, not the mobile artifacts directly.** The actual Android `.so`/iOS `.a` are built for ARM device ABIs and cannot execute on a generic amd64/arm64 Linux/macOS CI runner. The fix is cheap: build `bind/capi` a **fourth** way — a plain host-arch `-buildmode=c-shared` (e.g., `linux/amd64` or the CI runner's native OS/arch) — purely for this conformance lane. This validates the exact same cgo JSON-marshalling shim code (`bind/capi/cshim` + `bind/capi/core`) that the mobile builds use, without needing an Android emulator or an iOS simulator in the loop; per-device/emulator smoke tests (a small, non-corpus-scale subset) remain a separate, later CI lane for validating the actual mobile `.so`/`.a` artifacts themselves.
- Both `RenderFunc` implementations should run the **same subset** of the Objective 0 corpus (DART-05 says "a shared subset," not the full sweep) — pick a subset that exercises every battery `press.Render` composes (strikethrough, emoji, highlight, math, autofit, sanitize), not just plain CommonMark, since the boundary being tested is JSON marshalling of the *whole* `Output` shape, not just HTML string equality.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Android/iOS native binding generation | A custom JNI/ObjC stub generator, or `gomobile bind` | Plain `go build -buildmode=c-shared`/`-buildmode=c-archive` + `dart:ffi` | `dart:ffi` talks the C ABI natively — no stub layer needed; `gomobile bind` solves a problem (idiomatic per-language bindings) this objective doesn't have |
| Web JS interop shim for calling into Go/Wasm | A bespoke Dart↔JS message-passing protocol | `dart:js_interop`/`package:web` calling the `syscall/js`-registered global that Go's own `wasm_exec.js` sets up | The loader is a Go-shipped, Go-versioned artifact — same "external toolchain output, not our code" framing as Chrome discovery elsewhere in this project |
| Native Dart math typesetting | A from-scratch TeX-subset renderer in Dart | `flutter_math_fork` (direct KaTeX-parser port) | Real engine port with KaTeX-fidelity goal; no better-maintained pure-Dart alternative surfaced |
| Native Dart syntax highlighting | A from-scratch tokenizer per language | `highlighting` + `flutter_highlighting` (highlight.js port, ~190+ languages) | Breadth matches "highlight whatever fenced-code language an author wrote," which a curated/small hand-rolled set would not |
| JSON marshalling under a future TinyGo build, if `encoding/json` proves unreliable | A bespoke reflection-free encoder | `tinyjson`/`json-iterator/tinygo`/`go-json-ice` (existing TinyGo-targeted codegen JSON libraries) | The ecosystem has already converged on codegen-based JSON for TinyGo specifically because reflection there is unreliable — don't reinvent this if the TinyGo spike is ever greenlit |

**Key insight:** every "port this to Dart natively" requirement in this objective (math, highlighting) already has a direct, fidelity-matched Dart port of the exact same upstream library this project's Go side already committed to (KaTeX-lineage math, highlight.js-lineage code highlighting) — the risk in this objective is never "which library," it's "what shape of data does that library need, and does the JSON boundary actually carry it" (Architecture Patterns §5).

## Common Pitfalls

### Pitfall 1: Treating the cgo core as literally "the same thing" the WASM build compiles
**What goes wrong:** A `bind/capi` package written with `import "C"` and `//export` directives fails to build at all under `GOOS=js GOARCH=wasm` (cgo is unavailable on that port), discovered only when someone tries to wire the WASM build late in the objective.
**Why it happens:** The objective's own framing ("one Go C-ABI core... built three ways") reads as if it's one source tree compiled three ways with only `GOOS`/`GOARCH`/`-buildmode` flags changing — true for Android/iOS, false for Web.
**How to avoid:** Structure `bind/capi` as core (no cgo) + cgo shim + separate `syscall/js` shim from the start (Architecture Patterns §1).
**Warning signs:** Any attempt to `GOOS=js GOARCH=wasm go build ./bind/capi/...` on the cgo-shim package specifically.

### Pitfall 2: Assuming the two `wasm_exec.js` files (standard Go vs. TinyGo) are interchangeable
**What goes wrong:** "undefined function" JS errors at runtime after a compiler version bump, or after accidentally shipping the wrong loader.
**Why it happens:** TinyGo's `wasm_exec.js` is explicitly described by TinyGo's own docs as "based on" but "slightly different" from Go's — a copy-paste of the wrong one, or a version mismatch with whichever compiler produced the `.wasm`, breaks silently until runtime.
**How to avoid:** Pin the loader file to the exact producing compiler and version in CI (fail the build if they drift); this applies whether or not TinyGo is ever adopted, since standard Go's own `wasm_exec.js`/compiler-version pairing has the identical constraint.
**Warning signs:** `.wasm` and `wasm_exec.js` sourced from different build steps/caches without an explicit version check.

### Pitfall 3: Building DART-04 against `Output.HTML` because it's "already there"
**What goes wrong:** A first implementation attempt DOM-parses `Output.HTML` in Dart to extract code/math content, working for code (lossy but recoverable) and silently wrong or incomplete for math (not recoverable — MathML has no annotation-preserved TeX in this pipeline).
**Why it happens:** `Output.HTML` is the most complete-looking field in the JSON payload, and reaching for a Dart HTML-parsing package feels like the path of least resistance compared to touching an already-merged, "Complete" Objective 2 package.
**How to avoid:** Resolve Open Question 1 first — enrich `chase/model` with raw math/code content per block before starting the Dart rendering-surface work, not after discovering the gap mid-implementation.
**Warning signs:** A Dart dependency on any HTML/DOM parsing package appearing in `bind/dart`'s `pubspec.yaml`.

### Pitfall 4: Trying to conformance-test the actual mobile `.so`/`.a` artifacts directly in generic CI
**What goes wrong:** A DART-05 boundary-runner task stalls trying to "run the Android .so" on a Linux CI container, or "run the iOS .a" without a Mac/simulator, and either gets abandoned or silently narrows to WASM-only coverage.
**Why it happens:** The mobile artifacts are built for ARM device/emulator ABIs that don't execute on typical CI hardware architectures.
**How to avoid:** Build a fourth, host-arch `-buildmode=c-shared` specifically for the conformance boundary lane (Architecture Patterns §6); keep actual on-device/emulator testing as a separate, smaller smoke-test lane.
**Warning signs:** DART-05 "passing" in CI while only ever having exercised the WASM `RenderFunc`.

### Pitfall 5: Citing `golang/go#47296` as a plain-cgo bug
**What goes wrong:** CI mitigation work targets a bug that can't reproduce in this project's actual toolchain, wasting effort or misdirecting root-cause analysis if an unrelated iOS build failure occurs on an Apple-Silicon+NDK machine.
**Why it happens:** Prior research (PITFALLS.md Pitfall 9) cited this issue in a way that reads as a generic "c-shared build" risk.
**How to avoid:** The issue is scoped entirely to `gomobile bind`'s own `env.go` (`archNDK()` panicking because gomobile's iOS path unconditionally probes for an Android NDK) — verified directly against the issue text. Since `gomobile bind` is already excluded (DART-02), this specific panic does not apply. Still isolate Android/iOS CI pipelines as general hygiene (Architecture Patterns §2), but don't over-cite this specific closed issue as the reason.
**Warning signs:** N/A — this is a documentation-accuracy pitfall, not a runtime one.

## Code Examples

No application code is prescribed by this research (feeds the planner, not the implementer). The build-recipe shell commands in Architecture Patterns §2 are the concrete, citable artifacts for this objective; treat them as the starting point, not literal copy-paste-ready CI YAML.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| `golang.org/x/mobile/cmd/gomobile bind` for Go↔Dart native binding | Plain `go build -buildmode=c-shared`/`-buildmode=c-archive` + `dart:ffi` | Already decided (STACK.md) | Avoids gomobile's own toolchain-panic bug class and unneeded JNI/ObjC stub generation entirely |
| `$GOROOT/misc/wasm/wasm_exec.js` | `$GOROOT/lib/wasm/wasm_exec.js` | Go 1.24 | Any CI script or doc still referencing `misc/wasm/` is stale |
| Legacy `plugin_ffi` Flutter FFI template as the default for all C interop | `package_ffi` + build hooks as the new default, **except** static linking on iOS/macOS, where `plugin_ffi` remains correct | Since Flutter 3.38 (current docs dated 2026-06-08, reflecting Flutter 3.44.0) | This objective's iOS static-linking need means `plugin_ffi` is still the right template — don't reflexively "upgrade" to `package_ffi` project-wide |
| Flutter Web WebAssembly (`dart2wasm`) as experimental, opt-in only | Stable (`flutter build web --wasm`), with automatic JS fallback per-browser, roadmapped toward becoming the *default* web output in 2026 | Graduated to stable within the 2026 cycle | Reinforces `dart:js_interop`/`package:web` (not `dart:html`/`package:js`) as the correct, forward-compatible choice for the Dart-side web loader, independent of the Go-core's own separate `GOOS=js/wasm` compile |
| "TinyGo can't do `encoding/json`" (2020-2023-era community consensus) | TinyGo's own official table (2026-04, TinyGo 0.41.0/Go 1.26.2): `encoding/json` importable, but explicitly not verified to pass its own tests; `reflect` now both importable and passing tests | Ongoing, actively improving | Softens but does not resolve the risk — still not a green light without a direct spike against this project's actual dependency chain |
| `go:wasmexport` framed as a general "export functions from Wasm" mechanism | Confirmed scoped to `GOOS=wasip1 GOARCH=wasm` (Go 1.24+), not the browser `GOOS=js GOARCH=wasm` port this objective needs | Go 1.24 | Not usable for DART-03 as a substitute for `syscall/js` registration — the browser target still requires the `js.Global().Set`/`js.FuncOf` pattern |

**Deprecated/outdated:**
- `misc/wasm/wasm_exec.js` path (moved to `lib/wasm/`) — any reference to the old path in scripts/docs should be corrected.
- Treating `golang/go#47296` as evidence against plain `go build -buildmode=c-archive` — it is gomobile-specific and closed.

## Open Questions

1. **Does `chase/model` need a schema-v2 enrichment (raw math/code content per block) before DART-04's Dart rendering surface can be built, and if so, is that enrichment task inside Objective 7 or a prerequisite fast-follow to Objective 2/3?**
   - What we know: `Output.HTML`/`Model` as they exist today (both "Complete," merged) do not carry raw TeX or raw code+language in a JSON-native, lossless form; the raw values exist transiently in the AST `chase/model.Build` already walks.
   - What's unclear: whether the planner should scope this as Objective 7's first task (touching an already-merged package) or flag it as a small, separate, preceding objective/patch, given it changes a "Complete" package's public JSON shape (additive, but still a contract change consumers should be told about via the `SchemaVersion` bump).
   - Recommendation: scope it inside Objective 7, Wave 1, as an explicit prerequisite task — it blocks DART-04 correctness either way, and doing it in isolation from the rest of the Dart work risks the same information getting rediscovered mid-implementation.

2. **Should `bind/capi`'s JSON request/response schema be versioned independently of `press.Options`/`press.Output`'s own Go-level stability, or treated as a direct 1:1 JSON projection with no separate version field?**
   - What we know: `press.Options`/`press.Output` are the frozen API-03 surface (Objective 3); `chase/model.Document` already carries its own `SchemaVersion` string.
   - What's unclear: whether the capi JSON envelope needs its own version field (e.g., to let a Dart client detect "server/core built with an older schema" independent of the Model's own version), or whether piggybacking on `Model.SchemaVersion` is sufficient.
   - Recommendation: lean toward a thin, explicit envelope version at the capi JSON layer too — it's cheap now and avoids an ambiguous "which layer changed" debugging question later, but this is a low-stakes decision either way.

3. **Exact current NDK version and Xcode version to pin in CI.**
   - What we know: the pattern (per-ABI clang invocation with an API-level-suffixed triple) is stable and well-documented.
   - What's unclear: this research did not pin a specific NDK/Xcode version number, since that should be chosen against the project's actual minimum-supported OS versions, not researched in the abstract.
   - Recommendation: pin explicitly in Objective 7's Wave 2 CI setup, verified against whatever Android/iOS minimum-version support the project already commits to elsewhere.

## Sources

### Primary (HIGH confidence)
- This repository, read directly: `press/press.go`, `press/options.go` (frozen `Options`/`Output` shape), `chase/model/document.go` + `chase/model/build.go` (Model's actual scope, `SchemaVersion`), `press/math/math.go` (raw TeX carried pre-render), `press/sanitize/policy.go` (MathML allow-list, no annotation/semantics), `press/highlight.go` (chroma wiring), `chase/directive/frontmatter.go` + `chase/directive/yaml.go` (hand-rolled front-matter parser, no yaml.v3), `conformance/runner/runner.go` + `engine.go` (the existing `RenderFunc` seam DART-05 extends)
- `github.com/yuin/goldmark@v1.8.4` (local module cache), read directly: `util/util_safe.go`, `util/util_unsafe_go120.go`, `util/util_unsafe_go121.go` (confirms `js`-build-tag-safe fallback, no reflect dependency)
- [pkg.go.dev/cmd/cgo](https://pkg.go.dev/cmd/cgo) — `C.CString`/`C.GoString` semantics, malloc-ownership rules, pointer-passing restrictions, quoted directly
- [go.dev/wiki/WebAssembly](https://go.dev/wiki/WebAssembly) — `GOOS=js GOARCH=wasm` build steps, `wasm_exec.js` location (`lib/wasm/`), binary-size figures
- [go.dev/blog/wasmexport](https://go.dev/blog/wasmexport) — `go:wasmexport`, scoped to `GOOS=wasip1`, not js/wasm
- [github.com/golang/go/issues/47296](https://github.com/golang/go/issues/47296) — the toolchain-panic issue, read directly: confirmed gomobile-specific, closed
- [tinygo.org/docs/guides/compatibility](https://tinygo.org/docs/guides/compatibility/) — general reflect/encoding-json limitation framing
- [tinygo.org/docs/reference/lang-support/stdlib](https://tinygo.org/docs/reference/lang-support/stdlib/) — current (2026-04-20, TinyGo 0.41.0/Go 1.26.2) per-package importable/passes-tests table, fetched directly
- [tinygo.org/docs/guides/webassembly/wasm](https://tinygo.org/docs/guides/webassembly/wasm/) — TinyGo's browser wasm build command, its own `wasm_exec.js` location and non-interchangeability
- [docs.flutter.dev/platform-integration/ios/c-interop](https://docs.flutter.dev/platform-integration/ios/c-interop) — `plugin_ffi` vs `package_ffi`, static-linking exception, Xcode strip-style gotcha; dated 2026-06-08, reflects Flutter 3.44.0
- [docs.flutter.dev/platform-integration/android/c-interop](https://docs.flutter.dev/platform-integration/android/c-interop) — Android dynamic-`.so`-only confirmation, `DynamicLibrary.open`
- [docs.flutter.dev/platform-integration/web/wasm](https://docs.flutter.dev/platform-integration/web/wasm) — Flutter Wasm stable status, `dart:html`/`package:js` incompatibility under `--wasm`

### Secondary (MEDIUM confidence)
- WebSearch cross-checks (verified against the primary sources above, not standalone): cgo unsupported on js/wasm (consistent across `go:wasmexport` blog + multiple `golang/go` issues, no single canonical "unsupported" sentence found); TinyGo `encoding/json` ecosystem workarounds (`tinyjson`, `json-iterator/tinygo`, `go-json-ice`); Flutter/Dart current stable version (3.44.6 / Dart 3.12.2, cross-checked against flutterreleases.com); cross-compilation command patterns for Android NDK/iOS `clangwrap.sh` (consistent across multiple independent write-ups, no single official "how to cross-compile cgo for Android/iOS" doc page found)
- `milinddethe15/render-md` (GitHub) — a real, working goldmark+Go+standard-Go-WASM project, cited as existence-proof for that specific combination, not an authoritative reference

### Tertiary (LOW confidence)
- TinyGo issue #1848 (2021, closed, stack-overflow importing `goldmark/parser`) — stale, TinyGo has changed substantially since; kept only as historical context that goldmark-under-TinyGo has not been a smooth path

## Metadata

**Confidence breakdown:**
- cgo build modes / dart:ffi platform mechanics: HIGH — official docs, dated 2026, cross-checked
- TinyGo-vs-Go WASM decision: MEDIUM-HIGH — TinyGo's own stdlib table checked directly and current, but the actual third-party dependency chain (chroma/latex2mathml/tdewolff/bluemonday) has zero official TinyGo compatibility signal; the recommendation (standard Go) is well-supported, but "TinyGo would definitely fail" is not proven, only "TinyGo is unverified"
- `chase/model` content gap for DART-04: HIGH — verified by reading this repo's actual source, not assumed or inferred from the objective brief
- DART-05 harness shape: HIGH — grounded directly in the existing `conformance/runner` code, which already defines the exact seam needed

**Research date:** 2026-07-21
**Valid until:** ~30 days for the Flutter/Dart/TinyGo version-specific facts (fast-moving ecosystem — Flutter/Dart/TinyGo all ship on regular cadences); the cgo/build-mode mechanics and the `chase/model` gap finding are stable facts about this codebase and don't expire on a calendar.

---
*Research for: Eden Press Objective 7 — Dart/Flutter Binding (bind/capi + bind/dart)*
