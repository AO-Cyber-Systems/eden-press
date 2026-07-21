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
	"bytes"
	"testing"
)

// TestParseWithEngineParityWithParse covers TRD 03-01 Test-list case 1: given
// the SAME caller-supplied engine shape the package builds internally
// (NewEngine()), ParseWithEngine returns a finalized *ast.Document with the
// SAME shape Parse returns -- identical Section count AND byte-identical
// rendered HTML -- proving the engine-parameterized copy is faithful and drops
// no transform.
func TestParseWithEngineParityWithParse(t *testing.T) {
	md := "<!-- _class: lead -->\n\n# Slide 1\n\nfirst\n\n---\n\n# Slide 2\n\nsecond\n"
	source := []byte(md)

	engine := NewEngine()
	docEng, _ := ParseWithEngine(md, engine)
	docDef, _ := Parse(md)

	if got, want := countSectionsAnywhere(docEng), countSectionsAnywhere(docDef); got != want {
		t.Fatalf("Section count via ParseWithEngine = %d, want %d (Parse)", got, want)
	}
	if got := countSectionsAnywhere(docEng); got != 2 {
		t.Fatalf("multi-slide deck: got %d Sections, want 2", got)
	}

	// Render both finalized trees -- the ParseWithEngine tree through the
	// CALLER's engine, the Parse tree through defaultEngine -- and require
	// byte-identical HTML. Same input + same engine config => same output;
	// any divergence means the copy dropped a pre-seed line or a transform.
	var bufEng bytes.Buffer
	if err := engine.Renderer().Render(&bufEng, source, docEng); err != nil {
		t.Fatalf("render ParseWithEngine doc: %v", err)
	}
	var bufDef bytes.Buffer
	if err := defaultEngine.Renderer().Render(&bufDef, source, docDef); err != nil {
		t.Fatalf("render Parse doc: %v", err)
	}
	if bufEng.String() != bufDef.String() {
		t.Fatalf("ParseWithEngine render != Parse render:\nengine:\n%s\ndefault:\n%s", bufEng.String(), bufDef.String())
	}
}

// TestParseWithEnginePreSeedsContext covers TRD 03-01 Test-list case 2:
// ParseWithEngine pre-seeds the parser.Context IDENTICALLY to Parse --
// SvgOptionsKey with Enabled=true always, and, for a deck whose front matter
// carries headingDivider: 2, the resolved HeadingDividerKey []int{1,2}.
func TestParseWithEnginePreSeedsContext(t *testing.T) {
	// SvgOptionsKey: enabled on a plain deck, exactly as Parse sets it.
	{
		_, pc := ParseWithEngine("# Hello\n", NewEngine())
		v, ok := pc.Get(SvgOptionsKey).(*SvgOptions)
		if !ok || v == nil {
			t.Fatalf("SvgOptionsKey missing/wrong type on ParseWithEngine context: %#v", pc.Get(SvgOptionsKey))
		}
		if !v.Enabled {
			t.Fatalf("SvgOptionsKey.Enabled = false, want true (Parse pre-seed parity)")
		}
	}

	// HeadingDividerKey: resolved to []int{1,2} for headingDivider: 2, and
	// identical to what Parse resolves for the same source.
	{
		md := "---\nheadingDivider: 2\n---\n\n# H1\n\n## H2\n"
		_, pcEng := ParseWithEngine(md, NewEngine())
		_, pcDef := Parse(md)

		levelsEng, okEng := pcEng.Get(HeadingDividerKey).([]int)
		if !okEng {
			t.Fatalf("HeadingDividerKey missing/wrong type on ParseWithEngine context: %#v", pcEng.Get(HeadingDividerKey))
		}
		levelsDef, okDef := pcDef.Get(HeadingDividerKey).([]int)
		if !okDef {
			t.Fatalf("HeadingDividerKey missing/wrong type on Parse context: %#v", pcDef.Get(HeadingDividerKey))
		}
		if want := []int{1, 2}; !intsEqual(levelsEng, want) {
			t.Fatalf("ParseWithEngine HeadingDividerKey = %v, want %v", levelsEng, want)
		}
		if !intsEqual(levelsEng, levelsDef) {
			t.Fatalf("ParseWithEngine HeadingDividerKey %v != Parse %v", levelsEng, levelsDef)
		}
	}
}

// TestParseWithEngineIsAdditiveNonBreaking covers TRD 03-01 Test-list case 3:
// the existing seam is untouched, so Render(md, nil) still succeeds and
// produces output byte-identical to driving the NEW ParseWithEngine seam
// through the same engine. (git diff on seam.go/chase.go being empty plus the
// full ./chase/... suite staying green complete the "before == after" proof.)
func TestParseWithEngineIsAdditiveNonBreaking(t *testing.T) {
	md := "<!-- _class: lead -->\n\n# Slide 1\n\nfirst\n\n---\n\n# Slide 2\n\nsecond\n"
	source := []byte(md)

	// Existing seam entrypoint, unchanged.
	seamOut, err := Render(md, nil)
	if err != nil {
		t.Fatalf("existing Render(md, nil): %v", err)
	}

	// New additive seam driven manually through NewEngine().
	engine := NewEngine()
	doc, _ := ParseWithEngine(md, engine)
	var buf bytes.Buffer
	if err := engine.Renderer().Render(&buf, source, doc); err != nil {
		t.Fatalf("render via ParseWithEngine: %v", err)
	}

	if buf.String() != seamOut {
		t.Fatalf("ParseWithEngine seam output != existing Render output (seam should be unaffected):\nnew:\n%s\nexisting:\n%s", buf.String(), seamOut)
	}
}

// intsEqual reports whether two int slices are element-wise equal.
func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
