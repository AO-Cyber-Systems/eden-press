---
objective: 04-cli
job: "01"
subsystem: press
tags: [press, theme, chase-theme, additive-api, cli-enabler]

# Dependency graph
requires:
  - objective: 03-press-batteries
    provides: "press.Options/Output frozen API-03 surface (press/options.go), press.Render one-parse-two-sinks composition + packThemeCSS (press/press.go), press/themes.ThemeSet + BrowserFitJS (press/themes/themes.go), chase/theme.Load + ThemeSet.Add intake path (chase/theme/theme.go, pack.go)"
provides:
  - "press.Options.ThemeCSS []string — additive field carrying raw custom-theme CSS text; caller (CLI) reads files, press/ never touches the filesystem"
  - "press.Render registers each ThemeCSS entry via chase/theme.Load + ThemeSet.Add (the same intake the 3 embedded themes use), making a custom theme selectable by name through the existing opts.Theme / front-matter theme: resolution chain"
  - "press.BrowserFitJS() — package-root re-export of press/themes.BrowserFitJS(), letting a press-only consumer splice the auto-fit helper without importing press/themes"
affects: [04-02-cobra-skeleton, 04-03-htmldoc-convert, 04-05-theme-set-flag]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive-only extension of an Objective-3-frozen API surface: new field appended after the last existing field, zero value proven a no-op via a golden regression test (HTML pinned verbatim; CSS pinned by length + SHA-256 digest since the packed default theme is ~47KB, too large to usefully inline as literal text)"
    - "Custom-theme registration reuses the exact chase/theme.Load + ThemeSet.Add call the 3 embedded themes already go through (press/themes.ThemeSet), so a caller-supplied theme is just another registered name — no parallel lookup/resolution path was introduced"

key-files:
  created:
    - press/browserjs.go
    - press/themecss_test.go
  modified:
    - press/options.go
    - press/press.go

key-decisions:
  - "Golden additive-parity test pins Output.HTML verbatim (small, ~180 bytes) but pins Output.CSS by length + SHA-256 digest rather than embedding the ~47KB packed default-theme CSS as inline literal text — both are exact byte-level identity checks over the FULL output, chosen to keep the test file readable while still proving zero-value behavior is unchanged."
  - "TestThemeCSSAdditive and TestBrowserFitJSReexport (Task 1) were authored alongside their implementation rather than strictly before it; RED status was reconstructed and verified retroactively (see TDD Evidence) by temporarily removing/reverting the just-added code and re-running the tests, confirming each fails for the expected reason before restoring the GREEN implementation. No production code shipped without a verified failing-then-passing test."

patterns-established:
  - "For a large/derived golden CSS blob, prefer length+hash pinning over inline text embedding to keep additive-regression tests both exact and reviewable."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 5
  gates_passed: 5
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 15min
completed: 2026-07-21
---

# Objective 4 TRD 01: press.Options.ThemeCSS + BrowserFitJS Re-export Summary

