---
objective: 07-dart-binding
job: "01"
subsystem: bindings
tags: [cgo, ffi, c-abi, json-envelope, dart, wasm-shared-core, capi]

# Dependency graph
requires:
  - objective: 03-press-batteries
    provides: "press.Render(md, Options) (Output, error) — the frozen one-parse-two-sinks public API (API-01/03); press.Options input surface (Sanitize *bluemonday.Policy deliberately non-serializable) and press.Output shape (HTML/CSS/Model/Meta/Comments)"
  - objective: 02-model-profile
    provides: "chase/model.Document with json tags + SchemaVersion \"eden-press.model/v1\" — marshals for free inside the response envelope"
provides:
  - "bind/capi/core.RenderJSON([]byte) []byte — the SINGLE pure-Go JSON marshalling seam over press.Render; versioned envelope (eden-press.capi/v1); always well-formed, never nil/empty, never lets a panic escape"
  - "bind/capi (package main) cgo front door: //export PressRender / //export PressFree with a documented FFI memory-ownership contract"
  - "scripts/build-capi-host.sh — host-arch c-shared build emitting libpress.so + libpress.h (literal .so name on all OSes); the artifact 07-04's conformance lane reuses"
  - "The eden-press.capi/v1 request/response envelope schema, frozen for 07-02 (wasm shim), 07-04 (conformance harness), and 07-05 (Dart client)"
affects: [07-02-wasm-shim, 07-03-native-builds, 07-04-conformance, 07-05-dart-client]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Three-front-doors-one-core split (RESEARCH §1): the marshalling logic (core.RenderJSON) is pure Go / no-cgo / no-build-tags so the SAME function compiles into BOTH the cgo c-shared library (this TRD) and 07-02's standard-Go GOOS=js GOARCH=wasm module — cgo and js/wasm are mutually exclusive build modes"
    - "Uniform JSON boundary: errors are FOLDED into a versioned response envelope (never a Go error / nil slice across the ABI), so every front door has one parseable return shape; a hardcoded marshalFailFallback guarantees non-nil even if marshalling the error itself fails"
    - "FFI memory-ownership documented at the export site: caller owns+frees the input buffer; Go allocates the output on the C heap via C.CString; caller copies out then hands the SAME pointer to PressFree (C.free) — allocators never crossed"
    - "renderFn package-var seam (mirrors press.go's parseWithEngine idiom) makes the recover-guard deterministically testable without coupling to a fragile go-latex panic construct"

key-files:
  created:
    - bind/capi/core/render.go
    - bind/capi/core/render_test.go
    - bind/capi/doc.go
    - bind/capi/capi.go
    - scripts/build-capi-host.sh
  modified:
    - scripts/check-no-chromedp.sh
    - .gitignore

key-decisions:
  - "bind/capi/doc.go is `package main`, NOT the TRD's parenthetical `package capi`: a directory holds exactly one package name, and capi.go must be `package main` (c-shared/c-archive mandate). The two files form one `package main`; doc.go carries the package doc, capi.go the export glue."
  - "Test-list case 6 (panic recovery) is realized via a renderFn seam-injection (deterministic proof RenderJSON recovers a panicking render) PLUS a real \\begin{aligned} construct asserted to yield a well-formed envelope. press/math already recovers the known go-latex panics INTERNALLY (STATE.md), so a real construct no longer panics up to RenderJSON — it degrades to a success envelope. The recover guard remains as defense-in-depth and is proven via the seam."
  - "Envelope version eden-press.capi/v1 frozen at the capi layer, independent of Model.SchemaVersion (RESEARCH Open Question 2, resolved for versioning). Empty request envelopeVersion is tolerated (lenient client); a non-empty mismatch is rejected into an error envelope."
  - "Standardized on the LITERAL output name libpress.so on every OS (Go honors an explicit -o name verbatim, no .dylib rewrite on macOS) so 07-04/07-05 look up one stable filename."

patterns-established:
  - "Wire-contract test: parse the response into structs that redeclare the EXACT on-the-wire keys (capitalized HTML/CSS/Model/Meta/Comments, since press.Output has no json tags) rather than round-tripping into the source Go types — a key rename is caught, never hidden."
  - "check-no-chromedp gate grows OUTWARD (press → chase → profiles → bind) as new pure-Go surface is added; never narrow it back to silence an offending import."

requirements-completed: [DART-01]

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 12min
completed: 2026-07-21
---

# Objective 7 TRD 01: JSON-in/JSON-out C-ABI Core (DART-01) Summary

**A pure-Go `core.RenderJSON` marshalling seam over `press.Render` (versioned `eden-press.capi/v1` envelope, always well-formed, panic-safe) plus a cgo `//export PressRender`/`PressFree` front door with a documented FFI memory-ownership contract — the one JSON boundary every downstream Dart-binding front door stands on.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-07-21T15:51:38Z
- **Completed:** 2026-07-21T16:03:00Z
- **Tasks:** 2/2 complete
- **Files modified:** 7 (5 created, 2 modified)

