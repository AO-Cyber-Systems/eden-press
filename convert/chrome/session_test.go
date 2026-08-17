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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/convert"
)

// TestSessionMultiTab is Test-list case 6: one Session/browser serves
// multiple tabs. Gated on Chrome presence -- t.Skip cleanly (never fail) if
// Discover cannot find a Chrome/Chromium executable anywhere, which is
// exactly the EXP-04 no-system-Chrome case 05-05 hardens in CI.
func TestSessionMultiTab(t *testing.T) {
	if _, _, err := Discover(DiscoverOptions{}); errors.Is(err, ErrChromeNotFound) {
		t.Skip("no Chrome discovered")
	}

	sess, err := New(convert.Options{})
	if err != nil {
		if errors.Is(err, ErrChromeNotFound) {
			t.Skip("no Chrome discovered")
		}
		t.Fatalf("New: %v", err)
	}
	defer sess.Close()

	tab1, cancel1 := sess.NewTab()
	defer cancel1()
	if err := chromedp.Run(tab1, chromedp.Navigate("about:blank")); err != nil {
		t.Fatalf("tab1 Navigate: %v", err)
	}

	tab2, cancel2 := sess.NewTab()
	defer cancel2()
	if err := chromedp.Run(tab2, chromedp.Navigate("about:blank")); err != nil {
		t.Fatalf("tab2 Navigate: %v", err)
	}

	c1 := chromedp.FromContext(tab1)
	c2 := chromedp.FromContext(tab2)
	if c1.Browser == nil || c2.Browser == nil {
		t.Fatal("expected both tabs to have a non-nil Browser")
	}
	if c1.Browser != c2.Browser {
		t.Error("expected both tabs to share the SAME Browser (one browser, many tabs)")
	}
	if c1.Target == c2.Target {
		t.Error("expected each tab to have its OWN Target (distinct tabs)")
	}
}

// fakeHangingBrowser writes an executable stand-in for Chrome that starts
// successfully and then never says anything: it prints no "DevTools listening
// on ..." line and does not exit until killed. It returns its path.
//
// The fake is load-bearing, not a convenience. A REAL Chrome cannot be asked
// to hang on demand -- the failure being pinned here is exactly the one that
// only appears when a browser process starts but never completes its DevTools
// handshake, which is not a state a healthy browser can be driven into. The
// fake reproduces that state precisely, and because it IS the browser as far
// as the allocator is concerned, the test needs no Chrome installed and must
// never skip.
//
// "exec sleep" rather than a bare "sleep" matters: exec REPLACES the shell, so
// the process chromedp launches is itself the sleeping process. Without it,
// cancellation would kill the shell while its sleep child survived as an
// orphan for the full duration.
func fakeHangingBrowser(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fake-chrome.sh")
	const script = "#!/bin/sh\n" +
		"# Deliberately silent: never prints \"DevTools listening on ...\".\n" +
		"exec sleep 120\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake browser: %v", err)
	}
	// Explicit chmod: WriteFile's perm argument is subject to the process
	// umask, and the allocator will refuse a non-executable path.
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod fake browser: %v", err)
	}
	return path
}

// TestNewStartTimeoutBoundsHangingBrowser is the regression test for the
// production failure: browser allocation had no bound of any kind, so a
// browser that started but never completed its DevTools handshake blocked New
// forever -- no error, no log line, nothing to alert on.
//
// The single most important assertion here is that New RETURNS AT ALL.
// Unbounded, this case never finishes.
func TestNewStartTimeoutBoundsHangingBrowser(t *testing.T) {
	fake := fakeHangingBrowser(t)

	const startTimeout = 300 * time.Millisecond
	// A generous multiple of startTimeout. The claim under test is "New
	// returns", not "New returns quickly", and the honest failure mode of a
	// regression is a hang -- so the select below converts that hang into a
	// named test failure instead of a ten-minute package timeout and a
	// goroutine dump.
	const returnDeadline = 5 * time.Second

	type result struct {
		sess *Session
		err  error
	}
	// Buffered: if New has regressed and returns late, its goroutine must
	// still be able to send without blocking forever.
	done := make(chan result, 1)

	start := time.Now()
	go func() {
		sess, err := New(convert.Options{BrowserPath: fake, StartTimeout: startTimeout})
		done <- result{sess: sess, err: err}
	}()

	var got result
	select {
	case got = <-done:
	case <-time.After(returnDeadline):
		t.Fatalf("New did not return within %s against a browser that never completes its "+
			"DevTools handshake -- browser allocation is unbounded again", returnDeadline)
	}
	elapsed := time.Since(start)
	t.Logf("New returned in %s (StartTimeout %s, deadline %s)", elapsed, startTimeout, returnDeadline)

	if got.err == nil {
		t.Fatal("New returned a nil error for a browser that never became usable")
	}
	if got.sess != nil {
		t.Error("New returned a non-nil Session alongside an error; the caller would leak a " +
			"browser it has no reason to Close")
	}
	if elapsed < startTimeout {
		t.Errorf("New returned in %s, sooner than its own %s bound -- the error came from "+
			"something other than the startup timeout, so this test is no longer "+
			"exercising the hang", elapsed, startTimeout)
	}

	msg := got.err.Error()
	if !strings.Contains(msg, "did not become usable") {
		t.Errorf("error %q does not identify the failure as a startup timeout (want it to "+
			"contain %q)", msg, "did not become usable")
	}
	// Discover resolves the executable through a three-tier fallback chain
	// (BrowserPath, then CHROME_PATH, then PATH auto-detection), so "a browser
	// failed to start" is not actionable on its own: whoever reads this error
	// cannot tell WHICH executable was launched, and therefore cannot tell
	// which tier to fix.
	if !strings.Contains(msg, fake) {
		t.Errorf("error %q does not name the executable it launched (want it to contain %q)",
			msg, fake)
	}
}

// TestDefaultStartTimeoutIsPositive pins the backstop itself. New treats a
// StartTimeout of zero or less as "unset" and substitutes DefaultStartTimeout,
// so setting that constant to zero would restore unbounded allocation while
// still compiling and still passing every other test in this package. This
// test is the only thing standing between that edit and production.
func TestDefaultStartTimeoutIsPositive(t *testing.T) {
	if DefaultStartTimeout <= 0 {
		t.Fatalf("DefaultStartTimeout = %s, want > 0: New reads a non-positive value as "+
			"\"unset\", which restores the unbounded browser allocation this package "+
			"exists to prevent", DefaultStartTimeout)
	}
}
