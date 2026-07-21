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

/// Web backend selected by the conditional import in `eden_press.dart` when
/// `dart.library.js_interop` is available at compile time (Web only).
///
/// Calls the `pressRender` global registered on `globalThis` by the 07-02
/// wasm module (`bind/wasm/main.go`'s `js.Global().Set("pressRender", ...)`)
/// via `dart:js_interop` -- **never** `dart:html` or `package:js` (both
/// deprecated) and never a DOM/webview bridge. This keeps the entire render
/// path JS-free from the Dart application's point of view: the only
/// JavaScript involved is the wasm host glue itself, invoked through a
/// single typed static-interop call.
library;

import 'dart:js_interop';

/// Static interop binding to the `pressRender(request: string): string`
/// function that `bind/wasm/main.go` registers on `globalThis` via
/// `js.Global().Set("pressRender", js.FuncOf(pressRender))`.
@JS('pressRender')
external JSString _pressRender(JSString requestJson);

/// Renders [requestJson] through the wasm-registered `pressRender` global,
/// returning the raw response envelope JSON string.
String renderJson(String requestJson) => _pressRender(requestJson.toJS).toDart;
