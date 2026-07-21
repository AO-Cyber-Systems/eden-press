---
objective: 07-dart-binding
job: "02"
subsystem: bindings
tags: [wasm, js-wasm, syscall-js, standard-go, wasm-exec, version-pin, node-smoke, dart-web]

# Dependency graph
requires:
  - objective: 07-dart-binding
    provides: "bind/capi/core.RenderJSON([]byte) []byte — the SINGLE pure-Go JSON marshalling seam (eden-press.capi/v1 envelope), PROVEN in 07-01 to compile under GOOS=js GOARCH=wasm (no import \"C\", no build tags)"
  - objective: 03-press-batteries
    provides: "press.Render(md, Options) (Output, error) — the frozen one-parse-two-sinks API whose full battery chain (goldmark GFM + chroma + latex2mathml + bluemonday) is js-target-safe"
provides:
  - "bind/wasm (package main, //go:build js && wasm) — the DART-03 WEB front door: registers globalThis.pressRender over syscall/js, delegating to core.RenderJSON; standard-Go GOOS=js GOARCH=wasm, no cgo, no second marshalling"
  - "scripts/build-wasm.sh — GOOS=js GOARCH=wasm CGO_ENABLED=0 build emitting bind/wasm/press.wasm + copying $(go env GOROOT)/lib/wasm/wasm_exec.js (the Go 1.24+ path); the artifacts 07-04's WASM RenderFunc and 07-05's Dart web loader consume"
  - "scripts/check-wasm-exec-version.sh — RESEARCH Pitfall 2 guard: fails the build if the checked-in wasm_exec.js drifts from the active toolchain (loader and producing-compiler pinned together)"
  - "bind/wasm/smoke/smoke.mjs — Node boundary gate proving the compiled press.wasm answers through the same JSON entrypoint (# Hi -> <h1; ~~struck~~ -> press <s>)"
  - "bind/wasm/wasm_exec.js — version-pinned Go 1.26.4 loader (tracked baseline for the drift check)"
affects: [07-03-native-builds, 07-04-conformance, 07-05-dart-client]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Three-front-doors-one-core, WEB leg realized (RESEARCH §1 / 07-01): cgo and GOOS=js GOARCH=wasm are mutually exclusive build modes, so the browser target is a SEPARATE package main (bind/wasm) that re-exposes the identical core.RenderJSON JSON entrypoint over syscall/js instead of the cgo //export glue — the web and native boundaries return byte-identical envelopes for the same input."
    - "select{}-kept-alive export: main() registers the JS global then blocks on select{} so the Go program never exits and pressRender stays callable (a returning main tears the export down -> 'pressRender is not a function')."
    - "Loader/compiler version-pin discipline (RESEARCH Pitfall 2): press.wasm (gitignored build product) + a CHECKED-IN wasm_exec.js copied from the SAME toolchain, guarded by a diff gate that fails on drift — the mismatch that otherwise fails silently at runtime is turned into a build-time error."
    - "Executable boundary gate: the boundary is proven by loading the COMPILED artifact under Node and round-tripping real requests through globalThis.pressRender, not by a host-side unit test that never touches the .wasm."

key-files:
  created:
    - bind/wasm/main.go
    - bind/wasm/wasm_exec.js
    - bind/wasm/smoke/smoke.mjs
    - scripts/build-wasm.sh
    - scripts/check-wasm-exec-version.sh
  modified: []

key-decisions:
  - "Standard Go, NOT TinyGo (resolved decision, STATE.md decision gate + RESEARCH §3): the full press chain compiles clean under standard-Go GOOS=js GOARCH=wasm; TinyGo tracks none of chroma/latex2mathml/tdewolff/bluemonday and lists encoding/json as importable-but-not-test-passing, a correctness risk against goldmark/yaml.v3. TinyGo is recorded ONLY as a future size-optimization spike note, never planned."
  - "The smoke asserts the REAL wire key output.HTML (capitalized), not the TRD text's lowercase output.html: press.Output carries no json tags, so its fields cross as Go-default keys (HTML/CSS/Model/Meta/Comments) — verified against 07-01's wire-contract test. A lowercase assertion would never match; the capitalized key is the frozen contract."
  - "Battery assertion uses ~~struck~~ -> <s> (and NOT <del>): this proves the compiled wasm carries the press strikethrough battery AND its <s> override over goldmark GFM's default <del>, matching press_test's proven post-sanitize output — a stronger 'not bare CommonMark' proof than an emoji <img."
  - "wasm_exec.js sourced from $(go env GOROOT)/lib/wasm/ (Go 1.24+), never the removed misc/wasm/ path. The Go 1.24+ lib/wasm loader is the pure embeddable loader that only assigns globalThis.Go (no auto-run block), which is exactly what the Node smoke and the Dart web loader need."

