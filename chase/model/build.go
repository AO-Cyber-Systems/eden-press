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

// MODEL-01: Build derives a *Document by a single, direct, read-only walk of
// the SAME finalized post-transform AST chase/markdown/seam.go's two-phase
// seam (Parse then Render) produces for HTML -- never a second parse, never
// reverse-engineered from rendered HTML. See ARCHITECTURE.md Pattern 3.
package model

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"

	"github.com/AO-Cyber-Systems/eden-press/chase/directive"
	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

// Build walks doc and returns the Document it describes.
//
// doc and pc MUST be exactly the two return values of markdown.Parse(source)
// (see chase/markdown/seam.go) -- the SAME finalized AST the two-phase seam
// hands to a goldmark Renderer for HTML. source must be the identical byte
// slice Parse was called with (goldmark's ast.Node.Text/AttributeString
// resolve against source-indexed spans, never portable strings on their
// own).
//
// Build is READ-ONLY: it never mutates doc (no AppendChild/RemoveChild/
// SetAttribute), never re-parses source, and never renders. Every field on
// the returned Document is derived purely by inspecting nodes and Context
// values chase/markdown.Parse already produced. This is the MODEL-01
// single-source guarantee: calling Build on a *ast.Document does not change
// what a subsequent Render of that SAME doc produces (see
// TestBuildNonMutation).
func Build(doc ast.Node, source []byte, pc parser.Context) *Document {
	d := &Document{
		SchemaVersion: SchemaVersion,
		Meta:          buildMeta(pc),
	}

	// sectionIdx indexes the Section (in d.Sections) currently being walked,
	// or -1 when outside any Section. An index (not a pointer into the
	// slice) is used deliberately: append() may reallocate d.Sections'
	// backing array, which would silently invalidate a previously taken
	// pointer.
	sectionIdx := -1

	// A single ast.Walk over the WHOLE document -- not a shallow
	// FirstChild/NextSibling scan -- because chase/markdown/inlinesvg.go's
	// svgTransformer (priority 400, last) wraps every *markdown.Section in
	// <Svg><ForeignObject>...</ForeignObject></Svg> whenever
	// SvgOptionsKey.Enabled is set, which seam.go's Parse ALWAYS does.
	// *markdown.Section nodes are therefore nested two levels deep
	// (Document > Svg > ForeignObject > Section), never direct children of
	// doc; a full-tree walk finds them regardless of wrapper depth.
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *markdown.Section:
			if entering {
				d.Sections = append(d.Sections, Section{
					ID:    node.ID,
					Attrs: attrsToMap(node.Attrs),
				})
				sectionIdx = len(d.Sections) - 1
			} else {
				sectionIdx = -1
			}
		case *ast.Heading:
			if entering && sectionIdx >= 0 {
				d.Outline = append(d.Outline, OutlineEntry{
					SectionID: d.Sections[sectionIdx].ID,
					Level:     node.Level,
					Text:      string(node.Text(source)),
					Slug:      headingSlug(node),
				})
			}
		case *markdown.CommentNode:
			if entering && sectionIdx >= 0 && isNote(node.Raw) {
				d.Sections[sectionIdx].Notes = append(d.Sections[sectionIdx].Notes, node.Raw)
			}
		case *markdown.CommentInline:
			// A block-level comment (*markdown.CommentNode, handled above)
			// carries its own nested *CommentInline child with an identical
			// Raw (chase/markdown's inline-parsing pass re-detects the same
			// "<!--...-->" span inside the block's own content). Skip that
			// nested shadow copy here -- it was already counted via the
			// parent CommentNode case -- and only process a genuine
			// mid-paragraph inline comment (parented by something other than
			// its own CommentNode, e.g. a Paragraph).
			if _, nestedInCommentNode := node.Parent().(*markdown.CommentNode); nestedInCommentNode {
				break
			}
			if entering && sectionIdx >= 0 && isNote(node.Raw) {
				d.Sections[sectionIdx].Notes = append(d.Sections[sectionIdx].Notes, node.Raw)
			}
		}
		return ast.WalkContinue, nil
	})

	return d
}

