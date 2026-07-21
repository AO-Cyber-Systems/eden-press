---
objective: 04-cli
trd: "06"
subsystem: cli
tags: [fsnotify, sse, debounce, watch, live-reload, cobra]

# Dependency graph
requires:
  - objective: 04-cli
    provides: "04-03's render pipeline (resolveInput -> buildOptions -> press.Render -> assembleHTML) + the assembleHTML InjectScripts seam this TRD splices the reload client through"
  - objective: 04-cli
    provides: "04-02's cobra skeleton: the newWatchCmd/runWatch stub, registerWatchFlags(--output/-o), and the package-level `cfg *koanf.Koanf` this TRD reads cfg.Strings(\"theme-set\")/cfg.Bool(\"auto-fit-script\") through"
provides:
  - "cmd/eden-press/reload: a stdlib-only SSE Hub (subscribe/unsubscribe/Broadcast + http.Handler writing text/event-stream frames) and an embedded (go:embed) ~5-line EventSource client.js, exposed via NewHub/Start/URL/Broadcast/Close and ClientJS(url) -- the SHARED live-reload plumbing serve (04-07) reuses verbatim"
  - "runWatch: a scoped (input-file dir + theme-set dirs), atomic-save-safe (watches the PARENT DIR, name-filtered), debounced (300ms) fsnotify watcher that reruns 04-03's render pipeline on change and broadcasts a reload over the SSE Hub"
  - "watchScope/eventTriggersRebuild/isBackupOrSwap/debounced/rebuildOnce/writeWatchOutput: small, independently-unit-tested pure helpers factored out of the fsnotify event loop"
affects: [04-07-serve]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Watch the PARENT DIRECTORY, never the target file directly, and filter fsnotify.Event.Name against a resolved watched-file set -- the only pattern that survives an editor's atomic write-temp-then-rename save (research Pitfall 2)"
    - "Debounce via a single mutex-guarded *time.Timer, Stop+reschedule (time.AfterFunc) on every call -- collapses N raw fsnotify events per logical save into exactly one rebuild"
    - "SSE (Server-Sent Events), stdlib-only: http.NewResponseController(w).Flush() per frame, subscribe-before-header-write so a test's http.Get race-free proves subscription, EventSource client auto-reconnects (no hand-rolled retry) -- deliberately NOT websocket (research Pattern 4)"
    - "go:embed client.js + fmt.Sprintf(%s) splices the live SSE endpoint URL into the ~5-line embedded EventSource snippet at ClientJS(url) call time"
    - "Reload script only ever reaches the document through 04-03's assembleHTML InjectScripts seam -- watch/serve opt in per-session; default convert stays zero-JS"
    - "cmd.Context().Done() bounds the otherwise-infinite watch loop -- cobra's ExecuteContext(ctx) propagates to a resolved subcommand's cmd.Context(), and falls back safely to a non-nil, never-firing context.Background() in normal Execute() usage"

key-files:
  created:
    - cmd/eden-press/reload/server.go
    - cmd/eden-press/reload/client.js
    - cmd/eden-press/reload/server_test.go
    - cmd/eden-press/watch_test.go
  modified:
    - cmd/eden-press/watch.go

key-decisions:
  - "reload.Hub binds an EPHEMERAL loopback port (net.Listen(\"tcp\", \"127.0.0.1:0\")), never a fixed port -- automatically satisfies the project's hard NEVER-8080 rule and needs no --port flag of its own in watch mode"
  - "eventTriggersRebuild is a pure predicate (Event, watched-set, dir-set) -> bool, factored out of the fsnotify event loop specifically so the Chmod/backup-swap/name-filter/dir-rescan rules are unit-testable without a real fsnotify.Watcher"
  - "watchScope is a small resolver (input dir + theme-set dirs), deliberately isolated so widening v1's scope to recursive-later is a change to one function, not a rewrite of runWatch's event loop (research Open Question 3)"

requirements-completed: [CLI-02]

# Verification evidence
verification:
  gates_defined: 7
  gates_passed: 7
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 10min
completed: 2026-07-21
---

# Objective 4 TRD 06: Watch Mode -- fsnotify + Debounce + SSE Reload (CLI-02) Summary

**`eden-press watch <in.md>` rebuilds via 04-03's pipeline on every save (parent-dir-watched, atomic-rename-safe, 300ms-debounced) and live-reloads any open browser over a stdlib SSE Hub shared verbatim with serve (04-07).**

## Performance

- **Duration:** ~10 min (first commit `da9ab7b` 17:11:53Z -> last commit `2d48136` 17:21:39Z)
- **Tasks:** 2/2 complete
- **Files:** 4 created, 1 modified

