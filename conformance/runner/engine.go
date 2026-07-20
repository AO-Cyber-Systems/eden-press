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

package runner

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

// engine.go is the SINGLE place a goldmark.Markdown is constructed for the
// conformance harness. The spec sweep (00-04) and the Marp corpus runner (00-05)
// import these constructors; neither defines its own newGM(), which would
// duplicate-symbol on merge.

// NewGoldmark returns a base CommonMark goldmark engine with no extensions — used
// by the CommonMark spec suite.
func NewGoldmark() goldmark.Markdown {
	return goldmark.New()
}

// NewGoldmarkGFM returns a goldmark engine with the GitHub Flavored Markdown
// extension (tables, strikethrough, linkify, task lists) and raw-HTML passthrough
// (WithUnsafe) — used by the GFM / extensions spec suite.
func NewGoldmarkGFM() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(ghtml.WithUnsafe()),
	)
}

// NewGoldmarkMarp returns a goldmark engine configured to approximate Marp Core's
// markdown-it setup: GFM + raw-HTML passthrough (WithUnsafe) + hard line breaks
// (WithHardWraps). This matches the prior-art spike's newGM() and is the engine
// the Marp corpus runner (00-05) uses.
func NewGoldmarkMarp() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(ghtml.WithUnsafe(), ghtml.WithHardWraps()),
	)
}

// GoldmarkRenderFunc adapts a goldmark.Markdown into a runner.RenderFunc so it can
// be driven by RunCase. The opts argument is accepted for interface compatibility;
// goldmark options are baked into the engine at construction time, so opts is not
// consulted here.
func GoldmarkRenderFunc(md goldmark.Markdown) RenderFunc {
	return func(markdown string, opts map[string]any) (string, error) {
		var buf bytes.Buffer
		if err := md.Convert([]byte(markdown), &buf); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
}
