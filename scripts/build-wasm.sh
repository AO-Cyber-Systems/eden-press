#!/usr/bin/env bash
# Copyright (c) 2026 AO Cyber Systems
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#
# SPDX-License-Identifier: MIT

# build-wasm.sh compiles the DART-03 web front door (bind/wasm) into
# bind/wasm/press.wasm with STANDARD Go (GOOS=js GOARCH=wasm, CGO_ENABLED=0 --
# cgo is unavailable on the js port) and copies the CURRENT toolchain's
# wasm_exec.js loader next to it, pinning the two together.
#
# It uses the Go 1.24+ loader path $(go env GOROOT)/lib/wasm/wasm_exec.js -- NOT
# the pre-1.24 misc/wasm/ path (RESEARCH State of the Art). press.wasm is a build
# product (gitignored via *.wasm); the copied wasm_exec.js IS checked in so the
# version-pin has a tracked baseline. This is NOT TinyGo: standard Go is the
# resolved decision (TinyGo tracks none of the press dependency chain).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

WASM_OUT="bind/wasm/press.wasm"
WASM_EXEC_DST="bind/wasm/wasm_exec.js"
WASM_EXEC_SRC="$(go env GOROOT)/lib/wasm/wasm_exec.js"

echo "build-wasm: GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./bind/wasm -> $WASM_OUT"
GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o "$WASM_OUT" ./bind/wasm
test -f "$WASM_OUT" || { echo "FAIL: $WASM_OUT not produced" >&2; exit 1; }

test -f "$WASM_EXEC_SRC" || {
  echo "FAIL: $WASM_EXEC_SRC not found -- expected the Go 1.24+ lib/wasm loader path" >&2
  exit 1
}
cp "$WASM_EXEC_SRC" "$WASM_EXEC_DST"
echo "build-wasm: pinned wasm_exec.js from $WASM_EXEC_SRC -> $WASM_EXEC_DST"

# Fail loudly if the just-copied loader somehow does not match the toolchain
# (RESEARCH Pitfall 2: press.wasm and wasm_exec.js must share a producing version).
bash scripts/check-wasm-exec-version.sh

echo "PASS: press.wasm + toolchain-matched wasm_exec.js built with $(go version)."
