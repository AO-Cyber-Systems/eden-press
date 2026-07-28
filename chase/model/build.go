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
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/AO-Cyber-Systems/eden-press/chase/directive"
	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
)

// rawMath is the duck-typed seam by which Build recovers a math construct's RAW
// TeX + display flag WITHOUT chase/model importing press/math. press/math's
// (unexported) *mathNode satisfies it via its additive MathRaw()/MathDisplay()
// getters (press/math/math.go), so chase/model reaches the raw TeX with no
// direct import -- no dependency-closure coupling, no import cycle, and the
// no-chromedp closure of press/chase/profiles is unaffected. ANY ast.Node
// implementing these two methods is materialized as a math Block; the raw TeX
// is the lossless surface Objective 7's DART-04 consumes (never the presentation
// MathML in Output.HTML, which carries no <annotation> TeX).
type rawMath interface {
	MathRaw() string
	MathDisplay() bool
}

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
				// The Outline entry is FROZEN v1 behavior -- unchanged. Schema
				// v2 ADDITIVELY also appends a heading Block so a Section's
				// body content carries its headings in document order (the
				// Outline is a flat deck-wide index; Blocks is per-section body).
				text := string(node.Text(source))
				d.Outline = append(d.Outline, OutlineEntry{
					SectionID: d.Sections[sectionIdx].ID,
					Level:     node.Level,
					Text:      text,
					Slug:      headingSlug(node),
				})
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind:  BlockHeading,
					Level: node.Level,
					Text:  text,
				})
			}
		case *ast.Paragraph, *ast.TextBlock:
			// A paragraph (or a list-item's TextBlock, though those are never
			// reached here -- the *ast.List case below skips its children)
			// contributes its concatenated plain text as an editable paragraph
			// Block. TWO paragraphs are skipped so they don't emit a spurious
			// text Block:
			//   - a whitespace-only paragraph; and
			//   - a MATH-ONLY paragraph. goldmark wraps a standalone `$$…$$` /
			//     `$…$` line in a Paragraph whose only child is the math node,
			//     yet Paragraph.Text reconstructs the RAW `$$…$$` source (the
			//     math node does not strip it), so a naive text check would
			//     double-emit the raw TeX as prose. The math node itself is
			//     materialized as a math Block when the walk descends into this
			//     paragraph (the rawMath case in default, below) -- preserving
			//     document order + the display flag (error_recovery).
			//
			// An IMAGE-ONLY paragraph is skipped for the same reason a
			// math-only one is: goldmark wraps a standalone `![alt](src)` in a
			// Paragraph whose Text reconstructs the ALT text, which would
			// double-emit the alt as prose alongside the image Block the walk
			// materializes when it descends into the paragraph (EPD-R1).
			if entering && sectionIdx >= 0 && !isMathOnlyParagraph(n, source) && !isImageOnlyParagraph(n, source) {
				if txt := string(n.Text(source)); strings.TrimSpace(txt) != "" {
					d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
						Kind: BlockParagraph,
						Text: txt,
					})
				}
			}
		case *ast.List:
			// Emit one list Block carrying every item (including nested items,
			// each with its 0-based nesting Level) then SKIP the list's
			// children, so the item TextBlocks/Paragraphs are not ALSO emitted
			// as loose paragraph Blocks. Nested lists are handled entirely by
			// collectListItems, never re-encountered by this walk.
			if entering && sectionIdx >= 0 {
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind:    BlockList,
					Ordered: node.IsOrdered(),
					Items:   collectListItems(node, source),
				})
				return ast.WalkSkipChildren, nil
			}
		case *east.Table:
			// EPD-R1. Emit one table Block carrying the header row, every body
			// row and the per-column alignments, then SKIP the table's
			// children so its cells' inline text is not ALSO emitted as loose
			// prose -- the same containment the *ast.List case above applies.
			// Before this case existed a table's content reached Output.HTML
			// but was absent from the model entirely.
			if entering && sectionIdx >= 0 {
				headers, rows := collectTable(node, source)
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind:    BlockTable,
					Headers: headers,
					Rows:    rows,
					Aligns:  alignNames(node.Alignments),
				})
				return ast.WalkSkipChildren, nil
			}
		case *ast.Blockquote:
			// EPD-R1. A blockquote previously fell through to its child
			// Paragraphs and surfaced as an indistinguishable paragraph Block,
			// so downstream had no way to style a quote as a quote. Emit one
			// quote Block carrying the concatenated text and skip the
			// children, so the text is not double-emitted as prose.
			if entering && sectionIdx >= 0 {
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind: BlockQuote,
					Text: quoteText(node, source),
				})
				return ast.WalkSkipChildren, nil
			}
		case *ast.Image:
			// EPD-R1. Destination + alt text + optional title. Reached whether
			// the image stands alone in its own paragraph (whose prose Block is
			// suppressed by isImageOnlyParagraph) or sits mid-sentence (where
			// the surrounding prose is kept, mirroring mixed prose+math).
			if entering && sectionIdx >= 0 {
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind:  BlockImage,
					Src:   string(node.Destination),
					Text:  string(node.Text(source)),
					Title: string(node.Title),
				})
			}
		case *ast.FencedCodeBlock:
			// RAW source (pre-chroma) reconstructed from the node's line
			// segments, plus the info-string Language -- the lossless surface
			// Objective 7's flutter_highlighting consumes.
			if entering && sectionIdx >= 0 {
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind:     BlockCode,
					Language: string(node.Language(source)),
					Text:     rawLinesText(node.Lines(), source),
				})
			}
		case *ast.CodeBlock:
			// Indented code block: RAW source, no info-string language.
			if entering && sectionIdx >= 0 {
				d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
					Kind: BlockCode,
					Text: rawLinesText(node.Lines(), source),
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
		default:
			// Math construct: reached via the duck-typed rawMath seam (press/
			// math's *mathNode satisfies it) so chase/model never imports press/
			// math. Present ONLY under a math-battery engine (press.Render);
			// under the default chase engine `$$x$$` is plain text -> a
			// paragraph Block, which is expected + correct. Text is the RAW TeX,
			// Display the $$…$$ (block) vs. $…$ (inline) flag.
			if entering && sectionIdx >= 0 {
				if mn, ok := n.(rawMath); ok {
					d.Sections[sectionIdx].Blocks = append(d.Sections[sectionIdx].Blocks, Block{
						Kind:    BlockMath,
						Text:    mn.MathRaw(),
						Display: mn.MathDisplay(),
					})
				}
			}
		}
		return ast.WalkContinue, nil
	})

	return d
}

