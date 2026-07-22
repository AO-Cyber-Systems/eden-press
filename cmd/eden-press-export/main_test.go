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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/convert/chrome"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// exportTestDeck is a hand-built (no_llm_test_data), three-slide deck
// covering a plain text slide, a math slide, and a closing slide. It
// carries NO relative asset URLs (mirrors convert/export_integration_test.go's
// capstoneDeck self-containment contract), so the live end-to-end test never
// has to resolve a filesystem/network reference.
const exportTestDeck = "---\n" +
	"marp: true\n" +
	"---\n\n" +
	"# Slide One\n\n" +
	"A plain text slide.\n\n" +
	"---\n\n" +
	"# Slide Two -- Math\n\n" +
	"Inline $a^2 + b^2 = c^2$.\n\n" +
	"---\n\n" +
	"# Slide Three\n\n" +
	"A closing slide.\n"

// TestFlagSurface is test-list case 1: newRootCmd registers --format
// (default pdf), -o/--output, --browser-path, and the render knobs, and
// accepts at most one positional argument. Structural -- runs everywhere,
// no Chrome required.
func TestFlagSurface(t *testing.T) {
	cmd := newRootCmd()

	wantFlags := []struct {
		name string
		def  string
	}{
		{"format", "pdf"},
		{"output", ""},
		{"browser-path", ""},
		{"theme", ""},
		{"profile", ""},
		{"math", ""},
		{"no-highlight", "false"},
		{"highlight-style", ""},
		{"inline-svg", "false"},
	}
	for _, wf := range wantFlags {
		f := cmd.Flags().Lookup(wf.name)
		if f == nil {
			t.Fatalf("flag %q not registered", wf.name)
		}
		if f.DefValue != wf.def {
			t.Errorf("flag %q default = %q, want %q", wf.name, f.DefValue, wf.def)
		}
	}

	if cmd.Flags().ShorthandLookup("o") == nil {
		t.Error(`"-o" shorthand not registered for --output`)
	}

	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("expected an error for 2 positional args (Args: cobra.MaximumNArgs(1))")
	}
	if err := cmd.Args(cmd, []string{"a"}); err != nil {
		t.Errorf("expected exactly 1 positional arg to be accepted: %v", err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Errorf("expected 0 positional args to be accepted: %v", err)
	}
}

// TestResolveInput is test-list case 2: a temp deck.md reads its bytes; "-"
// reads injected stdin via the io.Reader seam (never the real process
// stdin).
func TestResolveInput(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "deck.md")
		if err := os.WriteFile(p, []byte("# hello\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		md, isStdin, err := resolveInput(p, nil)
		if err != nil {
			t.Fatalf("resolveInput: %v", err)
		}
		if isStdin {
			t.Error("isStdin = true, want false for a file argument")
		}
		if md != "# hello\n" {
			t.Errorf("md = %q, want %q", md, "# hello\n")
		}
	})

	t.Run("stdin", func(t *testing.T) {
		md, isStdin, err := resolveInput("-", strings.NewReader("# stdin deck\n"))
		if err != nil {
			t.Fatalf("resolveInput: %v", err)
		}
		if !isStdin {
			t.Error(`isStdin = false, want true for "-"`)
		}
		if md != "# stdin deck\n" {
			t.Errorf("md = %q, want %q", md, "# stdin deck\n")
		}
	})
}

