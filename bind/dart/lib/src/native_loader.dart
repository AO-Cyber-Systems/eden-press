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

/// Native (Android/iOS/desktop) backend selected by the conditional import in
/// `eden_press.dart` when `dart.library.js_interop` is unavailable at compile
/// time (i.e. everywhere except Web).
library;

import 'dart:ffi';
import 'dart:io';

import 'ffi_bindings.dart';

/// Opens the platform-appropriate native library for the 07-01 C ABI.
///
/// - **Android**: the shared object is dynamically loaded by soname
///   (`DynamicLibrary.open('libpress.so')`); the per-ABI `.so` produced by
///   07-03's `build-android.sh` is vendored into
///   `android/src/main/jniLibs/<abi>/libpress.so`.
/// - **iOS** (and macOS): the static archive is linked directly into the app
///   binary at build time via the plugin's podspec `vendored_libraries`
///   (07-03's `build-ios.sh` `EdenPress.xcframework`), so there is nothing to
///   `dlopen()` by name -- the symbols are already resident in the process
///   image. This is *why* this package is a `plugin_ffi` template rather than
///   the newer `package_ffi` template: only `plugin_ffi` supports this
///   static-link-on-iOS pattern (07-RESEARCH.md Sec 4).
DynamicLibrary _openNativeLibrary() {
  if (Platform.isAndroid) {
    return DynamicLibrary.open('libpress.so');
  }
  if (Platform.isIOS || Platform.isMacOS) {
    return DynamicLibrary.process();
  }
  if (Platform.isWindows) {
    // Desktop targets are outside DART-02's Android/iOS scope; best-effort
    // fallback for local development only.
    return DynamicLibrary.open('press.dll');
  }
  // Linux and any other platform: best-effort fallback for local development.
  return DynamicLibrary.open('libpress.so');
}

/// Lazily-initialized native bindings. Dart top-level variables initialize
/// on first access, not at import time, so merely importing this file (e.g.
/// transitively through `eden_press.dart`) never triggers a native library
/// load -- only calling [renderJson] does.
final NativeRenderBindings _bindings = NativeRenderBindings(
  _openNativeLibrary(),
);

/// Renders [requestJson] through the native FFI round trip. See
/// `ffi_bindings.dart` for the full memory-ownership contract.
String renderJson(String requestJson) => _bindings.renderJson(requestJson);
