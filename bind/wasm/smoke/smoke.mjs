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

// smoke.mjs is the executable boundary gate for DART-03: it loads the compiled
// bind/wasm/press.wasm under Node via the version-pinned wasm_exec.js, calls the
// registered globalThis.pressRender with a JSON request envelope, and asserts the
// parsed response. This proves the compiled wasm artifact answers through the
// SAME JSON entrypoint (bind/capi/core.RenderJSON, 07-01) that 07-05's Dart web
// loader will call via dart:js_interop -- with ZERO cgo on this target.
//
// Run it AFTER scripts/build-wasm.sh (which produces press.wasm + the pinned
// wasm_exec.js). Exits non-zero on any assertion failure.
//
// The Go 1.24+ lib/wasm/wasm_exec.js needs globalThis.crypto / performance /
// TextEncoder / TextDecoder; Node's LTS provides all of these natively (host
// Node is v24), so no polyfill shim is required here. If a future older Node
// throws a missing-global, add the minimal polyfill at the top of this file.

import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const WASM_DIR = join(HERE, "..");
const WASM_EXEC = join(WASM_DIR, "wasm_exec.js");
const WASM_BIN = join(WASM_DIR, "press.wasm");
const ENVELOPE_VERSION = "eden-press.capi/v1";

function fail(msg) {
  console.error(`smoke FAIL: ${msg}`);
  process.exit(1);
}

// wasm_exec.js is a classic script (not an ES module) that assigns
// globalThis.Go. Read it and execute it in this process so globalThis.Go exists.
try {
  const src = readFileSync(WASM_EXEC, "utf8");
  new Function(src)();
} catch (e) {
  fail(`could not load wasm_exec.js at ${WASM_EXEC}: ${e.message}`);
}
if (typeof globalThis.Go !== "function") {
  fail("wasm_exec.js did not define globalThis.Go -- wrong/empty loader");
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
// select{}, handing control back here. Do NOT await it -- it resolves only when
// the Go program exits, which (by design) it never does.
go.run(instance);
// Yield one macrotask so the Go scheduler settles at select{} and the export is
// installed before we call it.
await new Promise((resolve) => setTimeout(resolve, 0));

if (typeof globalThis.pressRender !== "function") {
  fail(
    "pressRender is not a function -- main() likely lacks select{} (Task 1) " +
      "or wasm_exec.js is stale (run scripts/check-wasm-exec-version.sh)",
  );
}

function render(reqObj) {
  const raw = globalThis.pressRender(JSON.stringify(reqObj));
  if (typeof raw !== "string") {
    fail(`pressRender returned ${typeof raw}, expected a JSON string`);
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (e) {
    fail(`pressRender returned non-JSON (${e.message}): ${raw.slice(0, 200)}`);
  }
  if (parsed.envelopeVersion !== ENVELOPE_VERSION) {
    fail(
      `unexpected envelopeVersion ${JSON.stringify(parsed.envelopeVersion)}, ` +
        `want ${ENVELOPE_VERSION}`,
    );
  }
  return parsed;
}

// Assertion 1: `# Hi` renders an <h1 through the JSON entrypoint -- the compiled
// wasm answers the same envelope shape the native C ABI does. Note the wire key
// is output.HTML (capitalized): press.Output carries no json tags, so its fields
// cross as Go-default keys (verified against 07-01's wire-contract test).
{
  const parsed = render({ markdown: "# Hi" });
  if (parsed.error) fail(`"# Hi" returned an error envelope: ${parsed.error}`);
  const html = parsed?.output?.HTML;
  if (typeof html !== "string") {
    fail(
      `"# Hi": response missing output.HTML; output keys = ` +
        JSON.stringify(Object.keys(parsed.output ?? {})),
    );
  }
  if (!html.includes("<h1")) {
    fail(`"# Hi": output.HTML missing <h1; got: ${html.slice(0, 200)}`);
  }
  console.log('smoke ok: "# Hi" -> output.HTML contains <h1');
}

// Assertion 2: a battery case (~~struck~~) round-trips as press's <s> override
// -- NOT bare-CommonMark literal ~~ (which would carry no strike tag at all) and
// NOT goldmark GFM's default <del>. This proves the compiled wasm carries the
// press batteries, not bare CommonMark.
{
  const parsed = render({ markdown: "~~struck~~" });
  if (parsed.error) fail(`"~~struck~~" returned an error envelope: ${parsed.error}`);
  const html = parsed?.output?.HTML ?? "";
  if (!html.includes("<s>")) {
    fail(
      `"~~struck~~": output.HTML missing <s> (strikethrough battery not carried); ` +
        `got: ${html.slice(0, 200)}`,
    );
  }
  if (html.includes("<del>")) {
    fail(
      `"~~struck~~": output.HTML has goldmark's default <del>, expected press <s> ` +
        `override; got: ${html.slice(0, 200)}`,
    );
  }
  console.log('smoke ok: "~~struck~~" -> output.HTML contains <s> (press strikethrough battery)');
}

console.log(
  "smoke PASS: press.wasm answers through the JSON entrypoint (# Hi + strikethrough battery).",
);
process.exit(0);
