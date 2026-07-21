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

import "testing"

// --- Task 1: comment detection + front-matter + YAML-ish value parser ---
// (PARSE-03). Test-list cases 1-4.

// Test-list case 1: front-matter `theme: gaia` -> global directive
// candidate {theme: "gaia"} (deck-level). Mirrors the front-matter block of
// conformance/corpus/cases/marp-paginate/input.md's shape.
func TestFrontMatterThemeGlobal(t *testing.T) {
	src := "---\ntheme: gaia\n---\n\n# Slide\n"
	body, rest, ok := DetectFrontMatter(src)
	if !ok {
		t.Fatalf("expected front matter to be detected in %q", src)
	}
	if rest == src {
		t.Fatalf("expected rest to exclude the front-matter fence, got %q", rest)
	}
	kvs := ParseFrontMatter(body)
	if len(kvs) != 1 {
		t.Fatalf("expected exactly 1 kv pair, got %#v", kvs)
	}
	if kvs[0].Key != "theme" || kvs[0].Val != "gaia" {
		t.Fatalf("expected {theme: gaia}, got %#v", kvs[0])
	}
}

// Test-list case 2: comment `<!-- _class: lead -->` -> raw kv
// {_class: "lead"} detected (not yet recognized -- recognition is Task 2).
func TestCommentSpotClassDetected(t *testing.T) {
	body, ok := DetectComment("<!-- _class: lead -->")
	if !ok {
		t.Fatalf("expected comment to be detected")
	}
	kvs := ParseComment(body)
	if len(kvs) != 1 || kvs[0].Key != "_class" || kvs[0].Val != "lead" {
		t.Fatalf("expected {_class: lead}, got %#v", kvs)
	}
}

// Test-list case 3: multi-line comment span detected; a line not starting
// with "<" fast-fails detection.
func TestCommentMultiLineDetectedAndNonCommentFastFails(t *testing.T) {
	body, ok := DetectComment("<!--\nclass: a\n-->")
	if !ok {
		t.Fatalf("expected multi-line comment span to be detected")
	}
	if body != "class: a" {
		t.Fatalf("expected trimmed body %q, got %q", "class: a", body)
	}

	if _, ok := DetectComment("not a comment at all"); ok {
		t.Fatalf("expected fast-fail for a line not starting with '<'")
	}
}

// Test-list case 4: a non-directive comment (`<!-- just a note -->`) is
// DETECTED but produces no recognized directive (i.e. it parses to zero
// raw kv pairs, since "just a note" has no "key: value" shape).
func TestCommentNonDirectiveDetectedButNoKV(t *testing.T) {
	body, ok := DetectComment("<!-- just a note -->")
	if !ok {
		t.Fatalf("expected non-directive comment to still be DETECTED")
	}
	kvs := ParseComment(body)
	if len(kvs) != 0 {
		t.Fatalf("expected no recognized kv pairs for a non-directive comment, got %#v", kvs)
	}
}

// Extra yaml.go-level coverage: bare strings, quoted strings, and flow
// lists, exercised directly (not only through comment/front-matter).
func TestYamlParseScalarsAndFlowList(t *testing.T) {
	kvs := ParseYAMLish("class: [a, b]\nheader: 'Eden Press'\nfooter: \"CONFIDENTIAL\"\n")
	if len(kvs) != 3 {
		t.Fatalf("expected 3 kv pairs, got %#v", kvs)
	}
	classVal, ok := kvs[0].Val.([]string)
	if kvs[0].Key != "class" || !ok || len(classVal) != 2 || classVal[0] != "a" || classVal[1] != "b" {
		t.Fatalf("expected class: [a b], got %#v", kvs[0])
	}
	if kvs[1].Key != "header" || kvs[1].Val != "Eden Press" {
		t.Fatalf("expected header: Eden Press (quotes stripped), got %#v", kvs[1])
	}
	if kvs[2].Key != "footer" || kvs[2].Val != "CONFIDENTIAL" {
		t.Fatalf("expected footer: CONFIDENTIAL (quotes stripped), got %#v", kvs[2])
	}
}

// --- Task 2: directive tables + value coercion + spot rule ---
// (PARSE-02 tables). Test-list cases 5, 8.

