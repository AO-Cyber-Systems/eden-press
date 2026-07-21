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

package math

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// parseMathNodes parses src with the math option and returns every *mathNode in
// document order. It drives the parser directly (Parser().Parse) rather than a
// full Convert, so the $/$$ InlineParser + custom AST node can be exercised in
// isolation from the NodeRenderer (which lands in Task 2).
func parseMathNodes(t *testing.T, opt goldmark.Option, src string) []*mathNode {
	t.Helper()
	md := goldmark.New(opt)
	doc := md.Parser().Parse(text.NewReader([]byte(src)))
	var nodes []*mathNode
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if m, ok := n.(*mathNode); ok {
				nodes = append(nodes, m)
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		t.Fatalf("ast.Walk: %v", err)
	}
	return nodes
}

// TestMathParse drives the bespoke $-trigger InlineParser: inline $…$ and block
// $$…$$ each parse to exactly one mathNode carrying the raw LaTeX and the
// inline/block distinction.
func TestMathParse(t *testing.T) {
	t.Run("inline", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$x^2$`)
		if len(nodes) != 1 {
			t.Fatalf("got %d math nodes, want 1", len(nodes))
		}
		if nodes[0].Raw != "x^2" {
			t.Errorf("Raw = %q, want %q", nodes[0].Raw, "x^2")
		}
		if nodes[0].Block {
			t.Errorf("Block = true, want false for inline $…$")
		}
	})

	t.Run("block", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$$\frac{a}{b}$$`)
		if len(nodes) != 1 {
			t.Fatalf("got %d math nodes, want 1", len(nodes))
		}
		if nodes[0].Raw != `\frac{a}{b}` {
			t.Errorf("Raw = %q, want %q", nodes[0].Raw, `\frac{a}{b}`)
		}
		if !nodes[0].Block {
			t.Errorf("Block = false, want true for block $$…$$")
		}
	})

	t.Run("block_aligned", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$$\begin{aligned}a&=b\\c&=d\end{aligned}$$`)
		if len(nodes) != 1 {
			t.Fatalf("got %d math nodes, want 1", len(nodes))
		}
		if !nodes[0].Block {
			t.Errorf("Block = false, want true")
		}
		if !strings.Contains(nodes[0].Raw, `\begin{aligned}`) {
			t.Errorf("Raw = %q, want it to contain \\begin{aligned}", nodes[0].Raw)
		}
	})

	t.Run("two_inline", func(t *testing.T) {
		nodes := parseMathNodes(t, Option(""), `$a$ and $b$`)
		if len(nodes) != 2 {
			t.Fatalf("got %d math nodes, want 2", len(nodes))
		}
		if nodes[0].Raw != "a" || nodes[1].Raw != "b" {
			t.Errorf("Raw = %q,%q want a,b", nodes[0].Raw, nodes[1].Raw)
		}
	})
}

// TestCurrency is the currency mis-trigger guard (error_recovery): a bare `$`
// used as a currency sign must NOT open math. Per the Pandoc inline-math rule
// the opening `$` needs a non-space to its right AND the closing `$` a non-space
// to its left and no digit to its right — `$5 and $10` satisfies none.
func TestCurrency(t *testing.T) {
	cases := []string{
		`price is $5 and $10`,
		`it costs $100`,
		`from $5 to $10 per item`,
	}
	for _, src := range cases {
		nodes := parseMathNodes(t, Option(""), src)
		if len(nodes) != 0 {
			t.Errorf("parse(%q): got %d math nodes, want 0 (currency must stay literal)", src, len(nodes))
		}
	}
}

// TestMathOff asserts the MathMode "off" composition: Option("off") is a no-op
// goldmark.Option, so the $-parser is never registered and `$x^2$` stays literal
// text (zero math nodes). Full press.Options wiring lands in 03-09.
func TestMathOff(t *testing.T) {
	nodes := parseMathNodes(t, Option("off"), `$x^2$`)
	if len(nodes) != 0 {
		t.Errorf("MathMode off: got %d math nodes, want 0 (math disabled)", len(nodes))
	}
}
