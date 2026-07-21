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
	"testing"
)

// writeTempConfig writes a config file with the given content into dir/name
// and returns its absolute path.
func writeTempConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", p, err)
	}
	return p
}

// chdir switches the process cwd to dir for the duration of the test,
// restoring the original cwd on cleanup -- needed because discoverConfigPath
// searches the CURRENT WORKING DIRECTORY for project-local ".marprc.*".
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%s): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restore Chdir(%s): %v", orig, err)
		}
	})
}

// TestConfigFileToOptions is test-list case 1: a temp .marprc.yaml supplies
// theme/math with no flag set, and buildOptions maps the resolved cfg values
// straight onto press.Options.
func TestConfigFileToOptions(t *testing.T) {
	resetCfg()
	dir := t.TempDir()
	writeTempConfig(t, dir, ".marprc.yaml", "theme: gaia\nmath: \"off\"\n")
	chdir(t, dir)

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	if got := cfg.String("theme"); got != "gaia" {
		t.Errorf(`cfg.String("theme") = %q, want %q`, got, "gaia")
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: %v", err)
	}
	if opts.Theme != "gaia" {
		t.Errorf("opts.Theme = %q, want %q", opts.Theme, "gaia")
	}
	if opts.MathMode != "off" {
		t.Errorf("opts.MathMode = %q, want %q", opts.MathMode, "off")
	}
}

// TestPrecedenceFlagOverFile is test-list case 2: a flag set on the command
// line overrides the same key present in the config file.
func TestPrecedenceFlagOverFile(t *testing.T) {
	resetCfg()
	dir := t.TempDir()
	writeTempConfig(t, dir, ".marprc.yaml", "theme: gaia\n")
	chdir(t, dir)

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--theme", "uncover"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		t.Fatalf("buildOptions: %v", err)
	}
	if opts.Theme != "uncover" {
		t.Errorf("opts.Theme = %q, want %q (flag must win over file)", opts.Theme, "uncover")
	}
}

// TestPrecedenceEnvOverFileFlagOverEnv is test-list case 3: env overrides
// file, and a flag overrides env -- proving the full three-way ordering.
func TestPrecedenceEnvOverFileFlagOverEnv(t *testing.T) {
	resetCfg()
	dir := t.TempDir()
	writeTempConfig(t, dir, ".marprc.yaml", "theme: gaia\n")
	chdir(t, dir)
	t.Setenv("EDEN_PRESS_THEME", "default")

	cmd := newTestConvertCmd()
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
	if opts.Theme != "default" {
		t.Errorf("opts.Theme = %q, want %q (env must win over file)", opts.Theme, "default")
	}

	resetCfg()
	cmd2 := newTestConvertCmd()
	if err := cmd2.ParseFlags([]string{"--theme", "uncover"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd2); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	opts2, err := buildOptions(cmd2)
	if err != nil {
		t.Fatalf("buildOptions: %v", err)
	}
	if opts2.Theme != "uncover" {
		t.Errorf("opts2.Theme = %q, want %q (flag must win over env)", opts2.Theme, "uncover")
	}
}

// TestExtRouting is test-list case 4: .json and .toml configs each load
// their theme via the correct parser, and an unsupported extension errors.
func TestExtRouting(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		resetCfg()
		dir := t.TempDir()
		writeTempConfig(t, dir, ".marprc.json", `{"theme": "uncover"}`)
		chdir(t, dir)

		cmd := newTestConvertCmd()
		if err := cmd.ParseFlags(nil); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if err := applyConfig(cmd); err != nil {
			t.Fatalf("applyConfig: %v", err)
		}
		if got := cfg.String("theme"); got != "uncover" {
			t.Errorf(`cfg.String("theme") = %q, want %q`, got, "uncover")
		}
	})

	t.Run("toml", func(t *testing.T) {
		resetCfg()
		dir := t.TempDir()
		writeTempConfig(t, dir, ".marprc.toml", `theme = "gaia"`+"\n")
		chdir(t, dir)

		cmd := newTestConvertCmd()
		if err := cmd.ParseFlags(nil); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if err := applyConfig(cmd); err != nil {
			t.Fatalf("applyConfig: %v", err)
		}
		if got := cfg.String("theme"); got != "gaia" {
			t.Errorf(`cfg.String("theme") = %q, want %q`, got, "gaia")
		}
	})

	t.Run("unsupported extension", func(t *testing.T) {
		resetCfg()
		dir := t.TempDir()
		p := writeTempConfig(t, dir, "custom.ini", "theme=gaia\n")

		cmd := newTestConvertCmd()
		if err := cmd.ParseFlags([]string{"--config", p}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if err := applyConfig(cmd); err == nil {
			t.Fatalf("applyConfig: want error for unsupported extension %q, got nil", filepath.Ext(p))
		}
	})
}

// TestConfigFlagOverridesDiscovery is test-list case 5: --config is used
// even when a discoverable .marprc.yml also exists in cwd.
func TestConfigFlagOverridesDiscovery(t *testing.T) {
	resetCfg()
	dir := t.TempDir()
	writeTempConfig(t, dir, ".marprc.yml", "theme: gaia\n")
	customPath := writeTempConfig(t, dir, "custom.toml", `theme = "uncover"`+"\n")
	chdir(t, dir)

	cmd := newTestConvertCmd()
	if err := cmd.ParseFlags([]string{"--config", customPath}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := applyConfig(cmd); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if got := cfg.String("theme"); got != "uncover" {
		t.Errorf(`cfg.String("theme") = %q, want %q (--config must override discovery)`, got, "uncover")
	}
}

// TestPitfall5GuardThroughFullChain is test-list case 6: a config-file value
// survives loadConfigSources' full chain (file -> env -> posflag) when its
// corresponding flag is left unset -- the regression check that posflag's
// instance-guard (Pitfall 5) still holds once file+env are layered in ahead
// of it.
func TestPitfall5GuardThroughFullChain(t *testing.T) {
	resetCfg()
	dir := t.TempDir()
	writeTempConfig(t, dir, ".marprc.yaml", "theme: gaia\n")
	chdir(t, dir)

	cmd := newTestConvertCmd() // --theme left unset
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	if err := loadConfigSources(cfg, cmd); err != nil {
		t.Fatalf("loadConfigSources: %v", err)
	}
	if got := cfg.String("theme"); got != "gaia" {
		t.Errorf(`cfg.String("theme") = %q, want %q (unset flag must not stomp file value)`, got, "gaia")
	}
}
