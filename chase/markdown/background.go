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

// Background-image extraction (PARSE-06): finds every `![bg ...]` image in a
// slide's Section (image.go's ParseBgOptions decides which images are
// bg-marked), removes them from the visible content tree (mirroring
// sweep.js's hidden-paragraph cleanup), and aggregates them into
// backgroundSlideData for either the non-SVG backgroundImage directive path
// (applyNonSVGBackground, reusing apply.go's applyDirectivesToSection
// directly) or the inline-SVG advanced-background 3-layer structure
// (advancedbg.go, TRD 01-07 Task 3).
package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// bgImage is one extracted `![bg ...]` image, ready for either the non-SVG
// backgroundImage directive path or the advanced-background <figure> layer.
type bgImage struct {
	URL    string
	Alt    string
	Size   string // BgOptions.EffectiveSize()
	Filter string // BgOptions.FilterCSS()
}

// backgroundSlideData is one slide's aggregated background-image state,
// gathered by extractBackgroundImages. SplitSide/SplitSize/Direction are
// slide-level (Marpit encodes them redundantly per-image); the LAST
// bg-marked image whose alt text carries them wins, mirroring
// background_image/apply.js's own "only the last image's directives apply"
// precedent for backgroundImage/backgroundSize.
type backgroundSlideData struct {
	Images    []bgImage
	SplitSide string
	SplitSize string
	Direction string
}

// imageAltText reconstructs an *ast.Image's alt text by concatenating the
// Text() of its inline children -- an image's "alt" in markdown-it/goldmark
// is the rendered text content of its child inline nodes, not a single
// string field.
func imageAltText(n ast.Node, source []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b.Write(c.Text(source))
	}
	return b.String()
}

// extractBackgroundImages walks sec for bg-marked images (ParseBgOptions on
// each *ast.Image's alt text), detaches every match from the visible content
// tree, and aggregates them into a backgroundSlideData.
//
// A two-phase (read-then-mutate) walk is used deliberately: ast.Walk itself
// is read-only, and mutating the tree (RemoveChild) WHILE a Walk is
// traversing it is a concurrent-mutation hazard (01-RESEARCH.md). Phase 1
// collects (*ast.Image, BgOptions) matches without mutating; phase 2
// detaches each matched image and -- mirroring sweep.js's hidden-paragraph
// cascade, adapted for goldmark's AST (no generic "hidden" render-skip
// mechanism exists there) -- removes any parent block left childless as a
// result.
func extractBackgroundImages(sec *Section, source []byte) backgroundSlideData {
	var data backgroundSlideData

	type match struct {
		img  *ast.Image
		opts BgOptions
	}
	var matches []match

	_ = ast.Walk(sec, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		opts := ParseBgOptions(imageAltText(img, source))
		if !opts.Background {
			return ast.WalkContinue, nil
		}
		matches = append(matches, match{img: img, opts: opts})
		return ast.WalkContinue, nil
	})

	for _, m := range matches {
		data.Images = append(data.Images, bgImage{
			URL:    string(m.img.Destination),
			Alt:    m.opts.Alt,
			Size:   m.opts.EffectiveSize(),
			Filter: m.opts.FilterCSS(),
		})
		if m.opts.SplitSide != "" {
			data.SplitSide = m.opts.SplitSide
			data.SplitSize = m.opts.SplitSize
		}
		if m.opts.Direction != "" {
			data.Direction = m.opts.Direction
		}

		parent := m.img.Parent()
		if parent == nil {
			continue
		}
		parent.RemoveChild(parent, m.img)
		if parent.ChildCount() == 0 {
			if gp := parent.Parent(); gp != nil {
				gp.RemoveChild(gp, parent)
			}
		}
	}

	return data
}

// applyNonSVGBackground materializes a slide's LAST background image (per
// background_image/apply.js: "only the last image in a slide's images array
// applies" in non-SVG mode) as a backgroundImage local directive, reusing
// apply.go's applyDirectivesToSection directly (same package, NOT
// reimplemented) -- so the SAME generic key loop (data-{kebab}/--{kebab})
// fires for backgroundImage/backgroundSize exactly as it would for an
// author-written directive, matching real Marpit's own
// inject-into-resolved-map-before-the-generic-pass behavior.
//
// Filter has no equivalent directive-system hook (apply.go's branch set is
// fixed and not modified by this TRD), so it is merged directly onto the
// section's style attribute afterward.
func applyNonSVGBackground(sec *Section, data backgroundSlideData) {
	if len(data.Images) == 0 {
		return
	}
	last := data.Images[len(data.Images)-1]

	resolved := map[string]any{"backgroundImage": `url("` + last.URL + `")`}
	keys := []string{"backgroundImage"}
	if last.Size != "" {
		resolved["backgroundSize"] = last.Size
		keys = append(keys, "backgroundSize")
	}

	var paginating []*Section
	applyDirectivesToSection(sec, keys, resolved, 0, &paginating)

	if last.Filter != "" {
		sec.Attrs = mergeStyleDecl(sec.Attrs, "filter", last.Filter)
	}
}

// cloneAttrs returns an independent copy of attrs, safe to mutate via
// overrideAttr without disturbing the original slice -- advancedbg.go needs
// three independent copies (content/background/pseudo) derived from one
// base Attrs snapshot.
func cloneAttrs(attrs []Attr) []Attr {
	out := make([]Attr, len(attrs))
	copy(out, attrs)
	return out
}

// overrideAttr sets the value of the attr named name within attrs to value,
// in place (preserving position) if already present, else appends a new
// Attr. Returns the (possibly reallocated) slice.
func overrideAttr(attrs []Attr, name, value string) []Attr {
	for i, a := range attrs {
		if a.Name == name {
			attrs[i].Value = value
			return attrs
		}
	}
	return append(attrs, Attr{Name: name, Value: value})
}

// mergeStyleDecl sets prop:value on attrs' existing "style" attribute (in
// place, preserving its position and any other declarations already in it),
// or appends a new "style" attribute if none exists yet.
func mergeStyleDecl(attrs []Attr, prop, value string) []Attr {
	style := NewInlineStyle()
	idx := -1
	for i, a := range attrs {
		if a.Name == "style" {
			idx = i
			seedInlineStyle(style, a.Value)
			break
		}
	}
	style.Set(prop, value)
	if idx >= 0 {
		attrs[idx].Value = style.String()
		return attrs
	}
	return append(attrs, Attr{Name: "style", Value: style.String()})
}

// seedInlineStyle parses a raw "prop:value;prop2:value2;" style string
// (e.g. an existing section's style attribute) into style, preserving
// first-seen declaration order for any NEW prop added afterward via Set.
func seedInlineStyle(style *InlineStyle, raw string) {
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		style.Set(kv[0], kv[1])
	}
}
