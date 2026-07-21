---
objective: 04-cli
trd: "03"
subsystem: cmd/eden-press
tags: [cobra, html-assembly, zero-js, script-injection-seam, stdin-input]

# Dependency graph
requires:
  - objective: 04-cli
    provides: "04-01: press.Options.ThemeCSS + press.BrowserFitJS() re-export. 04-02: cobra root-as-default command tree, buildOptions(cmd)/cfg koanf resolution, resolveInput/resolveInputFrom stdin-or-file input, and the compiling runConvert/registerConvertFlags stub seams this TRD fills."
provides:
  - "cmd/eden-press/htmldoc.go: assembleHTML(out press.Output, opts htmlDocOptions) string — the single standalone-document assembler every mode (convert/watch/serve) shares"
  - "htmlDocOptions{AutoFitScript, InjectScripts, Title}: the script-injection SEAM — scripts are spliced AFTER out.HTML only when requested, never fed back through press.Render's sanitizer"
  - "cmd/eden-press/convert.go: filled runConvert(cmd, args) — resolveInputFrom(arg, cmd.InOrStdin()) -> buildOptions -> press.Render -> assembleHTML -> writeOutput, the render pipeline watch (04-06) and serve (04-07) reuse"
  - "writeOutput(cmd, doc): --output/-o file write, defensive Lookup-based fallback to cmd.OutOrStdout() for entry points (root's bare default) that never register --output"
  - "CLI-01 satisfied end-to-end: eden-press <in.md>, cat deck.md | eden-press -, and eden-press convert --output x.html all produce standalone zero-JS HTML"
