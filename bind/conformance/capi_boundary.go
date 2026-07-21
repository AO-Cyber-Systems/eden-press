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

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

// press_render_fn / press_free_fn are the C function-pointer TYPES the two
// dlsym'd symbols resolve to. Go cannot call a raw void* directly, so these
// tiny trampolines exist purely to give the dlopen'd address a callable C
// type -- mirroring, at the ABI level, what dart:ffi's
// NativeFunction<...>.asFunction() does for a real Dart client (07-04
// embedded_context's illustrative cgo dlopen shape).
typedef char *(*press_render_fn)(char *);
typedef void  (*press_free_fn)(char *);

static char *call_press_render(void *fn, char *req) {
    return ((press_render_fn)fn)(req);
}

static void call_press_free(void *fn, char *p) {
    ((press_free_fn)fn)(p);
}
*/
import "C"

import (
	"fmt"
	"os"
	"sync"
	"unsafe"
)

// hostLibPath is the host-arch c-shared artifact scripts/build-capi-host.sh
// produces. Its filename is the literal "libpress.so" on every OS -- Go
// honors an explicit `go build -o` name verbatim -- so this path is stable
// regardless of platform. bind/conformance and bind/capi are siblings under
// bind/, so one ".." reaches it.
const hostLibPath = "../capi/build/host/libpress.so"

// hostLibAvailable reports whether the host-arch libpress.so has been built
// (scripts/build-capi-host.sh), letting tests SKIP cleanly (07-04
// error_recovery) rather than failing on a machine that hasn't built it.
func hostLibAvailable() bool {
	_, err := os.Stat(hostLibPath)
	return err == nil
}

var (
	dlOnce    sync.Once
	dlErr     error
	dlHandle  unsafe.Pointer
	renderSym unsafe.Pointer
	freeSym   unsafe.Pointer
)

// openHostLib dlopen's hostLibPath and dlsym's PressRender/PressFree EXACTLY
// once per test binary, caching the handle and both resolved symbols --
// mirroring how a real Dart client resolves the two natives once at load
// time via DynamicLibrary.open + .lookup, not per render call.
func openHostLib() error {
	dlOnce.Do(func() {
		cPath := C.CString(hostLibPath)
		defer C.free(unsafe.Pointer(cPath))

		dlHandle = C.dlopen(cPath, C.RTLD_NOW)
		if dlHandle == nil {
			dlErr = fmt.Errorf("capi_boundary: dlopen(%s) failed: %s", hostLibPath, C.GoString(C.dlerror()))
			return
		}

		renderName := C.CString("PressRender")
		defer C.free(unsafe.Pointer(renderName))
		renderSym = C.dlsym(dlHandle, renderName)
		if renderSym == nil {
			dlErr = fmt.Errorf("capi_boundary: dlsym(PressRender) failed: %s", C.GoString(C.dlerror()))
			return
		}

		freeName := C.CString("PressFree")
		defer C.free(unsafe.Pointer(freeName))
		freeSym = C.dlsym(dlHandle, freeName)
		if freeSym == nil {
			dlErr = fmt.Errorf("capi_boundary: dlsym(PressFree) failed: %s", C.GoString(C.dlerror()))
			return
		}
	})
	return dlErr
}

// pressRenderCalls / pressFreeCalls count invocations across a test run so
// Test-list case 5 (capi_boundary_test.go) can assert PressFree is called on
// EVERY PressRender return -- no leaked C-heap output across the subset
// loop. Tests in this package run sequentially (no t.Parallel), so a plain
// int counter is safe without additional synchronization.
var (
	pressRenderCalls int
	pressFreeCalls   int
)

// callPressRender performs the EXACT PressRender/PressFree round-trip
// dart:ffi performs, exercising the FULL memory-ownership contract
// bind/capi/capi.go documents:
//  1. Go allocates the INPUT C string and OWNS/frees it (defer C.free) --
//     mirroring Dart owning and freeing its own input buffer.
//  2. The dlopen'd PressRender is invoked through the C trampoline, returning
//     a malloc'd OUTPUT C string Go does not yet own.
//  3. C.GoString COPIES the output into Go memory (mirroring Dart's
//     toDartString copy) -- the C pointer is not retained past this call.
//  4. The dlopen'd PressFree releases that C-heap output -- NEVER C.free --
//     completing the exact ownership contract the real C ABI requires.
func callPressRender(reqJSON string) (string, error) {
	if err := openHostLib(); err != nil {
		return "", err
	}

	cReq := C.CString(reqJSON)
	defer C.free(unsafe.Pointer(cReq))

	cResp := C.call_press_render(renderSym, cReq)
	pressRenderCalls++
	if cResp == nil {
		return "", fmt.Errorf("capi_boundary: PressRender returned NULL")
	}

	resp := C.GoString(cResp)
	C.call_press_free(freeSym, cResp)
	pressFreeCalls++

	return resp, nil
}
