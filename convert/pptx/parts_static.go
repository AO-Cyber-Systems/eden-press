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

// parts_static.go builds every INVARIANT part in the minimal OPC part graph
// (06-RESEARCH Pattern 1): package/presentation relationships, docProps,
// presProps/viewProps/tableStyles, the theme, and the slide master/layout
// pair -- everything except the per-slide content itself (which 06-04 makes
// model-driven; this TRD's own trivial slide1.xml lives in openable_test.go,
// deliberately test-only scaffolding).
package pptx

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// OPC/PresentationML XML namespace URIs used throughout the static part
// graph. Declared once here so every builder in this file uses
// byte-identical namespace strings.
const (
	nsDrawingML      = "http://schemas.openxmlformats.org/drawingml/2006/main"
	nsRelationships  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsPresentationML = "http://schemas.openxmlformats.org/presentationml/2006/main"
	nsPackageRels    = "http://schemas.openxmlformats.org/package/2006/relationships"
	nsCoreProps      = "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
	nsDublinCore     = "http://purl.org/dc/elements/1.1/"
	nsDublinCoreTerm = "http://purl.org/dc/terms/"
	nsXSI            = "http://www.w3.org/2001/XMLSchema-instance"
	nsExtendedProps  = "http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
)

// Relationship TYPE URIs (the Type attribute of a <Relationship> element),
// one per distinct relationship this minimal part graph declares.
const (
	relTypeOfficeDocument = nsRelationships + "/officeDocument"
	relTypeCoreProperties = "http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties"
	relTypeExtendedProps  = nsRelationships + "/extended-properties"
	relTypeSlideMaster    = nsRelationships + "/slideMaster"
	relTypeSlide          = nsRelationships + "/slide"
	relTypeTheme          = nsRelationships + "/theme"
	relTypePresProps      = nsRelationships + "/presProps"
	relTypeViewProps      = nsRelationships + "/viewProps"
	relTypeTableStyles    = nsRelationships + "/tableStyles"
	relTypeSlideLayout    = nsRelationships + "/slideLayout"
)

// Content-Type Override strings (the ContentType attribute for each
// distinct, non-boilerplate-extension part in this minimal graph).
const (
	ctPresentation  = "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"
	ctSlideMaster   = "application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"
	ctSlideLayout   = "application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"
	ctSlide         = "application/vnd.openxmlformats-officedocument.presentationml.slide+xml"
	ctTheme         = "application/vnd.openxmlformats-officedocument.theme+xml"
	ctCoreProps     = "application/vnd.openxmlformats-package.core-properties+xml"
	ctExtendedProps = "application/vnd.openxmlformats-officedocument.extended-properties+xml"
	ctPresProps     = "application/vnd.openxmlformats-officedocument.presentationml.presProps+xml"
	ctViewProps     = "application/vnd.openxmlformats-officedocument.presentationml.viewProps+xml"
	ctTableStyles   = "application/vnd.openxmlformats-officedocument.presentationml.tableStyles+xml"
)

// Fixed r:id values this minimal part graph's parts and their sibling .rels
// files agree on, declared once as named constants (rather than repeating
// literal strings at every call site) so a part and its .rels can never
// drift out of sync -- the exact "#1 unresolved r:id bug" this TRD warns
// against. Each .rels file has its own independent Id namespace, so "rId1"
// is legitimately reused across different .rels files below.
const (
	rIDMaster1     = "rId1" // presentation.xml(.rels): -> slideMaster1.xml
	rIDTheme1      = "rId2" // presentation.xml.rels:   -> theme1.xml
	rIDPresProps   = "rId3" // presentation.xml.rels:   -> presProps.xml
	rIDViewProps   = "rId4" // presentation.xml.rels:   -> viewProps.xml
	rIDTableStyles = "rId5" // presentation.xml.rels:   -> tableStyles.xml
	rIDSlide1      = "rId6" // presentation.xml(.rels): -> slides/slide1.xml

	rIDMasterLayout1 = "rId1" // slideMaster1.xml(.rels): -> slideLayout1.xml
	rIDMasterTheme1  = "rId2" // slideMaster1.xml.rels:   -> theme1.xml (not referenced by an explicit r:id in the master's own content)

	rIDLayoutMaster1 = "rId1" // slideLayout1.xml.rels: -> slideMaster1.xml

	rIDSlideLayout1 = "rId1" // slide1.xml.rels: -> slideLayout1.xml
)

