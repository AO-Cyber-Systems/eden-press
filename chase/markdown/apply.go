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

// Directive -> attribute/style materialization, ported verbatim (branch by
// branch) from directives/apply.js: for every recognized directive key on a
// slide, a `data-{kebab}` attribute AND a `--{kebab}` CSS custom property
// are set (the generic loop); THEN four directives additionally override
// real CSS/HTML in a FIXED code order regardless of declaration order
// (color -> backgroundColor -> backgroundImage -> class/header/footer),
// exactly mirroring apply.js's hardcoded branch sequence.
package markdown

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
)

// KindHeaderElement is the ast.NodeKind of a *HeaderElement node.
var KindHeaderElement = ast.NewNodeKind("MarpitHeaderElement")

// HeaderElement is the `<header>` element the `header` local directive
// inserts as the first child of its slide's Section.
//
// The field is named Content, not Text -- ast.Node already requires a
// `Text(source []byte) []byte` method (BaseNode's default extracts a
// node's rendered text span from source), which a same-named field would
// collide with.
type HeaderElement struct {
	ast.BaseBlock
	Content string
}

func newHeaderElement(text string) *HeaderElement {
	return &HeaderElement{Content: text}
}

// Kind implements ast.Node.
func (n *HeaderElement) Kind() ast.NodeKind { return KindHeaderElement }

// Dump implements ast.Node.
func (n *HeaderElement) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Content": n.Content}, nil)
}

// KindFooterElement is the ast.NodeKind of a *FooterElement node.
var KindFooterElement = ast.NewNodeKind("MarpitFooterElement")

// FooterElement is the `<footer>` element the `footer` local directive
// inserts as the last child of its slide's Section.
type FooterElement struct {
	ast.BaseBlock
	Content string
}

func newFooterElement(text string) *FooterElement {
	return &FooterElement{Content: text}
}

// Kind implements ast.Node.
func (n *FooterElement) Kind() ast.NodeKind { return KindFooterElement }

// Dump implements ast.Node.
func (n *FooterElement) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Content": n.Content}, nil)
}

// applyDirectives materializes each slide's resolved directive map onto its
// Section, in document order. sections, resolvedPerSlide, and keysPerSlide
// are expected to be the same length and index-aligned (one entry per
// slide); a short keysPerSlide/resolvedPerSlide (e.g. an empty document) is
// tolerated defensively.
func applyDirectives(sections []*Section, resolvedPerSlide []map[string]any, keysPerSlide [][]string) {
	for i, sec := range sections {
		var resolved map[string]any
		var keys []string
		if i < len(resolvedPerSlide) {
			resolved = resolvedPerSlide[i]
		}
		if i < len(keysPerSlide) {
			keys = keysPerSlide[i]
		}
		applyDirectivesToSection(sec, keys, resolved)
	}
}

