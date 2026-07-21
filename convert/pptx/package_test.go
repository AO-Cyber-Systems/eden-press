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
	"archive/zip"
	"bytes"
	"testing"
)

// TestBuildZipDeterminism proves Test-list case 1: building the exact same
// ordered parts slice twice yields byte-identical ZIP output. This is the
// outermost byte contract every later TRD (06-04, 06-05) depends on for
// golden-hash tests to be stable across machines/Go versions.
func TestBuildZipDeterminism(t *testing.T) {
	parts := []part{
		{name: "[Content_Types].xml", content: []byte("<Types/>")},
		{name: "_rels/.rels", content: []byte("<Relationships/>")},
		{name: "ppt/presentation.xml", content: []byte("<p:presentation/>")},
	}

	b1, err := buildZip(parts)
	if err != nil {
		t.Fatalf("buildZip (build 1): %v", err)
	}
	b2, err := buildZip(parts)
	if err != nil {
		t.Fatalf("buildZip (build 2): %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("buildZip is non-deterministic: build 1 (%d bytes) != build 2 (%d bytes)", len(b1), len(b2))
	}
}

// TestBuildZipDeterminismMetadata locks the two concrete determinism
// mechanisms (06-RESEARCH Pitfall 4): every entry's Modified timestamp is the
// fixed constant (never "now"), every entry's compression Method is
// zip.Store (never zip.Deflate), and entries appear in the EXACT slice order
// given (never map order).
func TestBuildZipDeterminismMetadata(t *testing.T) {
	parts := []part{
		{name: "b.xml", content: []byte("<b/>")},
		{name: "a.xml", content: []byte("<a/>")},
	}
	b, err := buildZip(parts)
	if err != nil {
		t.Fatalf("buildZip: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	if len(zr.File) != len(parts) {
		t.Fatalf("got %d zip entries, want %d", len(zr.File), len(parts))
	}
	for i, f := range zr.File {
		if f.Name != parts[i].name {
			t.Errorf("entry %d name = %q, want %q (explicit slice ordering not preserved)", i, f.Name, parts[i].name)
		}
		if f.Method != zip.Store {
			t.Errorf("entry %q Method = %d, want zip.Store (%d)", f.Name, f.Method, zip.Store)
		}
		if !f.Modified.Equal(fixedModified) {
			t.Errorf("entry %q Modified = %v, want fixed %v", f.Name, f.Modified, fixedModified)
		}
	}
}

// TestContentTypesCoverage proves Test-list case 2: every part name in a zip
// part graph is covered by [Content_Types].xml (an exact Override, or a
// Default matching its extension), and every declared Override actually
// corresponds to a real, zipped part (no dangling/orphan Override).
func TestContentTypesCoverage(t *testing.T) {
	partNames := []string{
		"_rels/.rels",
		"docProps/core.xml",
		"ppt/presentation.xml",
		"ppt/theme/theme1.xml",
	}
	overrides := []contentTypeOverride{
		{PartName: "/docProps/core.xml", ContentType: "application/vnd.openxmlformats-package.core-properties+xml"},
		{PartName: "/ppt/presentation.xml", ContentType: "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"},
		{PartName: "/ppt/theme/theme1.xml", ContentType: "application/vnd.openxmlformats-officedocument.theme+xml"},
	}

	ctXML := buildContentTypesXML(overrides)
	pc, err := parseContentTypesXML(ctXML)
	if err != nil {
		t.Fatalf("parseContentTypesXML: %v", err)
	}

	for _, name := range partNames {
		if !pc.covers("/" + name) {
			t.Errorf("part %q is not covered by [Content_Types].xml (no Override, no matching Default)", name)
		}
	}

	partSet := make(map[string]bool, len(partNames))
	for _, name := range partNames {
		partSet["/"+name] = true
	}
	for pn := range pc.Overrides {
		if !partSet[pn] {
			t.Errorf("Override %q has no corresponding zipped part (orphan override)", pn)
		}
	}
}

// TestContentTypesCoverageNegative is a negative control proving the
// coverage check isn't vacuously true: a part whose extension has no
// Default and no Override must be reported as NOT covered.
func TestContentTypesCoverageNegative(t *testing.T) {
	ctXML := buildContentTypesXML(nil)
	pc, err := parseContentTypesXML(ctXML)
	if err != nil {
		t.Fatalf("parseContentTypesXML: %v", err)
	}
	if pc.covers("/ppt/media/image1.png") {
		t.Fatal("covers() incorrectly reports .png as covered with no Default/Override declared for it")
	}
}
