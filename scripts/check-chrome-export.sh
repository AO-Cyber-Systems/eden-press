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

# check-chrome-export.sh runs eden-press's convert/ export test suite (the
# exporters + the 05-05 capstone) and encodes + mechanically enforces the
# EXP-04 Chrome-version-pin + PDF-path re-validation PROCESS RULE
# (05-RESEARCH Pitfall A):
#
#   Two INDEPENDENTLY-DOCUMENTED Chrome regressions hit the PDF export path
#   ONLY (SVG-in-PDF issues at Chrome >=108; a print-pipeline regression
#   around Chrome ~125) -- a PNG-screenshot-path pass does NOT imply the PDF
#   path still works. So on ANY deliberate CHROME_VERSION bump, BOTH of the
#   following MUST run and pass before the bump is accepted:
#     - convert/pdf.TestToPDFInlineSVGFixture  (05-03 Test-list case 4 --
#       the inline-SVG PDF conformance fixture)
#     - convert.TestCapstoneExportEndToEnd     (05-05 Task 1 -- the
#       press.Render -> ToPDF/ToImages capstone)
#
# CHROME_VERSION is read from the environment (default below), the SAME
# value the Makefile's CHROME_VERSION var and the pinned no-system-Chrome CI
# export job (.github/workflows/ci.yml's `export` job) use -- see
# convert/EXPORT.md for the full version-pin + re-validation process this
# script mechanizes. NEVER pin/accept "latest" (Pitfall A/11).
#
# Wired into the CI export job via `make check-chrome-export`; runnable
# locally too -- the tests themselves are Chrome-presence-gated (t.Skip
# cleanly when no Chrome/Chromium is discoverable), so a local run outside a
# provisioned Chrome/container documents-skips rather than proving the
# no-system-Chrome container path for real (that proof is the CI job's job).
set -euo pipefail

# The SINGLE pinned CHROME_VERSION value this script documents + runs
# against. Kept in sync with the Makefile's CHROME_VERSION var and
# .github/workflows/ci.yml's export job container tag -- bump ALL THREE
# together, then re-run this script (which enforces the PDF re-validation
# rule below) before accepting the bump.
CHROME_VERSION="${CHROME_VERSION:-151.0.7922.34}"

# REQUIRED_PDF_TESTS are the two Go test function names the PDF-path
# re-validation rule requires to have RUN (pass or Chrome-presence-skip --
# never simply absent from the run) on every CHROME_VERSION bump.
REQUIRED_PDF_TESTS=("TestToPDFInlineSVGFixture" "TestCapstoneExportEndToEnd")

echo "check-chrome-export: pinned CHROME_VERSION=${CHROME_VERSION} (chromedp/headless-shell tag; NEVER latest)"
echo "check-chrome-export: PROCESS RULE (Pitfall A) -- a deliberate CHROME_VERSION bump REQUIRES"
echo "  re-running BOTH: ${REQUIRED_PDF_TESTS[*]}"
echo "  before the bump is accepted. A PNG-path-only pass does NOT imply a PDF-path pass"
echo "  (SVG-in-PDF regressions at Chrome >=108; print-pipeline regressions at Chrome ~125)."
echo ""
echo "check-chrome-export: running ./convert/... export tests (expecting Chrome to be"
echo "  discovered via the EXP-04 fallback chain: BrowserPath -> CHROME_PATH -> PATH"
echo "  auto-detect -> the pinned headless-shell image supplying CHROME_PATH in CI)..."

set +e
TEST_OUTPUT="$(go test ./convert/... -v 2>&1)"
TEST_STATUS=$?
set -e

echo "$TEST_OUTPUT"

if [ "$TEST_STATUS" -ne 0 ]; then
	echo "" >&2
	echo "FAIL: ./convert/... export tests failed under CHROME_VERSION=${CHROME_VERSION}." >&2
	exit 1
fi

missing=()
for name in "${REQUIRED_PDF_TESTS[@]}"; do
	if ! grep -q "=== RUN   ${name}" <<<"$TEST_OUTPUT"; then
		missing+=("$name")
	fi
done

if [ "${#missing[@]}" -gt 0 ]; then
	echo "" >&2
	echo "FAIL: required PDF-path re-validation test(s) did not run: ${missing[*]}" >&2
	echo "  The EXP-04 process rule (Pitfall A) requires BOTH tests to run on every" >&2
	echo "  CHROME_VERSION bump -- confirm convert/pdf and convert/ are both included in" >&2
	echo "  the ./convert/... scope and that neither test function has been renamed." >&2
	exit 1
fi

echo ""
echo "PASS: check-chrome-export -- CHROME_VERSION=${CHROME_VERSION}; PDF-path re-validation"
echo "  tests (${REQUIRED_PDF_TESTS[*]}) ran (passing, or cleanly Chrome-skipped when no"
echo "  Chrome/Chromium is discoverable in this environment)."
