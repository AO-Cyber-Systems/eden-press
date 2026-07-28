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

package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

// The container class the render path wraps a whole document in was a literal
// "marpit" in renderDocument. chase/profile.Profile.Container() supplies the
// CSS SELECTOR a unit is scoped under, so a second profile could describe a
// container its own markup never emitted -- CSS that cannot match its own DOM.
// These tests pin the seam that closes that gap.

func renderWith(t *testing.T, md string, opts ...goldmark.Option) string {
	t.Helper()
	var buf bytes.Buffer
	if err := NewEngine(opts...).Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return buf.String()
}

// TestContainerClassDefaultsToMarpit is the Marp-compatibility guard: the
// default MUST stay "marpit" or every conformance-corpus case and every
// existing consumer breaks.
func TestContainerClassDefaultsToMarpit(t *testing.T) {
	got := renderWith(t, "# Hello\n")
	if !strings.Contains(got, `<div class="marpit">`) {
		t.Errorf("default container class is not marpit: %s", got)
	}
}

// TestContainerClassOverride: a profile-supplied class reaches the rendered
// document.
func TestContainerClassOverride(t *testing.T) {
	got := renderWith(t, "# Hello\n",
		goldmark.WithRendererOptions(WithContainerClass("edenpress-paged")))
	if !strings.Contains(got, `<div class="edenpress-paged">`) {
		t.Errorf("container class override not applied: %s", got)
	}
	if strings.Contains(got, `class="marpit"`) {
		t.Errorf("marpit container still emitted alongside the override: %s", got)
	}
}

// TestContainerClassEmptyFallsBack: an empty class would emit
// <div class=""> and silently detach every scoped rule, so it falls back.
func TestContainerClassEmptyFallsBack(t *testing.T) {
	got := renderWith(t, "# Hello\n",
		goldmark.WithRendererOptions(WithContainerClass("")))
	if !strings.Contains(got, `<div class="marpit">`) {
		t.Errorf("empty container class did not fall back to the default: %s", got)
	}
}

// TestContainerClassEscaped: the class reaches an HTML attribute, so it must
// not be able to break out of it.
func TestContainerClassEscaped(t *testing.T) {
	got := renderWith(t, "# Hello\n",
		goldmark.WithRendererOptions(WithContainerClass(`x" onload="alert(1)`)))
	if strings.Contains(got, `onload="alert(1)"`) {
		t.Errorf("container class escaped its attribute: %s", got)
	}
}
