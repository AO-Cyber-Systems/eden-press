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

// Test-list case 1: bg -> background flag; bg fit -> size=contain (fit
// alias); bg left:40% -> split left @40%; bg vertical -> direction
// vertical; bg 50% -> size 50%.
func TestBgOptionParseKeywords(t *testing.T) {
	t.Run("bg alone sets Background", func(t *testing.T) {
		opts := ParseBgOptions("bg")
		if !opts.Background {
			t.Fatalf("Background = false, want true")
		}
	})

	t.Run("fit aliases contain", func(t *testing.T) {
		opts := ParseBgOptions("bg fit")
		if !opts.Background {
			t.Fatalf("Background = false, want true")
		}
		if opts.SizeKeyword != "contain" {
			t.Fatalf("SizeKeyword = %q, want %q (fit is an alias for contain)", opts.SizeKeyword, "contain")
		}
	})

	t.Run("split left with percentage", func(t *testing.T) {
		opts := ParseBgOptions("bg left:40%")
		if opts.SplitSide != "left" {
			t.Fatalf("SplitSide = %q, want %q", opts.SplitSide, "left")
		}
		if opts.SplitSize != "40%" {
			t.Fatalf("SplitSize = %q, want %q", opts.SplitSize, "40%")
		}
	})

	t.Run("vertical direction", func(t *testing.T) {
		opts := ParseBgOptions("bg vertical")
		if opts.Direction != "vertical" {
			t.Fatalf("Direction = %q, want %q", opts.Direction, "vertical")
		}
	})

	t.Run("bare percentage resize", func(t *testing.T) {
		opts := ParseBgOptions("bg 50%")
		if opts.ResizeSize != "50%" {
			t.Fatalf("ResizeSize = %q, want %q", opts.ResizeSize, "50%")
		}
	})
}

// Test-list case 2: bg w:300px h:50% -> width/height with units; bare
// number -> px.
func TestBgOptionParseDimensions(t *testing.T) {
	opts := ParseBgOptions("bg w:300px h:50%")
	if opts.Width != "300px" {
		t.Fatalf("Width = %q, want %q", opts.Width, "300px")
	}
	if opts.Height != "50%" {
		t.Fatalf("Height = %q, want %q", opts.Height, "50%")
	}

	bare := ParseBgOptions("bg w:300 h:200")
	if bare.Width != "300px" {
		t.Fatalf("bare Width = %q, want %q (bare number -> px)", bare.Width, "300px")
	}
	if bare.Height != "200px" {
		t.Fatalf("bare Height = %q, want %q (bare number -> px)", bare.Height, "200px")
	}
}

// Test-list case 3: bg blur -> filter blur(10px) default; bg drop-shadow ->
// default '0 5px 10px rgba(0,0,0,.4)'.
func TestBgOptionParseFilters(t *testing.T) {
	blur := ParseBgOptions("bg blur")
	if got := blur.FilterCSS(); got != "blur(10px)" {
		t.Fatalf("FilterCSS() = %q, want %q", got, "blur(10px)")
	}

	shadow := ParseBgOptions("bg drop-shadow")
	if got := shadow.FilterCSS(); got != "drop-shadow(0 5px 10px rgba(0,0,0,.4))" {
		t.Fatalf("FilterCSS() = %q, want %q", got, "drop-shadow(0 5px 10px rgba(0,0,0,.4))")
	}
}
