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

// Package conformance proves DART-05: that the two compiled artifacts
// Objective 7 ships -- the syscall/js wasm module (bind/wasm, 07-02) and the
// cgo c-shared library (bind/capi, 07-01) -- answer IDENTICALLY to in-process
// press.Render when driven through the exact same JSON entrypoint
// (bind/capi/core.RenderJSON's "eden-press.capi/v1" envelope) a real Dart
// client uses. This is not a press-vs-Marp conformance check (that is other
// objectives' concern, exercised in-process by conformance/runner's existing
// goldmark RenderFuncs) -- it is a boundary-vs-in-process PARITY check,
// isolating "the JSON marshalling is lossless" from "press matches Marp".
//
// It reuses the existing engine-agnostic conformance seam unchanged:
// runner.RenderFunc / runner.RunCase / htmldiff.Equal (conformance/runner,
// conformance/htmldiff). This package supplies a THIRD and FOURTH RenderFunc
// implementation of that same seam -- one that execs a Node runner against
// the compiled press.wasm (wasm_boundary_test.go, via wasm_runner.mjs), and
// one that dlopen's the host-arch libpress.so and calls the real
// PressRender/PressFree C ABI (capi_boundary.go, capi_boundary_test.go) --
// instead of calling a goldmark-backed engine in-process.
//
// subset.go selects a small, hand-curated, battery-spanning slice of
// Objective 0's hand-seeded Marp golden corpus (conformance/corpus/cases/):
// the boundary under test is JSON marshalling of the WHOLE press.Output
// shape (HTML, CSS, Model, Meta, Comments), so the subset spans every
// battery press.Render composes -- strikethrough, emoji, highlight, math,
// autofit, sanitize -- plus plain CommonMark, not CommonMark alone.
//
// Both boundary tests are SKIP-guarded (t.Skip) when their compiled artifact
// is absent (press.wasm / bind/capi/build/host/libpress.so) or when a
// required tool (node) is unavailable, so a bare `go test ./...` on a machine
// that has not run scripts/build-wasm.sh / scripts/build-capi-host.sh does
// not hard-fail -- the TRD's own verify commands build both artifacts first.
package conformance