patterns-established:
  - "Loader loaded as a classic script under ESM: smoke.mjs reads wasm_exec.js and executes it via new Function(src)() so globalThis.Go is defined without an import (it is not an ES module). Node LTS supplies globalThis.crypto/performance/TextEncoder/TextDecoder natively, so no polyfill shim is needed."
  - "Version-pin gate is bidirectional-proven: check-wasm-exec-version.sh was exercised against a deliberately stale loader (exit 1) AND the toolchain-matched copy (exit 0) — the guard is demonstrated to actually catch drift, not merely to pass."

requirements-completed: [DART-03]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: true

# Metrics
duration: 6min
completed: 2026-07-21
---

# Objective 7 TRD 02: Web WASM Binding (DART-03) Summary

**A standard-Go `GOOS=js GOARCH=wasm` `syscall/js` shim (`bind/wasm`) that registers `globalThis.pressRender` over the SAME pure-Go `core.RenderJSON` seam the cgo front door uses — plus a version-pinned `wasm_exec.js` loader, a drift-catching version-pin gate, and a Node smoke that round-trips `# Hi` and a strikethrough battery through the compiled `.wasm`, proving the web boundary with zero cgo.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-07-21T16:16:16Z
- **Completed:** 2026-07-21T16:22:15Z
- **Tasks:** 2/2 complete
- **Files modified:** 5 (5 created, 0 modified)

## Accomplishments
- `bind/wasm/main.go` (`package main`, `//go:build js && wasm`): registers `globalThis.pressRender` via `js.Global().Set("pressRender", js.FuncOf(pressRender))`, reads the JSON request from `args[0].String()`, hands it verbatim to `bind/capi/core.RenderJSON` (the shared 07-01 seam — **no cgo, no second marshalling**), returns the JSON response string, and blocks `main()` on `select {}` so the export stays callable. An arity guard returns a well-formed error envelope on a zero-arg call.
- **The full press battery chain compiles clean under standard-Go `GOOS=js GOARCH=wasm CGO_ENABLED=0`** — goldmark GFM + chroma + latex2mathml + bluemonday all js-target-safe; no dependency failed to compile, so there is NO Dart-web-support gap to report as a blocker.
- `scripts/build-wasm.sh`: emits `bind/wasm/press.wasm` and copies `$(go env GOROOT)/lib/wasm/wasm_exec.js` (the Go 1.24+ loader path, NOT the removed `misc/wasm/`) next to it, then invokes the version-pin gate.
- `scripts/check-wasm-exec-version.sh`: diffs the checked-in `bind/wasm/wasm_exec.js` against the active toolchain's loader and exits non-zero on any drift (RESEARCH Pitfall 2 — the silent-runtime-failure mismatch turned into a build-time error). **Proven in both directions** (stale copy -> exit 1; matched copy -> exit 0).
- `bind/wasm/smoke/smoke.mjs`: loads `press.wasm` under Node via `wasm_exec.js`, calls `globalThis.pressRender`, and asserts (1) `# Hi` -> `output.HTML` contains `<h1`, (2) `~~struck~~` -> `output.HTML` contains `<s>` and NOT `<del>` (press strikethrough battery + `<s>` override carried into the wasm). Envelope version `eden-press.capi/v1` asserted on every response.
- `bash scripts/check-no-chromedp.sh` stays **GREEN** — `bind/wasm` adds no chromedp to the pure-Go closure.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: bind/wasm syscall/js shim | `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o /dev/null ./bind/wasm && GOOS=js GOARCH=wasm go vet ./bind/wasm && gofmt -l bind/wasm/main.go` | 0 | PASS |
| 1: (extra) host build unaffected by js&&wasm file | `go build ./... && go vet ./bind/...` | 0 | PASS |
| 2: build-wasm + version-pin + Node smoke | `bash scripts/build-wasm.sh && node bind/wasm/smoke/smoke.mjs && bash scripts/check-wasm-exec-version.sh` | 0 | PASS |
| 2: version-pin catches drift (stale loader) | `printf '// stale\n' >> bind/wasm/wasm_exec.js && bash scripts/check-wasm-exec-version.sh` | 1 | PASS (correctly fails) |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: bind/wasm syscall/js web front door over core.RenderJSON** — `aced805` (feat)
2. **Task 2: build-wasm.sh + version-pinned wasm_exec.js + Node smoke** — `235c54f` (feat)

