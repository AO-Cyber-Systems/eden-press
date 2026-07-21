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
	"time"
)

// part is one OPC (Open Packaging Conventions) ZIP entry: a part name (its
// ZIP entry path, e.g. "ppt/presentation.xml") paired with its raw content.
// Parts are always assembled into an EXPLICITLY ORDERED []part -- never a
// map -- because Go map iteration order is randomized and would break the
// byte-for-byte determinism this package guarantees (06-RESEARCH Pitfall 4).
type part struct {
	name    string
	content []byte
}

// fixedModified is the single fixed timestamp stamped onto EVERY zip entry's
// FileHeader.Modified. archive/zip defaults Modified to "now" when left
// unset, which would make every rebuild of the same deck produce different
// bytes -- fatal for a golden-hash determinism test. 1980-01-01 is used
// because it is the ZIP/DOS date format's own floor (no earlier date is
// even representable in the format), not because the date carries any other
// meaning.
var fixedModified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// buildZip assembles an OPC ZIP package from parts, in the EXACT order
// given. Every entry uses zip.Store (never zip.Deflate -- compress/flate's
// output is not a documented cross-Go-version byte-stability guarantee,
// 06-RESEARCH Pitfall 4) and the fixed Modified timestamp above, so calling
// buildZip twice with an identical parts slice always produces
// byte-identical output.
func buildZip(parts []part) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		fh := &zip.FileHeader{
			Name:   p.name,
			Method: zip.Store,
		}
		fh.Modified = fixedModified
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(p.content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
