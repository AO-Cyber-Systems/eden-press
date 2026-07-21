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

package markdown

import (
	"strings"
	"testing"
)

// renderDocFixtures covers Test-list case 5 across several distinct
// shapes -- single slide, multi-slide, directives -- proving RenderDoc
// matches Render byte-for-byte regardless of content shape. Every fixture
// here is ALSO an inline-SVG fixture: seam.go's Parse always enables
// inline-SVG mode (SvgOptionsKey), so there is no separate "inline-SVG
// disabled" case to cover for the seam's own Parse/Render/RenderDoc path.
var renderDocFixtures = []string{
	"# Single slide\n\nHello, world.\n",
	"# Slide 1\n\nfirst\n\n---\n\n# Slide 2\n\nsecond\n",
	"<!-- _class: lead -->\n\n# Slide 1\n\nfirst\n\n---\n\n<!-- paginate: true -->\n\n# Slide 2\n\nsecond\n",
	"![bg](https://example.com/bg.jpg)\n\n# Slide with background\n",
}

// TestRenderDocMatchesRenderAcrossFixtures covers Test-list case 5:
// markdown.RenderDoc(doc, source) -- for doc obtained via
// markdown.Parse(md) -- equals markdown.Render(md, nil) byte-for-byte,
// across several fixtures (single slide, multi-slide, directives,
// inline-SVG).
func TestRenderDocMatchesRenderAcrossFixtures(t *testing.T) {
	for i, md := range renderDocFixtures {
		doc, _ := Parse(md)

		got, err := RenderDoc(doc, []byte(md))
		if err != nil {
			t.Fatalf("fixture %d: RenderDoc: %v", i, err)
		}

		want, err := Render(md, nil)
		if err != nil {
			t.Fatalf("fixture %d: Render: %v", i, err)
		}

		if got != want {
			t.Fatalf("fixture %d: RenderDoc(doc, source) != Render(md, nil):\nRenderDoc:\n%s\nRender:\n%s", i, got, want)
		}
	}
}

// TestRenderDocDoesNotMutateOrReparse is the structural half of Test-list
// case 5: RenderDoc must render the *ast.Document it was GIVEN without
// mutating it (Section count identical before/after) and without any
// hidden internal state that would let its output drift across repeated
// calls on the SAME doc/source -- exactly what a hidden second Parse
// inside RenderDoc would risk (a fresh parser.Context/ID-generator per
// call could, in principle, produce different heading-slug results on a
// second pass; RenderDoc must never do that, because it never calls Parse
// at all -- see renderdoc.go's CRITICAL comment).
func TestRenderDocDoesNotMutateOrReparse(t *testing.T) {
	md := "# Slide 1\n\nfirst\n\n---\n\n# Slide 2\n\nsecond\n"
	doc, _ := Parse(md)
	source := []byte(md)

	before := countSectionsAnywhere(doc)
	if before != 2 {
		t.Fatalf("sanity check: doc has %d Section nodes before RenderDoc, want 2", before)
	}

	out1, err := RenderDoc(doc, source)
	if err != nil {
		t.Fatalf("RenderDoc (1st): %v", err)
	}
	if got := strings.Count(out1, `<section id="`); got != 2 {
		t.Fatalf("RenderDoc emitted %d <section> elements, want 2:\n%s", got, out1)
	}

	after := countSectionsAnywhere(doc)
	if after != before {
		t.Fatalf("RenderDoc mutated doc's Section count: before=%d after=%d", before, after)
	}

	out2, err := RenderDoc(doc, source)
	if err != nil {
		t.Fatalf("RenderDoc (2nd): %v", err)
	}
	if out1 != out2 {
		t.Fatalf("RenderDoc(doc, source) is not idempotent (proves no hidden re-parse/state):\n1st:\n%s\n2nd:\n%s", out1, out2)
	}
}
