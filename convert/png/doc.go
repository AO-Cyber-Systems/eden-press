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

// Package png implements EXP-02: per-slide PNG/JPEG export via chromedp
// element screenshots. ToImages composes the self-contained HTML (out.HTML +
// chrome.ComposeCSS(out.CSS)), opens ONE tab on the caller-supplied
// convert/chrome.Session sized to the deck's own size-table entry, folds in
// the 05-02 determinism recipe (chrome.ApplyDeterminism + chrome.LoadHTML --
// never re-derived here), then loops over the deck's slides -- one
// chromedp.Screenshot call, one buffer, one <section>, in document order --
// returning N image buffers for an N-slide deck.
//
// This is chromedp.Screenshot in a LOOP, deliberately NOT ScreenshotNodes:
// 05-RESEARCH Open Question #2 resolved in favor of the individually-verified
// Screenshot-per-section call (one call -> one file -> one slide) over
// ScreenshotNodes' stitched-image semantics (a slice of nodes collapsed into
// ONE picbuf) -- the wrong shape for N separate per-slide files.
//
// PNG is the default output format (convert.PNG, ImageFormat's zero value);
// JPEG (convert.JPEG) is opt-in and produced by decoding the PNG buffer
// chromedp.Screenshot always emits (page.CaptureScreenshotFormatPng is
// hardcoded inside chromedp's ScreenshotNodes -- Screenshot exposes no format
// parameter of its own) and re-encoding it via image/jpeg.
//
// Inline-SVG-mode caveat (05-RESEARCH Open Question #3 / Pitfall 4): Marpit's
// inline-SVG render mode (chase/markdown/inlinesvg.go) wraps EACH slide's
// <section> in its OWN <svg data-marpit-svg><foreignObject>...</foreignObject>
// </svg> sibling directly under div.marpit -- not one shared <svg> containing
// N foreignObjects. profiles/slides.Container(true) ("div.marpit > svg >
// foreignObject") is a CSS-SCOPING selector (it matches every slide
// uniformly, by design -- a theme rule does not care which slide it is
// scoped to) and carries no positional information on its own. To select the
// k-th slide's <section> for a per-slide Screenshot call, ToImages attaches
// the position discriminator to the "svg" sibling index instead --
// "div.marpit > svg:nth-of-type(k) > foreignObject > section" -- since every
// foreignObject has exactly one <section> child (nth-of-type on "section"
// itself would always resolve to position 1 inside a single-child parent,
// never selecting slide 2+). See png_test.go's inline-SVG smoke test, which
// proves Chrome resolves this three-levels-deep, foreignObject-nested
// bounding box correctly for a real multi-slide fixture; the error_recovery
// fallback (screenshot the wrapping foreignObject/svg instead) is documented
// there and in 05-04-SUMMARY.md if the smoke ever finds a blank/zero-sized
// capture.
package png
