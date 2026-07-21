#!/usr/bin/env node
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

// wasm_runner.mjs is the DART-05 WASM boundary driver: it loads the compiled
// bind/wasm/press.wasm under Node via the version-pinned wasm_exec.js (the
// SAME pair bind/wasm/smoke/smoke.mjs already exercises), reads ONE JSON
// request envelope (the bind/capi/core "eden-press.capi/v1" contract) from
// stdin, calls the registered globalThis.pressRender with it -- the SAME
// JS-callable export bind/wasm/main.go registers, the one the Dart web
// loader will call via dart:js_interop -- and writes the JSON response
// envelope to stdout verbatim.
//
// One process handles ONE request (simple + robust, per the TRD's own task
// action doc): bind/conformance/wasm_boundary_test.go execs this script once
// per subset case via exec.Command, piping the request to stdin and reading
// the response from stdout.
//
// Exits non-zero (with a stderr message, never touching stdout) on any setup
// failure -- missing press.wasm/wasm_exec.js, a missing pressRender export,
// or empty stdin -- so the Go caller surfaces a clear error rather than
// silently parsing garbage as a JSON response.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
// bind/conformance and bind/wasm are siblings under bind/.
const WASM_DIR = join(HERE, "..", "wasm");
const WASM_EXEC = join(WASM_DIR, "wasm_exec.js");
const WASM_BIN = join(WASM_DIR, "press.wasm");

function fail(msg) {
  process.stderr.write(`wasm_runner FAIL: ${msg}\n`);
  process.exit(1);
}

function readStdin() {
  try {
    return readFileSync(0, "utf8");
  } catch (e) {
    fail(`could not read stdin: ${e.message}`);
  }
}

const reqRaw = readStdin();
if (!reqRaw || reqRaw.trim() === "") {
  fail("empty stdin -- expected a JSON request envelope on stdin");
}

// wasm_exec.js is a classic (non-module) script that assigns globalThis.Go --
// the same load technique bind/wasm/smoke/smoke.mjs uses.
try {
  const src = readFileSync(WASM_EXEC, "utf8");
  new Function(src)();
} catch (e) {
  fail(`could not load wasm_exec.js at ${WASM_EXEC}: ${e.message}`);
}
if (typeof globalThis.Go !== "function") {
  fail("wasm_exec.js did not define globalThis.Go -- wrong or empty loader file");
}

let wasmBytes;
try {
  wasmBytes = readFileSync(WASM_BIN);
} catch {
  fail(`press.wasm not found at ${WASM_BIN} -- run scripts/build-wasm.sh first`);
}

const go = new globalThis.Go();
let instance;
try {
  ({ instance } = await WebAssembly.instantiate(wasmBytes, go.importObject));
} catch (e) {
  fail(`WebAssembly.instantiate(press.wasm) failed: ${e.message}`);
}

// go.run() starts the Go program: main() registers pressRender then blocks on
// select{}. Do NOT await it -- it resolves only when the Go program exits,
// which (by design) it never does (mirrors smoke.mjs's own comment).
go.run(instance);
// Yield one macrotask so the Go scheduler settles at select{} before the
// export is called.
await new Promise((resolve) => setTimeout(resolve, 0));

if (typeof globalThis.pressRender !== "function") {
  fail(
    "pressRender is not a function -- main() likely lacks select{} or " +
      "wasm_exec.js is stale (run scripts/check-wasm-exec-version.sh)",
  );
}

const raw = globalThis.pressRender(reqRaw);
if (typeof raw !== "string") {
  fail(`pressRender returned ${typeof raw}, expected a JSON string`);
}
process.stdout.write(raw);
process.exit(0);
