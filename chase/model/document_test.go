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
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestDocumentSchemaVersionConstant covers Test-list case 5's constant half:
// SchemaVersion must be a non-empty, stable constant.
func TestDocumentSchemaVersionConstant(t *testing.T) {
	if SchemaVersion == "" {
		t.Fatal("SchemaVersion must be non-empty")
	}
	const want = "eden-press.model/v1"
	if SchemaVersion != want {
		t.Fatalf("SchemaVersion = %q, want %q", SchemaVersion, want)
	}
}

// TestDocumentJSONRoundTrip covers Test-list case 5: json.Marshal(doc) then
// json.Unmarshal round-trips to an equal Document, and the marshaled bytes
// contain "schemaVersion" with the constant value.
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
	if !strings.Contains(string(b), `"schemaVersion":"eden-press.model/v1"`) {
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
