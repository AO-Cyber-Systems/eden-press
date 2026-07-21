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

import "strings"

// InlineStyle is an ordered, dedupe-by-property inline-style builder --
// mirroring helpers/inline_style.js's `decls` object. It is deliberately
// backed by an ordered slice + index map, NEVER a bare Go map used for
// iteration: declaration order in a `style="..."` attribute is
// cascade-significant and directly compared byte-for-byte by htmldiff
// (01-RESEARCH.md "Don't Hand-Roll" -- a Go map's random iteration order
// would flake this).
//
// Set(prop, value) dedupes by property name: the FIRST time a property is
// set determines its position in the final declaration list; every
// subsequent Set for the same property only updates the value in place
// (last-write-wins), exactly like reassigning an existing key on a JS
// object leaves its enumeration position untouched.
type InlineStyle struct {
	keys   []string
	values map[string]string
}

// NewInlineStyle returns a new, empty *InlineStyle.
func NewInlineStyle() *InlineStyle {
	return &InlineStyle{values: map[string]string{}}
}

// Set assigns value to prop, preserving first-seen declaration order and
// overwriting any prior value for the same prop in place. Returns the
// receiver for chaining, mirroring inline_style.js's `set` (used for the
// backgroundImage special override's `.set(...).set(...)` chain).
func (s *InlineStyle) Set(prop, value string) *InlineStyle {
	if _, exists := s.values[prop]; !exists {
		s.keys = append(s.keys, prop)
	}
	s.values[prop] = value
	return s
}

// Empty reports whether no declarations have been set.
func (s *InlineStyle) Empty() bool {
	return len(s.keys) == 0
}

// String serializes the declarations in first-seen order as
// "prop:value;prop2:value2;", matching inline_style.js's toString() shape
// (minus its PostCSS-based sanitization pass, which is not needed here: all
// values chase/markdown feeds through this builder already come from
// chase/directive's coercion tables, never raw untrusted CSS).
func (s *InlineStyle) String() string {
	var b strings.Builder
	for _, k := range s.keys {
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(s.values[k])
		b.WriteByte(';')
	}
	return b.String()
}
