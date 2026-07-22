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

// fonts_test.go is 08-05's two deliverables:
//
//  1. TestWoff2MathTableSurvivesConversion: a dependency-free (Go stdlib
//     only) proof that the bundled STIX Two Math WOFF2 companion's OpenType
//     MATH table is intact -- decoding JUST the WOFF2 header + table
//     directory (no brotli decompression needed; the directory alone lists
//     every SFNT table tag present). A subsetting tool -- the regression the
//     verbatim-OTF bundling decision exists to prevent (05-RESEARCH Pitfall
//     6) -- drops whole tables from the directory, so directory presence is
//     a complete survival proof.
//
//  2. TestStixMathTableSmoke: a Chrome-gated CI smoke that renders a KNOWN
//     formula (a parenthesized fraction) through the SAME
//     ApplyDeterminism+LoadHTML path every convert/ exporter uses, and
//     pixel-checks that the OpenType MATH table's stretchy-construction data
//     actually drove the render -- catching a missing/stripped-MATH
//     regression that a bare "did anything render" check would miss (see
//     that test's doc comment for the empirical derivation of its
//     thresholds). t.Skips cleanly with no Chrome present, matching every
//     other Chrome-gated test in this package.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/convert"
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

// --- Task 2: Chrome-gated MATH-table render smoke --------------------------

// TestStixMathTableSmoke is the CI MATH-table pixel-check smoke (08-05 Task
// 2): render a KNOWN formula that requires the OpenType MATH table's
// glyph-variant/stretchy-construction data -- a parenthesized fraction,
// "(1/2)" -- through the SAME headless-Chrome determinism path
// (ApplyDeterminism + LoadHTML) every convert/ exporter uses, with STIX Two
// Math as the SOLE font-family (no fallback), and pixel-check the result two
// ways:
//
//  1. Ink presence: the captured region is not blank/empty (guards the
//     coarse "font failed to load at all" failure).
//  2. Stretchy-height: the rendered construct's bounding-box HEIGHT must
//     clear a threshold that only a WORKING MathML stretchy-operator lookup
//     (driven by the font's MATH table MathVariants/Construction data) can
//     produce.
//
// The stretchy-height check's thresholds were EMPIRICALLY DERIVED during
// this TRD: the exact same markup+font-size, rendered against STIX Two Math
// with its MATH table intact, produces a ~152px-tall bounding box; rendered
// against a byte-for-byte copy of that SAME font with ONLY its MATH table
// surgically removed (via fonttools, as a throwaway negative-control
// fixture -- never shipped), it collapses to ~85px -- MathML Core simply
// stops stretching the parens to match the fraction's height once the
// resolved font lacks MATH data. A stripped/subsetted-away MATH table
// (05-RESEARCH Pitfall 6, the exact regression the verbatim-font bundling
// decision exists to prevent) reproduces that SAME collapse. This is a much
// stronger signal than a bare "did anything render" check: a LONE operator
// (e.g. a solo "∑" with nothing to stretch to match) was ALSO empirically
// confirmed to render IDENTICALLY with or without a MATH table -- this
// smoke deliberately uses a construct that FORCES the stretchy mechanism to
// engage, so a stripped MATH table cannot hide behind a fine-looking solo
// glyph.
//
// Reuses convert/chrome's documented "pixel-diff-under-threshold, not
// byte-identical" determinism contract (determinism.go) -- Go stdlib
// image/png only, no image-diff dependency (08-05 anti_patterns). Gated on
// Chrome presence: t.Skip cleanly (mirroring load_test.go /
// session_test.go's Chrome-gated tests) when no Chrome/Chromium is
// discoverable, so `go test ./...` stays green in a browserless CI leg.
func TestStixMathTableSmoke(t *testing.T) {
	if _, _, err := Discover(DiscoverOptions{}); err != nil {
		t.Skipf("no Chrome discovered, skipping MATH-table render smoke: %v", err)
	}

	fontFace := FontFaceDataURIWoff2()
	if fontFace == "" {
		fontFace = FontFaceDataURI()
	}
	if fontFace == "" {
		t.Skip("neither STIX Two Math WOFF2 nor OTF is embedded (deferred asset) -- nothing to render")
	}

	sess, err := New(convert.Options{})
	if err != nil {
		t.Skipf("could not start a Chrome session, skipping MATH-table render smoke: %v", err)
	}
	defer sess.Close()

	tab, cancel := sess.NewTab()
	defer cancel()

	if err := ApplyDeterminism(tab, 800, 600); err != nil {
		t.Fatalf("ApplyDeterminism: %v", err)
	}

	const fontSizePx = 80
	html := `<!doctype html><html><head><meta charset="utf-8"><style>` +
		fontFace +
		`body{margin:0;background:#ffffff;}` +
		`#x{font-family:'STIX Two Math';font-size:` + strconv.Itoa(fontSizePx) + `px;display:inline-block;}` +
		`</style></head><body><math id="x"><mrow><mo>(</mo><mfrac><mn>1</mn><mn>2</mn></mfrac><mo>)</mo></mrow></math></body></html>`

	if err := LoadHTML(tab, html); err != nil {
		t.Fatalf("LoadHTML: %v", err)
	}

	var shot []byte
	if err := chromedp.Run(tab, chromedp.Screenshot("#x", &shot, chromedp.ByQuery)); err != nil {
		t.Fatalf("screenshotting #x: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(shot))
	if err != nil {
		t.Fatalf("decoding screenshot PNG: %v", err)
	}

	inkFrac, bbox := mathGlyphInkBBox(img)

	const minInkFrac = 0.03
	if inkFrac < minInkFrac {
		t.Fatalf("MATH-table smoke: rendered \"(1/2)\" has almost no ink (%.4f, want >= %.4f) inside its own bounding box %v -- looks like the font failed to render at all (tofu/blank)", inkFrac, minInkFrac, bbox)
	}

	const minStretchHeightPx = 110
	if bbox.Dy() < minStretchHeightPx {
		t.Fatalf("MATH-table smoke: rendered \"(1/2)\" bounding-box height is %dpx (want >= %dpx) -- the stretchy parens did not grow to enclose the fraction, which is exactly the symptom of a MISSING/STRIPPED OpenType MATH table (05-RESEARCH Pitfall 6); got bbox %v", bbox.Dy(), minStretchHeightPx, bbox)
	}
}

// mathGlyphInkBBox scans img for "ink" pixels (anything meaningfully darker
// than the white background the smoke's HTML fixture sets) and returns the
// fraction of the image that is ink, plus the tightest axis-aligned bounding
// box enclosing all ink pixels. Go stdlib image/png decoding only -- no
// image-diff dependency (08-05 anti_patterns).
func mathGlyphInkBBox(img image.Image) (inkFraction float64, bbox image.Rectangle) {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	inkCount, total := 0, 0

	// A generous "near-white" cutoff: real glyph ink (even thin stems, with
	// AA edge pixels blended toward white) reads well below this on at
	// least one channel; pure background does not.
	const whiteCutoff = 0xF000 // out of 0xFFFF per RGBA() channel.

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			total++
			r, g, b, _ := img.At(x, y).RGBA()
			if r < whiteCutoff || g < whiteCutoff || b < whiteCutoff {
				inkCount++
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if total == 0 || inkCount == 0 {
		return 0, image.Rectangle{}
	}
	return float64(inkCount) / float64(total), image.Rect(minX, minY, maxX+1, maxY+1)
}
