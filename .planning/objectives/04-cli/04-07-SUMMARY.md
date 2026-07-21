---
objective: 04-cli
trd: "07"
subsystem: cli
tags: [http, serve, live-reload, sse, traversal-guard, cobra]

# Dependency graph
requires:
  - objective: 04-cli
    provides: "04-03's render pipeline (buildOptions -> press.Render -> assembleHTML) + the assembleHTML InjectScripts seam this TRD splices the reload client through"
  - objective: 04-cli
    provides: "04-06's cmd/eden-press/reload package: the SSE Hub (subscribe/unsubscribe/Broadcast/ServeHTTP) and embedded ClientJS(url) EventSource snippet -- reused verbatim, not rebuilt"
provides:
  - "runServe: fills the 04-02 stub -- an http.ServeMux rooted at an absolute server directory, serving static files, converting markdown-extension requests to HTML on every request through 04-03's pipeline, and live-reloading via the reused 04-06 SSE Hub mounted directly on the same mux"
  - "safeJoin: the directory-traversal guard (research Pitfall 7) -- normalizes any request path to a single rooted path via filepath.Clean(\"/\"+urlPath) before Join'ing onto the resolved-once absolute root, then re-verifies containment via filepath.Rel as an explicit second guard"
  - "serveMux/serveFileHandler/serveMarkdown/isMarkdown/buildServeOptions/resolveServeAddr: small, independently-unit-tested pure/near-pure helpers factored out of runServe's HTTP wiring"
affects: [04-08-preview]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "filepath.Clean(\"/\"+urlPath) before Join is the core traversal-containment trick: Clean always resolves a leading \"..\" on a rooted path down to \"/\", never past it, so the joined result can never point outside root by construction -- the follow-on filepath.Rel prefix check is kept anyway as an explicit, auditable second guard (Marp CLI's own defense-in-depth precedent)"
    - "serve mounts reload.Hub.ServeHTTP directly on its OWN existing mux (no hub.Start()) -- distinct from watch (04-06), which owns no other HTTP server and so calls hub.Start() to bind its own ephemeral listener; both reuse the identical Hub type"
    - "Convert-on-request, no cache: every GET to a markdown-extension path re-runs buildServeOptions -> press.Render -> assembleHTML from scratch (matches Marp CLI's own serve behavior; a v1 scope decision, not an oversight)"
    - "HTML-only serve, no ?pdf/?png query switching: the handler never inspects r.URL.RawQuery at all (research Pattern 5 scope correction -- format conversion is convert/'s job, Objectives 5/6)"

key-files:
  created:
    - cmd/eden-press/serve_test.go
  modified:
    - cmd/eden-press/serve.go

key-decisions:
  - "safeJoin operates on r.URL.Path (already percent-decoded by net/http), not the raw escaped request target -- '%2f' and a literal '/' are indistinguishable by the time safeJoin sees them, so one containment check covers both encoded and unencoded traversal attempts"
  - "A plain '../' traversal attempt is neutralized by http.ServeMux's own escaped-path cleaning before it even reaches our handler (Go 1.22+ ServeMux cleans r.URL.EscapedPath()); a '%2f'-encoded variant has no literal slash in its escaped form, so it reaches serveFileHandler directly -- safeJoin still contains it, but http.ServeFile's own containsDotDot(r.URL.Path) guard then independently 400s the original request as a third, redundant layer. All three layers (ServeMux clean-redirect, safeJoin, ServeFile's own guard) agree: the sentinel file placed outside root is never served, across 400/403/404 outcomes"
  - "buildServeOptions is kept as its own named function (a thin call to buildOptions) rather than calling buildOptions directly from serveMarkdown, matching the TRD's explicit action spec for a documented serve-side entry point, without diverging from buildOptions' resolution logic at all"
  - "resolveServeAddr defensively re-checks port==0 -> 8321 even though the --port flag's own registered default is already 8321 (flags.go) -- consistent with the TRD's own reference implementation and defensive against a bare struct/test path that never ran the flag through cobra's default-population"

requirements-completed: [CLI-03]

# Verification evidence
verification:
  gates_defined: 4
  gates_passed: 4
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 20min
completed: 2026-07-21
---

# Objective 4 TRD 07: Serve Mode -- Static + Convert-on-Request + Traversal Guard + SSE Reload (CLI-03) Summary

