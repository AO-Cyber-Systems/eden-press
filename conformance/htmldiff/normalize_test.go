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

package htmldiff

import "testing"

// --- Happy path: the allow-list of provably-cosmetic differences compares Equal ---
//
// The allow-list is EXACTLY four named classes:
//   1. void-element syntax (<br> vs <br/> vs <br />)
//   2. <hr> void syntax
//   3. attribute order
//   4. inter-block whitespace
// All test fixtures are hand-built inline string literals (no_llm_test_data).

func TestEqual_CosmeticDifferencesCompareEqual(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		actual   string
	}{
		{
			name:     "identical fragments",
			expected: "<p>hello world</p>",
			actual:   "<p>hello world</p>",
		},
		{
			name:     "void element <br> vs <br/>",
			expected: "<p>a<br>b</p>",
			actual:   "<p>a<br/>b</p>",
		},
		{
			name:     "void element <br/> vs <br />",
			expected: "<p>a<br/>b</p>",
			actual:   "<p>a<br />b</p>",
		},
		{
			name:     "void element <hr> vs <hr/>",
			expected: "<hr>",
			actual:   "<hr/>",
		},
		{
			name:     "attribute order differences",
			expected: `<a href="x" title="y">link</a>`,
			actual:   `<a title="y" href="x">link</a>`,
		},
		{
			name:     "inter-block whitespace / newlines",
			expected: "<p>a</p>\n<p>b</p>\n",
			actual:   "<p>a</p><p>b</p>",
		},
		{
			name:     "leading/trailing whitespace around blocks",
			expected: "   <ul><li>one</li><li>two</li></ul>   ",
			actual:   "<ul>\n  <li>one</li>\n  <li>two</li>\n</ul>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eq, diff := Equal(tc.expected, tc.actual)
			if !eq {
				t.Fatalf("expected Equal=true (cosmetic-only difference), got false.\ndiff:\n%s", diff)
			}
			if diff != "" {
				t.Errorf("expected empty diff when Equal, got:\n%s", diff)
			}
		})
	}
}

// TestEqual_VoidElementSyntax asserts the void-element canonicalization contract
// explicitly (not incidentally) across all three spellings simultaneously.
func TestEqual_VoidElementSyntax(t *testing.T) {
	forms := []string{"<p>a<br>b</p>", "<p>a<br/>b</p>", "<p>a<br />b</p>"}
	for i := 0; i < len(forms); i++ {
		for j := 0; j < len(forms); j++ {
			eq, diff := Equal(forms[i], forms[j])
			if !eq {
				t.Fatalf("void-element forms %q vs %q must be Equal; diff:\n%s", forms[i], forms[j], diff)
			}
		}
	}
	// <hr> spellings too.
	hrForms := []string{"<hr>", "<hr/>", "<hr />"}
	for i := 0; i < len(hrForms); i++ {
		for j := 0; j < len(hrForms); j++ {
			eq, _ := Equal(hrForms[i], hrForms[j])
			if !eq {
				t.Fatalf("hr forms %q vs %q must be Equal", hrForms[i], hrForms[j])
			}
		}
	}
}

// --- Negative: the allow-list must NOT over-reach (CONF-02 acceptance) ---
//
// These prove the DOM-normalized diff still reports real fidelity bugs as a DIFF.
// If any of these unexpectedly compares Equal, the normalizer is over-reaching.

func TestEqual_RealDifferencesReportedAsDiff(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		actual   string
	}{
		{
			// Broken code-span: inner text a<b vs a>b. Angle brackets are entity-
			// encoded in HTML; inside <code> they are character-significant.
			name:     "code span inner text changed (a<b vs a>b)",
			expected: "<p><code>a&lt;b</code></p>",
			actual:   "<p><code>a&gt;b</code></p>",
		},
		{
			// Altered <pre> content: an extra leading space on a line. <pre> is
			// whitespace-significant and must NOT be normalized.
			name:     "pre content leading-space altered",
			expected: "<pre>alpha\n  beta</pre>",
			actual:   "<pre>alpha\nbeta</pre>",
		},
		{
			// Altered <pre> content: a changed line entirely.
			name:     "pre content line changed",
			expected: "<pre><code>foo\nbar\n</code></pre>",
			actual:   "<pre><code>foo\nbaz\n</code></pre>",
		},
		{
			name:     "different element em vs strong",
			expected: "<p><em>x</em></p>",
			actual:   "<p><strong>x</strong></p>",
		},
		{
			name:     "attribute VALUE differs (not order)",
			expected: `<a href="x">l</a>`,
			actual:   `<a href="z">l</a>`,
		},
		{
			name:     "text content differs",
			expected: "<p>hello</p>",
			actual:   "<p>goodbye</p>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eq, diff := Equal(tc.expected, tc.actual)
			if eq {
				t.Fatalf("expected Equal=false (real difference must be reported as DIFF), got true — normalizer is over-reaching")
			}
			if diff == "" {
				t.Errorf("expected a non-empty human-readable diff when not Equal")
			}
		})
	}
}
