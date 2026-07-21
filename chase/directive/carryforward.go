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
// DirectiveCommentEvent Events, preserving declaration order (RESEARCH
// anti-pattern: merge order is significant -- never re-derive it from an
// unordered map).
func EventsFromKV(kvs []KV) []Event {
	events := make([]Event, len(kvs))
	for i, kv := range kvs {
		events[i] = Event{Kind: DirectiveCommentEvent, Key: kv.Key, Raw: kv.Val}
	}
	return events
}

// Resolve walks an ordered event stream (SlideOpen / SlideClose /
// DirectiveCommentEvent) and returns one resolved directive map per slide,
// applying Marpit's exact carry-forward semantics -- verbatim from
// directives/parse.js's 'marpit_directives_parse' +
// 'marpit_directives_global_parse' core rules (01-RESEARCH.md "chase/directive
// carry-forward state machine"):
//
//   - local directives persist in cursor.local across slides -- never reset
//     except by being overridden by a later directive of the same key.
//   - spot directives (a DirectiveCommentEvent whose Key is "_"-prefixed)
//     are collected into cursor.spot, merged into ONLY the current slide at
//     SlideClose, then cursor.spot is reset to empty -- this reset is what
//     makes spot directives single-slide-only.
//   - global directives are resolved (via CoerceGlobal) across the WHOLE
//     event stream and stamped onto EVERY slide identically, AFTER the
//     local/spot loop -- mirroring parse.js's trailing
//     `for (const token of slides) token.meta.marpitDirectives = {...}` loop.
//
// A DirectiveCommentEvent occurring BEFORE the first SlideOpen (i.e. a
// front-matter-derived event) is treated exactly like a comment at the top
// of the document: it can seed cursor.local (if the key is a recognized
// local directive) AND/OR contribute to the document-wide globals map (if
// the key is a recognized global directive) -- mirroring Marpit's
// `if (frontMatterObject.yaml) applyDirectives(...)` call, which appears in
// BOTH the global-parse and local-parse core rules.
func Resolve(events []Event, themeExists ThemeExists) []map[string]any {
	globals := map[string]any{}
	local := map[string]any{}
	spot := map[string]any{}

	var slides []map[string]any
	current := -1

	for _, ev := range events {
		switch ev.Kind {
		case SlideOpen:
			slides = append(slides, map[string]any{})
			current = len(slides) - 1
		case SlideClose:
			if current < 0 {
				continue // malformed stream: close without a matching open
			}
			for k, v := range local {
				slides[current][k] = v
			}
			for k, v := range spot {
				slides[current][k] = v
			}
			spot = map[string]any{}
		case DirectiveCommentEvent:
			if v, isKnown := CoerceGlobal(ev.Key, ev.Raw, themeExists); isKnown && v != nil {
				globals[ev.Key] = v
			}
			if v, isKnown := CoerceLocal(ev.Key, ev.Raw); isKnown && v != nil {
				local[ev.Key] = v
			}
			if base, ok := SpotKey(ev.Key); ok {
				if v, isKnown := CoerceLocal(base, ev.Raw); isKnown && v != nil {
					spot[base] = v
				}
			}
		}
	}

	for _, s := range slides {
		for k, v := range globals {
			s[k] = v
		}
	}
	return slides
}