**`eden-press serve [dir]` serves static files and converts markdown on every request through 04-03's pipeline, guarded by a Pitfall-7 traversal check (`filepath.Clean("/"+urlPath)` + `filepath.Rel` containment) and live-reloaded over the 04-06 SSE Hub mounted directly on its own mux, on a non-8080 default port (8321).**

## Performance

- **Duration:** ~20 min (research/read-through + implementation + verification)
- **Tasks:** 1/1 complete
- **Files:** 1 created, 1 modified

## Accomplishments
- `runServe` fills the 04-02 stub: resolves the server root to an absolute path once (`filepath.Abs`), builds an `http.ServeMux` via the new `serveMux` (routing `/__reload` -> the reused `reload.Hub` and `/` -> `serveFileHandler`), resolves the listen address via `resolveServeAddr` (loopback `127.0.0.1` default, product default port **8321**, never 8080), prints the serving URL, and calls `http.ListenAndServe`.
- `safeJoin` is the directory-traversal guard (research Pitfall 7): every request path is normalized to a single rooted path (`filepath.Clean("/"+urlPath)`) before being joined onto the resolved root -- a rooted `Clean` always resolves a leading `..` down to `/`, never past it, so the joined result can never escape root by construction; `filepath.Rel` + prefix rejection is kept as an explicit, auditable second guard, matching Marp CLI's own defense-in-depth precedent even atop stdlib routing.
- Markdown-extension requests (`.md`/`.markdown`, case-insensitive via `isMarkdown`) are converted **on every request** (no cache in v1, matching Marp CLI) through the identical `buildServeOptions -> press.Render -> assembleHTML` chain 04-03 built, splicing the reload client through the `InjectScripts` seam via `reload.ClientJS("/__reload")`; the query string is never inspected anywhere in the handler (v1 is HTML-only -- no `?pdf`/`?png` format-switching, research Pattern 5 scope correction). Everything else falls through to `http.ServeFile` verbatim.
- serve reuses `reload.Hub` by mounting its `ServeHTTP` directly on serve's own mux (no `hub.Start()`), distinct from watch (04-06), which owns no other HTTP server and binds its own ephemeral listener -- one Hub type, two valid mounting patterns, zero duplicated reload logic.
- 8 new test functions (some with subtests) in `serve_test.go` cover all 6 TRD test-list cases plus a dedicated `safeJoin` containment proof across 6 crafted inputs (deep `../` chains, an absolute-looking path, empty path) -- all resolve to a path contained under root, several sub-cases proving a real HTTP round-trip through `httptest.NewServer` never leaks a sentinel file placed one directory above root.
- Zero `go.mod`/`go.sum` changes; confirmed via `check-no-chromedp.sh` that the CLI package still imports no `chase/`/`profiles/`/chromedp.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Fill runServe -- static + convert-on-request + traversal guard + SSE reload | `go build ./... && go test ./cmd/eden-press/ -run 'TestServe\|TestTraversal\|TestSafeJoin' -v && go test ./cmd/eden-press/... && go vet ./... && gofmt -l cmd/eden-press/serve.go cmd/eden-press/serve_test.go && bash scripts/check-no-chromedp.sh && addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/serve.go cmd/eden-press/serve_test.go` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Fill runServe** -- `65ad5a5` (feat)

_Note: this task is `tdd="true"`; RED was confirmed via a compile failure (`undefined: serveMux/safeJoin/resolveServeAddr`) before the GREEN implementation was written -- see TDD Evidence below. Per this TRD's explicit "one atomic commit per task" instruction (a single-task TRD), the RED test file and GREEN implementation were verified together and committed as one `feat` commit, rather than a separate `test` + `feat` commit pair._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole-repo) | `go test ./...` | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| gofmt | `gofmt -l cmd/eden-press/serve.go cmd/eden-press/serve_test.go` | 0 (no output) | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/serve.go cmd/eden-press/serve_test.go` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED | `go test ./cmd/eden-press/...` against the 04-02 `runServe` stub + new `serve_test.go` | 1 (`undefined: serveMux`, `undefined: safeJoin`, `undefined: resolveServeAddr`) | FAIL (correct) |
| GREEN | `go test ./cmd/eden-press/ -run 'TestServe\|TestTraversal\|TestSafeJoin' -v` (8 tests, some with subtests) | 0 | PASS (correct) |
| GREEN (whole-repo) | `go test ./...` | 0 | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 04-07-TRD.md frontmatter: static + convert-on-request + reused reload HTML-only serve; traversal guard resolved-root-once + contained-before-file-I/O; non-8080 default port 8321 overridable + loopback default; reused reload Hub + reused render pipeline, zero duplication)
- **Gate failures:** None (one test-expectation refinement made mid-verification, documented below -- not a gate failure)

