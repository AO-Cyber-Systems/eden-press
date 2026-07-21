---
objective: 04-cli
trd: "08"
subsystem: cli
tags: [preview, browser-open, integration-test, ci-gate, import-boundary, cobra]

# Dependency graph
requires:
  - objective: 04-cli
    provides: "04-03's render pipeline (resolveInput -> buildOptions -> press.Render -> assembleHTML), reused verbatim by runPreview and by the integration test's convert legs"
  - objective: 04-cli
    provides: "04-07's serveMux/prepServeCmd (testable via httptest.NewServer), reused by the integration test's serve leg"
provides:
  - "runPreview: fills the 04-02 stub -- converts input to a standalone HTML doc, writes a temp eden-press-*.html, and opens it in the user's default browser via an injectable openURL seam (github.com/pkg/browser, no chromedp)"
  - "scripts/check-cli-imports.sh: the mechanical CI gate proving cmd/eden-press's OWN source imports ONLY press/ from the engine (never chase/, profiles/, or chromedp directly) -- the CLI analogue of API-02's check-no-chromedp.sh, wired into the same Makefile + CI job"
  - "cmd/eden-press/integration_test.go: one pass proving convert (file/stdin/-o) + theme/config precedence + --theme-set + the serve mux (convert-on-request + traversal reject) all compose through the same buildOptions/press.Render/assembleHTML chain"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "openURL package var (defaulting to browser.OpenURL) is the ONE seam preview needs to be testable without ever spawning a real browser -- the same swap-and-restore-via-t.Cleanup idiom already used elsewhere in this package"
    - "check-cli-imports.sh checks .Imports (a package's OWN direct imports) rather than `go list -deps` (transitive) -- the transitive closure legitimately contains chase/profiles VIA press, so only the direct-import check can assert the CLI-is-a-consumer-only boundary without false-failing"
    - "The integration test reuses every helper (resetCfg, newTestConvertCmd, writeTempConfig/chdir, writeTempCSS/brandCSS, prepServeCmd) from the OTHER _test.go files in the same package rather than re-implementing them -- proving composition, not duplicating per-mode unit coverage"

key-files:
  created:
    - cmd/eden-press/preview_test.go
    - cmd/eden-press/integration_test.go
    - scripts/check-cli-imports.sh
  modified:
    - cmd/eden-press/preview.go
    - Makefile
    - .github/workflows/ci.yml

key-decisions:
  - "preview's v1 stays scoped to the file:// convert-and-open path (no --url flag to preview a running `serve` instance) -- the TRD's action spec called this optional/only-if-trivial, and the file:// path alone satisfies CLI-04's must_haves without adding scope"
  - "The integration test's traversal leg targets the literal path from the TRD's test list (/../../etc/passwd) rather than a synthetic sentinel file (unlike serve_test.go's own dedicated traversal test, which uses a controlled sentinel for a stronger leak-proof assertion) -- both the safe-rejection status code (400/403/404) and a defensive 'root:' substring check are asserted; serve_test.go's sentinel-based test remains the authoritative containment proof"
  - "check-cli-imports.sh's grep pattern matches '(chase|profiles)' and 'chromedp' against cmd/eden-press's DIRECT .Imports only -- deliberately NOT press subpackage paths (press/themes, press/math), which are within the allowed boundary and must never trip this gate"

requirements-completed: [CLI-04]

# Verification evidence
verification:
  gates_defined: 7
  gates_passed: 7
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 25min
completed: 2026-07-21
---

# Objective 4 TRD 08: Preview (pkg/browser) + Integration Test + CLI-Imports CI Gate -- CLI Capstone (CLI-04) Summary

**`eden-press preview <in.md>` opens the rendered deck in the user's default browser via `github.com/pkg/browser` behind a testable `openURL` seam; `scripts/check-cli-imports.sh` mechanically proves `cmd/eden-press` imports only `press/` (never `chase/`/`profiles/`/chromedp), wired into CI beside `check-no-chromedp.sh`; and `integration_test.go` proves convert/theme/config/serve compose in one pass -- closing Objective 4 at 8/8 TRDs.**

