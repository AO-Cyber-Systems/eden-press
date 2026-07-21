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
	"io"
	"regexp"
	"strings"
	"testing"
)

// rIDPattern matches every r:id="..." attribute value appearing in a raw
// OOXML part's XML content -- the officeDocument-relationships namespace
// prefix ("r:") used consistently throughout this minimal part graph.
var rIDPattern = regexp.MustCompile(`r:id="([^"]+)"`)

// extractRIDs returns the distinct r:id attribute values found in
// xmlContent, in first-seen order. A regexp/string-scan (rather than a full
// encoding/xml unmarshal) is the TRD-sanctioned approach for this kind of
// structural assert against our own hand-rolled, always-well-formed XML.
func extractRIDs(xmlContent []byte) []string {
	matches := rIDPattern.FindAllSubmatch(xmlContent, -1)
	seen := map[string]bool{}
	var ids []string
	for _, m := range matches {
		id := string(m[1])
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// parseRelsXML decodes a PARTNAME.rels document (as produced by
// buildRelsXML) back into a []relationship, for r:id-resolution assertions.
func parseRelsXML(b []byte) ([]relationship, error) {
	var doc relsDocXML
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	rels := make([]relationship, 0, len(doc.Relationships))
	for _, r := range doc.Relationships {
		rels = append(rels, relationship{ID: r.ID, Type: r.Type, Target: r.Target})
	}
	return rels, nil
}

// assertRIDsResolve asserts every r:id found in partXML resolves to a
// Relationship declared in relsXML -- Test-list case 3 (r:id-resolution
// closure), scoped to a single part + its sibling .rels file.
func assertRIDsResolve(t *testing.T, partName string, partXML, relsXML []byte) {
	t.Helper()
	rels, err := parseRelsXML(relsXML)
	if err != nil {
		t.Fatalf("%s: parseRelsXML: %v", partName, err)
	}
	declared := map[string]bool{}
	for _, r := range rels {
		declared[r.ID] = true
	}
	for _, id := range extractRIDs(partXML) {
		if !declared[id] {
			t.Errorf("%s references r:id=%q, but no Relationship with that Id is declared in its .rels", partName, id)
		}
	}
}

// TestRelsResolutionPresentation proves presentation.xml's r:id references
// (sldMasterId, each sldId) all resolve against presentation.xml.rels --
// the #1 "unresolved r:id" bug the TRD warns about.
func TestRelsResolutionPresentation(t *testing.T) {
	slides := []slideRef{{RelID: rIDSlide1, Target: "slides/slide1.xml"}}
	assertRIDsResolve(t, "ppt/presentation.xml", presentationXML(SlideSize16x9, slides), presentationRelsXML(slides))
}

// TestRelsResolutionSlideMaster proves slideMaster1.xml's sldLayoutId r:id
// resolves against slideMaster1.xml.rels.
func TestRelsResolutionSlideMaster(t *testing.T) {
	assertRIDsResolve(t, "ppt/slideMasters/slideMaster1.xml", slideMaster1XML(), slideMaster1RelsXML())
}

// TestRelsTargetsAreWellFormed spot-checks every declared relationship in
// this static part graph has a non-empty Id/Type/Target -- guarding against
// an accidentally-omitted Target, which would otherwise "resolve" vacuously
// under assertRIDsResolve's Id-only check.
func TestRelsTargetsAreWellFormed(t *testing.T) {
	slides := []slideRef{{RelID: rIDSlide1, Target: "slides/slide1.xml"}}
	mustParse := func(name string, b []byte) []relationship {
		rels, err := parseRelsXML(b)
		if err != nil {
			t.Fatalf("%s: parseRelsXML: %v", name, err)
		}
		return rels
	}
	allRels := [][]relationship{
		mustParse("_rels/.rels", rootRelsXML()),
		mustParse("ppt/_rels/presentation.xml.rels", presentationRelsXML(slides)),
		mustParse("ppt/slideMasters/_rels/slideMaster1.xml.rels", slideMaster1RelsXML()),
		mustParse("ppt/slideLayouts/_rels/slideLayout1.xml.rels", slideLayout1RelsXML()),
	}
	for _, rels := range allRels {
		for _, r := range rels {
			if r.ID == "" || r.Type == "" || r.Target == "" {
				t.Errorf("relationship with an empty field: %+v", r)
			}
		}
	}
}

// TestClrMap12Attrs proves Test-list case 4 / Pitfall 2: slideMaster1.xml's
// <p:clrMap> carries ALL 12 required attributes, no more, no fewer.
func TestClrMap12Attrs(t *testing.T) {
	xmlContent := slideMaster1XML()
	requiredAttrs := []string{
		"bg1", "tx1", "bg2", "tx2",
		"accent1", "accent2", "accent3", "accent4", "accent5", "accent6",
		"hlink", "folHlink",
	}
	clrMapRe := regexp.MustCompile(`<p:clrMap([^/]*)/>`)
	m := clrMapRe.FindSubmatch(xmlContent)
	if m == nil {
		t.Fatalf("slideMaster1.xml has no <p:clrMap .../> element")
	}
	attrsBlob := string(m[1])
	for _, attr := range requiredAttrs {
		attrRe := regexp.MustCompile(attr + `="[^"]+"`)
		if !attrRe.MatchString(attrsBlob) {
			t.Errorf("<p:clrMap> is missing required attribute %q", attr)
		}
	}
	count := strings.Count(attrsBlob, `="`)
	if count != 12 {
		t.Errorf("<p:clrMap> has %d attributes, want exactly 12", count)
	}
}

// TestFmtSchemeThreeEntriesPerList proves Test-list case 5 / Pitfall 3:
// theme1.xml's fillStyleLst/lnStyleLst/effectStyleLst/bgFillStyleLst each
// have EXACTLY 3 direct children.
func TestFmtSchemeThreeEntriesPerList(t *testing.T) {
	xmlContent := string(theme1XML())
	cases := []struct {
		list      string
		childOpen string
	}{
		{"fillStyleLst", "<a:solidFill>"},
		{"lnStyleLst", "<a:ln "},
		{"effectStyleLst", "<a:effectStyle>"},
		{"bgFillStyleLst", "<a:solidFill>"},
	}
	for _, c := range cases {
		openTag := "<a:" + c.list + ">"
		closeTag := "</a:" + c.list + ">"
		start := strings.Index(xmlContent, openTag)
		end := strings.Index(xmlContent, closeTag)
		if start == -1 || end == -1 || end < start {
			t.Fatalf("theme1.xml missing well-formed <a:%s>...</a:%s>", c.list, c.list)
		}
		inner := xmlContent[start+len(openTag) : end]
		got := strings.Count(inner, c.childOpen)
		if got != 3 {
			t.Errorf("<a:%s> has %d %q entries, want exactly 3", c.list, got, c.childOpen)
		}
	}
}

// TestStaticPartsProduceWellFormedXML asserts every static-part builder in
// this file emits non-empty, well-formed XML. Map iteration here is a
// TEST-ONLY well-formedness sweep, not zip/content-types assembly order --
// it carries no determinism requirement.
func TestStaticPartsProduceWellFormedXML(t *testing.T) {
	slides := []slideRef{{RelID: rIDSlide1, Target: "slides/slide1.xml"}}
	builders := map[string][]byte{
		"_rels/.rels":                                  rootRelsXML(),
		"docProps/core.xml":                            docPropsCoreXML(),
		"docProps/app.xml":                             docPropsAppXML(len(slides)),
		"ppt/presentation.xml":                         presentationXML(SlideSize16x9, slides),
		"ppt/_rels/presentation.xml.rels":              presentationRelsXML(slides),
		"ppt/presProps.xml":                            presPropsXML(),
		"ppt/viewProps.xml":                            viewPropsXML(),
		"ppt/tableStyles.xml":                          tableStylesXML(),
		"ppt/theme/theme1.xml":                         theme1XML(),
		"ppt/slideMasters/slideMaster1.xml":            slideMaster1XML(),
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": slideMaster1RelsXML(),
		"ppt/slideLayouts/slideLayout1.xml":            slideLayout1XML(),
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": slideLayout1RelsXML(),
	}
	for name, content := range builders {
		if len(content) == 0 {
			t.Errorf("%s: builder produced empty content", name)
			continue
		}
		dec := xml.NewDecoder(bytes.NewReader(content))
		for {
			if _, err := dec.Token(); err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("%s: not well-formed XML: %v", name, err)
				break
			}
		}
	}
}
