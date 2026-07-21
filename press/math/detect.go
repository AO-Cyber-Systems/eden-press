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

package math

import "regexp"

// fallbackRE is the construct-detection predicate's raw-LaTeX pre-scan (research
// Open Question #4, MEDIUM-confidence synthesis). It matches the classes Chromium
// MathML-Core cannot render faithfully today, so they must degrade to the raster
// fallback rather than emit broken MathML:
//
//   - \tag / \label — equation numbering/anchoring latex2mathml lowers to
//     <mtable>/<mlabeledtr>, which Chromium MathML-Core does not implement.
//   - \begin{aligned|align|alignat|cases|array} — multi-line alignment
//     environments that lower to <mtable>, also unsupported.
//
// A \b word boundary follows \tag and \label so \tagged / \labelled (and other
// commands that merely share the prefix) are NOT matched; the environment arm
// requires the literal `{name}`, so \begingroup and friends never trip it.
//
// This is a deliberately BASELINE rule: Objective 8's decision gate finalizes
// the exact fallback-trigger set once a math corpus exists to validate against.
var fallbackRE = regexp.MustCompile(`\\tag\b|\\label\b|\\begin\{(?:aligned|align|alignat|cases|array)\}`)

// needsFallback reports whether rawLatex contains a heavy construct that must
// route to the PNG fallback instead of native MathML. It is a pure, allocation-
// free function of its input (no I/O, no globals mutated) — the routing decision
// is made on the RAW source, cheaply and bounded, BEFORE any MathML conversion
// is attempted.
func needsFallback(rawLatex string) bool {
	return fallbackRE.MatchString(rawLatex)
}
