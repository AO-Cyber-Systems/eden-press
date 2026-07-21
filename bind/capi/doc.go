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

// Command capi is the JSON-in / JSON-out C-ABI front door of the DART-01 core: a
// cgo `package main` exposing //export PressRender / PressFree, built as a
// c-shared/c-archive library (see scripts/build-capi-host.sh) so a native host
// -- ultimately 07-05's Dart FFI client -- renders a Marp deck without mirroring
// any Go struct.
//
// The pure-Go marshalling seam lives in subpackage bind/capi/core
// (core.RenderJSON): JSON request -> press.Render -> JSON response. That seam is
// deliberately cgo-free so the SAME function also compiles into the standard-Go
// syscall/js wasm front door (bind/wasm, TRD 07-02) -- cgo and
// GOOS=js GOARCH=wasm are mutually exclusive build modes. This package (capi.go)
// is the ONLY cgo in the objective; it is thin C-string glue over core.RenderJSON
// and documents the FFI memory-ownership contract at the export site.
package main
