// Copyright (c) 2026 AO Cyber Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// SPDX-License-Identifier: MIT

package chrome

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/convert"
)

// Session is the one-browser-many-tabs pool primitive every convert/
// exporter (convert/pdf, convert/png) drives its rendering through. New
// allocates exactly ONE Chrome process; NewTab hands out additional tabs on
// that SAME browser -- never a fresh process per call.
type Session struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc

	// rootCtx is an internal, already-Run anchor tab context. It exists
	// solely so NewTab's chromedp.NewContext(rootCtx) calls inherit a
	// non-nil Browser at creation time -- see the package-level comment on
	// New for why this indirection is required.
	rootCtx    context.Context
	rootCancel context.CancelFunc

	userDataDir string
}

// DefaultStartTimeout bounds browser allocation when Options.StartTimeout is
// zero. Generous on purpose: a cold Chrome in a CPU-throttled container can
// take well over ten seconds to complete its DevTools handshake, and a bound
// that trips on a slow-but-healthy start would trade a rare hang for a common
// false failure. This is a backstop against never returning, not a latency SLO.
const DefaultStartTimeout = 90 * time.Second

// browserOutput resolves the caller's optional convert.Options.BrowserLog into
// the writer handed to chromedp.CombinedOutput. It returns w unchanged when the
// caller set one, and io.Discard when they did not.
//
// Never pass a caller's nil writer straight through to chromedp -- routing it
// through here is what keeps Chrome's output pipe alive. chromedp's
// allocate.go readOutput scans Chrome's output for the "DevTools listening on"
// line and then branches on the forward writer: with a nil writer it calls
// Close on the pipe, on the reasoning that the process's output is no longer
// needed. Everything Chrome emits after startup is destroyed at the source.
//
// io.Discard takes the other branch instead. allocate.go:253 then runs io.Copy
// on the allocator's WaitGroup for the browser's whole lifetime, so the pipe
// stays open and drained. Behaviour for callers who never set BrowserLog is
// unchanged -- the bytes go nowhere either way -- but a caller who DOES set it
// gets a live diagnostic rather than a pipe chromedp already closed.
//
// This is not hypothetical: it is why two failed production deploys produced no
// browser output at all while Chrome's GPU process crash-looped continuously.
// The crash messages were emitted after the websocket URL was parsed, which is
// precisely when the pipe had already been closed.
func browserOutput(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// New builds ONE chromedp ExecAllocator/browser with the CI-hardening and
// determinism launch flags baked in as DEFAULTS (not opt-ins), resolving the
// executable via Discover.
//
// Implementation note (why a root anchor tab, not just the allocator ctx):
// chromedp.NewContext(parent) copies parent's *Browser field at CALL time --
// it does not lazily re-read it later. The allocator context returned by
// chromedp.NewExecAllocator is never itself Run (doing so directly is a
// chromedp error), so its attached Context.Browser field is permanently
// nil. If NewTab called chromedp.NewContext(s.allocCtx) directly on every
// invocation, each call would independently see a nil Browser and allocate
// its OWN Chrome process -- exactly the per-render-process anti-pattern this
// Session exists to avoid. Instead, New creates one internal tab context
// (rootCtx) and immediately Runs it (zero actions) so the browser is
// allocated eagerly and rootCtx's Context.Browser is populated once; every
// subsequent NewTab derives from that SAME already-populated rootCtx, so it
// correctly inherits the live Browser and only ever creates a new Target
// (tab) -- exactly chromedp's own documented multi-tab pattern
// (ExampleNewContext_manyTabs in the chromedp package itself).
func New(opts convert.Options) (*Session, error) {
	execPath, _, err := Discover(DiscoverOptions{BrowserPath: opts.BrowserPath})
	if err != nil {
		return nil, err
	}

	userDataDir, err := os.MkdirTemp("", "eden-press-chrome-*")
	if err != nil {
		return nil, fmt.Errorf("convert/chrome: creating unique user-data-dir: %w", err)
	}

	allocOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocOpts = append(allocOpts,
		// NoSandbox: guarded-for-container posture -- CI/container
		// execution commonly runs as root with no user-namespace sandbox
		// available, so the sandbox must be disabled for Chrome to launch
		// at all (05-05 hardens the surrounding non-root/CI posture).
		chromedp.NoSandbox,
		// Unique per-run profile dir -- avoids cross-run contention if
		// multiple Sessions exist concurrently (Pitfall 11).
		chromedp.UserDataDir(userDataDir),
		// Container /dev/shm is often too small; disabling shared-memory
		// usage avoids BUS_ADRERR crashes (Pitfall 11). Also already present
		// in chromedp.DefaultExecAllocatorOptions; set again explicitly so
		// the determinism recipe is self-documenting and order-independent.
		chromedp.Flag("disable-dev-shm-usage", true),
		// Determinism: fixed device-scale-factor so screenshots/PDFs render
		// pixel-identically across hosts with different display densities.
		chromedp.Flag("force-device-scale-factor", "1"),
		// Determinism: fixed locale, independent of the host's system locale.
		chromedp.Flag("lang", "en-US"),
		// Determinism: fixed timezone for the Chrome process's own clock,
		// independent of the host machine's TZ.
		chromedp.Env("TZ=UTC"),
	)
	if execPath != "" {
		// Only pin an explicit executable when Discover resolved one, e.g.
		// tier 1 (BrowserPath) or tier 2 (CHROME_PATH). An empty execPath
		// (tier 3, "auto") deliberately omits ExecPath so chromedp's own
		// ExecAllocator performs its own auto-detection.
		allocOpts = append(allocOpts, chromedp.ExecPath(execPath))
	}

	// context.Background(), NOT a deadline: this context owns the BROWSER'S
	// LIFETIME, so a timeout here would kill a healthy long-lived Session
	// mid-render. Startup is bounded separately, below.
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)

	rootCtx, rootCancel := chromedp.NewContext(allocCtx)

	// Bound ALLOCATION -- the process launch plus the DevTools handshake.
	//
	// This Run previously had no bound of any kind, so a browser that started
	// but never completed its handshake blocked the caller forever, with no
	// error and nothing logged. That is unobservable by construction, and it
	// is exactly how an export sidecar died in production: the pod ran, the
	// port listened, and its readiness endpoint simply never answered.
	//
	// Done in a goroutine rather than context.WithTimeout(rootCtx, ...) on
	// purpose. rootCtx carries the chromedp *Context that Run populates and
	// that every later NewTab inherits; wrapping it in a deadline would make
	// the SUCCESS path's browser reachable through a context that expires,
	// which is the bug this fixes wearing a different hat.
	//
	// The goroutine can outlive this function on the timeout path -- that is
	// deliberate. It is parked on a Run that may never return, so waiting for
	// it would reintroduce the hang. Cancelling the contexts below is what
	// stops the browser; the goroutine then observes that and exits.
	startTimeout := opts.StartTimeout
	if startTimeout <= 0 {
		startTimeout = DefaultStartTimeout
	}
	allocated := make(chan error, 1) // buffered: never block the sender
	go func() { allocated <- chromedp.Run(rootCtx) }()

	select {
	case err := <-allocated:
		if err != nil {
			rootCancel()
			allocCancel()
			_ = os.RemoveAll(userDataDir)
			return nil, fmt.Errorf("convert/chrome: allocating browser: %w", err)
		}
	case <-time.After(startTimeout):
		rootCancel()
		allocCancel()
		_ = os.RemoveAll(userDataDir)
		return nil, fmt.Errorf(
			"convert/chrome: browser did not become usable within %s (launched %q; "+
				"the process may have started without completing the DevTools handshake)",
			startTimeout, execPath)
	}

	return &Session{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
		userDataDir: userDataDir,
	}, nil
}

// NewTab creates a child chromedp.NewContext -- a new TAB on the SAME
// browser, never a new process. The returned context is not itself Run;
// the caller drives it via chromedp.Run(ctx, actions...), whose first call
// creates the actual Target (tab) on the shared browser.
func (s *Session) NewTab() (context.Context, context.CancelFunc) {
	return chromedp.NewContext(s.rootCtx)
}

// Close tears the browser down: cancels the root tab, then the allocator
// (which stops the Chrome process and waits for it to exit), then removes
// the unique user-data-dir created for this run.
func (s *Session) Close() {
	s.rootCancel()
	s.allocCancel()
	if s.userDataDir != "" {
		_ = os.RemoveAll(s.userDataDir)
	}
}
