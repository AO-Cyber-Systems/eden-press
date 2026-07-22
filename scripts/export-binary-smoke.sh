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

# export-binary-smoke.sh proves the 05.1-01 turnkey CLI path end-to-end: it
# builds cmd/eden-press-export from source, then drives the COMPILED BINARY
# (not the convert/ package directly -- that's what `make export-test`
# already covers) through a real pdf export and a real png export of a
# hand-built fixture deck, asserting the output files carry the expected
# magic bytes.
#
# SELF-GATING on the binary's own documented exit-code contract
# (cmd/eden-press-export/main.go): exit code 3 means "no Chrome/Chromium
# discoverable" -- this script treats that as a CLEAN SKIP (prints a
# documented SKIP message, exits 0) rather than a failure, mirroring the
# t.Skip discipline every live-Chrome test in this module already follows.
# This is the ONE seam that makes the script safe to run both in the pinned
# no-system-Chrome CI container (.github/workflows/ci.yml's `export` job,
# where CHROME_PATH resolves the pinned headless-shell -- so the smoke
# actually PASSes there) and locally on a machine with no Chrome at all
# (where it no-ops instead of failing). The script deliberately has NO Chrome
# -detection logic of its own -- the binary's exit code IS the signal.
#
# Wired into the CI export job via `make export-binary-smoke`, run
# immediately after `make export-test` in the SAME pinned-headless-shell
# container (see .github/workflows/ci.yml).
set -euo pipefail

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "export-binary-smoke: building eden-press-export..."
go build -o "$tmp/eden-press-export" ./cmd/eden-press-export

cat >"$tmp/smoke.md" <<'DECK'
---
marp: true
---

# Smoke slide one

Plain text.

---

# Smoke slide two

More text.
DECK

# --- PDF ---
echo "export-binary-smoke: running --format pdf..."
set +e
"$tmp/eden-press-export" "$tmp/smoke.md" -o "$tmp/out.pdf" --format pdf
rc=$?
set -e

if [ "$rc" -eq 3 ]; then
	echo "SKIP: no Chrome discoverable (eden-press-export exited 3) -- export-binary-smoke is a documented no-op in this environment"
	exit 0
fi
if [ "$rc" -ne 0 ]; then
	echo "FAIL: eden-press-export --format pdf exited $rc (want 0)" >&2
	exit 1
fi

head -c 5 "$tmp/out.pdf" | grep -q '%PDF-' || {
	echo "FAIL: $tmp/out.pdf is missing the %PDF- magic" >&2
	exit 1
}

# --- PNG ---
echo "export-binary-smoke: running --format png..."
mkdir -p "$tmp/pngs"
"$tmp/eden-press-export" "$tmp/smoke.md" -o "$tmp/pngs" --format png

count="$(find "$tmp/pngs" -name '*.png' | wc -l | tr -d ' ')"
if [ "$count" -lt 1 ]; then
	echo "FAIL: --format png produced no .png files in $tmp/pngs" >&2
	exit 1
fi

first="$(find "$tmp/pngs" -name '*.png' | sort | head -n1)"
head -c 4 "$first" | grep -q $'\x89PNG' || {
	echo "FAIL: $first is missing the PNG magic" >&2
	exit 1
}

echo "PASS: export-binary-smoke -- pdf (%PDF-) + ${count} png(s) via eden-press-export"
