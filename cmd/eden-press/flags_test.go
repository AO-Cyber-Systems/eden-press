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
	"testing"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

// newTestConvertCmd builds a bare cobra.Command carrying the full
// persistent + convert-mode flag surface, mirroring what newRootCmd() +
// newConvertCmd() register -- enough for buildOptions tests to set flags
// and read them back through cfg, without constructing the whole
// subcommand tree.
func newTestConvertCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	registerPersistentFlags(cmd)
	registerConvertFlags(cmd)
	return cmd
}

// resetCfg replaces the package koanf instance with a fresh one so each
// test starts from a clean slate -- cfg is a package-level singleton shared
// with production code (root.go's PersistentPreRunE).
func resetCfg() {
	cfg = koanf.New(".")
}

// TestBuildOptionsMapsSetFlags is test-list case 1: set flags flow through
// applyConfig (posflag) into cfg, and buildOptions maps them verbatim onto
// press.Options.
func TestBuildOptionsMapsSetFlags(t *testing.T) {
	resetCfg()
	cmd := newTestConvertCmd()

	// ParseFlags (not Flags().Set) so cobra merges the persistent flagset
	// (registerPersistentFlags registered "theme"/"math"/etc. as PERSISTENT
	// flags) into cmd.Flags() first -- exactly what happens during a real
	// Execute(), and what posflag.Provider(cmd.Flags(), ...) then reads.
	if err := cmd.ParseFlags([]string{
		"--theme", "gaia",
		"--math", "off",
		"--no-highlight",
		"--highlight-style", "dracula",
	}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: %v", err)
	}

	if opts.Theme != "gaia" {
		t.Errorf("Theme = %q, want %q", opts.Theme, "gaia")
	}
	if opts.MathMode != "off" {
		t.Errorf("MathMode = %q, want %q", opts.MathMode, "off")
	}
	if !opts.NoHighlight {
		t.Errorf("NoHighlight = false, want true")
	}
	if opts.HighlightStyle != "dracula" {
		t.Errorf("HighlightStyle = %q, want %q", opts.HighlightStyle, "dracula")
	}
}

// TestBuildOptionsLeavesUnsetFlagsZero is test-list case 2: unset flags
// must NOT be pre-resolved to press.Render's own fallbacks -- the CLI
// passes empty/zero values through verbatim.
func TestBuildOptionsLeavesUnsetFlagsZero(t *testing.T) {
	resetCfg()
	cmd := newTestConvertCmd()

	// ParseFlags with no arguments still merges the persistent flagset into
	// cmd.Flags() (mirroring a real Execute()), so posflag genuinely visits
	// "theme"/"math"/etc. as UNCHANGED defaults rather than skipping them
	// because they were never merged in.
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: %v", err)
	}

	if opts.Theme != "" {
		t.Errorf(`Theme = %q, want zero value ""`, opts.Theme)
	}
	if opts.Profile != "" {
		t.Errorf(`Profile = %q, want zero value ""`, opts.Profile)
	}
	if opts.MathMode != "" {
		t.Errorf(`MathMode = %q, want zero value ""`, opts.MathMode)
	}
	if opts.NoHighlight {
		t.Errorf("NoHighlight = true, want zero value false")
	}
	if opts.HighlightStyle != "" {
		t.Errorf(`HighlightStyle = %q, want zero value ""`, opts.HighlightStyle)
	}
	if opts.InlineSVG {
		t.Errorf("InlineSVG = true, want zero value false")
	}
}

// TestPosflagInstanceGuard is test-list case 3, the Pitfall-5 guard: an
// unset/unchanged flag must NOT overwrite a value already present in cfg,
// as if a config file or env provider (layered in ahead of posflag by
// 04-04) had already set it. This is only true because loadConfigSources
// passes the LIVE koanf instance as posflag.Provider's third argument.
func TestPosflagInstanceGuard(t *testing.T) {
	resetCfg()
	if err := cfg.Set("theme", "pre-seeded"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}

	cmd := newTestConvertCmd() // --theme left unset/unchanged
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	if err := loadConfigSources(cfg, cmd); err != nil {
		t.Fatalf("loadConfigSources: %v", err)
	}

	if got := cfg.String("theme"); got != "pre-seeded" {
		t.Errorf(`cfg.String("theme") = %q, want %q (unset flag must not stomp a pre-seeded value)`, got, "pre-seeded")
	}
}
