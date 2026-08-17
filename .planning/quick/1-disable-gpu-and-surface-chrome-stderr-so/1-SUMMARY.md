---
objective: quick-1
job: 1
subsystem: convert/chrome
tags: [chrome, chromedp, gpu, diagnostics, production-hang]
requires: [chromedp v0.16.0]
provides:
  - convert.Options.BrowserLog
  - convert/chrome.browserOutput
  - "--disable-gpu on every browser eden-press launches"
affects: [convert/pdf, convert/png, cmd/eden-press-export]
tech-stack:
  added: []
  patterns: [fake-browser fixture, baseline-free argv recording]
key-files:
  created: []
  modified:
    - convert/convert.go
    - convert/chrome/session.go
    - convert/chrome/session_test.go
decisions:
  - "--disable-gpu passed as a bare chromedp.Flag, NOT via chromedp.DisableGPU"
  - "io.Discard substituted for a nil BrowserLog so chromedp never closes the pipe"
  - "recording fakes get a 3s lifetime; 300ms was measurably racy under -race"
metrics:
  tasks: 3
  commits: 3
  duration: ~35m
  completed: 2026-08-17
---

# Quick Job 1: disable-gpu and surface Chrome stderr so browser launch cannot hang silently

Chrome now launches with `--disable-gpu` and its combined stdout+stderr is
reachable through the new `convert.Options.BrowserLog`, closing a production
hang in `convert/chrome.New` that was invisible by construction.

## Commits

| Task | Commit | Subject |
|---|---|---|
| 1 | `83fc6d1` | feat(convert): add Options.BrowserLog for Chrome's combined output |
| 2 | `21a544e` | test(convert/chrome): pin disable-gpu and browser-output forwarding |
| 3 | `010a848` | fix(convert/chrome): disable GPU and forward Chrome output so launch cannot hang |

## What changed

**`convert/convert.go`** — added `BrowserLog io.Writer` to `Options`, and
corrected the `StartTimeout` doc comment, which told the handshake-only story.

**`convert/chrome/session.go`** — added the `browserOutput` helper, appended
`chromedp.Flag("disable-gpu", true)` and
`chromedp.CombinedOutput(browserOutput(opts.BrowserLog))` to the same
`allocOpts` slice passed to `NewExecAllocator`, and rewrote the root-cause
comments plus the timeout error text.

**`convert/chrome/session_test.go`** — added `fakeRecordingBrowser`,
`syncWriter`, and five tests.

## Verification

Run from `/Users/justin/dev/eden-press`, real output:

```
$ go build ./...
BUILD_EXIT=0

$ go test ./convert/...
ok   github.com/AO-Cyber-Systems/eden-press/convert         (cached)
ok   github.com/AO-Cyber-Systems/eden-press/convert/chrome  9.668s
ok   github.com/AO-Cyber-Systems/eden-press/convert/docx    (cached)
ok   github.com/AO-Cyber-Systems/eden-press/convert/pdf     (cached)
ok   github.com/AO-Cyber-Systems/eden-press/convert/png     (cached)
ok   github.com/AO-Cyber-Systems/eden-press/convert/pptx    (cached)
ok   github.com/AO-Cyber-Systems/eden-press/convert/xlsx    (cached)
TEST_EXIT=0
```

Validation gates: `go build ./...` exit 0, `go vet ./...` exit 0,
`go test ./... -race -count=1` — 36 packages ok, 0 FAIL (run twice), plus
`go test ./convert/chrome/ -race -count=3` ok.

### Task evidence

| Task | Verify command | Exit | Status |
|---|---|---|---|
| 1 | `go build ./... && go vet ./convert/...` | 0 | PASS |
| 1 | `go test ./convert/chrome/ -race -run TestBrowserOutput` | 0 | PASS |
| 2 | `go test ./convert/chrome/ -race -run 'TestNew...'` | 1 | RED as designed |
| 3 | `go test ./... -race -count=1` | 0 | PASS |

### TDD evidence

| Phase | Command | Exit | Expected |
|---|---|---|---|
| RED (task 1) | `go test ./convert/chrome/ -run TestBrowserOutput` | 1 | FAIL — `undefined: browserOutput` at both call sites |
| GREEN (task 1) | same | 0 | PASS |
| RED (task 2) | `go test ./convert/chrome/ -race -run 'TestNewLaunches\|TestNewDoesNot\|TestNewForwards'` | 1 | FAIL — see below |
| GREEN (task 3) | same | 0 | PASS |

