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

package docx

import "fmt"

// xmlDeclaration is the fixed prolog every OOXML part in this package is
// written with, matching what Word-authored parts use (standalone="yes",
// which encoding/xml's own xml.Header omits).
const xmlDeclaration = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

// The WordprocessingML + OPC namespaces this package emits.
const (
	nsWordprocessing = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsRelationships  = "http://schemas.openxmlformats.org/package/2006/relationships"
	nsDocRels        = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsContentTypes   = "http://schemas.openxmlformats.org/package/2006/content-types"
	nsCoreProps      = "http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
	nsExtendedProps  = "http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"
	nsDublinCore     = "http://purl.org/dc/elements/1.1/"
	nsXSI            = "http://www.w3.org/2001/XMLSchema-instance"
)

// OOXML content types for the parts in the minimal Wordprocessing graph.
const (
	ctDocument      = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"
	ctStyles        = "application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"
	ctNumbering     = "application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"
	ctCorePropsType = "application/vnd.openxmlformats-package.core-properties+xml"
	ctExtendedType  = "application/vnd.openxmlformats-officedocument.extended-properties+xml"
)

// Relationship type URIs.
const (
	relTypeOfficeDocument = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
	relTypeCoreProps      = "http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties"
	relTypeExtendedProps  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties"
	relTypeStyles         = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles"
	relTypeNumbering      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering"
)

// contentTypeOverride pairs a part name (WITH its leading slash) with its
// OOXML content type, emitted in the exact order supplied (never re-sorted --
// determinism).
type contentTypeOverride struct {
	PartName    string
	ContentType string
}

// buildContentTypesXML renders [Content_Types].xml: fixed Default entries for
// the "rels" and "xml" extensions, then one Override per part.
func buildContentTypesXML(overrides []contentTypeOverride) []byte {
	var b []byte
	b = append(b, xmlDeclaration...)
	b = append(b, fmt.Sprintf(`<Types xmlns="%s">`, nsContentTypes)...)
	b = append(b, `<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`...)
	b = append(b, `<Default Extension="xml" ContentType="application/xml"/>`...)
	for _, o := range overrides {
		b = append(b, fmt.Sprintf(`<Override PartName="%s" ContentType="%s"/>`, o.PartName, o.ContentType)...)
	}
	b = append(b, `</Types>`...)
	return b
}

// relationship is one <Relationship> entry in a .rels part.
type relationship struct {
	ID     string
	Type   string
	Target string
}

// buildRelsXML renders an OPC .rels part from rels, in the given order.
func buildRelsXML(rels []relationship) []byte {
	var b []byte
	b = append(b, xmlDeclaration...)
	b = append(b, fmt.Sprintf(`<Relationships xmlns="%s">`, nsRelationships)...)
	for _, r := range rels {
		b = append(b, fmt.Sprintf(`<Relationship Id="%s" Type="%s" Target="%s"/>`, r.ID, r.Type, r.Target)...)
	}
	b = append(b, `</Relationships>`...)
	return b
}

// rootRelsXML is /_rels/.rels: the package entry point. The officeDocument
// relationship is the one Word follows first -- without it the package is
// reported as corrupt even when every part is present and valid.
func rootRelsXML() []byte {
	return buildRelsXML([]relationship{
		{ID: "rId1", Type: relTypeOfficeDocument, Target: "word/document.xml"},
		{ID: "rId2", Type: relTypeCoreProps, Target: "docProps/core.xml"},
		{ID: "rId3", Type: relTypeExtendedProps, Target: "docProps/app.xml"},
	})
}

// documentRelsXML is /word/_rels/document.xml.rels.
func documentRelsXML() []byte {
	return buildRelsXML([]relationship{
		{ID: "rId1", Type: relTypeStyles, Target: "styles.xml"},
		{ID: "rId2", Type: relTypeNumbering, Target: "numbering.xml"},
	})
}

