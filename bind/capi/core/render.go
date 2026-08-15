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

// Package core is the pure-Go, platform-agnostic JSON marshalling seam of the
// DART-01 C-ABI core (RenderJSON): the SINGLE JSON-in / JSON-out boundary over
// press.Render that every front door calls -- the cgo shim (bind/capi, this
// objective's TRD 07-01), the syscall/js wasm shim (07-02), and, through the
// compiled artifacts, 07-05's Dart client.
//
// It contains NO `import "C"` and NO build tags on purpose: cgo and
// GOOS=js GOARCH=wasm are mutually exclusive build modes, so keeping the
// marshalling logic here (pure Go) lets the SAME function compile into both the
// cgo library and the standard-Go wasm module. The cgo `//export` glue lives one
// directory up in bind/capi (package main); this package is a normal importable
// library.
package core

import (
	"encoding/json"
	"fmt"

	"github.com/AO-Cyber-Systems/eden-press/press"

	// Blank import: registers profiles/paged via its init() side-effect.
	// press blank-imports only profiles/slides (press.go), so without this a
	// caller passing Profile: "paged" across the C/Dart boundary got
	// `unknown profile "paged"` -- the paged profile was unreachable from
	// the binding despite being fully implemented.
	//
	// It belongs HERE, in the pure-Go core, and not in bind/capi (package
	// main): this package is the one shared by BOTH the cgo library and the
	// GOOS=js GOARCH=wasm module, so one import serves both front doors.
	// Putting it in the cgo main package would leave wasm broken in exactly
	// the same way.
	_ "github.com/AO-Cyber-Systems/eden-press/profiles/paged"
)

// EnvelopeVersion is the frozen version tag stamped on every response and
// validated on every request (03-RESEARCH Open Question 2, RESOLVED in favor of
// a thin explicit capi-layer version, independent of Model.SchemaVersion). It is
// consumed unchanged by 07-04's boundary conformance harness and 07-05's Dart
// client, so it is a stability contract: bump it only on a breaking wire change.
const EnvelopeVersion = "eden-press.capi/v1"

// marshalFailFallback is the last-resort return so RenderJSON NEVER hands a
// nil/empty slice across the C/Dart boundary -- even if json.Marshal of the
// error envelope itself somehow fails. It is a hardcoded, already-valid JSON
// error envelope.
var marshalFailFallback = []byte(`{"envelopeVersion":"eden-press.capi/v1","error":"internal: response marshal failed"}`)

// renderFn is the seam to press.Render, held in a package-level variable ONLY so
// the load-bearing "no panic escapes the boundary" invariant is deterministically
// testable: render_test.go swaps in a panicking stub to prove renderOnce's
// recover guard folds a panic into an error envelope, without coupling the test
// to a fragile go-latex construct that Objective 8 may later stop panicking on.
// This mirrors press.go's identical parseWithEngine seam idiom; production
// behavior is byte-for-byte press.Render.
var renderFn = press.Render

// requestOptions is the JSON-serializable subset of press.Options carried on the
// wire. It deliberately OMITS press.Options.Sanitize (a *bluemonday.Policy, not
// JSON-serializable): a nil Sanitize selects the built-in always-on policy, so
// the wire never needs to -- and never can -- turn sanitization off.
type requestOptions struct {
	Theme          string `json:"theme"`
	Profile        string `json:"profile"`
	InlineSVG      bool   `json:"inlineSvg"`
	MathMode       string `json:"mathMode"`
	NoHighlight    bool   `json:"noHighlight"`
	HighlightStyle string `json:"highlightStyle"`
}

// request is the inbound envelope: an optional envelopeVersion, the markdown
// source, and the option subset. An empty envelopeVersion is tolerated (lenient
// first-cut client); a non-empty mismatch is rejected.
type request struct {
	EnvelopeVersion string         `json:"envelopeVersion"`
	Markdown        string         `json:"markdown"`
	Options         requestOptions `json:"options"`
}

// response is the outbound envelope. It carries the frozen EnvelopeVersion plus
// EXACTLY ONE of Output (success) or Error (failure). Output embeds *press.Output
// verbatim: press.Output has no json tags, so its shape (HTML, CSS, Model, Meta,
// Comments) crosses as Go-default keys -- Model itself is JSON-tagged and marshals
// for free (schemaVersion "eden-press.model/v1" included).
type response struct {
	EnvelopeVersion string        `json:"envelopeVersion"`
	Output          *press.Output `json:"output,omitempty"`
	Error           string        `json:"error,omitempty"`
}

// RenderJSON is the pure-Go JSON boundary: JSON request in, JSON response out. It
// is ALWAYS well-formed -- it folds a malformed request, an unsupported
// envelopeVersion, a press.Render error, and even a render panic into a JSON
// error envelope, and NEVER returns a nil/empty slice or lets a panic escape
// toward the C/Dart host. Every front door hands the returned bytes across its
// boundary unchanged, so the response is uniformly parseable on the far side.
func RenderJSON(cmd []byte) []byte {
	var req request
	if err := json.Unmarshal(cmd, &req); err != nil {
		return errorEnvelope("malformed request JSON: " + err.Error())
	}
	if req.EnvelopeVersion != "" && req.EnvelopeVersion != EnvelopeVersion {
		return errorEnvelope(fmt.Sprintf("unsupported envelopeVersion %q; this core speaks %q", req.EnvelopeVersion, EnvelopeVersion))
	}

	out, err := renderOnce(req)
	if err != nil {
		return errorEnvelope(err.Error())
	}
	return marshal(response{EnvelopeVersion: EnvelopeVersion, Output: &out})
}

// renderOnce maps the request options onto press.Options and calls press.Render
// EXACTLY ONCE, wrapped in a recover so a render panic (e.g. a go-latex-panicking
// math construct that slips past press/math's own recovery) degrades into a
// returned error rather than crossing the C ABI as a process crash. The recover
// wraps the render call directly, before any Output field is touched.
func renderOnce(req request) (out press.Output, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = press.Output{}
			err = fmt.Errorf("render panicked: %v", r)
		}
	}()

	opts := press.Options{
		Theme:          req.Options.Theme,
		Profile:        req.Options.Profile,
		InlineSVG:      req.Options.InlineSVG,
		MathMode:       req.Options.MathMode,
		NoHighlight:    req.Options.NoHighlight,
		HighlightStyle: req.Options.HighlightStyle,
		// Sanitize is deliberately left nil => the built-in always-on policy.
	}
	return renderFn(req.Markdown, opts)
}

// errorEnvelope builds a well-formed error response (output null, error set).
func errorEnvelope(msg string) []byte {
	return marshal(response{EnvelopeVersion: EnvelopeVersion, Error: msg})
}

// marshal serializes a response, falling back to marshalFailFallback so the
// return is never nil/empty and always valid JSON.
func marshal(r response) []byte {
	b, err := json.Marshal(r)
	if err != nil {
		return marshalFailFallback
	}
	return b
}