Task 2's RED output was exactly as planned — two fail, one passes, none skip:

- `TestNewLaunchesBrowserWithDisableGPU` **FAIL**, dumping the full recorded
  argv. That dump is the useful artifact: it proves the fake really was
  executed with chromedp's real flag set, and that `--disable-gpu` is genuinely
  absent from `DefaultExecAllocatorOptions`.
- `TestNewDoesNotDisableSoftwareRasterizer` **PASS** — correct for a guard
  against a future edit.
- `TestNewForwardsBrowserOutputToBrowserLog` **FAIL** — `got ""`.

## Verified against the chromedp source, not assumed

Read `chromedp@v0.16.0/allocate.go` directly rather than trusting the framing:

- `--disable-gpu` is **absent** from `DefaultExecAllocatorOptions` (confirmed
  by reading the full list). This is a real change, not a no-op.
- `readOutput` closes the pipe when the forward writer is nil
  (`if forward == nil { rc.Close() }`), rather than the "full 64KB pipe buffer
  blocks Chrome" story. Post-startup output is destroyed at the source, which
  is why the GPU crash messages — emitted *after* the websocket URL is parsed —
  never reached the operator.
- `cmd.Stderr = cmd.Stdout` before `StdoutPipe()`, so the fake's stderr
  sentinel reaches the same pipe.

## Deviations from plan

**1. [Rule 1 - Bug] Fixed a timing race in the new tests (task 2's fixture)**

- **Found during:** task 3, on the full-repo `go test ./... -race` gate. The
  chrome package alone had passed repeatedly.
- **Issue:** `TestNewLaunchesBrowserWithDisableGPU` failed with "fake browser
  never recorded its argv". The fake is killed when `New` cancels its context
  at `StartTimeout`; at 300ms, under whole-repo `-race` CPU contention, the
  process was not exec'd in time to write argv. The other two tests using the
  same fixture passed in that run, confirming a race rather than a wiring bug.
- **Fix:** introduced `fakeBrowserLifetime = 3 * time.Second` for the three
  recording tests and raised `runNewAgainstFake`'s return deadline to match.
  Polling harder could not have fixed this — the poll begins after `New`
  returns, by which point the process is already dead, so the margin has to be
  in the browser's lifetime.
- **Not changed:** `TestNewStartTimeoutBoundsHangingBrowser` keeps its own
  300ms/5s bounds; its subject is the timeout itself.
- **Confirmed:** two consecutive full-repo `-race` runs and `-count=3` on the
  chrome package, all clean.
- **Commit:** `010a848`

**2. [Rule 3 - Blocking] DevFlow edit gate resolved against the wrong repo**

The session's cwd is an aodex worktree, and `gate-edits.js` resolves
`.planning` from `process.cwd()` — so it never saw the marker written into
eden-press. Set the marker via `df-tools skill-active --start quick` (the
gate's own documented mechanism) so edits to eden-press could proceed. No
aodex source was touched; the ephemeral markers are removed at exit.

## Observation for the operator (no action taken)

`chromedp.DisableGPU` — the helper, as opposed to the bare flag used here —
also sets `--enable-unsafe-swiftshader`, because per its own doc comment
**Chromium 139+ does not fall back to SwiftShader without it**. This job
deliberately ships only `--disable-gpu`, matching what was empirically verified
in the pod. If the export image is ever bumped to Chromium 139 or newer, PDF/PNG
rasterization should be re-verified — that is the version boundary where the
current flag set could stop rasterizing. Flagged, not acted on: adding the flag
would deviate from what was verified.

## Constraints honoured

- `--disable-software-rasterizer` is **not** added. It appears in the source
  only as a standing prohibition comment and as the guard test that fails if
  anyone adds it.
- `DefaultStartTimeout` (90s) and the goroutine/select mechanism unchanged;
  only comment wording changed.
- The timeout error still contains `did not become usable` and the `%q`
  executable path, both asserted by `TestNewStartTimeoutBoundsHangingBrowser`.
- `cmd/eden-press-export/main.go` deliberately **not** wired to `BrowserLog`
  (explicit non-goal).
- No Dockerfile, Kubernetes, gitops, or aodex-side changes. No tag, push, or
  deploy. ROADMAP.md untouched.

## Self-Check: PASSED

- `convert/convert.go`, `convert/chrome/session.go`,
  `convert/chrome/session_test.go` — all present and modified.
- Commits `83fc6d1`, `21a544e`, `010a848` — all present on
  `quick/chrome-disable-gpu`.
