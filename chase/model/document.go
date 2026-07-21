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

// Package model is Eden Press's own versioned, JSON-serializable document
// model (the "docmodel"): Document{Meta, Sections, Outline}. This is
// differentiator #3 ("output-as-data") made a first-class package on day
// one, per ARCHITECTURE.md Pattern 3.
//
// goldmark's ast.Node tree is a transient parse-time working structure: it
// references source-buffer segments rather than owning copies of text, has
// no built-in JSON marshaling, and its shape is an internal implementation
// detail of whichever goldmark version chase/markdown happens to be built
// against. Document is the durable product a consumer (Eden-Biz, AOCore, an
// LLM ingestion pipeline) actually wants -- outline/notes/metadata WITHOUT
// parsing rendered HTML back out of HTML.
//
// Document is built by Build (see build.go): a read-only, direct recursive
// walk of the SAME finalized *ast.Document chase/markdown/seam.go's Parse
// returns -- the exact tree chase/markdown's own goldmark renderer
// consumes to produce HTML. Never a second parse; never reverse-engineered
// from rendered HTML.
package model

// SchemaVersion identifies the shape of Document below. It is owned and
// versioned independently of goldmark/Marp upstream churn (ARCHITECTURE.md
// Pattern 3's trade-off note: "a schema you now own and must version"). A
// future incompatible change to Document's JSON shape bumps this constant,
// giving a consumer a stable signal to branch on rather than silently
// misinterpreting an old/new shape.
//
// v1 -> v2 (06-01): ADDITIVE per-section body-content enrichment. Section
// gained a Blocks []Block field (ordered paragraph/list/code/math/heading
// content, materialized by the SAME single read-only Build walk). Every new
// field carries `omitempty`, so a v1-shaped (block-less) document's JSON is
// byte-for-byte unchanged apart from this version string -- v2 is a strict
// superset of v1, never a breaking reshape.
const SchemaVersion = "eden-press.model/v2"

// BlockKind enumerates the kinds of body-content block a Section can carry.
type BlockKind string

const (
	// BlockParagraph is a plain-text paragraph (editable prose).
	BlockParagraph BlockKind = "paragraph"
	// BlockList is a bullet or ordered list of items.
	BlockList BlockKind = "list"
	// BlockCode is a fenced or indented code block (raw source + language).
	BlockCode BlockKind = "code"
	// BlockMath is a `$…$`/`$$…$$` math construct (raw TeX + display flag).
	BlockMath BlockKind = "math"
	// BlockHeading is a heading (level + text); mirrors an Outline entry.
	BlockHeading BlockKind = "heading"
)

// Block is one ordered piece of a Section's body content, materialized by the
// same read-only AST walk that builds Sections/Outline (see build.go). Its
// zero-value fields are `omitempty` so a Section with no blocks serializes
// without a `blocks` key, and a block that does not use a field (e.g. a
// paragraph has no Language) omits it -- keeping v2 JSON a strict, additive
// superset of v1.
//
// This is the ONE shared schema-v2 surface both downstream objectives consume:
// Objective 6's PPTX writer maps paragraph/list/heading blocks into editable
// text-box shapes; Objective 7's DART-04 serializes code (source+language) and
// math (rawTeX+display) blocks verbatim for native Flutter rendering. Both the
// raw code source and the raw math TeX are JSON-native and lossless here --
// neither is recoverable from the rendered Output.HTML.
type Block struct {
	// Kind is the block's discriminator (paragraph/list/code/math/heading).
	Kind BlockKind `json:"kind"`

	// Text is the block's primary text payload: paragraph/heading plain text,
	// a code block's RAW source (pre-chroma), or a math construct's RAW TeX.
	// Empty for a pure list block (its content lives in Items).
	Text string `json:"text,omitempty"`

	// Level is the heading level 1-6 (heading blocks only).
	Level int `json:"level,omitempty"`

	// Language is the fenced-code info-string language, e.g. "go" (code blocks
	// only; empty for an indented code block or a fence with no info string).
	Language string `json:"language,omitempty"`

	// Display reports display ($$…$$) vs. inline ($…$) math (math blocks only).
	Display bool `json:"display,omitempty"`

	// Ordered reports a numbered vs. bullet list (list blocks only).
	Ordered bool `json:"ordered,omitempty"`

	// Items are the entries of a list block, in document order (list blocks
	// only).
	Items []ListItem `json:"items,omitempty"`
}

