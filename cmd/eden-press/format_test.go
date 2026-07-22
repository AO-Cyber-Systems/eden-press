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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// formatTestDeck carries a heading (non-empty model.outline), a fenced ```go
// code block, and a display "$$...$$" math block -- exactly the schema-v2
// surfaces (Block{Kind:"code"|"math"}) this TRD's JSON envelope must expose
// verbatim, with NO renamed source/tex keys.
const formatTestDeck = "# Format Test\n\n" +
	"```go\nfmt.Println(1)\n```\n\n" +
	"$$E=mc^2$$\n"

// renderFormatTestDeck renders formatTestDeck through the same press.Render
// entry point runConvert uses (press.Options{} zero value -- the documented
// safe default, math battery ON) -- these are format.go unit tests, so the
// press.Output is built directly rather than driven through cobra flag
// parsing (the --format flag itself is wired into flags.go/convert.go/
// root.go by Task 2, not this file's own scope).
func renderFormatTestDeck(t *testing.T) press.Output {
	t.Helper()
	out, err := press.Render(formatTestDeck, press.Options{})
	if err != nil {
		t.Fatalf("press.Render(formatTestDeck): %v", err)
	}
	return out
}

// TestFormatJSONEnvelopeStructure is test-list cases 1+2: writeJSON emits a
// single JSON object with lowercase top-level keys
// {html,css,model,comments,meta}; model.sections[].blocks carries the
// schema-v2 model VERBATIM -- a code block as {kind:"code", language:"go",
// text:<raw source>} and a math block as {kind:"math", display:true,
// text:<raw TeX>} -- with NO renamed source/tex keys, and model.outline is
// non-empty.
func TestFormatJSONEnvelopeStructure(t *testing.T) {
	out := renderFormatTestDeck(t)

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := writeJSON(cmd, out); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal envelope: %v\nraw: %s", err, buf.String())
	}

	for _, key := range []string{"html", "css", "model", "comments", "meta"} {
		if _, ok := envelope[key]; !ok {
			t.Errorf("envelope missing top-level key %q: %v", key, envelope)
		}
	}

	model, ok := envelope["model"].(map[string]any)
	if !ok {
		t.Fatalf("envelope[model] is not an object: %T (%v)", envelope["model"], envelope["model"])
	}

	sections, ok := model["sections"].([]any)
	if !ok || len(sections) == 0 {
		t.Fatalf("model.sections missing or empty: %v", model["sections"])
	}

	outline, ok := model["outline"].([]any)
	if !ok || len(outline) == 0 {
		t.Errorf("model.outline missing or empty (want the deck's heading): %v", model["outline"])
	}

	var sawCode, sawMath bool
	for _, s := range sections {
		sec, ok := s.(map[string]any)
		if !ok {
			continue
		}
		blocks, _ := sec["blocks"].([]any)
		for _, b := range blocks {
			block, ok := b.(map[string]any)
			if !ok {
				continue
			}
			switch block["kind"] {
			case "code":
				sawCode = true
				if block["language"] != "go" {
					t.Errorf("code block language = %v, want %q", block["language"], "go")
				}
				text, _ := block["text"].(string)
				if !strings.Contains(text, "fmt.Println") {
					t.Errorf("code block text = %q, want it to contain the raw source", text)
				}
				if _, ok := block["source"]; ok {
					t.Errorf("code block carries a `source` key -- schema-v2 forbids it (raw source lives in `text`): %v", block)
				}
			case "math":
				sawMath = true
				if display, _ := block["display"].(bool); !display {
					t.Errorf("math block display = %v, want true (display math $$...$$)", block["display"])
				}
				text, _ := block["text"].(string)
				if !strings.Contains(text, "E=mc^2") {
					t.Errorf("math block text = %q, want it to contain the raw TeX", text)
				}
				if _, ok := block["tex"]; ok {
					t.Errorf("math block carries a `tex` key -- schema-v2 forbids it (raw TeX lives in `text`): %v", block)
				}
			}
		}
	}

	if !sawCode {
		t.Error("no code block found in model.sections[].blocks")
	}
	if !sawMath {
		t.Error("no math block found in model.sections[].blocks")
	}
}

