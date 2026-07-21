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

package selector

import (
	"testing"

	"github.com/tdewolff/parse/v2/css"
)

// ---------------------------------------------------------------------
// Task 1 — Test-list cases 3, 4, 8, 9 (comma split + compound
// segmentation over []css.Token). Written FIRST, before selector.go
// exists, per TDD: this file intentionally fails to compile/pass until
// SplitList/ParseSelectorTokens/String are implemented.
// ---------------------------------------------------------------------

// Test-list case 3: a comma-separated selector list splits into one
// compound per entry, at the top level only.
func TestSplitList_MultiSelector(t *testing.T) {
	tokens, err := ParseSelectorTokens("h1, h2")
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	got := SplitList(tokens)
	if len(got) != 2 {
		t.Fatalf("SplitList(%q) = %d compounds, want 2", String(tokens), len(got))
	}
	if s := String(got[0]); s != "h1" {
		t.Errorf("compound[0] = %q, want %q", s, "h1")
	}
	if s := String(got[1]); s != "h2" {
		t.Errorf("compound[1] = %q, want %q", s, "h2")
	}
}

// Test-list case 4: an outer combinator survives segmentation and a
// FunctionToken's arguments are kept OPAQUE (not decomposed) during
// splitting — `:is(...)` must remain a single token, not be split on its
// own internal comma.
func TestSplitList_FunctionArgsOpaqueWithCombinator(t *testing.T) {
	const sel = ":is(h3, h4) + p"
	tokens, err := ParseSelectorTokens(sel)
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	got := SplitList(tokens)
	if len(got) != 1 {
		t.Fatalf("SplitList(%q) = %d compounds, want 1 (no top-level comma)", sel, len(got))
	}
	if s := String(got[0]); s != sel {
		t.Errorf("compound[0] = %q, want %q", s, sel)
	}
	found := false
	for _, tok := range got[0] {
		if tok.TokenType == css.FunctionToken {
			found = true
			if string(tok.Data) != "is(" {
				t.Errorf("FunctionToken.Data = %q, want %q", tok.Data, "is(")
			}
		}
	}
	if !found {
		t.Fatalf("no FunctionToken found in segmented compound %q", String(got[0]))
	}
}

// Test-list case 8: top-level comma split ignores commas nested inside
// [attr="a,b"] string values and :is(a, b) function arguments.
func TestSplitList_IgnoresNestedCommas(t *testing.T) {
	tests := []struct {
		name string
		sel  string
		want []string
	}{
		{
			name: "attribute value comma",
			sel:  `[data-x="a,b"]`,
			want: []string{`[data-x="a,b"]`},
		},
		{
			name: "function arg comma only",
			sel:  ":is(h1, h2)",
			want: []string{":is(h1, h2)"},
		},
		{
			name: "function arg comma plus real top-level comma",
			sel:  ":is(h1, h2), p",
			want: []string{":is(h1, h2)", "p"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := ParseSelectorTokens(tt.sel)
			if err != nil {
				t.Fatalf("ParseSelectorTokens(%q): %v", tt.sel, err)
			}
			got := SplitList(tokens)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitList(%q) = %d compounds, want %d", tt.sel, len(got), len(tt.want))
			}
			for i, w := range tt.want {
				if s := String(got[i]); s != w {
					t.Errorf("compound[%d] = %q, want %q", i, s, w)
				}
			}
		})
	}
}

// Test-list case 9 (SplitList half — the Prepend/idempotency half is
// covered in Task 2): an empty or whitespace-only selector is a safe
// no-op, never a panic, never a phantom compound.
func TestSplitList_EmptyIsNoOp(t *testing.T) {
	if got := SplitList(nil); len(got) != 0 {
		t.Errorf("SplitList(nil) = %d compounds, want 0", len(got))
	}
	if got := SplitList([]css.Token{}); len(got) != 0 {
		t.Errorf("SplitList(empty) = %d compounds, want 0", len(got))
	}
	tokens, err := ParseSelectorTokens("   ")
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	if got := SplitList(tokens); len(got) != 0 {
		t.Errorf("SplitList(whitespace-only) = %d compounds, want 0", len(got))
	}
}