## Files Created/Modified
- `cmd/eden-press/serve.go` -- filled the 04-02 stub: `runServe`, `serveMux`, `serveFileHandler`, `serveMarkdown`, `buildServeOptions`, `isMarkdown`, `safeJoin`, `resolveServeAddr`
- `cmd/eden-press/serve_test.go` -- 8 test functions (some with subtests): convert-on-request, traversal-rejected (plain + `%2f`-encoded, sentinel-file proof), `safeJoin` containment across 6 crafted inputs, static-file-served, reload-endpoint broadcast, format-switch-query-ignored, port-default-and-override

## Decisions Made
- `safeJoin` operates on `r.URL.Path` (already percent-decoded by `net/http`) rather than the raw escaped request target -- a `%2f`-encoded traversal attempt and a literal `/` are indistinguishable by the time `safeJoin` sees them, so one containment check covers both variants from the test-list.
- Discovered during verification (not a plan deviation, a test-expectation refinement): Go 1.22+'s `http.ServeMux` cleans `r.URL.EscapedPath()`, not the decoded `r.URL.Path` -- a plain `../` request (a real slash in its escaped form) gets neutralized by ServeMux's own clean-and-redirect before reaching our handler, while a `%2f`-encoded variant (no literal slash in its escaped form) reaches `serveFileHandler` directly with a still-`..`-bearing decoded `Path`. `safeJoin` still safely contains it, but `http.ServeFile`'s own `containsDotDot(r.URL.Path)` guard then independently rejects the *original* request with 400 (a third, redundant layer). The test asserts on the actually-observed, security-relevant property (sentinel content never leaks; status is one of 400/403/404) rather than a single fixed status code, and documents why each of the two crafted inputs is rejected by a different layer.
- `buildServeOptions` is kept as its own named function per the TRD's explicit action spec, even though it is currently a one-line pass-through to `buildOptions` -- it does not diverge from `buildOptions`'s flag/config-file/env resolution in any way.
- `resolveServeAddr` re-checks `port == 0 -> 8321` defensively, even though the `--port` flag's own registered default (`flags.go`) is already 8321 -- matches the TRD's reference implementation and stays correct for any call path that bypasses cobra's flag-default population.

## Deviations from Plan

None - TRD executed exactly as written. (The test-expectation refinement above was a correction made while writing/verifying the test itself during the same TDD cycle, before any commit was made -- not a deviation from an already-committed plan.)

## Issues Encountered
None beyond the test-expectation refinement documented above (resolved before commit, not a gate failure).

## User Setup Required
None -- no external service configuration required. Zero `go.mod`/`go.sum` changes; this TRD imports only `press/`, `cmd/eden-press/reload` (04-06), and stdlib (`net/http`, `net`, `path/filepath`, `strconv`, `strings`, `os`, `fmt`).

## Next Objective Readiness
- 04-08 (preview + CLI-imports CI gate, CLI-04) inherits a serve mode that already passes the no-chromedp/CLI-imports-only-press gate, and can reuse the same `buildServeOptions`/render-pipeline composition pattern if preview needs to spin up a local server.
- CLI-03 is fully covered: static file serving, convert-on-request (HTML-only, no format-switching), a directory-traversal guard verified against both plain and URL-encoded attack variants with a real sentinel file outside root, and live-reload reusing 04-06's Hub verbatim -- one reload mechanism, one render pipeline, no duplication.

## Self-Check: PASSED

All claimed files confirmed present on disk; all claimed commit hashes confirmed present in `git log --oneline --all`.

- FOUND: cmd/eden-press/serve.go
- FOUND: cmd/eden-press/serve_test.go
- FOUND: .planning/objectives/04-cli/04-07-SUMMARY.md
- FOUND commit: 65ad5a5 (Task 1, feat)

---
*Objective: 04-cli*
*Completed: 2026-07-21*
