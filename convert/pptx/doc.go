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

// Package pptx is Eden Press's hand-rolled, stdlib-only (archive/zip +
// encoding/xml) OOXML PresentationML writer: it emits editable-text-box
// .pptx files directly from chase/model.Document, with NO headless browser
// and NO third-party OOXML library (unioffice and its forks were evaluated
// and rejected -- AGPLv3 licensing / commercial license-key + network
// check-in; see 06-RESEARCH.md's re-confirmed decision gate). This package
// is a new top-level export surface and is never imported by press/, chase/,
// or profiles/ -- it consumes chase/model.Document as an output-only
// consumer, keeping the no-chromedp render-path boundary those packages
// enforce completely untouched.
package pptx
