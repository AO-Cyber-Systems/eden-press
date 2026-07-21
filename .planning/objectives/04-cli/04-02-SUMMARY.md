---
objective: 04-cli
trd: "02"
subsystem: cmd/eden-press
tags: [cobra, koanf, cli-skeleton, posflag, stdin-input]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "press.Render(md, opts) public API; press.Options{Theme,Profile,InlineSVG,MathMode,NoHighlight,HighlightStyle,Sanitize}; the CI-enforced zero-chromedp boundary this CLI must stay inside"
provides:
  - "cmd/eden-press: root-as-default cobra command tree (root RunE=runConvert; convert/watch/serve/preview explicit subcommands), Pitfall-1 collision documented in root Long help"
  - "koanf-backed buildOptions(cmd) (press.Options, error): single flag->Options resolution point every mode calls, reading through package cfg"
  - "loadConfigSources(k, cmd) posflag-only baseline (three-arg instance form, Pitfall 5) — 04-04 prepends file+env providers ahead of posflag without editing options.go"
  - "themeCSS(cmd) stub seam ([]string, error) for 04-05"
  - "resolveInput/resolveInputFrom: stdin(-)/file input resolution with an OBSERVABLE inputSource (fileSource/stdinSource), no rejection policy baked in"
  - "Compiling stub seams: runConvert(04-03), runWatch(04-06), runServe(04-07), runPreview(04-08) — each returns a clear not-implemented error naming its owning TRD"
  - "go.mod provisioned ONCE with all four CLI dep groups (cobra, fsnotify, koanf+parsers+providers, pkg/browser) — Wave-2+ CLI TRDs only import, never touch go.mod"
affects: [04-03-convert, 04-04-config, 04-05-themeset, 04-06-watch, 04-07-serve, 04-08-preview-and-cli-gate]

# Tech tracking
tech-stack:
  added:
    - "github.com/spf13/cobra v1.10.2 (+ github.com/spf13/pflag, github.com/inconshreveable/mousetrap — cobra transitive deps)"
    - "github.com/fsnotify/fsnotify v1.10.1"
    - "github.com/knadh/koanf/v2 v2.3.5 + parsers/{yaml,json,toml} + providers/{file,posflag,env} (+ knadh/koanf/maps, go-viper/mapstructure/v2, mitchellh/{copystructure,reflectwalk}, pelletier/go-toml, go.yaml.in/yaml/v3 — koanf transitive deps)"
    - "github.com/pkg/browser (+ golang.org/x/sys — transitive)"
  patterns:
    - "Root-as-default cobra tree: root's own RunE performs the default CONVERT action for zero/one positional arg with no subcommand; convert/watch/serve/preview are ALSO explicitly registered via AddCommand for users who prefer the explicit form — both paths call the same runXxx stubs."
    - "posflag.Provider(cmd.Flags(), \".\", k) three-arg instance form (Pitfall 5): passing the LIVE koanf instance as the third argument lets posflag skip merging an unset flag's default over an already-set value, so a later file/env provider (04-04) can layer in ahead of posflag without this TRD's code changing."
    - "Split-for-testability I/O seam: resolveInput(arg) wraps resolveInputFrom(arg, stdin io.Reader) so tests inject a strings.Reader and never touch the real process stdin."
    - "Persistent vs. local flag scoping: --auto-fit-script and the press.Options-mapped flags are PERSISTENT (bound on root, inherited by every subcommand) because convert/watch/serve all read the same cfg keys; --output/-o, --port, --host are LOCAL to their owning subcommand."