// TestFormatJSONOutputFile is test-list case 6: emitFormat under
// cfg["format"]=="json" with --output/-o set writes the envelope to the
// file (not stdout), and the file parses as JSON. --output is registered
// via the EXISTING registerConvertFlags (flags.go, 04-02) -- this test does
// not depend on Task 2's new --format flag wiring.
func TestFormatJSONOutputFile(t *testing.T) {
	resetCfg()
	if err := cfg.Set("format", "json"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}

	out := renderFormatTestDeck(t)

	cmd := &cobra.Command{Use: "test"}
	registerConvertFlags(cmd)
	outPath := filepath.Join(t.TempDir(), "out.json")
	if err := cmd.ParseFlags([]string{"--output", outPath}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := emitFormat(cmd, out); err != nil {
		t.Fatalf("emitFormat: %v", err)
	}

	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty (json output should go to the file)", stdout.String())
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", outPath, err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(written, &envelope); err != nil {
		t.Fatalf("json.Unmarshal file contents: %v\nraw: %s", err, written)
	}
	if _, ok := envelope["html"]; !ok {
		t.Errorf("file envelope missing %q key: %v", "html", envelope)
	}
}

// TestFormatErrorEnvelope is test-list case 3 (unit-level): under
// cfg["format"]=="json", cliFail writes a JSON error envelope
// ({"error":{"code":<int>,"message":<string>}}) to stderr and returns a
// *cliError carrying the same code with printed=true.
func TestFormatErrorEnvelope(t *testing.T) {
	resetCfg()
	if err := cfg.Set("format", "json"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}

	cmd := &cobra.Command{Use: "test"}
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)

	underlying := errors.New(`read file "/does/not/exist.md": no such file or directory`)
	err := cliFail(cmd, exitRender, underlying)

	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("cliFail did not return a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitRender {
		t.Errorf("ce.code = %d, want %d (exitRender)", ce.code, exitRender)
	}
	if !ce.printed {
		t.Error("ce.printed = false, want true (the json envelope was already written to stderr)")
	}

	var envelope jsonError
	if jsonErr := json.Unmarshal(errBuf.Bytes(), &envelope); jsonErr != nil {
		t.Fatalf("json.Unmarshal stderr envelope: %v\nraw: %s", jsonErr, errBuf.String())
	}
	if envelope.Error.Code != exitRender {
		t.Errorf("stderr envelope error.code = %d, want %d", envelope.Error.Code, exitRender)
	}
	if envelope.Error.Message == "" {
		t.Error("stderr envelope error.message is empty, want the underlying error text")
	}
}

