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

// notes.go maps chase/model.Section.Notes -> the speaker-notes OPC parts
// (06-RESEARCH Pattern 1 "notes slide" + Pattern 4 notes-slide shape):
// ppt/notesSlides/notesSlideN.xml (one <a:p> per note string, in the
// mandatory <p:ph type="body"> placeholder shape) plus the once-per-deck
// ppt/notesMasters/notesMaster1.xml, and the .rels wiring that closes the
// slide -> notesSlide -> notesMaster -> theme relationship graph
// (06-RESEARCH Code Examples / Pattern 1). Notes parts are CONDITIONAL:
// buildNotesSlide is emitted only for a Section that HAS notes; notesMaster1
// appears at most once, iff ANY section in the deck has notes. This file
// extends shapes.go's escaping helper and parts_static.go's rels/theme
// plumbing rather than duplicating them (06-04 key-link).
package pptx

import (
	"fmt"
	"strings"
)

// Content-Type Override strings for the two notes-related part kinds,
// alongside parts_static.go's ct* constants.
const (
	ctNotesSlide  = "application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"
	ctNotesMaster = "application/vnd.openxmlformats-officedocument.presentationml.notesMaster+xml"
)

// Relationship TYPE URIs the notes part graph adds beyond parts_static.go's
// relType* set.
const (
	relTypeNotesSlide  = nsRelationships + "/notesSlide"
	relTypeNotesMaster = nsRelationships + "/notesMaster"
)

// Fixed r:id values the notes part graph's OWN .rels files use. Each .rels
// file has its own independent Id namespace (parts_static.go's rIDMaster1
// convention), so these literals are safely reused across every
// notesSlideN.xml.rels / slideN.xml.rels / notesMaster1.xml.rels instance.
const (
	rIDNotesSlideOwner  = "rId1" // notesSlideN.xml.rels -> ../slides/slideN.xml
	rIDNotesSlideMaster = "rId2" // notesSlideN.xml.rels -> ../notesMasters/notesMaster1.xml

	rIDSlideLayoutForNotes = rIDSlideLayout1 // slideN.xml.rels -> ../slideLayouts/slideLayout1.xml (unchanged)
	rIDSlideNotesSlide     = "rId2"          // slideN.xml.rels -> ../notesSlides/notesSlideN.xml (added only when the slide has notes)

	rIDNotesMasterTheme = "rId1" // notesMaster1.xml.rels -> ../theme/theme1.xml
)

// buildNotesSlide renders one ppt/notesSlides/notesSlideN.xml document: the
// mandatory spTree group header (matching every other spTree in this
// package) plus the <p:notes> body-placeholder shape (06-RESEARCH Code
// Examples) with one <a:p> per notes[i] (escaped via shapes.go's
// escapeXML). Returns nil when notes is empty -- notes parts are
// CONDITIONAL, never emitted for a notes-free section (anti-pattern: no
// dangling empty notes part with a would-be-orphan rel).
func buildNotesSlide(notes []string) []byte {
	if len(notes) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString(xmlDeclaration)
	fmt.Fprintf(&sb, `<p:notes xmlns:a="%s" xmlns:r="%s" xmlns:p="%s">`, nsDrawingML, nsRelationships, nsPresentationML)
	sb.WriteString(`<p:cSld><p:spTree>`)
	sb.WriteString(`<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>`)
	sb.WriteString(`<p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>`)
	sb.WriteString(`<p:sp><p:nvSpPr><p:cNvPr id="2" name="Notes Placeholder"/><p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>`)
	sb.WriteString(`<p:nvPr><p:ph type="body" idx="1"/></p:nvPr></p:nvSpPr>`)
	sb.WriteString(`<p:spPr/>`)
	sb.WriteString(`<p:txBody><a:bodyPr/><a:lstStyle/>`)
	for _, note := range notes {
		fmt.Fprintf(&sb, `<a:p><a:r><a:t>%s</a:t></a:r></a:p>`, escapeXML(note))
	}
	sb.WriteString(`</p:txBody></p:sp>`)
	sb.WriteString(`</p:spTree></p:cSld></p:notes>`)
	return []byte(sb.String())
}

// buildNotesSlideRels builds "ppt/notesSlides/_rels/notesSlideN.xml.rels":
// -> ../slides/slideN.xml (the owning slide) and ->
// ../notesMasters/notesMaster1.xml -- the notesSlide side of the 4-way
// notes rels chain (06-RESEARCH Pattern 1).
func buildNotesSlideRels(slideNum int) []byte {
	return buildRelsXML([]relationship{
		{ID: rIDNotesSlideOwner, Type: relTypeSlide, Target: fmt.Sprintf("../slides/slide%d.xml", slideNum)},
		{ID: rIDNotesSlideMaster, Type: relTypeNotesMaster, Target: "../notesMasters/notesMaster1.xml"},
	})
}

// buildSlideRelsWithNotes builds "ppt/slides/_rels/slideN.xml.rels" for a
// slide whose section HAS notes: the usual -> slideLayout1.xml relationship
// (identical to slide.go's buildSlide) PLUS -> ../notesSlides/notesSlideN.xml
// -- the slide-side half of the notes wiring. Slides with no notes keep
// buildSlide's own unmodified rels (layout only); this builder is only
// invoked by ToPPTX when section.Notes is non-empty.
func buildSlideRelsWithNotes(slideNum int) []byte {
	return buildRelsXML([]relationship{
		{ID: rIDSlideLayoutForNotes, Type: relTypeSlideLayout, Target: "../slideLayouts/slideLayout1.xml"},
		{ID: rIDSlideNotesSlide, Type: relTypeNotesSlide, Target: fmt.Sprintf("../notesSlides/notesSlide%d.xml", slideNum)},
	})
}

// buildNotesMaster builds "ppt/notesMasters/notesMaster1.xml": the
// once-per-deck static notes master -- an empty shape tree plus the
// mandatory full 12-attribute clrMap (parts_static.go's clrMapAttrs,
// Pitfall 2), following slideMaster1XML's exact pattern.
func buildNotesMaster() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(`<p:notesMaster xmlns:a="%s" xmlns:r="%s" xmlns:p="%s">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMap %s/>
</p:notesMaster>`, nsDrawingML, nsRelationships, nsPresentationML, clrMapAttrs))
}

// buildNotesMasterRels builds "ppt/notesMasters/_rels/notesMaster1.xml.rels":
// -> ../theme/theme1.xml -- a relationship-only association (the master's
// own content carries no explicit r:id reference to it, exactly like
// slideMaster1RelsXML's theme relationship).
func buildNotesMasterRels() []byte {
	return buildRelsXML([]relationship{
		{ID: rIDNotesMasterTheme, Type: relTypeTheme, Target: "../theme/theme1.xml"},
	})
}