// relationship is one row of an OPC .rels part: Id (referenced by an r:id
// attribute in the owning part's own content), Type (a relationship-type
// URI), and Target (a path relative to the OWNING PART's directory, per the
// OPC convention).
type relationship struct {
	ID     string
	Type   string
	Target string
}

type relsRelationshipXML struct {
	XMLName xml.Name `xml:"Relationship"`
	ID      string   `xml:"Id,attr"`
	Type    string   `xml:"Type,attr"`
	Target  string   `xml:"Target,attr"`
}

type relsDocXML struct {
	XMLName       xml.Name              `xml:"Relationships"`
	Xmlns         string                `xml:"xmlns,attr"`
	Relationships []relsRelationshipXML `xml:"Relationship"`
}

// buildRelsXML renders a PARTNAME.rels document from rels, in the EXACT
// order given (determinism, Pitfall 4).
func buildRelsXML(rels []relationship) []byte {
	doc := relsDocXML{Xmlns: nsPackageRels}
	for _, r := range rels {
		doc.Relationships = append(doc.Relationships, relsRelationshipXML{ID: r.ID, Type: r.Type, Target: r.Target})
	}
	return marshalPart(doc)
}

// rootRelsXML builds "_rels/.rels": the package-level relationships
// pointing at the presentation's officeDocument part and docProps.
func rootRelsXML() []byte {
	return buildRelsXML([]relationship{
		{ID: "rId1", Type: relTypeOfficeDocument, Target: "ppt/presentation.xml"},
		{ID: "rId2", Type: relTypeCoreProperties, Target: "docProps/core.xml"},
		{ID: "rId3", Type: relTypeExtendedProps, Target: "docProps/app.xml"},
	})
}

// docPropsCoreXML builds "docProps/core.xml" (Dublin-Core metadata).
func docPropsCoreXML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<cp:coreProperties xmlns:cp="%s" xmlns:dc="%s" xmlns:dcterms="%s" xmlns:xsi="%s">`+
			`<dc:title>Eden Press Presentation</dc:title>`+
			`<dc:creator>Eden Press</dc:creator>`+
			`</cp:coreProperties>`,
		nsCoreProps, nsDublinCore, nsDublinCoreTerm, nsXSI))
}

// docPropsAppXML builds "docProps/app.xml" (PowerPoint extended
// properties), parametrized by the deck's slide count.
func docPropsAppXML(slideCount int) []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<Properties xmlns="%s" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">`+
			`<Application>Eden Press</Application>`+
			`<Slides>%d</Slides>`+
			`</Properties>`,
		nsExtendedProps, slideCount))
}

// presPropsXML builds "ppt/presProps.xml" -- a minimal, valid presentation
// properties part. Included per the "boring parts" anti-pattern: PowerPoint
// silently repairs its absence, LibreOffice/stricter consumers may not.
func presPropsXML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<p:presentationPr xmlns:a="%s" xmlns:r="%s" xmlns:p="%s"/>`,
		nsDrawingML, nsRelationships, nsPresentationML))
}

// viewPropsXML builds "ppt/viewProps.xml" -- minimal, valid editor view
// properties.
func viewPropsXML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<p:viewPr xmlns:a="%s" xmlns:r="%s" xmlns:p="%s">`+
			`<p:normalViewPr><p:restoredLeft sz="15620"/><p:restoredTop sz="94660"/></p:normalViewPr>`+
			`<p:slideViewPr><p:cSldViewPr><p:cViewPr varScale="1"><p:scale><a:sx n="64" d="100"/><a:sy n="64" d="100"/></p:scale><p:origin x="0" y="0"/></p:cViewPr><p:guideLst/></p:cSldViewPr></p:slideViewPr>`+
			`</p:viewPr>`,
		nsDrawingML, nsRelationships, nsPresentationML))
}

