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

# build-capi-host.sh compiles the DART-01 cgo shim (bind/capi) into a host-arch
# c-shared library as a buildability smoke test AND as the reusable artifact
# 07-04's boundary-conformance lane loads. It proves the //export PressRender /
# PressFree ABI is a valid, linkable C library whose generated header declares
# both exports.
#
# The output name is the LITERAL "libpress.so" on every OS: Go honors an explicit
# `-o` name verbatim (it does not rewrite the suffix to .dylib on macOS), so
# 07-04/07-05 can look up one stable filename regardless of platform.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

OUT="bind/capi/build/host"
LIB="$OUT/libpress.so"
HDR="$OUT/libpress.h"

echo "build-capi-host: c-shared build of ./bind/capi -> $LIB"
mkdir -p "$OUT"

CGO_ENABLED=1 go build -buildmode=c-shared -o "$LIB" ./bind/capi

# The artifact and its generated header must both exist, and the header must
# declare the two C-ABI exports downstream binds link against.
test -f "$LIB" || { echo "FAIL: $LIB not produced" >&2; exit 1; }
test -f "$HDR" || { echo "FAIL: $HDR not produced" >&2; exit 1; }
grep -q 'PressRender' "$HDR" || { echo "FAIL: $HDR does not declare PressRender" >&2; exit 1; }
grep -q 'PressFree' "$HDR" || { echo "FAIL: $HDR does not declare PressFree" >&2; exit 1; }

echo "PASS: libpress.so + libpress.h built; header declares PressRender/PressFree."