_Both tasks are `type="auto"` (this TRD is `type: standard`, not tdd)._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build (host) | `go build ./...` | 0 | PASS |
| build (wasm) | `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o /dev/null ./bind/wasm` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (whole repo, incl. bind/capi/core, press/...) | 0 | PASS |
| lint | `gofmt -l bind/wasm/main.go` (empty) + `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| smoke | `bash scripts/build-wasm.sh && node bind/wasm/smoke/smoke.mjs` | 0 | PASS |
| license | `addlicense -l mit -s -c "AO Cyber Systems" -ignore ... -check .` | 0 | PASS |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (both tasks built and verified on the first pass).
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 07-02-TRD.md frontmatter):
  1. `bind/wasm` registers `pressRender` via `js.Global().Set`/`js.FuncOf`, reads the request from JS, calls the SAME `core.RenderJSON`, returns the JSON response, and blocks with `select{}` — no cgo, no second marshalling — ✓ (bind/wasm/main.go; smoke round-trip)
  2. `build-wasm.sh` produces `press.wasm` via standard-Go `GOOS=js GOARCH=wasm CGO_ENABLED=0` and copies `$(go env GOROOT)/lib/wasm/wasm_exec.js` — ✓ (build-wasm.sh run, PASS)
  3. `check-wasm-exec-version.sh` fails on a stale loader, passes on a matched one — ✓ (proven both directions)
  4. `smoke.mjs` loads `press.wasm` under Node via `wasm_exec.js`, calls `pressRender` for `# Hi`, and asserts the parsed response output HTML contains `<h1` (plus a battery case) — ✓ (smoke PASS; asserts the real wire key `output.HTML`)
- **Gate failures:** None.
- **Test-list cases (4/4):** (1) js/wasm build succeeds — PASS; (2) `# Hi` -> `<h1` — PASS; (3) `~~struck~~` battery round-trips as `<s>` — PASS; (4) version-pin catches drift / passes matched — PASS.

## Files Created/Modified
- `bind/wasm/main.go` — `package main` / `//go:build js && wasm`; `main()` registers `pressRender` + `select{}`; `pressRender(this, args)` delegates to `core.RenderJSON` with an arity guard. Doc-comment explains the cgo/wasm mutual-exclusivity split.
- `bind/wasm/wasm_exec.js` — version-pinned Go 1.26.4 loader (upstream BSD header kept; NOT stamped with the Eden MIT header, per the TRD gotcha). Tracked baseline for the drift check.
- `bind/wasm/smoke/smoke.mjs` — Node ESM boundary gate: loads `press.wasm` via `wasm_exec.js`, asserts `# Hi` -> `<h1` and `~~struck~~` -> `<s>`/not-`<del>`, envelope version pinned.
- `scripts/build-wasm.sh` — GOOS=js GOARCH=wasm build + wasm_exec.js copy + version-pin invocation (MIT header, `set -euo pipefail`).
- `scripts/check-wasm-exec-version.sh` — diff gate between checked-in loader and active toolchain (MIT header).

## Decisions Made
- **Standard Go, not TinyGo** — see key-decisions. The full press chain compiles clean under standard-Go js/wasm; TinyGo is a documented future size spike ONLY.
- **Assert `output.HTML` (capitalized), not the TRD's `output.html`** — `press.Output` has no json tags, so the wire keys are Go-default-cased; a lowercase assertion would never match the frozen contract.
- **Strikethrough battery as the round-trip proof (`<s>`, not `<del>`)** — proves both the GFM strikethrough battery and press's `<s>` override are carried into the compiled wasm.
- **`.gitignore` untouched** — `press.wasm` is already covered by the existing `*.wasm` rule (line 9); `git check-ignore bind/wasm/press.wasm` confirms it is ignored, so no redundant entry was added.

## Deviations from Plan

### Auto-fixed / adjusted

