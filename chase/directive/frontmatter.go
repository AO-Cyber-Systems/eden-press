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

package directive

// DetectFrontMatter detects a leading YAML front-matter block delimited by
// "---" fence lines at the very start of the document (mirrors
// markdown-it-front-matter, as wired by Marpit's directives/parse.js
// `frontMatter` option). It returns the raw front-matter body (the lines
// between the fences) and the remaining markdown after the closing fence,
// or ok=false if s does not open with a terminated "---" fence.
func DetectFrontMatter(s string) (body string, rest string, ok bool) {
	return "", s, false
}

// ParseFrontMatter parses a front-matter body into an ordered slice of raw
// deck-level directive candidates, using the SAME YAML-ish parser as
// HTML-comment directives (mirrors Marpit's own reuse of directives/yaml.js
// for both comment content and front-matter text).
func ParseFrontMatter(body string) []KV {
	return nil
}