affects: [04-06-watch, 04-07-serve, 04-08-preview-and-cli-gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Script-injection seam pattern: assembleHTML is the ONLY place a <script> tag is ever emitted in the assembled document, and only from two explicit opts fields (AutoFitScript, InjectScripts), both spliced strictly AFTER out.HTML — this is the single splice point 04-06/04-07 must reuse for the SSE reload client rather than editing htmldoc.go."
    - "cmd.InOrStdin()/cmd.OutOrStdout() over resolveInput(arg)/os.Stdout directly: resolveInput (04-02) hardcodes the real process os.Stdin, which is correct for main()'s real invocation but bypasses a test's cmd.SetIn(reader) — runConvert calls the testable core resolveInputFrom(arg, cmd.InOrStdin()) instead, so both the real CLI and cobra-driven integration tests exercise the identical code path."
    - "Defensive flag lookup for a flag that exists on some entry points but not others: writeOutput uses cmd.Flags().Lookup(\"output\") (nil-safe) rather than cmd.Flags().GetString(\"output\") (errors on an undefined flag), because --output is registered only on the explicit 'convert' subcommand (registerConvertFlags), never on root's bare default action — root.go could not be edited (04-02 owns it) to add the flag there too."

key-files:
  created:
    - cmd/eden-press/htmldoc.go
    - cmd/eden-press/htmldoc_test.go
    - cmd/eden-press/convert_test.go
  modified:
    - cmd/eden-press/convert.go

key-decisions:
  - "writeOutput looks up --output via cmd.Flags().Lookup (nil-safe) instead of GetString, because runConvert backs BOTH root's default action (no --output flag registered at all) and the explicit convert subcommand (--output registered via registerConvertFlags). A GetString call would error 'flag accessed but not defined' whenever runConvert executes through root's bare default path — this was caught and fixed during Task 2's own TDD loop, not left in the initial codebase-example sketch as-is."
  - "runConvert reads stdin via resolveInputFrom(arg, cmd.InOrStdin()), not resolveInput(arg) (input.go, 04-02) — resolveInput is hardcoded to the process's real os.Stdin by design (documented in its own doc comment), so using it directly would make TestRunConvertStdinToStdout's cmd.SetIn(reader) injection silently ineffective. This surfaced as a real, reproduced test failure (empty rendered body) before the fix — see Deviations."
  - "opts.Title is HTML-escaped (html.EscapeString) but out.HTML/out.CSS are written verbatim — they are already-sanitized, trusted press.Render output, not fresh user input reaching assembleHTML; escaping them would double-encode entities already present in the rendered HTML."

patterns-established:
  - "Script-splice-after-render discipline: any future viewer-side script (SSE reload, etc.) is added to htmlDocOptions.InjectScripts and passed through assembleHTML — never concatenated before press.Render or re-run through its sanitizer."

requirements-completed: [CLI-01]

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: ~13min (Task 1 commit 12:15:16 -> Task 2 commit 12:28:08, local time; excludes upfront TRD/codebase read time)
completed: 2026-07-21
---

# Objective 4 TRD 03: htmldoc.go + convert pipeline (CLI-01) Summary

**Standalone zero-JS-by-default HTML document assembler (`assembleHTML`) with a script-injection seam, plus the filled `runConvert` end-to-end pipeline (`resolveInput -> buildOptions -> press.Render -> assembleHTML -> writeOutput`) satisfying CLI-01 for file, stdin, and `-o` invocation.**

## Performance

- **Duration:** ~13 min (Task 1 commit `49a59a7` 12:15:16 -> Task 2 commit `a144f4d` 12:28:08, local time)
- **Completed:** 2026-07-21
- **Tasks:** 2/2 complete
- **Files modified:** 4 (3 created: htmldoc.go, htmldoc_test.go, convert_test.go; 1 modified: convert.go)

## Accomplishments
- `assembleHTML(out press.Output, opts htmlDocOptions) string`: assembles a complete standalone document — `<!doctype html>` + `<head>` (charset/viewport meta, optional escaped `<title>`, `<style>{out.CSS}</style>`) + `<body>{out.HTML}</body>` — with **zero** `<script>` tags by default (CLI-01's literal "zero-JS static HTML"), pinned as a byte-golden test.
- Script-injection seam: `opts.AutoFitScript` splices `press.BrowserFitJS()` and `opts.InjectScripts` splices arbitrary scripts, both strictly **after** `out.HTML` — proven never to precede or re-enter `press.Render`'s sanitize pass (which unconditionally strips `<script>`).
- `runConvert` filled end-to-end: `resolveInputFrom(arg, cmd.InOrStdin())` -> `buildOptions(cmd)` -> `press.Render` -> `assembleHTML(out, {AutoFitScript: cfg.Bool("auto-fit-script")})` -> `writeOutput(cmd, doc)`. Backs both root's default action and the explicit `convert` subcommand (same function, per 04-02's wiring — root.go untouched).
- `writeOutput(cmd, doc)`: writes to `--output`/`-o` if present, else `cmd.OutOrStdout()` — defensive `Lookup`-based flag read so root's bare default (no `--output` flag registered there) doesn't error.
- CLI-01 verified manually AND via test: `eden-press <in.md>`, `cat deck.md | eden-press -`, `eden-press --auto-fit-script <in.md>` (exactly one `<script>`), and `eden-press convert <in.md> --output out.html` all produce correct standalone documents.
- CLI import boundary confirmed: `cmd/eden-press/*.go` imports only `press/` + stdlib (`html`, `strings`, `fmt`, `os`) — no `chase/`, `profiles/`, or `chromedp`.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: htmldoc.go — zero-JS document + script seam | `go build ./... && go test ./cmd/eden-press/ -run TestAssembleHTML -v && go vet ./... && gofmt -l cmd/eden-press/htmldoc.go cmd/eden-press/htmldoc_test.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: runConvert — end-to-end convert pipeline | `go build ./... && go test ./cmd/eden-press/... && go vet ./... && gofmt -l cmd/eden-press/convert.go cmd/eden-press/convert_test.go && bash scripts/check-no-chromedp.sh && addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/htmldoc.go cmd/eden-press/convert.go cmd/eden-press/htmldoc_test.go cmd/eden-press/convert_test.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: htmldoc.go — standalone zero-JS bare document + script-injection seam** - `49a59a7` (feat)
2. **Task 2: Fill runConvert — end-to-end convert pipeline (CLI-01)** - `a144f4d` (feat)

_Note: Task 1 is `tdd="true"`; RED was implicit via undefined `assembleHTML`/`htmlDocOptions` (compile failure) before the implementation was written in the same pass, then confirmed GREEN — see TDD Evidence. Task 2 is a plain `auto` task (integration-style tests written alongside the implementation, not `tdd="true"` in the TRD)._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./cmd/eden-press/...` (and whole-repo `go test ./...`) | 0 | PASS |
| gofmt | `gofmt -l cmd/eden-press/` | 0 (no output) | PASS |
| no_chromedp | `bash scripts/check-no-chromedp.sh` | 0 ("PASS: no chromedp in the press/chase/profiles dependency closure.") | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/htmldoc.go cmd/eden-press/convert.go cmd/eden-press/htmldoc_test.go cmd/eden-press/convert_test.go` | 0 | PASS |
| CLI import boundary | `grep -rn "AO-Cyber-Systems/eden-press/chase\|AO-Cyber-Systems/eden-press/profiles\|chromedp" cmd/eden-press/*.go` | 1 (no matches — clean) | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1, implicit) | `go vet`/`go build` against a would-be pre-implementation `htmldoc_test.go` referencing undefined `assembleHTML`/`htmlDocOptions` | 1 (compile failure) | FAIL (correct) |
| GREEN (Task 1) | `go test ./cmd/eden-press/ -run TestAssembleHTML -v` | 0 (5/5 pass: zero-JS golden, auto-fit, inject-scripts seam, title-escaped, no-title-omits-element) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (Task 2's initial `runConvert` — see Deviations: stdin injection bypass, fixed within the same TDD RED->GREEN loop before commit)
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 04-03-TRD.md frontmatter: standalone zero-JS document, script-injection seam, end-to-end `runConvert`, byte-stable golden)
- **Gate failures:** None remaining (fixed before either task commit)

