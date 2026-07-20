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

// Package edenpress is the root of Eden Press: a clean-room, JavaScript-free Go
// implementation that renders Marp-compatible presentation documents from Markdown,
// emitting structured data (not just HTML) with no Node runtime and no headless browser
// for HTML/structured output.
//
// This root package intentionally carries only the module's package documentation.
// The functional surface lives in subpackages:
//
//   - conformance/ — the language-neutral acceptance gate (golden corpus, CommonMark/GFM
//     spec sweep, DOM-normalized HTML diff, and CSS-AST diff comparator) that every engine
//     objective is validated against.
//   - themes/      — vendored Marp assets (default/gaia/uncover themes + browser-fit script),
//     landing in Objective 3 under their original MIT license and Marp copyright.
//   - tools/       — build-time-only tooling (e.g. the throwaway corpus generator), excluded
//     from the shipped runtime graph.
//
// Eden Press is NOT affiliated with, endorsed by, or sponsored by the Marp team.
// See the repository NOTICE file for full third-party attribution.
package edenpress
