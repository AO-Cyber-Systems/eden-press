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

// Package convert is the ONE AND ONLY package in the eden-press module that
// imports github.com/chromedp/chromedp (and its cdproto companion). Every
// chromedp-touching type, launch flag, and CDP call lives under convert/ or
// one of its subpackages (convert/chrome, and the sibling exporters
// convert/pdf and convert/png added by later TRDs).
//
// press/, chase/, and profiles/ MUST NEVER import convert/ (directly or
// transitively) -- doing so would pull chromedp into the public render path,
// breaking the zero-chromedp boundary that scripts/check-no-chromedp.sh
// mechanically enforces in CI (`go list -deps ./press/... ./chase/...
// ./profiles/...` must contain zero chromedp packages). The dependency edge
// is strictly one-directional: convert/ imports press/ (to consume
// press.Output as its rendering input), and press/ never imports convert/.
//
// This boundary is why chromedp is provisioned here, in this TRD, and
// nowhere else: the moment chromedp entered go.mod, check-no-chromedp.sh
// became the load-bearing proof that the one-directional edge holds.
package convert
