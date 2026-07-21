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
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// parseDoc runs ONLY Phase 1 (Parser().Parse()) of the two-phase seam --
// never md.Convert() (RESEARCH/anti_patterns: "DO NOT run md.Convert() in
// production paths").
func parseDoc(md goldmark.Markdown, src string) ast.Node {
	reader := text.NewReader([]byte(src))
	return md.Parser().Parse(reader)
}

// findNode returns the first node of the given kind found via a pre-order
// walk, or nil if none exists.
func findNode(doc ast.Node, kind ast.NodeKind) ast.Node {
	var found ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == kind {
			found = n
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

// Test-list case 5: `<!-- foo: bar -->` -> a hidden CommentNode carrying raw
// kv {foo: bar}; not rendered as visible text; not yet recognized as a
// directive.
func TestCommentBlockDetection(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "<!-- foo: bar -->\n")

	n := findNode(doc, KindCommentNode)
	if n == nil {
		t.Fatalf("expected a CommentNode, found none in:\n%v", doc)
	}
	cn, ok := n.(*CommentNode)
	if !ok {
		t.Fatalf("expected *CommentNode, got %T", n)
	}
	if !cn.Hidden {
		t.Fatalf("expected CommentNode.Hidden == true")
	}
	if got, want := cn.KV["foo"], "bar"; got != want {
		t.Fatalf("KV[\"foo\"] = %q, want %q (KV=%v)", got, want, cn.KV)
	}

	// GOTCHA (task 1): assert the produced node KIND is CommentNode, NOT
	// goldmark's own HTMLBlock -- i.e. our parser won the priority race.
	if bad := findNode(doc, ast.KindHTMLBlock); bad != nil {
		t.Fatalf("comment was parsed as ast.HTMLBlock, not CommentNode -- priority race lost")
	}
}

// Test-list case 6: inline comment mid-paragraph (`text <!-- x --> more`) ->
// detected as an inline CommentNode.
func TestCommentInlineDetection(t *testing.T) {
	md := goldmark.New(goldmark.WithExtensions(New()))
	doc := parseDoc(md, "text <!-- x --> more\n")

	n := findNode(doc, KindCommentInline)
	if n == nil {
		t.Fatalf("expected a CommentInline node, found none in:\n%v", doc)
	}
	ci, ok := n.(*CommentInline)
	if !ok {
		t.Fatalf("expected *CommentInline, got %T", n)
	}
	if !ci.Hidden {
		t.Fatalf("expected CommentInline.Hidden == true")
	}
	if ci.Raw != "x" {
		t.Fatalf("Raw = %q, want %q", ci.Raw, "x")
	}

	if bad := findNode(doc, ast.KindRawHTML); bad != nil {
		t.Fatalf("inline comment was parsed as ast.RawHTML, not CommentInline -- priority race lost")
	}
}
