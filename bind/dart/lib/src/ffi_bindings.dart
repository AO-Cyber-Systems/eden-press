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

/// dart:ffi typedefs and the memory-owning round trip for the 07-01 C ABI
/// (`bind/capi/capi.go`): `PressRender(const char*) -> char*` and
/// `PressFree(char*)`.
///
/// **Ownership contract (load-bearing -- do not "simplify"):**
/// - The input buffer is allocated by Dart via `toNativeUtf8` inside an
///   `Arena` (`package:ffi`'s `using`). Dart owns it and the arena frees it
///   automatically when the callback returns -- Go only ever reads it via
///   `C.GoString`, which copies, so Go never touches Dart's allocator.
/// - The returned pointer is Go/C-heap memory, allocated with `C.CString`
///   inside `PressRender`. Dart must copy it out with `toDartString()` and
///   then hand the *exact same pointer* back to the exported `PressFree`,
///   which calls `C.free` on it. Dart's own allocator (`calloc.free` /
///   arena release) must NEVER be called on this pointer -- it did not come
///   from Dart's allocator and freeing it that way is undefined behavior.
library;

import 'dart:ffi';

import 'package:ffi/ffi.dart';

typedef _PressRenderNative = Pointer<Utf8> Function(Pointer<Utf8> request);
typedef _PressRenderDart = Pointer<Utf8> Function(Pointer<Utf8> request);
typedef _PressFreeNative = Void Function(Pointer<Utf8> ptr);
typedef _PressFreeDart = void Function(Pointer<Utf8> ptr);

/// Symbol names exported by `bind/capi/capi.go` (`//export PressRender`,
/// `//export PressFree`). Kept as constants so native_loader.dart and any
/// future ffigen-generated bindings stay in lockstep with the C ABI.
const String pressRenderSymbol = 'PressRender';
const String pressFreeSymbol = 'PressFree';

/// Thin wrapper around the two looked-up native symbols, exposing a single
/// String-in/String-out `renderJson` that performs the full arena-scoped
/// memory round trip described above.
class NativeRenderBindings {
  NativeRenderBindings(DynamicLibrary lib)
    : _pressRender = lib.lookupFunction<_PressRenderNative, _PressRenderDart>(
        pressRenderSymbol,
      ),
      _pressFree = lib.lookupFunction<_PressFreeNative, _PressFreeDart>(
        pressFreeSymbol,
      );

  final _PressRenderDart _pressRender;
  final _PressFreeDart _pressFree;

  /// Renders [requestJson] (an `eden-press.capi/v1` request envelope) through
  /// PressRender, returning the raw response envelope JSON string.
  String renderJson(String requestJson) {
    return using((Arena arena) {
      final Pointer<Utf8> requestPtr = requestJson.toNativeUtf8(
        allocator: arena,
      );
      final Pointer<Utf8> responsePtr = _pressRender(requestPtr);
      try {
        return responsePtr.toDartString();
      } finally {
        // responsePtr is Go/C-heap memory -- free it via the exported
        // PressFree, never via the arena or calloc.
        _pressFree(responsePtr);
      }
    });
  }
}
