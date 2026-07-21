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

package slides

// scaffold.go embeds the two static, verbatim CSS blocks 01-RESEARCH.md
// records from Marpit's own postcss/scaffold.js and
// postcss/advanced_background.js: the base slide-reset ("scaffold") CSS
// every packed theme (other than the scaffold theme itself) is prepended
// with, and the inline-SVG advanced-background support CSS appended when
// inline-SVG rendering is enabled. Both are relocated byte-for-byte from
// chase/theme/scaffold.go (TRD 02-03, MODEL-04's de-hardcoding move) --
// this TRD's must_haves require the values profiles/slides hands to
// chase/theme to be byte-identical to what chase/theme used to hardcode,
// which is what keeps the Objective-1 cssdiff/corpus acceptance gates
// green.
//
// Neither constant carries an `@theme` metadata header: chase/theme parses
// them via its plain, meta-less loader wherever they are consumed (Pack),
// matching how Marpit's own scaffold/advanced-background CSS is plain,
// unauthored-as-a-theme CSS text.

// ScaffoldCSS is Marpit's slide-reset CSS (postcss/scaffold.js), prepended
// to every packed theme except the scaffold theme itself.
const ScaffoldCSS = `
section {
  width: 1280px;
  height: 720px;
  box-sizing: border-box;
  overflow: hidden;
  position: relative;
  scroll-snap-align: center center;
  -webkit-text-size-adjust: 100%;
  text-size-adjust: 100%;
}
section::after {
  bottom: 0;
  content: attr(data-marpit-pagination);
  padding: inherit;
  pointer-events: none;
  position: absolute;
  right: 0;
}
section:not([data-marpit-pagination])::after {
  display: none;
}
:where(h1) {
  font-size: 2em;
  margin-block: 0.67em;
}
video::-webkit-media-controls {
  will-change: transform;
}
`

// AdvancedBackgroundCSS is Marpit's inline-SVG advanced-background support
// CSS (postcss/advanced_background.js), appended when inline-SVG rendering
// is enabled.
//
// One rule in this block -- `:marpit-container > svg[data-marpit-svg] >
// foreignObject[...]` -- uses chase/theme/selector's container placeholder
// ALONE (not paired with the slide placeholder scope.go's Prepend/Replace
// expect together), so 01-01's locked selector.Replace cannot resolve it;
// chase/theme's pass_advancedbg.go documents this as a deliberate,
// deferred scope-narrowing gap rather than modifying the 01-01-owned
// selector package.
const AdvancedBackgroundCSS = `
section[data-marpit-advanced-background="background"] {
  columns: initial !important;
  display: block !important;
  padding: 0 !important;
}
section[data-marpit-advanced-background="background"]::before,
section[data-marpit-advanced-background="background"]::after,
section[data-marpit-advanced-background="content"]::before,
section[data-marpit-advanced-background="content"]::after {
  display: none !important;
}
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] {
  all: initial;
  display: flex;
  flex-direction: row;
  height: 100%;
  overflow: hidden;
  width: 100%;
}
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container][data-marpit-advanced-background-direction="vertical"] {
  flex-direction: column;
}
section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split] > div[data-marpit-advanced-background-container] {
  width: var(--marpit-advanced-background-split, 50%);
}
section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split="right"] > div[data-marpit-advanced-background-container] {
  margin-left: calc(100% - var(--marpit-advanced-background-split, 50%));
}
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] > figure {
  all: initial;
  background-position: center;
  background-repeat: no-repeat;
  background-size: cover;
  flex: auto;
  margin: 0;
}
section[data-marpit-advanced-background="content"],
section[data-marpit-advanced-background="pseudo"] {
  background: transparent !important;
}
section[data-marpit-advanced-background="pseudo"],
:marpit-container > svg[data-marpit-svg] > foreignObject[data-marpit-advanced-background="pseudo"] {
  pointer-events: none !important;
}
section[data-marpit-advanced-background-split] {
  width: 100%;
  height: 100%;
}
`
