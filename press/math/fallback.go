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

package math

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"codeberg.org/go-latex/latex/drawtex/drawimg"
	"codeberg.org/go-latex/latex/mtex"
	"github.com/yuin/goldmark/util"
)

// PNG-only fallback tuning. drawtex has NO SVG canvas (research pitfall — the
// requirement's "SVG/PNG" framing is corrected to PNG-only): its only canvases
// are drawimg (raster) and drawpdf. We raster to PNG and embed as a base64
// data-URI so press.Render stays a pure function (no temp files, no asset
// server, no network at render time).
const (
	mathFontSize = 12.0  // pt
	mathDPI      = 150.0 // crisp enough for slide rendering, small enough inline
)

// renderFallbackIMG renders raw LaTeX to a PNG via go-latex/latex's
// drawtex/drawimg and embeds it as a base64 data-URI `<img>`. block adds a
// modifier class so 03-08's sanitize allow-list / theme CSS can distinguish
// display from inline fallbacks.
//
// go-latex/latex/mtex is a limited renderer that PANICS on constructs it does
// not implement (e.g. superscripts, and every \begin{…} environment this
// battery routes here). safeRasterPNG contains that panic; on any failure the
// function degrades to a documented, sanitize-safe alt-only <img> stub — never
// a crash, never a silent drop (error_recovery). The stub is also the path when
// the go-latex dep is unreachable.
func renderFallbackIMG(raw string, block bool) string {
	alt := string(util.EscapeHTML([]byte(raw)))
	class := "math-fallback"
	if block {
		class += " math-fallback-block"
	}

	png, err := safeRasterPNG(raw)
	if err != nil || len(png) == 0 {
		// Documented stub: go-latex cannot raster this construct (or is
		// unreachable). Show the raw LaTeX as alt text so nothing breaks.
		return `<img class="` + class + `" alt="` + alt + `">`
	}
	b64 := base64.StdEncoding.EncodeToString(png)
	return `<img class="` + class + `" alt="` + alt + `" src="data:image/png;base64,` + b64 + `">`
}

// safeRasterPNG rasters `$raw$` to PNG bytes with go-latex/latex, recovering
// from mtex's panics (it panics rather than returning an error for unsupported
// constructs). nil fonts selects go-latex's built-in Go font backend, so the
// path has no font-file dependency.
func safeRasterPNG(raw string) (png []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			png = nil
			err = fmt.Errorf("go-latex mtex could not render %q: %v", raw, r)
		}
	}()

	var buf bytes.Buffer
	rd := drawimg.NewRenderer(&buf)
	// mtex expects math delimited by `$…$` (see go-latex mtex render tests).
	if e := mtex.Render(rd, "$"+raw+"$", mathFontSize, mathDPI, nil); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
