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

package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// wire* mirror the EXACT on-the-wire JSON keys a downstream front door (the cgo
// shim, the wasm shim 07-02, and 07-05's Dart client) parses. press.Output has
// no json tags, so its fields cross as Go-default capitalized keys
// (HTML/CSS/Model/Meta/Comments); these structs PIN that wire contract, rather
// than round-tripping straight back into the source Go types (which would hide a
// key rename). Never string-match raw JSON -- always parse then assert.
type wireModel struct {
	SchemaVersion string `json:"schemaVersion"`
}

type wireOutput struct {
	HTML     string          `json:"HTML"`
	CSS      string          `json:"CSS"`
	Model    *wireModel      `json:"Model"`
	Meta     json.RawMessage `json:"Meta"`
	Comments []string        `json:"Comments"`
}

type wireResp struct {
	EnvelopeVersion string      `json:"envelopeVersion"`
	Output          *wireOutput `json:"output"`
	Error           string      `json:"error"`
}

// mustParse asserts the load-bearing boundary invariant common to EVERY case:
// RenderJSON never returns a nil/empty slice, and whatever it returns is always
// valid JSON parseable by the (Dart) host.
func mustParse(t *testing.T, b []byte) wireResp {
	t.Helper()
	if len(b) == 0 {
		t.Fatal("RenderJSON returned a nil/empty slice -- the boundary must always be a well-formed JSON envelope")
	}
	var r wireResp
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("response is not valid JSON: %v\nraw: %s", err, b)
	}
	return r
}

// reqJSON hand-builds a request envelope from literal fixtures. It marshals a
// literal map only to sidestep manual backtick/quote escaping for markdown that
// itself contains code fences or quotes -- the inputs are still fixed, inline
// literals (no generator, no property library).
func reqJSON(t *testing.T, markdown string, options map[string]any) []byte {
	t.Helper()
	m := map[string]any{"markdown": markdown}
	if options != nil {
		m["options"] = options
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("build request fixture: %v", err)
	}
	return b
}

// Test-list case 1: happy-path whole-envelope round-trip.
func TestRenderJSON_HappyPathRoundTrip(t *testing.T) {
	resp := mustParse(t, RenderJSON([]byte(`{"envelopeVersion":"eden-press.capi/v1","markdown":"# Hi"}`)))

	if resp.EnvelopeVersion != EnvelopeVersion {
		t.Errorf("envelopeVersion = %q, want %q", resp.EnvelopeVersion, EnvelopeVersion)
	}
	if resp.Error != "" {
		t.Errorf("error = %q, want empty on the happy path", resp.Error)
	}
	if resp.Output == nil {
		t.Fatal("output is nil on the happy path")
	}
	if !strings.Contains(resp.Output.HTML, "<h1") {
		t.Errorf("output.HTML missing rendered <h1: %q", resp.Output.HTML)
	}
}

