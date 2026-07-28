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

package paged

// ScaffoldCSS is the base stylesheet chase/theme prepends before a packed
// theme's own rules for paged output — the paged analogue of
// profiles/slides' ScaffoldCSS.
//
// Three things it must get right, none of which apply to a slide deck:
//
//  1. A page is a PHYSICAL box. Each section is a fixed-size page with real
//     margins, and page breaks between sections are enforced for print. The
//     screen presentation stacks pages vertically with a gap and a shadow, so
//     an on-screen preview reads as a document rather than one endless column.
//
//  2. Page NUMBERS come from a CSS counter, not from an attribute. Marpit
//     stamps data-marpit-pagination onto each slide at parse time; a paged
//     document instead increments a counter per section, so numbering survives
//     sections being added or reordered without re-running the directive pass.
//
//  3. @page carries the physical size for print/PDF. Without it a browser
//     prints at ITS default page size and the carefully-sized section boxes
//     get scaled or clipped — the single most common way "it looked right on
//     screen" turns into a broken PDF. The size is re-stated per named size by
//     chase/theme's size handling; the @page rule here carries the margin box
//     and the running header/footer slots.
//
// The `--edenpress-*` custom properties are the documented override surface: a
// theme sets them rather than re-declaring the structural rules.
const ScaffoldCSS = `
:root {
  --edenpress-page-margin: 20mm;
  --edenpress-page-gap: 12px;
  --edenpress-header-text: "";
  --edenpress-footer-text: "";
  --edenpress-rule-color: #d0d0d0;
}

/* Page-number counter: reset once on the container, incremented per page. */
div.edenpress-paged {
  counter-reset: edenpress-page;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--edenpress-page-gap);
  background: #f2f2f2;
}

/* Each section IS a page: a fixed physical box, never a flowing column. */
div.edenpress-paged > section {
  counter-increment: edenpress-page;
  position: relative;
  box-sizing: border-box;
  padding: var(--edenpress-page-margin);
  background: #fff;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.18);
}

/* Running header and footer live inside the page's own margin box, so they
   cannot collide with body content however long the content runs. */
div.edenpress-paged > section::before {
  content: var(--edenpress-header-text);
  position: absolute;
  top: calc(var(--edenpress-page-margin) / 2);
  left: var(--edenpress-page-margin);
  right: var(--edenpress-page-margin);
  font-size: 0.75em;
  color: #666;
  border-bottom: 1px solid var(--edenpress-rule-color);
  padding-bottom: 2px;
}

/* Page number. chase/theme's pagination pass rewrites this content value via
   the PaginationRule; the counter is the default. */
div.edenpress-paged > section::after {
  content: counter(edenpress-page);
  position: absolute;
  bottom: calc(var(--edenpress-page-margin) / 2);
  right: var(--edenpress-page-margin);
  font-size: 0.75em;
  color: #666;
}

/* Long-form typography defaults: a measure that stays readable, and headings
   that do not strand themselves at the foot of a page. */
div.edenpress-paged > section > * {
  max-width: 100%;
}

div.edenpress-paged > section h1,
div.edenpress-paged > section h2,
div.edenpress-paged > section h3 {
  break-after: avoid;
  page-break-after: avoid;
}

div.edenpress-paged > section p,
div.edenpress-paged > section li {
  orphans: 2;
  widows: 2;
}

div.edenpress-paged > section table {
  border-collapse: collapse;
  width: 100%;
}

div.edenpress-paged > section th,
div.edenpress-paged > section td {
  border: 1px solid var(--edenpress-rule-color);
  padding: 0.35em 0.5em;
  text-align: left;
}

div.edenpress-paged > section thead {
  display: table-header-group; /* repeat the header across page breaks */
}

div.edenpress-paged > section blockquote {
  margin: 0.8em 0;
  padding-left: 1em;
  border-left: 3px solid var(--edenpress-rule-color);
  color: #444;
}

/* Print: drop the screen chrome, enforce one section per physical page. */
@media print {
  @page {
    margin: 0;
  }

  div.edenpress-paged {
    display: block;
    gap: 0;
    background: none;
  }

  div.edenpress-paged > section {
    box-shadow: none;
    break-after: page;
    page-break-after: always;
  }

  div.edenpress-paged > section:last-child {
    break-after: auto;
    page-break-after: auto;
  }
}
`
