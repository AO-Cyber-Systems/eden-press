---
objective: quick-1
job: 1
type: standard
wave: 1
depends_on: []
files_modified:
  - convert/convert.go
  - convert/chrome/session.go
  - convert/chrome/session_test.go
autonomous: true
must_haves:
  goal: >-
    eden-press's Chrome launch cannot hang silently inside a GPU-less
    container, and when a launch does fail the operator gets Chrome's own
    output instead of nothing.
  truths:
    - Every Chrome that eden-press launches receives --disable-gpu.
    - No Chrome that eden-press launches receives --disable-software-rasterizer,
      so SwiftShader survives and PDF/PNG output still rasterizes.
    - A caller that sets Options.BrowserLog receives Chrome's combined
      stdout+stderr.
    - A caller that leaves Options.BrowserLog nil still causes chromedp to keep
      the Chrome output pipe open and drained (io.Discard is passed, never nil).
    - The existing StartTimeout bound still turns a browser that never becomes
      usable into a named error naming the executable it launched.
    - session.go's and convert.go's doc comments name the proven mechanism
      (GPU crash-loop starving Runtime.enable), not the earlier handshake guess.
  artifacts:
    - path: convert/convert.go
      provides: Options.BrowserLog io.Writer field with accurate doc comment
    - path: convert/chrome/session.go
      provides: browserOutput helper, disable-gpu flag, CombinedOutput wiring,
        corrected root-cause doc comments and timeout error message
    - path: convert/chrome/session_test.go
      provides: fakeRecordingBrowser fixture + argv/log regression tests
  key_links:
    - convert.Options.BrowserLog -> browserOutput() -> chromedp.CombinedOutput()
      -> allocOpts -> chromedp.NewExecAllocator. If any hop is missing the field
      is inert and the pipe still gets closed.
    - chromedp.Flag("disable-gpu", true) must be appended to the SAME allocOpts
      slice that is passed to NewExecAllocator, not to a copy.
---

# Quick Job 1 — disable-gpu and surface Chrome stderr so browser launch cannot hang silently

## Objective

Close the production hang in `convert/chrome.New`. Two changes, both scoped to
eden-press:

1. Pass `--disable-gpu` to Chrome.
2. Stop discarding Chrome's combined output; give callers an opt-in
   `io.Writer` and pass `io.Discard` (never `nil`) when they don't set one.

Then correct the doc comments so they describe the mechanism that was actually
proven, not the one that was originally guessed.

## Root cause (proven live, not inferred)

Reproduced in a production scratch pod on the same image with a CDP wire trace:

- Chrome's GPU/viz process crash-loops inside a GPU-less container.
- The page renderer therefore never obtains an execution context and never
  answers `Runtime.enable` — the third command chromedp sends while attaching
  to the first target (chromedp v0.16.0 `chromedp.go:445`, inside
  `attachTarget`).
- `Browser.Execute`'s only escape is `ctx.Done()`, so `chromedp.Run(rootCtx)`
  blocks.
- `New()` runs `chromedp.Run(rootCtx)` with zero actions purely to populate the
  Browser. That call takes chromedp's `first`-context path
  (`chromedp.go:385-425`), which waits for the browser's initial tab.
- Neither `chromedp.DefaultExecAllocatorOptions` nor eden-press passes
  `--disable-gpu`. **Verified this session** — see the flag list at
  `allocate.go` `DefaultExecAllocatorOptions`; `disable-gpu` is absent. This is
  a real change, not a no-op.

Measured in the real pod, same image, everything else held constant:

| Flags | Result |
|-------|--------|
| production flags (today) | HUNG past 45s |
| `+ --disable-gpu` | allocation returned in **808ms**, full raster render succeeded (5754-byte PNG of styled HTML) |

