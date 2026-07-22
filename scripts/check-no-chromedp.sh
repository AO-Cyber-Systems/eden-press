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

# check-no-chromedp.sh is the mechanical enforcement of OBJECTIVE.md success
# criterion 2 and the Objective-7-begin gate (criterion 4): the public render
# path (press/) and everything it composes (chase/, profiles/) must stay
# pure-Go with NO headless-browser dependency. It fails (exit 1) if chromedp
# appears anywhere in the transitive dependency closure of those three package
# trees -- so a consumer that imports ONLY press/ never pulls in a browser.
#
# 04.1-02 (Objective 4.1) extended this same gate to ./cmd/... -- the CLI's
# own "stays chromedp-free" property, previously only an unenforced
# intention, became mechanical too: the CLI's --format pptx wiring imports
# convert/pptx (stdlib OOXML, zero chromedp), so cmd/'s transitive closure
# stayed chromedp-free and the gate kept passing.
#
# 05.1-01 RE-SCOPES that ./cmd/... entry to ./cmd/eden-press/... : a NEW,
# SEPARATE Chrome-permitting binary, cmd/eden-press-export, now legitimately
# imports convert/ (and therefore chromedp) to drive PDF/PNG raster export.
# Scanning bare ./cmd/... would flag that legitimate import as a violation;
# narrowing TREES to ./cmd/eden-press/... keeps proving the CORE cli is
# chromedp-free while correctly not flagging the new export binary. The
# appended assertion block (below the core PASS) then proves the OTHER half
# of the boundary mechanically: cmd/eden-press-export builds, its closure
# DOES contain chromedp (it is the genuine raster exporter), and it is the
# ONLY cmd allowed to.
#
# Wired into CI (.github/workflows/ci.yml) beside the addlicense check and
# runnable locally via `make check-no-chromedp`.
set -euo pipefail

# The package trees whose transitive deps must be chromedp-free. press/ is the
# public surface; chase/ and profiles/ are everything it composes;
# ./cmd/eden-press/... is the CORE CLI consumer of press/ (04.1-02's
# CI-enforced invariant, re-scoped by 05.1-01 from bare ./cmd/... so the new
# Chrome-permitting cmd/eden-press-export -- asserted separately below -- is
# not flagged here). Asserting all of these (not just press/) proves the
# boundary holds one level down too.
TREES=("./press/..." "./chase/..." "./profiles/..." "./bind/..." "./cmd/eden-press/...")

echo "check-no-chromedp: scanning transitive deps of ${TREES[*]}"

# go list -deps prints the full transitive import closure, one package path per
# line. grep for any package whose path contains "chromedp".
if offending="$(go list -deps "${TREES[@]}" 2>/dev/null | grep -i 'chromedp' || true)"; [ -n "$offending" ]; then
	echo "FAIL: chromedp found in the press/chase/profiles/cmd/eden-press dependency closure:" >&2
	echo "$offending" | sed 's/^/  /' >&2
	echo "" >&2
	echo "The public render path (and the core CLI) must stay pure-Go (OBJECTIVE.md criterion 2)." >&2
	echo "Run 'go mod why <offending-package>' to find the import path and remove it." >&2
	exit 1
fi

echo "PASS: no chromedp in the press/chase/profiles/cmd/eden-press dependency closure."

# --- 05.1-01: eden-press-export is the SOLE chromedp-permitting cmd -------
#
# The core-trees scan above proves cmd/eden-press is chromedp-free. This
# block proves the other, equally load-bearing half of the 05.1 split: the
# new export binary is real (it builds), it genuinely needs chromedp (a
# positive check -- catches an accidental future refactor that silently
# drops the Chrome dependency without anyone noticing), and no OTHER cmd
# under ./cmd/... has quietly started importing it.
echo "check-no-chromedp: verifying cmd/eden-press-export builds and is the sole chromedp-permitting cmd"

if ! go build -o /dev/null ./cmd/eden-press-export 2>&1; then
	echo "FAIL: cmd/eden-press-export does not build." >&2
	exit 1
fi

export_deps_chromedp_count="$(go list -deps ./cmd/eden-press-export/... 2>/dev/null | grep -c -i 'chromedp' || true)"
if [ "$export_deps_chromedp_count" -eq 0 ]; then
	echo "FAIL: cmd/eden-press-export's dependency closure contains NO chromedp -- it is supposed to be the raster exporter." >&2
	exit 1
fi

# Enumerate every main-package import path under ./cmd/... and assert that
# every one OTHER than cmd/eden-press-export has a chromedp-free closure --
# i.e. eden-press-export is the ONLY cmd allowed chromedp.
offenders=""
while IFS= read -r pkg; do
	[ -z "$pkg" ] && continue
	case "$pkg" in
	*/cmd/eden-press-export) continue ;;
	esac

	count="$(go list -deps "$pkg" 2>/dev/null | grep -c -i 'chromedp' || true)"
	if [ "$count" -gt 0 ]; then
		offenders="${offenders}${pkg}\n"
	fi
done <<<"$(go list ./cmd/... 2>/dev/null)"

if [ -n "$offenders" ]; then
	echo "FAIL: chromedp found outside cmd/eden-press-export, in:" >&2
	printf '%b' "$offenders" | sed 's/^/  /' >&2
	echo "" >&2
	echo "cmd/eden-press-export must be the ONLY cmd whose closure contains chromedp." >&2
	exit 1
fi

echo "PASS: cmd/eden-press-export builds, contains chromedp, and is the sole chromedp-permitting cmd."
