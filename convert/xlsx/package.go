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
	"archive/zip"
	"bytes"
	"hash/crc32"
	"time"
)

// part is one OPC ZIP entry. Always assembled into an EXPLICITLY ORDERED
// []part -- never a map -- so the package is byte-for-byte reproducible.
type part struct {
	name    string
	content []byte
}

// fixedModified is the single fixed timestamp stamped onto every zip entry, so
// rebuilding the same workbook yields identical bytes.
var fixedModified = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// buildZip assembles an OPC ZIP package from parts, in the EXACT order given.
//
// CreateRaw, not CreateHeader -- see convert/pptx/package.go for the full
// account. In short: CreateHeader sets the data-descriptor flag and zeroes the
// CRC and sizes in the local header, which for a STORED entry leaves strict
// readers (LibreOffice among them) unable to find where the data ends, and they
// reject the archive before reading any OOXML. That defect shipped undetected
// in convert/pptx; this package does not repeat it.
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
		if _, err := w.Write(p.content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// setDOSModified writes t into fh's MS-DOS date/time words. CreateRaw does not
// derive them from FileHeader.Modified the way CreateHeader does; left zero
// they encode a DOS underflow that reads back as 1979-11-30.
func setDOSModified(fh *zip.FileHeader, t time.Time) {
	t = t.UTC()
	fh.ModifiedDate = uint16((t.Year()-1980)<<9 | int(t.Month())<<5 | t.Day())
	fh.ModifiedTime = uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2)
}
