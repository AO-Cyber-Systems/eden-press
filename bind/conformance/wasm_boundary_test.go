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

package conformance

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus"
	"github.com/AO-Cyber-Systems/eden-press/conformance/report"
	"github.com/AO-Cyber-Systems/eden-press/conformance/runner"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// wasmBinPath is the artifact scripts/build-wasm.sh produces; bind/conformance
// and bind/wasm are siblings under bind/, so one ".." reaches it.
var wasmBinPath = filepath.Join("..", "wasm", "press.wasm")

// wasmRunnerPath is resolved relative to the package directory: `go test`
// runs test binaries with their working directory set to the package's
// source directory, and wasm_runner.mjs itself resolves bind/wasm/ relative
// to its OWN file location (import.meta.url), not the process cwd, so this
// bare filename is all exec.Command needs.
const wasmRunnerPath = "wasm_runner.mjs"

func wasmArtifactAvailable() bool {
	_, err := os.Stat(wasmBinPath)
	return err == nil
}

func nodeAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

// skipIfWASMUnavailable SKIP-guards the WASM boundary lane (07-04
// error_recovery) so a bare `go test ./...` on a machine without Node or an
// unbuilt press.wasm does not hard-fail.
func skipIfWASMUnavailable(t *testing.T) {
	t.Helper()
	if !nodeAvailable() {
		t.Skip("node not found on PATH -- install Node to run the WASM boundary lane")
	}
	if !wasmArtifactAvailable() {
		t.Skip("bind/wasm/press.wasm not found -- run scripts/build-wasm.sh first")
	}
}

// renderThroughWASM execs wasm_runner.mjs, piping the built request envelope
// to its stdin and reading the response envelope from its stdout -- the SAME
// JSON entrypoint (pressRender) the Dart web loader will call.
func renderThroughWASM(markdown string, opts map[string]any) (responseEnvelope, error) {
	reqJSON := buildRequestJSON(markdown, opts)

	cmd := exec.Command("node", wasmRunnerPath)
	cmd.Stdin = bytes.NewReader(reqJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return responseEnvelope{}, fmt.Errorf("wasm_runner.mjs: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	return parseResponse(stdout.Bytes())
}

// wasmRenderFunc adapts renderThroughWASM into a runner.RenderFunc, the SAME
// engine-agnostic seam conformance/runner's own goldmark RenderFuncs and the
// capi lane (capi_boundary_test.go) share.
func wasmRenderFunc() runner.RenderFunc {
	return func(markdown string, opts map[string]any) (string, error) {
		resp, err := renderThroughWASM(markdown, opts)
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return "", fmt.Errorf("wasm boundary error envelope: %s", resp.Error)
		}
		if resp.Output == nil {
			return "", fmt.Errorf("wasm boundary: response has neither output nor error")
		}
		return resp.Output.HTML, nil
	}
}

// TestSubsetCoverage is Test-list case 1: the selected subset is non-empty
// and its union covers every RequiredBatteries entry.
func TestSubsetCoverage(t *testing.T) {
	cases, err := Subset()
	if err != nil {
		t.Fatalf("Subset(): %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("Subset() returned no cases")
	}

	covered := make(map[string]bool, len(RequiredBatteries))
	for _, c := range cases {
		battery, ok := BatteryOf[c.ID]
		if !ok {
			t.Errorf("case %q has no BatteryOf entry", c.ID)
			continue
		}
		covered[battery] = true
	}

	for _, want := range RequiredBatteries {
		if !covered[want] {
			t.Errorf("subset does not cover required battery %q", want)
		}
	}
}

// TestWASMBoundaryParity is Test-list case 2: for every subset case, the
// compiled press.wasm (driven through Node via the pressRender JSON
// entrypoint) reproduces in-process press.Render's HTML losslessly.
func TestWASMBoundaryParity(t *testing.T) {
	skipIfWASMUnavailable(t)

	cases, err := Subset()
	if err != nil {
		t.Fatalf("Subset(): %v", err)
	}

	rf := wasmRenderFunc()
	rep := report.New()
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			expected, err := press.Render(c.InputMD, pressOptionsFromMap(c.Options))
			if err != nil {
				t.Fatalf("in-process press.Render(%q): %v", c.ID, err)
			}
			synthetic := corpus.Case{ID: c.ID, InputMD: c.InputMD, Options: c.Options, ExpectedHTML: expected.HTML}
			pass, diff := runner.RunCase(rf, synthetic, "wasm/"+c.ID, rep)
			if !pass {
				t.Errorf("wasm boundary parity mismatch for %q:\n%s", c.ID, diff)
			}
		})
	}
	t.Logf("wasm boundary parity report:\n%s", rep.Render())
}

// TestWASMBoundaryWholeShape is Test-list case 3 (wasm half): the parsed wasm
// response envelope carries the whole Output shape -- a non-nil Model with
// schemaVersion set, non-empty CSS, and Comments present when the case
// carries a speaker note -- proving the whole Output crosses, not just the
// HTML string.
func TestWASMBoundaryWholeShape(t *testing.T) {
	skipIfWASMUnavailable(t)

	cases, err := Subset()
	if err != nil {
		t.Fatalf("Subset(): %v", err)
	}

	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			resp, err := renderThroughWASM(c.InputMD, c.Options)
			if err != nil {
				t.Fatalf("renderThroughWASM(%q): %v", c.ID, err)
			}
			if resp.Error != "" {
				t.Fatalf("wasm boundary error envelope for %q: %s", c.ID, resp.Error)
			}
			out := resp.Output
			if out == nil {
				t.Fatalf("case %q: response has no output", c.ID)
			}
			if out.Model == nil {
				t.Errorf("case %q: output.Model is nil -- whole Output shape did not cross", c.ID)
			} else if out.Model.SchemaVersion == "" {
				t.Errorf("case %q: output.Model.schemaVersion is empty", c.ID)
			}
			if strings.TrimSpace(out.CSS) == "" {
				t.Errorf("case %q: output.CSS is empty", c.ID)
			}
			if strings.Contains(c.InputMD, "<!--") && strings.Contains(c.InputMD, "note") {
				if len(out.Comments) == 0 {
					t.Errorf("case %q: input carries a speaker note but output.Comments is empty", c.ID)
				}
			}
		})
	}
}
