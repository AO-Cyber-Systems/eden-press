---
objective: 08-math-autofit
job: "02"
subsystem: latex2mathml-fork-converter
tags: [go, mathml, latex2mathml, converter-patch, munderover, mroot, structural-dom-test, tdd, spike, katex-parity]

# Dependency graph
requires:
  - objective: 08-math-autofit
    job: "01"
    provides: "internal/latex2mathml/ — the in-repo latex2mathml fork vendored behind a go.mod `replace` directive (behaviour-identical verbatim copy). This TRD patches converter.go + walker.go DIRECTLY; the 5 converter bugs were deferred here from 08-01."
provides:
  - "internal/latex2mathml/converter.go: BIG_OPERATORS class + isDisplayMode() ancestor-walk + display-gated tag-selection that promotes <msubsup>/<msub>/<msup> to <munderover>/<munder>/<mover> for big n-ary operators and the \\lim-family — big-operator limit STACKING at KaTeX-parity (Open Q1 resolved to the tag-switch, NOT movablelimits)"
  - "internal/latex2mathml/walker.go: fixed \\sqrt[n]{radicand} read order — reads the bracket index to CLOSING_BRACKET THEN a second processToken for the base, builds ROOT with children [base, index]; the radicand is no longer lost/misassembled"
  - "press/math/math_test.go: 3 new structural MathML-DOM regression tests (TestBigOperatorStacking, TestSqrtRootChildOrder, TestMathRegressionBaseline) + reusable xmlElem/findElem/flatText DOM-parse helpers — the corpus regression pattern for the remaining spike cases"
affects:
  - "08-03 (Converter patches B) depends_on [08-02] and edits the SAME two files (converter.go, walker.go) + math_test.go — it runs STRICTLY sequential after this TRD, never in a parallel worktree. The tag-selection chain + xmlElem helpers landed here are its starting point."
  - "08-04 (fallback-trigger detector) consumes the now-correct converter output for its routing corpus; \\sum/\\prod/\\lim/\\sqrt[n] no longer need the PNG fallback."

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Display-mode detection by ancestor-walk: isDisplayMode(el) walks el.Parent() chain — nearest explicit displaystyle attribute wins (so a big operator inside \\tfrac/\\text does NOT auto-stack), else the root <math display=\"block\"> decides. This threads display context into the tag decision WITHOUT a converter-wide signature change."
    - "Structural MathML-DOM assertion (no oracle): parse the emitted <math> with encoding/xml into a recursive xmlElem, then assert element shape/child-count/child-order (e.g. <mroot> has exactly 2 children, first is the radicand). Replaces byte-diffing Marp's MathJax-SVG (marp-math is permanently blocked)."
    - "Read-order mirror: the buggy \\sqrt[n] SQRT branch was corrected to mirror the already-correct ROOT branch — read the bracketed index first, THEN a second processToken for the base — instead of reusing the '[' marker node as the radicand."
    - "Surgical, contained fork patch: two root-cause fixes to an otherwise-working converter; the 10 already-KaTeX-quality cases stay byte-identical (no tokenizer/converter rewrite)."

key-files:
  created:
    - .planning/objectives/08-math-autofit/08-02-SUMMARY.md
  modified:
    - internal/latex2mathml/converter.go
    - internal/latex2mathml/walker.go
    - press/math/math_test.go