The browser **did** complete its DevTools handshake in the incident. The
websocket URL was parsed and the connection established; the hang came later,
at `Runtime.enable`. The current error text ("the process may have started
without completing the DevTools handshake") describes a *different* failure
than the one that happened, and must be broadened.

## HARD CONSTRAINT — do not add --disable-software-rasterizer

`--disable-gpu` **alone** was verified sufficient AND verified to still
rasterize correctly. `--disable-software-rasterizer` disables SwiftShader,
which is what actually rasterizes the PDF/PNG output. Adding it would risk
breaking real output. Task 2 pins this with an explicit absence assertion.

## Planner note — get the CombinedOutput rationale RIGHT

The briefing framed the nil-writer hazard as "nothing reads the pipe, so a
chatty Chrome blocks on a full 64KB pipe buffer." I read chromedp v0.16.0's
`allocate.go` this session and the actual mechanism is different — write the
doc comment from the source, not from the framing:

- `readOutput` (allocate.go ~line 286) reads line-by-line hunting for
  `"DevTools listening on"`. While hunting, it forwards every complete line to
  `forward` **only if `forward != nil`**.
- Once the URL is found: `if forward == nil { rc.Close() }` — chromedp
  **CLOSES the output pipe**. Everything Chrome says after startup is destroyed
  at the source. Chrome's later writes hit a closed pipe (EPIPE), they do not
  queue.
- With a writer set, `readOutput` instead returns a `copy` closure, and
  `allocate.go:253` (`if a.combinedOutputWriter != nil && copy != nil`) runs
  `io.Copy(forward, bufr)` on the allocator's WaitGroup for the browser's whole
  lifetime — output preserved *and* drained.

This is a stronger story than the buffer-block one, and it explains the
observed facts exactly: **the GPU crash-loop happens after the websocket URL is
parsed**, i.e. after the pipe was already closed. That is why two failed
production deploys produced ZERO browser diagnostics while Chrome was emitting
crash messages the entire time. Setting the writer produced the root cause in a
single run.

Passing `io.Discard` rather than leaving the option unset keeps the pipe open
and drained with no behaviour change for existing callers.

## Explicit non-goals

- **No aodex-side wiring, image rebuild, Docker, Kubernetes, gitops, or Cilium
  work.** Handled separately.
- **No deploy, tag, or push step.**
- **Do not change `DefaultStartTimeout` or the StartTimeout mechanism.** That
  bound is what converted an invisible hang into a diagnosable error and stays
  as a backstop. Only its *error message wording* changes.
- **Do not wire `BrowserLog` into `cmd/eden-press-export/main.go`.** Tempting
  (that binary is a real in-repo consumer) but out of scope for this job; the
  library surface is what the sidecar needs.

## Test list

Write these before the code they cover (`test_list_first: required`,
`tdd_default: strict`). Behaviour cases, outermost first:

1. **argv carries the fix** — a browser launched through `New` receives
   `--disable-gpu` in its argv.
2. **SwiftShader survives** — that same argv does NOT contain
   `--disable-software-rasterizer`.
3. **Output reaches an opted-in caller** — a fake browser that prints a
   sentinel line to stderr, with `Options.BrowserLog` set, lands that sentinel
   in the caller's writer.
4. **nil is never handed to chromedp** — `browserOutput(nil)` returns a
   non-nil writer (`io.Discard`), so chromedp never takes its pipe-closing
   branch.
5. **a caller's writer is passed through unchanged** — `browserOutput(w)`
   returns `w`.
6. **existing bound intact** — `TestNewStartTimeoutBoundsHangingBrowser` and
   `TestDefaultStartTimeoutIsPositive` still pass unmodified in substance.

## Codebase context

**Local test environment has NO discoverable Chrome.** Verified this session:

```
=== RUN   TestSessionMultiTab
    session_test.go:44: no Chrome discovered
--- SKIP: TestSessionMultiTab (0.00s)
--- PASS: TestNewStartTimeoutBoundsHangingBrowser (0.30s)
--- PASS: TestDefaultStartTimeoutIsPositive (0.00s)
```

`Discover` tier 3 does a `LookPath` for candidate binary *names*; on this macOS
box Chrome lives in `/Applications` and is not on `PATH`, so every live-Chrome
test skips. **The new tests must therefore be fake-browser based and must never
skip.** `Discover` tier 1 (`opts.BrowserPath`) short-circuits everything, which
is what makes the fake work with no Chrome installed.

**Existing fixture to mimic** — `session_test.go` already has
`fakeHangingBrowser`, a hand-written shell script standing in for Chrome. The
`exec sleep 120` detail is load-bearing (`exec` REPLACES the shell so
cancellation kills the actual sleeping process rather than orphaning a child).
Copy that idiom. **Leave `fakeHangingBrowser` and its test untouched** — it is
deliberately silent, and adding output to it would change what that test
exercises.

**How chromedp turns options into argv** (`allocate.go` `Allocate`):

```go
for name, value := range a.initFlags {
    case string: args = append(args, fmt.Sprintf("--%s=%s", name, value))
    case bool:   if value { args = append(args, fmt.Sprintf("--%s", name)) }
}
```

So `chromedp.Flag("disable-gpu", true)` becomes the bare argv token
`--disable-gpu`. That is what the argv assertion should look for.

**Stderr is merged into the same pipe** — `allocate.go` sets
`cmd.Stderr = cmd.Stdout` before `cmd.StdoutPipe()`, so a fake writing to
stderr reaches `readOutput`.

**Single construction point** — every caller in the repo goes through
`chrome.New(convert.Options{...})` (`cmd/eden-press-export/main.go:217`,
`convert/export_integration_test.go:100`, `convert/pdf/pdf_test.go:70`,
`convert/png/png_test.go:60`), all using keyed struct literals. Adding a field
to `convert.Options` is source-compatible with all of them.

## Anti-patterns

- Do NOT try to assert on `chromedp.ExecAllocator`'s fields. `initFlags` and
  `combinedOutputWriter` are unexported in package `chromedp`; being in package
  `chrome` does not help. Assert on **observed behaviour** (recorded argv, text
  arriving in a writer) instead.
- Do NOT gate the new tests on `Discover` / Chrome presence. They must run and
  pass on a box with no Chrome. A skipping regression test is not a regression
  test.
- Do NOT use an unsynchronised `bytes.Buffer` as the `BrowserLog` in a test.
  chromedp writes to it from `readOutput`'s goroutine while the test reads —
  `go test -race` will flag it. Use a small mutex-guarded writer.
- Do NOT generate test data with an LLM, add a property-based testing library,
  or introduce `.feature`/Cucumber scaffolding. Fixtures are hand-built and
  inline (`fixture_strategy: inline`).

## Gotchas

- The `--disable-gpu` flag must be appended to `allocOpts` **before**
  `chromedp.NewExecAllocator(context.Background(), allocOpts...)` is called.
- `chromedp.CombinedOutput(io.Discard)` and *omitting* the option are NOT
  equivalent — see the planner note above. Never pass a nil writer through;
  route it through the helper.
- `TestNewStartTimeoutBoundsHangingBrowser` asserts the error contains
  `"did not become usable"` and the executable path. Rewording the message's
  parenthetical is safe; removing either of those two substrings is not.
- `readOutput` only forwards **complete lines** (`ReadBytes('\n')`). The fake
  must terminate its sentinel with a newline — plain `echo` does.
- `New` returns at `StartTimeout` while the fake is still alive; the fake
  writes its argv within milliseconds of launch, so it is written well before
  `New` returns. A short bounded poll is cheap insurance on a loaded machine.

## Tasks

<task type="auto" tdd="true">
  <name>Add Options.BrowserLog and the nil-safe writer helper</name>
  <files>convert/convert.go, convert/chrome/session.go, convert/chrome/session_test.go</files>
  <action>
RED first. Add to `convert/chrome/session_test.go`:

- `TestBrowserOutputNilYieldsDrainingWriter` — `browserOutput(nil)` must return
  a non-nil `io.Writer`, and it must be `io.Discard`. Explain in the test's doc
  comment WHY: chromedp closes the Chrome output pipe when
  `combinedOutputWriter == nil` (allocate.go `readOutput`:
  `if forward == nil { rc.Close() }`), destroying every post-startup line at
  the source. `io.Discard` keeps the pipe open and drained.
- `TestBrowserOutputPassesCallerWriterThrough` — `browserOutput(w)` returns the
  same writer for a non-nil `w`.

Confirm both FAIL to compile (helper does not exist). Then GREEN:

1. `convert/convert.go` — add `io` to imports and a `BrowserLog io.Writer`
   field to `Options`, after `StartTimeout`. Doc comment states: when non-nil,
   Chrome's combined stdout+stderr is written here for the browser's lifetime;
   when nil, output is discarded but the pipe is still drained — and names why
   that distinction is not cosmetic (chromedp closes the pipe outright if no
   writer is set, which is why two failed production deploys produced zero
   browser diagnostics).
2. `convert/chrome/session.go` — add `io` to imports and an unexported
   `browserOutput(w io.Writer) io.Writer` returning `io.Discard` when `w == nil`
   and `w` otherwise. Doc comment carries the same reasoning, citing
   `allocate.go` `readOutput` and `allocate.go:253`.

Do not wire the helper into `New` yet — that is task 3.

# PATTERN: keyed struct literals everywhere, so a new Options field is source-compatible
# GOTCHA: the helper is unexported; session_test.go is `package chrome` so it can reach it
  </action>
  <verify>
`cd /Users/justin/dev/eden-press && go build ./... && go vet ./convert/...`
`go test ./convert/chrome/ -race -count=1 -run 'TestBrowserOutput' -v` — both pass.
`go test ./... -count=1` — no regressions.
  </verify>
  <done>
`convert.Options` has a documented `BrowserLog io.Writer` field; `browserOutput`
exists and is covered by two passing tests; the whole repo still builds and
every existing test still passes.
  </done>
</task>

<task type="auto" tdd="true">
  <name>RED: recording fake browser + argv and output-forwarding regression tests</name>
  <files>convert/chrome/session_test.go</files>
  <action>
Add a hand-written fixture and three tests that FAIL against the current `New`.

Fixture `fakeRecordingBrowser(t *testing.T) (execPath, argvPath string)`:
write an executable `/bin/sh` script into `t.TempDir()` that
(a) records its own argv one-per-line into `argvPath`,
(b) echoes a unique sentinel line to stderr,
(c) `exec sleep 120` so it never completes the DevTools handshake.

Mirror `fakeHangingBrowser`'s existing idioms exactly: `os.WriteFile` with 0o755
FOLLOWED BY an explicit `os.Chmod(path, 0o755)` (WriteFile's perm is subject to
umask and the allocator refuses a non-executable path), and `exec` on the final
sleep so cancellation kills the real process rather than orphaning a child.
Document in the fixture's comment that this fake is what makes these tests
runnable with no Chrome installed: `Discover` tier 1 (`BrowserPath`)
short-circuits the whole fallback chain.

Also add a small `type syncWriter struct { mu sync.Mutex; buf bytes.Buffer }`
with `Write` and a `String()` that both take the lock — chromedp writes from
`readOutput`'s goroutine while the test reads, and `-race` will flag a bare
`bytes.Buffer`.

Add a helper that calls `New(convert.Options{BrowserPath: fake, StartTimeout:
300 * time.Millisecond, BrowserLog: ...})` in a goroutine with a `select` on a
generous `returnDeadline` (~5s), exactly like the existing timeout test does, so
a regression surfaces as a named failure rather than a package timeout. `New` is
EXPECTED to return an error here; that is not the assertion.

Tests:

- `TestNewLaunchesBrowserWithDisableGPU` — after `New` returns, read `argvPath`
  (bounded poll up to ~2s if not yet present) and assert the argv contains the
  bare token `--disable-gpu`. Doc comment carries the measured evidence:
  production flags hung past 45s; with `--disable-gpu` allocation returned in
  808ms and a full raster render succeeded. Name the mechanism — the GPU/viz
  process crash-loops in a GPU-less container, so the renderer never obtains an
  execution context and never answers `Runtime.enable`, the third command
  chromedp sends inside `attachTarget`, whose only escape is `ctx.Done()`.
- `TestNewDoesNotDisableSoftwareRasterizer` — assert `--disable-software-
  rasterizer` is ABSENT from the recorded argv. Doc comment states the reason
  as a standing prohibition: that flag disables SwiftShader, which is what
  actually rasterizes the PDF/PNG output; `--disable-gpu` alone was verified
  sufficient and verified to still rasterize, so adding the second flag would
  trade a fixed hang for broken output.
- `TestNewForwardsBrowserOutputToBrowserLog` — pass a `*syncWriter` as
  `BrowserLog`, then assert the fake's sentinel line appears in it (bounded
  poll). Doc comment: this is the diagnostic that was missing during two failed
  production deploys, and setting it produced the root cause in a single run.

