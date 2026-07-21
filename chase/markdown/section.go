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
	"strconv"

	"github.com/yuin/goldmark/ast"
)

// KindSection is the ast.NodeKind of a *Section node.
var KindSection = ast.NewNodeKind("MarpitSection")

// Attr is a single ordered HTML attribute (name/value) attached to a Section.
// An ordered slice -- not a map -- is used deliberately: declaration order in
// an HTML attribute list (and in a "style" attribute's custom-property list)
// is significant, and a Go map has no deterministic iteration order
// (01-RESEARCH.md "Directive -> Attribute/Style materialization").
//
// Attrs is the clean extension point TRD 01-06 populates (dataset/style
// directive attrs); this TRD leaves it empty for every Section it creates.
type Attr struct {
	Name  string
	Value string
}

// Section is the chase/markdown AST node produced by the slide-split
// ASTTransformer (slide.go): it wraps one Marpit slide's block-level
// content. ID is the slide's 1-based index (RESEARCH: the corpus uses
// id="{index+1}", first slide -> "1").
//
// Section embeds ast.BaseBlock because it wraps block-level content
// (headings, paragraphs, lists, ...) exactly like the ast.Document root it
// replaces children of.
type Section struct {
	ast.BaseBlock

	// ID is the slide's 1-based index.
	ID int

	// Attrs holds directive-derived HTML attributes, in declaration order.
	// Populated by TRD 01-06 (dataset/style attrs); empty in this TRD.
	Attrs []Attr
}

// NewSection returns a new, empty *Section node with the given 1-based id.
func NewSection(id int) *Section {
	return &Section{ID: id}
}

// Kind implements ast.Node.
func (n *Section) Kind() ast.NodeKind {
	return KindSection
}

// Dump implements ast.Node.
func (n *Section) Dump(source []byte, level int) {
	m := map[string]string{"ID": strconv.Itoa(n.ID)}
	for _, a := range n.Attrs {
		m[a.Name] = a.Value
	}
	ast.DumpHelper(n, source, level, m, nil)
}

// KindCommentNode is the ast.NodeKind of a block-level *CommentNode.
var KindCommentNode = ast.NewNodeKind("MarpitComment")

// CommentNode is a hidden block-level HTML comment carrying raw
// (not-yet-recognized) directive key/value data.
//
// This is DETECTION ONLY -- per RESEARCH Pitfall 4, chase/markdown's comment
// parsers extract raw key/value pairs onto this node without deciding
// whether any key is a recognized directive; that recognition step is TRD
// 01-06's job (a later ASTTransformer that reads KV and calls into
// chase/directive's carry-forward machine).
type CommentNode struct {
	ast.BaseBlock

	// Raw is the comment's trimmed inner body (the text between "<!--" and
	// "-->", as returned by chase/directive.DetectComment).
	Raw string

	// KV holds the raw, not-yet-recognized key/value pairs extracted from
	// Raw by chase/directive.ParseComment. A non-directive comment (e.g.
	// "<!-- just a note -->") is still detected and produces a CommentNode,
	// just with an empty (or partial) KV map.
	KV map[string]string

	// Hidden reports that this node must never contribute visible text to
	// rendered output. Always true for a detected comment; kept as an
	// explicit field (rather than an implicit render-time assumption) so a
	// later "well-known magic comment" passthrough (RESEARCH Open Question,
	// out of scope for this TRD) has a place to override it.
	Hidden bool
}

// NewCommentNode returns a new block-level *CommentNode.
func NewCommentNode(raw string, kv map[string]string) *CommentNode {
	return &CommentNode{Raw: raw, KV: kv, Hidden: true}
}

// Kind implements ast.Node.
func (n *CommentNode) Kind() ast.NodeKind {
	return KindCommentNode
}

// Dump implements ast.Node.
func (n *CommentNode) Dump(source []byte, level int) {
	m := map[string]string{"Raw": n.Raw}
	ast.DumpHelper(n, source, level, m, nil)
}

// KindCommentInline is the ast.NodeKind of the inline counterpart of
// CommentNode -- a directive comment appearing mid-paragraph (Test-list
// case 6: "text <!-- x --> more").
var KindCommentInline = ast.NewNodeKind("MarpitCommentInline")

// CommentInline is the inline sibling of CommentNode; same detection-only
// contract, embedding ast.BaseInline because it appears inside inline
// (paragraph/text) content rather than as a standalone block.
type CommentInline struct {
	ast.BaseInline

	Raw    string
	KV     map[string]string
	Hidden bool
}

// NewCommentInline returns a new inline *CommentInline.
func NewCommentInline(raw string, kv map[string]string) *CommentInline {
	return &CommentInline{Raw: raw, KV: kv, Hidden: true}
}

// Kind implements ast.Node.
func (n *CommentInline) Kind() ast.NodeKind {
	return KindCommentInline
}

// Dump implements ast.Node.
func (n *CommentInline) Dump(source []byte, level int) {
	m := map[string]string{"Raw": n.Raw}
	ast.DumpHelper(n, source, level, m, nil)
}
