---
objective: 08-math-autofit
job: "03"
subsystem: latex2mathml-fork-converter
tags: [go, mathml, latex2mathml, converter-patch, fence, mtable, mathvariant, unicode-codepoint, tokenizer, structural-dom-test, tdd, spike, katex-parity]

# Dependency graph
requires:
  - objective: 08-math-autofit
    job: "02"
    provides: "internal/latex2mathml/{converter,walker}.go with big-op stacking + \\sqrt[n] fixes; press/math/math_test.go with 4 of 8 spike cases locked + reusable xmlElem/findElem/flatText DOM helpers. This TRD extends the SAME converter.go + math_test.go, strictly sequential after 08-02."
provides:
  - "internal/latex2mathml/converter.go: appendPrefixElement/appendPostfixElement now emit a MATCHED, content-sized stretchy fence pair for \\binom and pmatrix — opening \\lparen + CLOSING \\rparen, both carrying minsize/maxsize (was reusing \\lparen as the close AND pmatrix carried no sizing). PMOD split out and left unchanged."
  - "internal/latex2mathml/{commands,converter}.go: \\aligned registered as a MATRICES environment (ALIGNED const + MATRICES membership + CONVERSION_MAP mtable style + rl-align/mi-cell/columnspacing dispatch) — \\begin{aligned} renders <mtable> like the working \\align*, not the literal <mi>&</mi> + <mspace linebreak> fallthrough."
  - "internal/latex2mathml/tokenizer.go: a line-leading \\math…-prefixed FONT command (\\mathbb/\\mathbf/\\mathcal) is no longer silently DROPPED (Symbols-miss with no else) — it falls through to normal handling so the LOCAL_FONTS path reaches the converter."
  - "internal/latex2mathml/converter.go setFont(): a single Latin letter under double-struck/bold/script is emitted as its Unicode Mathematical-Alphanumeric CODEPOINT (base offsets + named holes ℝ U+211D, ℒ U+2112) instead of the MathML-Core-ignored mathvariant attribute."
  - "press/math/math_test.go: TestBinomFence, TestPmatrixFence, TestMatrixFenceRegression, TestAlignedTable, TestMathvariantCodepoint, and the consolidated TestSpikeCorpus locking ALL 8 PROPOSAL §11 cases as the permanent structural-regression set (criterion-1 gate)."
affects:
  - "08-04 (fallback-trigger detector) may now REMOVE aligned/align from detect.go's PNG-fallback trigger — this TRD's aligned→<mtable> fix is the landed precondition (research Pitfall 1: never remove the trigger before the fix lands)."
  - "08-05's STIX-fonts smoke covers the new codepoints (ℝ/𝐯/ℒ); an off-plane mapping would surface there as tofu — the mapping honours the Letterlike-Symbols named holes."

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Empirical-first fence diagnosis: dumped the UNPATCHED MathML for \\binom alone, pmatrix alone, and both-in-one BEFORE patching (research Open Q2). Confirmed the defect is (a) the opening '(' reused as the CLOSING fence for BOTH binom and pmatrix, plus (b) pmatrix's fence missing minsize/maxsize — and there is NO cross-contamination when both appear together."
    - "In-fork environment registration: \\aligned made behaviour-identical to \\align* by adding it to the same MATRICES list + CONVERSION_MAP mtable style + the three ALIGN-guarded convertMatrix/convertCommand sites, rather than a press/math textual pre-process. The 'right left' split manifests as per-<mtd> columnalign=\"right\"/\"left\" (the mechanism \\align* already uses), not a single mtable attribute."
    - "Root-cause at the tokenizer, one coherent codepoint site: the mathvariant styling was lost at TOKENIZATION (font command dropped), not merely mis-emitted downstream — so the fix is two contained in-fork edits (tokenizer preserves the command; setFont maps the letter to its codepoint), keeping setFont the single mathvariant→codepoint site."
    - "Structural MathML-DOM assertion (no oracle), extended: xmlElem gained an Attrs (`,any,attr`) capture so fence sizing (minsize/maxsize) and cell alignment (columnalign) are asserted structurally; findAll/parseMathML/fenceParens/moTexts/mtdAligns helpers added. No Marp MathJax-SVG byte-diff."

