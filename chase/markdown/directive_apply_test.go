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

package markdown

import "testing"

// Test-list case 8: InlineStyle order -- setting the same property twice
// keeps the LAST value at the FIRST-seen position; distinct properties keep
// insertion order. Mirrors helpers/inline_style.js's decls object semantics
// (a bare Go map would flake this -- 01-RESEARCH.md "Don't Hand-Roll").
func TestInlineStyleOrder(t *testing.T) {
	s := NewInlineStyle()
	s.Set("color", "red")
	s.Set("background-color", "blue")
	s.Set("color", "green") // overwrite: keeps position 0, updates value

	got := s.String()
	want := "color:green;background-color:blue;"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestInlineStyleEmpty(t *testing.T) {
	s := NewInlineStyle()
	if !s.Empty() {
		t.Fatalf("Empty() = false, want true for a fresh InlineStyle")
	}
	if got := s.String(); got != "" {
		t.Fatalf("String() = %q, want empty string", got)
	}
	s.Set("color", "red")
	if s.Empty() {
		t.Fatalf("Empty() = true after Set, want false")
	}
}