// TestCliFailClassification is test-list case 4: cliFail classifies a
// failure as code=1(render)/2(usage)/printed=false when the active format
// is NOT json, and printed=true plus a parseable JSON envelope on stderr
// when the active format IS json.
func TestCliFailClassification(t *testing.T) {
	resetCfg()
	cmd := &cobra.Command{Use: "test"}
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)

	// Non-json format: plain failure -- cliFail itself must not print.
	err := cliFail(cmd, exitRender, errors.New("render boom"))
	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("cliFail did not return a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitRender {
		t.Errorf("code = %d, want %d (exitRender)", ce.code, exitRender)
	}
	if ce.printed {
		t.Error("printed = true, want false for a non-json failure")
	}
	if errBuf.Len() != 0 {
		t.Errorf("stderr = %q, want empty for a non-json failure (main.go prints it, not cliFail)", errBuf.String())
	}

	// Non-json usage error.
	err = cliFail(cmd, exitUsage, errors.New(`unknown --format "zzz"`))
	if !errors.As(err, &ce) {
		t.Fatalf("cliFail did not return a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitUsage {
		t.Errorf("code = %d, want %d (exitUsage)", ce.code, exitUsage)
	}
	if ce.printed {
		t.Error("printed = true, want false for a non-json usage failure")
	}

	// json format: cliFail writes the envelope itself and marks printed.
	resetCfg()
	if err := cfg.Set("format", "json"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}
	errBuf.Reset()

	err = cliFail(cmd, exitUsage, errors.New("bad --format value"))
	if !errors.As(err, &ce) {
		t.Fatalf("cliFail did not return a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitUsage {
		t.Errorf("code = %d, want %d (exitUsage)", ce.code, exitUsage)
	}
	if !ce.printed {
		t.Error("printed = false, want true for a json failure")
	}

	var envelope jsonError
	if jsonErr := json.Unmarshal(errBuf.Bytes(), &envelope); jsonErr != nil {
		t.Fatalf("json.Unmarshal: %v\nraw: %s", jsonErr, errBuf.String())
	}
	if envelope.Error.Code != exitUsage {
		t.Errorf("error.code = %d, want %d", envelope.Error.Code, exitUsage)
	}
	if envelope.Error.Message == "" {
		t.Error("error.message is empty")
	}
}

// TestEmitFormatDispatch exercises the ONE format-dispatch seam directly
// against a hand-built press.Output: html routes through the existing
// assembleHTML/writeOutput string pipeline, json routes through writeJSON,
// pptx registers (parses) but is not yet wired (usage error), and an
// unknown value is a usage error.
func TestEmitFormatDispatch(t *testing.T) {
	sampleOut := press.Output{HTML: "<p>hi</p>", CSS: "body{color:red}"}

	// html (default/unset): unchanged zero-JS document, no JSON envelope.
	resetCfg()
	cmd := &cobra.Command{Use: "test"}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := emitFormat(cmd, sampleOut); err != nil {
		t.Fatalf("emitFormat(html): %v", err)
	}
	if !strings.Contains(out.String(), "<!doctype html>") {
		t.Errorf("emitFormat(html) output missing doctype: %q", out.String())
	}
	if strings.Contains(out.String(), `"html"`) {
		t.Errorf("emitFormat(html) output looks like JSON, want the HTML document: %q", out.String())
	}

	// json: routes through writeJSON -- the lowercase envelope.
	resetCfg()
	if err := cfg.Set("format", "json"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}
	out.Reset()
	if err := emitFormat(cmd, sampleOut); err != nil {
		t.Fatalf("emitFormat(json): %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal: %v\nraw: %s", err, out.String())
	}
	if envelope["html"] != sampleOut.HTML {
		t.Errorf("envelope[html] = %v, want %q", envelope["html"], sampleOut.HTML)
	}

	// pptx: registered (flag surface complete, once Task 2 wires --format)
	// but not yet wired -- 04.1-02 fills this case. Today it is a usage
	// error, never a panic/nil-deref.
	resetCfg()
	if err := cfg.Set("format", "pptx"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}
	err := emitFormat(cmd, sampleOut)
	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("emitFormat(pptx) error is not a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitUsage {
		t.Errorf("emitFormat(pptx) code = %d, want %d (not-yet-wired usage error)", ce.code, exitUsage)
	}

	// unknown value: usage error.
	resetCfg()
	if err := cfg.Set("format", "zzz"); err != nil {
		t.Fatalf("cfg.Set: %v", err)
	}
	err = emitFormat(cmd, sampleOut)
	if !errors.As(err, &ce) {
		t.Fatalf("emitFormat(zzz) error is not a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitUsage {
		t.Errorf("emitFormat(zzz) code = %d, want %d", ce.code, exitUsage)
	}
}

// --- Task 2: end-to-end CLI tests, driven through newRootCmd()/SetArgs now
// that --format is wired as a persistent flag (flags.go) and runConvert
// routes through emitFormat/cliFail (convert.go). ---

// TestRunConvertDefaultFormatUnchanged is test-list case 5, the CLI-01
// regression guard: `eden-press deck.md` with NO --format flag still emits
// the standalone zero-JS `<!doctype html>` document, byte-behavior
// unchanged by this TRD's --format wiring.
func TestRunConvertDefaultFormatUnchanged(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(testDeck), 0o644); err != nil {
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

	got := out.String()
	if !strings.Contains(got, "<!doctype html>") {
		t.Errorf("stdout missing <!doctype html>: %q", got)
	}
	if !strings.Contains(got, "Hello Convert") {
		t.Errorf("stdout missing rendered deck content: %q", got)
	}
	if strings.Contains(got, "<script") {
		t.Errorf("default (no --format) convert output contains <script>, want zero-JS: %q", got)
	}
}

// TestEndToEndFormatJSON is the full-CLI happy path (test-list cases 1+2,
// end-to-end): `eden-press <deck> --format json` (root's bare default,
// exercising the actual --format flag registered by flags.go) emits a
// parseable envelope whose model.sections[].blocks carry the schema-v2
// shape verbatim -- text+language for code, text+display for math -- with
// NO renamed source/tex keys, and model.outline non-empty.
func TestEndToEndFormatJSON(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(formatTestDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{path, "--format", "json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (output: %s)", err, out.String())
	}

	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal envelope: %v\nraw: %s", err, out.String())
	}

	for _, key := range []string{"html", "css", "model", "comments", "meta"} {
		if _, ok := envelope[key]; !ok {
			t.Errorf("envelope missing top-level key %q: %v", key, envelope)
		}
	}

	model, ok := envelope["model"].(map[string]any)
	if !ok {
		t.Fatalf("envelope[model] is not an object: %T", envelope["model"])
	}
	sections, _ := model["sections"].([]any)
	outline, _ := model["outline"].([]any)
	if len(outline) == 0 {
		t.Errorf("model.outline empty, want the deck's heading entry")
	}

	var sawCode, sawMath bool
	for _, s := range sections {
		sec, _ := s.(map[string]any)
		blocks, _ := sec["blocks"].([]any)
		for _, b := range blocks {
			block, _ := b.(map[string]any)
			switch block["kind"] {
			case "code":
				sawCode = true
				if block["language"] != "go" {
					t.Errorf("code block language = %v, want %q", block["language"], "go")
				}
				text, _ := block["text"].(string)
				if !strings.Contains(text, "fmt.Println") {
					t.Errorf("code block text = %q, want raw source", text)
				}
				if _, ok := block["source"]; ok {
					t.Errorf("code block has a forbidden `source` key: %v", block)
				}
			case "math":
				sawMath = true
				if disp, _ := block["display"].(bool); !disp {
					t.Errorf("math block display = %v, want true", block["display"])
				}
				text, _ := block["text"].(string)
				if !strings.Contains(text, "E=mc^2") {
					t.Errorf("math block text = %q, want raw TeX", text)
				}
				if _, ok := block["tex"]; ok {
					t.Errorf("math block has a forbidden `tex` key: %v", block)
				}
			}
		}
	}
	if !sawCode {
		t.Error("no code block found end-to-end")
	}
	if !sawMath {
		t.Error("no math block found end-to-end")
	}
}

// TestEndToEndFormatJSONErrorExitCode is the full-CLI failure path
// (test-list case 3, end-to-end): a nonexistent input path with
// `--format json` fails with exit code 1 (exitRender), and stderr carries
// the {"error":{"code":1,"message":...}} envelope.
func TestEndToEndFormatJSONErrorExitCode(t *testing.T) {
	resetCfg()

	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs([]string{"/does/not/exist.md", "--format", "json"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute: want an error for a nonexistent input file, got nil")
	}

	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("Execute error is not a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitRender {
		t.Errorf("ce.code = %d, want %d (exitRender)", ce.code, exitRender)
	}
	if out.String() != "" {
		t.Errorf("stdout = %q, want empty on failure", out.String())
	}

	var envelope jsonError
	if jsonErr := json.Unmarshal(errBuf.Bytes(), &envelope); jsonErr != nil {
		t.Fatalf("json.Unmarshal stderr envelope: %v\nraw: %s", jsonErr, errBuf.String())
	}
	if envelope.Error.Code != exitRender {
		t.Errorf("stderr envelope error.code = %d, want %d", envelope.Error.Code, exitRender)
	}
	if envelope.Error.Message == "" {
		t.Error("stderr envelope error.message is empty")
	}
}

// TestEndToEndFormatUnknownIsUsageError is the full-CLI usage-error path:
// `eden-press deck.md --format zzz` fails with exit code 2 (exitUsage).
func TestEndToEndFormatUnknownIsUsageError(t *testing.T) {
	resetCfg()

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(testDeck), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs([]string{path, "--format", "zzz"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute: want an error for an unknown --format value, got nil")
	}

	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("Execute error is not a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitUsage {
		t.Errorf("ce.code = %d, want %d (exitUsage)", ce.code, exitUsage)
	}
}

// TestEndToEndFlagErrorIsUsageError proves an actual cobra/pflag flag-parse
// failure (an unregistered flag) is classified as a *cliError{code:
// exitUsage} via root.go's FlagErrorFunc, and cobra does not double-print
// (SilenceErrors: true) -- stderr carries at most the plain error, never
// cobra's own "Error: ..." + usage text.
func TestEndToEndFlagErrorIsUsageError(t *testing.T) {
	resetCfg()

	root := newRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs([]string{"--no-such-flag"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute: want an error for an unregistered flag, got nil")
	}

	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("Execute error is not a *cliError: %v (%T)", err, err)
	}
	if ce.code != exitUsage {
		t.Errorf("ce.code = %d, want %d (exitUsage)", ce.code, exitUsage)
	}
	if strings.Contains(errBuf.String(), "Error:") {
		t.Errorf("cobra printed its own error text (SilenceErrors not effective): %q", errBuf.String())
	}
}
