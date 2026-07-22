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
# 04.1-02 (Objective 4.1) extends this same gate to ./cmd/... -- the CLI's
# own "stays chromedp-free" property, previously only an unenforced
# intention, is now mechanical too: the CLI's --format pptx wiring imports
# convert/pptx (stdlib OOXML, zero chromedp), so cmd/'s transitive closure
# stays chromedp-free and this gate keeps passing.
#
# Wired into CI (.github/workflows/ci.yml) beside the addlicense check and
# runnable locally via `make check-no-chromedp`.
set -euo pipefail

# The package trees whose transitive deps must be chromedp-free. press/ is the
# public surface; chase/ and profiles/ are everything it composes; cmd/ is the
# CLI consumer of press/ (04.1-02: now a CI-enforced invariant, not just a
# convention). Asserting all of these (not just press/) proves the boundary
# holds one level down too.
TREES=("./press/..." "./chase/..." "./profiles/..." "./bind/..." "./cmd/...")

echo "check-no-chromedp: scanning transitive deps of ${TREES[*]}"

# go list -deps prints the full transitive import closure, one package path per
# line. grep for any package whose path contains "chromedp".
if offending="$(go list -deps "${TREES[@]}" 2>/dev/null | grep -i 'chromedp' || true)"; [ -n "$offending" ]; then
	echo "FAIL: chromedp found in the press/chase/profiles/cmd dependency closure:" >&2
	echo "$offending" | sed 's/^/  /' >&2
	echo "" >&2
	echo "The public render path (and the CLI) must stay pure-Go (OBJECTIVE.md criterion 2)." >&2
	echo "Run 'go mod why <offending-package>' to find the import path and remove it." >&2
	exit 1
fi

echo "PASS: no chromedp in the press/chase/profiles/cmd dependency closure."
