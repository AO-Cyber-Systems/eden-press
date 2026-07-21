---
objective: 04-cli
job: "04"
subsystem: cli
tags: [koanf, cobra, config-file, precedence, yaml, json, toml, posflag]

# Dependency graph
requires:
  - objective: 04-cli
    provides: "04-02's cobra skeleton: registerPersistentFlags, the package-level `cfg *koanf.Koanf`, `applyConfig`/`buildOptions`, and the posflag-only baseline stub in config.go (loadConfigSources) that this TRD replaces"
provides:
  - "loadConfigSources: full koanf load chain -- project-local .marprc.{yml,yaml,json,toml} (or --config override) via an explicit extension->parser switch, then EDEN_PRESS_* env, then posflag.Provider(cmd.Flags(), \".\", k) LAST -- precedence flags > env > file > compiled defaults"
  - "discoverConfigPath: --config override else first-match .marprc.* discovery in cwd (no global/XDG path in v1)"
  - "parserFor: extension-routed yaml/json/toml koanf.Parser selection with a clear error on unsupported extensions"
affects: [04-03-htmldoc, 04-05-themeset, 04-06-watch, 04-07-serve]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "koanf load-order-is-precedence: file loaded first (lowest), env second, posflag.Provider(cmd.Flags(), \".\", k) LAST (highest) with the LIVE koanf instance as the third arg -- the Pitfall-5 instance guard that lets posflag skip merging a flag's unchanged default over an already-set file/env value"
    - "Explicit extension->parser switch (parserFor) instead of relying on koanf's format sniffing, which does not exist by design"
    - "Env key mapper (EDEN_PRESS_HIGHLIGHT_STYLE -> highlight-style) reuses the SAME dash-separated lowercase key namespace the flags/koanf/buildOptions already share -- no new mapping table needed"

key-files:
  created: []
  modified:
    - cmd/eden-press/config.go
    - cmd/eden-press/config_test.go

key-decisions:
  - "Config-file discovery is project-local only (.marprc.{yml,yaml,json,toml} in cwd, first match wins) with --config as the sole override -- no global/XDG path in v1, matching Marp CLI and the TRD's resolved Open Question 5"
  - "Only config.go/config_test.go were touched -- buildOptions/options.go/flags.go untouched, so config-file/env support flows into every mode (convert/watch/serve) through the existing cfg read seam without any downstream TRD editing that mapping"

patterns-established:
  - "Precedence-chain test harness: newTestConvertCmd() + resetCfg() + t.TempDir()+os.Chdir (for discovery) or absolute --config paths, t.Setenv for EDEN_PRESS_* -- reusable by any future config-surface test"

requirements-completed: [CLI-06]

# Verification evidence
verification:
  gates_defined: 5
  gates_passed: 5
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 5min
completed: 2026-07-21
---

# Objective 4 TRD 04: koanf Config-File Loading (CLI-06) Summary

**loadConfigSources now chains project-local .marprc.{yml,yaml,json,toml} (or --config) → EDEN_PRESS_* env → posflag LAST, giving every CLI mode precedence flags > env > file > defaults with the Pitfall-5 posflag-instance guard intact.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-21T16:14:00Z
- **Completed:** 2026-07-21T16:16:51Z
- **Tasks:** 1/1 complete
- **Files modified:** 2 (cmd/eden-press/config.go, cmd/eden-press/config_test.go)

## Accomplishments
- `loadConfigSources(k *koanf.Koanf, cmd *cobra.Command) error` replaces the 04-02 posflag-only baseline: file (ext-routed via `parserFor`) → env (`EDEN_PRESS_` prefix, dash-mapped) → `posflag.Provider(cmd.Flags(), ".", k)` LAST, in that exact order.
- `discoverConfigPath`: `--config` flag overrides entirely; otherwise searches cwd for `.marprc.yml`/`.marprc.yaml`/`.marprc.json`/`.marprc.toml`, first match wins. No global/XDG path (v1 scope, matches Marp CLI).
- `parserFor`: explicit extension switch (koanf does not sniff formats) routing `.yml`/`.yaml` → yaml.Parser(), `.json` → json.Parser(), `.toml` → toml.Parser(); unsupported extensions return a clear error.
- 9 new test functions/subtests in `config_test.go` cover all 6 test-list cases: file→options, flag-over-file, env-over-file/flag-over-env, ext routing (json/toml/unsupported-error), `--config` overriding discovery, and the Pitfall-5 instance guard through the full chain.
- Only `config.go`/`config_test.go` changed — `buildOptions`, `options.go`, `flags.go` untouched, so config-file/env support now flows into convert/watch/serve through the same `cfg` reads those modes already perform.
- 04-02's stdin/file input path (`input.go`) reconfirmed unaffected — its 3 existing tests (`TestResolveInputStdin/File/MissingFile`) still pass unchanged.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Fill loadConfigSources — file(ext-routed)→env→posflag-last + .marprc.* discovery | `go build ./... && go test ./cmd/eden-press/ -run 'TestConfig\|TestPrecedence' -v && go test ./cmd/eden-press/... && go vet ./... && gofmt -l cmd/eden-press/config.go cmd/eden-press/config_test.go && bash scripts/check-no-chromedp.sh && addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/config.go cmd/eden-press/config_test.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Fill loadConfigSources** — `d1bbf77` (test, RED phase) → `3aab013` (feat, GREEN phase)

