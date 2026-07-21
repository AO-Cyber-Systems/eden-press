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

// Package pdf delivers EXP-01: deterministic PDF export of an
// already-rendered deck (press.Output) via chromedp's Chrome DevTools
// Protocol Page.PrintToPDF call, driven through convert/chrome's Session
// pool and its shared determinism substrate (ComposeCSS/PageCSSInches/
// ApplyDeterminism/LoadHTML -- 05-02). This package never re-derives any
// piece of that substrate; it only composes on top of it.
//
// PrintToPDF is NOT a first-class chromedp.Action -- its Do(ctx) method
// returns THREE values (data []byte, stream io.StreamHandle, err error),
// so it cannot be passed directly to chromedp.Run alongside ordinary
// Actions. ToPDF (pdf.go) wraps the call in a chromedp.ActionFunc
// (05-RESEARCH Pattern 1) specifically to bridge that multi-value return
// into chromedp's single-error Action shape.
//
// Paper size is CSS-@page-driven, not a raw inches call: ToPDF sets
// WithPreferCSSPageSize(true) and relies on chrome.PageCSSInches(size) (via
// the composed <style>) to supply the @page rule -- there is no
// WithFormat("A4") convenience on this CDP command, and CSS-driven sizing
// is the correct idiom for a fixed-size slide deck (as opposed to a
// variable-length flowed document). WithPrintBackground(true) is set
// UNCONDITIONALLY, not offered as an opt-out: a Marp theme's background
// colors/images are real deck content, and Chrome's print pipeline drops
// background graphics by default (CDP's own PrintToPDFParams.PrintBackground
// doc: "Defaults to false"). All four margins are pinned to 0 so the
// printed page exactly matches the @page content box, with no printer-style
// gutter Chrome would otherwise impose.
//
// Determinism bar -- pixel-diff-under-threshold, NOT byte-identical:
// Chrome's own rendering/print pipeline has ACKNOWLEDGED PRNG-influenced
// non-determinism (font hinting/anti-aliasing, GPU-vs-software
// rasterization paths -- 05-RESEARCH Pitfall C) that convert/chrome's
// ApplyDeterminism recipe minimizes but cannot fully eliminate. Two ToPDF
// runs over the identical input are therefore verified STRUCTURALLY here
// (identical page count, identical page dimensions) as a proxy for the
// real acceptance bar, which in a full image pipeline is
// pixel-diff-under-threshold -- never a byte-for-byte comparison of the two
// PDFs. This is a DELIBERATE, documented contrast with the pure-Go press/
// path, which legitimately claims byte-identical output for identical
// input because it has no browser rendering step in it at all.
//
// The PDF export path also carries its OWN regression risk, distinct from
// the screenshot/PNG path (convert/png, 05-04): Chrome's print compositor
// is a structurally different subsystem, with two independently-documented
// PDF-only regressions (SVG-in-PDF since Chrome 108; print-pipeline/LPAC
// changes around Chrome 125 -- 05-RESEARCH Pitfall A/4). pdf_test.go's
// inline-SVG/foreignObject fixture exists specifically to catch a
// foreignObject that silently renders empty in the PDF path while still
// passing an HTML string comparison -- this is why the PDF path gets its
// own re-validation discipline (hardened further in 05-05), independent of
// the PNG path's own coverage.
package pdf
