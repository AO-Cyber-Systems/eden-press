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

import (
	_ "embed"
	"encoding/base64"
	"fmt"
)

// stixTwoMathOTF embeds the STIX Two Math OpenType font VERBATIM (unmodified,
// as its SIL Open Font License 1.1 requires) -- see NOTICE for the full
// attribution. Bundling the project's own OTF (rather than a Google Fonts CDN
// copy) sidesteps a repeatedly-reported subsetting bug that strips the
// OpenType MATH table from CDN-served copies, which would silently
// reintroduce tofu (05-RESEARCH Pitfall 6).
//
// Sourced from https://github.com/stipub/stixfonts, tag v2.13,
// fonts/static_otf/STIXTwoMath-Regular.otf.
//
//go:embed fonts/STIXTwoMath-Regular.otf
var stixTwoMathOTF []byte

// stixTwoMathWOFF2 embeds the OFFICIAL stipub/stixfonts WOFF2 companion to
// stixTwoMathOTF above -- the SAME v2.13 tag, the SAME "static_otf" build
// family, just its WOFF2 sibling directory (research Open Q4: resolved YES,
// an official build exists, so no local conversion tool is needed). It is
// the byte-identical-in-substance, smaller-on-the-wire twin of the OTF: same
// glyph count (6760), same OpenType MATH table (see fonts_test.go's
// TestWoff2MathTableSurvivesConversion), same OFL-1.1 license -- see NOTICE
// for the full attribution, including the survival verification.
//
// Sourced from https://github.com/stipub/stixfonts, tag v2.13,
// fonts/static_otf_woff2/STIXTwoMath-Regular.woff2.
//
//go:embed fonts/STIXTwoMath-Regular.woff2
var stixTwoMathWOFF2 []byte

// fontFaceCSS is the @font-face rule template FontFaceDataURI fills in with
// the embedded font's base64 payload. format('opentype') matches the OTF
// container (an OpenType/CFF font, magic bytes "OTTO").
const fontFaceCSS = `@font-face{font-family:'STIX Two Math';src:url(data:font/otf;base64,%s) format('opentype');}`

// fontFaceCSSWoff2 mirrors fontFaceCSS for the WOFF2 companion --
// format('woff2') matches the WOFF2 container. Same font-family name (a
// caller picks ONE of the two rules; they are not meant to be concatenated
// into a single multi-src rule -- see FontFaceDataURIWoff2's doc comment).
const fontFaceCSSWoff2 = `@font-face{font-family:'STIX Two Math';src:url(data:font/woff2;base64,%s) format('woff2');}`

// FontFaceDataURI returns an @font-face rule embedding STIX Two Math as a
// base64 data-URI so headless Chrome renders MathML glyphs (no tofu) with no
// external font fetch -- no network round-trip, no local-file-access
// posture, consistent with LoadHTML's SetDocumentContent-only contract.
//
// Returns "" when the OTF asset was deferred (see the TRD's error_recovery
// path): the mechanism (this function, ComposeCSS's injection point) still
// lands even if the binary itself is temporarily absent, so no caller need
// special-case a missing font.
func FontFaceDataURI() string {
	if len(stixTwoMathOTF) == 0 {
		return ""
	}
	b64 := base64.StdEncoding.EncodeToString(stixTwoMathOTF)
	return fmt.Sprintf(fontFaceCSS, b64)
}

// FontFaceDataURIWoff2 returns an @font-face rule embedding STIX Two Math's
// WOFF2 companion as a base64 data-URI -- the SAME font-family
// ('STIX Two Math'), same glyph/MATH-table content as FontFaceDataURI's OTF,
// just WOFF2's brotli-compressed container (base64-inflated payload roughly
// two-thirds the OTF's -- 08-05 must_haves: "so the live serve/preview HTTP
// path can ship the smaller payload instead of the 838KB base64-inflated OTF
// on every reload").
//
// This is ADDITIVE, not a replacement: FontFaceDataURI (OTF) is unchanged and
// stays available for any caller/environment that needs it (08-05
// error_recovery -- keep the OTF variant available rather than risk breaking
// an existing format('opentype') assertion). A caller wanting the smaller
// payload picks THIS accessor instead; ComposeCSS itself is untouched by this
// TRD (out of its declared file scope) and keeps calling FontFaceDataURI.
//
// Returns "" when the WOFF2 asset was deferred, mirroring FontFaceDataURI's
// same-shaped fallback contract.
func FontFaceDataURIWoff2() string {
	if len(stixTwoMathWOFF2) == 0 {
		return ""
	}
	b64 := base64.StdEncoding.EncodeToString(stixTwoMathWOFF2)
	return fmt.Sprintf(fontFaceCSSWoff2, b64)
}