## Performance

- **Duration:** ~25 min (research/read-through + implementation + verification)
- **Tasks:** 2/2 complete
- **Files:** 3 created, 3 modified

## Accomplishments

- `runPreview` fills the 04-02 stub: rejects stdin (`"-"`) up front like watch (04-06), then reuses 04-03's `resolveInput -> buildOptions -> press.Render -> assembleHTML` chain verbatim, writes the result to an `os.CreateTemp("", "eden-press-*.html")` file, and opens it via the `openURL` package var (default `browser.OpenURL`) -- no chromedp, no hand-rolled `open`/`xdg-open`/`rundll32` shelling.
- `openURL` is the ONLY seam needed to test preview without spawning a real browser: `preview_test.go` swaps it to a capturing closure (restored via `t.Cleanup`), asserting both the `file://` target and the temp file's actual standalone-doc contents.
- `scripts/check-cli-imports.sh` (mirrors `check-no-chromedp.sh`'s MIT header + `set -euo pipefail` + PASS/FAIL echo style) asserts `go list -f '{{range .Imports}}...'` on `./cmd/eden-press/...` -- the package's OWN direct imports, not a transitive `-deps` scan (which would legitimately show `chase`/`profiles` via `press`, and must NOT trip this gate) -- contains no `chase/`, `profiles/`, or `chromedp`. Wired into the `Makefile` (`check-cli-imports` target, `.PHONY`) and `.github/workflows/ci.yml` (a step immediately after "Check no chromedp (API-02)").
- `cmd/eden-press/integration_test.go` composes the objective's modes in one file: `TestIntegrationConvertModes` (file->stdout, stdin->stdout, file->`-o`, all zero-JS), `TestIntegrationThemeAndConfigPrecedence` (`.marprc.yaml` theme applies; `--theme` overrides it; `--theme-set` custom theme's scoped CSS renders), and `TestIntegrationServeConvertAndTraversal` (the 04-07 `serveMux` converts a `.md` on request and rejects `/../../etc/passwd`, run against `httptest.NewServer` -- no real port bind, 8080/8091 untouched).
- Zero `go.mod`/`go.sum` changes: `github.com/pkg/browser` was already provisioned (indirect) by 04-02; this TRD only promotes its use into `preview.go`.
- Both mechanical gates stay green: `check-no-chromedp.sh` (unaffected -- this TRD touches only `cmd/` + `scripts/` + CI) and the new `check-cli-imports.sh`.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Fill runPreview via pkg/browser (injectable openURL seam) | `go build ./... && go test ./cmd/eden-press/ -run TestPreview -v && go vet ./... && gofmt -l cmd/eden-press/preview.go cmd/eden-press/preview_test.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: Import-boundary CI gate + full-stack integration test (capstone) | `go build ./... && go test ./cmd/eden-press/... && go vet ./... && bash scripts/check-cli-imports.sh && bash scripts/check-no-chromedp.sh && gofmt -l cmd/eden-press/integration_test.go && addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/preview.go cmd/eden-press/preview_test.go cmd/eden-press/integration_test.go && make check-cli-imports` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Fill runPreview via pkg/browser** -- `d6097c6` (feat)
2. **Task 2: CLI-imports boundary gate + full-stack integration test** -- `062f625` (feat)

_Note: Task 1 is `tdd="true"`; RED was confirmed as a genuine compile failure (`undefined: openURL` across `preview_test.go`) against the pre-existing 04-02 stub before `preview.go` was filled in -- see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole-repo) | `go test ./...` | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| cli-imports (new) | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| gofmt | `gofmt -l cmd/eden-press/preview.go cmd/eden-press/preview_test.go cmd/eden-press/integration_test.go` | 0 (no output) | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/preview.go cmd/eden-press/preview_test.go cmd/eden-press/integration_test.go` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED | `go test ./cmd/eden-press/ -run TestPreview -v` against the 04-02 `runPreview` stub + new `preview_test.go` | 1 (build failed: `undefined: openURL`) | FAIL (correct) |
| GREEN | `go test ./cmd/eden-press/ -run TestPreview -v` (2 tests) | 0 | PASS (correct) |
| GREEN (whole-repo) | `go test ./...` | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 04-08-TRD.md frontmatter: preview opens the rendered output via pkg/browser behind an injectable seam; the integration test exercises convert/theme/config/serve composition in one pass; `check-cli-imports.sh` mechanically enforces the direct-import boundary and is wired into Makefile + CI; `check-no-chromedp.sh` stays green and the whole objective's standing gates pass)
- **Gate failures:** None