Run the suite and CONFIRM all three fail for the right reasons: the two argv
tests because `--disable-gpu` is not passed, the forwarding test because
`chromedp.CombinedOutput` is never set. Record the observed failure output.

# GOTCHA: `readOutput` only forwards complete lines (ReadBytes('\n')) — sentinel needs a newline
# GOTCHA: allocate.go sets cmd.Stderr = cmd.Stdout, so a stderr sentinel reaches the same pipe
# CRITICAL: these tests must never t.Skip — no Chrome is discoverable in this environment
  </action>
  <verify>
`cd /Users/justin/dev/eden-press && go vet ./convert/chrome/`
`go test ./convert/chrome/ -race -count=1 -run 'TestNewLaunchesBrowserWithDisableGPU|TestNewDoesNotDisableSoftwareRasterizer|TestNewForwardsBrowserOutputToBrowserLog' -v`
Expect: `TestNewLaunchesBrowserWithDisableGPU` FAIL,
`TestNewForwardsBrowserOutputToBrowserLog` FAIL,
`TestNewDoesNotDisableSoftwareRasterizer` PASS (the flag is genuinely absent
today — it is a guard against a future edit, and passing now is correct).
No test reports SKIP.
  </verify>
  <done>
Three new tests plus `fakeRecordingBrowser` and `syncWriter` exist; the two
tests covering behaviour that does not exist yet fail with messages naming the
missing flag / missing output; nothing skips; the failure output is recorded for
the GREEN step.
  </done>
  <recovery>
