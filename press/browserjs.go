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

package press

import "github.com/AO-Cyber-Systems/eden-press/press/themes"

// BrowserFitJS re-exports press/themes.BrowserFitJS at the press package
// root (TRD 04-01): it returns the verbatim Marp Core browser fit/
// auto-scaling helper (lib/browser.js) that reads the DOM markers CORE-09's
// auto-fit battery emits.
//
// The re-export exists so a consumer that imports ONLY press/ (the
// Objective-3-frozen import boundary -- see press/press.go's blank
// profiles/slides import doc) can splice this script into its own HTML
// shell without reaching into press/themes directly. The CLI (04-03's
// `--auto-fit-script`) is the first named consumer.
func BrowserFitJS() string { return themes.BrowserFitJS() }
