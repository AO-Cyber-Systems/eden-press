---
objective: 03-press-batteries-api
job: "06"
subsystem: press-math-battery
tags: [go, goldmark, mathml, latex2mathml, go-latex, tdd, inline-parser, node-renderer, fallback, baseline]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    job: "01"
    provides: "press.Options (MathMode field consumed here), the six battery deps provisioned in go.mod (latex2mathml + codeberg.org/go-latex/latex v0.3.0 both present), the no-chromedp invariant, and the ParseWithEngine one-parse seam 03-09 folds this Option into"
provides:
  - "press/math.Option(mode string) goldmark.Option — the self-contained math battery: registers a bespoke $-trigger InlineParser + custom math AST node + routing NodeRenderer. mode \"\"/\"mathml\" enables; \"off\" is a no-op (math disabled, $x$ stays literal). 03-09 folds this into pressExtraOpts, passing press.Options.MathMode through"
  - "Native-MathML render path: common `$…$`/`$$…$$` LaTeX → `<math xmlns=… display=inline|block>…</math>` via the vendored git.sr.ht/~mekyt/latex2mathml"
  - "Construct-detection predicate needsFallback(rawLatex) — pure, independently-tested raw-LaTeX pre-scan (\\tag|\\label|\\begin{aligned|align|alignat|cases|array}) that routes heavy constructs to the fallback BEFORE any conversion"
  - "PNG-ONLY raster fallback: heavy constructs → base64 `data:image/png;base64,` `<img class=\"math-fallback\">` via codeberg.org/go-latex/latex drawtex/drawimg (drawtex has NO SVG canvas — the requirement's SVG/PNG framing is corrected to PNG for baseline)"
  - "The emitted element shapes for 03-08's sanitize allow-list: <math> + MathML children (mrow/msup/msub/mfrac/mi/mn/mo/mtext…) with xmlns+display attrs; <img class=\"math-fallback[ math-fallback-block]\" alt src>; and the <code class=\"math-error\"> panic stub"
affects:
  - "03-08 (sanitize) MUST add <math>+MathML children and the fallback <img> (data: URI src, class, alt) to the bluemonday allow-list, or the battery's output is stripped"
  - "03-09 (compose) wires press/math.Option(opts.MathMode) into NewEngine(pressExtraOpts...) so press.Render renders math in the single parse pass"
  - "Objective 8 (math hardening) owns: the 8 known latex2mathml converter bugs, KaTeX-parity, the FINAL fallback-trigger rule, and improving/replacing the go-latex raster (which today cannot render the aligned-family it is the fallback FOR — see Deviations)"

# Tech tracking
tech-stack:
  added:
    - "git.sr.ht/~sbinet/gg v0.7.0 (indirect — go-latex drawtex/drawimg raster canvas)"
    - "github.com/golang/freetype (indirect — go-latex glyph rasterization)"
    - "golang.org/x/image v0.40.0 (indirect — go-latex font faces)"
    - "golang.org/x/text v0.40.0 (indirect — transitive)"
    - "(promoted indirect→direct now that press/math imports them: codeberg.org/go-latex/latex v0.3.0, git.sr.ht/~mekyt/latex2mathml)"
  patterns:
    - "From-scratch goldmark-integration: no reusable library exists for math (unlike emoji/chroma), so the $-trigger InlineParser + custom mathNode (ast.BaseInline; Raw; Block) + routing NodeRenderer are hand-built, shaped after chase/markdown's bespoke inline/comment parsers"
    - "Route-on-raw-source-first: the NodeRenderer calls needsFallback(m.Raw) BEFORE any conversion — the cheap bounded regix decides MathML vs raster; a partial MathML conversion is NEVER inspected after the fact"
    - "Pandoc currency guard: inline $…$ requires a non-space after the opening $ and a non-space (and no digit) before/after the closing $, so `$5 and $10` backs off to literal text (return nil leaves the $ as text)"
    - "Panic-containment on fragile C-like libs: BOTH latex2mathml (Convert) and go-latex (mtex.Render) are wrapped in recover(); a converter/renderer panic degrades to a safe <code>/alt-only <img> so press.Render never crashes on untrusted input"
    - "Purity preserved: the fallback rasters to an in-memory base64 data-URI <img> — no temp files, no asset server, no network at render time — so press.Render stays a pure function"

