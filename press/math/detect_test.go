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

import "testing"

// TestNeedsFallback is the FINALIZED construct-detection predicate (Objective
// 8's RESOLVED DECISION #2, 08-04): the pure predicate, driven test-first
// against a hand-built positive/negative case list. Positives are the
// permanent Chromium MathML-Core structural ceiling -- \tag/\label (no
// <mlabeledtr>, not even recognized converter tokens) and the amsmath
// NUMBERED alignment environments \align / \alignat / \alignat* (\align
// shares \tag's <mlabeledtr> gap by design; \alignat/\alignat*'s `{n}`
// column-count argument is empirically confirmed unmodeled by the converter --
// see detect.go's doc comment and 08-04-SUMMARY.md for the re-confirmed
// PROPOSAL §11 evidence). Negatives are every construct 08-02/08-03 fixed (or
// that PROPOSAL §11 / an empirical re-check showed was never actually broken)
// -- these MUST stay on the native-MathML path: \cases, \aligned, \align*,
// \array, and the 8 PROPOSAL §11 spike cases.
func TestNeedsFallback(t *testing.T) {
	positives := []string{
		`\tag{1}`,
		`a = b \tag{2}`,
		`\label{eq}`,
		`x = y \label{eq:main}`,
		`\begin{align} a &= b \end{align}`,
		`\begin{alignat}{2} a &= b \end{alignat}`,
		`\begin{alignat*}{2} a &= b \end{alignat*}`,
	}
	for _, raw := range positives {
		if !needsFallback(raw) {
			t.Errorf("needsFallback(%q) = false, want true (structural-ceiling construct must route to fallback)", raw)
		}
	}

	negatives := []string{
		`x^2`,
		`\frac{a}{b}`,
		`\sqrt{2}`,
		`\sum_{i=1}^n`,
		`E = mc^2`,
		`\alpha + \beta`,
		// Fixed in 08-03 (or confirmed never-broken) -- now-supported constructs
		// that must stay on the native path.
		`\begin{aligned} a &= b \end{aligned}`,
		`\begin{align*} a &= b \end{align*}`,
		`\begin{cases} 1 & x>0 \\ 0 & x\le 0 \end{cases}`,
		`\begin{array}{cc} a & b \end{array}`,
		// word-boundary guards: these must NOT trip \tag / \label / \begin.
		`\tagged{x}`,
		`\labelled{y}`,
		`\begingroup x \endgroup`,
	}
	for _, raw := range negatives {
		if needsFallback(raw) {
			t.Errorf("needsFallback(%q) = true, want false (common math must stay on MathML path)", raw)
		}
	}
}

// TestFallbackRouting is the CORPUS/TABLE proof required by success criterion
// 2 (must_haves truth 3): routing is asserted structurally over a table, not
// via manual inspection. It covers the finalized trigger set (\tag, \label,
// \align, \alignat, \alignat*) as TRUE, and every now-supported construct --
// \cases, \aligned, \align*, \array, plus the 8 PROPOSAL §11 spike-corpus raw
// strings (math_test.go TestSpikeCorpus) -- as FALSE, plus the word-boundary
// guard (Test-list case 4: \tagged/\labelled must NOT trip the trigger).
func TestFallbackRouting(t *testing.T) {
	cases := []struct {
		name           string
		raw            string
		expectFallback bool
	}{
		// Permanent structural ceiling -> PNG fallback.
		{"tag", `a = b \tag{1}`, true},
		{"label", `x = y \label{eq:main}`, true},
		{"align_numbered", `\begin{align} a &= b \\ c &= d \end{align}`, true},
		{"alignat", `\begin{alignat}{2} a &= b & c &= d \end{alignat}`, true},
		{"alignat_star", `\begin{alignat*}{2} a &= b & c &= d \end{alignat*}`, true},

		// Now-supported (08-02/08-03 fixes, or confirmed never-broken) -> native.
		{"cases", `\begin{cases} 1 & x>0 \\ 0 & x\le 0 \end{cases}`, false},
		{"aligned", `\begin{aligned} a &= b \\ c &= d \end{aligned}`, false},
		{"align_star", `\begin{align*} a &= b \\ c &= d \end{align*}`, false},
		{"array", `\begin{array}{cc} a & b \\ c & d \end{array}`, false},

		// The 8 PROPOSAL §11 spike-corpus cases (math_test.go TestSpikeCorpus) --
		// all fixed in 08-02/08-03, all must route native.
		{"spike_1_sum", `\sum_{i=1}^{n}`, false},
		{"spike_2_prod", `\prod_{i=1}^{n}`, false},
		{"spike_3_lim", `\lim_{x \to 0}`, false},
		{"spike_4_sqrt_index", `\sqrt[3]{x}`, false},
		{"spike_5_binom", `\binom{n}{k}`, false},
		{"spike_6_pmatrix", `\begin{pmatrix}1&0\\0&1\end{pmatrix}`, false},
		{"spike_7_aligned", `\begin{aligned}a&=b\\c&=d\end{aligned}`, false},
		{"spike_8_mathvariant", `\mathbb{R}`, false},

		// Test-list case 4: word-boundary guard -- prefix-sharing commands must
		// NOT trip \tag / \label.
		{"tagged_guard", `\tagged{x}`, false},
		{"labelled_guard", `\labelled{y}`, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := needsFallback(c.raw)
			if got != c.expectFallback {
				t.Errorf("needsFallback(%q) = %v, want %v", c.raw, got, c.expectFallback)
			}
		})
	}
}
