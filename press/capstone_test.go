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

// capstone_test.go is the Objective-3 capstone: it renders a deck exercising
// EVERY battery under ALL THREE bundled themes through the PUBLIC press.Render
// API only, proving API-01's composition end-to-end.
//
// It is deliberately the EXTERNAL test package `press_test` and imports ONLY
// `github.com/AO-Cyber-Systems/eden-press/press` (plus stdlib) -- never chase/,
// profiles/, press/themes, press/sanitize, or press/math. That import list IS
// the Objective-7-begin gate proof (OBJECTIVE.md criterion 4): a downstream
// consumer -- the CLI, the exporters, Objective 7's Dart binding -- needs
// nothing but press/ to render a complete deck. If this file ever needs a
// chase// profiles/ import to compile, the public API is incomplete.
package press_test

import (
	"strings"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// bundledThemes are the three verbatim-bundled Marp theme names press.Render
// resolves opts.Theme against (CORE-01). Hardcoded (rather than imported from
// press/themes) to keep this capstone's import list press-only -- the
// Objective-7-begin gate. These names are the stable public contract.
var bundledThemes = []string{"default", "gaia", "uncover"}

// capstoneDeck exercises every battery in a single deck spanning two slides:
//   - GFM tables, task list, autolink
//   - strikethrough (~~ -> <s>)
//   - heading slug (auto heading id)
//   - shortcode + unicode emoji
//   - fenced code block (chroma highlight -> .hljs-*)
//   - inline $…$ + block $$…$$ math (native MathML) AND a \tag construct that
//     ROUTES to the go-latex fallback path (math-fallback <img>) -- \tag hits
//     the PERMANENT Chromium MathML-Core structural ceiling (no <mlabeledtr>);
//     \begin{aligned} is no longer a fallback example as of 08-04/08-03 (it now
//     renders native MathML, so it can't fingerprint the fallback path)
//   - global `size` + `math` directives
//   - a `# <!--fit-->` auto-fit heading marker
//   - speaker notes on both slides
//   - an XSS <script> payload that must be neutralized
const capstoneDeck = "---\n" +
	"marp: true\n" +
	"size: 4:3\n" +
	"math: mathml\n" +
	"---\n\n" +
	"# Cap :smile: ❤\n\n" +
	"<!-- opening note -->\n\n" +
	"~~struck~~ and `code`, see <https://example.com>.\n\n" +
	"- [x] done\n" +
	"- [ ] todo\n\n" +
	"| col a | col b |\n" +
	"| ----- | ----- |\n" +
	"| 1     | 2     |\n\n" +
	"```go\nfunc main() { _ = 1 }\n```\n\n" +
	"Inline $a^2 + b^2$ and block:\n\n" +
	"$$\\frac{1}{2}$$\n\n" +
	"Heavy (fallback-routed): $$x = 1 \\tag{1}$$\n\n" +
	"---\n\n" +
	"# <!--fit--> Big finish\n\n" +
	"<!-- closing note -->\n\n" +
	"<script>alert('xss')</script>\n"

// batteryFingerprints are the substrings that MUST appear in the composed,
// post-sanitize HTML -- one per battery -- proving each battery both ran and
// SURVIVED the last-pass sanitize. Theme-independent (the same deck body under
// any theme).
var batteryFingerprints = map[string]string{
	"inline-SVG container (seam)":   `<foreignObject`,
	"SVG viewBox case-restored":     `viewBox=`,
	"strikethrough <s> (03-03)":     `<s>struck</s>`,
	"twemoji <img> (03-04)":         `class="emoji"`,
	"GFM table":                     `<table>`,
	"heading slug":                  `id="cap-smile`,
	"highlight .hljs-* (03-05)":     `hljs-`,
	"autofit shrink wrap (03-07)":   `marp-fit-shrink`,
	"autofit fit marker (03-07)":    `data-auto-scaling="fit"`,
	"native MathML (03-06)":         `<math`,
	"block MathML (03-06)":          `display="block"`,
	"math fallback routing (03-06)": `math-fallback`,
}

// TestCapstoneAllThemesEveryBattery is the Objective-3 capstone: the deck
// renders correctly under ALL THREE themes exercising EVERY battery, each theme
// packs distinct non-empty CSS, the XSS payload is neutralized, and
// Comments/Meta are correct.
func TestCapstoneAllThemesEveryBattery(t *testing.T) {
	cssByTheme := make(map[string]string, len(bundledThemes))

	for _, themeName := range bundledThemes {
		t.Run(themeName, func(t *testing.T) {
			out, err := press.Render(capstoneDeck, press.Options{Theme: themeName})
			if err != nil {
				t.Fatalf("Render(theme=%s): %v", themeName, err)
			}

			// (a) non-empty CSS, recorded for cross-theme distinctness below.
			if strings.TrimSpace(out.CSS) == "" {
				t.Fatalf("theme %s packed empty CSS", themeName)
			}
			cssByTheme[themeName] = out.CSS

			// (b) every battery present in the composed HTML.
			for label, want := range batteryFingerprints {
				if !strings.Contains(out.HTML, want) {
					t.Errorf("theme %s: composed HTML missing %s (want %q)", themeName, label, want)
				}
			}

			// (c) XSS neutralized while battery markup survived.
			if strings.Contains(out.HTML, "<script") || strings.Contains(out.HTML, "alert('xss')") {
				t.Errorf("theme %s: <script> XSS payload survived sanitize", themeName)
			}

			// (d) Comments aggregated in document order; Meta directives
			//     materialized (marp/size/math from front matter).
			if got := strings.Join(out.Comments, "|"); got != "opening note|closing note" {
				t.Errorf("theme %s: Comments = %v, want [opening note, closing note]", themeName, out.Comments)
			}
			if out.Model == nil {
				t.Fatalf("theme %s: Output.Model is nil", themeName)
			}
			if out.Meta.Directives["marp"] != "true" {
				t.Errorf("theme %s: Meta.Directives[marp] = %q, want true", themeName, out.Meta.Directives["marp"])
			}
			if out.Meta.Directives["size"] != "4:3" {
				t.Errorf("theme %s: Meta.Directives[size] = %q, want 4:3", themeName, out.Meta.Directives["size"])
			}
			// Model.Meta and top-level Meta are the same value (convenience
			// alias contract).
			if out.Model.Meta.Directives["marp"] != out.Meta.Directives["marp"] {
				t.Error("Output.Meta is not a faithful alias of Output.Model.Meta")
			}
		})
	}

	// Each theme packs DISTINCT CSS (theme selection is real, not a no-op).
	for i := 0; i < len(bundledThemes); i++ {
		for j := i + 1; j < len(bundledThemes); j++ {
			a, b := bundledThemes[i], bundledThemes[j]
			if cssByTheme[a] == cssByTheme[b] {
				t.Errorf("themes %s and %s packed identical CSS; selection is a no-op", a, b)
			}
		}
	}
}

// TestCapstonePressOnlyConsumer proves the Objective-7-begin gate directly: a
// consumer importing ONLY press/ (see this file's package-level import block --
// there is no chase// profiles/ import) renders a COMPLETE deck, with all five
// Output fields populated from the public API alone.
func TestCapstonePressOnlyConsumer(t *testing.T) {
	out, err := press.Render(capstoneDeck, press.Options{})
	if err != nil {
		t.Fatalf("press-only consumer Render: %v", err)
	}

	if out.HTML == "" {
		t.Error("Output.HTML empty")
	}
	if out.CSS == "" {
		t.Error("Output.CSS empty")
	}
	if out.Model == nil {
		t.Error("Output.Model nil")
	}
	if out.Meta.Directives == nil {
		t.Error("Output.Meta.Directives nil")
	}
	if len(out.Comments) == 0 {
		t.Error("Output.Comments empty (expected the deck's two speaker notes)")
	}
}

// TestCapstoneZeroValueDefaultsToBundledDefault confirms the default-fallback
// resolves to the bundled "default" theme (not a bare scaffold): Options{} and
// Options{Theme:"default"} pack byte-identical CSS.
func TestCapstoneZeroValueDefaultsToBundledDefault(t *testing.T) {
	zero, err := press.Render(capstoneDeck, press.Options{})
	if err != nil {
		t.Fatalf("zero-value Render: %v", err)
	}
	explicit, err := press.Render(capstoneDeck, press.Options{Theme: "default"})
	if err != nil {
		t.Fatalf("explicit-default Render: %v", err)
	}
	if zero.CSS != explicit.CSS {
		t.Error("Options{} did not fall back to the bundled \"default\" theme (scaffold leak?)")
	}
}