**1. [Rule 3 - Wire-key correction] smoke asserts `output.HTML`, not the TRD's lowercase `output.html`**
- **Found during:** Task 2 (writing the smoke assertion)
- **Issue:** The TRD's `must_haves`/test-list phrase the assertion as `parsed.output.html`, but `press.Output` carries NO json tags, so `json.Marshal` emits Go-default-cased keys (`HTML`/`CSS`/`Model`/`Meta`/`Comments`). A `output.html` lookup would be `undefined` and the smoke would falsely fail.
- **Fix:** The smoke asserts `parsed.output.HTML.includes('<h1')` — the real, frozen wire key (verified against 07-01's wire-contract test which redeclares the capitalized keys).
- **Verification:** `node bind/wasm/smoke/smoke.mjs` — PASS.
- **Committed in:** `235c54f`

**2. [Choice] Battery assertion uses `~~struck~~` -> `<s>` rather than the emoji `<img` alternative**
- **Found during:** Task 2
- **Issue:** The TRD offered either `~~x~~` -> `<s>` OR `:smile:` -> twemoji `<img` as the battery case.
- **Fix:** Chose the strikethrough case and additionally asserted the output is NOT goldmark's default `<del>` — a stronger "carries the press batteries, not bare CommonMark" proof, matching press_test's proven post-sanitize output.
- **Verification:** smoke PASS.
- **Committed in:** `235c54f`

**3. [Hygiene] `.gitignore` left unchanged**
- **Found during:** Task 2 (the TRD says "add `bind/wasm/press.wasm` to .gitignore")
- **Issue:** An explicit `bind/wasm/press.wasm` entry would be redundant — the existing `*.wasm` rule already ignores it.
- **Fix:** No edit; confirmed via `git check-ignore bind/wasm/press.wasm` (IGNORED). `press.wasm` never appears in `git status`.
- **Verification:** `git status --short` shows no `press.wasm`.
- **Committed in:** n/a (no change)

---

**Total deviations:** 3 (1 wire-key correction, 1 battery-case choice, 1 gitignore hygiene). None change production behavior — the wasm boundary returns byte-identical envelopes to the native C ABI for the same input.

## Issues Encountered
- The Go module/toolchain is **go1.26.4** (module `go 1.26`), not the `go 1.25.0` the TRD gotcha mentions. Immaterial here: the `lib/wasm/wasm_exec.js` loader path exists in every Go 1.24+, and the version-pin gate pins `press.wasm` and `wasm_exec.js` to whatever toolchain built them (currently 1.26.4). Recorded so a future 07-04/07-05 run on a different toolchain re-runs `build-wasm.sh` to re-pin.
- No press dependency failed to compile under `GOOS=js GOARCH=wasm` — the RESEARCH-flagged risk (a third-party battery breaking on the js port) did NOT materialize, so there is no blocker to report.
- **addlicense stays green with the copied `wasm_exec.js`:** addlicense recognizes the loader's existing upstream BSD header and skips it, so no CI ignore-rule change was needed (confirmed `addlicense ... -check .` exit 0).

## User Setup Required
None — no external service configuration required. The Node smoke needs `node` on PATH (host is v24.13.1, LTS); the wasm build needs a standard Go 1.24+ toolchain (host is go1.26.4). `press.wasm` is a build product (`scripts/build-wasm.sh`), not checked in.

## Next Objective Readiness
- **07-04 (conformance harness):** unblocked — `bind/wasm/press.wasm` + the pinned `wasm_exec.js` are the exact artifacts its WASM `RenderFunc` drives under Node; `scripts/build-wasm.sh` produces them and the smoke pattern shows the load/call/assert sequence.
- **07-05 (Dart web loader):** unblocked — `globalThis.pressRender(jsonString) -> jsonString` is the registered global the Dart side calls via `dart:js_interop`/`package:web`; the `eden-press.capi/v1` envelope and the `output.HTML`-cased wire keys are frozen.
- **07-03 (native builds):** independent — no dependency on this TRD.
- **TinyGo size spike:** documented as a future optimization only (the `.wasm` floors ~2MB uncompressed -> ~500-660KB Brotli; serve pre-compressed and measure real first-load latency at web-deploy time), never planned.
- No blockers.

---
*Objective: 07-dart-binding*
*Completed: 2026-07-21*

## Self-Check: PASSED

- Files verified present: `bind/wasm/main.go`, `bind/wasm/wasm_exec.js`, `bind/wasm/smoke/smoke.mjs`, `scripts/build-wasm.sh`, `scripts/check-wasm-exec-version.sh`, `07-02-SUMMARY.md` — all FOUND. (`bind/wasm/press.wasm` intentionally absent from git — build product, gitignored via `*.wasm`.)
- Commits verified present: `aced805` (Task 1, feat), `235c54f` (Task 2, feat) — both FOUND in git log.
- Gates re-confirmed green: `go build ./...`, `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o /dev/null ./bind/wasm`, `go vet ./...`, `go test ./...` (whole repo, all `ok`), `gofmt -l bind/wasm/main.go` (empty), `bash scripts/check-no-chromedp.sh` (0), `bash scripts/build-wasm.sh` + `node bind/wasm/smoke/smoke.mjs` (smoke PASS), `bash scripts/check-wasm-exec-version.sh` (matched=0, stale=1), `addlicense -check` (0).