key-files:
  created:
    - press/math/detect.go
    - press/math/detect_test.go
    - press/math/math.go
    - press/math/math_test.go
    - press/math/mathml.go
    - press/math/fallback.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Fallback is PNG-ONLY, confirmed empirically: codeberg.org/go-latex/latex/drawtex ships only drawimg (raster) + drawpdf (PDF) — NO SVG canvas. The requirement's 'SVG/PNG' framing is corrected to PNG for this baseline (research pitfall confirmed, not just assumed)"
  - "go-latex/latex/mtex PANICS (does not return an error) on constructs it cannot render — including superscripts (ast.Sup) and EVERY \\begin{…} environment. The fallback therefore wraps mtex.Render in recover() and degrades to an alt-only <img> stub. Net baseline behavior: the aligned-family heavy constructs the predicate routes to the fallback currently render as the graceful STUB (alt-only <img>), not a real raster — go-latex cannot raster them. The real base64-PNG path IS proven (TestFallback rasters \\frac{a}{b}, a construct go-latex CAN render). This is BASELINE-correct: degrade visibly, never silently break. Objective 8 owns improving/replacing the raster."
  - "Recover guard added to renderMathML too: latex2mathml's 8 known converter bugs (research Pitfall 5) are Objective 8's fix; here a converter panic degrades to <code class=\"math-error\"> rather than crashing the pure render function"
  - "MathMode honored at the Option level: Option(\"off\") returns goldmark.WithExtensions() (no extenders) so the $-parser is never registered and math is fully disabled; Option(\"\")/Option(\"mathml\") register the mathExtension. 03-09 passes press.Options.MathMode straight through"
  - "Transitive fallback deps (gg, freetype, x/image, x/text) provisioned via `go get` (the sanctioned battery-dep exception), NEVER `go mod tidy`; changes are purely ADDITIVE (go.sum has zero removals) and go-latex/latex2mathml were promoted indirect→direct because press/math now imports them"

requirements-completed: [CORE-08]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true
  blockers: none

# Metrics
duration: ~15min
completed: 2026-07-21
---

# Objective 03 TRD 06: CORE-08 BASELINE Math — MathML + construct-detection + PNG-only fallback Summary

**The riskiest battery is delivered as a self-contained `press/math` subpackage: a from-scratch goldmark integration (there is NO reusable library for math) — a bespoke `$`/`$$` InlineParser, a custom `mathNode` AST node, and a routing NodeRenderer. Common `$x^2$` / `$$\frac{a}{b}$$` render to native MathML via the vendored `latex2mathml`; a pure, test-first construct-detection predicate (`\tag`/`\label`/`\begin{aligned|align|alignat|cases|array}`) pre-scans the RAW source and routes heavy constructs to a PNG-ONLY `go-latex/latex` raster fallback embedded as a base64 `data:` `<img>`. Scope is deliberately BASELINE: render common math, degrade the known-unsupported constructs gracefully, and hand math-quality/final-fallback-rule to Objective 8.**

## What was built

### Task 1 — construct-detection predicate (test-first) + `$`/`$$` parser + math AST node (commit `76610d3`)
- `press/math/detect.go` → `needsFallback(rawLatex string) bool`: the riskiest-item spike, driven test-FIRST in isolation. A single pre-compiled regex `\\tag\b|\\label\b|\\begin\{(?:aligned|align|alignat|cases|array)\}` scans the RAW LaTeX. `\b` word-boundaries after `\tag`/`\label` mean `\tagged`/`\labelled` do NOT trip it; the environment arm requires the literal `{name}` so `\begingroup` never matches. Pure, allocation-free, no I/O.
- `press/math/math.go` → the bespoke `$`-trigger `InlineParser` (handles BOTH inline `$…$` and display `$$…$$` from the one `$` trigger byte), the custom `mathNode` (`ast.BaseInline; Raw string; Block bool`) + `KindMath`, and `Option(mode string) goldmark.Option` with the `mathExtension` Extender. Inline parsing applies the **Pandoc currency guard** (non-space after opening `$`, non-space + non-digit around the closing `$`) so `$5 and $10` stays literal; display `$$…$$` scans same-line then across the block's lines (mirroring `chase/markdown`'s comment inline parser).
- `detect_test.go` / `math_test.go` (RED→GREEN): predicate positives (`\tag{1}`, `\label{eq}`, aligned/align/alignat/cases/array) + negatives (`x^2`, `\frac`, `\sqrt`, `\sum`, and the word-boundary guards); inline/block/multi-node parse; currency back-off; `MathMode:"off"` disables math.

