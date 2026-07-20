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

// Package runner is the shared conformance-harness scaffolding that the spec
// sweep (00-04) and the Marp corpus runner (00-05) both build on.
//
// RenderFunc is the pluggable engine seam: Objective 0 passes a goldmark-backed
// func (see GoldmarkRenderFunc / NewGoldmark*), while later objectives pass Eden
// Press's own engine. The corpus format, the DOM-normalized diff (htmldiff), and
// the per-section report never change when the engine is swapped.
//
// RunCase renders a corpus.Case through a RenderFunc, structurally compares the
// output against the golden HTML with htmldiff.Equal, and records the outcome into
// a report.SectionReport.
//
// The shared goldmark constructors (NewGoldmark, NewGoldmarkGFM, NewGoldmarkMarp)
// live in engine.go — the ONE place a goldmark.Markdown is built, so the two
// wave-3 runners import them instead of each defining a local newGM() (which would
// duplicate-symbol on merge).
package runner

import (
	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus"
	"github.com/AO-Cyber-Systems/eden-press/conformance/htmldiff"
	"github.com/AO-Cyber-Systems/eden-press/conformance/report"
)

// RenderFunc renders Markdown to HTML under the given options. It is the pluggable
// engine hook: the harness is engine-agnostic and only depends on this signature.
type RenderFunc func(markdown string, opts map[string]any) (html string, err error)

// RunCase renders c via rf, DOM-normalizes both sides with htmldiff.Equal, and
// records the result into rep under the given section key. It returns whether the
// case passed and, when it did not, a human-readable diff (a render error is
// recorded as a failure and surfaced through diff).
func RunCase(rf RenderFunc, c corpus.Case, section string, rep *report.SectionReport) (pass bool, diff string) {
	out, err := rf(c.InputMD, c.Options)
	if err != nil {
		rep.Add(section, false)
		return false, "render error: " + err.Error()
	}
	eq, d := htmldiff.Equal(c.ExpectedHTML, out)
	rep.Add(section, eq)
	return eq, d
}
