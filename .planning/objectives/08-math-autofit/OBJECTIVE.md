---
work: feature
requirements: []
depends_on: [3, 7]
---

# Math-Fidelity Hardening + Auto-Fit Resolution  [Objective 8]

## Goal
Close the gap between "math renders without crashing" (Obj-3 baseline) and "math renders at
KaTeX-parity quality with a concrete, tested fallback rule" — and resolve the one remaining
viewer-side-JS holdout (auto-fit). Hardens CORE-08 / CORE-09 to production quality; owns no new v1 req IDs.

## Success Criteria (from ROADMAP)
1. All 8 previously-wrong math-spike cases (big-operator limit stacking, \binom/pmatrix shared-fence
   bug, \sqrt[n] argument parsing, aligned→mtable conversion, mathvariant→Unicode-codepoint mapping)
   render at KaTeX-parity quality AND are promoted into the permanent conformance-corpus regression set.
2. A concrete, testable fallback-trigger detector auto-routes \tag/\label/complex-multi-column-aligned
   to the go-latex/latex SVG/PNG path, covered by a corpus test (not manual inspection) — reflecting the
   permanent Chromium MathML-Core structural ceiling, not a bug awaiting a fix.
3. STIX Two Math bundled from the STIX-fonts-project's own OTF/WOFF2 (never a Google Fonts CDN copy),
   with a CI smoke test that renders + pixel-checks a known formula to confirm MATH-table presence.
4. The auto-fit mechanism is resolved per the decision gate and implemented with no remaining silent
   viewer-side JavaScript dependency.

## Current state (verified)
- press/math: mathml.go (latex2mathml→MathML), detect.go (baseline predicate
  `\tag|\label|\begin{aligned|align|alignat|cases|array}`, `needsFallback`), fallback.go (go-latex PNG), math.go.
- STIX Two Math OTF already bundled VERBATIM from github.com/stipub/stixfonts static_otf (NOTICE'd) — criterion 3
  font-source already satisfied; needs the MATH-table CI pixel-check (+ WOFF2 if web needs it).
- The 8 spike cases documented in .planning/research/{STACK,PITFALLS,SUMMARY,FEATURES}.md; marp-math corpus exists.

## Decision gates (resolve at planning; auto-fit likely needs the user)
1. Concrete MathML fallback-trigger rule — exact TeX constructs (\tag, \label, complex multi-column aligned)
   as a testable detection function (detect.go is the baseline — finalize + corpus-test it).
2. Auto-fit mechanism — Flutter TextPainter fit (client-side, Obj-7 binding) vs a CSS-only cqw/SVG-text spike
   (browser/PDF, Obj-5 path) vs dropping auto-fit if neither pixel-matches acceptably. NO silent viewer-side JS.

---
*Created: 2026-07-22 (/devflow:build 8)*

## Decisions (resolved by user, 2026-07-22)
1. **Math fidelity → FORK + patch latex2mathml.** Vendor the dormant `git.sr.ht/~mekyt/latex2mathml`
   into the repo and patch its converter so ALL 8 spike cases render as live MathML at KaTeX-parity
   (literal criterion-1 compliance). Accept owning the fork. Because the converter bugs are now fixed
   IN the fork, the fallback-trigger set shrinks to ONLY the permanent Chromium MathML-Core structural
   ceiling (\tag / \label / complex multi-column aligned) — NOT the converter bugs. Remove over-broad
   entries from detect.go (e.g. `cases` renders fine → drop it); re-confirm array/alignat from
   PROPOSAL.md §11. Promote all 8 as STRUCTURAL MathML-DOM assertion corpus tests (extend
   press/math/math_test.go's TestMathML pattern — do NOT byte-diff Marp's MathJax-SVG; that oracle
   doesn't exist / marp-math corpus case is permanently blocked).
2. **Auto-fit → FLUTTER-ONLY; drop JS from HTML.** REMOVE/deprecate the `--auto-fit-script` opt-in +
   the browser-fit.js splice from press/CLI (plain HTML/PDF never auto-fits; `<!--fit-->` markers stay
   emitted but inert on the web path). ADD native Flutter `TextPainter` measure-then-binary-search fit
   in bind/dart (zero JS by construction). Confirm NO viewer-side JS auto-fit remains anywhere.
3. STIX Two Math: font SOURCE already correct (stipub/stixfonts OTF, NOTICE'd). Add a CI smoke that
   renders a known formula + pixel-checks MATH-table presence (catch tofu); add a WOFF2 companion
   (official stipub build if it exists, else lossless non-subsetting convert + verify MATH-table
   survives + NOTICE the tool/version).
