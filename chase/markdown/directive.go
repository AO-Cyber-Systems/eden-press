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

// PARSE-04: the directive-apply ASTTransformer. It walks the Section/Comment
// nodes produced by 01-05, builds the ordered event stream chase/directive's
// carry-forward state machine expects (SlideOpen / DirectiveCommentEvent /
// SlideClose, plus any front-matter-derived events BEFORE the first
// SlideOpen), calls chase/directive.Resolve, and hands the resolved
// per-slide directive maps off to apply.go for materialization onto each
// Section node.
//
// This file does NOT re-implement carry-forward -- it only builds the
// ordered event stream and an ordered-KEY companion (chase/directive.Resolve
// returns unordered Go maps by design; the key ORDER used for style/attr
// materialization is tracked here by replaying the identical control flow
// against the same chase/directive.Coerce*/SpotKey functions -- never a
// separate, hand-rolled resolution algorithm).
package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/AO-Cyber-Systems/eden-press/chase/directive"
)

// FrontMatterKey is the parser.Context key the front-matter BlockParser
// stashes its ordered, parsed KV pairs under (chase/directive.KV, in
// declaration order), for the directive-apply transformer to feed into the
// event stream as events occurring BEFORE the first SlideOpen -- exactly
// like Marpit's `if (frontMatterObject.yaml) applyDirectives(...)` call
// (directives/parse.js).
var FrontMatterKey = parser.NewContextKey()

// ThemeExistsKey is an optional parser.Context key a caller MAY set to a
// directive.ThemeExists predicate (e.g. wired to chase/theme's registry in
// a later objective). chase/markdown intentionally has ZERO import of
// chase/theme (RESEARCH "zero cross-import boundary"); when unset, the
// "theme" global directive is resolved permissively (every theme name is
// considered to exist) -- theme *validation* is deferred to whichever
// caller injects a real predicate.
var ThemeExistsKey = parser.NewContextKey()

// frontMatterBlockParser detects and strips a leading "---\n...\n---"
// front-matter block (chase/directive.DetectFrontMatter), storing its
// parsed KV pairs at FrontMatterKey and removing the placeholder node from
// the tree, mirroring the goldmark-meta precedent (yuin/goldmark-meta).
type frontMatterBlockParser struct{}

func newFrontMatterBlockParser() parser.BlockParser {
	return &frontMatterBlockParser{}
}

// Trigger implements parser.BlockParser: front-matter always opens with a
// "---" fence line.
func (b *frontMatterBlockParser) Trigger() []byte {
	return []byte{'-'}
}

// Open implements parser.BlockParser. Front-matter can only ever start at
// the very first line of the document; DetectFrontMatter is holistic (it
// needs the whole remaining source to find the closing fence), so on a
// match the reader is advanced past the ENTIRE front-matter block in one
// shot.
func (b *frontMatterBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.Position()
	if line != 0 {
		return nil, parser.NoChildren
	}

	src := reader.Source()
	body, rest, ok := directive.DetectFrontMatter(string(src))
	if !ok {
		return nil, parser.NoChildren
	}

	consumed := len(src) - len(rest)
	reader.Advance(consumed)
	pc.Set(FrontMatterKey, directive.ParseFrontMatter(body))

	return ast.NewTextBlock(), parser.NoChildren
}

// Continue implements parser.BlockParser: the whole block was already
// consumed in Open, so immediately close.
func (b *frontMatterBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

// Close implements parser.BlockParser: remove the placeholder node -- it
// carries no renderable content of its own (mirrors goldmark-meta's
// `node.Parent().RemoveChild(node.Parent(), node)`).
func (b *frontMatterBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	if p := node.Parent(); p != nil {
		p.RemoveChild(p, node)
	}
}

func (b *frontMatterBlockParser) CanInterruptParagraph() bool { return false }
func (b *frontMatterBlockParser) CanAcceptIndentedLine() bool { return false }

// directiveApplyTransformer builds the ordered event stream from the
// finalized AST (Sections + their Comment descendants) and materializes
// chase/directive's resolved carry-forward output onto each Section.
type directiveApplyTransformer struct{}

func newDirectiveApplyTransformer() parser.ASTTransformer {
	return &directiveApplyTransformer{}
}

// Transform implements parser.ASTTransformer. Registered at priority 300 --
// AFTER slide-split (200) -- so the document's top-level children are
// guaranteed to already be *Section nodes.
func (t *directiveApplyTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	events := buildEventStream(doc, pc)
	themeExists := themeExistsFromContext(pc)

	resolved := directive.Resolve(events, themeExists)
	keysPerSlide := buildOrderedKeysPerSlide(events, themeExists)

	applyDirectives(sectionsOf(doc), resolved, keysPerSlide)
}

// themeExistsFromContext reads an injected directive.ThemeExists predicate
// from pc, defaulting to a permissive always-true predicate when unset
// (chase/markdown has no chase/theme import to validate against).
func themeExistsFromContext(pc parser.Context) directive.ThemeExists {
	if v, ok := pc.Get(ThemeExistsKey).(directive.ThemeExists); ok && v != nil {
		return v
	}
	return func(string) bool { return true }
}

// sectionsOf returns doc's top-level children that are *Section nodes, in
// document order. After slide-split (priority 200) this is EVERY top-level
// child, but the filter is kept explicit/defensive rather than assumed.
func sectionsOf(doc *ast.Document) []*Section {
	var out []*Section
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if s, ok := c.(*Section); ok {
			out = append(out, s)
		}
	}
	return out
}