## Files Created/Modified
- `cmd/eden-press/preview.go` -- filled the 04-02 stub: `openURL` package var + `runPreview` (reject stdin, reuse 04-03's render chain, temp-file write, `openURL` call)
- `cmd/eden-press/preview_test.go` -- 2 test functions: `openURL`-capture (file:// path + doc contents) and stdin-rejection (no conversion, no `openURL` call)
- `cmd/eden-press/integration_test.go` -- 3 test functions (with subtests): convert modes, theme+config precedence, serve convert-and-traversal -- reuses every existing test helper from `config_test.go`/`themeset_test.go`/`serve_test.go` verbatim
- `scripts/check-cli-imports.sh` -- new CI gate: direct-import scan of `./cmd/eden-press/...` for `chase/`, `profiles/`, `chromedp`
- `Makefile` -- `check-cli-imports` target added to `.PHONY` and defined beside `check-no-chromedp`
- `.github/workflows/ci.yml` -- "Check CLI imports (CLI-04)" step added immediately after "Check no chromedp (API-02)"

## Decisions Made
- Preview's v1 scope stays the file:// convert-and-open path only -- the TRD marked previewing a running `serve` URL as optional/trivial-only, and it wasn't needed to satisfy CLI-04's must_haves.
- The integration test's traversal assertion targets the TRD's literal `/../../etc/passwd` test-list wording (status-code + defensive `root:`-substring check) rather than duplicating `serve_test.go`'s stronger sentinel-file proof, which remains the authoritative containment test for that behavior.
- `check-cli-imports.sh`'s grep is scoped to `(chase|profiles)|chromedp` against `.Imports` only, confirmed NOT to false-fail on `press/themes`/`press/math` subpackage imports already present in the CLI's dependency surface.

## Deviations from Plan

None - TRD executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None -- no external service configuration required. Zero `go.mod`/`go.sum` changes (`github.com/pkg/browser` was already provisioned indirectly by 04-02); this TRD imports only `press/`, stdlib, `github.com/spf13/cobra`, and `github.com/pkg/browser`.

## Next Objective Readiness
Objective 4 (CLI) is now **complete at 8/8 TRDs**: `eden-press` ships convert (CLI-01), watch (CLI-02), serve (CLI-03), preview (CLI-04), `--theme`/`--theme-set` (CLI-05), and koanf config + stdin (CLI-06) -- all composing through one shared render pipeline, with two mechanical CI gates (`check-no-chromedp.sh`, `check-cli-imports.sh`) proving the engine boundary holds at both the `press/` and `cmd/eden-press` layers.

## Self-Check: PASSED

All claimed files confirmed present on disk; all claimed commit hashes confirmed present in `git log --oneline --all`.

- FOUND: cmd/eden-press/preview.go
- FOUND: cmd/eden-press/preview_test.go
- FOUND: cmd/eden-press/integration_test.go
- FOUND: scripts/check-cli-imports.sh
- FOUND: .planning/objectives/04-cli/04-08-SUMMARY.md
- FOUND commit: d6097c6 (Task 1, feat)
- FOUND commit: 062f625 (Task 2, feat)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
