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

// Package chrome implements the EXP-04 Chrome-discovery fallback chain
// (Discover), the one-browser-many-tabs Session pool primitive that every
// convert/ exporter (convert/pdf, convert/png) drives its rendering through,
// and the SHARED determinism substrate (determinism.go, load.go, fonts.go)
// those same exporters fold in exactly once: ComposeCSS/PageCSSInches (pure
// CSS transforms), ApplyDeterminism (the ordered CDP recipe), LoadHTML (the
// SetDocumentContent loader), and the embedded STIX Two Math font (EXP-04
// font provisioning).
package chrome

import (
	"errors"
	"os"
	"os/exec"
)

// defaultGetenv is the production Getenv default: the real os.Getenv.
func defaultGetenv(key string) string {
	return os.Getenv(key)
}

// defaultLookPath is the production LookPath default: the real
// exec.LookPath.
func defaultLookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// candidateBinaries is the ordered list of known Chrome/Chromium executable
// names probed by the tier-3 "auto" precedence step. lookPath is tried
// against each in order; the FIRST one found is enough to conclude "a
// browser exists on PATH" -- Discover still hands back an empty execPath so
// chromedp's own ExecAllocator performs its own (more exhaustive,
// platform-aware) auto-detection rather than eden-press re-implementing it.
var candidateBinaries = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium-browser",
	"chromium",
	"google-chrome-beta",
	"google-chrome-unstable",
	"/usr/bin/google-chrome",
}

// ErrChromeNotFound is returned by Discover when no Chrome/Chromium
// executable can be resolved via any tier of the EXP-04 fallback chain. Its
// message documents the remaining, deliberately-not-automated remedy: pin a
// chromedp/headless-shell image, or fetch a known-good build via
// Chrome-for-Testing.
var ErrChromeNotFound = errors.New(
	"convert/chrome: no Chrome/Chromium executable found via BrowserPath, " +
		"CHROME_PATH, or PATH auto-detection; pin a chromedp/headless-shell " +
		"container image, or download a known-good Chrome-for-Testing build " +
		"from https://googlechromelabs.github.io/chrome-for-testing/ and set " +
		"CHROME_PATH (or convert.Options.BrowserPath) to its executable path",
)

// DiscoverOptions configures Discover. Getenv and LookPath are injected so
// Discover is pure and unit-testable with hand-built fakes -- no real Chrome
// or live environment required. Their zero value (nil) defaults to the real
// os.Getenv / exec.LookPath, so production callers can leave them unset.
type DiscoverOptions struct {
	// BrowserPath is an explicit Chrome/Chromium executable override -- the
	// highest-precedence tier. Mirrors convert.Options.BrowserPath.
	BrowserPath string

	// Getenv defaults to os.Getenv when nil. Injected for testing.
	Getenv func(string) string

	// LookPath defaults to exec.LookPath when nil. Injected for testing.
	LookPath func(string) (string, error)
}

// Discover resolves a Chrome/Chromium executable via the EXP-04 fallback
// chain, in strict precedence, stopping at the first tier that hits:
//
//  1. opts.BrowserPath, if non-empty -- source="browser-path".
//  2. The CHROME_PATH environment variable, if set -- source="chrome-path-env".
//     CHROME_PATH is NOT a chromedp built-in; it is eden-press-owned glue
//     matching the Lighthouse/marp-cli convention, read manually here via
//     the injected Getenv.
//  3. A known Chrome/Chromium binary name found on PATH via the injected
//     LookPath -- source="auto". This tier deliberately returns an EMPTY
//     execPath: chromedp's own ExecAllocator performs its own auto-detection
//     when no ExecPath option is supplied, so Discover just confirms
//     "something is there" and steps aside.
//  4. Nothing found anywhere -- returns ErrChromeNotFound, whose message
//     documents the pinned-download remedy (chromedp/headless-shell or
//     Chrome-for-Testing). This tier is deliberately DOCUMENTED, not
//     automated, in v1 (EXP-04).
func Discover(opts DiscoverOptions) (execPath, source string, err error) {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = defaultGetenv
	}
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = defaultLookPath
	}

	// Tier 1: explicit override.
	if opts.BrowserPath != "" {
		return opts.BrowserPath, "browser-path", nil
	}

	// Tier 2: CHROME_PATH env var (eden-press glue, not a chromedp feature).
	if p := getenv("CHROME_PATH"); p != "" {
		return p, "chrome-path-env", nil
	}

	// Tier 3: PATH auto-detection -- confirm presence, delegate resolution.
	for _, name := range candidateBinaries {
		if _, lookErr := lookPath(name); lookErr == nil {
			return "", "auto", nil
		}
	}

	// Tier 4: nothing found -- documented pinned-download remedy.
	return "", "", ErrChromeNotFound
}
