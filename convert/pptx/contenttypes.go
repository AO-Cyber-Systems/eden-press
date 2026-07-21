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

package pptx

import (
	"bytes"
	"encoding/xml"
	"path"
	"strings"
)

// xmlDeclaration is the fixed XML prolog every OOXML/OPC part in this
// package is written with. Kept as a literal string (rather than
// encoding/xml's own xml.Header, which omits standalone="yes") to match the
// exact declaration real PowerPoint/LibreOffice-authored parts use.
const xmlDeclaration = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

// marshalPart encodes v (an XML-tagged struct) with the fixed xmlDeclaration
// prepended -- the single shared XML-part serializer every struct-based
// static-part builder in this package funnels through.
func marshalPart(v any) []byte {
	var buf bytes.Buffer
	buf.WriteString(xmlDeclaration)
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		// v is always one of this package's own static, always-marshalable
		// XML structs -- a failure here is a programmer error, not a
		// reachable runtime condition.
		panic("pptx: marshalPart: " + err.Error())
	}
	return buf.Bytes()
}

// contentTypesNamespace is [Content_Types].xml's fixed package-content-types
// XML namespace (ECMA-376 / OPC).
const contentTypesNamespace = "http://schemas.openxmlformats.org/package/2006/content-types"

type ctDefault struct {
	XMLName     xml.Name `xml:"Default"`
	Extension   string   `xml:"Extension,attr"`
	ContentType string   `xml:"ContentType,attr"`
}

type ctOverride struct {
	XMLName     xml.Name `xml:"Override"`
	PartName    string   `xml:"PartName,attr"`
	ContentType string   `xml:"ContentType,attr"`
}

type ctTypes struct {
	XMLName   xml.Name     `xml:"Types"`
	Xmlns     string       `xml:"xmlns,attr"`
	Defaults  []ctDefault  `xml:"Default"`
	Overrides []ctOverride `xml:"Override"`
}

// contentTypeOverride pairs a part name (WITH its leading slash, e.g.
// "/ppt/presentation.xml") with its OOXML content type -- one entry per
// non-boilerplate-extension part in [Content_Types].xml's Override list, in
// the exact order supplied (never re-sorted -- determinism, Pitfall 4).
type contentTypeOverride struct {
	PartName    string
	ContentType string
}

// buildContentTypesXML renders [Content_Types].xml: fixed Default entries
// for the "rels" and "xml" extensions (covering every OPC-relationship part
// and every OOXML XML part in the minimal graph), followed by an Override
// per part in overrides, in the given order.
func buildContentTypesXML(overrides []contentTypeOverride) []byte {
	ct := ctTypes{
		Xmlns: contentTypesNamespace,
		Defaults: []ctDefault{
			{Extension: "rels", ContentType: "application/vnd.openxmlformats-package.relationships+xml"},
			{Extension: "xml", ContentType: "application/xml"},
		},
	}
	for _, o := range overrides {
		ct.Overrides = append(ct.Overrides, ctOverride{PartName: o.PartName, ContentType: o.ContentType})
	}
	return marshalPart(ct)
}

// parsedContentTypes is the decoded, lookup-ready form of a built
// [Content_Types].xml: Defaults keyed by (lowercase, no-dot) extension,
// Overrides keyed by exact part name (leading slash).
type parsedContentTypes struct {
	Defaults  map[string]string
	Overrides map[string]string
}

// parseContentTypesXML decodes a [Content_Types].xml document produced by
// buildContentTypesXML (or an equivalent OPC content-types manifest) into a
// parsedContentTypes for coverage checks.
func parseContentTypesXML(b []byte) (*parsedContentTypes, error) {
	var ct ctTypes
	if err := xml.Unmarshal(b, &ct); err != nil {
		return nil, err
	}
	pc := &parsedContentTypes{Defaults: map[string]string{}, Overrides: map[string]string{}}
	for _, d := range ct.Defaults {
		pc.Defaults[strings.ToLower(d.Extension)] = d.ContentType
	}
	for _, o := range ct.Overrides {
		pc.Overrides[o.PartName] = o.ContentType
	}
	return pc, nil
}

// covers reports whether partName (e.g. "/ppt/presentation.xml") is covered
// by an exact Override, or failing that, by a Default matching its file
// extension -- the two content-type coverage forms OPC defines.
func (pc *parsedContentTypes) covers(partName string) bool {
	if _, ok := pc.Overrides[partName]; ok {
		return true
	}
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(partName), "."))
	_, ok := pc.Defaults[ext]
	return ok
}
