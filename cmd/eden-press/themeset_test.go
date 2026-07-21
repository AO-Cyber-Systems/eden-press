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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// writeTempCSS writes content to a uniquely-named file inside t.TempDir() and
// returns its path -- the on-disk fixture themeCSS reads back via os.ReadFile.
func writeTempCSS(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", p, err)
	}
	return p
}

// brandCSS/brandCSS2 are minimal, self-naming custom theme blocks (each names
// itself via its own leading `/* @theme name */` comment, theme.Load's
// requirement) carrying one rule whose color survives Pack's scoping pass --
// mirroring press/themecss_test.go's brandxCSS fixture.
const (
	brandCSS  = "/* @theme brand */\nsection { color: #d4a853; }"
	brandCSS2 = "/* @theme brand2 */\nsection { color: #111111; }"
)

// TestThemeCSSMultiFile is Test-list case 1: themeCSS over two --theme-set
// paths returns both files' CSS text, in the order the flag was repeated.
func TestThemeCSSMultiFile(t *testing.T) {
	resetCfg()
	p1 := writeTempCSS(t, "a.css", brandCSS)
	p2 := writeTempCSS(t, "b.css", brandCSS2)

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--theme-set", p1, "--theme-set", p2}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	got, err := themeCSS(cmd)
	if err != nil {
		t.Fatalf("themeCSS: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("themeCSS returned %d entries, want 2 (got=%v)", len(got), got)
	}
	if got[0] != brandCSS {
		t.Errorf("got[0] = %q, want %q", got[0], brandCSS)
	}
	if got[1] != brandCSS2 {
		t.Errorf("got[1] = %q, want %q", got[1], brandCSS2)
	}
}

// TestThemeSetEndToEnd is Test-list case 2: a rootCmd-equivalent flag set of
// "--theme-set brand.css --theme brand" on a temp deck renders (via
// buildOptions + a direct press.Render call, since runConvert is still
// 04-03's stub) an Output.CSS containing the scoped "brand" rule -- proving
// files -> themeCSS -> Options.ThemeCSS -> press.Render round-trips.
func TestThemeSetEndToEnd(t *testing.T) {
	resetCfg()
	p := writeTempCSS(t, "brand.css", brandCSS)

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--theme-set", p, "--theme", "brand"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: unexpected error: %v", err)
	}
	if opts.Theme != "brand" {
		t.Errorf("opts.Theme = %q, want %q", opts.Theme, "brand")
	}
	if len(opts.ThemeCSS) != 1 || opts.ThemeCSS[0] != brandCSS {
		t.Fatalf("opts.ThemeCSS = %v, want [%q]", opts.ThemeCSS, brandCSS)
	}

	out, err := press.Render("# Hi\n", opts)
	if err != nil {
		t.Fatalf("press.Render with custom theme-set: unexpected error: %v", err)
	}
	if !strings.Contains(out.CSS, "#d4a853") {
		t.Errorf("Output.CSS does not contain the custom theme's scoped rule (#d4a853): CSS=%d bytes", len(out.CSS))
	}
}

// TestThemePassThroughBundled is Test-list case 3: "--theme gaia" (a bundled
// name, no --theme-set) still selects gaia -- verbatim pass-through, no
// regression from this TRD's changes.
func TestThemePassThroughBundled(t *testing.T) {
	resetCfg()
	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--theme", "gaia"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: unexpected error: %v", err)
	}
	if opts.Theme != "gaia" {
		t.Errorf("opts.Theme = %q, want %q", opts.Theme, "gaia")
	}
	if opts.ThemeCSS != nil {
		t.Errorf("opts.ThemeCSS = %v, want nil (no --theme-set given)", opts.ThemeCSS)
	}

	out, err := press.Render("# Hi\n", opts)
	if err != nil {
		t.Fatalf("press.Render with --theme gaia: unexpected error: %v", err)
	}
	if out.CSS == "" {
		t.Error("Output.CSS is empty for the bundled gaia theme")
	}
}

// TestThemeCSSMissingFile is Test-list case 4: an unreadable --theme-set path
// returns a clear "theme-set: read ..." error (never a bare os.PathError).
func TestThemeCSSMissingFile(t *testing.T) {
	resetCfg()
	missing := filepath.Join(t.TempDir(), "nonexistent.css")

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--theme-set", missing}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	_, err := themeCSS(cmd)
	if err == nil {
		t.Fatal("themeCSS with a missing file returned no error")
	}
	if !strings.Contains(err.Error(), "theme-set: read") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "theme-set: read")
	}
}

// TestThemeSetMalformedErrorsAtRender is Test-list case 5: a --theme-set file
// lacking a leading "/* @theme */" comment is read successfully by themeCSS
// (the CLI does not parse/validate CSS) but surfaces press.Render's clear
// "load custom theme CSS" error at render time -- never a silent ignore.
func TestThemeSetMalformedErrorsAtRender(t *testing.T) {
	resetCfg()
	malformed := "section { color: red; }" // no leading @theme comment
	p := writeTempCSS(t, "malformed.css", malformed)

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--theme-set", p}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: unexpected error: %v", err)
	}

	_, err = press.Render("# Hi\n", opts)
	if err == nil {
		t.Fatal("press.Render with a malformed theme-set CSS block returned no error")
	}
	if !strings.Contains(err.Error(), "load custom theme CSS") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "load custom theme CSS")
	}
}