**Additive `press.Options.ThemeCSS []string` hook lets a caller register raw custom-theme CSS text through the exact `chase/theme.Load` + `ThemeSet.Add` path the 3 embedded themes use, plus a `press.BrowserFitJS()` re-export — both closing the single blocking cross-package gap CLI-05 (`--theme-set`) needs, with zero change to `press.Options{}`'s zero-value behavior.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-21T15:55:19Z
- **Tasks:** 2/2 complete
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments
- `press.Options.ThemeCSS []string` — a purely additive field (placed after `Sanitize`, the last existing field) documented as raw theme-CSS TEXT; press/ never reads the filesystem, keeping `Render` a pure function of `(md, opts)`.
- `packThemeCSS` (press/press.go) registers each `opts.ThemeCSS` entry via `theme.Load(css, p.UnitElement(), p.Sizes().ByName)` + `ts.Add(th)` — inserted immediately after `themes.ThemeSet(...)` and before `resolveThemeName(...)`, so a custom theme becomes just another name the existing `opts.Theme` → front-matter `theme:` → `"default"` chain can resolve. A malformed block (missing leading `/* @theme name */`) surfaces as a wrapped `press: Render: load custom theme CSS: %w` error, never a panic.
- `press.BrowserFitJS()` (new `press/browserjs.go`) re-exports `press/themes.BrowserFitJS()` at the package root, so a consumer importing only `press/` (the CLI) can splice the Marp Core auto-fit script without reaching into `press/themes`.
- Golden additive-parity proof: `press.Render("# Hi\n", press.Options{})` is verified byte-identical to a pre-change capture — `Output.HTML` pinned verbatim, `Output.CSS` (47,104 bytes, the full packed default theme) pinned by length + SHA-256 digest.
- End-to-end custom-theme proof: a `brandx` theme supplied via `ThemeCSS` packs its scoped `#d4a853` rule into `Output.CSS` both when selected via front-matter `theme: brandx` and via `Options.Theme: "brandx"` override with no front matter present.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Options.ThemeCSS field + BrowserFitJS re-export | `go build ./... && go test ./press/ -run 'TestThemeCSSAdditive\|TestBrowserFitJSReexport' -v && go vet ./... && gofmt -l press/options.go press/browserjs.go press/themecss_test.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: Thread ThemeCSS through packThemeCSS | `go build ./... && go test ./press/... && go vet ./... && gofmt -l press/press.go && bash scripts/check-no-chromedp.sh && addlicense -l mit -s -c "AO Cyber Systems" -check press/browserjs.go press/themecss_test.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Add press.Options.ThemeCSS field + press.BrowserFitJS re-export** - `0307887` (feat)
2. **Task 2: Thread ThemeCSS through packThemeCSS via theme.Load + ThemeSet.Add** - `5f5de94` (feat)

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (whole repo, incl. Obj-3 capstone) | 0 | PASS |
| gofmt | `gofmt -l press/options.go press/press.go press/browserjs.go press/themecss_test.go` | 0 (no output) | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check press/browserjs.go press/themecss_test.go` | 0 | PASS |

## TDD Evidence

Both tasks are `tdd="true"`. Tests and implementation were authored together rather than strictly test-first; RED status was reconstructed retroactively and verified before restoring GREEN (no production code shipped without a verified fail→pass cycle):

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./press/ -run TestBrowserFitJSReexport -v` with `press/browserjs.go` temporarily removed | 1 (build failed: `undefined: BrowserFitJS`) | FAIL (correct) |
| GREEN (Task 1) | `go test ./press/ -run 'TestThemeCSSAdditive\|TestBrowserFitJSReexport' -v` with `press/browserjs.go` restored | 0 | PASS (correct) |
| RED (Task 2) | `go test ./press/ -run 'TestCustomThemeByFrontMatter\|TestCustomThemeByOptsOverride\|TestMalformedThemeCSSErrors' -v` with `press/press.go` temporarily reverted to the pre-loop (commit `0307887`) version | 1 (all 3 tests FAIL: `unknown theme "brandx"` x2, and the malformed-block test found no error) | FAIL (correct) |
| GREEN (Task 2) | Same command with `press/press.go` restored to the committed `5f5de94` version | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 04-01-TRD.md frontmatter — additive zero-value parity, theme.Load/ThemeSet.Add threading, BrowserFitJS re-export, purely-additive API-03 surface)
- **Gate failures:** None

## Files Created/Modified
- `press/options.go` - added `ThemeCSS []string` field (additive only; `git diff` confirms no existing field touched)
- `press/press.go` - `packThemeCSS` gained the custom-theme registration loop (`theme.Load` + `ts.Add`) between `themes.ThemeSet(...)` and `resolveThemeName(...)`
- `press/browserjs.go` (new) - `press.BrowserFitJS()` re-export of `press/themes.BrowserFitJS()`
- `press/themecss_test.go` (new) - golden additive-parity test, BrowserFitJS re-export identity test, custom-theme-by-front-matter test, custom-theme-by-opts.Theme-override test, malformed-block error test

## Decisions Made
- `Output.CSS` golden pinned by length (47,104 bytes) + SHA-256 digest rather than an inline ~47KB text literal — an equally exact byte-identity check without bloating the test file (documented in frontmatter `key-decisions`).
- No JSON struct tags added to `ThemeCSS` (per TRD anti-pattern guidance — `Options` has none today; Objective 7 owns serialization decisions).

## Deviations from Plan

None - TRD executed exactly as written. The custom-theme registration loop, error-wrapping message, and re-export shape all match the TRD's `codebase_examples` verbatim.

## Issues Encountered
None. The coordinator flagged mid-execution that `ThemeCSS` appeared inert after Task 1's commit alone — this was expected: Task 1 (committed first) intentionally scoped to the additive field + re-export only, with the consuming loop landing in Task 2 (already implemented locally at that point and committed immediately after as `5f5de94`).

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- `press.Options.ThemeCSS` is the field 04-05 (`--theme-set`) populates from files it reads itself (press/ import boundary preserved).
- `press.BrowserFitJS()` is what 04-03's `htmldoc` (`--auto-fit-script`) splices after `Output.HTML`.
- Both press/ seams the CLI objective's theme + script features stand on are now landed; 04-02 (cobra skeleton) and downstream waves are unblocked.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: press/options.go
- FOUND: press/press.go
- FOUND: press/browserjs.go
- FOUND: press/themecss_test.go
- FOUND: .planning/objectives/04-cli/04-01-SUMMARY.md
- FOUND commit: 0307887 (Task 1)
- FOUND commit: 5f5de94 (Task 2)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
