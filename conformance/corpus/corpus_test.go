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

package corpus

import (
	"os"
	"path/filepath"
	"testing"
)

// writeCase creates a case directory <root>/<id> and writes the given files.
// All fixture content is hand-built inline (no_llm_test_data).
func writeCase(t *testing.T, root, id string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s/%s: %v", id, name, err)
		}
	}
	return dir
}

func TestLoadCases_PopulatesFields(t *testing.T) {
	root := t.TempDir()

	// Case 1: minimal required trio with a requires_engine hint, no CSS.
	writeCase(t, root, "001-basic", map[string]string{
		"input.md":      "# Title\n\nhello\n",
		"options.json":  `{"requires_engine":"commonmark"}`,
		"expected.html": "<h1>Title</h1>\n<p>hello</p>\n",
	})

	// Case 2: includes the OPTIONAL expected.css and a marpit engine hint.
	writeCase(t, root, "002-themed", map[string]string{
		"input.md":      "---\nmarp: true\n---\n\n# Slide\n",
		"options.json":  `{"requires_engine":"marpit"}`,
		"expected.html": "<section><h1>Slide</h1></section>\n",
		"expected.css":  "section { color: red; }\n",
	})

	cases, err := LoadCases(root)
	if err != nil {
		t.Fatalf("LoadCases returned error: %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(cases))
	}

	// Cases must be returned in a deterministic (sorted-by-ID) order.
	c1 := cases[0]
	if c1.ID != "001-basic" {
		t.Fatalf("expected first case ID 001-basic, got %q", c1.ID)
	}
	if c1.InputMD != "# Title\n\nhello\n" {
		t.Errorf("case1 InputMD not populated: %q", c1.InputMD)
	}
	if c1.ExpectedHTML != "<h1>Title</h1>\n<p>hello</p>\n" {
		t.Errorf("case1 ExpectedHTML not populated: %q", c1.ExpectedHTML)
	}
	if c1.RequiresEngine != "commonmark" {
		t.Errorf("case1 RequiresEngine = %q, want commonmark", c1.RequiresEngine)
	}
	if c1.Options["requires_engine"] != "commonmark" {
		t.Errorf("case1 Options map not decoded: %#v", c1.Options)
	}
	if c1.ExpectedCSS != "" {
		t.Errorf("case1 ExpectedCSS should be empty when absent, got %q", c1.ExpectedCSS)
	}
	if c1.Dir == "" {
		t.Errorf("case1 Dir should be populated")
	}

	c2 := cases[1]
	if c2.ID != "002-themed" {
		t.Fatalf("expected second case ID 002-themed, got %q", c2.ID)
	}
	if c2.RequiresEngine != "marpit" {
		t.Errorf("case2 RequiresEngine = %q, want marpit", c2.RequiresEngine)
	}
	if c2.ExpectedCSS != "section { color: red; }\n" {
		t.Errorf("case2 ExpectedCSS not populated: %q", c2.ExpectedCSS)
	}
}

// A case whose requires_engine is omitted must yield RequiresEngine == "".
func TestLoadCases_MissingRequiresEngineDefaultsEmpty(t *testing.T) {
	root := t.TempDir()
	writeCase(t, root, "no-engine", map[string]string{
		"input.md":      "plain\n",
		"options.json":  `{}`,
		"expected.html": "<p>plain</p>\n",
	})
	cases, err := LoadCases(root)
	if err != nil {
		t.Fatalf("LoadCases error: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(cases))
	}
	if cases[0].RequiresEngine != "" {
		t.Errorf("RequiresEngine should default to empty, got %q", cases[0].RequiresEngine)
	}
}

// A case missing a REQUIRED file must surface a clear error (not silently skip).
func TestLoadCases_MissingRequiredFileErrors(t *testing.T) {
	root := t.TempDir()
	// expected.html deliberately omitted.
	writeCase(t, root, "broken", map[string]string{
		"input.md":     "x\n",
		"options.json": `{}`,
	})
	_, err := LoadCases(root)
	if err == nil {
		t.Fatalf("expected an error for a case missing expected.html, got nil")
	}
}

// Malformed options.json must surface a clear error.
func TestLoadCases_MalformedOptionsErrors(t *testing.T) {
	root := t.TempDir()
	writeCase(t, root, "bad-json", map[string]string{
		"input.md":      "x\n",
		"options.json":  `{not valid json`,
		"expected.html": "<p>x</p>\n",
	})
	_, err := LoadCases(root)
	if err == nil {
		t.Fatalf("expected an error for malformed options.json, got nil")
	}
}

// A missing root directory must error rather than returning an empty slice.
func TestLoadCases_MissingRootErrors(t *testing.T) {
	_, err := LoadCases(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatalf("expected an error for a nonexistent root, got nil")
	}
}