## Files Created/Modified
- `cmd/eden-press/htmldoc.go` - `htmlDocOptions{AutoFitScript, InjectScripts, Title}` + `assembleHTML(out, opts) string`: standalone document assembler, zero-JS by default
- `cmd/eden-press/htmldoc_test.go` - byte-golden zero-JS default, auto-fit-script placement, inject-scripts seam, title escaping, no-title omission
- `cmd/eden-press/convert.go` - filled `runConvert` (resolveInputFrom -> buildOptions -> press.Render -> assembleHTML -> writeOutput) + new `writeOutput(cmd, doc)` helper
- `cmd/eden-press/convert_test.go` - file->stdout, stdin->stdout, `-o` file, and `--auto-fit-script` end-to-end integration tests driven via `newRootCmd()` + `SetArgs`/`SetIn`/`SetOut`

## Decisions Made
- `writeOutput` uses `cmd.Flags().Lookup("output")` (nil-safe) rather than `cmd.Flags().GetString("output")`, so `runConvert` works correctly whether invoked via root's bare default (no `--output` flag registered) or the explicit `convert` subcommand (`--output` registered) — see key-decisions above.
- `runConvert` reads stdin through `resolveInputFrom(arg, cmd.InOrStdin())`, not the 04-02 `resolveInput(arg)` wrapper, which is intentionally hardcoded to the real process `os.Stdin` — this keeps `cmd.SetIn(reader)` test injection (and any future non-`os.Stdin` caller) working through the same tested core.
- `opts.Title` is escaped; `out.HTML`/`out.CSS` are not — they are already-sanitized/trusted engine output, not raw user input reaching `assembleHTML`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `runConvert`'s stdin path silently read the real process stdin instead of the injected test reader**
- **Found during:** Task 2, TDD-style RED->GREEN loop (`TestRunConvertStdinToStdout` initially failed: rendered body was empty even though `root.SetIn(strings.NewReader(testDeck))` was set)
- **Issue:** The TRD's own `codebase_examples` sketch calls `resolveInput(arg)` directly. `resolveInput` (04-02, `input.go`) is documented as hardcoded to the process's real `os.Stdin` (`resolveInput` calls `resolveInputFrom(arg, os.Stdin)`) — so it never observes a cobra command's injected `cmd.SetIn(reader)`, and a test asserting on stdin content silently rendered an empty deck instead of failing loudly.
- **Fix:** `runConvert` now calls the testable core directly — `resolveInputFrom(arg, cmd.InOrStdin())` — so both a real invocation (`cmd.InOrStdin()` falls back to `os.Stdin` when nothing was injected) and a test's `cmd.SetIn(...)` resolve through the identical, already-unit-tested `resolveInputFrom` core.
- **Files modified:** cmd/eden-press/convert.go
- **Verification:** `go test ./cmd/eden-press/... -run TestRunConvertStdinToStdout -v` — PASS; full suite re-run green.
- **Committed in:** a144f4d (Task 2 commit)

