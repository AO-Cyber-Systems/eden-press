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