// attrsToMap materializes chase/markdown's ordered []Attr into the JSON-sink
// map Section carries, returning nil (not an empty map) when attrs is empty
// so an untouched Section serializes with `attrs` omitted (see
// Section.Attrs' `omitempty` tag).
func attrsToMap(attrs []markdown.Attr) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name] = a.Value
	}
	return m
}

// headingSlug returns h's AutoHeadingID-generated "id" attribute value.
// goldmark's parser.WithAutoHeadingID stores it as a []byte via
// pc.IDs().Generate; a caller-set attribute (rare, but permitted by
// ast.Node.SetAttribute) could instead be a plain string, so both shapes are
// handled.
func headingSlug(h *ast.Heading) string {
	v, ok := h.AttributeString("id")
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return ""
	}
}

// buildMeta reads the front-matter KV pairs chase/markdown's
// frontMatterBlockParser stashed at markdown.FrontMatterKey (see
// chase/markdown/directive.go) and materializes them onto Meta.Directives,
// keyed by their raw declaration name (e.g. "theme", "size", "class") --
// deliberately NOT filtered through chase/directive.CoerceGlobal/CoerceLocal,
// so a front-matter key chase/directive does not itself recognize (e.g.
// "size", which only chase/theme's metadata layer interprets) still survives
// onto the model, per the must_have "Meta carries front-matter".
func buildMeta(pc parser.Context) Meta {
	kvs, _ := pc.Get(markdown.FrontMatterKey).([]directive.KV)
	if len(kvs) == 0 {
		return Meta{}
	}
	directives := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		directives[kv.Key] = stringifyRawValue(kv.Val)
	}
	return Meta{Directives: directives}
}

// stringifyRawValue renders a chase/directive.RawValue (always a string or a
// []string -- see chase/directive/yaml.go's RawValue doc comment) as a
// single display string for Meta.Directives.
func stringifyRawValue(v directive.RawValue) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, ",")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// isNote reports whether a detected comment's raw body represents a plain
// presenter note that belongs in Section.Notes, as opposed to a recognized
// directive comment that chase/markdown's own directiveApplyTransformer
// already materialized onto Section.Attrs (see chase/markdown/apply.go) --
// which must NOT also leak into Notes as redundant text.
//
// raw is re-parsed via chase/directive.ParseComment -- the exact same
// re-derivation chase/markdown's own buildEventStream/
// collectSectionCommentEvents use (chase/markdown/directive.go) -- rather
// than trusting CommentNode/CommentInline's already-lossy .KV
// map[string]string, which cannot round-trip array values or key order.
//
// A comment with zero parsed key/value pairs (no ":"-shaped lines at all,
// e.g. "just a presenter note") is always a note. A comment where every
// parsed key is a recognized global/local/spot directive is never a note
// (fully absorbed into Attrs already). A comment mixing recognized and
// unrecognized keys is conservatively treated as a note, so free-form
// presenter text is never silently dropped.
func isNote(raw string) bool {
	kvs := directive.ParseComment(raw)
	if len(kvs) == 0 {
		return true
	}
	for _, kv := range kvs {
		if !isRecognizedDirectiveKey(kv.Key) {
			return true
		}
	}
	return false
}

// isRecognizedDirectiveKey reports whether key names a global, local, or
// spot-prefixed local directive chase/directive recognizes at all --
// mirroring the exact same three-check sequence
// chase/markdown/directive.go's buildOrderedKeysPerSlide uses (CoerceGlobal,
// then CoerceLocal, then SpotKey+CoerceLocal), replayed here read-only for
// classification instead of key-order tracking.
//
// The raw value passed to CoerceGlobal/CoerceLocal is irrelevant to isKnown
// for every key those functions recognize (isKnown depends only on key; see
// chase/directive/directives.go) -- an empty string is passed defensively,
// never inspected for recognition purposes.
func isRecognizedDirectiveKey(key string) bool {
	if _, isKnown := directive.CoerceGlobal(key, "", nil); isKnown {
		return true
	}
	if _, isKnown := directive.CoerceLocal(key, ""); isKnown {
		return true
	}
	if base, ok := directive.SpotKey(key); ok {
		if _, isKnown := directive.CoerceLocal(base, ""); isKnown {
			return true
		}
	}
	return false
}