key-decisions:
  - "Open Q1 (movablelimits vs hard munderover) RESOLVED to the munderover/munder/mover TAG-SWITCH, empirically. Two independent reasons: (1) The vendored converter renders \\sum/\\prod as an <mi> (U+2211/U+220F fall through to the <mi> branch, NOT <mo>), so movablelimits — an <mo>-only operator-dictionary attribute — does not even apply without first restructuring the operator emission. (2) Chromium's MathML-Core only produces under/over layout via munder*/mover*; it does NOT reposition msub/msup/msubsup based on movablelimits (MathML Core dropped the displaystyle-dependent script-movement that full MathML 3 had) — which is exactly why MathJax and KaTeX emit <munderover> directly for display-mode big-operator limits, and why PROPOSAL §11 corrected.png verified the munderover shape. The tag-switch is the known-good, spec-correct choice."
  - "Integrals (\\int, \\oint, \\iint, …) DELIBERATELY EXCLUDED from BIG_OPERATORS despite the TRD codebase_examples listing \\int. By TeX/KaTeX convention integral limits stay to the SIDE in display mode (side-set), only stacking under an explicit \\limits (already handled by the untouched LIMITS-modifier branch). Including them would REGRESS \\int_a^b away from KaTeX-parity — and the TRD anti-patterns explicitly warn 'so \\int accents … [do not] regress'. BIG_OPERATORS = the n-ary sum/prod/coprod/big* family; the \\lim-family is carried via the existing LIMIT set."
  - "Stacking is strictly gated on (big-operator OR \\lim-family) AND a script command (SUBSUP/SUBSCRIPT/SUPERSCRIPT) AND display style. Inline `$\\sum_i^n$` stays side-by-side <msubsup>; a limitless display \\sum gains no spurious wrapper. Both are locked by TestMathRegressionBaseline."
  - "\\sqrt[n] fix guards on the actual '[' OPENING_BRACKET: only that path takes the mroot branch; plain \\sqrt{x} (no index) falls through to the unchanged single-arg <msqrt>. A degenerate \\sqrt[]{x} (empty index) also degrades to a plain radical rather than an empty-index mroot."
  - "Vendored files keep their original (mekyt) headers — no AO-Cyber header added to internal/latex2mathml/*.go (they remain addlicense-ignored per 08-01). Only press/math/math_test.go carries the Eden MIT header (already present)."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 9
  gates_passed: 9
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true
  blockers: none

# Metrics
duration: ~9min
completed: 2026-07-22
---

# Objective 08 TRD 02: latex2mathml fork — big-operator limit stacking + `\sqrt[n]` radicand fix Summary

**The two fork-only converter root-causes are fixed at KaTeX-parity and locked with structural MathML-DOM regression tests. (1) Big n-ary operators (`\sum`/`\prod`/…) and the `\lim`-family now STACK their limits in display mode — the tag-selection promotes `<msubsup>`/`<msub>`/`<msup>` to `<munderover>`/`<munder>`/`<mover>`, gated on a new `isDisplayMode()` ancestor-walk so inline and limitless operators are untouched. (2) `\sqrt[3]{x}` now emits a well-formed `<mroot>` with exactly 2 children `[radicand x, index 3]` in MathML order — the walker's `SQRT` branch reads the bracket index THEN the radicand base (mirroring the already-correct `ROOT` branch), instead of reusing the `[` marker as the radicand and leaking the real base out as a sibling. Open Q1 (movablelimits vs tag-switch) is resolved empirically to the munderover tag-switch: the operator renders as `<mi>`, and Chromium MathML-Core only stacks via `munder*`/`mover*`, never by repositioning `msubsup` on a `movablelimits` attribute. Together these close 4 of the 8 PROPOSAL §11 spike cases (`\sum`, `\prod`, `\lim`, `\sqrt[3]`). Two surgical patches; the 10 already-correct cases stay byte-identical; all standing gates green.**

## Empirical baseline (captured before any patch)

A throwaway dump of the UNPATCHED converter established the exact RED shapes and drove the Open Q1 decision:

| Input (display) | Unpatched output (root shape) | Diagnosis |
|---|---|---|
| `\sum_{i=1}^{n}` | `<msubsup><mi>&#x2211;</mi>…</msubsup>` | side-by-side; operator is `<mi>` not `<mo>` |
| `\prod_{i=1}^{n}` | `<msubsup><mi>&#x220F;</mi>…</msubsup>` | side-by-side |
| `\lim_{x \to 0}` | `<msub><mo>lim</mo>…</msub>` | side-by-side |
| `\sqrt[3]{x}` | `<mroot><mo>[</mo><mn>3</mn></mroot><mrow><mi>x</mi></mrow>` | **radicand `x` leaked as a sibling; `[` misassembled as base** |

Because `\sum`/`\prod` emit an `<mi>` (not an `<mo>`), the `movablelimits` operator-attribute path is inapplicable without restructuring operator emission; and MathML-Core does not reposition `msubsup` on `movablelimits` regardless. → **Open Q1 = munderover/munder/mover tag-switch** (the PROPOSAL §11-verified, MathJax/KaTeX-standard shape).

