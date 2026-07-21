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

// CORE-06's bespoke half: literal unicode emoji typed directly in prose (as
// opposed to ":shortcode:", which emoji.go's reused goldmark-emoji parser
// already handles). goldmark-emoji's InlineParser triggers ONLY on ':' -- it
// never scans for raw unicode emoji runes -- so this file supplies ONLY the
// missing trigger + lookup. It deliberately emits the SAME east.Emoji AST
// node goldmark-emoji's own emojiHTMLRenderer already renders (registered by
// emoji.New's Extend, see emoji.go); no second NodeRenderer is registered
// here, and no <img> string is ever built by hand in this file.
//
// definition.Github() (goldmark-emoji's own shortcode table) exposes no
// enumeration method -- only Get(shortName) -- so a full reverse-index walk
// of its ~1870 entries is not possible through the public API. Instead,
// unicodeEmojiShortnames below is a SEED list of canonical Github shortnames
// covering the common, mostly-single-rune emoji this baseline targets
// (error_recovery: ZWJ sequences / skin-tone modifiers are a documented long
// tail, not a blocker). Each seed name's *definition.Emoji is fetched via
// Get -- never hand-transcribed -- so its Name/Unicode/ShortNames fields are
// guaranteed byte-identical to what the shortcode parser itself resolves.
package press

import (
	"unicode/utf8"

	"github.com/yuin/goldmark"
	east "github.com/yuin/goldmark-emoji/ast"
	"github.com/yuin/goldmark-emoji/definition"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// unicodeEmojiShortnames is the seed list the reverse index (below) is built
// from -- canonical definition.Github() shortnames for common emoji. This is
// a baseline floor, not an exhaustive port of Github's ~1870-entry table;
// extending coverage is purely additive (append a name here).
var unicodeEmojiShortnames = []string{
	"grinning", "smiley", "smile", "grin", "laughing", "sweat_smile", "joy",
	"rofl", "relaxed", "blush", "innocent", "slight_smile", "upside_down",
	"wink", "relieved", "heart_eyes", "kissing_heart", "thinking",
	"star_struck", "sunglasses", "nerd", "money_mouth", "hugs",
	"thumbsup", "thumbsdown", "clap", "wave", "raised_hands", "pray",
	"muscle", "eyes", "tada", "confetti_ball", "fire", "100", "star",
	"sparkles", "heart", "broken_heart", "white_check_mark", "x",
	"rocket", "warning", "question", "exclamation", "boom", "zzz",
}

// buildUnicodeEmojiIndex reverse-indexes definition.Github()'s unicode-
// bearing entries named in unicodeEmojiShortnames into a rune-sequence (its
// exact UTF-8 byte string) -> *definition.Emoji map, plus the max rune count
// across every key (the longest-match search bound below). Entries that
// don't resolve, or whose IsUnicode() is false (custom, non-unicode-backed
// emoji), are skipped -- every seed name above is expected to resolve and
// carry unicode, but this stays defensive rather than panicking.
func buildUnicodeEmojiIndex() (map[string]*definition.Emoji, int) {
	gh := definition.Github()
	index := make(map[string]*definition.Emoji, len(unicodeEmojiShortnames))
	maxRunes := 0
	for _, name := range unicodeEmojiShortnames {
		e, ok := gh.Get(name)
		if !ok || !e.IsUnicode() {
			continue
		}
		index[string(e.Unicode)] = e
		if n := len(e.Unicode); n > maxRunes {
			maxRunes = n
		}
	}
	return index, maxRunes
}

// unicodeEmojiIndex/unicodeEmojiMaxRunes are built once at package init from
// buildUnicodeEmojiIndex -- the live table unicodeEmojiParser.Parse consults.
var unicodeEmojiIndex, unicodeEmojiMaxRunes = buildUnicodeEmojiIndex()

// longestUnicodeEmojiMatch finds the LONGEST rune-sequence prefix of line
// present in unicodeEmojiIndex. Longest-first matching matters even at this
// baseline's mostly-single-rune coverage: some Github unicode entries (e.g.
// "heart") are themselves 2 runes (base codepoint + variation selector), so
// a naive single-rune-only lookup would miss or mis-split them.
func longestUnicodeEmojiMatch(line []byte) (*definition.Emoji, int) {
	if unicodeEmojiMaxRunes == 0 {
		return nil, 0
	}
	offsets := make([]int, 0, unicodeEmojiMaxRunes)
	pos := 0
	for i := 0; i < unicodeEmojiMaxRunes && pos < len(line); i++ {
		r, size := utf8.DecodeRune(line[pos:])
		if r == utf8.RuneError && size <= 1 {
			break
		}
		pos += size
		offsets = append(offsets, pos)
	}
	for i := len(offsets) - 1; i >= 0; i-- {
		if entry, ok := unicodeEmojiIndex[string(line[:offsets[i]])]; ok {
			return entry, offsets[i]
		}
	}
	return nil, 0
}

// unicodeEmojiParser is the bespoke InlineParser: a lookup against
// unicodeEmojiIndex, emitting the exact east.Emoji node goldmark-emoji's own
// renderer already handles. It never builds HTML itself.
type unicodeEmojiParser struct{}

// Trigger returns the "halfspace" marker (' ') -- per parser.InlineParser's
// own contract, this fires the parser at every whitespace character AND at
// the head of every line (goldmark/parser.parseBlock's scan loop only tests
// registered triggers when the current byte is punctuation, whitespace, or
// position 0; raw UTF-8 lead bytes are neither punctuation nor whitespace,
// so they can NEVER be trigger bytes themselves -- the emoji rune has to be
// found by peeking FORWARD from the preceding boundary). This mirrors
// goldmark's own extension.Linkify parser, which triggers on the identical
// boundary set for the same reason (word-boundary-anchored inline matches).
func (unicodeEmojiParser) Trigger() []byte {
	return []byte{' '}
}

// Parse peeks past the triggering boundary (an actual space, or nothing if
// already at true line-head) and looks for the longest known emoji rune
// sequence starting there. A leading space, if present, is preserved as its
// own text node (mirroring goldmark's own linkifyParser.Parse) so it is
// never swallowed by the emoji match.
func (p unicodeEmojiParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if len(line) == 0 {
		return nil
	}

	consumes := 0
	if line[0] == ' ' {
		consumes = 1
		line = line[1:]
	}
	if len(line) == 0 {
		return nil
	}

	entry, byteLen := longestUnicodeEmojiMatch(line)
	if entry == nil {
		return nil
	}

	if consumes != 0 {
		ast.MergeOrAppendTextSegment(parent, segment.WithStop(segment.Start+1))
	}

	consumes += byteLen
	block.Advance(consumes)

	shortName := entry.ShortNames[0]
	return east.NewEmoji([]byte(shortName), entry)
}

// unicodeEmojiExtender registers unicodeEmojiParser as an additional
// InlineParser -- and ONLY that. goldmark-emoji's own Extend already
// registers the east.KindEmoji NodeRenderer (emoji.go's emojiOption bundles
// both goldmark.Extenders together); registering a second NodeRenderer here
// would be redundant and is deliberately never done.
type unicodeEmojiExtender struct{}

// Extend implements goldmark.Extender.
func (unicodeEmojiExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(unicodeEmojiParser{}, 100), // low number -> tried early
	))
}