// TestBuildPressOptions is test-list case 3: flags map 1:1 onto
// press.Options.
func TestBuildPressOptions(t *testing.T) {
	cmd := newRootCmd()
	if err := cmd.Flags().Parse([]string{"--theme", "dracula", "--inline-svg"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := buildPressOptions(cmd)
	want := press.Options{Theme: "dracula", InlineSVG: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildPressOptions = %+v, want %+v", got, want)
	}
}

// TestBuildPressOptionsZeroValues is test-list case 3's other half: unset
// flags pass through as press.Options zero values (buildPressOptions never
// substitutes a default of its own -- that is press.Render's job).
func TestBuildPressOptionsZeroValues(t *testing.T) {
	cmd := newRootCmd()
	if err := cmd.Flags().Parse(nil); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := buildPressOptions(cmd)
	want := press.Options{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildPressOptions (unset flags) = %+v, want the zero value %+v", got, want)
	}
}

// TestPNGOutputPath is test-list case 4: the directory branch, the
// %d-pattern branch, and the empty-"-o" CWD branch.
func TestPNGOutputPath(t *testing.T) {
	tests := []struct {
		name string
		o    string
		stem string
		n    int
		want string
	}{
		{"directory slide 1", "dir", "deck", 1, filepath.Join("dir", "deck-001.png")},
		{"directory slide 2", "dir", "deck", 2, filepath.Join("dir", "deck-002.png")},
		{"pattern slide 1", "frame-%03d.png", "deck", 1, "frame-001.png"},
		{"pattern slide 2", "frame-%03d.png", "deck", 2, "frame-002.png"},
		{"empty -o (CWD)", "", "deck", 1, "deck-001.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pngOutputPath(tt.o, tt.stem, tt.n)
			if got != tt.want {
				t.Errorf("pngOutputPath(%q, %q, %d) = %q, want %q", tt.o, tt.stem, tt.n, got, tt.want)
			}
		})
	}
}

// TestChromeUnavailableError is test-list case 5: wrapping
// chrome.ErrChromeNotFound yields exit code 3 and a message naming both
// --browser-path and CHROME_PATH. Pure and deterministic -- no environment
// gating needed.
func TestChromeUnavailableError(t *testing.T) {
	err := chromeUnavailableError(chrome.ErrChromeNotFound)

	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("chromeUnavailableError did not return a *cliError: %v", err)
	}
	if ce.code != exitNoChrome {
		t.Errorf("code = %d, want %d (exitNoChrome)", ce.code, exitNoChrome)
	}

	msg := ce.Error()
	if !strings.Contains(msg, "--browser-path") {
		t.Errorf("message does not name --browser-path: %q", msg)
	}
	if !strings.Contains(msg, "CHROME_PATH") {
		t.Errorf("message does not name CHROME_PATH: %q", msg)
	}
}

// TestEndToEndPDFAndPNG is test-list case 6, the SKIP-guarded live case:
// driving runExport through newRootCmd end-to-end over a hand-built,
// multi-slide, one-math-slide deck. "--format pdf -o t.pdf" must yield a
// %PDF- file; "--format png -o <tmpdir>" must yield len(out.Model.Sections)
// PNG files, each starting with the PNG magic header. Skips cleanly (never
// fails) when no Chrome/Chromium is discoverable, exactly like every other
// live-Chrome test in this module (convert/export_integration_test.go,
// convert/pdf/pdf_test.go, convert/png/png_test.go) -- this sandbox has
// none.
func TestEndToEndPDFAndPNG(t *testing.T) {
	if _, _, err := chrome.Discover(chrome.DiscoverOptions{}); err != nil {
		t.Skipf("no Chrome discovered, skipping live-Chrome export test: %v", err)
	}

	out, err := press.Render(exportTestDeck, press.Options{InlineSVG: true})
	if err != nil {
		t.Fatalf("press.Render (test setup): %v", err)
	}
	if out.Model == nil || len(out.Model.Sections) != 3 {
		t.Fatalf("exportTestDeck produced %d sections, want 3 (authoring bug)", len(out.Model.Sections))
	}
	wantSlides := len(out.Model.Sections)

	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(deckPath, []byte(exportTestDeck), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("pdf", func(t *testing.T) {
		pdfPath := filepath.Join(dir, "t.pdf")

		cmd := newRootCmd()
		cmd.SetArgs([]string{"--format", "pdf", "--inline-svg", "-o", pdfPath, deckPath})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute (--format pdf): %v", err)
		}

		b, err := os.ReadFile(pdfPath)
		if err != nil {
			t.Fatalf("reading %q: %v", pdfPath, err)
		}
		if !bytes.HasPrefix(b, []byte("%PDF-")) {
			n := len(b)
			if n > 16 {
				n = 16
			}
			t.Fatalf("pdf output does not start with the %%PDF- magic header: first bytes %q", b[:n])
		}
	})

	t.Run("png", func(t *testing.T) {
		pngDir := filepath.Join(dir, "pngs")

		cmd := newRootCmd()
		cmd.SetArgs([]string{"--format", "png", "--inline-svg", "-o", pngDir, deckPath})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute (--format png): %v", err)
		}

		entries, err := os.ReadDir(pngDir)
		if err != nil {
			t.Fatalf("reading dir %q: %v", pngDir, err)
		}
		if len(entries) != wantSlides {
			t.Fatalf("wrote %d PNG files, want %d (== len(out.Model.Sections))", len(entries), wantSlides)
		}
		for _, e := range entries {
			b, err := os.ReadFile(filepath.Join(pngDir, e.Name()))
			if err != nil {
				t.Fatalf("reading %q: %v", e.Name(), err)
			}
			if !bytes.HasPrefix(b, []byte("\x89PNG")) {
				t.Errorf("%q does not start with the PNG magic header", e.Name())
			}
		}
	})
}
