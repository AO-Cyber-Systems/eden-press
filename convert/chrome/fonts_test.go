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

package chrome

// fonts_test.go carries 08-05's MATH-table survival proof:
// TestWoff2MathTableSurvivesConversion is a dependency-free (Go stdlib only)
// proof that the bundled STIX Two Math WOFF2 companion's OpenType MATH table
// is intact -- decoding JUST the WOFF2 header + table directory (no brotli
// decompression needed; the directory alone lists every SFNT table tag
// present). A subsetting tool -- the regression the verbatim-OTF bundling
// decision exists to prevent (05-RESEARCH Pitfall 6) -- drops whole tables
// from the directory, so directory presence is a complete survival proof.
//
// (A second deliverable, a Chrome-gated CI render smoke, lands in this same
// file in a follow-up commit.)

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

// --- Task 1: WOFF2 table-directory parsing (MATH-table survival) ----------

// woff2KnownTags is the WOFF2 spec's fixed 63-entry "Known Table Tags" array
// (https://www.w3.org/TR/WOFF2/#table_dir_format, section 6.1.1) --
// table-directory entries reference these tags by a 6-bit index (0-62)
// instead of spelling out all 4 bytes; index 63 means "arbitrary tag,
// spelled out literally in the next 4 bytes". MATH is index 31.
var woff2KnownTags = [63]string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca", "prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar", "mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill",
}

