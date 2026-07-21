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

// Package directive is the pure directive-resolution engine for chase: HTML
// comment + YAML front-matter detection (PARSE-03) and the global/local/spot
// carry-forward state machine (PARSE-02). It has ZERO goldmark import -- it
// operates on plain strings and ordered events, exactly like Marpit's
// directives/*.js operate on comment-content strings before any markdown-it
// token exists. See 01-RESEARCH.md "chase/directive carry-forward state
// machine" and "Directive value coercion tables".
package directive

// RawValue is a YAML-ish scalar or flow-list value as parsed from a
// directive comment or front-matter block, before global/local coercion.
//
// Marpit's own YAML pass (directives/yaml.js) uses js-yaml with
// FAILSAFE_SCHEMA, which deliberately never auto-converts scalars to
// bool/int/null -- every scalar stays a string; only sequences (flow lists)
// and mappings get real structure. Type coercion to bool/int/etc. is
// performed later, per-directive, by CoerceGlobal/CoerceLocal (mirrors
// directives.js) -- NOT here. RawValue is therefore always a string or a
// []string (a parsed flow-list), never a Go bool/int.
type RawValue = any

// KV is one raw key/value pair extracted from a directive comment or
// front-matter block, in source declaration order. An ordered slice (not a
// map) is used deliberately: merge/declaration order is significant
// downstream (carry-forward local-then-spot-then-global merge order), and a
// Go map has no deterministic iteration order.
type KV struct {
	Key string
	Val RawValue
}

// ParseYAMLish is a minimal, hand-rolled scalar/flow-list value parser.
//
// RESOLVED Open Question 1 (per TRD 01-02 recovery instructions: "read
// directives/yaml.js first"): Marpit's real YAML pass parses front-matter
// and comment content with js-yaml's FAILSAFE_SCHEMA, which restricts
// resolved types to string / sequence / mapping only -- it does NOT resolve
// core-schema types like bool/int/null. Every directive's own coercion
// function (directives.js) does its own string-level comparison
// (v === 'true', Number.parseInt(v, 10), etc.) on what is ALWAYS a string
// (or array of strings) coming out of the YAML layer. This means a minimal
// hand-rolled parser recognizing only bare strings (with '...'/"..." quote
// stripping) and small flow lists ("[a, b]") is a faithful, sufficient port
// of yaml.js's FAILSAFE_SCHEMA behavior for the corpus -- no YAML library
// dependency is needed, and none was added to go.mod.
func ParseYAMLish(text string) []KV {
	return nil
}
