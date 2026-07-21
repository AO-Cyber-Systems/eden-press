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

// Package math is CORE-08's BASELINE math battery: a self-contained
// goldmark.Option that renders Pandoc/Marp-style `$…$` (inline) and `$$…$$`
// (block) LaTeX. Common math becomes native MathML via the vendored
// latex2mathml; heavy constructs the construct-detection predicate flags
// (detect.go) degrade to a PNG-only raster <img> via go-latex/latex.
//
// No reusable goldmark-integration library exists for this layer (unlike emoji
// or chroma highlighting), so the $-trigger InlineParser, the custom math AST
// node, and the NodeRenderer are a genuine from-scratch build, shaped after the
// bespoke inline parsers in chase/markdown.
package math

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mathNode is the custom inline AST node the $-parser emits. Raw is the LaTeX
// between the delimiters (never re-parsed by goldmark — the NodeRenderer owns
// it); Block distinguishes $$…$$ (display) from $…$ (inline).
type mathNode struct {
	ast.BaseInline
	Raw   string
	Block bool
}

// KindMath is the NodeKind for mathNode.
var KindMath = ast.NewNodeKind("Math")

// Kind implements ast.Node.
func (n *mathNode) Kind() ast.NodeKind { return KindMath }

// Dump implements ast.Node.
func (n *mathNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Raw":   n.Raw,
		"Block": boolStr(n.Block),
	}, nil)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func newMathNode(raw string, block bool) *mathNode {
	return &mathNode{Raw: raw, Block: block}
}

// dollarDollar is the display-math delimiter.
var dollarDollar = []byte("$$")

// mathInlineParser is the bespoke $-trigger InlineParser. It recognizes both
// inline `$…$` and display `$$…$$` from the single '$' trigger byte, emitting a
// mathNode carrying the raw LaTeX. It performs NO conversion — routing and
// rendering are the NodeRenderer's job (mathml.go / fallback.go, Task 2).
type mathInlineParser struct{}

func (p *mathInlineParser) Trigger() []byte { return []byte{'$'} }

// Parse consumes an inline or display math span at the current reader position.
// Returning nil leaves the '$' as literal text (goldmark advances past it),
// which is exactly the currency mis-trigger back-off.
func (p *mathInlineParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 2 || line[0] != '$' {
		return nil
	}
	if line[1] == '$' {
		return p.parseDisplay(block, line)
	}
	return p.parseInline(block, line)
}

// parseInline handles `$…$`, applying the Pandoc inline-math delimiter rule as
// the currency guard: the opening `$` must be followed by a non-space, and the
// closing `$` must be preceded by a non-space and NOT followed by a digit. Any
// violation backs off to literal text (nil) — `$5 and $10` never becomes math.
func (p *mathInlineParser) parseInline(block text.Reader, line []byte) ast.Node {
	if util.IsSpace(line[1]) {
		return nil // opening `$` followed by space → not math (Pandoc rule)
	}
	for i := 1; i < len(line); i++ {
		if line[i] != '$' {
			continue
		}
		if util.IsSpace(line[i-1]) {
			return nil // closing `$` preceded by space → not a valid close
		}
		if i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9' {
			return nil // closing `$` followed by a digit → currency, not math
		}
		raw := string(line[1:i])
		block.Advance(i + 1) // consume `$` + content + `$`
		return newMathNode(raw, false)
	}
	return nil // no closing `$` on this line → literal
}

// parseDisplay handles `$$…$$`. It scans the current line first, then across
// subsequent lines of the same block (mirroring chase/markdown's comment inline
// parser) so multi-line display math parses; if no closing `$$` is found it
// restores the reader and backs off to literal text.
func (p *mathInlineParser) parseDisplay(block text.Reader, line []byte) ast.Node {
	rest := line[2:]
	if idx := bytes.Index(rest, dollarDollar); idx >= 0 {
		raw := string(rest[:idx])
		block.Advance(2 + idx + 2) // `$$` + content + `$$`, all on one line
		return newMathNode(raw, true)
	}

	savedLine, savedSeg := block.Position()
	var buf bytes.Buffer
	buf.Write(rest) // first line's post-`$$` content (includes its newline)
	block.AdvanceLine()
	for {
		line, _ := block.PeekLine()
		if line == nil {
			block.SetPosition(savedLine, savedSeg)
			return nil // no closing `$$` in this block → literal
		}
		if idx := bytes.Index(line, dollarDollar); idx >= 0 {
			buf.Write(line[:idx])
			block.Advance(idx + 2)
			return newMathNode(buf.String(), true)
		}
		buf.Write(line)
		block.AdvanceLine()
	}
}

// mathExtension is the goldmark.Extender the math battery ships. It registers
// the bespoke $-parser AND the routing NodeRenderer (mathRenderer, mathml.go)
// that sends each mathNode through the detection predicate to MathML (mathml.go)
// or the PNG fallback (fallback.go).
type mathExtension struct{}

// Extend implements goldmark.Extender.
func (e *mathExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&mathInlineParser{}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&mathRenderer{}, 500),
	))
}

// Option returns the composable goldmark.Option for the math battery, honoring
// press.Options.MathMode: "" (or "mathml") enables the battery; "off" disables
// it entirely (a no-op Option that never registers the $-parser, so `$x$` stays
// literal text). TRD 03-09 folds this into the one-parse engine's pressExtraOpts.
func Option(mode string) goldmark.Option {
	if mode == "off" {
		return goldmark.WithExtensions() // no-op: math disabled
	}
	return goldmark.WithExtensions(&mathExtension{})
}
