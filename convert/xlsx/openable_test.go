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

package xlsx

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
// OOXML writer's most common failure is almost-well-formed markup that a
// substring assertion happily accepts, so every part goes through a decoder.
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
// install locations. Checking PATH alone silently skips on macOS, which is how
// convert/pptx's data-descriptor defect survived undetected.
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

func acceptanceWorkbook() *model.Document {
	return &model.Document{
		SchemaVersion: model.SchemaVersion,
		Sections: []model.Section{
			{ID: 1, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: "Key Metrics"},
				{Kind: model.BlockTable,
					Headers: []string{"Metric", "Q2", "Q3", "Delta"},
					Rows: [][]string{
						{"p95 latency", "840", "550", "-34"},
						{"Error rate", "1.2", "0.4", "-0.8"},
						{"Docs exported", "0", "0", "0"},
					},
					Aligns: []string{"left", "right", "right", "right"}},
			}},
			{ID: 2, Blocks: []model.Block{
				{Kind: model.BlockHeading, Level: 1, Text: `Costs / Forecast`},
				{Kind: model.BlockTable,
					Headers: []string{"Item", "Amount"},
					Rows: [][]string{
						{"Sidecar pods", "120.50"},
						{`Escaping check: a & b <c>`, "7"},
					}},
			}},
		},
		Outline: []model.OutlineEntry{
			{SectionID: 1, Level: 1, Text: "Key Metrics", Slug: "key-metrics"},
		},
	}
}

// TestAcceptanceWorkbookLibreOffice is the functional artifact check: a green
// unit suite has never proven an OOXML file opens. LibreOffice converting the
// workbook to CSV both proves the package loads AND lets the cell values be
// read back, so this asserts content survived rather than merely that a file
// appeared.
func TestAcceptanceWorkbookLibreOffice(t *testing.T) {
	soffice := findSoffice()
	if soffice == "" {
		t.Skip("no LibreOffice binary found; skipping openability smoke")
	}

	pkg, err := ToXLSX(acceptanceWorkbook(), Options{})
	if err != nil {
		t.Fatalf("ToXLSX: %v", err)
	}

	tmp := t.TempDir()
	src := filepath.Join(tmp, "acceptance.xlsx")
	if err := os.WriteFile(src, pkg, 0o644); err != nil {
		t.Fatalf("write xlsx: %v", err)
	}

	cmd := exec.Command(soffice,
		"-env:UserInstallation=file://"+filepath.Join(tmp, "loprofile"),
		"--headless", "--norestore", "--convert-to", "csv", "--outdir", tmp, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("soffice convert failed: %v\n%s", err, out)
	}

	csvPath := filepath.Join(tmp, "acceptance.csv")
	body, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("LibreOffice produced no CSV (package rejected): %v\n%s", err, out)
	}
	got := string(body)

	// LibreOffice converts only the FIRST sheet to CSV, so assert on sheet 1.
	for _, want := range []string{"Metric", "p95 latency", "840", "550", "Error rate"} {
		if !strings.Contains(got, want) {
			t.Errorf("cell %q missing from the round-tripped sheet:\n%s", want, got)
		}
	}
	t.Logf("LibreOffice accepted the workbook: %d-byte xlsx -> CSV:\n%s", len(pkg), got)
}
