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
	"archive/zip"
	"bytes"
	"hash/crc32"
	"time"
)

// part is one OPC (Open Packaging Conventions) ZIP entry: a part name (its ZIP
// entry path, e.g. "word/document.xml") paired with its raw content. Parts are
// always assembled into an EXPLICITLY ORDERED []part -- never a map -- because
// Go map iteration order is randomized and would break the byte-for-byte
// determinism this package guarantees.
type part struct {
	name    string
	content []byte
}

// fixedModified is the single fixed timestamp stamped onto EVERY zip entry's
// FileHeader.Modified. archive/zip defaults Modified to "now" when left unset,
// which would make every rebuild of the same document produce different bytes.
// 1980-01-01 is the ZIP/DOS date format's own floor.
var fixedModified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// buildZip assembles an OPC ZIP package from parts, in the EXACT order given.
// Every entry uses zip.Store (never zip.Deflate -- compress/flate's output is
// not a documented cross-Go-version byte-stability guarantee) and the fixed
// Modified timestamp above.
//
// CreateRaw, not CreateHeader. zip.Writer.CreateHeader unconditionally sets
// general-purpose flag bit 3 (0x8) and writes zeros for the CRC and both sizes
// in the local file header, deferring them to a trailing data descriptor. That
// is legal, and fine for a DEFLATE entry whose end a reader can detect from the
// compressed stream -- but for a STORED entry it leaves a strict reader with no
// way to know where the entry's data ends without scanning for the descriptor
// signature. LibreOffice's zip reader refuses such an archive outright ("source
// file could not be loaded"), which means a package built with CreateHeader is
// rejected before a single byte of its OOXML is examined.
//
// CreateRaw writes the caller-supplied CRC32 and sizes directly into the local
// header and leaves the flag clear, producing the fully-specified STORED entry
// every consumer accepts. Determinism is unaffected: the CRC and sizes are pure
// functions of the content.
//
// NOTE: this mirrors convert/pptx/package.go deliberately. Both exporters keep
// their own copy so neither can break the other, matching the self-contained
// shape convert/pptx already established; extracting a shared convert/opc
// package is a worthwhile follow-up once a third OOXML writer exists to prove
// the abstraction against.
func buildZip(parts []part) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		fh := &zip.FileHeader{Name: p.name, Method: zip.Store}
		fh.Modified = fixedModified
		setDOSModified(fh, fixedModified)
		fh.CRC32 = crc32.ChecksumIEEE(p.content)
		fh.CompressedSize64 = uint64(len(p.content))
		fh.UncompressedSize64 = uint64(len(p.content))

		w, err := zw.CreateRaw(fh)
		if err != nil {
			return nil, err
		}
		// Method is Store, so the "raw" (already-compressed) bytes ARE the
		// uncompressed content.
		if _, err := w.Write(p.content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// setDOSModified writes t into fh's MS-DOS date/time fields.
//
// CreateHeader derives these from FileHeader.Modified for the caller;
// CreateRaw does not -- it writes the header fields exactly as supplied, and
// leaving them zero encodes the DOS epoch underflow value that reads back as
// 1979-11-30. Setting them explicitly keeps every entry stamped with the fixed
// timestamp determinism depends on, and keeps zip.Reader's round-trip of
// FileHeader.Modified exact.
//
// The MS-DOS encoding (ECMA-119 / PKWARE APPNOTE 4.4.6): the date word packs
// (year-1980)<<9 | month<<5 | day, and the time word hour<<11 | minute<<5 |
// second/2 -- two-second resolution, which is why seconds are halved.
func setDOSModified(fh *zip.FileHeader, t time.Time) {
	t = t.UTC()
	fh.ModifiedDate = uint16((t.Year()-1980)<<9 | int(t.Month())<<5 | t.Day())
	fh.ModifiedTime = uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2)
}