key-files:
  created:
    - cmd/eden-press/main.go
    - cmd/eden-press/root.go
    - cmd/eden-press/flags.go
    - cmd/eden-press/options.go
    - cmd/eden-press/input.go
    - cmd/eden-press/convert.go
    - cmd/eden-press/watch.go
    - cmd/eden-press/serve.go
    - cmd/eden-press/preview.go
    - cmd/eden-press/config.go
    - cmd/eden-press/themeset.go
    - cmd/eden-press/flags_test.go
    - cmd/eden-press/input_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "press.Options.ThemeCSS does not exist in this worktree (added by sibling Wave-1 TRD 04-01, executed in a separate parallel git worktree, not yet merged here). buildOptions calls themeCSS(cmd) — satisfying the single-resolution-point must-have and propagating its error — but does NOT assign the ([]string) result to a nonexistent Options field. An inline doc comment on buildOptions states the exact one-line follow-up (`ThemeCSS: tcss`) needed once 04-01 merges. This is the expected parallel-worktree pattern, independently confirmed by sibling TRD 04-03's own error_recovery text (\"04-01 is Wave 1 and MUST be merged before this Wave-2 TRD runs\")."
  - "options.go and flag-registration wiring were created MINIMALLY in Task 1 (package cfg + applyConfig only) rather than deferred entirely to Task 2, because Go compiles a package as a whole — root.go's PersistentPreRunE needs applyConfig to exist to compile at all. Task 2 then extended options.go with buildOptions and added the registerXxxFlags(cmd) calls into root.go/convert.go/watch.go/serve.go (all within THIS TRD's own exclusive file set — no cross-TRD file ownership was touched)."
  - "cobra's PersistentFlags() are only merged into cmd.Flags() by ParseFlags()/Execute() (mergePersistentFlags()) — a standalone unexecuted *cobra.Command does not expose persistent flags via cmd.Flags().Set(...). flags_test.go exercises this correctly via cmd.ParseFlags([]string{...}) / cmd.ParseFlags(nil), which is also what genuinely proves the Pitfall-5 posflag-instance guard (rather than passing vacuously)."

patterns-established:
  - "Compiling-stub-seam discipline: every downstream Wave-2+ TRD gets an exclusive file (runConvert/runWatch/runServe/runPreview/loadConfigSources-body/themeCSS-body) that already compiles and returns a clear, TRD-numbered not-implemented error — the whole parallel-wave structure depends on these seams existing untouched until their owning TRD lands."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 5
  gates_passed: 5
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 4min (task-commit span; excludes upfront go.mod provisioning + design/spec-reading time)
completed: 2026-07-21
---

# Objective 4 TRD 02: CLI Skeleton (cobra + koanf) Summary

**Root-as-default cobra command tree (convert/watch/serve/preview) with a koanf-backed buildOptions flag->press.Options mapping (posflag Pitfall-5 instance guard), stdin/file input resolution, and compiling stub seams for every downstream Wave-2+ CLI TRD — go.mod provisioned once, additively, for the whole parallel workstream.**

## Performance

- **Duration:** ~4 min (Task 1 commit `14046ac` 11:59:38 -> Task 3 commit `b9f5480` 12:03:24, local time)
- **Completed:** 2026-07-21
- **Tasks:** 3/3 complete
- **Files modified:** 15 (13 created, 2 modified: go.mod, go.sum)

