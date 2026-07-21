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

// pptx.go is the PUBLIC surface of the whole convert/pptx objective: ToPPTX
// composes 06-01 (Section.Blocks -> editable body text), 06-02 (EMU placement +
// grouped-shape transform), 06-03 (deterministic OPC packager + static part
// scaffold), and 06-05 (notes.go: Section.Notes -> notesSlideN.xml) into a
// real, editable-text-box .pptx built DIRECTLY from the chase/model docmodel
// -- zero rendered HTML, zero chromedp. Each Section becomes one
// ppt/slides/slideN.xml (slide.go), wired into presentation.xml's sldIdLst,
// presentation.xml.rels, and [Content_Types].xml Overrides three-fold per
// slide; a Section WITH notes additionally gets a ppt/notesSlides/notesSlideN.xml
// (+ rels -> slideN + notesMaster1) and the slide's own rels gain a ->
// notesSlideN entry, with the once-per-deck ppt/notesMasters/notesMaster1.xml
// emitted iff any section has notes. Assembled by 06-03's
// fixed-timestamp/zip.Store packager, in a FIXED order (determinism).
package pptx

import (
	"fmt"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// Options configures ToPPTX.
type Options struct {
	// SlideSize selects the deck aspect ratio. The zero value defaults to
	// SlideSize16x9 (widescreen); pass SlideSize4x3 for the 4:3 standard size.
	SlideSize SlideSize
}

// ToPPTX renders a chase/model.Document to an editable .pptx byte slice,
// DIRECTLY from the docmodel -- no HTML parsing, no browser. Each doc.Section
// becomes one slide whose title (its lowest-Level Outline heading) and body
// (its Blocks: paragraphs, lists, headings) are real, editable <p:sp> text-box
// shapes with <a:t> runs (never a screenshot image). Output is deterministic:
// calling ToPPTX twice with the same document and Options yields byte-identical
// bytes.
func ToPPTX(doc *model.Document, opts Options) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("pptx: ToPPTX: doc is nil")
	}

	size := opts.SlideSize
	if size == (SlideSize{}) {
		size = SlideSize16x9
	}

	n := len(doc.Sections)

	// The FIXED singleton relationships in presentation.xml.rels occupy rId1..
	// rId5 (master, theme, presProps, viewProps, tableStyles -- see
	// parts_static.go); per-slide relationships therefore start at rId6.
	const firstSlideRelNum = 6

	// slidePart carries a rendered slide's two OPC parts (content + .rels) with
	// their zip entry names, so they can be appended to the package in a fixed
	// order after all singleton parts.
	type slidePart struct {
		xmlName, relsName string
		xml, rels         []byte
	}

	slides := make([]slideRef, 0, n)
	renderedSlides := make([]slidePart, 0, n)
	// notesSlideParts holds the notesSlideN.xml + rels for every section
	// that HAS notes, appended in the SAME order sections are visited below
	// (i.e. slide order) -- the fixed, deterministic order the anti_patterns
	// section requires for conditional notes emission.
	notesSlideParts := make([]slidePart, 0, n)
	// notesOverrides holds the per-notesSlide content-type Override, in the
	// same slide order, appended to overrides AFTER every ctSlide Override
	// (deterministic, never map-ranged).
	notesOverrides := make([]contentTypeOverride, 0, n)
	hasNotes := false

	// overrides begins with every fixed non-.rels part's content-type Override,
	// then gains one ctSlide Override per slide below -- the third of the
	// per-slide 3-fold bookkeeping (sldIdLst + presentation rels + this).
	overrides := []contentTypeOverride{
		{PartName: "/docProps/core.xml", ContentType: ctCoreProps},
		{PartName: "/docProps/app.xml", ContentType: ctExtendedProps},
		{PartName: "/ppt/presentation.xml", ContentType: ctPresentation},
		{PartName: "/ppt/presProps.xml", ContentType: ctPresProps},
		{PartName: "/ppt/viewProps.xml", ContentType: ctViewProps},
		{PartName: "/ppt/tableStyles.xml", ContentType: ctTableStyles},
		{PartName: "/ppt/theme/theme1.xml", ContentType: ctTheme},
		{PartName: "/ppt/slideMasters/slideMaster1.xml", ContentType: ctSlideMaster},
		{PartName: "/ppt/slideLayouts/slideLayout1.xml", ContentType: ctSlideLayout},
	}

	for i, section := range doc.Sections {
		num := i + 1
		slideXML, relsXML := buildSlide(section, doc.Outline, size)

		xmlName := fmt.Sprintf("ppt/slides/slide%d.xml", num)
		relsName := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", num)
		relID := fmt.Sprintf("rId%d", firstSlideRelNum-1+num)

		// (1) sldIdLst + (2) presentation.xml.rels entry (both via slideRef).
		slides = append(slides, slideRef{RelID: relID, Target: fmt.Sprintf("slides/slide%d.xml", num)})
		// (3) content-types Override for the slide part.
		overrides = append(overrides, contentTypeOverride{PartName: "/" + xmlName, ContentType: ctSlide})

		// Notes are CONDITIONAL: only a section with Notes gains a
		// notesSlideN.xml + the extra slide-rels entry pointing at it
		// (anti_patterns: never emit a notes part for a notes-free section).
		if len(section.Notes) > 0 {
			hasNotes = true
			relsXML = buildSlideRelsWithNotes(num)

			notesXMLName := fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", num)
			notesRelsName := fmt.Sprintf("ppt/notesSlides/_rels/notesSlide%d.xml.rels", num)
			notesSlideParts = append(notesSlideParts, slidePart{
				xmlName: notesXMLName, relsName: notesRelsName,
				xml: buildNotesSlide(section.Notes), rels: buildNotesSlideRels(num),
			})
			notesOverrides = append(notesOverrides, contentTypeOverride{PartName: "/" + notesXMLName, ContentType: ctNotesSlide})
		}

		renderedSlides = append(renderedSlides, slidePart{
			xmlName: xmlName, relsName: relsName, xml: slideXML, rels: relsXML,
		})
	}

	// notesMaster1 is emitted at most ONCE, iff any section has notes, with
	// its own content-type Override and its presentation.xml.rels entry at
	// the first unused rId (one past the last slide's rId6..rId(5+n)).
	var notesMasterRelID string
	if hasNotes {
		notesMasterRelID = fmt.Sprintf("rId%d", firstSlideRelNum+n)
		overrides = append(overrides, contentTypeOverride{PartName: "/ppt/notesMasters/notesMaster1.xml", ContentType: ctNotesMaster})
	}
	overrides = append(overrides, notesOverrides...)

	presRels := presentationRelsXML(slides)
	if hasNotes {
		presRels = presentationRelsXML(slides, relationship{ID: notesMasterRelID, Type: relTypeNotesMaster, Target: "notesMasters/notesMaster1.xml"})
	}

	// Assemble the full part graph in a FIXED order (determinism, 06-RESEARCH
	// Pitfall 4): the singleton static parts first, then (iff any notes exist)
	// the once-per-deck notesMaster, then each slide's content + .rels in
	// slide order, then each notes slide's content + .rels in slide order.
	parts := []part{
		{name: "[Content_Types].xml", content: buildContentTypesXML(overrides)},
		{name: "_rels/.rels", content: rootRelsXML()},
		{name: "docProps/core.xml", content: docPropsCoreXML()},
		{name: "docProps/app.xml", content: docPropsAppXML(n)},
		{name: "ppt/presentation.xml", content: presentationXML(size, slides)},
		{name: "ppt/_rels/presentation.xml.rels", content: presRels},
		{name: "ppt/presProps.xml", content: presPropsXML()},
		{name: "ppt/viewProps.xml", content: viewPropsXML()},
		{name: "ppt/tableStyles.xml", content: tableStylesXML()},
		{name: "ppt/theme/theme1.xml", content: theme1XML()},
		{name: "ppt/slideMasters/slideMaster1.xml", content: slideMaster1XML()},
		{name: "ppt/slideMasters/_rels/slideMaster1.xml.rels", content: slideMaster1RelsXML()},
		{name: "ppt/slideLayouts/slideLayout1.xml", content: slideLayout1XML()},
		{name: "ppt/slideLayouts/_rels/slideLayout1.xml.rels", content: slideLayout1RelsXML()},
	}
	if hasNotes {
		parts = append(parts,
			part{name: "ppt/notesMasters/notesMaster1.xml", content: buildNotesMaster()},
			part{name: "ppt/notesMasters/_rels/notesMaster1.xml.rels", content: buildNotesMasterRels()},
		)
	}
	for _, s := range renderedSlides {
		parts = append(parts,
			part{name: s.xmlName, content: s.xml},
			part{name: s.relsName, content: s.rels},
		)
	}
	for _, s := range notesSlideParts {
		parts = append(parts,
			part{name: s.xmlName, content: s.xml},
			part{name: s.relsName, content: s.rels},
		)
	}

	return buildZip(parts)
}
