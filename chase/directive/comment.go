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

import "regexp"

// commentPattern mirrors Marpit's exact detection regex, verbatim from
// lib/markdown/comment.js: /<!--+\s*([\s\S]*?)\s*--+>/. The (?s) flag makes
// "." match newlines too, standing in for JS's "[\s\S]" idiom.
var commentPattern = regexp.MustCompile(`(?s)^<!--+\s*(.*?)\s*--+>`)

// DetectComment detects a (possibly multi-line) HTML comment span starting
// at position 0 of s and returns its trimmed inner body.
//
// This is DETECTION ONLY (RESEARCH Pitfall 4: detection != recognition) --
// it does not decide whether the body is a recognized directive. A
// non-directive comment (e.g. "<!-- just a note -->") is still detected
// (ok=true); ParseComment will simply produce no recognized key/value pairs
// for it. Mirrors comment.js's fast-fail ("charCodeAt(pos) !== 0x3c") plus
// its opening/closing regex match.
func DetectComment(s string) (body string, ok bool) {
	if len(s) == 0 || s[0] != '<' {
		return "", false
	}
	m := commentPattern.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// ParseComment parses a detected comment's raw body into an ordered slice of
// raw key/value pairs using the YAML-ish scalar/flow-list parser (yaml.go).
func ParseComment(body string) []KV {
	return ParseYAMLish(body)
}