// Test-list case 2: full-shape losslessness -- the entire press.Output shape
// (HTML, CSS, Model{schemaVersion}, Meta, Comments) crosses JSON intact.
func TestRenderJSON_FullShapeLossless(t *testing.T) {
	// Front matter -> Meta; heading -> HTML; HTML comment -> Comments.
	resp := mustParse(t, RenderJSON([]byte(
		`{"markdown":"---\ntitle: Deck\n---\n\n# Hello\n\n<!-- a speaker note -->"}`)))

	if resp.Error != "" {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
	o := resp.Output
	if o == nil {
		t.Fatal("output is nil")
	}
	if o.HTML == "" {
		t.Error("output.HTML is empty")
	}
	if o.CSS == "" {
		t.Error("output.CSS is empty")
	}
	if o.Model == nil {
		t.Fatal("output.Model is nil")
	}
	if o.Model.SchemaVersion != model.SchemaVersion {
		t.Errorf("output.Model.schemaVersion = %q, want %q", o.Model.SchemaVersion, model.SchemaVersion)
	}
	if len(o.Meta) == 0 || string(o.Meta) == "null" {
		t.Errorf("output.Meta did not cross: %s", o.Meta)
	}
	if len(o.Comments) == 0 || o.Comments[0] != "a speaker note" {
		t.Errorf("output.Comments = %#v, want the speaker note to cross intact", o.Comments)
	}
}

// Test-list case 3: options map correctly onto press.Options.
func TestRenderJSON_OptionMapping(t *testing.T) {
	const codeMD = "```go\nfunc main(){}\n```"

	t.Run("noHighlight strips chroma spans", func(t *testing.T) {
		on := mustParse(t, RenderJSON(reqJSON(t, codeMD, nil)))
		off := mustParse(t, RenderJSON(reqJSON(t, codeMD, map[string]any{"noHighlight": true})))
		if on.Output == nil || off.Output == nil {
			t.Fatal("nil output")
		}
		if !strings.Contains(on.Output.HTML, "<span") {
			t.Error("default render should carry chroma <span highlight markup")
		}
		if strings.Contains(off.Output.HTML, "<span") {
			t.Error("noHighlight:true should emit no chroma <span highlight markup")
		}
	})

	t.Run("mathMode off leaves $x$ literal", func(t *testing.T) {
		off := mustParse(t, RenderJSON(reqJSON(t, "$x^2$", map[string]any{"mathMode": "off"})))
		on := mustParse(t, RenderJSON(reqJSON(t, "$x^2$", nil)))
		if off.Output == nil || on.Output == nil {
			t.Fatal("nil output")
		}
		if !strings.Contains(off.Output.HTML, "x^2") {
			t.Errorf("mathMode:off should leave $x^2$ literal, got: %q", off.Output.HTML)
		}
		if strings.Contains(on.Output.HTML, "$x^2$") {
			t.Error("default mathMode should render math, not leave the raw $x^2$ delimiters")
		}
	})

	t.Run("theme gaia yields distinct CSS", func(t *testing.T) {
		def := mustParse(t, RenderJSON(reqJSON(t, "# Hi", nil)))
		gaia := mustParse(t, RenderJSON(reqJSON(t, "# Hi", map[string]any{"theme": "gaia"})))
		if def.Output == nil || gaia.Output == nil {
			t.Fatal("nil output")
		}
		if def.Output.CSS == "" || gaia.Output.CSS == "" {
			t.Fatal("CSS empty")
		}
		if def.Output.CSS == gaia.Output.CSS {
			t.Error("theme:gaia should map onto press.Options.Theme and yield CSS distinct from default")
		}
	})
}

// Test-list case 4: Sanitize-over-the-wire. The wire request carries NO Sanitize
// field; the built-in always-on policy (nil => built-in) strips a raw <script>.
func TestRenderJSON_SanitizeAlwaysOn(t *testing.T) {
	resp := mustParse(t, RenderJSON(reqJSON(t, "# Title\n\n<script>alert(1)</script>", nil)))
	if resp.Output == nil {
		t.Fatal("nil output")
	}
	if strings.Contains(resp.Output.HTML, "<script") {
		t.Errorf("built-in sanitize policy should strip <script>, got: %q", resp.Output.HTML)
	}
}

// Test-list case 5: malformed input and unknown envelopeVersion each fold into a
// well-formed error envelope (error non-empty, output null) -- never nil/empty.
func TestRenderJSON_ErrorEnvelopes(t *testing.T) {
	t.Run("malformed request JSON", func(t *testing.T) {
		resp := mustParse(t, RenderJSON([]byte("not json")))
		if resp.Error == "" {
			t.Error("malformed request should yield a non-empty error")
		}
		if resp.Output != nil {
			t.Errorf("malformed request should yield null output, got %#v", resp.Output)
		}
		if resp.EnvelopeVersion != EnvelopeVersion {
			t.Errorf("error envelope should still carry envelopeVersion %q, got %q", EnvelopeVersion, resp.EnvelopeVersion)
		}
	})

	t.Run("unknown envelopeVersion", func(t *testing.T) {
		resp := mustParse(t, RenderJSON([]byte(`{"envelopeVersion":"eden-press.capi/v99","markdown":"# Hi"}`)))
		if resp.Error == "" {
			t.Error("unknown envelopeVersion should yield a non-empty error")
		}
		if resp.Output != nil {
			t.Errorf("unknown envelopeVersion should yield null output, got %#v", resp.Output)
		}
	})
}

// Test-list case 6: a render panic is RECOVERED into an error envelope and never
// escapes toward the C/Dart host.
//
// press/math already recovers the known go-latex-panicking constructs INTERNALLY
// (STATE.md: "fallback wraps recover() and degrades"), so a real heavy construct
// no longer panics up through press.Render -- it degrades to a well-formed
// success envelope. Case 6 therefore has two halves: (a) a deterministic proof
// that RenderJSON's own recover guard catches a panicking render, driven through
// the renderFn seam (the same package-var seam idiom press.go uses for
// parseWithEngine); and (b) the real heavy construct still yields a well-formed
// envelope with no panic escaping the process.
func TestRenderJSON_RecoversRenderPanic(t *testing.T) {
	t.Run("deterministic: a panicking render folds into an error envelope", func(t *testing.T) {
		orig := renderFn
		defer func() { renderFn = orig }()
		renderFn = func(md string, opts press.Options) (press.Output, error) {
			panic("simulated go-latex math panic")
		}

		resp := mustParse(t, RenderJSON([]byte(`{"markdown":"# anything"}`)))
		if resp.Error == "" {
			t.Error("a recovered render panic must produce a non-empty error")
		}
		if resp.Output != nil {
			t.Errorf("a recovered render panic must produce null output, got %#v", resp.Output)
		}
	})

	t.Run("real heavy math construct yields a well-formed envelope, no escape", func(t *testing.T) {
		// $$\begin{aligned}...$$ is a known go-latex-panicking construct; the
		// surrounding go test process must not crash and the result must parse.
		resp := mustParse(t, RenderJSON(reqJSON(t, "$$\\begin{aligned} a &= b \\\\ c &= d \\end{aligned}$$", nil)))
		if resp.EnvelopeVersion != EnvelopeVersion {
			t.Errorf("envelopeVersion = %q, want %q", resp.EnvelopeVersion, EnvelopeVersion)
		}
		// Either a degraded success (press/math recovered internally) or an
		// error envelope is acceptable -- the invariant is a well-formed,
		// non-crashing JSON boundary, which mustParse already asserted.
	})
}
