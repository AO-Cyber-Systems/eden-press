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

// fontFaceCSS is the @font-face rule template FontFaceDataURI fills in with
// the embedded font's base64 payload. format('opentype') matches the OTF
// container (an OpenType/CFF font, magic bytes "OTTO").
const fontFaceCSS = `@font-face{font-family:'STIX Two Math';src:url(data:font/otf;base64,%s) format('opentype');}`

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