// isMathOnlyParagraph reports whether n is a paragraph/text-block whose only
// non-whitespace content is math node(s) -- e.g. a standalone `$$E=mc^2$$` or
// `$x$` line goldmark wraps in a Paragraph. Such a paragraph must NOT emit a
// text Block (its Paragraph.Text reconstructs the raw `$$…$$` source); the math
// node it wraps is emitted as a math Block by the walk instead. Returns false
// for a paragraph that mixes prose with math, or one with no math at all.
func isMathOnlyParagraph(n ast.Node, source []byte) bool {
	hasMath := false
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(rawMath); ok {
			hasMath = true
			continue
		}
		if strings.TrimSpace(string(c.Text(source))) != "" {
			return false // a non-math, non-whitespace child -> real prose
		}
	}
	return hasMath
}

// isImageOnlyParagraph reports whether n is a paragraph/text-block whose only
// non-whitespace content is image node(s) -- e.g. a standalone
// `![alt](src)` line goldmark wraps in a Paragraph. Such a paragraph must NOT
// emit a text Block: Paragraph.Text reconstructs the image's ALT text, which
// would surface as prose duplicating the image Block's own Text. Exactly
// parallel to isMathOnlyParagraph; returns false for a paragraph mixing prose
// with an image (that prose is real and is kept), or one with no image at all.
func isImageOnlyParagraph(n ast.Node, source []byte) bool {
	hasImage := false
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(*ast.Image); ok {
			hasImage = true
			continue
		}
		if strings.TrimSpace(string(c.Text(source))) != "" {
			return false // a non-image, non-whitespace child -> real prose
		}
	}
	return hasImage
}