// docPropsCoreXML carries no timestamps: dcterms:created/modified would
// otherwise make every export of the same document differ, defeating the
// determinism guarantee.
func docPropsCoreXML(title string) []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<cp:coreProperties xmlns:cp="%s" xmlns:dc="%s" xmlns:xsi="%s">`+
			`<dc:title>%s</dc:title><dc:creator>Eden Press</dc:creator>`+
			`<cp:lastModifiedBy>Eden Press</cp:lastModifiedBy></cp:coreProperties>`,
		nsCoreProps, nsDublinCore, nsXSI, escapeXML(title)))
}

func docPropsAppXML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(
		`<Properties xmlns="%s"><Application>Eden Press</Application></Properties>`,
		nsExtendedProps))
}

// stylesXML defines the paragraph styles body.go references by name. Word has
// built-in semantics for Heading1..6, Quote and ListParagraph (navigation
// pane, TOC fields, quote formatting); "Code" is Eden Press's own, since Word
// has no built-in preformatted style.
func stylesXML() []byte {
	var b []byte
	b = append(b, xmlDeclaration...)
	b = append(b, fmt.Sprintf(`<w:styles xmlns:w="%s">`, nsWordprocessing)...)

	// Document defaults: 11pt body with sensible paragraph spacing.
	b = append(b, `<w:docDefaults><w:rPrDefault><w:rPr>`+
		`<w:rFonts w:ascii="Calibri" w:hAnsi="Calibri"/><w:sz w:val="22"/>`+
		`</w:rPr></w:rPrDefault><w:pPrDefault><w:pPr>`+
		`<w:spacing w:after="160" w:line="259" w:lineRule="auto"/>`+
		`</w:pPr></w:pPrDefault></w:docDefaults>`...)

	b = append(b, `<w:style w:type="paragraph" w:default="1" w:styleId="Normal">`+
		`<w:name w:val="Normal"/><w:qFormat/></w:style>`...)

	// Heading1..6. Sizes step down 32 -> 22 half-points; keepNext stops a
	// heading from being orphaned at the foot of a page.
	sizes := []int{32, 28, 26, 24, 22, 22}
	for i, sz := range sizes {
		lvl := i + 1
		b = append(b, fmt.Sprintf(
			`<w:style w:type="paragraph" w:styleId="Heading%d">`+
				`<w:name w:val="heading %d"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>`+
				`<w:pPr><w:keepNext/><w:outlineLvl w:val="%d"/>`+
				`<w:spacing w:before="240" w:after="120"/></w:pPr>`+
				`<w:rPr><w:b/><w:sz w:val="%d"/></w:rPr></w:style>`,
			lvl, lvl, i, sz)...)
	}

	b = append(b, `<w:style w:type="paragraph" w:styleId="Quote">`+
		`<w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>`+
		`<w:pPr><w:ind w:left="720"/><w:spacing w:before="120" w:after="120"/></w:pPr>`+
		`<w:rPr><w:i/><w:color w:val="404040"/></w:rPr></w:style>`...)

	b = append(b, `<w:style w:type="paragraph" w:styleId="Code">`+
		`<w:name w:val="HTML Preformatted"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>`+
		`<w:pPr><w:shd w:val="clear" w:fill="F5F5F5"/><w:ind w:left="360"/>`+
		`<w:spacing w:before="120" w:after="120" w:line="240" w:lineRule="auto"/></w:pPr>`+
		`<w:rPr><w:rFonts w:ascii="Consolas" w:hAnsi="Consolas"/><w:sz w:val="20"/></w:rPr></w:style>`...)

	b = append(b, `<w:style w:type="paragraph" w:styleId="ListParagraph">`+
		`<w:name w:val="List Paragraph"/><w:basedOn w:val="Normal"/><w:qFormat/>`+
		`<w:pPr><w:ind w:left="720"/><w:contextualSpacing/></w:pPr></w:style>`...)

	b = append(b, `<w:style w:type="paragraph" w:styleId="Caption">`+
		`<w:name w:val="caption"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/>`+
		`<w:rPr><w:i/><w:sz w:val="18"/><w:color w:val="595959"/></w:rPr></w:style>`...)

	b = append(b, `<w:style w:type="table" w:styleId="TableGrid">`+
		`<w:name w:val="Table Grid"/><w:tblPr><w:tblBorders>`+
		`<w:top w:val="single" w:sz="4" w:color="BFBFBF"/>`+
		`<w:left w:val="single" w:sz="4" w:color="BFBFBF"/>`+
		`<w:bottom w:val="single" w:sz="4" w:color="BFBFBF"/>`+
		`<w:right w:val="single" w:sz="4" w:color="BFBFBF"/>`+
		`<w:insideH w:val="single" w:sz="4" w:color="BFBFBF"/>`+
		`<w:insideV w:val="single" w:sz="4" w:color="BFBFBF"/>`+
		`</w:tblBorders></w:tblPr></w:style>`...)

	b = append(b, `</w:styles>`...)
	return b
}

// numberingXML defines exactly two numbering instances body.go references:
// numId 1 (bullet) and numId 2 (decimal), each with 9 indent levels so a
// nested ListItem.Level maps straight onto w:ilvl.
func numberingXML() []byte {
	var b []byte
	b = append(b, xmlDeclaration...)
	b = append(b, fmt.Sprintf(`<w:numbering xmlns:w="%s">`, nsWordprocessing)...)

	// Bullet glyphs cycle by depth the way Word's own default list does. Plain
	// Unicode characters are used rather than Symbol-font code points, so the
	// glyph renders identically without a font-substitution dependency.
	bulletGlyphs := []string{"\u2022", "\u25e6", "\u25aa"}

	for _, def := range []struct {
		abstractID int
		format     string
	}{
		{0, "bullet"},
		{1, "decimal"},
	} {
		b = append(b, fmt.Sprintf(`<w:abstractNum w:abstractNumId="%d">`, def.abstractID)...)
		for lvl := 0; lvl < 9; lvl++ {
			// A bullet level shows a literal glyph; a decimal level shows the
			// "%N." placeholder Word substitutes with that level's counter.
			lvlText := bulletGlyphs[lvl%len(bulletGlyphs)]
			if def.format == "decimal" {
				lvlText = fmt.Sprintf("%%%d.", lvl+1)
			}
			b = append(b, fmt.Sprintf(
				`<w:lvl w:ilvl="%d"><w:start w:val="1"/><w:numFmt w:val="%s"/>`+
					`<w:lvlText w:val="%s"/><w:lvlJc w:val="left"/>`+
					`<w:pPr><w:ind w:left="%d" w:hanging="360"/></w:pPr></w:lvl>`,
				lvl, def.format, escapeXML(lvlText), 720+lvl*360)...)
		}
		b = append(b, `</w:abstractNum>`...)
	}

	// numId 1 -> bullets, numId 2 -> decimal. body.go depends on these values.
	b = append(b, `<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`...)
	b = append(b, `<w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num>`...)
	b = append(b, `</w:numbering>`...)
	return b
}