## Accomplishments
- `cmd/eden-press` binary package: root command whose own `RunE` performs a default CONVERT for zero/one positional arg with no subcommand (`eden-press deck.md` just works), plus explicitly-registered `convert`/`watch`/`serve`/`preview` subcommands — both paths call the same `runXxx` stub functions. The positional-arg-vs-subcommand-name collision (research Pitfall 1) is documented in root's `Long` help text (`eden-press convert -- watch.md` escape hatch).
- Full persistent flag surface (`--theme`, `--theme-set`, `--profile`, `--math`, `--no-highlight`, `--highlight-style`, `--inline-svg`, `--config`, `--auto-fit-script`) registered once on root and inherited by every subcommand, plus per-mode local flags (`--output/-o` on convert+watch, `--port`[default 8321]/`--host` on serve).
- `buildOptions(cmd) (press.Options, error)`: the single flag->Options resolution point, reading resolved values through the package `koanf` instance `cfg` (never re-implementing `press.Render`'s own `""`->default fallbacks) and calling the `themeCSS` stub.
- `loadConfigSources(k, cmd)`: posflag-only baseline wired with the load-bearing three-arg `posflag.Provider(cmd.Flags(), ".", k)` instance form (Pitfall 5) — proven by `TestPosflagInstanceGuard` that an unset flag does not stomp a value pre-seeded in `cfg` (the exact mechanism 04-04's file/env providers will rely on).
- `resolveInput`/`resolveInputFrom`: `-` reads all of stdin, any other value reads that file, and the source (`stdinSource`/`fileSource`) is exposed as an observable result rather than resolveInput itself enforcing any rejection policy — that's watch/serve's (04-06/04-07) job downstream (Pitfall 8).
- Compiling stub seams for every parallel Wave-2+ TRD: `runConvert` (04-03), `runWatch` (04-06), `runServe` (04-07), `runPreview` (04-08), `loadConfigSources` posflag-only body (04-04 prepends file+env), `themeCSS` (04-05) — each returns a clear, TRD-numbered not-implemented error.
- go.mod/go.sum provisioned ONCE, purely additively, via `go get` only (never `go mod tidy`): `github.com/spf13/cobra@v1.10.2`, `github.com/fsnotify/fsnotify@v1.10.1`, `github.com/knadh/koanf/v2@v2.3.5` + `parsers/{yaml,json,toml}` + `providers/{file,posflag,env}`, `github.com/pkg/browser` — confirmed via `git diff 3100281 HEAD -- go.mod` that every changed line is a pure addition (no existing `require` line touched or removed).
- Manually verified the CLI-imports-only-press/ boundary (04-08's mechanical `check-cli-imports.sh` gate doesn't exist yet): `go list -f '{{join .Imports "\n"}}' ./cmd/eden-press/...` shows exactly `fmt`, `io`, `os`, `github.com/AO-Cyber-Systems/eden-press/press`, `github.com/knadh/koanf/providers/posflag`, `github.com/knadh/koanf/v2`, `github.com/spf13/cobra` — no `chase/`, no `profiles/`, no `chromedp`.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: cobra tree + stub seams + go.mod deps | `go build ./... && go run ./cmd/eden-press --help && go vet ./... && gofmt -l cmd/eden-press/ && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: flag surface + koanf buildOptions | `go build ./... && go test ./cmd/eden-press/ -run 'TestBuildOptions\|TestPosflag' -v && go vet ./... && gofmt -l cmd/eden-press/flags.go cmd/eden-press/options.go cmd/eden-press/flags_test.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 3: input resolution (stdin/file) | `go build ./... && go test ./cmd/eden-press/ -run TestResolveInput -v && go vet ./... && gofmt -l cmd/eden-press/input.go cmd/eden-press/input_test.go && bash scripts/check-no-chromedp.sh && addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: cobra command tree + stub seams + go.mod deps** - `14046ac` (feat)
2. **Task 2: flag surface + koanf-backed buildOptions** - `daa1c71` (feat)
3. **Task 3: input resolution — stdin (-) vs file** - `b9f5480` (feat)

_Note: Tasks 2 and 3 are `tdd="true"`; RED (compile failure against undefined symbols) confirmed before each GREEN implementation — see TDD Evidence below. Task 1 is glue verified by `go build` + a smoke `--help`, per the TRD's own test-list framing._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./cmd/eden-press/...` (and whole-repo `go test ./...`) | 0 | PASS |
| gofmt | `gofmt -l cmd/eden-press/` | 0 (no output) | PASS |
| no_chromedp | `bash scripts/check-no-chromedp.sh` | 0 ("PASS: no chromedp in the press/chase/profiles dependency closure.") | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 2) | `go test ./cmd/eden-press/ -run 'TestBuildOptions\|TestPosflag' -v` | 1 (compile failure: undefined `registerPersistentFlags`/`registerConvertFlags`/`buildOptions`) | FAIL (correct) |
| GREEN (Task 2) | `go test ./cmd/eden-press/ -run 'TestBuildOptions\|TestPosflag' -v` | 0 (3/3 pass, after 1 fix iteration — see Deviations) | PASS (correct) |
| RED (Task 3) | `go test ./cmd/eden-press/ -run TestResolveInput -v` | 1 (compile failure: undefined `resolveInputFrom`/`stdinSource`/`fileSource`) | FAIL (correct) |
| GREEN (Task 3) | `go test ./cmd/eden-press/ -run TestResolveInput -v` | 0 (3/3 pass) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1 (Task 2's initial `flags_test.go` used `cmd.Flags().Set(...)` directly on an unexecuted `*cobra.Command`, which fails with "no such flag" because persistent flags are only merged into `cmd.Flags()` by `ParseFlags()`/`Execute()`'s `mergePersistentFlags()` — fixed by switching all 3 tests to `cmd.ParseFlags(...)`, which is also what makes the Pitfall-5 guard test genuinely exercise the merge path instead of passing vacuously — see Deviations)
- **Must-haves verified:** 6/6 (all `must_haves.truths` from 04-02-TRD.md frontmatter — cobra tree, flag surface + buildOptions, posflag three-arg wiring, input.go source-observable resolution, all stub seams compiling, CLI imports only press/)
- **Gate failures:** None remaining (fixed within Task 2's own TDD RED->GREEN loop, before the task commit)

## Files Created/Modified
- `cmd/eden-press/main.go` - entrypoint: builds root cmd via `newRootCmd()`, `Execute()`, exits 1 with stderr message on error
- `cmd/eden-press/root.go` - root cmd: `Args: cobra.MaximumNArgs(1)`, `PersistentPreRunE: applyConfig`, `RunE: runConvert` (default-convert path), `AddCommand` the four subcommands, Pitfall-1 escape hatch documented in `Long`
- `cmd/eden-press/flags.go` - `registerPersistentFlags(root)` + `registerConvertFlags`/`registerWatchFlags`/`registerServeFlags(cmd)` per-mode local flags
- `cmd/eden-press/options.go` - package `cfg = koanf.New(".")`, `applyConfig(cmd)`, `buildOptions(cmd) (press.Options, error)`
- `cmd/eden-press/input.go` - `inputSource` enum (`fileSource`/`stdinSource`), `resolveInput(arg)`, testable core `resolveInputFrom(arg, stdin io.Reader)`
- `cmd/eden-press/convert.go` - `newConvertCmd()` + `runConvert` STUB (not implemented — 04-03)
- `cmd/eden-press/watch.go` - `newWatchCmd()` + `runWatch` STUB (not implemented — 04-06)
- `cmd/eden-press/serve.go` - `newServeCmd()` + `runServe` STUB (not implemented — 04-07)
- `cmd/eden-press/preview.go` - `newPreviewCmd()` + `runPreview` STUB (not implemented — 04-08)
- `cmd/eden-press/config.go` - `loadConfigSources(k, cmd)` posflag-only baseline STUB (04-04 prepends file+env)
- `cmd/eden-press/themeset.go` - `themeCSS(cmd)` STUB (returns `nil, nil` — 04-05 fills)
- `cmd/eden-press/flags_test.go` - `TestBuildOptionsMapsSetFlags`, `TestBuildOptionsLeavesUnsetFlagsZero`, `TestPosflagInstanceGuard`
- `cmd/eden-press/input_test.go` - `TestResolveInputStdin`, `TestResolveInputFile`, `TestResolveInputMissingFile`
- `go.mod` / `go.sum` - additive-only `go get` provisioning of all four CLI dep groups (cobra, fsnotify, koanf+parsers+providers, pkg/browser)

## Decisions Made
- `press.Options.ThemeCSS` wiring is deferred to post-merge with sibling Wave-1 TRD 04-01 (see key-decisions above) — documented inline in `options.go`'s `buildOptions` doc comment with the exact one-line follow-up.
- `options.go`/flag-registration calls were introduced incrementally across Tasks 1 and 2 (both within this TRD's own exclusive file set) rather than strictly per the TRD's per-task `<files>` list, because Go compiles a package as a whole and `root.go` (Task 1) needs `applyConfig` to exist to compile — see key-decisions above.
- `--auto-fit-script` is registered as a PERSISTENT (root-level) flag, not per-subcommand, because convert/watch/serve all read the identical `cfg.Bool("auto-fit-script")` key and must resolve it identically regardless of invocation mode.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `cmd.Flags().Set(...)` fails on unexecuted cobra.Command — persistent flags not yet merged**
- **Found during:** Task 2, TDD RED->GREEN loop (initial `flags_test.go` draft)
- **Issue:** `flags_test.go:65: Flags().Set("theme", "gaia"): no such flag -theme`. Root cause: `cobra.Command.PersistentFlags()` is only merged into `cmd.Flags()` by `ParseFlags()` (internally, `mergePersistentFlags()`), which normally runs as part of `Execute()`. A standalone, never-executed `*cobra.Command` built purely for a unit test does NOT expose its persistent flags via `cmd.Flags().Set(...)` until that merge happens.
- **Fix:** Rewrote all 3 `flags_test.go` tests to call `cmd.ParseFlags([]string{...})` (or `cmd.ParseFlags(nil)` when no flags need setting) instead of `cmd.Flags().Set(...)`. This is not just a workaround — it makes the tests genuinely exercise the real merge + posflag path (identical to what a real `Execute()` does), rather than passing vacuously against an incompletely-initialized command.
- **Files modified:** cmd/eden-press/flags_test.go
- **Verification:** `go test ./cmd/eden-press/ -run 'TestBuildOptions|TestPosflag' -v` — 3/3 PASS.
- **Committed in:** daa1c71 (Task 2 commit)

**2. [Rule 2 - Missing functionality, documented deferral] `press.Options.ThemeCSS` field does not exist in this worktree**
- **Found during:** Task 2, `buildOptions` implementation (codebase_examples shows `ThemeCSS: tcss` assigned onto the returned `press.Options`)
- **Issue:** `press.Options` in this worktree (forked from commit `3100281`, before Wave-1 TRD 04-01 merges) has no `ThemeCSS` field — that field is added by 04-01, executed in a separate parallel git worktree not yet merged into this one at execution time. This is the expected wave-1/wave-2 parallel-worktree pattern (independently confirmed by sibling TRD 04-03's own `error_recovery` text stating 04-01 must merge first), not a planning error.
- **Fix:** `buildOptions` still calls `themeCSS(cmd)` and propagates any error (satisfying the "single resolution point" must-have and keeping the seam exercised), but does not assign the `[]string` result onto a nonexistent `Options.ThemeCSS` field. A doc comment states the exact one-line change needed once 04-01 merges.
- **Files modified:** cmd/eden-press/options.go
- **Verification:** `go build ./...` green against the current (pre-04-01-merge) `press.Options` shape; `themeCSS` stub is exercised via `buildOptions`'s call, just not yet wired into the struct literal.
- **Committed in:** daa1c71 (Task 2 commit)

---

**Total deviations:** 2 (1 Rule 3 - Blocking test-authoring bug, fixed within Task 2's own TDD loop before commit; 1 Rule 2 - documented deferral pending a sibling Wave-1 TRD's merge, not a defect)
**Impact on plan:** Neither changes 04-02's shipped scope. The ThemeCSS deferral is a deliberate, clearly-marked seam for 04-01/04-05 to complete post-merge — exactly the kind of cross-worktree reconciliation the wave structure anticipates.

## Issues Encountered
None beyond the two auto-fixed/documented deviations above.

## User Setup Required
None — no external service configuration required. `go get` fetched all four dependency groups successfully (no offline/BLOCKER path triggered).

## Next Objective Readiness
- 04-01 (Wave 1, separate worktree) must merge before 04-03/04/05 can complete their `press.Options.ThemeCSS` wiring — the one-line follow-up in `buildOptions` is already documented.
- 04-03 (convert pipeline) fills `runConvert` in `convert.go` exclusively.
- 04-04 (config file + env precedence) fills `loadConfigSources` in `config.go` exclusively, prepending file+env providers ahead of the existing posflag load — `options.go`/`buildOptions` needs no changes.
- 04-05 (theme-set loading) fills `themeCSS` in `themeset.go` exclusively — `buildOptions` already calls it, so once 04-01's `ThemeCSS` field exists, only the one-line struct-literal edit in `options.go` is needed.
- 04-06/04-07/04-08 fill `runWatch`/`runServe`/`runPreview` respectively, each an exclusive file with no cross-TRD collision.
- All 8 CLI dependency groups are now in go.mod; no downstream CLI TRD needs to touch go.mod.

## Self-Check: PASSED

All claimed files confirmed present on disk; all three task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: cmd/eden-press/main.go
- FOUND: cmd/eden-press/root.go
- FOUND: cmd/eden-press/flags.go
- FOUND: cmd/eden-press/options.go
- FOUND: cmd/eden-press/input.go
- FOUND: cmd/eden-press/convert.go
- FOUND: cmd/eden-press/watch.go
- FOUND: cmd/eden-press/serve.go
- FOUND: cmd/eden-press/preview.go
- FOUND: cmd/eden-press/config.go
- FOUND: cmd/eden-press/themeset.go
- FOUND: cmd/eden-press/flags_test.go
- FOUND: cmd/eden-press/input_test.go
- FOUND commit: 14046ac (Task 1)
- FOUND commit: daa1c71 (Task 2)
- FOUND commit: b9f5480 (Task 3)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