key-files:
  created:
    - .planning/objectives/08-math-autofit/08-03-SUMMARY.md
  modified:
    - internal/latex2mathml/converter.go
    - internal/latex2mathml/commands.go
    - internal/latex2mathml/tokenizer.go
    - press/math/math_test.go

key-decisions:
  - "Open Q2 (binom/pmatrix shared-fence) RESOLVED empirically: the visible defect is the opening '(' emitted AS the closing fence (appendPostfixElement passed \\lparen, not \\rparen) for BOTH \\binom and pmatrix, AND pmatrix's fence branch passed an empty attribute map (no minsize/maxsize). Both branches share convertAndAppendCommand but render independently — the both-in-one-expression case confirmed NO cross-contamination. Fix: postfix emits \\rparen for both; pmatrix prefix/postfix gain minsize/maxsize to match binom. Scoped to the \\lparen fence path — bmatrix/matrix untouched."
  - "PMOD split OUT of the shared \\pmatrix||PMOD fence branch and left EXACTLY as-is (both parens \\lparen, unsized). \\pmod is out of scope (the TRD scopes the change to the pmatrix/binom \\lparen path); giving it the pmatrix sizing would over-size \\pmod parens, and no test exercises \\pmod. Its pre-existing reused-open-paren is documented as untouched, not fixed here."
  - "aligned fixed IN-FORK (MATRICES registration), NOT via the press/math textual pre-process fallback — the registration was a small, contained addition (one const + MATRICES entry + CONVERSION_MAP entry + three '|| command == ALIGNED' clauses). The 'right left' split is per-<mtd> columnalign, reconciling the truth's `<mtable columnalign=\"right left\">` shorthand with the mechanism \\align* already uses. press/math/mathml.go untouched."
  - "mathvariant root cause differed from the research trace and forced a SECOND in-fork edit. Research predicted `<mi mathvariant=\"double-struck\">R</mi>` (attribute emitted, hence shim-able by MathML post-process). EMPIRICALLY the fork DROPS \\mathbb entirely at the tokenizer (Tokenize(\\mathbb{R}) = [{,R,}]) → plain `<mi>R</mi>` with NO attribute and NO codepoint. A mathvariant-attribute post-process shim therefore cannot work (no signal in the output). Root cause: tokenizer.go's `index==0 && HasPrefix(MATH)` branch looked up Symbols[match] and, on a miss (font commands are not precomposed symbols), fell through with NO else — dropping the token. Fixed at the tokenizer (preserve the command) + setFont (letter→codepoint). Kept in-fork per the TRD's stated preference; the setFont map covers only the 3 spike variants with a clear extension point (base offsets + named holes)."
  - "Vendored files keep their original (mekyt) headers — no AO-Cyber header added to internal/latex2mathml/*.go (they remain addlicense-ignored). Only press/math/math_test.go carries the Eden MIT header (already present). go.mod unchanged (existing replace only)."

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
duration: ~23min
completed: 2026-07-22
---

# Objective 08 TRD 03: latex2mathml fork — binom/pmatrix fence + aligned→mtable + mathvariant→codepoint Summary

**The remaining three converter root-causes are fixed in-fork at KaTeX-parity and ALL 8 PROPOSAL §11 spike cases are now locked as one permanent structural-regression set (`TestSpikeCorpus`) — criterion 1 closed. (1) `\binom` and pmatrix now emit a MATCHED, content-sized stretchy fence pair `( … )` — the unpatched converter reused the opening `\lparen` as the closing fence (`<mo>(…<mo>(`) for both, and pmatrix's fence additionally carried no `minsize`/`maxsize`; the empirical Open Q2 dump confirmed both defects and confirmed NO cross-contamination when a binom and a pmatrix share one expression. (2) `\begin{aligned}` renders as `<mtable>` with the `right`/`left` column-alignment split by registering `\aligned` as a MATRICES environment behaviour-identical to the working `\align*` — no more literal `<mi>&</mi>` + `<mspace linebreak>`. (3) `\mathbb{R}`/`\mathbf{v}`/`\mathcal{L}` emit the actual Unicode Mathematical-Alphanumeric codepoint (ℝ U+211D, 𝐯 U+1D42F, ℒ U+2112) instead of the MathML-Core-ignored `mathvariant` attribute. The mathvariant fix required a SECOND in-fork edit the research had not predicted: the fork was DROPPING the font command at the tokenizer (not mis-emitting an attribute downstream), so the fix landed at tokenizer.go (preserve the command) + `setFont` (letter→codepoint), keeping `setFont` the single codepoint site. Standing gates all green; the aligned fix is the landed precondition for 08-04's fallback-trigger tightening.**

