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
	"strings"
	"testing"
)

// TestDiscoverBrowserPath is Test-list case 2: an explicit BrowserPath wins
// regardless of what getenv/lookPath would otherwise report.
func TestDiscoverBrowserPath(t *testing.T) {
	opts := DiscoverOptions{
		BrowserPath: "/explicit/chrome",
		Getenv:      func(string) string { return "/env/chrome" },
		LookPath:    func(string) (string, error) { return "/found/chrome", nil },
	}

	execPath, source, err := Discover(opts)
	if err != nil {
		t.Fatalf("Discover returned unexpected error: %v", err)
	}
	if execPath != "/explicit/chrome" {
		t.Errorf("execPath = %q, want %q", execPath, "/explicit/chrome")
	}
	if source != "browser-path" {
		t.Errorf("source = %q, want %q", source, "browser-path")
	}
}

// TestDiscoverChromePathEnv is Test-list case 3: no BrowserPath, but
// CHROME_PATH set in the injected getenv wins over lookPath auto-detection.
func TestDiscoverChromePathEnv(t *testing.T) {
	opts := DiscoverOptions{
		Getenv: func(key string) string {
			if key == "CHROME_PATH" {
				return "/x/chrome"
			}
			return ""
		},
		LookPath: func(string) (string, error) { return "/should/not/be/used", nil },
	}

	execPath, source, err := Discover(opts)
	if err != nil {
		t.Fatalf("Discover returned unexpected error: %v", err)
	}
	if execPath != "/x/chrome" {
		t.Errorf("execPath = %q, want %q", execPath, "/x/chrome")
	}
	if source != "chrome-path-env" {
		t.Errorf("source = %q, want %q", source, "chrome-path-env")
	}
}

// TestDiscoverAutoDetect is Test-list case 4: neither BrowserPath nor
// CHROME_PATH set, but lookPath finds a known browser binary name --
// Discover delegates to chromedp's own auto-detection by returning an EMPTY
// execPath with source="auto".
func TestDiscoverAutoDetect(t *testing.T) {
	opts := DiscoverOptions{
		Getenv: func(string) string { return "" },
		LookPath: func(name string) (string, error) {
			if name == "google-chrome" {
				return "/usr/bin/google-chrome", nil
			}
			return "", errors.New("not found")
		},
	}

	execPath, source, err := Discover(opts)
	if err != nil {
		t.Fatalf("Discover returned unexpected error: %v", err)
	}
	if execPath != "" {
		t.Errorf("execPath = %q, want empty (delegate to chromedp auto-detect)", execPath)
	}
	if source != "auto" {
		t.Errorf("source = %q, want %q", source, "auto")
	}
}

// TestDiscoverNotFound is Test-list case 5: nothing found anywhere returns
// ErrChromeNotFound, whose message documents the pinned-download remedy
// (chromedp/headless-shell + Chrome-for-Testing).
func TestDiscoverNotFound(t *testing.T) {
	opts := DiscoverOptions{
		Getenv:   func(string) string { return "" },
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	}

	execPath, source, err := Discover(opts)
	if execPath != "" {
		t.Errorf("execPath = %q, want empty", execPath)
	}
	if source != "" {
		t.Errorf("source = %q, want empty", source)
	}
	if !errors.Is(err, ErrChromeNotFound) {
		t.Fatalf("err = %v, want ErrChromeNotFound", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "headless-shell") {
		t.Errorf("error message %q does not mention chromedp/headless-shell pinned-download tier", msg)
	}
	if !strings.Contains(msg, "Chrome-for-Testing") && !strings.Contains(msg, "chrome-for-testing") {
		t.Errorf("error message %q does not mention Chrome-for-Testing", msg)
	}
}

// TestDiscoverDefaults verifies the zero-value DiscoverOptions (no injected
// Getenv/LookPath) falls back to the real os.Getenv/exec.LookPath rather
// than panicking on a nil func field.
func TestDiscoverDefaults(t *testing.T) {
	// With a zero-value DiscoverOptions, production defaults apply. We can't
	// assert a specific outcome (depends on the host), but Discover must not
	// panic and must return a result consistent with one of the four tiers.
	execPath, source, err := Discover(DiscoverOptions{})
	if err != nil && !errors.Is(err, ErrChromeNotFound) {
		t.Fatalf("unexpected error type: %v", err)
	}
	if err == nil && source == "" {
		t.Errorf("expected a non-empty source when no error is returned, got execPath=%q source=%q", execPath, source)
	}
}
