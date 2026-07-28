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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// slide1XML builds a TEST-ONLY, hardcoded trivial slide: one title
// <p:sp> text box with a single <a:t> run. This TRD proves packaging in
// isolation from the docmodel (anti-pattern: "do not depend on the model
// here") -- 06-04 is the consumer that replaces this with real,
// model-driven slide content while reusing every other builder in this
// package unchanged.
func slide1XML() []byte {
	return []byte(xmlDeclaration + fmt.Sprintf(`<p:sld xmlns:a="%s" xmlns:r="%s" xmlns:p="%s">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title 1"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph type="title"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="838200" y="365760"/><a:ext cx="10515600" cy="1143000"/></a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p><a:r><a:rPr lang="en-US" sz="4400" b="1"/><a:t>Eden Press</a:t></a:r></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`, nsDrawingML, nsRelationships, nsPresentationML))
}

// slide1RelsXML builds "ppt/slides/_rels/slide1.xml.rels": -> slideLayout1.xml.
func slide1RelsXML() []byte {
	return buildRelsXML([]relationship{
		{ID: rIDSlideLayout1, Type: relTypeSlideLayout, Target: "../slideLayouts/slideLayout1.xml"},
	})
}

// buildTrivialDeck assembles the FULL minimal OPC part graph (Pattern 1) for
// a single hardcoded title slide, at the given slide size: the deterministic
// zip packager (package.go) + content-types manifest (contenttypes.go) +
// every static boilerplate part (parts_static.go) + this file's trivial
// slide1.xml/slide1.xml.rels. This is the scaffold 06-04 fills with N
// model-driven slides, reusing this exact plumbing unchanged.
func buildTrivialDeck(size SlideSize) ([]byte, error) {
	slides := []slideRef{{RelID: rIDSlide1, Target: "slides/slide1.xml"}}

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
		{PartName: "/ppt/slides/slide1.xml", ContentType: ctSlide},
	}

	parts := []part{
		{name: "[Content_Types].xml", content: buildContentTypesXML(overrides)},
		{name: "_rels/.rels", content: rootRelsXML()},
		{name: "docProps/core.xml", content: docPropsCoreXML()},
		{name: "docProps/app.xml", content: docPropsAppXML(len(slides))},
		{name: "ppt/presentation.xml", content: presentationXML(size, slides)},
		{name: "ppt/_rels/presentation.xml.rels", content: presentationRelsXML(slides)},
		{name: "ppt/presProps.xml", content: presPropsXML()},
		{name: "ppt/viewProps.xml", content: viewPropsXML()},
		{name: "ppt/tableStyles.xml", content: tableStylesXML()},
		{name: "ppt/theme/theme1.xml", content: theme1XML()},
		{name: "ppt/slideMasters/slideMaster1.xml", content: slideMaster1XML()},
		{name: "ppt/slideMasters/_rels/slideMaster1.xml.rels", content: slideMaster1RelsXML()},
		{name: "ppt/slideLayouts/slideLayout1.xml", content: slideLayout1XML()},
		{name: "ppt/slideLayouts/_rels/slideLayout1.xml.rels", content: slideLayout1RelsXML()},
		{name: "ppt/slides/slide1.xml", content: slide1XML()},
		{name: "ppt/slides/_rels/slide1.xml.rels", content: slide1RelsXML()},
	}

	return buildZip(parts)
}

// unzipParts reads a built OPC ZIP and returns its parts as a name->content
// map, for structural assertions.
func unzipParts(t *testing.T, deck []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(deck), int64(len(deck)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	parts := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry %q: %v", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %q: %v", f.Name, err)
		}
		parts[f.Name] = content
	}
	return parts
}

// resolveRelTarget resolves a Relationship's Target (found inside
// "ownerDir/_rels/NAME.rels") against ownerDir, per the OPC convention that
// Target paths are relative to the directory of the PART THAT OWNS the
// .rels file (not the .rels file's own directory).
func resolveRelTarget(ownerDir, target string) string {
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/")
	}
	return path.Clean(path.Join(ownerDir, target))
}

// relsOwnerDir derives the directory of the part that owns relsPartName
// (e.g. "ppt/slideMasters/_rels/slideMaster1.xml.rels" -> "ppt/slideMasters",
// "_rels/.rels" -> ".").
func relsOwnerDir(relsPartName string) string {
	return path.Dir(path.Dir(relsPartName))
}