// buildEventStream walks the document in source order and produces the
// ordered event stream chase/directive.Resolve expects: any front-matter KV
// pairs first (as events occurring before the first SlideOpen), then for
// each Section: SlideOpen, one DirectiveCommentEvent per KV pair found in
// every Comment(Node|Inline) descendant (in document order), then
// SlideClose.
//
// KV pairs are re-derived from CommentNode/CommentInline's Raw field via
// chase/directive.ParseComment -- NOT from the .KV map those nodes already
// carry, which is a lossy map[string]string (unordered, and array values
// are already stringified via fmt.Sprintf, no longer re-parseable as
// []string). Raw is the exact trimmed comment body 01-05's comment parser
// already stored, so this is a clean re-derivation, not a re-implementation
// of comment detection.
func buildEventStream(doc *ast.Document, pc parser.Context) []directive.Event {
	var events []directive.Event

	if kvs, ok := pc.Get(FrontMatterKey).([]directive.KV); ok {
		events = append(events, directive.EventsFromKV(kvs)...)
	}

	for _, sec := range sectionsOf(doc) {
		events = append(events, directive.Event{Kind: directive.SlideOpen})
		events = append(events, collectSectionCommentEvents(sec)...)
		events = append(events, directive.Event{Kind: directive.SlideClose})
	}

	return events
}

// collectSectionCommentEvents walks sec's entire subtree (block AND inline
// descendants) collecting one DirectiveCommentEvent per KV pair from every
// Comment(Node|Inline) found, in document order.
func collectSectionCommentEvents(sec *Section) []directive.Event {
	var events []directive.Event
	_ = ast.Walk(sec, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n == ast.Node(sec) {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *CommentNode:
			events = append(events, directive.EventsFromKV(directive.ParseComment(node.Raw))...)
		case *CommentInline:
			events = append(events, directive.EventsFromKV(directive.ParseComment(node.Raw))...)
		}
		return ast.WalkContinue, nil
	})
	return events
}

// orderedKeys is a minimal ordered-set: first-seen insertion order,
// idempotent touch. It is the parallel companion to chase/directive's
// Go-map-backed Resolve() output -- Resolve() owns VALUES (looked up by
// key, never iterated); orderedKeys owns the deterministic key ORDER the
// style/data-attr materialization loop requires (RESEARCH "Don't Hand-Roll":
// a Go map has no deterministic iteration order).
type orderedKeys struct {
	keys []string
	seen map[string]bool
}

func (o *orderedKeys) touch(k string) {
	if o.seen == nil {
		o.seen = map[string]bool{}
	}
	if !o.seen[k] {
		o.seen[k] = true
		o.keys = append(o.keys, k)
	}
}

// buildOrderedKeysPerSlide replays the exact same control flow as
// chase/directive.Resolve (SlideOpen / DirectiveCommentEvent / SlideClose),
// calling the SAME CoerceGlobal/CoerceLocal/SpotKey functions to decide key
// eligibility, but tracks ORDER instead of VALUES:
//
//   - local directives accumulate in first-seen order across the WHOLE
//     stream (never reset -- they carry forward, mirroring cursor.local).
//   - spot directives accumulate in first-seen order per-slide, reset at
//     every SlideClose (mirroring cursor.spot).
//   - at SlideClose, a slide's key order is {local order} then {spot-only
//     order} -- mirroring JS's `{...cursor.local, ...cursor.spot}` spread
//     merge, where keys already present keep their original position.
//   - global directives accumulate in first-seen order across the WHOLE
//     document, appended onto EVERY slide's order in a trailing pass --
//     mirroring the trailing `{...token.meta.marpitDirectives,
//     ...marpit.lastGlobalDirectives}` merge in directives/parse.js.
func buildOrderedKeysPerSlide(events []directive.Event, themeExists directive.ThemeExists) [][]string {
	globalOrder := &orderedKeys{}
	localOrder := &orderedKeys{}
	spotOrder := &orderedKeys{}

	var slidesOrder [][]string
	current := -1

	for _, ev := range events {
		switch ev.Kind {
		case directive.SlideOpen:
			slidesOrder = append(slidesOrder, nil)
			current = len(slidesOrder) - 1
		case directive.SlideClose:
			if current < 0 {
				continue
			}
			merged := &orderedKeys{}
			for _, k := range localOrder.keys {
				merged.touch(k)
			}
			for _, k := range spotOrder.keys {
				merged.touch(k)
			}
			slidesOrder[current] = merged.keys
			spotOrder = &orderedKeys{}
		case directive.DirectiveCommentEvent:
			if v, isKnown := directive.CoerceGlobal(ev.Key, ev.Raw, themeExists); isKnown && v != nil {
				globalOrder.touch(ev.Key)
			}
			if v, isKnown := directive.CoerceLocal(ev.Key, ev.Raw); isKnown && v != nil {
				localOrder.touch(ev.Key)
			}
			if base, ok := directive.SpotKey(ev.Key); ok {
				if v, isKnown := directive.CoerceLocal(base, ev.Raw); isKnown && v != nil {
					spotOrder.touch(base)
				}
			}
		}
	}

	for i := range slidesOrder {
		merged := &orderedKeys{}
		for _, k := range slidesOrder[i] {
			merged.touch(k)
		}
		for _, k := range globalOrder.keys {
			merged.touch(k)
		}
		slidesOrder[i] = merged.keys
	}
	return slidesOrder
}