// Test-list case 9 (continued): an ALREADY fully-scoped selector chain
// round-trips through SplitList/String unchanged — segmentation and
// re-serialization are safe/idempotent on already-scoped text, not just
// on empty input. (The other reading of "already-scoped is a no-op" —
// that scope.go's Prepend must not double-prepend an intermediate
// placeholder — is covered in Task 2's TestPrepend_AlreadyPlaceholdered,
// since Prepend does not exist yet at this point in the file's history.)
func TestSplitList_AlreadyScopedIsNoOp(t *testing.T) {
	const already = "div.marpit > svg > foreignObject > section"
	tokens, err := ParseSelectorTokens(already)
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	got := SplitList(tokens)
	if len(got) != 1 {
		t.Fatalf("SplitList(%q) = %d compounds, want 1", already, len(got))
	}
	if s := String(got[0]); s != already {
		t.Errorf("round-trip = %q, want unchanged %q", s, already)
	}
}

// ---------------------------------------------------------------------
// Task 2 — Test-list cases 1, 2, 5, 6, 7 (two-step placeholder scoping
// in scope.go + the :root sentinel/specificity rewrite in root.go).
// Written FIRST, before scope.go/root.go exist, per TDD.
// ---------------------------------------------------------------------

// scoped runs the full Prepend -> Replace pipeline for a single bare
// selector against one container/slide chain, returning the final
// scoped text. Shared by cases 1 and 2.
func scoped(t *testing.T, sel string, container, slide []css.Token) string {
	t.Helper()
	tokens, err := ParseSelectorTokens(sel)
	if err != nil {
		t.Fatalf("ParseSelectorTokens(%q): %v", sel, err)
	}
	compounds := SplitList(tokens)
	if len(compounds) != 1 {
		t.Fatalf("SplitList(%q) = %d compounds, want 1", sel, len(compounds))
	}
	prepended := Prepend(compounds[0])
	replaced := Replace(prepended, container, slide)
	return String(replaced)
}

// Test-list case 1: "section" scopes to the full inline-SVG combinator
// chain.
func TestScopePrependReplace_InlineSVGChain(t *testing.T) {
	const want = "div.marpit > svg > foreignObject > section"
	got := scoped(t, "section", InlineSVGContainerChain(), SlideChain())
	if got != want {
		t.Errorf("scoped(%q) = %q, want %q", "section", got, want)
	}
}

// Test-list case 2: "section" scopes to the non-SVG combinator chain.
func TestScopePrependReplace_NonSVGChain(t *testing.T) {
	const want = "div.marpit > section"
	got := scoped(t, "section", NonSVGContainerChain(), SlideChain())
	if got != want {
		t.Errorf("scoped(%q) = %q, want %q", "section", got, want)
	}
}

// Supplementary (beyond the numbered test list): a descendant selector
// like "h1" must NOT fuse onto the container/slide chain the way bare
// "section" does — it is scoped as a DESCENDANT of the slide element,
// matching Marpit's own prepend.js (which only fuses when the compound
// literally starts with "section").
func TestScopePrependReplace_DescendantSelectorIsSpaced(t *testing.T) {
	const want = "div.marpit > svg > foreignObject > section h1"
	got := scoped(t, "h1", InlineSVGContainerChain(), SlideChain())
	if got != want {
		t.Errorf("scoped(%q) = %q, want %q", "h1", got, want)
	}
}

// Supplementary: Prepend on an already-placeholdered compound is a safe
// no-op (idempotent) — the Prepend-level reading of test-list case 9's
// "already-scoped selector is a no-op".
func TestPrepend_AlreadyPlaceholderedIsIdempotent(t *testing.T) {
	tokens, err := ParseSelectorTokens("section")
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	once := Prepend(SplitList(tokens)[0])
	twice := Prepend(once)
	if String(twice) != String(once) {
		t.Errorf("Prepend(Prepend(x)) = %q, want unchanged %q", String(twice), String(once))
	}
}

// Test-list case 5: a bare ":root" at add-time becomes the fused
// ":marpit-root" sentinel (as literal "section:marpit-root" text, which
// is what makes it fuse onto the container/slide chain the same way a
// literal "section" compound does).
func TestMarkRoot_BareRootBecomesSentinel(t *testing.T) {
	const want = "section:marpit-root"
	tokens, err := ParseSelectorTokens(":root")
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	got := String(MarkRoot(tokens))
	if got != want {
		t.Errorf("MarkRoot(%q) = %q, want %q", ":root", got, want)
	}
}

