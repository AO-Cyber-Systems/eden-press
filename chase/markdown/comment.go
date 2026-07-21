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
	"fmt"
	"regexp"
	"strings"

	"github.com/AO-Cyber-Systems/eden-press/chase/directive"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// commentBlockOpenRegexp mirrors goldmark's own htmlBlockType2OpenRegexp
// (parser/html_block.go) -- up to 3 leading spaces, then the literal "<!--"
// opening marker. Matching this exact shape (rather than something looser)
// is what lets our BlockParser and goldmark's built-in HTMLBlockParser race
// on the SAME trigger byte ('<') for the SAME candidate lines.
var commentBlockOpenRegexp = regexp.MustCompile(`^[ ]{0,3}<!--`)

// commentBlockClose is the literal HTML-comment closing marker.
var commentBlockClose = []byte("-->")

// commentBlockParser detects a (possibly multi-line) HTML comment as a
// block-level *CommentNode -- DETECTION ONLY (see CommentNode doc comment).
// It must be registered at a LOWER priority number than goldmark's built-in
// HTMLBlockParser (900, parser.DefaultBlockParsers) so it wins the race for
// any line starting with "<!--" (RESEARCH: "Pattern: HTML-Comment Directive
// Detection").
type commentBlockParser struct{}

// newCommentBlockParser returns a new block-level comment BlockParser.
func newCommentBlockParser() parser.BlockParser {
	return &commentBlockParser{}
}

func (b *commentBlockParser) Trigger() []byte {
	return []byte{'<'}
}

func (b *commentBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	if !commentBlockOpenRegexp.Match(line) {
		return nil, parser.NoChildren
	}
	node := NewCommentNode("", nil)
	reader.AdvanceToEOL()
	node.Lines().Append(segment)
	return node, parser.NoChildren
}

// Continue mirrors parser.htmlBlockParser's HTMLBlockType2 handling
// (multi-line-aware scan to a matching "-->"), simplified because a hidden
// CommentNode never needs to preserve a separate "ClosureLine" for
// rendering -- every consumed line (including the one carrying the closing
// marker) is appended to node.Lines() so Close can hand the FULL raw span to
// chase/directive.DetectComment.
func (b *commentBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	cn := node.(*CommentNode)
	lines := cn.Lines()

	if lines.Len() == 1 {
		firstLine := lines.At(0)
		if bytes.Contains(firstLine.Value(reader.Source()), commentBlockClose) {
			return parser.Close
		}
	}

	line, segment := reader.PeekLine()
	if line == nil {
		return parser.Close
	}
	reader.AdvanceToEOL()
	lines.Append(segment)
	if bytes.Contains(line, commentBlockClose) {
		return parser.Close
	}
	return parser.Continue | parser.NoChildren
}

func (b *commentBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	cn := node.(*CommentNode)
	var buf bytes.Buffer
	lines := cn.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(reader.Source()))
	}
	cn.Raw, cn.KV = parseCommentBytes(buf.Bytes())
}

func (b *commentBlockParser) CanInterruptParagraph() bool {
	return true
}

func (b *commentBlockParser) CanAcceptIndentedLine() bool {
	return false
}

// openComment / closeComment are the raw markers the inline parser scans
// for, mirroring parser.rawHTMLParser.parseComment's own openComment/
// closeComment byte slices (parser/raw_html.go) -- copied verbatim so our
// inline scan matches goldmark's own multi-line accumulation shape exactly.
var openComment = []byte("<!--")
var closeComment = []byte("-->")

// commentInlineParser detects a directive comment appearing mid-paragraph
// (Test-list case 6) as an inline *CommentInline node -- DETECTION ONLY.
// Must be registered at a lower priority number than goldmark's built-in
// RawHTMLParser (400, parser.DefaultInlineParsers) so it wins the trigger
// race for "<!--".
type commentInlineParser struct{}

// newCommentInlineParser returns a new inline comment InlineParser.
func newCommentInlineParser() parser.InlineParser {
	return &commentInlineParser{}
}

func (s *commentInlineParser) Trigger() []byte {
	return []byte{'<'}
}

// Parse is a direct structural port of parser.rawHTMLParser.parseComment
// (parser/raw_html.go), except it also accumulates the raw comment span
// into a buffer (for chase/directive.DetectComment/ParseComment) instead of
// only recording Segments for re-emission.
func (s *commentInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	savedLine, savedSegment := block.Position()
	line, segment := block.PeekLine()
	if !bytes.HasPrefix(line, openComment) {
		return nil
	}

	var raw bytes.Buffer
	offset := len(openComment)
	search := line[offset:]
	for {
		index := bytes.Index(search, closeComment)
		if index > -1 {
			full := segment.WithStop(segment.Start + offset + index + len(closeComment))
			raw.Write(full.Value(block.Source()))
			block.Advance(offset + index + len(closeComment))
			body, kv := parseCommentBytes(raw.Bytes())
			return NewCommentInline(body, kv)
		}
		raw.Write(segment.Value(block.Source()))
		offset = 0
		block.AdvanceLine()
		line, segment = block.PeekLine()
		if line == nil {
			break
		}
		search = line
	}
	block.SetPosition(savedLine, savedSegment)
	return nil
}

// parseCommentBytes trims raw to its first '<' (defensively -- both callers
// already start exactly at "<!--", but this keeps the function safe to
// reuse against any leading-indentation-containing span) and hands it to
// chase/directive.DetectComment + ParseComment. This is the ONE place
// chase/markdown crosses into chase/directive -- everything past this call
// is pure string/KV data, per the zero-cross-import boundary
// (01-RESEARCH.md "chase/directive ... zero goldmark import").
//
// A recognized KV whose value is not a plain string (i.e. a parsed flow
// list, "[a, b]") is stored as its fmt.Sprintf("%v", ...) rendering in the
// map[string]string this TRD's CommentNode/CommentInline carry -- real type
// coercion happens later via chase/directive.CoerceGlobal/CoerceLocal
// (TRD 01-06), which consumes directive.RawValue directly, not this map.
func parseCommentBytes(raw []byte) (body string, kv map[string]string) {
	text := string(raw)
	idx := strings.IndexByte(text, '<')
	if idx < 0 {
		return "", nil
	}
	text = text[idx:]

	b, ok := directive.DetectComment(text)
	if !ok {
		return "", nil
	}
	kvs := directive.ParseComment(b)
	if len(kvs) == 0 {
		return b, nil
	}
	m := make(map[string]string, len(kvs))
	for _, item := range kvs {
		if s, isStr := item.Val.(string); isStr {
			m[item.Key] = s
		} else {
			m[item.Key] = fmt.Sprintf("%v", item.Val)
		}
	}
	return b, m
}