If the fake's argv file is empty, check the script is genuinely executable
(explicit `os.Chmod` after `os.WriteFile`) and that `BrowserPath` is reaching
`Discover` tier 1. If the sentinel never arrives, verify it ends in a newline —
`readOutput` buffers on `ReadBytes('\n')` and will not forward a partial line.
  </recovery>
</task>

<task type="auto">
  <name>GREEN: wire disable-gpu and CombinedOutput into New, correct the root-cause docs</name>
  <files>convert/chrome/session.go, convert/convert.go</files>
  <action>
1. In `New`'s `allocOpts` append block add, with a comment carrying the proven
   evidence (crash-looping GPU/viz process -> renderer never obtains an
   execution context -> `Runtime.enable` never answered -> `Browser.Execute`
   blocks on `ctx.Done()`; measured: hung past 45s vs 808ms + successful
   5754-byte PNG raster):

       chromedp.Flag("disable-gpu", true),

   Include an explicit standing note that `--disable-software-rasterizer` must
   NOT be added alongside it, and why (SwiftShader is what rasterizes output).

2. In the same block add:

       chromedp.CombinedOutput(browserOutput(opts.BrowserLog)),

   Comment it with the mechanism read from chromedp v0.16.0's source, not the
   pipe-buffer guess: `readOutput` closes the output pipe outright when no
   writer is set (`if forward == nil { rc.Close() }`), so everything Chrome
   says after the websocket URL is parsed is destroyed at the source — and the
   GPU crash-loop happens after that point, which is exactly why two failed
   production deploys produced zero browser diagnostics. With a writer set,
   `allocate.go:253` runs `io.Copy` on the allocator's WaitGroup for the
   browser's lifetime, so output is preserved and drained.

