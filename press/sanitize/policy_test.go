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

package sanitize

import (
	"strings"
	"testing"
)

// TestTagFilter covers Test-list case 1: the GFM disallowed-raw-HTML tags
// goldmark's GFM extension does NOT filter must all be neutralized (their
// tags AND their inner content stripped) by Policy().
func TestTagFilter(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"script", `<p>before</p><script>alert(1)</script><p>after</p>`},
		{"iframe", `<iframe src="https://evil.example"></iframe>`},
		{"style", `<style>body{background:url(javascript:alert(1))}</style>`},
		{"textarea", `<textarea>payload</textarea>`},
		{"title", `<title>payload</title>`},
		{"xmp", `<xmp>payload</xmp>`},
		{"noembed", `<noembed>payload</noembed>`},
		{"noframes", `<noframes>payload</noframes>`},
		{"plaintext", `<plaintext>payload`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := Policy().Sanitize(tc.input)
			if strings.Contains(out, "payload") {
				t.Errorf("Sanitize(%q) = %q, want tag content stripped (no %q)", tc.input, out, "payload")
			}
			if strings.Contains(strings.ToLower(out), "alert(1)") {
				t.Errorf("Sanitize(%q) = %q, want script/style payload removed", tc.input, out)
			}
			if strings.Contains(strings.ToLower(out), "<"+tc.name) {
				t.Errorf("Sanitize(%q) = %q, want <%s> tag itself removed", tc.input, out, tc.name)
			}
		})
	}
}

// TestStyleAttr covers Test-list case 3: the style attribute is dropped
// everywhere -- bluemonday has no CSS-value sanitizer, so excluding style
// entirely is the safe, deliberate posture (never partially allowed).
func TestStyleAttr(t *testing.T) {
	in := `<p style="expression(alert(1))">hello</p>`
	out := Policy().Sanitize(in)
	if strings.Contains(out, "style") {
		t.Errorf("Sanitize(%q) = %q, want style attribute stripped entirely", in, out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("Sanitize(%q) = %q, want safe text content preserved", in, out)
	}
}

// TestURLScheme covers Test-list case 4: javascript: URIs are neutralized
// while http(s):// and data: (needed for the base64 PNG math fallback)
// survive.
func TestURLScheme(t *testing.T) {
	t.Run("javascript scheme blocked", func(t *testing.T) {
		in := `<a href="javascript:alert(1)">click</a>`
		out := Policy().Sanitize(in)
		if strings.Contains(strings.ToLower(out), "javascript:") {
			t.Errorf("Sanitize(%q) = %q, want javascript: scheme stripped", in, out)
		}
	})

	t.Run("https scheme survives", func(t *testing.T) {
		in := `<a href="https://example.com/x">click</a>`
		out := Policy().Sanitize(in)
		if !strings.Contains(out, `href="https://example.com/x"`) {
			t.Errorf("Sanitize(%q) = %q, want https:// href preserved", in, out)
		}
	})

	t.Run("data scheme survives", func(t *testing.T) {
		in := `<img src="data:image/png;base64,AA==" alt="math">`
		out := Policy().Sanitize(in)
		if !strings.Contains(out, `src="data:image/png;base64,AA=="`) {
			t.Errorf("Sanitize(%q) = %q, want data: src preserved", in, out)
		}
	})
}

// TestOnHandler covers Test-list case 5: on* event-handler attributes are
// removed. No special-case code is needed for this -- bluemonday's
// blank-slate policy strips any attribute that was never explicitly
// allow-listed, and Policy() never calls AllowAttrs("onclick") (or any
// on* variant).
func TestOnHandler(t *testing.T) {
	in := `<div onclick="alert(1)">hello</div>`
	out := Policy().Sanitize(in)
	if strings.Contains(strings.ToLower(out), "onclick") {
		t.Errorf("Sanitize(%q) = %q, want onclick handler attribute stripped", in, out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("Sanitize(%q) = %q, want safe text content preserved", in, out)
	}
}

// TestStripVsEscape covers Test-list case 6: a disallowed tag's DOCUMENTED
// behavior is STRIP, not escape. Marp's JS `xss` library instead escapes
// disallowed tags (rendering them as visible encoded text, e.g.
// "&lt;script&gt;..."). This is a deliberate, tested deviation for CORE-05:
// bluemonday's blank-slate posture strips rather than escapes, and this test
// pins that choice so it is never silently "fixed" to look like escaping.
func TestStripVsEscape(t *testing.T) {
	in := `<script>alert(1)</script>`
	out := Policy().Sanitize(in)

	// STRIP means the tag and its content are gone entirely -- not
	// re-rendered as escaped/visible text.
	if out != "" {
		t.Errorf("Sanitize(%q) = %q, want fully stripped (empty string) per the documented strip-vs-escape deviation", in, out)
	}

	// Explicitly assert the ESCAPE behavior (what Marp's JS xss would do)
	// is NOT what happens here, to make the deviation unmistakable.
	escaped := "&lt;script&gt;alert(1)&lt;/script&gt;"
	if out == escaped {
		t.Errorf("Sanitize(%q) = %q, unexpectedly matches Marp's escape behavior -- CORE-05 deliberately strips instead", in, out)
	}
}
