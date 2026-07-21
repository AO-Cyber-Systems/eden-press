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

package press

import "github.com/AO-Cyber-Systems/eden-press/chase/model"

// flattenNotes aggregates every model.Section.Notes entry across d.Sections
// into a single flat []string in document order -- the exact value
// press.Render surfaces as Output.Comments.
//
// This is a PURE AGGREGATION of data model.Build already populated
// (chase/model/build.go walks CommentNode/CommentInline into
// Section.Notes): it never re-walks the AST, never re-runs the isNote /
// directive-vs-note classification, and never touches the goldmark tree at
// all -- a second walk would duplicate chase/model's own note-detection logic
// (03-09 anti_patterns). It reads Notes slices, in Sections order, appending
// each Section's notes after the previous Section's.
//
// It returns nil (not an empty non-nil slice) for a document whose Sections
// carry no notes, mirroring model.Section.Notes' own `omitempty`/nil-when-
// absent contract so the JSON the Dart binding serializes is stable.
func flattenNotes(d *model.Document) []string {
	if d == nil {
		return nil
	}
	var out []string
	for i := range d.Sections {
		out = append(out, d.Sections[i].Notes...)
	}
	return out
}
