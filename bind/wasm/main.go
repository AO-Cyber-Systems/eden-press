//go:build js && wasm

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

// Command wasm is the DART-03 WEB front door: a standard-Go
// GOOS=js GOARCH=wasm module that registers a JS-global `pressRender` calling
// the SAME pure-Go bind/capi/core.RenderJSON seam (07-01) the cgo shim
// (bind/capi) calls. It is a SEPARATE package main from bind/capi because cgo
// and GOOS=js GOARCH=wasm are mutually exclusive build modes (RESEARCH
// Pitfall 1): the browser target cannot link the cgo `//export` glue, so the web
// boundary re-exposes the identical JSON entrypoint over syscall/js instead.
//
// Compiled with standard Go (the resolved decision -- NOT TinyGo; TinyGo's
// partial reflection is a correctness risk against goldmark/yaml.v3, and no
// TinyGo precedent exists for the full press dependency chain). The whole file
// is gated behind `//go:build js && wasm`, so it is excluded from the host
// `go build ./...` / `go vet ./...` / `go test ./...` builds and only compiles
// under GOOS=js GOARCH=wasm.
//
// Build it with scripts/build-wasm.sh (which also copies the version-pinned
// wasm_exec.js loader); exercise the compiled artifact with
// bind/wasm/smoke/smoke.mjs under Node.
package main

import (
	"syscall/js"

	"github.com/AO-Cyber-Systems/eden-press/bind/capi/core"
)

// main registers pressRender on the JS global object, then blocks forever with
// `select {}`. The block is load-bearing: if main returned, the Go program would
// exit and the registered pressRender would be torn down, so JS callers would see
// "pressRender is not a function" (RESEARCH Pitfall, mirrored in smoke recovery).
func main() {
	js.Global().Set("pressRender", js.FuncOf(pressRender))
	select {}
}

// pressRender is the JS-callable export. It reads the JSON request string from
// args[0], hands it verbatim to core.RenderJSON (the ONE marshalling seam --
// there is NO second marshalling implementation here), and returns the JSON
// response string to JS.
//
// core.RenderJSON already folds malformed input, unknown envelope versions, and
// render panics into a well-formed JSON error envelope and never returns
// nil/empty, so this layer needs no extra error handling -- that uniformity is
// exactly the payoff of the shared seam. The only guard here is arity: with no
// argument there is nothing to render, so we return an error envelope string
// rather than indexing an empty args slice.
func pressRender(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return `{"envelopeVersion":"` + core.EnvelopeVersion + `","error":"pressRender: missing request argument"}`
	}
	req := args[0].String()
	resp := core.RenderJSON([]byte(req))
	return string(resp)
}