// tableStylesXML builds "ppt/tableStyles.xml" -- an empty, valid table
// style list.
func tableStylesXML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<a:tblStyleLst xmlns:a="%s" def="{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}"/>`,
		nsDrawingML))
}

// theme1XML builds "ppt/theme/theme1.xml": the full 12-color clrScheme,
// major/minor fontScheme, and an EXACTLY-3-entries-per-list fmtScheme
// (Pitfall 3) -- fully static boilerplate, hand-copied once per 06-RESEARCH.
func theme1XML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(`<a:theme xmlns:a="%s" name="Eden Press Theme">
  <a:themeElements>
    <a:clrScheme name="Eden Press">
      <a:dk1><a:sysClr val="windowText" lastClr="000000"/></a:dk1>
      <a:lt1><a:sysClr val="window" lastClr="FFFFFF"/></a:lt1>
      <a:dk2><a:srgbClr val="1F1F1F"/></a:dk2>
      <a:lt2><a:srgbClr val="EEECE1"/></a:lt2>
      <a:accent1><a:srgbClr val="4472C4"/></a:accent1>
      <a:accent2><a:srgbClr val="ED7D31"/></a:accent2>
      <a:accent3><a:srgbClr val="A5A5A5"/></a:accent3>
      <a:accent4><a:srgbClr val="FFC000"/></a:accent4>
      <a:accent5><a:srgbClr val="5B9BD5"/></a:accent5>
      <a:accent6><a:srgbClr val="70AD47"/></a:accent6>
      <a:hlink><a:srgbClr val="0563C1"/></a:hlink>
      <a:folHlink><a:srgbClr val="954F72"/></a:folHlink>
    </a:clrScheme>
    <a:fontScheme name="Eden Press">
      <a:majorFont><a:latin typeface="Calibri Light"/><a:ea typeface=""/><a:cs typeface=""/></a:majorFont>
      <a:minorFont><a:latin typeface="Calibri"/><a:ea typeface=""/><a:cs typeface=""/></a:minorFont>
    </a:fontScheme>
    <a:fmtScheme name="Eden Press">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="6350"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
        <a:ln w="12700"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
        <a:ln w="19050"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>`, nsDrawingML))
}

// clrMapAttrs is the mandatory, canonical-identity 12-attribute <p:clrMap>
// (Pitfall 2) emitted verbatim -- there is no v1 reason for a hand-rolled
// writer to vary this.
const clrMapAttrs = `bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlk"`

// slideMaster1XML builds "ppt/slideMasters/slideMaster1.xml": an empty
// shape tree, the mandatory full clrMap, and a single-layout sldLayoutIdLst
// referencing slideLayout1.xml by its fixed r:id.
func slideMaster1XML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(`<p:sldMaster xmlns:a="%s" xmlns:r="%s" xmlns:p="%s">
  <p:cSld>
    <p:bg><p:bgRef idx="1001"><a:schemeClr val="bg1"/></p:bgRef></p:bg>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>
    </p:spTree>
  </p:cSld>
  <p:clrMap %s/>
  <p:sldLayoutIdLst><p:sldLayoutId id="2147483649" r:id="%s"/></p:sldLayoutIdLst>
</p:sldMaster>`, nsDrawingML, nsRelationships, nsPresentationML, clrMapAttrs, rIDMasterLayout1))
}

// slideMaster1RelsXML builds "ppt/slideMasters/_rels/slideMaster1.xml.rels":
// -> slideLayout1.xml (by rIDMasterLayout1, referenced from the master's own
// content) and -> theme1.xml (a relationship-only association; themes are
// never referenced by an explicit r:id inside a master/layout/slide's XML).
func slideMaster1RelsXML() []byte {
	return buildRelsXML([]relationship{
		{ID: rIDMasterLayout1, Type: relTypeSlideLayout, Target: "../slideLayouts/slideLayout1.xml"},
		{ID: rIDMasterTheme1, Type: relTypeTheme, Target: "../theme/theme1.xml"},
	})
}