// assertStructurallyOpenable is the reusable structural openability
// asserter -- this TRD's key deliverable, extended by 06-05 to the full
// model-driven + notes deck. Given a built deck's unzipped parts, it
// asserts: (a) every part is covered by [Content_Types].xml and every
// Override corresponds to a real part (no orphan), and (b) for every .rels
// file, every declared Relationship's Target resolves to a part that
// actually exists in the zip, AND every r:id used in that .rels's owning
// part's own content resolves to an Id declared in that .rels.
func assertStructurallyOpenable(t *testing.T, parts map[string][]byte) {
	t.Helper()

	ct, ok := parts["[Content_Types].xml"]
	if !ok {
		t.Fatal("deck is missing [Content_Types].xml")
	}
	pc, err := parseContentTypesXML(ct)
	if err != nil {
		t.Fatalf("parseContentTypesXML: %v", err)
	}
	for name := range parts {
		if name == "[Content_Types].xml" {
			continue
		}
		if !pc.covers("/" + name) {
			t.Errorf("part %q is not covered by [Content_Types].xml (no Override, no matching Default)", name)
		}
	}
	for pn := range pc.Overrides {
		trimmed := strings.TrimPrefix(pn, "/")
		if _, ok := parts[trimmed]; !ok {
			t.Errorf("[Content_Types].xml Override %q has no corresponding zipped part (orphan override)", pn)
		}
	}

	for name, content := range parts {
		if !strings.HasSuffix(name, ".rels") {
			continue
		}
		rels, err := parseRelsXML(content)
		if err != nil {
			t.Errorf("%s: parseRelsXML: %v", name, err)
			continue
		}

		for _, r := range rels {
			target := resolveRelTarget(relsOwnerDir(name), r.Target)
			if _, ok := parts[target]; !ok {
				t.Errorf("%s: Relationship Id=%q Target=%q resolves to %q, which does not exist in the zip", name, r.ID, r.Target, target)
			}
		}

		if name == "_rels/.rels" {
			// Package-level rels has no single "owning part" with r:id
			// attributes of its own to cross-check against.
			continue
		}
		ownerPartName := path.Join(relsOwnerDir(name), strings.TrimSuffix(path.Base(name), ".rels"))
		ownerContent, ok := parts[ownerPartName]
		if !ok {
			t.Errorf("%s: owning part %q not found in zip", name, ownerPartName)
			continue
		}
		declared := map[string]bool{}
		for _, r := range rels {
			declared[r.ID] = true
		}
		for _, id := range extractRIDs(ownerContent) {
			if !declared[id] {
				t.Errorf("%s: r:id=%q is used in %s but not declared in this .rels", name, id, ownerPartName)
			}
		}
	}
}

var sldSzPattern = regexp.MustCompile(`<p:sldSz cx="(\d+)" cy="(\d+)"(?: type="([^"]*)")?/>`)

// assertSlideSize asserts presentation.xml's <p:sldSz> matches want exactly
// -- Test-list cases 6/7's core assertion (criterion 4: both aspect ratios).
func assertSlideSize(t *testing.T, parts map[string][]byte, want SlideSize) {
	t.Helper()
	presXML, ok := parts["ppt/presentation.xml"]
	if !ok {
		t.Fatal("deck is missing ppt/presentation.xml")
	}
	m := sldSzPattern.FindSubmatch(presXML)
	if m == nil {
		t.Fatalf("ppt/presentation.xml has no <p:sldSz> element")
	}
	cx, _ := strconv.ParseInt(string(m[1]), 10, 64)
	cy, _ := strconv.ParseInt(string(m[2]), 10, 64)
	typ := string(m[3])
	if cx != want.CX || cy != want.CY || typ != want.Type {
		t.Errorf("<p:sldSz> = {cx:%d cy:%d type:%q}, want {cx:%d cy:%d type:%q}", cx, cy, typ, want.CX, want.CY, want.Type)
	}
}

var titleTextPattern = regexp.MustCompile(`(?s)<p:sp>.*?<a:t>([^<]+)</a:t>`)