// parseUintBase128 decodes one WOFF2 UIntBase128 value (spec section 5.1): a
// big-endian base-128 varint, continuation bit in each byte's MSB, at most 5
// bytes, no redundant leading-zero byte. Returns the value and the number of
// bytes consumed.
func parseUintBase128(b []byte) (value uint32, consumed int, err error) {
	for i := 0; i < 5; i++ {
		if i >= len(b) {
			return 0, 0, fmt.Errorf("truncated UIntBase128 at byte %d", i)
		}
		if i == 0 && b[0] == 0x80 {
			return 0, 0, fmt.Errorf("UIntBase128 has a disallowed leading zero byte")
		}
		if value&0xFE000000 != 0 {
			return 0, 0, fmt.Errorf("UIntBase128 overflows uint32")
		}
		value = value<<7 | uint32(b[i]&0x7F)
		if b[i]&0x80 == 0 {
			return value, i + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("UIntBase128 longer than 5 bytes")
}

// woff2TableTags decodes JUST a WOFF2 file's header + table directory (per
// the W3C WOFF2 spec sections 4 and 5) and returns every SFNT table tag it
// lists, WITHOUT touching the brotli-compressed table-data block. This is
// sufficient -- and far simpler than a full WOFF2 decoder -- to prove a
// conversion did not SUBSET the font: a subsetting tool drops whole tables
// (like MATH) from the directory entirely, so directory presence alone is
// the tofu-regression signal this function exists to check.
//
// Only glyf/loca (TrueType outlines) and hmtx carry an OPTIONAL
// transformation with its own length field in the directory; every other
// table (including MATH, CFF, GSUB, ...) never does, per spec section 5.
func woff2TableTags(data []byte) ([]string, error) {
	const headerSize = 48 // WOFF2Header, spec section 4.
	if len(data) < headerSize || string(data[0:4]) != "wOF2" {
		return nil, fmt.Errorf("not a WOFF2 file (missing 'wOF2' signature)")
	}
	numTables := int(binary.BigEndian.Uint16(data[12:14]))

	pos := headerSize
	tags := make([]string, 0, numTables)
	for i := 0; i < numTables; i++ {
		if pos >= len(data) {
			return nil, fmt.Errorf("table directory truncated at entry %d/%d", i, numTables)
		}
		flags := data[pos]
		pos++
		tagIndex := flags & 0x3F
		xformVersion := (flags >> 6) & 0x3

		var tag string
		switch {
		case tagIndex == 63:
			if pos+4 > len(data) {
				return nil, fmt.Errorf("truncated arbitrary tag at entry %d", i)
			}
			tag = string(data[pos : pos+4])
			pos += 4
		case int(tagIndex) < len(woff2KnownTags):
			tag = woff2KnownTags[tagIndex]
		default:
			return nil, fmt.Errorf("known-tag index %d out of range at entry %d", tagIndex, i)
		}

		if _, n, err := parseUintBase128(data[pos:]); err != nil {
			return nil, fmt.Errorf("origLength at entry %d (%s): %w", i, tag, err)
		} else {
			pos += n
		}

		hasTransformLength := false
		switch tag {
		case "glyf", "loca":
			hasTransformLength = xformVersion == 0
		case "hmtx":
			hasTransformLength = xformVersion == 1
		}
		if hasTransformLength {
			if _, n, err := parseUintBase128(data[pos:]); err != nil {
				return nil, fmt.Errorf("transformLength at entry %d (%s): %w", i, tag, err)
			} else {
				pos += n
			}
		}

		tags = append(tags, tag)
	}
	return tags, nil
}

// TestWoff2MathTableSurvivesConversion is the MATH-table survival proof this
// TRD exists for: decode the bundled/embedded STIX Two Math WOFF2's table
// directory and assert a MATH entry is present. Runs against the ACTUAL
// go:embed'd bytes (stixTwoMathWOFF2, fonts.go) -- proving both the on-disk
// asset AND the embed wiring are correct, dependency-free (no fonttools,
// no Chrome).
func TestWoff2MathTableSurvivesConversion(t *testing.T) {
	if len(stixTwoMathWOFF2) == 0 {
		t.Skip("STIX Two Math WOFF2 not embedded (deferred asset)")
	}
	if len(stixTwoMathWOFF2) < 1024 {
		t.Fatalf("embedded WOFF2 is suspiciously small (%d bytes) -- looks truncated, not a real font", len(stixTwoMathWOFF2))
	}

	tags, err := woff2TableTags(stixTwoMathWOFF2)
	if err != nil {
		t.Fatalf("decoding embedded WOFF2 table directory: %v", err)
	}

	found := false
	for _, tag := range tags {
		if tag == "MATH" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("MATH table missing from WOFF2 table directory (tables present: %v) -- the conversion likely SUBSETTED the font, reintroducing the exact tofu regression the verbatim-font bundling decision exists to prevent", tags)
	}
}

// TestFontFaceDataURIWoff2 mirrors determinism_test.go's TestFontFaceDataURI
// for the new WOFF2 accessor: same @font-face shape, format('woff2') instead
// of format('opentype'), and confirms the ORIGINAL FontFaceDataURI (OTF)
// keeps working unchanged alongside it -- additive, not replace-in-place
// (08-05 error_recovery).
func TestFontFaceDataURIWoff2(t *testing.T) {
	got := FontFaceDataURIWoff2()

	if got == "" {
		t.Skip("STIX Two Math WOFF2 not embedded (deferred asset)")
	}

	const wantPrefix = "@font-face{font-family:'STIX Two Math';src:url(data:font/woff2;base64,"
	if !strings.HasPrefix(got, wantPrefix) {
		n := len(got)
		if n > 120 {
			n = 120
		}
		t.Fatalf("FontFaceDataURIWoff2 has unexpected shape (want prefix %q), got: %q...", wantPrefix, got[:n])
	}
	if !strings.Contains(got, "format('woff2')") {
		t.Fatalf("FontFaceDataURIWoff2 missing format('woff2'): %q", got)
	}
	if !strings.HasSuffix(got, "}") {
		t.Fatalf("FontFaceDataURIWoff2 rule is not closed: %q", got)
	}

	if otf := FontFaceDataURI(); otf == "" || !strings.Contains(otf, "format('opentype')") {
		t.Fatalf("FontFaceDataURI (OTF) regressed after adding the WOFF2 accessor: %q", otf)
	}
}
