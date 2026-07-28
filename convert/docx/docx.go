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

// Package docx renders a chase/model.Document to an editable Word .docx,
// built DIRECTLY from the docmodel as hand-rolled OOXML -- zero rendered HTML,
// zero chromedp, zero Node. It is the sibling of convert/pptx: same
// Chrome-free posture, same deterministic OPC packager shape, same
// "real editable text, never a screenshot" guarantee.
//
// Every docmodel block kind maps onto a native Word construct: headings onto
// the built-in Heading1..6 styles (so Word's navigation pane and TOC fields
// work), lists onto numbering.xml definitions with real nesting levels, code
// onto a monospace preformatted style with its line structure intact, tables
// onto real <w:tbl> grids with repeating header rows and per-column
// alignment, and quotes onto Word's Quote style.
//
// Tables, images and quotes are only expressible here because schema v3
// (EPD-R1) added those block kinds: before it, a table's content was absent
// from the docmodel entirely and could not have reached this exporter at all.
//
// Two deliberate limitations, both visible rather than silent:
//   - An image is emitted as a captioned "[alt: src]" placeholder, not an
//     embedded picture. The docmodel carries only a URL, and resolving it
//     needs I/O and a base directory that belong to the caller.
//   - Math is emitted as its raw TeX in the code style, not converted to
//     OMML. TeX -> Office MathML is a sub-project in its own right.
//
// Both are documented in body.go at the point of emission.
package docx

import (
	"fmt"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// Options configures ToDOCX.
type Options struct {
	// Title sets the document's dc:title core property. When empty, the
	// first level-1 outline entry is used, then the first heading block,
	// then "Document".
	Title string

	// ContinuousSections suppresses the page break otherwise inserted
	// between consecutive model Sections. Set it for a flowing report;
	// leave it false for a document whose sections are genuine page or
	// slide boundaries.
	ContinuousSections bool

	// PageSize selects the page geometry. The zero value is US Letter;
	// pass PageA4 for A4.
	PageSize PageSize
}

// PageSize is a page geometry in twips (1/1440 inch), the unit
// WordprocessingML's w:pgSz and w:pgMar use.
type PageSize struct {
	WidthTwips  int
	HeightTwips int
}

// The two page geometries this package ships. Both leave 1-inch margins,
// giving the 9360-twip text width convert/docx's tables are sized against.
var (
	// PageLetter is 8.5in x 11in.
	PageLetter = PageSize{WidthTwips: 12240, HeightTwips: 15840}
	// PageA4 is 210mm x 297mm.
	PageA4 = PageSize{WidthTwips: 11906, HeightTwips: 16838}
)

// ToDOCX renders doc to an editable .docx byte slice, directly from the
// docmodel -- no HTML parsing, no browser. Output is deterministic: calling
// ToDOCX twice with the same document and Options yields byte-identical bytes,
// which lets a caller cache an export by content hash.
func ToDOCX(doc *model.Document, opts Options) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("docx: ToDOCX: doc is nil")
	}

	page := opts.PageSize
	if page == (PageSize{}) {
		page = PageLetter
	}

	title := opts.Title
	if title == "" {
		title = inferTitle(doc)
	}

	body := renderSections(doc.Sections, opts.ContinuousSections)

	// sectPr closes the body and carries page geometry + margins. It must be
	// the LAST child of <w:body>; Word treats a misplaced sectPr as corruption.
	sectPr := fmt.Sprintf(
		`<w:sectPr><w:pgSz w:w="%d" w:h="%d"/>`+
			`<w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"`+
			` w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`,
		page.WidthTwips, page.HeightTwips)

	documentXML := []byte(xmlDeclaration + fmt.Sprintf(
		`<w:document xmlns:w="%s" xmlns:r="%s"><w:body>%s%s</w:body></w:document>`,
		nsWordprocessing, nsDocRels, body, sectPr))

	overrides := []contentTypeOverride{
		{PartName: "/word/document.xml", ContentType: ctDocument},
		{PartName: "/word/styles.xml", ContentType: ctStyles},
		{PartName: "/word/numbering.xml", ContentType: ctNumbering},
		{PartName: "/docProps/core.xml", ContentType: ctCorePropsType},
		{PartName: "/docProps/app.xml", ContentType: ctExtendedType},
	}

	// Fixed part order -- never a map range -- so the package is byte-stable.
	// [Content_Types].xml first matches what Word itself writes.
	return buildZip([]part{
		{name: "[Content_Types].xml", content: buildContentTypesXML(overrides)},
		{name: "_rels/.rels", content: rootRelsXML()},
		{name: "docProps/core.xml", content: docPropsCoreXML(title)},
		{name: "docProps/app.xml", content: docPropsAppXML()},
		{name: "word/document.xml", content: documentXML},
		{name: "word/_rels/document.xml.rels", content: documentRelsXML()},
		{name: "word/styles.xml", content: stylesXML()},
		{name: "word/numbering.xml", content: numberingXML()},
	})
}

// inferTitle picks a document title: the first level-1 outline entry, else the
// first heading block in document order, else a neutral fallback. The outline
// is preferred because it is the deck-wide index and already reflects the
// authored heading hierarchy.
func inferTitle(doc *model.Document) string {
	for _, o := range doc.Outline {
		if o.Level == 1 && o.Text != "" {
			return o.Text
		}
	}
	for _, s := range doc.Sections {
		for _, b := range s.Blocks {
			if b.Kind == model.BlockHeading && b.Text != "" {
				return b.Text
			}
		}
	}
	return "Document"
}