// applyDirectivesToSection materializes one slide's resolved directives
// onto sec, following directives/apply.js's per-token loop verbatim:
//
//  1. generic loop over keys (ordered): data-{kebab} attr + --{kebab} style
//     var, for every truthy value.
//  2. lang attribute (skipped here -- chase/markdown has no marpit-instance
//     default lang to fall back to; only an explicit "lang" directive
//     value applies).
//  3. class -> attrJoin (append to any existing class attr).
//  4. color -> style override.
//  5. backgroundColor -> style override (+ background-image:none).
//  6. backgroundImage -> style override (+ position/repeat/size defaults,
//     each override-able by backgroundPosition/backgroundRepeat/
//     backgroundSize if ALSO present).
//  7. header/footer -> element insertion (first/last child).
//  8. style attrSet, only if any declaration was set.
//
// The pagination counter (data-marpit-pagination / -total) is NOT handled
// here -- see applyPagination (Task 3): apply.js sets
// `data-marpit-pagination` inline in this same per-token loop, but chase/
// markdown splits it out into its own pass over the fully materialized
// Section list so the running page-number counter and the two-pass total
// stay in one auditable place.
func applyDirectivesToSection(sec *Section, keys []string, resolved map[string]any) {
	if resolved == nil {
		return
	}

	style := NewInlineStyle()

	for _, k := range keys {
		v := resolved[k]
		if !truthy(v) {
			continue
		}
		kebab := kebabCase(k)
		sec.Attrs = append(sec.Attrs, Attr{Name: "data-" + kebab, Value: directiveValueString(v)})
		style.Set("--"+kebab, directiveValueString(v))
	}

	if v, ok := resolved["lang"]; ok && truthy(v) {
		sec.Attrs = append(sec.Attrs, Attr{Name: "lang", Value: directiveValueString(v)})
	}
	if v, ok := resolved["class"]; ok && truthy(v) {
		appendClassAttr(sec, directiveValueString(v))
	}
	if v, ok := resolved["color"]; ok && truthy(v) {
		style.Set("color", directiveValueString(v))
	}
	if v, ok := resolved["backgroundColor"]; ok && truthy(v) {
		style.Set("background-color", directiveValueString(v))
		style.Set("background-image", "none")
	}
	if v, ok := resolved["backgroundImage"]; ok && truthy(v) {
		style.Set("background-image", directiveValueString(v))
		style.Set("background-position", "center")
		style.Set("background-repeat", "no-repeat")
		style.Set("background-size", "cover")
		if pv, ok := resolved["backgroundPosition"]; ok && truthy(pv) {
			style.Set("background-position", directiveValueString(pv))
		}
		if rv, ok := resolved["backgroundRepeat"]; ok && truthy(rv) {
			style.Set("background-repeat", directiveValueString(rv))
		}
		if sv, ok := resolved["backgroundSize"]; ok && truthy(sv) {
			style.Set("background-size", directiveValueString(sv))
		}
	}

	if v, ok := resolved["header"]; ok {
		if s, isStr := v.(string); isStr && s != "" {
			prependHeaderElement(sec, s)
		}
	}
	if v, ok := resolved["footer"]; ok {
		if s, isStr := v.(string); isStr && s != "" {
			appendFooterElement(sec, s)
		}
	}

	if !style.Empty() {
		sec.Attrs = append(sec.Attrs, Attr{Name: "style", Value: style.String()})
	}
}

// truthy mirrors JS's `if (value)` truthiness for the value shapes
// chase/directive.CoerceGlobal/CoerceLocal ever produce: nil, false, "",
// and empty slices are falsy; everything else (including 0-length string
// checks aside) is truthy.
func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case []string:
		return len(t) != 0
	case []int:
		return len(t) != 0
	case int:
		return t != 0
	case float64:
		return t != 0
	default:
		return true
	}
}

// directiveValueString stringifies a resolved directive value for use as
// both an HTML attribute value and a CSS custom-property value.
func directiveValueString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case []string:
		return strings.Join(t, ",")
	case []int:
		parts := make([]string, len(t))
		for i, n := range t {
			parts[i] = strconv.Itoa(n)
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// kebabCase converts a camelCase directive name (e.g. "backgroundColor") to
// kebab-case ("background-color"), mirroring lodash.kebabcase for the small,
// closed vocabulary of Marpit directive names -- a full general-purpose
// port is not needed (no digits/acronyms/unicode appear in this vocabulary).
func kebabCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// appendClassAttr mirrors markdown-it Token#attrJoin: append to an existing
// "class" attribute's value (space-separated) if present, else push a new
// one -- NEVER replace.
func appendClassAttr(sec *Section, value string) {
	for i, a := range sec.Attrs {
		if a.Name == "class" {
			sec.Attrs[i].Value = a.Value + " " + value
			return
		}
	}
	sec.Attrs = append(sec.Attrs, Attr{Name: "class", Value: value})
}

// prependHeaderElement inserts a *HeaderElement carrying text as the FIRST
// child of sec, mirroring Marpit's header directive ("It will insert a
// <header> element to the first of each slide contents").
func prependHeaderElement(sec *Section, text string) {
	h := newHeaderElement(text)
	if first := sec.FirstChild(); first != nil {
		sec.InsertBefore(sec, first, h)
	} else {
		sec.AppendChild(sec, h)
	}
}

// appendFooterElement inserts a *FooterElement carrying text as the LAST
// child of sec, mirroring Marpit's footer directive ("It will insert a
// <footer> element to the last of each slide contents").
func appendFooterElement(sec *Section, text string) {
	sec.AppendChild(sec, newFooterElement(text))
}
