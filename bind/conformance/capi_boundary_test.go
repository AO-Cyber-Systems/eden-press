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
	"fmt"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus"
	"github.com/AO-Cyber-Systems/eden-press/conformance/report"
	"github.com/AO-Cyber-Systems/eden-press/conformance/runner"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// skipIfCapiUnavailable SKIP-guards the capi boundary lane (07-04
// error_recovery) so a bare `go test ./...` on a machine that hasn't built
// the host-arch libpress.so does not hard-fail.
func skipIfCapiUnavailable(t *testing.T) {
	t.Helper()
	if !hostLibAvailable() {
		t.Skip("bind/capi/build/host/libpress.so not found -- run scripts/build-capi-host.sh first")
	}
}

// renderThroughCapi builds the SAME wire request envelope the WASM lane
// builds (buildRequestJSON), calls the dlopen'd host libpress.so's
// PressRender/PressFree (capi_boundary.go) -- the exact C ABI dart:ffi uses
// -- and parses the response envelope.
func renderThroughCapi(markdown string, opts map[string]any) (responseEnvelope, error) {
	reqJSON := buildRequestJSON(markdown, opts)
	raw, err := callPressRender(string(reqJSON))
	if err != nil {
		return responseEnvelope{}, err
	}
	return parseResponse([]byte(raw))
}

// capiRenderFunc adapts renderThroughCapi into a runner.RenderFunc, the SAME
// engine-agnostic seam the WASM lane (wasm_boundary_test.go) and
// conformance/runner's own goldmark RenderFuncs share.
func capiRenderFunc() runner.RenderFunc {
	return func(markdown string, opts map[string]any) (string, error) {
		resp, err := renderThroughCapi(markdown, opts)
		if err != nil {
			return "", err
		}
		if resp.Error != "" {
			return "", fmt.Errorf("capi boundary error envelope: %s", resp.Error)
		}
		if resp.Output == nil {
			return "", fmt.Errorf("capi boundary: response has neither output nor error")
		}
		return resp.Output.HTML, nil
	}
}

// TestCapiBoundaryParity is Test-list case 4: for every subset case, the
// dlopen'd host libpress.so's PressRender reproduces in-process
// press.Render's HTML losslessly -- the real C ABI dart:ffi uses, NOT
// core.RenderJSON called in-process (07-04 anti_patterns).
func TestCapiBoundaryParity(t *testing.T) {
	skipIfCapiUnavailable(t)

	cases, err := Subset()
	if err != nil {
		t.Fatalf("Subset(): %v", err)
	}

	rf := capiRenderFunc()
	rep := report.New()
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			expected, err := press.Render(c.InputMD, pressOptionsFromMap(c.Options))
			if err != nil {
				t.Fatalf("in-process press.Render(%q): %v", c.ID, err)
			}
			synthetic := corpus.Case{ID: c.ID, InputMD: c.InputMD, Options: c.Options, ExpectedHTML: expected.HTML}
			pass, diff := runner.RunCase(rf, synthetic, "capi/"+c.ID, rep)
			if !pass {
				t.Errorf("capi boundary parity mismatch for %q:\n%s", c.ID, diff)
			}
		})
	}
	t.Logf("capi boundary parity report:\n%s", rep.Render())
}

// TestCapiBoundaryWholeShape is the capi half of Test-list case 3: the
// parsed capi response envelope carries the whole Output -- non-nil Model
// with schemaVersion set, non-empty CSS, and Comments present when the case
// carries a speaker note.
func TestCapiBoundaryWholeShape(t *testing.T) {
	skipIfCapiUnavailable(t)

	cases, err := Subset()
	if err != nil {
		t.Fatalf("Subset(): %v", err)
	}

	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			resp, err := renderThroughCapi(c.InputMD, c.Options)
			if err != nil {
				t.Fatalf("renderThroughCapi(%q): %v", c.ID, err)
			}
			if resp.Error != "" {
				t.Fatalf("capi boundary error envelope for %q: %s", c.ID, resp.Error)
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

// TestCapiBoundaryMemoryPlumbing is Test-list case 5: the dlopen'd
// PressRender/PressFree symbols resolve, and PressFree is invoked on EVERY
// PressRender return across the whole subset loop -- no leaked C-heap
// output.
func TestCapiBoundaryMemoryPlumbing(t *testing.T) {
	skipIfCapiUnavailable(t)

	if err := openHostLib(); err != nil {
		t.Fatalf("dlopen/dlsym host libpress.so: %v", err)
	}

	cases, err := Subset()
	if err != nil {
		t.Fatalf("Subset(): %v", err)
	}

	rendersBefore := pressRenderCalls
	freesBefore := pressFreeCalls

	for _, c := range cases {
		if _, err := callPressRender(string(buildRequestJSON(c.InputMD, c.Options))); err != nil {
			t.Errorf("case %q: callPressRender: %v", c.ID, err)
		}
	}

	gotRenders := pressRenderCalls - rendersBefore
	gotFrees := pressFreeCalls - freesBefore
	if gotRenders != len(cases) {
		t.Errorf("PressRender call count = %d, want %d (one per subset case)", gotRenders, len(cases))
	}
	if gotFrees != gotRenders {
		t.Errorf("PressFree call count = %d, want %d (PressFree must be called on EVERY PressRender return -- leak detected)", gotFrees, gotRenders)
	}
}