// collectTable flattens a GFM table into its header-row cell texts and its
// body rows, each a left-to-right slice of cell texts. A table with no
// *east.TableHeader yields a nil headers slice rather than an empty one, so
// the `omitempty` tag drops the key entirely. Ragged rows are reported as
// authored -- padding/truncating is an exporter's decision, not the model's.
func collectTable(t *east.Table, source []byte) (headers []string, rows [][]string) {
	cellsOf := func(row ast.Node) []string {
		var cells []string
		for c := row.FirstChild(); c != nil; c = c.NextSibling() {
			if cell, ok := c.(*east.TableCell); ok {
				cells = append(cells, string(cell.Text(source)))
			}
		}
		return cells
	}
	for n := t.FirstChild(); n != nil; n = n.NextSibling() {
		switch row := n.(type) {
		case *east.TableHeader:
			headers = cellsOf(row)
		case *east.TableRow:
			rows = append(rows, cellsOf(row))
		}
	}
	return headers, rows
}

// alignNames maps goldmark's per-column Alignment enum onto the stable
// lowercase strings the model exposes. east.AlignNone becomes "" so an
// unaligned column serializes as an empty entry rather than inventing a
// default an exporter might then apply as if it were explicit.
func alignNames(aligns []east.Alignment) []string {
	if len(aligns) == 0 {
		return nil
	}
	out := make([]string, len(aligns))
	for i, a := range aligns {
		switch a {
		case east.AlignLeft:
			out[i] = "left"
		case east.AlignRight:
			out[i] = "right"
		case east.AlignCenter:
			out[i] = "center"
		default:
			out[i] = ""
		}
	}
	return out
}

// quoteText concatenates a blockquote's block-level children into one string,
// separating them by a blank line so a multi-paragraph quote round-trips as
// readable prose. Nested blockquotes are flattened in document order -- the
// model records the quote's TEXT, not a recursive quote tree, which is what
// every current consumer (pptx text boxes, a future docx quote style, the Dart
// render surface) actually needs.
func quoteText(q *ast.Blockquote, source []byte) string {
	var parts []string
	for c := q.FirstChild(); c != nil; c = c.NextSibling() {
		if txt := strings.TrimSpace(string(c.Text(source))); txt != "" {
			parts = append(parts, txt)
		}
	}
	return strings.Join(parts, "\n\n")
}

// rawLinesText reconstructs a code block's RAW source by concatenating the
// value of each of its line segments against source -- the exact pre-chroma
// bytes goldmark parsed, never the highlighted HTML a renderer later produces.
func rawLinesText(lines *text.Segments, source []byte) string {
	var b strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	return b.String()
}

// collectListItems flattens list (and every list nested within its items) into
// a document-ordered []ListItem, each carrying its own leading text and its
// 0-based nesting Level. An item's text is taken from its first TextBlock/
// Paragraph child only (a tight list uses TextBlock, a loose list Paragraph);
// a nested *ast.List child is recursed into at Level+1, appended AFTER its
// owning item so document order is preserved (parent, then its descendants).
func collectListItems(list *ast.List, source []byte) []ListItem {
	var items []ListItem
	var walk func(l *ast.List, level int)
	walk = func(l *ast.List, level int) {
		for li := l.FirstChild(); li != nil; li = li.NextSibling() {
			item, ok := li.(*ast.ListItem)
			if !ok {
				continue
			}
			var itemText string
			var nested []*ast.List
			for c := item.FirstChild(); c != nil; c = c.NextSibling() {
				switch child := c.(type) {
				case *ast.List:
					nested = append(nested, child)
				case *ast.TextBlock, *ast.Paragraph:
					if itemText == "" {
						itemText = string(c.Text(source))
					}
				}
			}
			items = append(items, ListItem{Text: itemText, Level: level})
			for _, nl := range nested {
				walk(nl, level+1)
			}
		}
	}
	walk(list, 0)
	return items
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
