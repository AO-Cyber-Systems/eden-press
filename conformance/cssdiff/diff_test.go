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

package cssdiff

import "testing"

// eq is a test helper: asserts Equal(expected, actual) matches want, logging the
// diff on mismatch so a regression is legible.
func eq(t *testing.T, name, expected, actual string, want bool) {
	t.Helper()
	got, diff := Equal(expected, actual)
	if got != want {
		t.Errorf("%s: Equal = %v, want %v\n--- expected ---\n%s\n--- actual ---\n%s\n--- diff ---\n%s",
			name, got, want, expected, actual, diff)
	}
}

// --- POSITIVE: format-insensitive equality (parsing erases cosmetic noise) ---

func TestEqualIdentical(t *testing.T) {
	css := "section { color: #333; font-size: 28px; }\nh1 { margin: 0; }"
	eq(t, "identical", css, css, true)
}

func TestEqualFormatInsensitive(t *testing.T) {
	// Different whitespace, indentation, and a comment — same normalized model.
	expected := ".a{color:red;margin:0}"
	actual := ".a {\n  /* a comment */\n  color: red;\n  margin: 0;\n}\n"
	eq(t, "format-insensitive", expected, actual, true)
}

// --- NEGATIVE: the broken-theme gate (CONF-03) — these MUST be reported ---

func TestEqualChangedValue(t *testing.T) {
	eq(t, "changed-value", ".a{color:red}", ".a{color:blue}", false)
}

func TestEqualDroppedImportant(t *testing.T) {
	// Dropping !important changes the cascade — must be caught.
	eq(t, "dropped-important", ".a{color:red !important}", ".a{color:red}", false)
}

func TestEqualChangedSelector(t *testing.T) {
	eq(t, "changed-selector", ".a{color:red}", ".b{color:red}", false)
}

func TestEqualAddedDeclaration(t *testing.T) {
	eq(t, "added-decl", ".a{color:red}", ".a{color:red;margin:0}", false)
}

func TestEqualAddedRule(t *testing.T) {
	eq(t, "added-rule", "a{x:1}", "a{x:1}\nb{y:2}", false)
}

// --- NEGATIVE: cascade-significant reordering — order is meaningful ---

func TestEqualRuleReorder(t *testing.T) {
	// Two rules of equal specificity: swapping their order changes the cascade.
	expected := ".x{color:red}\n.y{color:blue}"
	actual := ".y{color:blue}\n.x{color:red}"
	eq(t, "rule-reorder", expected, actual, false)
}

func TestEqualDeclReorderSameProperty(t *testing.T) {
	// The same property declared twice: last wins, so reordering flips the result.
	expected := ".a{color:red;color:blue}"
	actual := ".a{color:blue;color:red}"
	eq(t, "decl-reorder-same-prop", expected, actual, false)
}