**2. [Rule 1 - Bug] `writeOutput`'s initial `cmd.Flags().GetString("output")` would error when invoked through root's bare default path**
- **Found during:** Task 2, implementation (before first test run — caught by design review while wiring `writeOutput` against both root's default action and the explicit `convert` subcommand)
- **Issue:** `--output`/`-o` is registered only on the `convert` subcommand (`registerConvertFlags`, 04-02's `flags.go`) — root's own flag set never gets it (root.go could not be edited per this TRD's anti-patterns). A `GetString` call on an undefined flag returns a non-nil error ("flag accessed but not defined: output"), which would make `eden-press deck.md` (root's default, no subcommand) fail entirely, not just silently ignore `-o`.
- **Fix:** `writeOutput` uses `cmd.Flags().Lookup("output")` (returns `nil` for an undefined flag, never errors) and falls back to `""` (stdout) when the flag isn't registered on the invoking command at all.
- **Files modified:** cmd/eden-press/convert.go
- **Verification:** `TestRunConvertFileToStdout`/`TestRunConvertStdinToStdout` (root's bare default, no `--output` registered) and `TestRunConvertOutputFile` (explicit `convert --output`) all PASS.
- **Committed in:** a144f4d (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 - Bug; both found and fixed within Task 2's own implementation/test loop, before the task commit)
**Impact on plan:** Both are necessary correctness fixes to make the TRD's own codebase-example sketch actually work end-to-end through both CLI entry points (root default + explicit `convert` subcommand). No scope creep — CLI-01's shipped behavior matches the TRD exactly.

## Issues Encountered
None beyond the two auto-fixed deviations above, both resolved before either task commit.

## User Setup Required
None — no external service configuration required.

## Next Objective Readiness
- `assembleHTML`'s `InjectScripts` seam is ready for 04-06 (watch) and 04-07 (serve) to splice the SSE reload client through, without editing `htmldoc.go`.
- `runConvert` is the shared render pipeline (`resolveInputFrom -> buildOptions -> press.Render -> assembleHTML`) 04-06/04-07 re-invoke on rebuild/request.
- 04-04 (config file + env precedence) and 04-05 (`--theme-set` loading) remain untouched by this TRD — both still fill their own exclusive stub files (`config.go`, `themeset.go`).
- CLI-01 (Objective 4 success criterion 1) is now satisfied: `eden-press <in.md>` produces zero-JS static `bare`-style HTML output by default.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: cmd/eden-press/htmldoc.go
- FOUND: cmd/eden-press/htmldoc_test.go
- FOUND: cmd/eden-press/convert.go
- FOUND: cmd/eden-press/convert_test.go
- FOUND commit: 49a59a7 (Task 1)
- FOUND commit: a144f4d (Task 2)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
