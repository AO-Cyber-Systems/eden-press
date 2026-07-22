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

package math

import "regexp"

// fallbackRE is the construct-detection predicate's raw-LaTeX pre-scan,
// FINALIZED (Objective 8's RESOLVED DECISION #2, TRD 08-04) to the permanent
// Chromium MathML-Core structural ceiling now that 08-02/08-03 fixed every
// converter BUG PROPOSAL.md §11's spike battery identified. Two genuinely
// different reasons put a construct in this set (research Pitfall 1 — do not
// conflate "permanent ceiling" with "currently-buggy converter"):
//
//   - \tag / \label — equation numbering/anchoring. MathML Core has no
//     <mlabeledtr> (absent from the spec, and confirmed absent from
//     latex2mathml's command table entirely — \tag/\label are not even
//     recognized tokens). PERMANENT: no converter fix can close this gap.
//   - \begin{align} (un-starred) — amsmath's NUMBERED alignment environment.
//     Each row gets an auto-generated equation number, so \align shares
//     \tag's <mlabeledtr> gap by design. Its unnumbered siblings, \aligned
//     and the starred \align*, do NOT carry per-row numbering and are
//     removed below.
//   - \begin{alignat}{n} / \begin{alignat*}{n} — amsmath's multi-column-pair
//     alignment environments, carrying an explicit `{n}` column-count
//     argument. Re-confirmed empirically for this TRD (08-04, task 1; see
//     08-04-SUMMARY.md): latex2mathml does not model this argument at all —
//     it leaks through as a stray `<mn>n</mn>`, alignment separators degrade
//     to a literal, unparseable `<mi>&</mi>`, and NO `<mtable>` wrapper is
//     emitted, for BOTH the starred and un-starred form. Not yet promoted to
//     TestSpikeCorpus; stays in the trigger until a converter patch adds
//     MATRICES registration + column-count parsing.
//
// REMOVED from the prior BASELINE rule, because the fix has LANDED (or the
// construct was never actually broken) and is locked by TestSpikeCorpus /
// TestFallbackRouting (research Pitfall 1 — never remove a trigger before its
// fix lands):
//
//   - \begin{cases} — PROPOSAL §11: KaTeX-quality as-shipped; never broken.
//   - \begin{aligned} — 08-03 registered it as a MATRICES environment
//     (behaviour-identical to \align*); now emits a correct right/left
//     <mtable>, no literal `&`, no `mspace linebreak`.
//   - \begin{array} — already a MATRICES environment; empirically
//     re-confirmed (08-04, task 1) it emits a correct <mtable>. PROPOSAL §11's
//     20-case battery did not exercise it, but the live converter renders it
//     correctly.
//   - \begin{align*} (starred) — amsmath's un-numbered sibling of \align;
//     registered under the ALIGN constant (`\align*`) from the start,
//     confirmed correct — the prior baseline regex's exact `{align}` brace
//     match never even covered `{align*}`, so this was already implicitly
//     native; it is now also explicitly asserted (TestNeedsFallback,
//     TestFallbackRouting).
//
// A \b word boundary follows \tag and \label so \tagged / \labelled (and other
// commands that merely share the prefix) are NOT matched; the environment arm
// requires the literal `{name}` (with an optional trailing `*` for
// alignat/alignat*), so \begingroup and friends never trip it.
var fallbackRE = regexp.MustCompile(`\\tag\b|\\label\b|\\begin\{(?:align|alignat\*?)\}`)

// needsFallback reports whether rawLatex contains a construct that hits the
// permanent Chromium MathML-Core structural ceiling and must route to the PNG
// fallback instead of native MathML. It is a pure, allocation-free function of
// its input (no I/O, no globals mutated) — the routing decision is made on the
// RAW source, cheaply and bounded, BEFORE any MathML conversion is attempted.
func needsFallback(rawLatex string) bool {
	return fallbackRE.MatchString(rawLatex)
}
