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

// TestNeedsFallback is the riskiest-item spike (research Open Question #4,
// MEDIUM-confidence synthesis): the pure construct-detection predicate, driven
// test-first against a hand-built positive/negative case list BEFORE any render
// path exists. Positives are the known Chromium MathML-Core gaps (\tag/\label +
// the aligned/align/alignat/cases/array environments); negatives are common
// math that MUST stay on the native-MathML path.
func TestNeedsFallback(t *testing.T) {
	positives := []string{
		`\tag{1}`,
		`a = b \tag{2}`,
		`\label{eq}`,
		`x = y \label{eq:main}`,
		`\begin{aligned} a &= b \end{aligned}`,
		`\begin{align} a &= b \end{align}`,
		`\begin{alignat}{2} a &= b \end{alignat}`,
		`\begin{cases} 1 & x>0 \\ 0 & x\le 0 \end{cases}`,
		`\begin{array}{cc} a & b \end{array}`,
	}
	for _, raw := range positives {
		if !needsFallback(raw) {
			t.Errorf("needsFallback(%q) = false, want true (heavy construct must route to fallback)", raw)
		}
	}

	negatives := []string{
		`x^2`,
		`\frac{a}{b}`,
		`\sqrt{2}`,
		`\sum_{i=1}^n`,
		`E = mc^2`,
		`\alpha + \beta`,
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