// ListItem is one entry of a list Block, carrying its editable Text and its
// 0-based nesting Level (0 = top level, 1 = one level of indentation, ...).
type ListItem struct {
	// Text is the item's editable plain text.
	Text string `json:"text"`

	// Level is the item's 0-based nesting depth within the list.
	Level int `json:"level,omitempty"`
}

// Document is the root of the docmodel: deck-level Meta, the ordered list
// of Sections (the generic unit a profile happens to give its own presentation
// name to -- see chase/profile and ARCHITECTURE.md Anti-Pattern 1, which is
// exactly why this package never bakes in a profile-specific name for it),
// and a flat Outline of every heading in the deck.
type Document struct {
	// SchemaVersion is always the package constant SchemaVersion; present
	// as a field (not just a package constant) so it round-trips through
	// JSON along with the rest of the document.
	SchemaVersion string `json:"schemaVersion"`

	// Meta carries deck-level metadata resolved from front matter.
	Meta Meta `json:"meta"`

	// Sections is one entry per Section, in document order. Nil (not an
	// empty, non-nil slice) for a deck with no Sections at all, e.g. an
	// empty source document.
	Sections []Section `json:"sections"`

	// Outline lists every heading across the whole Document in document
	// order, each entry carrying the SectionID of its owning Section --
	// i.e. already grouped by owning Section, without needing a nested
	// shape.
	Outline []OutlineEntry `json:"outline"`
}

// Section is the JSON-serializable projection of a *markdown.Section: its
// 1-based ID, its materialized directive-derived attributes, any speaker
// notes (non-directive HTML comments) found within it, and -- as of schema
// v2 (06-01) -- its ordered body-content Blocks.
//
// Blocks is ADDITIVE (schema v2): the ID/Attrs/Notes fields, their JSON tags,
// and their order are FROZEN v1 shape and unchanged. Blocks carries `omitempty`
// so a Section with no extracted body content serializes exactly as it did
// under v1 (no `blocks` key). Build materializes Blocks in the SAME single
// read-only AST walk that produces ID/Attrs/Notes/Outline -- never a second
// parse, never reverse-engineered from rendered HTML.
type Section struct {
	// ID is the Section's 1-based index, mirroring *markdown.Section.ID.
	ID int `json:"id"`

	// Attrs holds directive-derived attribute values (data-*/style/class/
	// lang/...), materialized from *markdown.Section.Attrs. A Go map is
	// fine here: declaration ORDER only matters for HTML attribute
	// emission (chase/markdown's job, done before this package ever sees
	// the Section), not for this JSON sink.
	Attrs map[string]string `json:"attrs,omitempty"`

	// Notes holds this Section's speaker notes: the trimmed body of every
	// HTML comment within it that did NOT resolve to a recognized
	// directive, in document order.
	Notes []string `json:"notes,omitempty"`

	// Blocks holds this Section's ordered body-content blocks (paragraph,
	// list, code, math, heading), in document order -- the editable/lossless
	// per-section surface schema v2 adds for downstream data sinks (PPTX text
	// shapes, native Flutter rendering). Nil (and omitted from JSON) for a
	// Section with no extracted body content.
	Blocks []Block `json:"blocks,omitempty"`
}

// Meta carries deck-level metadata resolved from the document's front
// matter (theme, size, class, and any other front-matter directive
// present) -- read from the parser.Context state chase/markdown's
// front-matter block parser already produced, never re-derived from
// rendered HTML.
type Meta struct {
	// Directives holds every front-matter key/value pair, materialized to
	// a string. A Go map is fine here for the same reason as
	// Section.Attrs: declaration order does not matter for a JSON sink.
	Directives map[string]string `json:"directives,omitempty"`
}

// OutlineEntry is one heading in Document.Outline: its owning Section's ID,
// heading Level (1-6), materialized Text, and AutoHeadingID Slug.
type OutlineEntry struct {
	// SectionID is the ID of the Section that owns this heading.
	SectionID int `json:"sectionId"`

	// Level is the heading's level, 1-6.
	Level int `json:"level"`

	// Text is the heading's materialized text content -- an owned string,
	// never a text.Segment held past the AST walk that produced it.
	Text string `json:"text"`

	// Slug is the heading's AutoHeadingID "id" attribute value, or "" if
	// the heading had none (e.g. an empty heading -- see build.go's
	// headingSlug).
	Slug string `json:"slug"`
}
