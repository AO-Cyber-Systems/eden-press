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
const SchemaVersion = "eden-press.model/v1"

// Document is the root of the docmodel: deck-level Meta, the ordered list
// of Sections (the generic unit a profile happens to call a "slide" or a
// "page" -- see chase/profile and ARCHITECTURE.md Anti-Pattern 1, which is
// exactly why nothing in this package is ever named Slide), and a flat
// Outline of every heading in the deck.
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
// 1-based ID, its materialized directive-derived attributes, and any
// speaker notes (non-directive HTML comments) found within it.
//
// Blocks/HTML content are deliberately NOT part of this shape -- this
// objective (MODEL-01) only needs outline+notes+meta+attrs; a field with
// no consumer in this TRD's Test list is a speculative superset and does
// not belong here (see build.go's non-mutation invariant tests for what
// IS exercised).
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
