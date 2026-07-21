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

package directive

// EventKind enumerates the ordered event stream that Resolve walks. These
// events are produced by chase/markdown's tree walk in a later objective;
// this package only ever consumes the abstract stream below -- never
// goldmark types (zero cross-import boundary).
type EventKind int

const (
	// SlideOpen marks the start of a new slide's directive scope.
	SlideOpen EventKind = iota
	// SlideClose merges {...local, ...spot} onto the current slide, then
	// resets spot to empty (single-slide-only scoping).
	SlideClose
	// DirectiveCommentEvent carries one raw key/value pair from a directive
	// comment (or a front-matter-derived candidate, if it occurs before the
	// first SlideOpen). One Event per key, not one Event per comment --
	// mirrors iterating Object.keys(parsedDirectives) in parse.js.
	DirectiveCommentEvent
)

// Event is one item in the ordered event stream driving carry-forward
// resolution.
type Event struct {
	Kind EventKind
	Key  string
	Raw  RawValue
}

// EventsFromKV converts an ordered slice of raw key/value pairs (as produced
// by ParseComment/ParseFrontMatter) into an ordered slice of
// DirectiveCommentEvent Events, preserving declaration order.
func EventsFromKV(kvs []KV) []Event {
	return nil
}

// Resolve walks an ordered event stream and returns one resolved directive
// map per slide (in slide order).
func Resolve(events []Event, themeExists ThemeExists) []map[string]any {
	return nil
}
