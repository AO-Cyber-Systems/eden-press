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

package pdf

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/chase/theme"
	"github.com/AO-Cyber-Systems/eden-press/convert/chrome"
	"github.com/AO-Cyber-Systems/eden-press/press"
	"github.com/AO-Cyber-Systems/eden-press/profiles/slides"
)

// Options is convert/pdf's exporter-specific knob surface. It is kept
// separate from convert.Options (the knob surface chrome.New consumes at
// Session-construction time) because ToPDF receives an ALREADY-BUILT
// *chrome.Session -- BrowserPath has nothing left to do by the time ToPDF
// runs. It is intentionally empty today: no named consumer has needed a
// PDF-specific knob yet, and it exists as a distinct type (not a bare
// struct{} inlined into ToPDF's signature) purely so a future knob (e.g. a
// print scale override) can be added without changing ToPDF's call
// signature.
type Options struct{}

// ToPDF exports out -- an already-rendered deck, typically press.Render's
// Output but equally a hand-built equivalent for testing -- to a
// deterministic PDF (see the package doc comment for the precise
// determinism bar this claims).
//
// Flow:
//  1. Resolve the slide size from out.Meta's "size" front-matter directive
//     (falling back to profiles/slides' 16:9 default when absent/unknown).
//  2. Compose a self-contained HTML document: out.HTML wrapped in a
//     <style> combining chrome.ComposeCSS(out.CSS) (base CSS + the
//     animation-kill override + the embedded STIX font) with
//     chrome.PageCSSInches(size) (the @page rule ToPDF's
//     WithPreferCSSPageSize(true) call honors).
//  3. Open a fresh tab on sess, apply the 05-02 ApplyDeterminism recipe
//     pinned to size's pixel dimensions, then load the composed document
//     via chrome.LoadHTML (SetDocumentContent, not a data: URL or
//     file://).
//  4. Run page.PrintToPDF INSIDE a chromedp.ActionFunc -- it is not a
//     first-class chromedp.Action; Do(ctx) returns three values (data,
//     stream, err) -- with WithPrintBackground(true),
//     WithPreferCSSPageSize(true), and all four margins pinned to 0.
//
// The returned []byte is the raw PDF payload (PrintToPDF's default
// TransferMode returns it inline as base64-decoded bytes, which is fine for
// slide-deck-sized output; a large-document streaming path is out of
// scope here).
func ToPDF(sess *chrome.Session, out press.Output, opts Options) ([]byte, error) {
	size := resolveSize(out)

	tab, cancel := sess.NewTab()
	defer cancel()

	if err := chrome.ApplyDeterminism(tab, int64(size.WidthPx), int64(size.HeightPx)); err != nil {
		return nil, fmt.Errorf("convert/pdf: ToPDF: %w", err)
	}

	if err := chrome.LoadHTML(tab, composeDocument(out, size)); err != nil {
		return nil, fmt.Errorf("convert/pdf: ToPDF: %w", err)
	}

	var data []byte
	err := chromedp.Run(tab, chromedp.ActionFunc(func(ctx context.Context) error {
		// PrintToPDF returns (data, stream, err) -- three values, so it is
		// NOT a first-class chromedp.Action and MUST be invoked from
		// inside an ActionFunc like this one (05-RESEARCH Pattern 1).
		pdfData, _, printErr := page.PrintToPDF().
			WithPrintBackground(true).
			WithPreferCSSPageSize(true).
			WithMarginTop(0).
			WithMarginBottom(0).
			WithMarginLeft(0).
			WithMarginRight(0).
			Do(ctx)
		if printErr != nil {
			return printErr
		}
		data = pdfData
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("convert/pdf: ToPDF: PrintToPDF: %w", err)
	}

	return data, nil
}

// resolveSize resolves the slide size ToPDF renders at: out.Meta's "size"
// front-matter directive value (Output.Meta is press.Render's top-level
// alias for Model.Meta), looked up against profiles/slides' own size
// table, falling back to that table's Default (16:9, 1280x720px) when the
// directive is absent or names an unregistered size.
func resolveSize(out press.Output) theme.Size {
	table := slides.New().Sizes()
	if name := out.Meta.Directives["size"]; name != "" {
		if s, ok := table.ByName[name]; ok {
			return s
		}
	}
	return table.Default
}

// composeDocument wraps out's rendered body-fragment HTML into a
// self-contained HTML document Chrome can load directly via
// page.SetDocumentContent: chrome.ComposeCSS(out.CSS) folds in the
// animation-kill override plus the embedded STIX font, and
// chrome.PageCSSInches(size) appends the @page rule that drives
// WithPreferCSSPageSize(true)'s paper sizing.
func composeDocument(out press.Output, size theme.Size) string {
	css := chrome.ComposeCSS(out.CSS) + chrome.PageCSSInches(size)

	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><style>`)
	b.WriteString(css)
	b.WriteString(`</style></head><body>`)
	b.WriteString(out.HTML)
	b.WriteString(`</body></html>`)
	return b.String()
}
