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

# check-wasm-exec-version.sh is the RESEARCH Pitfall 2 guard: the wasm_exec.js
# loader and the compiler that produced press.wasm MUST be the same Go version.
# A drifted loader fails silently at runtime with "undefined function" / a broken
# import object, so this script diffs the checked-in bind/wasm/wasm_exec.js
# against the active toolchain's $(go env GOROOT)/lib/wasm/wasm_exec.js and exits
# non-zero on any difference.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

CHECKED_IN="bind/wasm/wasm_exec.js"
TOOLCHAIN="$(go env GOROOT)/lib/wasm/wasm_exec.js"

test -f "$CHECKED_IN" || {
  echo "FAIL: $CHECKED_IN missing -- run scripts/build-wasm.sh to produce and pin it" >&2
  exit 1
}
test -f "$TOOLCHAIN" || {
  echo "FAIL: $TOOLCHAIN not found -- expected the Go 1.24+ lib/wasm loader path" >&2
  exit 1
}

if ! diff -q "$CHECKED_IN" "$TOOLCHAIN" >/dev/null 2>&1; then
  echo "FAIL: $CHECKED_IN drifted from the active Go toolchain ($(go version))." >&2
  echo "      The loader and the compiler that produced press.wasm must match" >&2
  echo "      (RESEARCH Pitfall 2). Re-run scripts/build-wasm.sh to re-pin." >&2
  exit 1
fi

echo "PASS: $CHECKED_IN matches the active Go toolchain ($(go version))."
