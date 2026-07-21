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

package convert

// ImageFormat selects the raster image format a screenshot-based exporter
// (convert/png, added by a later TRD) produces. The zero value is PNG.
type ImageFormat int

const (
	// PNG is the lossless raster format and the zero value of ImageFormat.
	PNG ImageFormat = iota
	// JPEG is the lossy raster format, selectable when smaller output size
	// matters more than pixel-perfect fidelity.
	JPEG
)

// String returns the lower-case name of the format ("png" or "jpeg"). An
// unrecognized value (outside the two declared constants) returns "unknown"
// rather than panicking.
func (f ImageFormat) String() string {
	switch f {
	case PNG:
		return "png"
	case JPEG:
		return "jpeg"
	default:
		return "unknown"
	}
}

// Options is the shared cross-exporter option surface both convert/pdf and
// convert/png (added by later TRDs) consume. It is deliberately minimal --
// mirroring the frozen-surface discipline of press.Options -- and grows a
// field only when a named consumer needs it, never speculatively.
type Options struct {
	// BrowserPath, when non-empty, is an explicit Chrome/Chromium executable
	// override -- the highest-precedence tier of convert/chrome.Discover's
	// fallback chain. Leave empty to let Discover fall through to
	// CHROME_PATH, then auto-detection, then the documented pinned-download
	// tier.
	BrowserPath string
}