## Empirical baseline (captured before any patch — research Open Q2)

A throwaway dump of the UNPATCHED converter pinned the exact defects and drove the fix mechanism:

| Input | Unpatched output (fence/shape) | Diagnosis |
|---|---|---|
| `\binom{n}{k}` | `<mo minsize maxsize>(</mo><mfrac/><mo minsize maxsize>(</mo>` | opening `(` **reused as the close**; sizing present |
| `\begin{pmatrix}…` | `<mo>(</mo><mtable/><mo>(</mo>` | opening `(` reused as the close **AND no sizing** |
| `\binom…\begin{pmatrix}…` | binom `( (` then pmatrix `( (`, independently | **no cross-contamination** — shared path, separate pairs |
| `\begin{aligned}a&=b\\c&=d` | `<mrow><mi>a</mi><mi>&</mi><mo>=</mo><mi>b</mi><mspace linebreak/>…` | literal `&` + linebreak (generic fallthrough) |
| `\begin{align*}…` (working ref) | `<mtable columnspacing="0em 2em"><mtr><mtd columnalign="right">…<mtd columnalign="left">…` | the target shape to mirror |
| `\mathbb{R}` | `<mi>R</mi>` (plain — **no** mathvariant, **no** codepoint) | `\mathbb` DROPPED at the tokenizer; research's `mathvariant="double-struck"` premise was wrong for this fork |

## What was built

### Task 1 — matched, content-sized fence for `\binom`/pmatrix (commit 7d6055c)
- `converter.go` `appendPostfixElement`: emit the matching **`\rparen`** for `\pmatrix` and `\binom`/`\dbinom`/`\tbinom` (was `\lparen`, the reused-open-paren bug). `appendPrefixElement`/`appendPostfixElement`: pmatrix's `\lparen`/`\rparen` now carry `minsize`/`maxsize` (was an empty map). `PMOD` split into its own branch and left byte-identical (out of scope).
- `math_test.go`: `TestBinomFence`, `TestPmatrixFence` (environment + `\pmatrix{…}` shorthand + the both-in-one-expression Open-Q2 independence guard), `TestMatrixFenceRegression` (bmatrix keeps `[ ]`, bare matrix has no fence). `xmlElem` extended with `Attrs` (`,any,attr`); added `findAll`/`parseMathML`/`fenceParens`/`assertSizedFencePair`/`moTexts`.

### Task 2 — aligned→mtable + mathvariant→codepoint (commit feaf2b3)
- **aligned (in-fork MATRICES registration):** `commands.go` — `ALIGNED = \aligned` const, added to `MATRICES` and to `CONVERSION_MAP` with the `\align*` mtable style. `converter.go` — three `|| command == ALIGNED` clauses (rl-alignment in `convertCommand`, the even-column `<mi>` cell insert and the `columnspacing` emit in `convertMatrix`). `\begin{aligned}` now renders `<mtable>` with per-`<mtd>` `columnalign="right"/"left"`, identical to `\align*`.
- **mathvariant (tokenizer + setFont):** `tokenizer.go` — the `index==0 && HasPrefix(MATH)` branch now only takes the precomposed-symbol path when `Symbols[match]` actually exists; a font command (`\mathbb`/`\mathbf`/`\mathcal`) falls through to normal handling instead of being dropped. `converter.go` — `setFont` maps a single Latin letter under `double-struck`/`bold`/`script` to its Mathematical-Alphanumeric codepoint (base offsets + named-hole exception maps for ℝ/ℒ etc.) and returns before the mathvariant attribute is written.
- `math_test.go`: `TestAlignedTable` (mtable, 2 rows, right/left split, no literal `&`, no `mspace linebreak`), `TestMathvariantCodepoint` (ℝ/𝐯/ℒ codepoints, no mathvariant attr). Added `mtdAligns` helper.