### Task 2 — MathML render path + PNG-only fallback + renderer wiring (commit `6116618`)
- `press/math/mathml.go` → `renderMathML(raw, block)`: calls the vendored `latex2mathml.Convert(raw, xmlns, "inline"|"block", 0)` → `<math xmlns=… display=…>…</math>`. Plus the routing `mathRenderer` NodeRenderer: on `entering`, it calls `needsFallback(m.Raw)` FIRST, then writes MathML or the fallback `<img>` (`WalkSkipChildren`). A `recover()` degrades a converter panic to `<code class="math-error">`.
- `press/math/fallback.go` → `renderFallbackIMG(raw, block)`: rasters `$raw$` via `go-latex/latex` `mtex.Render` → `drawtex/drawimg` PNG (nil fonts = built-in Go font backend, no font-file dependency) → base64 → `<img class="math-fallback[ math-fallback-block]" alt="{escaped LaTeX}" src="data:image/png;base64,…">`. `safeRasterPNG` wraps `mtex.Render` in `recover()` (it PANICS on unsupported constructs) and degrades to an alt-only `<img>` stub — never a crash, never a silent drop.
- `math.go` `Extend` now also wires the `mathRenderer` NodeRenderer alongside the `$`-parser.
- `math_test.go` (RED→GREEN): `TestMathML` (well-formed `<math>`/`<msup>`/`<mfrac>` + correct display), `TestFallback` (real base64 PNG for `\frac`; graceful stub for `\begin{aligned}`), `TestMathRender` (end-to-end routing: `$x^2$`→`<math>`, `$$\begin{aligned}…$$`→fallback `<img>`, and each does NOT emit the other).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: predicate + parser + node | `go test ./press/math/ -run 'TestNeedsFallback\|TestMathParse\|TestCurrency\|TestMathOff' -v && gofmt -l press/math/detect.go press/math/math.go` | 0 | PASS |
| 2: MathML + PNG fallback + renderer | `go test ./press/math/ -run 'TestMathML\|TestFallback\|TestMathRender' -v && go vet ./press/math/... && gofmt -l press/math/mathml.go press/math/fallback.go` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (predicate) | `go test ./press/math/ -run TestNeedsFallback` | 1 | FAIL — stub `needsFallback` returns false, 9 positives fail (correct) |
| GREEN (predicate) | `go test ./press/math/ -run TestNeedsFallback` | 0 | PASS (correct) |
| RED (parser) | `go test ./press/math/ -run 'TestMathParse\|TestCurrency\|TestMathOff'` | 1 | FAIL — stub parser emits 0 nodes, TestMathParse fails (correct) |
| GREEN (parser) | `go test ./press/math/ -run 'TestMathParse\|TestCurrency\|TestMathOff'` | 0 | PASS (correct) |
| RED (render) | `go test ./press/math/ -run 'TestMathML\|TestFallback\|TestMathRender'` | 1 | FAIL — stub renderers emit "" (`<p></p>` end-to-end) (correct) |
| GREEN (render) | `go test ./press/math/ -run 'TestMathML\|TestFallback\|TestMathRender'` | 0 | PASS — 3/3 (correct) |
| REFACTOR | (none needed — implementations are minimal + documented) | — | — |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok, incl press/math 7/7) |
| gofmt | `gofmt -l press/math/` | 0 | PASS (empty output) |
| addlicense (Go source) | `addlicense -l mit -s -c "AO Cyber Systems" -ignore … -check .` | 0 | PASS (all 6 new .go files licensed) |
| Obj-1 corpus/cssdiff | `go test ./conformance/...` | 0 | PASS (corpus, cssdiff, htmldiff, runner, report ok) |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS |
| no-chromedp invariant | `go list -deps ./press/... \| grep -c chromedp` | — | PASS (count = 0; whole-repo count = 0) |

## Deviations from Plan

### Recorded — library API surprises (per worktree_protocol: record precisely, implement smallest correct path, do NOT drop the fallback)

**1. [Recorded] go-latex/latex/mtex PANICS on unsupported constructs — including superscripts AND every `\begin{…}` environment**
- **Found during:** Task 2 (de-risked via an isolated spike before writing tests).
- **Surprise:** `mtex.Render` does not return an error for constructs it cannot render — it `panic`s. Empirically: `$\frac{a}{b}$` → 295-byte PNG OK; but `$x^2$` → `panic: unknown ast node *ast.Sup`, and `\begin{aligned}` / `\begin{cases}` / `\tag` / `\label` → `panic: unknown macro`.
- **Implication:** the aligned-family heavy constructs the predicate routes to the fallback are *exactly* the constructs go-latex cannot raster. So at BASELINE they render as the **graceful stub** (`<img class="math-fallback" alt="{raw LaTeX}">`), not a real raster image.
- **Fix (smallest correct path, fallback NOT dropped):** `safeRasterPNG` wraps `mtex.Render` in `recover()`; on panic/error/empty-output it degrades to the alt-only `<img>` stub. The **real base64-PNG path is proven** in `TestFallback` by rasterizing `\frac{a}{b}` (a construct go-latex CAN render), so the raster pipeline is wired and working — it is go-latex's construct coverage, not our plumbing, that is limited. This is BASELINE-correct (degrade visibly, never silently break). **Objective 8 owns improving/replacing the raster and the final fallback rule.**

