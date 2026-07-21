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

package model

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestDocumentSchemaVersionConstant covers Test-list case 5's constant half:
// SchemaVersion must be a non-empty, stable constant. 06-01 bumps it v1 -> v2
// for the additive Section.Blocks enrichment.
func TestDocumentSchemaVersionConstant(t *testing.T) {
	if SchemaVersion == "" {
		t.Fatal("SchemaVersion must be non-empty")
	}
	const want = "eden-press.model/v2"
	if SchemaVersion != want {
		t.Fatalf("SchemaVersion = %q, want %q", SchemaVersion, want)
	}
}

// TestDocumentJSONRoundTrip covers Test-list case 5: json.Marshal(doc) then
// json.Unmarshal round-trips to an equal Document, and the marshaled bytes
// contain "schemaVersion" with the constant value (now v2).
func TestDocumentJSONRoundTrip(t *testing.T) {
	doc := &Document{
		SchemaVersion: SchemaVersion,
		Meta: Meta{
			Directives: map[string]string{"theme": "gaia", "size": "16:9"},
		},
		Sections: []Section{
			{
				ID:    1,
				Attrs: map[string]string{"class": "lead"},
				Notes: []string{"a presenter note"},
			},
			{ID: 2},
		},
		Outline: []OutlineEntry{
			{SectionID: 1, Level: 1, Text: "Hello", Slug: "hello"},
		},
	}

	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(b), `"schemaVersion":"eden-press.model/v2"`) {
		t.Fatalf("marshaled JSON missing expected schemaVersion field: %s", b)
	}

	var got Document
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(*doc, got) {
		t.Fatalf("round-trip mismatch:\n got  = %+v\n want = %+v", got, *doc)
	}
}

// TestBlockOmitemptyAdditive covers Test-list case 1 (additive-non-breaking,
// outermost regression): a v1-era document carrying only headings/notes (NO
// extracted Blocks) marshals with NO `blocks` key anywhere (omitempty), and its
// JSON differs from the frozen v1 shape by ONLY the schemaVersion string. This
// is the proof that Section.Blocks + every new Block field is purely additive:
// a block-less document's bytes are byte-for-byte identical to v1 once the
// version string is normalized back.
func TestBlockOmitemptyAdditive(t *testing.T) {
	doc := &Document{
		SchemaVersion: SchemaVersion,
		Meta: Meta{
			Directives: map[string]string{"theme": "gaia"},
		},
		Sections: []Section{
			{ID: 1, Notes: []string{"a note"}},
		},
		Outline: []OutlineEntry{
			{SectionID: 1, Level: 1, Text: "H", Slug: "h"},
		},
	}

	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)

	if strings.Contains(got, "blocks") {
		t.Fatalf("block-less document must NOT serialize a `blocks` key (omitempty violated): %s", got)
	}
	if !strings.Contains(got, `"schemaVersion":"eden-press.model/v2"`) {
		t.Fatalf("expected v2 schemaVersion: %s", got)
	}

	// The FROZEN v1 shape: this exact byte string is what a pre-06-01 build of
	// the identical document produced. Normalizing v2 -> v1 must reproduce it
	// byte-for-byte -- i.e. the ONLY difference is the version string.
	const frozenV1 = `{"schemaVersion":"eden-press.model/v1","meta":{"directives":{"theme":"gaia"}},"sections":[{"id":1,"notes":["a note"]}],"outline":[{"sectionId":1,"level":1,"text":"H","slug":"h"}]}`
	normalized := strings.ReplaceAll(got, "eden-press.model/v2", "eden-press.model/v1")
	if normalized != frozenV1 {
		t.Fatalf("block-less v2 JSON differs from frozen v1 by more than the version string:\n got  = %s\n want = %s", normalized, frozenV1)
	}
}

// TestSchemaV2RoundTrip covers Test-list case 2: a Document carrying one Block
// of EVERY kind (heading, paragraph, list w/ nested items, code w/ language,
// display math) round-trips byte-stably (Marshal -> Unmarshal -> Marshal is
// identical) and reports SchemaVersion == v2.
func TestSchemaV2RoundTrip(t *testing.T) {
	doc := &Document{
		SchemaVersion: SchemaVersion,
		Meta:          Meta{Directives: map[string]string{"theme": "default"}},
		Sections: []Section{
			{
				ID: 1,
				Blocks: []Block{
					{Kind: BlockHeading, Level: 2, Text: "H2"},
					{Kind: BlockParagraph, Text: "a paragraph"},
					{Kind: BlockList, Ordered: true, Items: []ListItem{
						{Text: "one"},
						{Text: "two", Level: 1},
					}},
					{Kind: BlockCode, Language: "go", Text: "fmt.Println()\n"},
					{Kind: BlockMath, Text: "E=mc^2", Display: true},
				},
			},
		},
		Outline: []OutlineEntry{
			{SectionID: 1, Level: 2, Text: "H2", Slug: "h2"},
		},
	}

	if doc.SchemaVersion != "eden-press.model/v2" {
		t.Fatalf("SchemaVersion = %q, want v2", doc.SchemaVersion)
	}

	b1, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal #1: %v", err)
	}

	var got Document
	if err := json.Unmarshal(b1, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(*doc, got) {
		t.Fatalf("round-trip mismatch:\n got  = %+v\n want = %+v", got, *doc)
	}

	b2, err := json.Marshal(&got)
	if err != nil {
		t.Fatalf("Marshal #2: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("Marshal->Unmarshal->Marshal not byte-stable:\n b1 = %s\n b2 = %s", b1, b2)
	}
}
