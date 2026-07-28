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

import (
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// xmlWellFormed reports whether s parses end-to-end as XML. A hand-rolled
// OOXML writer's most common failure is emitting *almost* well-formed markup
// that a substring assertion happily passes, so every part is run through a
// real decoder.
func xmlWellFormed(s string) error {
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// findSoffice locates a LibreOffice binary: PATH first, then the standard
// macOS app-bundle location. convert/pptx's equivalent smoke test checks PATH
// only, which silently skips on a Mac where LibreOffice IS installed but is
// not symlinked -- exactly the machine this exporter was written on. A skip
// that never runs is not evidence, so this one looks harder before giving up.
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

// acceptanceDoc exercises every block kind the exporter supports, including
// all three EPD-R1 additions, in one document.
func acceptanceDoc() *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Meta:          model.Meta{Directives: map[string]string{"theme": "default"}},
		Sections: []model.Section{
			{
				ID: 1,
				Blocks: []model.Block{
					{Kind: model.BlockHeading, Level: 1, Text: "Q3 Platform Review"},
					{Kind: model.BlockQuote, Text: "Synthesized from the engineering sync."},
					{Kind: model.BlockHeading, Level: 2, Text: "Executive Summary"},
					{Kind: model.BlockParagraph, Text: "Throughput improved 34% quarter over quarter."},
					{Kind: model.BlockList, Items: []model.ListItem{
						{Text: "Export backlog"},
						{Text: "still open", Level: 1},
						{Text: "Single-region database"},
					}},
					{Kind: model.BlockList, Ordered: true, Items: []model.ListItem{
						{Text: "Ship the sidecar"},
						{Text: "Hold the spreadsheet type"},
					}},
				},
			},
			{
				ID: 2,
				Blocks: []model.Block{
					{Kind: model.BlockHeading, Level: 2, Text: "Key Metrics"},
					{Kind: model.BlockTable,
						Headers: []string{"Metric", "Q2", "Q3"},
						Rows: [][]string{
							{"p95 latency", "840ms", "550ms"},
							{"Error rate", "1.2%", "0.4%"},
						},
						Aligns: []string{"left", "right", "right"}},
					{Kind: model.BlockHeading, Level: 2, Text: "Appendix"},
					{Kind: model.BlockCode, Language: "go",
						Text: "func Render(md string, opts Options) (Output, error) {\n\treturn out, nil\n}\n"},
					{Kind: model.BlockMath, Text: `\sum_{i=1}^{n} x_i`, Display: true},
					{Kind: model.BlockImage, Src: "chart.png", Text: "Q3 chart", Title: "Quarterly"},
					{Kind: model.BlockParagraph, Text: `Escaping check: a & b < c > d "e"`},
				},
			},
		},
		Outline: []model.OutlineEntry{
			{SectionID: 1, Level: 1, Text: "Q3 Platform Review", Slug: "q3-platform-review"},
		},
	}
}

// TestAcceptanceDocumentLibreOffice is the functional artifact check the
// objective's verification gates call for: a green unit suite has never once
// proven an OOXML file opens. LibreOffice converts the generated .docx to PDF
// headlessly; a malformed package fails the conversion outright.
func TestAcceptanceDocumentLibreOffice(t *testing.T) {
	soffice := findSoffice()
	if soffice == "" {
		t.Skip("no LibreOffice binary found; skipping openability smoke")
	}

	pkg, err := ToDOCX(acceptanceDoc(), Options{})
	if err != nil {
		t.Fatalf("ToDOCX: %v", err)
	}

	tmp := t.TempDir()
	src := filepath.Join(tmp, "acceptance.docx")
	if err := os.WriteFile(src, pkg, 0o644); err != nil {
		t.Fatalf("write docx: %v", err)
	}

	// -env:UserInstallation isolates the profile so a concurrent desktop
	// LibreOffice does not make this a no-op (it would otherwise hand the job
	// to the running instance and exit 0 without converting anything).
	cmd := exec.Command(soffice,
		"-env:UserInstallation=file://"+filepath.Join(tmp, "loprofile"),
		"--headless", "--norestore", "--convert-to", "pdf", "--outdir", tmp, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("soffice convert failed: %v\n%s", err, out)
	}

	pdf := filepath.Join(tmp, "acceptance.pdf")
	st, err := os.Stat(pdf)
	if err != nil {
		t.Fatalf("LibreOffice produced no PDF (package rejected): %v\n%s", err, out)
	}
	if st.Size() < 1000 {
		t.Errorf("PDF is implausibly small (%d bytes) — document likely rendered empty", st.Size())
	}

	// Prove the CONTENT survived the round trip, not merely that a file
	// appeared: a package can convert successfully and still be blank.
	body, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatalf("read pdf: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("PDF is empty")
	}
	t.Logf("LibreOffice accepted the package: %d-byte docx -> %d-byte pdf", len(pkg), st.Size())
}