// Test-list case 6: AFTER scope-prefixing, the ":marpit-root" sentinel
// rewrites to the literal specificity-trick token sequence
// `:where(section):not([\20 root])` — for both the SVG and non-SVG
// container chains. IncreasingSpecificity MUST run after Replace
// (RESEARCH Pitfall 1); this test exercises the full ordered pipeline.
func TestIncreasingSpecificity_AfterScopePrefix(t *testing.T) {
	tests := []struct {
		name      string
		container []css.Token
		want      string
	}{
		{
			name:      "inline-SVG",
			container: InlineSVGContainerChain(),
			want:      `div.marpit > svg > foreignObject > :where(section):not([\20 root])`,
		},
		{
			name:      "non-SVG",
			container: NonSVGContainerChain(),
			want:      `div.marpit > :where(section):not([\20 root])`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := ParseSelectorTokens(":root")
			if err != nil {
				t.Fatalf("ParseSelectorTokens: %v", err)
			}
			marked := MarkRoot(tokens)
			prepended := Prepend(marked)
			replaced := Replace(prepended, tt.container, SlideChain())
			got := String(IncreasingSpecificity(replaced))
			if got != tt.want {
				t.Errorf("full :root pipeline (%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// Supplementary: demonstrates RESEARCH Pitfall 1 is real — running
// IncreasingSpecificity BEFORE Replace (wrong order) does NOT produce
// the corpus-correct fully-qualified specificity trick, because the
// marker is not yet adjacent to the real slide element.
func TestIncreasingSpecificity_WrongOrderDiffersFromCorrect(t *testing.T) {
	tokens, err := ParseSelectorTokens(":root")
	if err != nil {
		t.Fatalf("ParseSelectorTokens: %v", err)
	}
	marked := MarkRoot(tokens)
	prepended := Prepend(marked)

	wrongOrder := String(Replace(IncreasingSpecificity(prepended), InlineSVGContainerChain(), SlideChain()))
	correctOrder := String(IncreasingSpecificity(Replace(prepended, InlineSVGContainerChain(), SlideChain())))

	if wrongOrder == correctOrder {
		t.Fatalf("expected wrong-order pipeline to differ from correct-order pipeline, both produced %q", wrongOrder)
	}
	const want = `div.marpit > svg > foreignObject > :where(section):not([\20 root])`
	if correctOrder != want {
		t.Errorf("correct-order pipeline = %q, want %q", correctOrder, want)
	}
}

// Test-list case 7 (gaia regression): a ":root" marker nested inside
// `:where(:is(...))` is found and rewritten even though scope.go's
// Prepend never descends into FunctionToken arguments — root.go's
// MarkRoot/IncreasingSpecificity use Walk, which does.
func TestIncreasingSpecificity_NestedMarker(t *testing.T) {
	const sel = ":where(:is(:root, h1))"
	const want = `div.marpit > section :where(:is(:where(section):not([\20 root]), h1))`

	tokens, err := ParseSelectorTokens(sel)
	if err != nil {
		t.Fatalf("ParseSelectorTokens(%q): %v", sel, err)
	}
	compounds := SplitList(tokens)
	if len(compounds) != 1 {
		t.Fatalf("SplitList(%q) = %d compounds, want 1", sel, len(compounds))
	}
	marked := MarkRoot(compounds[0])
	prepended := Prepend(marked)
	replaced := Replace(prepended, NonSVGContainerChain(), SlideChain())

	var foundDepth = -1
	Walk(replaced, func(tok css.Token, index int, depth int) bool {
		if tok.TokenType == css.Colon && index+1 < len(replaced) &&
			replaced[index+1].TokenType == css.IdentToken &&
			string(replaced[index+1].Data) == "marpit-root" {
			foundDepth = depth
		}
		return true
	})
	if foundDepth < 2 {
		t.Fatalf("expected the :marpit-root marker to be found nested at depth >= 2 (inside :where(:is(...))), got depth %d", foundDepth)
	}

	got := String(IncreasingSpecificity(replaced))
	if got != want {
		t.Errorf("nested-marker pipeline = %q, want %q", got, want)
	}
}