## What was built

### Task 1 — big-operator limit stacking (commit 113ce04)
- `converter.go`: added the `BIG_OPERATORS` package var (`\sum \prod \coprod \bigcup \bigcap \bigsqcup \biguplus \bigvee \bigwedge \bigoplus \bigotimes \bigodot` — integrals excluded by design) and the `isDisplayMode(el *etree.Element) bool` helper (ancestor-walk: nearest explicit `displaystyle` wins, else root `<math display>`).
- Extended the tag-selection conditional (after the existing GCD / `\limits` / xarrow branches) with a display-gated branch: for a script node (`SUBSUP`/`SUBSCRIPT`/`SUPERSCRIPT`) whose `Children[0].Token` is in `BIG_OPERATORS` or the existing `LIMIT` set, promote the tag to `munderover`/`munder`/`mover`.
- `math_test.go` → `TestBigOperatorStacking` (sub-tests sum/prod/lim): structural assertion that display `\sum`/`\prod` emit `<munderover>` (and NOT `<msubsup>`) and `\lim` emits `<munder>` (NOT `<msub>`).

### Task 2 — `\sqrt[n]` radicand read-order (commit c77264e)
- `walker.go` `SQRT` branch: on an `OPENING_BRACKET`, read the bracketed index to `CLOSING_BRACKET`, THEN a SECOND `processToken` read for the actual radicand base; build `Node{Token: ROOT, Children: [base, index]}` (MathML `<mroot>` order is base first). Plain `\sqrt{x}` (no `[`) falls through to the unchanged single-arg `<msqrt>`; degenerate `\sqrt[]{x}` degrades to a plain radical.
- `math_test.go` → `TestSqrtRootChildOrder`: parses the emitted `<math>` with `encoding/xml` and asserts `<mroot>` has exactly 2 element children in order `[radicand, index]`, the base contains `x` (not the `[` marker), and `x` is not leaked as a sibling. Added reusable `xmlElem`/`findElem`/`flatText` DOM helpers.

### Task 3 — regression guard + full standing gates (commit 063acb7)
- `math_test.go` → `TestMathRegressionBaseline`: `x^2`→`<msup>`, `\frac{a}{b}`→`<mfrac>`, `\sqrt{x}`→`<msqrt>` (NOT `<mroot>`), limitless display `\sum` gains no spurious `<munderover>`/`<munder>`, and inline `\sum_{i=1}^{n}` stays side-by-side `<msubsup>` (stacking is display-only).
- Ran the full Objective-8 standing gate set (below) — all green.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: big-op stacking | `go test ./press/math/... -run 'TestMathML\|TestBigOperator' -v && gofmt -l … && go build ./... && go vet ./...` | 0 | PASS |
| 2: \sqrt[n] read order | `go test ./press/math/... -run 'TestMathML\|TestSqrt\|TestRoot' -v && gofmt -l … && go build ./... && go vet ./...` | 0 | PASS |
| 3: regression + gates | full standing gate set (see Validation Gate Results) | 0 | PASS |

## TDD Evidence