**2. [Recorded — scope correction confirmed] Fallback is PNG-ONLY (drawtex has no SVG canvas)**
- **Confirmed empirically:** `codeberg.org/go-latex/latex/drawtex` exposes only `drawimg` (raster) and `drawpdf` (PDF) sub-canvases — no SVG. The requirement's "SVG/PNG fallback" framing is corrected to **PNG-only** for baseline, exactly as the TRD's anti_patterns warned. No SVG output was attempted.

**3. [Rule 3 — blocking dep provisioning] go-latex fallback transitive deps provisioned via `go get`**
- **Found during:** Task 2. `go-latex/latex/drawtex/drawimg` + `mtex` pull `git.sr.ht/~sbinet/gg`, `github.com/golang/freetype`, `golang.org/x/image`, `golang.org/x/text` — `gg`/`freetype` were NOT in the module cache.
- **Fix:** `go get codeberg.org/go-latex/latex/drawtex/drawimg codeberg.org/go-latex/latex/mtex` (network reachable) added them. Changes are purely **ADDITIVE** (`go.sum` has zero removals; no existing-dep version bumps); `go-latex` + `latex2mathml` were promoted indirect→direct because `press/math` now imports them. `go mod tidy` was NOT run (forbidden — it prunes the graph).

**4. [Defensive] recover guard on the MathML path too**
- `renderMathML` also wraps `latex2mathml.Convert` in `recover()` → `<code class="math-error">`. The 8 known latex2mathml converter bugs (research Pitfall 5) are Objective 8's fix; this keeps `press.Render` from crashing on untrusted input meanwhile. Simple cases (`x^2`, `\frac{a}{b}`) produce well-formed MathML, asserted in `TestMathML`.

No other deviations — the predicate, parser, node, MathML path, and PNG fallback landed as the TRD specified.

## For 03-08 (sanitize allow-list) — emitted element shapes

The battery emits exactly these shapes; 03-08's bluemonday policy MUST allow them or the output is stripped:
- **MathML:** `<math xmlns="http://www.w3.org/1998/Math/MathML" display="inline|block">` wrapping `<mrow>` and MathML leaf/container elements (`<msup>`, `<msub>`, `<mfrac>`, `<mi>`, `<mn>`, `<mo>`, `<mtext>`, `<msqrt>`, …).
- **Fallback image:** `<img class="math-fallback[ math-fallback-block]" alt="{HTML-escaped LaTeX}" src="data:image/png;base64,…">` (the `src` is absent in the graceful stub). Requires allowing `class`, `alt`, and a `data:image/png;base64` `src` on `<img>`.
- **Error stub:** `<code class="math-error">{HTML-escaped LaTeX}</code>`.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0
- Must-haves verified: 4/4
  1. goldmark engine + `Option("")` renders inline `$…$` and block `$$…$$` to native MathML via latex2mathml through a from-scratch `$`-parser + `mathNode` + NodeRenderer (`TestMathParse`, `TestMathML`, `TestMathRender`).
  2. `needsFallback` is a pure, independently-tested pre-scan predicate routing `\tag`/`\label`/aligned-family to the fallback (`TestNeedsFallback`).
  3. Fallback rasters via go-latex drawtex/drawimg to a base64 PNG `data:` `<img>`, PNG-ONLY, render stays pure (`TestFallback`) — with the go-latex construct-coverage caveat recorded in Deviations #1.
  4. Self-contained `goldmark.Option`/Extender honoring `MathMode` (`""`→mathml, `"off"`→disabled; `TestMathOff`); emitted MathML + fallback `<img>` shapes documented for 03-08.
- Gate failures: None.
- Blockers: None (both math deps present; fallback deps provisioned online, additive).

## Commits

- `76610d3` feat(03-06): construct-detection predicate + `$`/`$$` parser and math AST node
- `6116618` feat(03-06): MathML render path + PNG-only go-latex fallback + renderer wiring

## Self-Check: PASSED

- Files verified on disk (6/6): `press/math/detect.go`, `press/math/detect_test.go`, `press/math/math.go`, `press/math/math_test.go`, `press/math/mathml.go`, `press/math/fallback.go` — all FOUND.
- Commits verified in `git log` (2/2): `76610d3`, `6116618` — all FOUND.
- Both TRD `<verify>` gates PASS; whole-repo `go build`/`go vet`/`go test`/`gofmt -l press/math/` clean; Go-source addlicense clean; Obj-1 conformance + Obj-2 grep-gate green; no-chromedp count 0.
- go.mod/go.sum changes are additive-only (zero go.sum removals, no existing-dep bumps); `go mod tidy` not run.
