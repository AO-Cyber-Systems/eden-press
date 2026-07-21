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

package pptx

import "testing"

// TestInches locks the inch-to-EMU conversion: 1 inch == 914400 EMU
// (ECMA-376, decade-stable). Whole-inch inputs must be EXACT; fractional
// inputs use the documented round-to-nearest rule.
func TestInches(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want int64
	}{
		{"one inch is exactly 914400 EMU", 1, 914400},
		{"ten inches is exactly 9144000 EMU", 10, 9144000},
		{"seven point five inches is exactly 6858000 EMU (16:9/4:3 shared slide height)", 7.5, 6858000},
		{"thirteen and one third inches (16:9 slide width) rounds to 12192000 EMU", 40.0 / 3.0, 12192000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Inches(c.in); got != c.want {
				t.Errorf("Inches(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

// TestPoints locks the point-to-EMU conversion, including the 72pt == 1in
// cross-check (encoded as an assertion, not a computed dependency between
// constants).
func TestPoints(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want int64
	}{
		{"one point is exactly 12700 EMU", 1, 12700},
		{"seventy-two points is exactly 914400 EMU", 72, 914400},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Points(c.in); got != c.want {
				t.Errorf("Points(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}

	t.Run("72pt equals 1in cross-check", func(t *testing.T) {
		if got, want := Points(72), Inches(1); got != want {
			t.Errorf("Points(72) = %d, want Inches(1) = %d", got, want)
		}
	})
}

// TestCentimeters locks the centimeter-to-EMU conversion.
func TestCentimeters(t *testing.T) {
	if got := Centimeters(1); got != 360000 {
		t.Errorf("Centimeters(1) = %d, want 360000", got)
	}
}

// TestMillimeters locks the millimeter-to-EMU conversion.
func TestMillimeters(t *testing.T) {
	if got := Millimeters(1); got != 36000 {
		t.Errorf("Millimeters(1) = %d, want 36000", got)
	}
}

// TestCentipoints guards the classic EMU/centipoint mixup: DrawingML's
// a:rPr/@sz attribute is in CENTIPOINTS (hundredths of a point), never EMU.
// 44pt must render sz="4400", not an EMU value.
func TestCentipoints(t *testing.T) {
	if got := Centipoints(44); got != 4400 {
		t.Errorf("Centipoints(44) = %d, want 4400 (centipoints, NOT EMU)", got)
	}
}

// TestSlideSizeConstants locks the three ECMA-376 slide/notes sizes as the
// single authoritative source later TRDs size <p:sldSz>/<p:notesSz> from.
func TestSlideSizeConstants(t *testing.T) {
	cases := []struct {
		name string
		got  SlideSize
		want SlideSize
	}{
		{"16:9 widescreen matches ECMA-376 <p:sldSz type=\"screen16x9\">", SlideSize16x9, SlideSize{CX: 12192000, CY: 6858000, Type: "screen16x9"}},
		{"4:3 standard matches ECMA-376 <p:sldSz type=\"screen4x3\">", SlideSize4x3, SlideSize{CX: 9144000, CY: 6858000, Type: "screen4x3"}},
		{"notes size matches ECMA-376 <p:notesSz> (portrait, no type attr)", NotesSize, SlideSize{CX: 6858000, CY: 9144000}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Errorf("got %+v, want %+v", c.got, c.want)
			}
		})
	}
}

// TestSlideSizeCrossCheckAgainstInches cross-verifies each slide-size EMU
// constant against the independently-tested Inches conversion, so both
// slide sizes are provably driven from the same fixed EMU-per-inch constant.
func TestSlideSizeCrossCheckAgainstInches(t *testing.T) {
	if got, want := SlideSize16x9.CX, Inches(40.0/3.0); got != want {
		t.Errorf("SlideSize16x9.CX = %d, want Inches(40/3) = %d", got, want)
	}
	if got, want := SlideSize16x9.CY, Inches(7.5); got != want {
		t.Errorf("SlideSize16x9.CY = %d, want Inches(7.5) = %d", got, want)
	}
	if got, want := SlideSize4x3.CX, Inches(10); got != want {
		t.Errorf("SlideSize4x3.CX = %d, want Inches(10) = %d", got, want)
	}
	if got, want := SlideSize4x3.CY, Inches(7.5); got != want {
		t.Errorf("SlideSize4x3.CY = %d, want Inches(7.5) = %d", got, want)
	}
}