| Case | Phase | Command | Exit | Expected |
|---|---|---|---|---|
| big-op (T1) | RED | `go test ./press/math/ -run TestBigOperatorStacking` | 1 | FAIL — sum/prod emit `<msubsup>`, lim emits `<msub>` (correct) |
| big-op (T1) | GREEN | `go test ./press/math/ -run 'TestMathML\|TestBigOperator' -v` | 0 | PASS — 3/3 stacked (correct) |
| \sqrt[n] (T2) | RED | `go test ./press/math/ -run TestSqrtRootChildOrder` | 1 | FAIL — radicand = `[`, `x` leaked as sibling (correct) |
| \sqrt[n] (T2) | GREEN | `go test ./press/math/ -run 'TestMathML\|TestSqrt' -v` | 0 | PASS — `<mroot>` [x, 3] (correct) |
| regression (T3) | GREEN | `go test ./press/math/ -run 'TestMathML\|TestBigOperator\|TestSqrt\|TestMathRegressionBaseline' -v` | 0 | PASS — 4 spike groups green, baseline intact |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l internal/latex2mathml/{converter,walker}.go press/math/math_test.go` | 0 | PASS (empty) |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok, no FAIL) |
| conformance | `go test ./conformance/...` | 0 | PASS (corpus/cssdiff/htmldiff/report/runner ok) |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS (fork closure chromedp-free; export is sole permitted) |
| cli-imports | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS |
| addlicense | `addlicense … -ignore 'internal/latex2mathml/**' -check .` | 0 | PASS (fork ignored; vendored headers untouched) |

## Deviations from Plan

### Recorded decisions (no user permission required)

**1. [Decision] Open Q1 resolved to the munderover tag-switch, NOT movablelimits**
- **Found during:** Task 1 (the mandated first step). The empirical baseline dump showed `\sum`/`\prod` render as `<mi>` (not `<mo>`), so `movablelimits` (an `<mo>` operator-dictionary attribute) is inapplicable without restructuring operator emission; and MathML-Core does not reposition `msubsup` on `movablelimits` (that displaystyle-dependent movement was dropped from MathML Core — the reason MathJax/KaTeX emit `<munderover>` directly). Resolution: the known-good, PROPOSAL §11-verified `<munderover>`/`<munder>`/`<mover>` tag-switch, exactly as the TRD error_recovery prescribes for an inconclusive-movablelimits situation. The structural test locks the munderover shape.

**2. [Decision] Integrals excluded from the auto-stack class**
- **Found during:** Task 1. The TRD codebase_examples listed `\int`/`\oint` among "big operators", but by TeX/KaTeX convention integral limits stay to the SIDE in display mode (stacking only under explicit `\limits`, already handled by the untouched LIMITS-modifier branch). Including them would regress `\int_a^b` away from KaTeX-parity — and the TRD anti-patterns explicitly warn against `\int` regressing. `BIG_OPERATORS` is therefore the n-ary sum/prod/coprod/big* family only; `\int`/`\oint` are left as side-limit `<msubsup>`. Guarded by conformance (no corpus DIFF) + the display-gated predicate.

No other deviations — the two fixes and the 4 structural regression cases landed as the TRD specified. No go.mod/import-path churn; `press/math/mathml.go` and the fallback trigger untouched.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0
- Must-haves verified: 5/5 (big-op display stacking via munderover/munder; `\sqrt[3]{x}`→`<mroot>` 2-child [radicand, index]; fixes are structural-DOM-asserted; no regression to the 10 correct cases incl. inline/limitless/single-arg-root; Open Q1 resolved + recorded)
- Success criteria: 4 of 8 spike cases (`\sum`, `\prod`, `\lim`, `\sqrt[3]`) at KaTeX-parity, locked as structural assertions; contained patches to `internal/latex2mathml/{converter,walker}.go`; whole tree + conformance + standing gates green.
- Gate failures: None
- Blockers: None

## Commits

- `113ce04` fix(08-02): stack big-operator limits (\sum/\prod/\lim) in display mode
- `c77264e` fix(08-02): read \sqrt[n] radicand after the bracket index (mroot base no longer lost)
- `063acb7` test(08-02): structural MathML-DOM regression for big-op limits + \sqrt[n]

## Self-Check: PASSED

- Files verified on disk (4/4): `internal/latex2mathml/converter.go`, `internal/latex2mathml/walker.go`, `press/math/math_test.go`, `.planning/objectives/08-math-autofit/08-02-SUMMARY.md` — all FOUND.
- Commits verified in `git log` (3/3): `113ce04`, `c77264e`, `063acb7` — all FOUND.
- All 3 task `<verify>` gates + the 9-gate standing set PASS; whole-repo `go build`/`go vet`/`go test`/`gofmt -l` clean; conformance + Obj-2 grep-gate green; no-chromedp & cli-imports PASS; addlicense clean with the `internal/latex2mathml/**` ignore (vendored mekyt headers untouched).
- Open Q1 resolved with recorded evidence (empirical baseline + MathML-Core reasoning); integrals-excluded decision documented.
