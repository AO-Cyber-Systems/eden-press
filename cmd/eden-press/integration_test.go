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

// integration_test.go is the objective's capstone proof (04-08-TRD.md): one
// pass exercising convert (file/stdin/-o), theme + config precedence,
// --theme-set custom themes, and the serve mux together -- proving 04-03
// (convert), 04-04 (config), 04-05 (theme), and 04-07 (serve) compose
// through the SAME buildOptions/press.Render/assembleHTML chain, rather
// than each mode silently drifting its own copy. Every helper reused below
// (resetCfg, newTestConvertCmd, writeTempConfig/chdir, writeTempCSS/
// brandCSS, prepServeCmd) already lives in this package's other _test.go
// files (config_test.go/themeset_test.go/serve_test.go) -- reused verbatim,
// never re-implemented, so this file proves composition instead of
// duplicating per-mode unit coverage.
package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/cmd/eden-press/reload"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

const integrationDeck = "# Hello Integration\n\nComposed content.\n"

// assertStandaloneZeroJS is the shared assertion every convert leg below
// applies: a complete standalone document containing want, with NO
// <script> tag anywhere (the zero-JS baseline every mode shares).
func assertStandaloneZeroJS(t *testing.T, doc, want string) {
	t.Helper()
	if !strings.Contains(doc, "<!doctype html>") {
		t.Errorf("doc missing <!doctype html>: %q", doc)
	}
	if !strings.Contains(doc, want) {
		t.Errorf("doc missing %q: %q", want, doc)
	}
	if strings.Contains(doc, "<script") {
		t.Errorf("doc contains <script>, want zero-JS: %q", doc)
	}
}

// TestIntegrationConvertModes is test-list case 3: file->stdout,
// stdin->stdout, and -o all produce a standalone zero-JS doc through the
// SAME root command tree -- proving convert's three input/output pairings
// compose without interfering with each other.
func TestIntegrationConvertModes(t *testing.T) {
	t.Run("file to stdout", func(t *testing.T) {
		resetCfg()
		dir := t.TempDir()
		path := filepath.Join(dir, "deck.md")
		if err := os.WriteFile(path, []byte(integrationDeck), 0o644); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}

		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{path})

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		assertStandaloneZeroJS(t, out.String(), "Hello Integration")
	})

	t.Run("stdin to stdout", func(t *testing.T) {
		resetCfg()
		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetIn(strings.NewReader(integrationDeck))
		root.SetArgs([]string{"-"})

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		assertStandaloneZeroJS(t, out.String(), "Hello Integration")
	})

	t.Run("file to -o", func(t *testing.T) {
		resetCfg()
		dir := t.TempDir()
		inPath := filepath.Join(dir, "deck.md")
		if err := os.WriteFile(inPath, []byte(integrationDeck), 0o644); err != nil {
			t.Fatalf("os.WriteFile: %v", err)
		}
		outPath := filepath.Join(dir, "out.html")

		root := newRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs([]string{"convert", inPath, "--output", outPath})

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if out.String() != "" {
			t.Errorf("stdout = %q, want empty (output should go to the file)", out.String())
		}

		written, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("os.ReadFile(%q): %v", outPath, err)
		}
		assertStandaloneZeroJS(t, string(written), "Hello Integration")
	})
}

// TestIntegrationThemeAndConfigPrecedence is test-list case 4: a
// .marprc.yaml (theme: gaia) applies with no --theme flag set; --theme
// uncover then overrides it; and a --theme-set custom theme + matching
// --theme applies its own scoped CSS -- proving 04-04 (config) and 04-05
// (theme) compose through the SAME buildOptions/press.Render chain convert
// uses, rather than each mode resolving options its own way.
func TestIntegrationThemeAndConfigPrecedence(t *testing.T) {
	t.Run("config file theme applies", func(t *testing.T) {
		resetCfg()
		dir := t.TempDir()
		writeTempConfig(t, dir, ".marprc.yaml", "theme: gaia\n")
		chdir(t, dir)

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
		if opts.Theme != "gaia" {
			t.Errorf("opts.Theme = %q, want %q", opts.Theme, "gaia")
		}
	})

	t.Run("flag overrides config file", func(t *testing.T) {
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
			t.Errorf("opts.Theme = %q, want %q (flag must win over config file)", opts.Theme, "uncover")
		}
	})

	t.Run("theme-set custom theme applies its scoped CSS", func(t *testing.T) {
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
			t.Fatalf("buildOptions: %v", err)
		}

		out, err := press.Render(integrationDeck, opts)
		if err != nil {
			t.Fatalf("press.Render with custom theme-set: %v", err)
		}
		if !strings.Contains(out.CSS, "#d4a853") {
			t.Errorf("Output.CSS missing the custom theme's scoped rule (#d4a853): %d bytes", len(out.CSS))
		}
	})
}

// TestIntegrationServeConvertAndTraversal is test-list case 5: the 04-07
// serve mux converts a .md on request through the SAME buildOptions/
// press.Render/assembleHTML chain the other modes share, and rejects
// /../../etc/passwd -- run against httptest.NewServer per error_recovery
// (no real port bind; 8080/8091 are never touched here).
func TestIntegrationServeConvertAndTraversal(t *testing.T) {
	cmd := prepServeCmd(t, nil)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "deck.md"), []byte(integrationDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	hub := reload.NewHub()
	srv := httptest.NewServer(serveMux(dir, hub, cmd))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/deck.md")
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if !strings.Contains(string(body), "Hello Integration") {
		t.Errorf("serve response missing rendered deck content: %q", body)
	}

	traversal, err := http.Get(srv.URL + "/../../etc/passwd")
	if err != nil {
		t.Fatalf("http.Get(traversal): %v", err)
	}
	defer traversal.Body.Close()
	traversalBody, err := io.ReadAll(traversal.Body)
	if err != nil {
		t.Fatalf("io.ReadAll(traversal): %v", err)
	}
	if strings.Contains(string(traversalBody), "root:") {
		t.Fatalf("traversal guard failed: /etc/passwd content leaked: %q", traversalBody)
	}
	switch traversal.StatusCode {
	case http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound:
		// all three are safe rejections -- see serve_test.go's layered-defense comment
	default:
		t.Errorf("traversal status = %d, want 400, 403, or 404", traversal.StatusCode)
	}
}