## Accomplishments
- `cmd/eden-press/reload`: a stdlib-only SSE `Hub` (subscribe/unsubscribe/Broadcast, `http.Handler` writing `text/event-stream` frames + `Flush` per message), an ephemeral-loopback listener (`Start`/`URL`/`Close`), and a `go:embed`'d ~5-line `client.js` spliced via `ClientJS(url)` -- built to be reused verbatim by serve (04-07), not re-implemented there.
- `runWatch` fills the 04-02 stub: rejects stdin (`-`) early (Pitfall 8); resolves the input file's parent dir (+ any `--theme-set` dirs) via `watchScope`; `fsnotify.NewWatcher().Add()`s the DIRECTORY (never the file, Pitfall 2); filters every event through `eventTriggersRebuild` (ignores `Chmod`, ignores editor backup/swap noise via `isBackupOrSwap`, treats a directory-level `Write` as a re-scan signal, matches the watched-file set by cleaned name -- which is exactly what survives an atomic rename); debounces rebuilds 300ms; reruns 04-03's `resolveInput -> buildOptions -> press.Render -> assembleHTML` pipeline via `rebuildOnce`, splicing the reload client through `InjectScripts` (watch-session output only); writes via `writeWatchOutput` (`-o` else `<input-stem>.html`); broadcasts `"reload"` on the Hub after every successful rebuild.
- The fsnotify event loop is bounded by `cmd.Context().Done()` (verified against `cobra@v1.10.2` source: `ExecuteContext(ctx)` propagates to a resolved subcommand's `cmd.Context()`), letting `TestRunWatchRebuildsOnAtomicSave` drive a real, time-bounded end-to-end atomic-save-and-rebuild proof rather than a mocked fsnotify.
- 12 new test functions/subtests across `reload/server_test.go` and `watch_test.go` cover all 6 TRD test-list cases (debounce-collapses-to-one, backup/swap filter, name filter + Chmod-ignored + dir-rescan-signal, stdin reject, SSE broadcast-to-one/broadcast-to-many, client.js-injected-only-in-watch) plus a real atomic-rename end-to-end proof.
- Zero `go.mod`/`go.sum` changes (fsnotify was already provisioned by 04-02); CLI package confirmed to import no `chase/`/`profiles/` directly.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: SSE reload Hub + embedded EventSource client | `go build ./... && go test ./cmd/eden-press/reload/... -v && go vet ./... && gofmt -l cmd/eden-press/reload/server.go cmd/eden-press/reload/server_test.go && bash scripts/check-no-chromedp.sh` | 0 | PASS |
| 2: Fill runWatch -- parent-dir watch, debounce, atomic-save-safe rebuild + reload | `go build ./... && go test ./cmd/eden-press/... && go vet ./... && gofmt -l cmd/eden-press/watch.go cmd/eden-press/watch_test.go && bash scripts/check-no-chromedp.sh && addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/watch.go cmd/eden-press/reload/server.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: SSE reload Hub + embedded client** -- `da9ab7b` (test, RED phase) -> `ec6df0a` (feat, GREEN phase)
2. **Task 2: Fill runWatch** -- `99c912b` (test, RED phase) -> `2d48136` (feat, GREEN phase)

_Note: both tasks are `tdd="true"`; RED confirmed by compile failure before each GREEN implementation was written -- see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l cmd/eden-press/watch.go cmd/eden-press/watch_test.go cmd/eden-press/reload/server.go cmd/eden-press/reload/server_test.go` | 0 (no output) | PASS |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (scoped) | `go test ./cmd/eden-press/... -v` (all 7 new watch/reload tests) | 0 | PASS |
| test (whole-repo) | `go test ./...` | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/watch.go cmd/eden-press/reload/server.go` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./cmd/eden-press/reload/... -v` against an empty `reload` package | 1 (`undefined: NewHub`, `undefined: ClientJS`) | FAIL (correct) |
| GREEN (Task 1) | `go test ./cmd/eden-press/reload/... -v` (5 tests: broadcast-to-one, broadcast-to-many, client.js splice, URL-empty-before-start, loopback-non-8080-port) | 0 | PASS (correct) |
| RED (Task 2) | `go build ./cmd/eden-press/...` against the 04-02 `runWatch` stub + new `watch_test.go` | 1 (`undefined: debounced`, `isBackupOrSwap`, `eventTriggersRebuild`, `watchScope`, `rebuildOnce`) | FAIL (correct) |
| GREEN (Task 2) | `go test ./cmd/eden-press/... -v -run 'TestDebounced\|TestIsBackupOrSwap\|TestEventTriggersRebuild\|TestWatchScopeResolvesInputAndThemeSetDirs\|TestRunWatchRejectsStdin\|TestRebuildOnceInjectsReloadClientOnly\|TestRunWatchRebuildsOnAtomicSave'` (7 tests) | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 1
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 04-06-TRD.md frontmatter: parent-dir watch + atomic-save-safe name filtering; 300ms debounce reusing 04-03's pipeline; stdlib SSE Hub with embedded client spliced via InjectScripts; watch-scope resolver + stdin rejection; the reload package built here is exactly what 04-07 will reuse)
- **Gate failures:** None (the one auto-fix cycle below was a pre-existing test-file bug found and fixed during Task 2 GREEN verification, not a validation-gate failure)

## Files Created/Modified
- `cmd/eden-press/reload/server.go` -- `Hub` (subscribe/unsubscribe/Broadcast/ServeHTTP/Start/URL/Close), `go:embed client.js` + `ClientJS(url)`
- `cmd/eden-press/reload/client.js` -- ~5-line `EventSource` reload snippet (Eden-authored, MIT header, NOT a Marp asset)
- `cmd/eden-press/reload/server_test.go` -- 5 tests: broadcast-to-one-subscriber, broadcast-to-many-subscribers, `ClientJS` URL splice, `URL()` empty before `Start`, loopback non-8080 port proof
- `cmd/eden-press/watch_test.go` -- 7 tests: debounce-collapses-to-one, `isBackupOrSwap` table, `eventTriggersRebuild` table (Write/Create/Rename-triggers, sibling-ignored, Chmod-ignored, backup-ignored, dir-rescan-triggers, unrelated-ignored), `watchScope` dir resolution, stdin-reject, reload-client-injected-exactly-once, end-to-end atomic-save-rebuild
- `cmd/eden-press/watch.go` -- filled the 04-02 stub: `runWatch`, `watchScope`, `eventTriggersRebuild`, `isBackupOrSwap`, `debounced`, `rebuildOnce`, `writeWatchOutput`

## Decisions Made
- `reload.Hub` binds an ephemeral loopback port (`127.0.0.1:0`), never a fixed port — the simplest way to guarantee it can never be `:8080` (the project's hard rule), and watch mode needs no `--port` flag of its own (that's serve's job in 04-07, product default 8321).
- `eventTriggersRebuild` is factored out as a pure function (not inlined in the `select` loop) specifically so the Chmod/backup-swap/name-filter/directory-rescan filtering rules are deterministically unit-testable without standing up a real `fsnotify.Watcher`.
- `cmd.Context().Done()` (not a hand-rolled `chan struct{}` / signal handler) bounds the watch loop — verified against `cobra@v1.10.2` source that a subcommand's `RunE` correctly receives the root's `ExecuteContext(ctx)` value, and falls back safely to a live, never-firing `context.Background()` in normal `Execute()` usage.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a duplicate-flag-registration panic in `TestRebuildOnceInjectsReloadClientOnly`**
- **Found during:** Task 2 GREEN verification (`go test ./cmd/eden-press/...`)
- **Issue:** `watch_test.go`'s `TestRebuildOnceInjectsReloadClientOnly` called `newTestConvertCmd()` (which already registers `--output`/`-o` via `registerConvertFlags`) and then also called `registerWatchFlags(cmd)`, which registers the identical `--output`/`-o` flag again on the same `cobra.Command` — `pflag.FlagSet.AddFlag` panics on a duplicate registration ("test flag redefined: output"), crashing the test binary.
- **Fix:** Removed the redundant `registerWatchFlags(cmd)` call; `newTestConvertCmd()`'s existing `--output` flag surface is sufficient (the comment already noted "persistent + auto-fit-script flag surface is enough").
- **Files modified:** `cmd/eden-press/watch_test.go`
- **Commit:** `2d48136` (folded into the Task 2 GREEN commit — the fix and the GREEN implementation were verified and committed together)

## Issues Encountered
None beyond the auto-fixed test bug above.

## User Setup Required
None — no external service configuration required. `fsnotify` was already provisioned in `go.mod` by 04-02; this TRD only imports it plus stdlib (`net/http`, `net`, `time`, `sync`, `path/filepath`, `embed`) and `press/` (no `go.mod`/`go.sum` changes).

## Next Objective Readiness
- 04-07 (serve, CLI-03) can import `cmd/eden-press/reload` verbatim — `NewHub`/`Start`/`URL`/`Broadcast`/`Close`/`ClientJS` are the exact shared plumbing it needs; no reload-mechanism work remains for that TRD.
- 04-08 (preview + CLI-imports CI gate, CLI-04) inherits a watch mode that already passes the no-chromedp/CLI-imports-only-press gate.
- CLI-02 is fully covered: scoped, atomic-save-safe, debounced fsnotify watch + stdlib SSE live-reload, reusing 04-03's render pipeline and script-injection seam without modifying either.

## Self-Check: PASSED

All claimed files confirmed present on disk; all claimed commit hashes confirmed present in `git log --oneline --all`.

- FOUND: cmd/eden-press/watch.go
- FOUND: cmd/eden-press/watch_test.go
- FOUND: cmd/eden-press/reload/server.go
- FOUND: cmd/eden-press/reload/client.js
- FOUND: cmd/eden-press/reload/server_test.go
- FOUND: .planning/objectives/04-cli/04-06-SUMMARY.md
- FOUND commit: da9ab7b (Task 1, RED)
- FOUND commit: ec6df0a (Task 1, GREEN)
- FOUND commit: 99c912b (Task 2, RED)
- FOUND commit: 2d48136 (Task 2, GREEN)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