// slideLayout1XML builds "ppt/slideLayouts/slideLayout1.xml": a minimal,
// empty-shape-tree "title" layout.
func slideLayout1XML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(`<p:sldLayout xmlns:a="%s" xmlns:r="%s" xmlns:p="%s" type="title" preserve="1">
  <p:cSld name="Title">
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>
    </p:spTree>
  </p:cSld>
</p:sldLayout>`, nsDrawingML, nsRelationships, nsPresentationML))
}

// slideLayout1RelsXML builds
// "ppt/slideLayouts/_rels/slideLayout1.xml.rels": -> slideMaster1.xml (a
// relationship-only association; layouts do not reference their master by
// an explicit r:id inside their own content).
func slideLayout1RelsXML() []byte {
	return buildRelsXML([]relationship{
		{ID: rIDLayoutMaster1, Type: relTypeSlideMaster, Target: "../slideMasters/slideMaster1.xml"},
	})
}

// slideRef pairs a presentation-level r:id with the slide part name it
// targets (relative to ppt/, e.g. "slides/slide1.xml") -- one entry per
// slide, in presentation order. 06-04 extends this to N slides by
// lengthening the slice passed to presentationXML/presentationRelsXML; this
// TRD's plumbing itself is otherwise unchanged.
type slideRef struct {
	RelID  string
	Target string
}

// presentationXML builds "ppt/presentation.xml": the master/slide id lists,
// and the requested slide size's <p:sldSz> (driven by 06-02's SlideSize
// constants) + the fixed <p:notesSz>.
func presentationXML(size SlideSize, slides []slideRef) []byte {
	var sldIDs strings.Builder
	for i, s := range slides {
		fmt.Fprintf(&sldIDs, `<p:sldId id="%d" r:id="%s"/>`, 256+i, s.RelID)
	}
	sldSzType := ""
	if size.Type != "" {
		sldSzType = fmt.Sprintf(` type="%s"`, size.Type)
	}
	return []byte(xmlDeclaration + fmt.Sprintf(`<p:presentation xmlns:a="%s" xmlns:r="%s" xmlns:p="%s">
  <p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="%s"/></p:sldMasterIdLst>
  <p:sldIdLst>%s</p:sldIdLst>
  <p:sldSz cx="%d" cy="%d"%s/>
  <p:notesSz cx="%d" cy="%d"/>
</p:presentation>`, nsDrawingML, nsRelationships, nsPresentationML, rIDMaster1, sldIDs.String(), size.CX, size.CY, sldSzType, NotesSize.CX, NotesSize.CY))
}

// presentationRelsXML builds "ppt/_rels/presentation.xml.rels": the fixed
// singleton relationships (master, theme, presProps, viewProps,
// tableStyles), one relationship per slide in slides (in order), then any
// caller-supplied extra relationships appended last (in order) -- 06-05
// uses this trailing extra slot to add the notesMaster1 relationship
// exactly once, without duplicating this fixed-relationship list. extra is
// variadic so every pre-existing call site (0 extra args) is unaffected.
func presentationRelsXML(slides []slideRef, extra ...relationship) []byte {
	rels := []relationship{
		{ID: rIDMaster1, Type: relTypeSlideMaster, Target: "slideMasters/slideMaster1.xml"},
		{ID: rIDTheme1, Type: relTypeTheme, Target: "theme/theme1.xml"},
		{ID: rIDPresProps, Type: relTypePresProps, Target: "presProps.xml"},
		{ID: rIDViewProps, Type: relTypeViewProps, Target: "viewProps.xml"},
		{ID: rIDTableStyles, Type: relTypeTableStyles, Target: "tableStyles.xml"},
	}
	for _, s := range slides {
		rels = append(rels, relationship{ID: s.RelID, Type: relTypeSlide, Target: s.Target})
	}
	rels = append(rels, extra...)
	return buildRelsXML(rels)
}