// Test-list case 5: global coercion -- headingDivider: 2 -> [1,2];
// headingDivider: false -> false; unknown theme: nope (predicate false) ->
// dropped.
func TestCoerceGlobalHeadingDividerAndTheme(t *testing.T) {
	v, known := CoerceGlobal("headingDivider", "2", nil)
	if !known {
		t.Fatalf("expected headingDivider to be a known global directive")
	}
	got, ok := v.([]int)
	if !ok || len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("expected headingDivider: 2 -> [1 2], got %#v", v)
	}

	v, known = CoerceGlobal("headingDivider", "false", nil)
	if !known {
		t.Fatalf("expected headingDivider to be a known global directive")
	}
	if b, ok := v.(bool); !ok || b != false {
		t.Fatalf("expected headingDivider: false -> false, got %#v", v)
	}

	themeExists := func(name string) bool { return name == "gaia" }
	v, known = CoerceGlobal("theme", "nope", themeExists)
	if !known {
		t.Fatalf("expected theme to be a known global directive name")
	}
	if v != nil {
		t.Fatalf("expected unknown theme name to be silently dropped, got %#v", v)
	}

	v, known = CoerceGlobal("theme", "gaia", themeExists)
	if !known || v != "gaia" {
		t.Fatalf("expected theme: gaia to resolve to \"gaia\", got %#v known=%v", v, known)
	}

	if _, known := CoerceGlobal("notADirective", "x", nil); known {
		t.Fatalf("expected an unrecognized global key to report isKnown=false")
	}
}

// Test-list case 8: local coercion -- paginate: hold -> "hold";
// paginate: true -> true; class: [a, b] -> "a b"; non-string footer
// rejected.
func TestCoerceLocalPaginateClassFooter(t *testing.T) {
	v, known := CoerceLocal("paginate", "hold")
	if !known || v != "hold" {
		t.Fatalf("expected paginate: hold -> \"hold\", got %#v known=%v", v, known)
	}

	v, known = CoerceLocal("paginate", "true")
	if b, ok := v.(bool); !known || !ok || !b {
		t.Fatalf("expected paginate: true -> true, got %#v known=%v", v, known)
	}

	v, known = CoerceLocal("class", []string{"a", "b"})
	if !known || v != "a b" {
		t.Fatalf("expected class: [a, b] -> \"a b\", got %#v known=%v", v, known)
	}

	v, known = CoerceLocal("footer", []string{"a", "b"})
	if !known {
		t.Fatalf("expected footer to be a known local directive name")
	}
	if v != nil {
		t.Fatalf("expected non-string footer to be rejected, got %#v", v)
	}

	v, known = CoerceLocal("footer", "Eden Press")
	if !known || v != "Eden Press" {
		t.Fatalf("expected string footer to pass through, got %#v known=%v", v, known)
	}
}

// Spot-rule coverage: an underscore-prefixed key maps to its base local
// directive name.
func TestSpotKeyMapsToBaseLocalDirective(t *testing.T) {
	base, ok := SpotKey("_class")
	if !ok || base != "class" {
		t.Fatalf("expected _class -> class, got base=%q ok=%v", base, ok)
	}
	if _, ok := SpotKey("class"); ok {
		t.Fatalf("expected a non-underscore-prefixed key to not be a spot key")
	}
	if _, ok := SpotKey("_"); ok {
		t.Fatalf("expected a bare underscore (empty base) to not be a spot key")
	}
}

// --- Task 3: carry-forward cursor state machine ---
// (PARSE-02/PARSE-07 semantics). Test-list cases 6, 7, 9.

// Test-list case 6: local carry-forward -- class: a on slide 1 persists to
// slide 2; overridden by class: b on slide 3.
func TestCarryForwardLocalPersistsAcrossSlides(t *testing.T) {
	events := []Event{
		{Kind: SlideOpen},
		{Kind: DirectiveCommentEvent, Key: "class", Raw: "a"},
		{Kind: SlideClose},
		{Kind: SlideOpen},
		{Kind: SlideClose},
		{Kind: SlideOpen},
		{Kind: DirectiveCommentEvent, Key: "class", Raw: "b"},
		{Kind: SlideClose},
	}
	slides := Resolve(events, nil)
	if len(slides) != 3 {
		t.Fatalf("expected 3 slides, got %d (%#v)", len(slides), slides)
	}
	if slides[0]["class"] != "a" {
		t.Fatalf("slide 1 class = %#v, want \"a\"", slides[0]["class"])
	}
	if slides[1]["class"] != "a" {
		t.Fatalf("slide 2 class should carry forward, got %#v, want \"a\"", slides[1]["class"])
	}
	if slides[2]["class"] != "b" {
		t.Fatalf("slide 3 class should be overridden, got %#v, want \"b\"", slides[2]["class"])
	}
}