_Note: `tdd="true"`; RED confirmed (5/6 new precedence/routing/discovery test functions failed against the 04-02 posflag-only stub — the 6th, flag-over-file, incidentally already passed since posflag alone already honored an explicitly-set flag) before the GREEN implementation — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (scoped) | `go test ./cmd/eden-press/... -v` | 0 | PASS |
| test (whole-repo) | `go test ./...` | 0 | PASS |
| gofmt | `gofmt -l cmd/eden-press/config.go cmd/eden-press/config_test.go` | 0 (no output) | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/config.go cmd/eden-press/config_test.go` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED | `go test ./cmd/eden-press/... -run 'TestConfig\|TestPrecedence\|TestExtRouting\|TestPitfall5' -v` (against 04-02's posflag-only stub) | 1 (`TestConfigFileToOptions`, `TestPrecedenceEnvOverFileFlagOverEnv`, `TestExtRouting/{json,toml,unsupported}`, `TestConfigFlagOverridesDiscovery`, `TestPitfall5GuardThroughFullChain` failed) | FAIL (correct) |
| GREEN | `go test ./cmd/eden-press/... -run 'TestConfig\|TestPrecedence\|TestExtRouting\|TestPitfall5' -v` (against filled loadConfigSources) | 0 (all 6 test-list cases pass) | PASS (correct) |
| REFACTOR | none needed — implementation matched the TRD's `codebase_examples` sketch on the first pass | — | — |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 04-04-TRD.md frontmatter: file→env→posflag-last chain; project-local `.marprc.*` discovery with `--config` override, no global path; Pitfall-5 instance guard proven by a dedicated regression test; only `config.go` modified, config flows through the existing `cfg`→`buildOptions` seam)
- **Gate failures:** None

## Files Created/Modified
- `cmd/eden-press/config.go` — `loadConfigSources` (file→env→posflag-last chain), `parserFor` (yaml/json/toml extension routing), `discoverConfigPath` (`--config` override + `.marprc.*` cwd discovery)
- `cmd/eden-press/config_test.go` — 6 test-list cases across 9 test functions/subtests (file→options, flag-over-file, env-over-file/flag-over-env, ext routing ×3, `--config` override, Pitfall-5 full-chain guard)

## Decisions Made
- No global/XDG config path in v1 — project-local `.marprc.*` (cwd) only, `--config` as the sole override, matching Marp CLI's own behavior (research Open Question 5, RESOLVED).
- The env key mapper (`EDEN_PRESS_HIGHLIGHT_STYLE` → `highlight-style`) reuses the exact dash-separated lowercase key namespace flags/koanf/buildOptions already share — no separate mapping table was introduced.
- Left `buildOptions`/`options.go`/`flags.go` completely untouched (per the TRD's own scope boundary), confirming the 04-02 `cfg` seam design pays off exactly as intended: filling one function gives every mode config support for free.

## Deviations from Plan

None - TRD executed exactly as written. The `codebase_examples` sketch in 04-04-TRD.md matched the working implementation on the first pass; no bugs, missing functionality, or blocking issues were discovered during execution.

## Issues Encountered
None.

## User Setup Required
None — no external service configuration required. koanf and its parsers/providers (`yaml`, `json`, `toml`, `env`, `file`, `posflag`) were already provisioned in `go.mod` by 04-02; this TRD only imports them (no `go.mod`/`go.sum` changes).

## Next Objective Readiness
- 04-03 (htmldoc/convert pipeline, CLI-01) and 04-05 (`--theme`/`--theme-set`, CLI-05) — both wave-2 siblings — can now rely on `cfg` carrying config-file/env-resolved values by the time `buildOptions` runs, with no changes needed on their end.
- 04-06 (watch, CLI-02) and 04-07 (serve, CLI-03) inherit config-file/env support transparently through the same `applyConfig`/`buildOptions` seam once merged.
- CLI-06 (config file loading + stdin input) is now fully covered: stdin (04-02's `input.go`) plus this TRD's koanf config chain.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: cmd/eden-press/config.go
- FOUND: cmd/eden-press/config_test.go
- FOUND: .planning/objectives/04-cli/04-04-SUMMARY.md
- FOUND commit: d1bbf77 (Task 1, RED)
- FOUND commit: 3aab013 (Task 1, GREEN)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
