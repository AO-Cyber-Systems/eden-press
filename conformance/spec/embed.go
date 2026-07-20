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

// Package spec vendors the official CommonMark and GFM specification test
// suites (generated via each project's own test/spec_tests.py --dump-tests) and
// exposes them as go:embed-ed byte slices so the conformance sweep is hermetic —
// no network fetch or file read at test time. See VERSIONS.txt for the pinned
// source refs and measured example counts, and README.md for regeneration.
package spec

import _ "embed"

// CommonMark is the full CommonMark spec test suite (tag 0.31.2, 652 examples).
//
//go:embed commonmark/spec.json
var CommonMark []byte

// GFM is the cmark-gfm master test/spec.txt suite (670 examples — a
// CommonMark-synced mirror plus inline "(extension)" sections).
//
//go:embed gfm/spec.json
var GFM []byte

// GFMExtensions is the cmark-gfm master test/extensions.txt suite (30 examples —
// GFM-only additions: tables, strikethrough, autolinks, disallowed-raw-HTML,
// footnotes, task lists).
//
//go:embed gfm/extensions.json
var GFMExtensions []byte
