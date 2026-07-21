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

package chase

import (
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/chase/markdown"
	"github.com/AO-Cyber-Systems/eden-press/chase/model"
)

// multiSlideMD is a hand-built, two-slide deck with front matter -- the
// shared fixture for this file's Test-list cases 1-4.
const multiSlideMD = `---
theme: default
paginate: true
---

# Slide 1

first slide body

---

## Slide 2

second slide body
`

// TestRenderReturnsHTMLCSSAndModelFromOneCall covers Test-list case 1:
// chase.Render on a multi-slide deck returns non-empty HTML (the
// .marpit/<section> deck), non-empty CSS (packed slides theme), and a
// populated *model.Document (Sections/Outline) -- all from one call.
func TestRenderReturnsHTMLCSSAndModelFromOneCall(t *testing.T) {
	out, err := Render(multiSlideMD)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(out.HTML, `class="marpit"`) {
		t.Fatalf("HTML missing .marpit container, got:\n%s", out.HTML)
	}
	if !strings.Contains(out.HTML, "<section ") {
		t.Fatalf("HTML missing <section>, got:\n%s", out.HTML)
	}
	if out.CSS == "" {
		t.Fatalf("CSS is empty, want packed slides theme CSS")
	}
	if out.Model == nil {
		t.Fatalf("Model is nil, want a populated *model.Document")
	}
	if len(out.Model.Sections) != 2 {
		t.Fatalf("Model.Sections = %d, want 2 (Sections: %+v)", len(out.Model.Sections), out.Model.Sections)
	}
	if len(out.Model.Outline) != 2 {
		t.Fatalf("Model.Outline = %d, want 2 (Outline: %+v)", len(out.Model.Outline), out.Model.Outline)
	}
	if got := out.Meta.Directives["theme"]; got != "default" {
		t.Fatalf(`Meta.Directives["theme"] = %q, want "default"`, got)
	}
	if out.Model.Meta.Directives["theme"] != out.Meta.Directives["theme"] {
		t.Fatalf("Output.Meta must be Output.Model.Meta (the same value), got Output.Meta=%+v Model.Meta=%+v", out.Meta, out.Model.Meta)
	}
}

// TestOneParseTwoSinksHTMLUnaffectedByModelBuild covers Test-list case 2
// (the MODEL-02 acceptance criterion): render the shared finalized doc to
// html1, call model.Build on that SAME doc, render it again to html2 --
// html1 must equal html2 byte-for-byte, proving the model sink does not
// perturb the HTML sink (Build is non-mutating per 02-01) and that both
// sinks are fed from a SINGLE markdown.Parse call, not two separate
// render passes.
func TestOneParseTwoSinksHTMLUnaffectedByModelBuild(t *testing.T) {
	doc, pc := markdown.Parse(multiSlideMD) // the ONE parse
	source := []byte(multiSlideMD)

	html1, err := markdown.RenderDoc(doc, source)
	if err != nil {
		t.Fatalf("RenderDoc (before Build): %v", err)
	}

	m := model.Build(doc, source, pc) // sink 2, over the SAME doc
	if m == nil || len(m.Sections) == 0 {
		t.Fatalf("model.Build produced an empty Document: %+v", m)
	}

	html2, err := markdown.RenderDoc(doc, source)
	if err != nil {
		t.Fatalf("RenderDoc (after Build): %v", err)
	}

	if html1 != html2 {
		t.Fatalf("MODEL-02 violated: HTML rendered before Build != HTML rendered after Build (Build perturbed the shared doc):\nbefore:\n%s\nafter:\n%s", html1, html2)
	}
}

// TestRenderHTMLMatchesStandaloneMarkdownRender covers Test-list case 3:
// chase.Render's HTML sink equals markdown.Render(md, nil) for the same
// source -- the entrypoint's HTML output matches the standalone renderer,
// proving RenderDoc (sink 1) is not diverging from the canonical seam.
func TestRenderHTMLMatchesStandaloneMarkdownRender(t *testing.T) {
	out, err := Render(multiSlideMD)
	if err != nil {
		t.Fatalf("chase.Render: %v", err)
	}
	want, err := markdown.Render(multiSlideMD, nil)
	if err != nil {
		t.Fatalf("markdown.Render: %v", err)
	}
	if out.HTML != want {
		t.Fatalf("chase.Render HTML != markdown.Render HTML:\nchase.Render:\n%s\nmarkdown.Render:\n%s", out.HTML, want)
	}
}

// TestRenderCSSIsProfileParameterizedAndScoped covers Test-list case 4:
// CSS comes from chase/theme.Pack parameterized by profiles/slides (the
// scoped "section" rules prove the profile fed the theme, not a
// hardcoded theme constant). profiles/slides.UnitElement() == "section";
// seam.go's Parse always enables inline-SVG, so the container chain is
// "div.marpit > svg > foreignObject".
func TestRenderCSSIsProfileParameterizedAndScoped(t *testing.T) {
	out, err := Render(multiSlideMD)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	const wantScopedRule = `div.marpit > svg > foreignObject > section {`
	if !strings.Contains(out.CSS, wantScopedRule) {
		t.Fatalf("CSS missing profile-scoped rule %q, got:\n%s", wantScopedRule, out.CSS)
	}
}