3. Correct the stale doc comments. The browser in the incident DID complete its
   DevTools handshake — the websocket URL was parsed and the connection
   established; it stalled later, at `Runtime.enable`. Update:
   - The comment block above the bounded `Run` (currently "a browser that
     started but never completed its handshake").
   - The timeout error's parenthetical, currently "the process may have started
     without completing the DevTools handshake". Broaden it to cover BOTH real
     modes — failed to complete the handshake, OR completed it and then stalled
     before the first tab became usable — and point the reader at the new
     diagnostic (`convert.Options.BrowserLog` captures Chrome's own output).
     MUST still contain the substring `did not become usable` and the executable
     path; `TestNewStartTimeoutBoundsHangingBrowser` asserts both.
   - `DefaultStartTimeout`'s comment, if it repeats the handshake-only framing.
   - `convert.Options.StartTimeout`'s doc comment in `convert/convert.go`, which
     carries the same stale "never completed its handshake" story.

   Do NOT change `DefaultStartTimeout`'s value or the goroutine/select
   mechanism. That bound is what turned an invisible hang into a diagnosable
   error and stays as a backstop.

# CRITICAL: append to the SAME allocOpts slice passed to NewExecAllocator
# CRITICAL: never pass opts.BrowserLog directly — always through browserOutput
# PATTERN: match the existing per-flag comment style in allocOpts (each flag says WHY)
  </action>
  <verify>
`cd /Users/justin/dev/eden-press && go build ./... && go vet ./...`
`go test ./convert/chrome/ -race -count=1 -v` — all three task-2 tests now PASS,
`TestNewStartTimeoutBoundsHangingBrowser` and `TestDefaultStartTimeoutIsPositive`
still PASS, nothing regressed.
`go test ./... -race -count=1` — whole repo green.
`git diff convert/chrome/session.go convert/convert.go` — read the doc comments
back and confirm no sentence still claims the handshake never completed as the
sole explanation.
  </verify>
  <done>
`--disable-gpu` is passed and `--disable-software-rasterizer` is not;
`Options.BrowserLog` reaches chromedp via `browserOutput`, with `io.Discard` on
the nil path; the previously-failing tests pass; the whole suite is green under
`-race`; and session.go/convert.go describe the proven GPU-crash-loop mechanism
rather than the original handshake guess.
  </done>
  <recovery>
If `TestNewStartTimeoutBoundsHangingBrowser` starts failing, the reworded error
message almost certainly dropped `did not become usable` or the `%q` executable
path — restore both substrings. If the whole suite slows noticeably, check that
`CombinedOutput` was not accidentally given a writer that blocks.
  </recovery>
</task>

## Validation gates

```bash
cd /Users/justin/dev/eden-press
go build ./...
go vet ./...
go test ./... -race -count=1
```

Live-Chrome tests (`TestSessionMultiTab`, `convert/pdf`, `convert/png`,
`convert/export_integration_test.go`) will SKIP here — no discoverable Chrome —
which is the pre-existing baseline, not a regression. The three new tests must
run and pass regardless.

## Success criteria

- [ ] `go build ./...`, `go vet ./...`, `go test ./... -race -count=1` all clean
- [ ] `--disable-gpu` present in the argv of a browser launched by `New`
- [ ] `--disable-software-rasterizer` provably absent, with a test guarding it
- [ ] `Options.BrowserLog` set -> Chrome output arrives; nil -> `io.Discard`,
      never a nil writer
- [ ] `DefaultStartTimeout` value and the StartTimeout mechanism unchanged
- [ ] Doc comments in `session.go` and `convert.go` describe the GPU crash-loop
      starving `Runtime.enable`, with the 45s-hang vs 808ms measurement
- [ ] No aodex, Docker, Kubernetes, gitops, Cilium, deploy, tag, or push changes

## Output

Atomic commits per task. Suggested subjects:

- `feat(convert): add Options.BrowserLog for Chrome's combined output`
- `test(convert/chrome): pin disable-gpu and browser-output forwarding`
- `fix(convert/chrome): disable GPU and forward Chrome output so launch cannot hang`