## Accomplishments
- `bind/capi/core.RenderJSON([]byte) []byte`: the shared, pure-Go JSON boundary. Unmarshals the `{envelopeVersion, markdown, options}` request, maps the JSON-serializable option subset onto `press.Options` (Sanitize deliberately left nil ⇒ built-in always-on policy), calls `press.Render` **exactly once** under a recover guard, and marshals the full `press.Output` (HTML/CSS/Model/Meta/Comments) into a `{envelopeVersion, output, error}` response. Never returns nil/empty; folds malformed input, unknown version, and a render panic into a well-formed error envelope.
- Pure-Go / no-cgo / no-build-tags — **verified to compile under `GOOS=js GOARCH=wasm`**, so 07-02's wasm shim imports the identical function.
- `bind/capi` cgo shim (`package main`): `//export PressRender` (C.GoString in → RenderJSON → C.CString out, malloc'd C heap) and `//export PressFree` (C.free), with the memory-ownership contract documented verbatim at the export site.
- `scripts/build-capi-host.sh`: `go build -buildmode=c-shared` emits `libpress.so` + `libpress.h` whose header declares `extern char* PressRender(char* cmd);` and `extern void PressFree(char* p);` — the same host artifact 07-04 reuses.
- `scripts/check-no-chromedp.sh` extended to scan `./bind/...` and **GREEN** (0 chromedp): the Dart binding surface holds the pure-Go boundary one level further out than press/chase/profiles.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: core.RenderJSON pure-Go JSON boundary | `go test ./bind/capi/core/ -v && go build ./bind/capi/core/ && go vet ./bind/capi/core/ && gofmt -l bind/capi/core/` | 0 | PASS |
| 1: (extra) pure-Go compiles under js/wasm | `GOOS=js GOARCH=wasm go build ./bind/capi/core/` | 0 | PASS |
| 2: cgo shim + host c-shared + no-chromedp/bind | `bash scripts/build-capi-host.sh && bash scripts/check-no-chromedp.sh && go build ./... && go vet ./... && go test ./...` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: core.RenderJSON — pure-Go JSON boundary** — `a384f85` (feat)
2. **Task 2: cgo PressRender/PressFree + host c-shared + no-chromedp gate extension** — `a15e9e2` (feat)

_Note: Task 1 is `tdd="true"` — RED (compile failure against undefined `RenderJSON`/`EnvelopeVersion`/`renderFn`) confirmed before GREEN; see TDD Evidence. Task 2 is `type="auto"`._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./... && CGO_ENABLED=1 go build -buildmode=c-shared -o /dev/null ./bind/capi` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (incl. `./bind/...` `./press/...`) | 0 | PASS |
| lint | `gofmt -l bind/ scripts/` (empty) + `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| license | `addlicense -l mit -s -c "AO Cyber Systems" ... -check .` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./bind/capi/core/` | 1 (compile failure: undefined RenderJSON/EnvelopeVersion/renderFn) | FAIL (correct) |
| GREEN (Task 1) | `go test ./bind/capi/core/ -v` | 0 (all 6 test-list cases pass) | PASS (correct) |
| REFACTOR (Task 1) | n/a | — | No refactor needed; implementation clean at GREEN |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (a gofmt reformat of `capi.go`'s comment-list indentation + inline-comment alignment, applied before the Task 2 commit)
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 07-01-TRD.md frontmatter):
  1. `core.RenderJSON` pure-Go, maps options onto frozen `press.Options` (Sanitize nil), calls `press.Render` once, marshals full `press.Output` losslessly — ✓ (TestRenderJSON_FullShapeLossless, wire-key pinned)
  2. malformed/unknown-version/panic → well-formed error envelope, never nil/empty, no panic escape — ✓ (TestRenderJSON_ErrorEnvelopes, TestRenderJSON_RecoversRenderPanic)
  3. `//export PressRender`/`PressFree` C-string glue over `core.RenderJSON`; ownership contract at export site — ✓ (bind/capi/capi.go)
  4. `go build -buildmode=c-shared ./bind/capi` emits libpress.so + libpress.h declaring the exports — ✓ (scripts/build-capi-host.sh, header grep verified)
  5. `check-no-chromedp.sh` scans `./bind/...` and stays green — ✓
- **Gate failures:** None remaining (the single gofmt nit was fixed before commit).

## Files Created/Modified
- `bind/capi/core/render.go` — `RenderJSON` + `EnvelopeVersion` const + `renderFn` seam + request/response envelope types + `renderOnce` recover guard + `errorEnvelope`/`marshal` (with `marshalFailFallback`).
- `bind/capi/core/render_test.go` — 6 test-list cases (happy round-trip, full-shape losslessness with pinned wire keys, option mapping, sanitize-always-on, error envelopes, panic recovery via renderFn seam + real heavy-math construct).
- `bind/capi/doc.go` — `package main` overview: the cgo/wasm front-door split rationale.
- `bind/capi/capi.go` — cgo `//export PressRender`/`PressFree` + empty `main()` + verbatim FFI memory-ownership contract.
- `scripts/build-capi-host.sh` — host-arch c-shared smoke build; asserts libpress.so/.h and header exports.
- `scripts/check-no-chromedp.sh` — MODIFIED: added `"./bind/..."` to the `TREES` array (one line).
- `.gitignore` — MODIFIED: ignore `/bind/capi/build/` (c-shared output).

## Decisions Made
- **doc.go is `package main`, not `package capi`** — see key-decisions. A directory cannot hold two package names; c-shared mandates `package main`.
- **Panic-recovery case realized via a `renderFn` seam** — press/math already recovers the go-latex panics internally, so the recover guard is defense-in-depth, proven deterministically via seam-injection (mirrors press.go's `parseWithEngine` seam). Also asserted a real `\begin{aligned}` construct yields a well-formed envelope.
- **Literal `libpress.so` filename on all OSes** — stable lookup for 07-04/07-05.
- **`envelopeVersion` empty tolerated, non-empty mismatch rejected** — lenient first-cut client, strict on explicit version drift.

## Deviations from Plan

### Auto-fixed / adjusted

**1. [Rule 3 - Blocking] doc.go package name corrected to `package main`**
- **Found during:** Task 1 (creating doc.go alongside the Task 2 `package main` capi.go)
- **Issue:** The TRD parenthetical said doc.go is "(package `capi`)", but a Go directory holds one package name and `capi.go` must be `package main` for c-shared — the two would not compile together.
- **Fix:** doc.go is `package main` (package doc only); capi.go supplies the exports + `func main(){}`.
- **Verification:** `go build ./...` + `go build -buildmode=c-shared` both pass.
- **Committed in:** `a384f85` (doc.go) / `a15e9e2` (capi.go)

**2. [Rule 1 - Bug/format] gofmt reformat of capi.go**
- **Found during:** Task 2 lint (`gofmt -l`)
- **Issue:** Comment-list indentation and inline-comment alignment were not gofmt-canonical.
- **Fix:** `gofmt -w bind/capi/capi.go`.
- **Verification:** `gofmt -l` empty afterward.
- **Committed in:** `a15e9e2` (part of Task 2 commit)

**3. [Rule 3 - Hygiene] .gitignore entry for c-shared build output**
- **Found during:** Task 2 (build-capi-host.sh emits libpress.so/.h under bind/capi/build/)
- **Issue:** Generated artifacts (and an unlicensed generated .h) must not be committed nor trip the addlicense gate.
- **Fix:** Added `/bind/capi/build/` to `.gitignore`; build artifacts removed from the working tree after the smoke.
- **Verification:** `addlicense ... -check .` passes; `git status` shows no build output.
- **Committed in:** `a15e9e2` (part of Task 2 commit)

---

**Total deviations:** 3 (1 blocking package-name correction, 1 gofmt fix, 1 gitignore hygiene).
**Impact on plan:** All necessary for correctness/buildability; no scope creep. The renderFn seam + case-6 realization are documented as decisions, not scope changes (production behavior is byte-for-byte `press.Render`).

## Issues Encountered
- The known go-latex-panicking heavy-math construct (`\begin{aligned}`) no longer panics up through `press.Render` because press/math recovers it internally and degrades — so the panic-recovery test was made deterministic via the `renderFn` seam (see Decisions). No functional issue; the recover guard is verified.

## User Setup Required
None — no external service configuration required. (Local c-shared build needs `CGO_ENABLED=1` + a C compiler on PATH; clang is present on this macOS host.)

## Next Objective Readiness
- **07-02 (wasm shim):** unblocked — `core.RenderJSON` is pure-Go and compiles under `GOOS=js GOARCH=wasm` (verified); the syscall/js front door imports the same seam.
- **07-04 (conformance harness) / 07-05 (Dart client):** the `eden-press.capi/v1` envelope schema and the `libpress.so`/`libpress.h` host artifact (via `scripts/build-capi-host.sh`) are frozen and ready.
- No blockers.

---
*Objective: 07-dart-binding*
*Completed: 2026-07-21*

## Self-Check: PASSED

- Files verified present: `bind/capi/core/render.go`, `bind/capi/core/render_test.go`, `bind/capi/doc.go`, `bind/capi/capi.go`, `scripts/build-capi-host.sh`, `scripts/check-no-chromedp.sh` (modified), `.gitignore` (modified), `07-01-SUMMARY.md` — all FOUND.
- Commits verified present: `a384f85` (Task 1, feat), `a15e9e2` (Task 2, feat) — both FOUND in git log.
- Gates re-confirmed green: `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l` (empty), `bash scripts/check-no-chromedp.sh` (0), `bash scripts/build-capi-host.sh`, `GOOS=js GOARCH=wasm go build ./bind/capi/core/`, `addlicense -check`.