// Test-list case 7: spot -- _class: lead applies only to the current slide;
// slide+1 has no class from it (spot reset). Mirrors
// conformance/corpus/cases/marp-class-spot/input.md.
func TestCarryForwardSpotResetsEverySlide(t *testing.T) {
	events := []Event{
		{Kind: SlideOpen},
		{Kind: DirectiveCommentEvent, Key: "_class", Raw: "lead"},
		{Kind: SlideClose},
		{Kind: SlideOpen},
		{Kind: SlideClose},
	}
	slides := Resolve(events, nil)
	if len(slides) != 2 {
		t.Fatalf("expected 2 slides, got %d (%#v)", len(slides), slides)
	}
	if slides[0]["class"] != "lead" {
		t.Fatalf("slide 1 class = %#v, want \"lead\"", slides[0]["class"])
	}
	if _, ok := slides[1]["class"]; ok {
		t.Fatalf("slide 2 should have no class (spot reset), got %#v", slides[1]["class"])
	}
}

// Test-list case 9: global stamped on every slide identically after the
// local/spot loop.
func TestCarryForwardGlobalsStampedOnEverySlide(t *testing.T) {
	themeExists := func(name string) bool { return name == "gaia" }
	events := []Event{
		{Kind: DirectiveCommentEvent, Key: "theme", Raw: "gaia"}, // front-matter-like, pre-slide
		{Kind: SlideOpen},
		{Kind: SlideClose},
		{Kind: SlideOpen},
		{Kind: DirectiveCommentEvent, Key: "class", Raw: "b"},
		{Kind: SlideClose},
	}
	slides := Resolve(events, themeExists)
	if len(slides) != 2 {
		t.Fatalf("expected 2 slides, got %d (%#v)", len(slides), slides)
	}
	for i, s := range slides {
		if s["theme"] != "gaia" {
			t.Fatalf("slide %d theme = %#v, want \"gaia\" (global must stamp every slide)", i+1, s["theme"])
		}
	}
}

// Extra corpus-mirroring coverage: a front-matter `paginate: true` (a LOCAL
// directive) applies to every slide, since it is seeded into cursor.local
// before the first slide open. Mirrors
// conformance/corpus/cases/marp-paginate/input.md.
func TestCarryForwardFrontMatterPaginateAppliesToAllSlides(t *testing.T) {
	events := []Event{
		{Kind: DirectiveCommentEvent, Key: "paginate", Raw: "true"},
		{Kind: SlideOpen},
		{Kind: SlideClose},
		{Kind: SlideOpen},
		{Kind: SlideClose},
	}
	slides := Resolve(events, nil)
	for i, s := range slides {
		if b, ok := s["paginate"].(bool); !ok || !b {
			t.Fatalf("slide %d paginate = %#v, want true", i+1, s["paginate"])
		}
	}
}

// Extra corpus-mirroring coverage: front-matter header/footer (LOCAL
// directives) apply to the single slide. Mirrors
// conformance/corpus/cases/marp-header-footer/input.md.
func TestCarryForwardFrontMatterHeaderFooterAppliesToSlide(t *testing.T) {
	events := []Event{
		{Kind: DirectiveCommentEvent, Key: "header", Raw: "Eden Press"},
		{Kind: DirectiveCommentEvent, Key: "footer", Raw: "CONFIDENTIAL"},
		{Kind: SlideOpen},
		{Kind: SlideClose},
	}
	slides := Resolve(events, nil)
	if len(slides) != 1 {
		t.Fatalf("expected 1 slide, got %d", len(slides))
	}
	if slides[0]["header"] != "Eden Press" || slides[0]["footer"] != "CONFIDENTIAL" {
		t.Fatalf("got %#v", slides[0])
	}
}

// EventsFromKV coverage: a parsed comment's ordered kv pairs convert to an
// ordered DirectiveCommentEvent slice.
func TestEventsFromKVPreservesOrder(t *testing.T) {
	kvs := []KV{{Key: "class", Val: "a"}, {Key: "_class", Val: "lead"}}
	events := EventsFromKV(kvs)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Kind != DirectiveCommentEvent || events[0].Key != "class" || events[0].Raw != "a" {
		t.Fatalf("event 0 = %#v", events[0])
	}
	if events[1].Kind != DirectiveCommentEvent || events[1].Key != "_class" || events[1].Raw != "lead" {
		t.Fatalf("event 1 = %#v", events[1])
	}
}