### Task 3 — all-8-lock + full standing gates (commit d786e78)
- `math_test.go`: `TestSpikeCorpus` — one enumerating table asserting each of the 8 cases' target MathML-DOM shape (1 sum, 2 prod, 3 lim stacked; 4 `\sqrt[3]` mroot order [08-02]; 5 binom, 6 pmatrix sized-fence; 7 aligned `<mtable>` right/left; 8 mathbb/mathbf/mathcal codepoints [08-03]), with a `len(cases)==8` guard so the permanent set cannot silently shrink. This is the criterion-1 evidence the checker greps for.
- Ran the full Objective-8 standing gate set (below) — all green.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: binom/pmatrix fence | `go test ./press/math/... -run 'TestBinomFence\|TestPmatrixFence\|TestMatrixFenceRegression\|TestMathML' -v && gofmt -l … && go build ./... && go vet ./...` | 0 | PASS |
| 2: aligned + mathvariant | `go test ./press/math/... -run 'TestAlignedTable\|TestMathvariantCodepoint\|TestMathML' -v && gofmt -l … && go build ./... && go vet ./...` | 0 | PASS |
| 3: spike corpus + gates | full standing gate set (see Validation Gate Results) | 0 | PASS |

## TDD Evidence

| Case | Phase | Command | Exit | Expected |
|---|---|---|---|---|
| binom/pmatrix (T1) | RED | `go test ./press/math/ -run 'TestBinomFence\|TestPmatrixFence'` | 1 | FAIL — closing fence is `(`, pmatrix unsized (correct) |
| binom/pmatrix (T1) | GREEN | `go test ./press/math/ -run 'TestBinomFence\|TestPmatrixFence\|TestMatrixFenceRegression' -v` | 0 | PASS — `( … )` matched + sized, no cross-contamination (correct) |
| aligned (T2A) | RED | `go test ./press/math/ -run TestAlignedTable` | 1 | FAIL — literal `<mi>&</mi>` (unparseable `&`), no mtable (correct) |
| aligned (T2A) | GREEN | `go test ./press/math/ -run TestAlignedTable -v` | 0 | PASS — `<mtable>` 2 rows, right/left split (correct) |
| mathvariant (T2B) | RED | `go test ./press/math/ -run TestMathvariantCodepoint` | 1 | FAIL — plain `<mi>R/v/L</mi>` (correct) |
| mathvariant (T2B) | GREEN | `go test ./press/math/ -run TestMathvariantCodepoint -v` | 0 | PASS — ℝ/𝐯/ℒ, no mathvariant (correct) |
| all-8 lock (T3) | GREEN | `go test ./press/math/ -run TestSpikeCorpus -v` | 0 | PASS — 8/8 sub-cases green (correct) |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l internal/latex2mathml/{converter,commands,tokenizer}.go press/math/math_test.go` | 0 | PASS (empty) |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok, no FAIL) |
| conformance | `go test ./conformance/...` | 0 | PASS (corpus/cssdiff/htmldiff/report/runner ok) |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS (fork closure chromedp-free; export is sole permitted) |
| cli-imports | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS |
| addlicense | `addlicense … -ignore 'internal/latex2mathml/**' -check .` | 0 | PASS (fork ignored; vendored headers untouched) |

## Deviations from Plan

### Auto-fixed / recorded (no user permission required)

**1. [Rule 3 - Blocking discovery] mathvariant required a tokenizer.go edit the plan did not list**
- **Found during:** Task 2B. The TRD (and research) predicted the fix was a single `converter.go` change at the `mathvariant` emission site (setFont), because the fork was believed to emit `<mi mathvariant="double-struck">R</mi>`. Empirically the fork DROPS `\mathbb`/`\mathbf`/`\mathcal` at the TOKENIZER (`Tokenize(\mathbb{R}) = [{,R,}]`), so the styled letter reaches the converter as a plain `<mi>R</mi>` with no signal at all — a `mathvariant`-attribute post-process shim (the research's alternative) would have nothing to key on.
- **Fix:** two contained in-fork edits — (a) `tokenizer.go`: the `index==0 && HasPrefix(MATH)` branch now only emits the precomposed codepoint when `Symbols[match]` exists, otherwise the font command falls through to normal handling (was silently dropped, no `else`); (b) `converter.go` `setFont`: single-letter mi → Mathematical-Alphanumeric codepoint. This keeps the fix in-fork (TRD's stated preference) with `setFont` as the single codepoint site.
- **Files modified:** `internal/latex2mathml/tokenizer.go` (+ `commands.go` for the ALIGNED constant/registration) — beyond the TRD frontmatter's `files_modified: [converter.go, math_test.go]`. All within the vendored fork, still addlicense-ignored.
- **Commit:** feaf2b3

**2. [Decision] aligned kept IN-FORK (not the press/math pre-process fallback)**
- The MATRICES registration was small and contained (one const + one MATRICES entry + one CONVERSION_MAP entry + three `|| command == ALIGNED` clauses), so the in-fork path the truth prefers was taken; the documented press/math textual pre-process fallback was not needed. `press/math/mathml.go` untouched. The `<mtable columnalign="right left">` truth manifests as per-`<mtd>` `columnalign="right"/"left"` — the exact mechanism `\align*` already uses — and the test asserts that shape.

**3. [Decision] PMOD left untouched**
- `\pmod` shares the pre/post-fix fence branch with `\pmatrix`. It was split into its own branch and left byte-identical (both parens `\lparen`, unsized): applying the pmatrix sizing would over-size `\pmod` parens, `\pmod` is out of the TRD's stated scope, and no test exercises it. Its pre-existing reused-open-paren is noted as knowingly out-of-scope, not fixed.

No other deviations — the three fixes and the 8-case lock landed as specified. No go.mod/import-path churn; detect.go and the fallback trigger untouched (that is 08-04).

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0 (the tokenizer discovery was a first-pass root-cause correction within Task 2, not a failed-gate retry)
- Must-haves verified: 5/5 — (1) binom + pmatrix matched sized fence, no reused-open, no cross-contamination; (2) `\binom`+`\pmatrix` render independently in one expression; (3) aligned → `<mtable>` right/left split, no literal `<mi>&</mi>`; (4) `\mathbb`/`\mathbf`/`\mathcal` → correct Unicode codepoints, no mathvariant attr; (5) all 8 PROPOSAL §11 cases promoted to the permanent `TestSpikeCorpus` regression set.
- Success criteria: the remaining 3 converter root-causes fixed in-fork with structural regression assertions; all 8 spike cases at KaTeX-parity in the permanent set (criterion 1 closed); aligned native-render precondition for 08-04 satisfied.
- Gate failures: None
- Blockers: None

## Commits

- `7d6055c` fix(08-03): matched content-sized fence pair for binom/pmatrix
- `feaf2b3` feat(08-03): aligned->mtable + mathvariant->Unicode codepoint
- `d786e78` test(08-03): TestSpikeCorpus locks all 8 PROPOSAL §11 spike cases

## Self-Check: PASSED

- Files verified on disk (5/5): `internal/latex2mathml/converter.go`, `internal/latex2mathml/commands.go`, `internal/latex2mathml/tokenizer.go`, `press/math/math_test.go`, `.planning/objectives/08-math-autofit/08-03-SUMMARY.md` — all FOUND.
- Commits verified in `git log` (3/3): `7d6055c`, `feaf2b3`, `d786e78` — all FOUND.
- All 3 task `<verify>` gates + the 9-gate standing set PASS; whole-repo `go build`/`go vet`/`go test`/`gofmt -l` clean; conformance + Obj-2 grep-gate green; no-chromedp & cli-imports PASS; addlicense clean with the `internal/latex2mathml/**` ignore (vendored mekyt headers untouched).
- `TestSpikeCorpus` locks EXACTLY 8 cases (guarded); no scratch/trace test files remain in press/math or internal/latex2mathml.
- Open Q2 resolved with recorded empirical evidence; the mathvariant tokenizer-drop root cause (differing from the research trace) documented with the in-fork mechanism used.
