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

import "math"

// EMU (English Metric Unit) is the fixed integer coordinate unit every
// DrawingML position/size attribute (a:off, a:ext, ...) is expressed in.
// These conversion constants are fixed by the ECMA-376/ISO-IEC 29500 OOXML
// specification and are decade-stable -- never recompute them from anything
// else, and never derive one from another (e.g. do not compute
// emuPerPoint from emuPerInch/72; encode each fixed constant literally so a
// typo in one can never silently corrupt another).
const (
	emuPerInch       int64 = 914400
	emuPerPoint      int64 = 12700
	emuPerCentimeter int64 = 360000
	emuPerMillimeter int64 = 36000
)

// round implements the single deterministic rounding rule used by every EMU
// conversion helper in this file: round-to-nearest (ties away from zero, via
// math.Round). Whole-unit inputs always land on an exact EMU value under this
// rule (e.g. Inches(1) == 914400 exactly); only genuinely fractional inputs
// depend on it.
func round(v float64) int64 {
	return int64(math.Round(v))
}

// Inches converts a measurement in inches to EMU. Whole-inch inputs are
// exact (Inches(1) == 914400); fractional inputs use the documented
// round-to-nearest rule.
func Inches(v float64) int64 {
	return round(v * float64(emuPerInch))
}

// Points converts a measurement in points to EMU. 72 points == 1 inch ==
// 914400 EMU is a useful cross-check (Points(72) == Inches(1)), asserted as
// its own test rather than derived as a computed dependency.
func Points(v float64) int64 {
	return round(v * float64(emuPerPoint))
}

// Centimeters converts a measurement in centimeters to EMU.
func Centimeters(v float64) int64 {
	return round(v * float64(emuPerCentimeter))
}

// Millimeters converts a measurement in millimeters to EMU.
func Millimeters(v float64) int64 {
	return round(v * float64(emuPerMillimeter))
}

// Centipoints converts a font size in points to CENTIPOINTS (hundredths of
// a point) -- the unit DrawingML's a:rPr/@sz attribute is expressed in.
//
// THIS IS NOT EMU. Font size is centipoints; positions and extents are EMU.
// Mixing the two units is the classic PPTX-writer bug this helper exists to
// prevent: Centipoints(44) == 4400 (sz="4400" for a 44pt run), never an EMU
// value.
func Centipoints(pt float64) int {
	return int(math.Round(pt * 100))
}

// SlideSize is a named EMU width/height pair matching the cx/cy/type
// attributes of a <p:sldSz> or <p:notesSz> element. Type is empty for sizes
// (like NotesSize) that have no type attribute in the schema.
type SlideSize struct {
	CX, CY int64
	Type   string
}

// Authoritative slide/notes-size constants, matching ECMA-376's <p:sldSz>
// and <p:notesSz> elements exactly. Every later TRD that emits a slide or
// notes size must source it from these values -- never recompute or
// hardcode the cx/cy pair again.
var (
	// SlideSize16x9 is the 16:9 widescreen slide size (13.333in x 7.5in).
	SlideSize16x9 = SlideSize{CX: 12192000, CY: 6858000, Type: "screen16x9"}

	// SlideSize4x3 is the 4:3 standard slide size (10in x 7.5in).
	SlideSize4x3 = SlideSize{CX: 9144000, CY: 6858000, Type: "screen4x3"}

	// NotesSize is the portrait speaker-notes page size (<p:notesSz>); it
	// carries no type attribute in the schema, so Type is left "".
	NotesSize = SlideSize{CX: 6858000, CY: 9144000}
)
