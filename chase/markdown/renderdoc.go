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

	"github.com/yuin/goldmark/ast"
)

// RenderDoc renders doc -- an ALREADY-parsed *ast.Document, e.g. the value
// returned by Parse (seam.go) -- via the SAME defaultEngine Render (Phase
// 2 of the two-phase seam) drives, WITHOUT ever calling Parse again.
//
// This is the seam Objective-2's chase.go entrypoint (02-04, MODEL-02)
// needs: Render(md, opts) (seam.go) re-parses md internally every time,
// which is unusable for a caller that must feed the SAME finalized tree
// to two independent sinks (HTML + chase/model.Build) from a single parse.
// RenderDoc lets a caller who already holds a doc (from its own Parse
// call) render that EXACT tree instead.
//
// source must be the identical byte slice doc was parsed from -- the same
// requirement chase/model.Build documents (goldmark's
// ast.Node.Text/AttributeString resolve against source-indexed spans, not
// portable strings on their own).
//
// CRITICAL: RenderDoc must NOT call Parse -- its whole reason to exist is
// to avoid the second parse markdown.Render performs.
func RenderDoc(doc *ast.Document, source []byte) (string, error) {
	var buf bytes.Buffer
	if err := defaultEngine.Renderer().Render(&buf, source, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}