// assertSlide1HasTitleText asserts slide1.xml contains a <p:sp> with a
// non-empty <a:t> title run.
func assertSlide1HasTitleText(t *testing.T, parts map[string][]byte) {
	t.Helper()
	slideXML, ok := parts["ppt/slides/slide1.xml"]
	if !ok {
		t.Fatal("deck is missing ppt/slides/slide1.xml")
	}
	m := titleTextPattern.FindSubmatch(slideXML)
	if m == nil || len(strings.TrimSpace(string(m[1]))) == 0 {
		t.Fatal("ppt/slides/slide1.xml has no <p:sp> containing a non-empty <a:t> run")
	}
}

// TestTrivialDeckOpenable16x9 proves Test-list case 6: the trivial static
// deck, built at 16:9, passes the structural openability asserter and
// carries the correct SlideSize16x9 <p:sldSz>.
func TestTrivialDeckOpenable16x9(t *testing.T) {
	deck, err := buildTrivialDeck(SlideSize16x9)
	if err != nil {
		t.Fatalf("buildTrivialDeck: %v", err)
	}
	parts := unzipParts(t, deck)
	assertStructurallyOpenable(t, parts)
	assertSlideSize(t, parts, SlideSize16x9)
	assertSlide1HasTitleText(t, parts)
}

// TestTrivialDeckOpenable4x3 proves Test-list case 7: the SAME build code
// path, only the SlideSize argument differing, passes at 4:3 -- criterion 4
// (both aspect ratios) proven on the static deck before any model content
// exists.
func TestTrivialDeckOpenable4x3(t *testing.T) {
	deck, err := buildTrivialDeck(SlideSize4x3)
	if err != nil {
		t.Fatalf("buildTrivialDeck: %v", err)
	}
	parts := unzipParts(t, deck)
	assertStructurallyOpenable(t, parts)
	assertSlideSize(t, parts, SlideSize4x3)
	assertSlide1HasTitleText(t, parts)
}

// TestTrivialDeckLibreOfficeSmoke is Test-list case 8 (optional): if
// soffice is on PATH, converts the trivial deck to PDF headlessly (an
// independent-consumer open proof beyond our own structural asserter) using
// a unique UserInstallation per invocation (06-RESEARCH anti-pattern: a
// shared profile directory causes lock hangs). Skips cleanly when soffice
// is absent so CI without LibreOffice still passes on the structural
// asserts alone.
func TestTrivialDeckLibreOfficeSmoke(t *testing.T) {
	sofficePath := findSoffice()
	if sofficePath == "" {
		t.Skip("no LibreOffice binary found; skipping LibreOffice-headless openability smoke")
	}

	deck, err := buildTrivialDeck(SlideSize16x9)
	if err != nil {
		t.Fatalf("buildTrivialDeck: %v", err)
	}

	tmpDir := t.TempDir()
	deckPath := filepath.Join(tmpDir, "trivial.pptx")
	if err := os.WriteFile(deckPath, deck, 0o644); err != nil {
		t.Fatalf("write deck: %v", err)
	}

	userInstall := "file://" + filepath.Join(tmpDir, "soffice-profile")
	cmd := exec.Command(sofficePath,
		"--headless",
		"--convert-to", "pdf",
		"--outdir", tmpDir,
		"-env:UserInstallation="+userInstall,
		deckPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("soffice --headless --convert-to pdf failed: %v\noutput: %s", err, out)
	}

	pdfPath := filepath.Join(tmpDir, "trivial.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("expected soffice to produce %s: %v", pdfPath, err)
	}
}

// findSoffice locates a LibreOffice binary: PATH first, then the standard
// install locations.
//
// This used to be exec.LookPath alone, which silently skipped on any machine
// where LibreOffice is installed but not symlinked onto PATH -- the default on
// macOS. That skip hid a real defect for the life of the package: buildZip
// emitted STORED entries carrying a data-descriptor flag, which LibreOffice
// refuses to open at all. A smoke test that never runs is not evidence, so
// this one looks in the app bundle before giving up.
func findSoffice() string {
	if p, err := exec.LookPath("soffice"); err == nil {
		return p
	}
	for _, p := range []string{
		"/Applications/LibreOffice.app/Contents/MacOS/soffice",
		"/usr/lib/libreoffice/program/soffice",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
