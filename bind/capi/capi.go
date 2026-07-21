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

// #include <stdlib.h>
import "C"

import (
	"unsafe"

	"github.com/AO-Cyber-Systems/eden-press/bind/capi/core"
)

// MEMORY-OWNERSHIP CONTRACT across the FFI boundary (RESEARCH §1,
// pkg.go.dev/cmd/cgo) -- the two allocators are NEVER crossed:
//
//  1. INPUT buffer: the caller (Dart) allocates and OWNS the NUL-terminated
//     UTF-8 request buffer, and the caller frees it. PressRender only READS it
//     via C.GoString, which copies the bytes into Go memory; Go keeps no
//     reference to the caller's pointer after the call returns ("C may not keep
//     a copy of a Go pointer after the call returns", and symmetrically Go
//     keeps no C pointer).
//
//  2. OUTPUT buffer: PressRender allocates the response on the C heap via
//     C.CString (malloc). That pointer OUTLIVES the call and is invisible to
//     Go's garbage collector, so the caller must copy the bytes out (Dart's
//     toDartString copies) and then hand the SAME pointer back to PressFree.
//
//  3. FREEING: the caller frees the Go-allocated OUTPUT by calling PressFree
//     (which calls C.free) -- NEVER with Dart's own allocator/free -- and frees
//     the INPUT with its own allocator. Mismatching the two corrupts the heap.
//
// PressRender is the sole C-ABI render entry point: it copies the C request in,
// runs the pure-Go core.RenderJSON seam (which never panics out and never returns
// nil/empty), and returns a freshly malloc'd C string the caller owns.
//
//export PressRender
func PressRender(cmd *C.char) *C.char {
	in := C.GoString(cmd)              // copy C -> Go (caller still owns cmd)
	out := core.RenderJSON([]byte(in)) // pure-Go JSON boundary, always well-formed
	return C.CString(string(out))      // malloc'd C heap, outlives this call
}

// PressFree releases a pointer previously returned by PressRender, and ONLY such
// a pointer. It is the Go side of the ownership contract above: Go allocated the
// output on the C heap, so Go frees it here via C.free -- the caller must not
// free it with any other allocator.
//
//export PressFree
func PressFree(p *C.char) {
	C.free(unsafe.Pointer(p))
}

// main is required and intentionally empty: c-shared / c-archive buildmodes
// demand a `package main` with a main function even though the artifact is a
// library, not an executable.
func main() {}
