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
	"context"
	"fmt"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// LoadHTML loads a self-contained HTML document into a tab via
// page.SetDocumentContent -- NOT a data: URL (a documented truncation bug
// silently blanks large decks) and NOT a temp file/file:// (which pulls in
// the local-file-access security posture Marp CLI gates behind
// --allow-local-files). It blocks until the browser's own
// `document.fonts.ready` promise resolves before returning, closing the
// data-URI-font FOUT race a caller would otherwise hit if it captured a
// screenshot/PDF immediately after the content loads (05-RESEARCH Pattern 2
// item 5, Pitfall B).
//
// ctx must be a chromedp tab context (e.g. from Session.NewTab()), ideally
// already run through ApplyDeterminism so the viewport/timezone/locale are
// pinned before this content loads.
//
// Steps, in order:
//  1. Navigate("about:blank") -- SetDocumentContent needs an existing TOP
//     FRAME to target; a fresh tab has none until a navigation establishes
//     one (chromedp #703/#827).
//  2. page.GetFrameTree -- resolve that top frame's FrameID.
//  3. page.SetDocumentContent(frameID, html) -- inject the HTML directly,
//     bypassing both the data-URL and file:// paths entirely.
//  4. Evaluate "document.fonts.ready.then(() => true)" with awaitPromise SET
//     -- without awaitPromise, the call returns immediately (the promise is
//     still pending), which is the exact FOUT race this step exists to
//     close.
func LoadHTML(ctx context.Context, html string) error {
	var fontsReady bool
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			ft, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return fmt.Errorf("getting frame tree: %w", err)
			}
			if ft == nil || ft.Frame == nil {
				return fmt.Errorf("no top frame available for SetDocumentContent (about:blank navigation did not establish one)")
			}
			if err := page.SetDocumentContent(ft.Frame.ID, html).Do(ctx); err != nil {
				return fmt.Errorf("setting document content: %w", err)
			}
			return nil
		}),
		chromedp.Evaluate("document.fonts.ready.then(() => true)", &fontsReady, awaitPromise),
	)
	if err != nil {
		return fmt.Errorf("convert/chrome: LoadHTML: %w", err)
	}
	return nil
}

// awaitPromise is a chromedp.EvaluateOption that sets runtime.EvaluateParams'
// AwaitPromise flag -- chromedp has no built-in option for this, so it is
// applied via the documented ActionFunc-adjacent pattern of composing a raw
// cdproto params mutator. Without this, Evaluate's "document.fonts.ready.then
// (() => true)" call returns the still-pending Promise object rather than
// waiting for it to resolve.
func awaitPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}
