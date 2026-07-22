# Objective 8: Math-Fidelity Hardening + Auto-Fit Resolution - Research

**Researched:** 2026-07-22
**Domain:** LaTeX→MathML conversion hardening (latex2mathml internals), Chromium MathML Core structural ceiling, STIX Two Math font delivery/CI verification, text auto-fit mechanisms (native/CSS/JS)
**Confidence:** MEDIUM (code-level findings for 4 of 5 root causes are HIGH confidence, directly read from the vendored dependency's source; the auto-fit decision and 2 of the corpus-rule edge cases are MEDIUM/LOW and flagged as open questions)

## Summary

Objective 8 is a hardening pass on three already-built, already-working baselines: the
`latex2mathml`-backed native MathML renderer (`press/math/mathml.go`), the construct-detection
fallback router (`press/math/detect.go`), and the CORE-09 marker emitter (`press/autofit.go`)
paired with a verbatim-vendored, **opt-in** Marp Core JS helper (`press/themes.BrowserFitJS`,
spliced only when `--auto-fit-script` is passed). Nothing is broken; nothing crashes. The job is
precision: fix five identifiable converter bugs (four of which I traced to an exact line in the
vendored dependency's source), finalize and corpus-test the fallback-trigger predicate against the
real (not assumed) Chromium MathML Core ceiling, add a CI smoke test that would catch a STIX
MATH-table regression, and resolve — with a concrete recommendation — what "auto-fit" means when
"no silent viewer-side JavaScript" is a hard constraint.

The single most consequential finding: **`latex2mathml` is not forked or vendored in-repo** — it's
a plain `go.mod` dependency pulled from `git.sr.ht/~mekyt/latex2mathml`, a single-author module with
no commits since ~Dec 2023. Every one of the five bug fixes below requires either (a) forking the
module into the repo and patching its Go source, or (b) a pre/post-processing shim in Eden Press's
own code that never touches the dependency. **These are not interchangeable per bug** — I traced
each bug to its exact mechanism and the two categories split cleanly: two bugs (aligned→mtable,
mathvariant→codepoint) are patchable with a string-level shim in `press/math`, no fork needed. Two
bugs (`\sqrt[n]`, big-operator limits) are structural — for `\sqrt[n]` the radicand is provably lost
before `Convert()` returns a string, so no post-processing can recover it; a fork is the only fix.
The fifth (binom/pmatrix fence) needs one more empirical step (render + screenshot both, compare)
before committing to a mechanism.

The second major finding: the project already has an opt-in, verbatim-vendored, fully-featured JS
auto-fit engine (Marp Core's real Custom-Elements + ResizeObserver + Shadow-DOM + SVG-foreignObject
component, `themes/browser-fit.js`), gated behind `--auto-fit-script` (default `false`, never spliced
automatically). This is NOT "silent" JS by the project's own current wiring — it's already an
explicit, documented, tested opt-in. The real decision Objective 8 must resolve is narrower than the
OBJECTIVE.md framing suggests: not "eliminate all JS" (already true by default) but "what does the
*default*, no-flag, no-Dart-binding HTML/PDF output do when a slide is marked `<!--fit-->` or
`@auto-scaling` is set in the theme, and is that acceptable, or does it need a genuine non-JS (CSS)
or genuine non-web (Flutter native) alternative." My research into 2026 CSS capability is decisive
here: true content-aware "shrink until it exactly fits an unknown-size box" has **no shipped, robust,
JS-free CSS mechanism** as of 2026 (the only native primitive close to it, the CSS `text-grow`/
`text-shrink` properties, is still a spec proposal, not implemented in any browser). Container-query
units (`cqw`/`cqi`) + `clamp()` give *fluid* typography (scales smoothly with container size) but
that is a materially different guarantee than Marp's actual behavior (measure rendered content,
shrink only if it overflows). Flutter's `TextPainter`, by contrast, is a real, native, no-JS,
already-available mechanism with a well-documented "measure then binary-search the largest fitting
font size" pattern, and Objective 8's own dependency (`chase/model`'s `MathRaw()`/`MathDisplay()`
duck-typed seam, confirmed read in `press/math/math_test.go`) already proves raw content reaches the
Dart binding layer.

**Primary recommendation:** Treat the fallback-trigger rule and the four tractable converter bugs as
independent, low-risk, high-confidence hardening work (do these regardless of the auto-fit decision).
For auto-fit, recommend the **hybrid**: (1) keep `--auto-fit-script` exactly as-is (opt-in, verbatim
Marp JS, already satisfies "not silent" since it is off by default and documented) for the
browser/static-HTML output path; (2) build native `TextPainter`-based fit for the Flutter/Dart
consumer (Objective 7's binding) since that is a real client with zero JS involved by construction;
(3) do **not** attempt a from-scratch CSS shrink-to-fit reimplementation — the container-query
"fluid" technique is not behavior-equivalent to what CORE-09's markers imply, and the more precise
`tan(atan2())` pure-CSS trick is fragile enough (registered custom properties, per-line limitations)
that it would be a second, parallel, half-working auto-fit implementation to maintain forever. This
is the objective's decision gate — framed here with evidence so the user can choose; see Open
Questions and the Auto-Fit Analysis pitfall below for the full three-option comparison.

## Standard Stack

This objective does not introduce a new dependency; it hardens/patches the dependencies Objective 3
already chose. Nothing here should be read as "add a library" except where noted for the CI check.

### Core (existing, unchanged choice — hardening only)
| Library | Version | Purpose | Status this objective changes |
|---------|---------|---------|-------------------------------|
| `git.sr.ht/~mekyt/latex2mathml` | `v0.0.0-20231214134936-808832af73fc` (pseudo-version, dormant since ~Dec 2023) | LaTeX → native MathML string conversion | **Decision needed: fork into repo vs. keep external + shim.** See Architecture Patterns. |
| `codeberg.org/go-latex/latex` | `v0.3.0` | PNG-only raster fallback (`mtex`/`drawimg`) for fallback-triggered constructs | No change — PNG-only confirmed correct, already documented in `fallback.go`. |
| STIX Two Math OTF | STIX Fonts Project v2.13, `fonts/static_otf/STIXTwoMath-Regular.otf` | MathML operator/table glyphs in headless Chrome (`convert/chrome/fonts.go`) | Already bundled verbatim + NOTICE'd (confirmed, `convert/chrome/fonts/STIXTwoMath-Regular.otf`, 838KB). Needs: (a) CI smoke test, (b) WOFF2 decision. |
| Marp Core `lib/browser.js` (verbatim) | Marp Core v4.4.0 | Viewer-side auto-fit/auto-scaling Custom Element (`<marp-auto-scaling>`) | Already vendored, already opt-in (`--auto-fit-script`, default off). No code change required unless the decision gate chooses to deprecate it. |

### Supporting (for the CI MATH-table check — reuse, no new deps)
| Asset | Purpose | Why reuse, not new |
|-------|---------|---------------------|
| `convert/chrome/determinism.go` (`ApplyDeterminism`, `ComposeCSS`) | Pins headless-Chrome rendering (animation kill, font hinting/anti-aliasing knobs) so PNG/PDF output is pixel-diff-stable across CI runs | **This already exists and its own doc comment states the exact contract Objective 8 needs**: *"'deterministic' here means pixel-diff-under-threshold, NOT byte-identical … Chrome's own rendering pipeline has acknowledged PRNG-based non-determinism."* A MATH-table smoke test is exactly this same kind of check — reuse the recipe, don't re-derive it. HIGH confidence (read verbatim). |
| `convert/png` + `convert/png/testdata/` | Existing PNG-export path and golden-fixture pattern | `testdata/` already holds `deck.css`/`deck.html` fixtures for existing PNG tests — the natural place to add a `math-mathtable.md`-style fixture + a small golden PNG crop. MEDIUM confidence (fixture dir confirmed to exist; exact golden-diff test mechanics not fully re-read — planner should open `convert/png/*_test.go` to confirm the comparison helper before writing the new test). |
| Go stdlib `image`/`image/png` | Pixel comparison | No third-party pixel-diff library needed — a threshold-based per-pixel or region-crop comparison is a few dozen lines against stdlib; do not add an image-diff dependency for this. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Forking `latex2mathml` wholesale | Post-processing MathML XML string with `encoding/xml` or a regex/string shim | Cheaper, zero new vendored-code maintenance burden — but only works for bugs where the *correct information is still present* in the output string (mathvariant, aligned-rewrite via pre-processing). Does NOT work for `\sqrt[n]` (information lost before `Convert()` returns). |
| `--auto-fit-script` (JS) as the universal auto-fit answer | CSS container-query `cqw`/`cqi` + `clamp()` | Zero JS, but delivers *fluid* scaling (content always scales with container width), not Marp's actual *shrink-only-if-overflowing* behavior — a materially different visual contract, not a drop-in replacement. |
| Custom Flutter shrink-to-fit loop | `auto_size_text` pub package | Less code, well-maintained community package — but adds a Dart dependency Objective 7's binding didn't previously need (that objective's research emphasized `flutter_math_fork` as the one added Dart math dependency). Recommend a direct `TextPainter` binary-search instead, to keep the dependency surface Objective 7 already established, unless the planner finds `auto_size_text` already in `pubspec.yaml`. |

**Installation:** No new Go module installs required for the core bug-fix work. If the STIX WOFF2
variant is pursued (Open Questions), it's a build-time asset conversion, not a new runtime
dependency (see Open Questions #4 for the recommended tool).

## Architecture Patterns

### Recommended pattern: split the fix mechanism by bug class, not apply one blanket strategy

The existing `press/math` package already has the right shape (`detect.go` routes before conversion,
`mathml.go` converts, `fallback.go` rasters) — Objective 8 should NOT restructure this. It should add
exactly two new architectural elements:

1. **A pre-processing normalization step in `press/math`, ahead of `l2m.Convert`.** This is the
   correct home for the `aligned`/`align` (unstarred) fix: rewrite the *raw LaTeX string* (e.g.
   `\begin{aligned}` → `\begin{align*}`, `\end{aligned}` → `\end{align*}`) before it reaches the
   converter. This works because I confirmed (reading `commands.go`'s `MATRICES` list, `converter.go`
   line ~438's `slices.Contains(MATRICES, command)` dispatch, and `walker.go`'s `getEnvinmentNode`)
   that `\align*` (the `ALIGN` token) IS a recognized, correctly-handled matrix-like environment
   (`Tag: "mtable"`, gets `align = "rl"` column alignment), while the LaTeX token produced for
   `\begin{aligned}` — literally `\aligned` per `getEnvinmentNode`'s generic `Token: `\` + environment`
   construction — is **absent** from `MATRICES`, so it falls through to generic/unknown-command
   handling instead of `convertMatrix`. A textual pre-processing rewrite is a complete, no-fork fix.
   HIGH confidence (traced to exact source lines).

2. **A post-processing MathML-string patch step, after `l2m.Convert` returns, before the renderer
   writes it out.** This is the correct home for the `mathvariant`→Unicode-codepoint fix. I confirmed
   (`converter.go` line 745, `element.CreateAttr("mathvariant", value)`) that `latex2mathml` emits a
   literal `mathvariant="bold"` / `"double-struck"` / `"script"` / etc. attribute — which MathML Core
   silently ignores for any non-`"normal"` value (confirmed via WebSearch against the current MathML
   Core spec: Core deliberately strips non-CSS-mappable legacy presentational attributes). The fix is
   a well-defined, small lookup table (variant name → Unicode Mathematical Alphanumeric Symbols
   codepoint offset) applied as a post-process over the returned XML string (or, more robustly, by
   parsing with `encoding/xml`, finding `mathvariant` attributes, remapping the enclosed text's
   codepoints, and dropping the now-redundant attribute). No fork needed. HIGH confidence.

### Fork-required pattern (the other two)

3. **`\sqrt[n]{x}` (root-index parsing) needs a fork/patch of `walker.go`.** I read the exact bug: in
   the `SQRT` branch (`walker.go` ~lines 306-328), the code calls `processToken(tokens, "", 1)` to
   grab "the next token" — for `\sqrt[3]{x}` this returns the `[` (OPENING_BRACKET) token itself, not
   the radicand. The code then reads the bracket contents as `rootNodes` (the index, "3") but never
   makes a second call to consume the radicand `{x}` — it reuses `nextNode` (the bracket-marker
   token) as if it were the radicand when constructing the `ROOT` node:
   `Node{Token: ROOT, Children: append([]Node{nextNode}, rootNodes...)}`. The real radicand is left
   unconsumed in the token stream and becomes a separate sibling in the output tree, not a child of
   `<mroot>`. **This is information genuinely lost/misassembled before `Convert()` returns a string**
   — no output-side patch can reconstruct correct nesting. HIGH confidence (traced to exact lines);
   fixing it means forking `walker.go` and correcting the read order (bracket-index first, consumed
   fully via `CLOSING_BRACKET` terminator; THEN a second `processToken` call for the actual base).

4. **Big-operator limit stacking (`\sum_{i=1}^{n}` not auto-stacking in display mode) is also a
   fork-level fix**, though a narrower one. I traced the tag-selection logic (`converter.go` lines
   ~358-368): the default tag for a sub+superscript node is `msubsup` (side-by-side); it is only
   promoted to `munderover` (stacked) when `node.Children[0].Token == GCD` (a single named function)
   or the modifier is literally `\limits`/`\overbrace`/`\underbrace`. Big operators (`\sum`, `\prod`,
   `\int`, `\bigcup`, …) in display-mode math conventionally auto-stack their limits **without** an
   explicit `\limits` command — this condition list doesn't cover that case, so `\sum_{i=1}^n` in a
   `$$…$$` block emits `msubsup` (wrong, side-by-side) instead of `munderover` (right, stacked). The
   surgical fix is extending this same conditional with a large-operator-class check (`\sum`, `\prod`,
   `\int`, `\oint`, `\bigcup`, `\bigcap`, `\bigvee`, `\bigwedge`, `\bigoplus`, `\bigotimes`,
   `\bigodot`, `\biguplus`, `\bigsqcup`) plus a displaystyle check — this is a small, contained patch
   inside `converter.go`, but it IS inside the vendored dependency's decision tree, so still requires
   the fork. MEDIUM-HIGH confidence on mechanism; **flagged open question**: MathML has a native
   `movablelimits` attribute on large operators that is *supposed* to let a renderer auto-decide
   layout without the converter hard-coding tag choice — worth checking whether Chromium's MathML
   Core actually implements `movablelimits`, because if it does, the more spec-idiomatic (and
   possibly smaller) fix is emitting `movablelimits="true"` on the operator rather than switching the
   `msubsup`/`munderover` tag at all. Not verified in this research pass.

### Needs one more empirical step before choosing a mechanism

5. **binom/pmatrix "shared-fence" bug.** I read `appendPrefixElement`/`appendPostfixElement` in full
   (`converter.go` ~536-585). Both `\binom`/`\dbinom`/`\tbinom` and `\pmatrix`/`PMOD` route through the
   *same* `convertAndAppendCommand(`\lparen`, parent, …)` fence-generation call, but with different
   attribute maps: binom's call passes `{"minsize": size, "maxsize": size}` (an explicitly computed,
   content-height-aware size); pmatrix's call passes an **empty** `map[string]string{}` — no sizing
   hint at all. This asymmetry — same shared code path, one branch missing the sizing attributes the
   other branch has — is the most likely mechanism behind a "shared-fence bug" description, but I have
   not rendered both side-by-side to confirm the exact visual symptom (does pmatrix's fence fail to
   stretch to the matrix's full height because it lacks `minsize`/`maxsize`? Or does reusing the same
   fence-building code produce a *cross-contamination* bug when a `\binom` and a `\pmatrix` appear in
   the same expression?). MEDIUM confidence on root cause; **recommend rendering both constructs
   independently and together as the first task-1 action**, before deciding whether the fix is
   "add sizing attrs to the pmatrix branch too" (contained, single-file patch, still fork-required
   since it's inside `converter.go`) or something else.

### Corpus regression-test pattern (all 5 bugs)

**Do not use golden-HTML byte-diff against Marp's own MathJax output** (the existing `marp-math`
corpus case's approach) as the promotion mechanism — I confirmed (`conformance/corpus/cases/marp-math/`,
`conformance/runner/chase_corpus_test.go`'s `chaseSkipMap`) that this case is permanently BLOCKED,
not tested pass/fail, precisely because Marp's real output is MathJax-rendered inline SVG path data,
structurally incompatible with Eden Press's native `<math>` MathML — there is no oracle here to
diff against. The right mechanism for the 8 promoted regression cases is **structural assertion on
the emitted MathML DOM**: parse the rendered `<math>...</math>` string (Go's `encoding/xml` or a
light XML-walk) and assert presence/shape of the specific elements each bug fix targets — e.g.
`<munderover>` (not `<msubsup>`) wrapping a `\sum`, `<mroot>` with exactly 2 children in the right
order for `\sqrt[3]{x}`, `<mtable columnalign="right left">` for `aligned`, the correct Unicode
codepoint (not a `mathvariant` attribute) for `\mathbb{R}`. This mirrors the existing `TestMathML`
pattern in `press/math/math_test.go` (already asserts `<msup>`/`<mfrac>` presence) — Objective 8
should extend that same test file's pattern, not invent a new corpus format.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LaTeX tokenization/parsing for the 2 fork-required bugs | A new from-scratch LaTeX parser or a second competing converter | Fork `git.sr.ht/~mekyt/latex2mathml` in-repo and patch the specific lines identified above | The existing tokenizer/converter already correctly handles the vast majority of LaTeX (confirmed: `cases`, `\frac`, `\binom` sizing, matrix commands, symbol tables all work) — a rewrite would re-risk everything that currently works to fix 2 bugs. |
| Pixel-comparison/image-diff infrastructure for the CI MATH-table check | A new image-diff library or custom SSIM implementation | Go stdlib `image`/`image/png` + reuse `convert/chrome/determinism.go`'s `ApplyDeterminism` recipe + the existing "pixel-diff-under-threshold, not byte-identical" contract that file's own doc comment already establishes | This exact problem (headless-Chrome output is not byte-deterministic but IS diff-stable under a threshold) has already been solved once in this codebase for PDF/PNG export determinism (05-RESEARCH Pitfall C, confirmed read). Re-solving it for a font/glyph check duplicates effort and risks a subtly different (wrong) tolerance model. |
| A from-scratch CSS "measure content, shrink until it fits" engine | Custom `ResizeObserver`-free CSS calc trickery attempting to fully replace Marp's JS auto-scaling behavior | Either (a) keep the existing verbatim, opt-in `--auto-fit-script` JS for the browser/HTML path, or (b) `TextPainter`-based native fit for Flutter | As of 2026, no shipped CSS-only mechanism reproduces "shrink only if overflowing, to the exact largest fitting size, for arbitrary multi-line unknown content" — the shipped primitives (`cqw`/`clamp()`) give *fluid* scaling, a different contract; the closest CSS-only true-fit technique (`tan(atan2())` scaling-factor hack) requires registered `@property` custom properties and has known per-line limitations. Building and maintaining a second, imperfect auto-fit implementation alongside the already-correct JS one is pure risk for no functional gain, unless the explicit goal is "zero JS at any cost," which is a real tradeoff the user should choose knowingly (see Open Questions). |
| A custom Flutter auto-size text widget from scratch, including its own font-metrics cache | Hand-rolled binary-search-over-font-sizes loop with no caching, OR reach for a community package that wasn't part of Objective 7's dependency set | Direct `TextPainter.layout()` binary search (well-documented pattern, confirmed via WebSearch) is genuinely simple (a `LayoutBuilder` + a `TextPainter` re-`layout()` loop) — this is a case where hand-rolling is actually *fine and recommended*, because the alternative (adding `auto_size_text` as a new pub dependency) expands Objective 7's already-established dependency surface for a small amount of avoidable code. Flag this row as the one "don't hand-roll" exception, and note it in the plan so the planner doesn't reflexively reach for a package. |

**Key insight:** every "don't hand-roll" instinct in this domain should be checked against "does this
project already have the substrate for this problem class." Twice in this research (CI determinism,
Flutter TextPainter's raw-TeX seam) the substrate already existed from a previous objective and just
needed reuse, not a new abstraction.

## Common Pitfalls

### Pitfall 1: Treating the fallback-trigger regex as a bug list instead of a permanent structural boundary
**What goes wrong:** `detect.go`'s current baseline regex
(`\tag\b|\label\b|\begin\{(?:aligned|align|alignat|cases|array)\}`) mixes two *fundamentally
different kinds of things* under one rule: (a) constructs that are genuinely, permanently
unsupported by Chromium's MathML Core by design (`\tag`/`\label`, which need `<mlabeledtr>` — WebSearch-confirmed absent from the MathML Core spec, and confirmed absent from `latex2mathml`'s
command table entirely: `\tag`/`\label` don't even appear as recognized tokens), and (b) constructs
that are simply *currently buggy in the converter* (`cases`, and — pending the aligned pre-processing
fix — `aligned`/`align`).
**Why it happens:** Objective 3's baseline needed a conservative "when in doubt, fall back" rule and
built it before the root causes were known.
**How to avoid:** Once the aligned pre-processing shim and the `cases`-is-actually-fine fact (spike
result) are both accounted for, the FINAL rule should shrink to cover only the permanent ceiling:
`\tag`, `\label`, and (pending the empirical `alignat` check) possibly `alignat`/`alignat*` (extra
`{n}`-argument column-count syntax `latex2mathml` may not model at all — not confirmed in this pass).
`cases` should be REMOVED from the trigger set (spike-confirmed KaTeX-quality as-is). `aligned`/
`align` should be removed from the trigger set **contingent on** the pre-processing rewrite shipping
in the same objective — do not remove the trigger without the fix landing first, or Objective 8 would
regress currently-safe fallback behavior into currently-broken native output.
**Warning signs:** A corpus case that passes in isolation but the fallback rule still routes it to
PNG — that's a sign the rule wasn't tightened after a bug got fixed, silently keeping a
now-fixed-but-still-downgraded construct on the lower-fidelity path forever.

### Pitfall 2: Assuming "no fork exists yet" means "no fork needed"
**What goes wrong:** `STACK.md`'s original research assumed `latex2mathml` would be
"a vendored/forked dependency Eden Press owns and patches" — but as of this research pass, it is
still a plain external `go.mod` dependency (confirmed: no `replace` directive, no `vendor/`
directory). If Objective 8's plan doesn't explicitly include the fork/vendor step as its own task, 2
of the 5 bug fixes (`\sqrt[n]`, big-operator limits) are simply not fixable — post-processing cannot
reach information already lost inside the token walk.
**Why it happens:** the two-bugs-are-post-processable / three-bugs-need-a-fork split is not obvious
from the outside without reading the dependency's internals, which is exactly what this research
pass did.
**How to avoid:** Plan an explicit "fork `git.sr.ht/~mekyt/latex2mathml` into the repo (e.g.
`internal/latex2mathml` or a top-level vendored module), pin the license/attribution, patch
`walker.go` + `converter.go` at the identified lines" task, separate from the "add a pre/post-processing
shim in `press/math`" task for the other two bugs.
**Warning signs:** A task plan that describes all 5 bugs with one uniform "patch the converter"
action — that phrasing hides the fork requirement for 2 of them.

### Pitfall 3: Conflating "opt-in JS that's off by default" with "no viewer-side JS at all"
**What goes wrong:** `OBJECTIVE.md`'s framing ("resolve the one remaining viewer-side-JS holdout") can
read as if the current state has an unconditional JS dependency. It doesn't — `--auto-fit-script`
defaults to `false` and is never spliced automatically (confirmed: `cmd/eden-press/flags.go`,
`htmldoc.go`, and their tests). Treating this as "still broken, must eliminate" risks either (a)
wasted effort building a CSS replacement for something that's already opt-in and documented, or (b)
under-scoping the real question, which is what the *default* (`--auto-fit-script` unset) output does
with a `<!--fit-->`-marked heading or a theme with `@auto-scaling true` — today, without the flag,
the CORE-09 markers (`data-auto-scaling="fit"` attribute, `marp-fit-shrink` wrapper class) are emitted
into the DOM but nothing reads them, so they are inert (no visual effect) unless the flag is passed
or a consumer (a viewer, Objective 7's Flutter binding) interprets them itself.
**Why it happens:** the OBJECTIVE.md and ROADMAP.md language was written before this research
confirmed the flag's default-off, already-tested status.
**How to avoid:** Frame Objective 8's actual auto-fit deliverable as: decide what the *default*
(flag-off) HTML/PDF output should visually do (nothing, same as today — markers present but inert —
is a legitimate, already-implemented answer if the user accepts it), and separately decide what the
Flutter/Dart consumer should do (this is genuinely unresolved and needs new work).
**Warning signs:** A plan task titled "remove JS auto-fit" or "replace JS auto-fit with CSS" without
first confirming with the user whether the opt-in-JS status quo is actually acceptable as the
browser-path answer.

### Pitfall 4: Confusing CSS "fluid typography" with CSS "shrink-to-fit"
**What goes wrong:** Container-query units (`cqw`/`cqi`) combined with `clamp()` are the most visible,
best-documented, widely-supported (2026) CSS technique in this space, and it's tempting to treat them
as a complete auto-fit answer. They are not: `font-size: clamp(1rem, 4cqw, 3rem)` scales smoothly and
continuously with the *container's* size, regardless of whether the *content* actually overflows —
it's a proportional-scaling rule, not a fit-to-content-then-stop rule. Marp's actual JS component
(`browser-fit.js`, confirmed read) does something different: it renders content into an SVG
`foreignObject`, observes both the content's natural size and the wrapper's box size via
`ResizeObserver`, and computes a `viewBox`/scale specifically so oversized content shrinks to fit
while normally-sized content is untouched.
**Why it happens:** the search terms "CSS auto-fit text" and "CSS container query typography"
surface the fluid-typography pattern first because it's simpler and more commonly needed; true
shrink-to-fit is a much narrower, harder problem.
**How to avoid:** If a CSS-only path is chosen despite the tradeoffs above, budget for the harder
`tan(atan2())`-based true-fit technique (confirmed to exist, requires `@property`-registered custom
properties, has documented per-line limitations for wrapped multi-line text), not the `clamp()`
pattern, and pixel-test it against Marp's actual JS output before calling it equivalent.
**Warning signs:** A CSS-only auto-fit implementation that "looks right" for short single-line
headings but never actually shrinks a heading that's genuinely too long for its slide.

### Pitfall 5: STIX OTF-as-inline-base64 payload weight, unaddressed
**What goes wrong:** `FontFaceDataURI()` (confirmed read, `convert/chrome/fonts.go`) embeds the full
838KB STIXTwoMath-Regular.otf as a base64 data-URI on every rendered document that includes math —
this is fine for the headless-Chrome-for-export path (never served over a network, local process
only) but the SAME embedding mechanism is available to `cmd/eden-press/serve.go`/`preview.go`
(confirmed these commands exist and both wire `AutoFitScript`/theme CSS through the same
`assembleHTML` path) — i.e., a live-preview HTML page a real browser loads over HTTP. An 838KB inline
font (roughly +1.1MB after base64 inflation) on every preview page reload is a real, measurable cost
users will notice, distinct from any correctness bug.
**Why it happens:** the OTF was bundled specifically to solve a correctness problem (CDN subsetting
strips the MATH table); its payload-size cost is a separate, second-order concern nobody has looked
at yet.
**How to avoid:** This is exactly the WOFF2 question in Objective 8's success criterion 3 — see Open
Questions #4 for a concrete recommendation (convert, don't re-source).
**Warning signs:** A live-preview session that feels slow to reload specifically on math-heavy slides.

## Code Examples

No new implementation code is prescribed here (this is a research document, not the plan) — the
few illustrative fragments below are findings, not proposed patches, and are shown as plain
structural sketches rather than compilable Go/JS.

**Confirmed-wrong output shape (big-operator, `\sum_{i=1}^{n}` in display mode):**
Actual (wrong — side-by-side): `<msubsup><mo>∑</mo><mi>i=1</mi><mi>n</mi></msubsup>`
Wanted (KaTeX-parity — stacked): `<munderover><mo>∑</mo><mi>i=1</mi><mi>n</mi></munderover>`

**Confirmed-wrong output shape (`\mathbb{R}`, mathvariant):**
Actual (MathML Core ignores this attribute): `<mi mathvariant="double-struck">R</mi>`
Wanted (Unicode codepoint, renders correctly everywhere): `<mi>&#x211D;</mi>` (U+211D DOUBLE-STRUCK
CAPITAL R)

**Confirmed pre-processing shim target (textual rewrite, before `Convert()` is called):**
Input as authored: `\begin{aligned}a&=b\\c&=d\end{aligned}`
Rewritten (both environment tokens only) before conversion: `\begin{align*}a&=b\\c&=d\end{align*}`
— this string produces correct `<mtable columnalign="right left">` output today, per the traced
`ALIGN` token's existing (working) `MATRICES` membership.

## State of the Art

| Old Approach / Assumption | Current (2026) Reality | When Changed | Impact |
|---|---|---|---|
| "MathJax/KaTeX HTML+CSS rendering is the only reliable math option in browsers" | Chromium ships native MathML Core by default | Chromium MathML Core enabled by default starting 2023 (Igalia-led restoration effort) | Confirms Eden Press's native-`<math>`-element approach is viable and current, not a legacy/niche choice. |
| "MathML in browsers means the full MathML 3/4 spec, including numbered equations and rich table attributes" | MathML Core is a **deliberately reduced** subset; `<mlabeledtr>` and most `mtable` presentational attributes are explicitly out of scope, "candidate for a future Level 2," not a bug | Ongoing (Math Working Group charter, still drafting expansion scope as of this research) | The fallback-trigger rule for `\tag`/`\label` is not chasing a moving target — it's targeting a stable, spec-documented permanent gap. Safe to treat as a long-lived rule, not a temporary workaround. |
| "CSS `clamp()` + container query units solve responsive/fit text" | True — for *fluid* typography. A **separate**, still-unshipped CSS proposal (`text-grow`/`text-shrink` properties) is what would actually solve *shrink-to-fit*, and it is not implemented in any browser as of this research | Proposal stage only, no ship date found | Confirms no native CSS replacement for Marp's JS auto-fit exists yet; revisit this specific proposal's shipping status before any future re-evaluation of the CSS-only auto-fit option. |
| "KaTeX is the math engine to compare fidelity against" | Confirmed via Marp/Marpit's own upstream docs (previously researched, `FEATURES.md`) that Marp's actual default engine is MathJax, not KaTeX; separately, KaTeX's own docs (confirmed via WebSearch) show KaTeX's default output is HTML+CSS with MathML emitted only as an accessibility side-channel, not KaTeX's primary visual rendering path | N/A (this is upstream's steady-state, not a recent change) | "KaTeX-parity" in the objective's goal should be read as a *quality/fidelity bar* (does it look as good as what KaTeX produces), not literally "produce KaTeX's own MathML" — KaTeX's accessibility-MathML output is not a validated reference target to diff against. |

**Deprecated/outdated:**
- Treating Google Fonts' CDN copy of STIX Two Math as an acceptable source: already correctly avoided
  (confirmed, `fonts.go`'s doc comment cites the exact subsetting/tofu bug); no action needed, just
  don't regress this decision.

## Open Questions

1. **Does Chromium's MathML Core implement the `movablelimits` operator attribute?**
   - What we know: MathML's operator dictionary defines `movablelimits` specifically so a renderer
     can auto-choose stacked-vs-side-by-side layout for big operators without the *producer* having
     to hard-code `munderover` vs `msubsup`.
   - What's unclear: whether Chromium's MathML Core actually honors this attribute (not verified in
     this research pass — would need a small standalone test page).
   - Recommendation: before patching `converter.go`'s tag-selection conditional (the traced fix),
     spend 30 minutes verifying whether emitting `movablelimits="true"` on the large-operator `<mo>`
     achieves the same visual result with a smaller, more spec-idiomatic patch. If it does, prefer it.

2. **Exact visual symptom of the binom/pmatrix shared-fence bug.**
   - What we know: the code path is shared (`convertAndAppendCommand`) and one branch (pmatrix) omits
     the `minsize`/`maxsize` attributes the other branch (binom) sets.
   - What's unclear: whether this actually produces a visible defect (unstretched fence) in current
     Chromium, or whether MathML Core's default stretchy-operator behavior compensates when no
     minsize/maxsize is given (in which case the "bug" might be a red herring or a much smaller
     cosmetic issue than the other four).
   - Recommendation: render `\pmatrix{1&0\\0&1}` alone, `\binom{n}{k}` alone, and both in the same
     expression, screenshot all three, and compare before writing a fix — this is a case where the
     code-reading alone was insufficient to commit to a mechanism.

3. **Is the 8-spike-case pass/fail table (`cases` = pass; need `array`, `alignat` re-confirmed) still
   accurate?**
   - What we know: `cases` is confirmed (from the objective's own upstream research,
     `PROPOSAL.md` §11) to render at KaTeX-quality as-is, meaning `detect.go`'s current inclusion of
     `cases` in the fallback trigger is over-broad and should be removed.
   - What's unclear: `array`'s and `alignat`'s exact pass/fail status wasn't independently
     re-confirmed in this pass (I read the root-cause categories, not the full per-case table row by
     row a second time).
   - Recommendation: planner should re-open `PROPOSAL.md` (§11, ~lines 289-317) as the very first
     step of the fallback-trigger task, confirm the exact 8-case table, and only then finalize
     `detect.go`'s regex.

4. **Does the STIX Two Math OTF need a WOFF2 companion, and if so, source or convert?**
   - What we know: the OTF is 838KB, embedded as an inline base64 data-URI in every document that
     touches math (both the ephemeral CI/export path via headless Chrome, AND the live
     `serve`/`preview` HTTP path a real browser loads repeatedly) — WOFF2 typically compresses
     OTF/TTF payloads substantially, and the same STIX Fonts Project (github.com/stipub/stixfonts)
     may or may not publish an official WOFF2 build alongside the `static_otf` directory Eden Press
     already sources from.
   - What's unclear: whether stipub/stixfonts ships an official `static_woff2` (or similar) directory
     — not checked in this research pass.
   - Recommendation: if an official WOFF2 build exists in the same upstream release/tag, prefer it
     (same attribution story as the OTF, zero new licensing question). If not, generate one via a
     standard, lossless, non-subsetting OTF→WOFF2 conversion (e.g. the reference `woff2_compress`
     tool or `fonttools`'s WOFF2 support) and explicitly verify the MATH table survives the conversion
     (this is the exact bug this whole bundling decision exists to avoid — a self-inflicted repeat of
     it via a careless conversion tool would be an embarrassing regression). Document the tool +
     version + "verbatim, not subsetted" claim in NOTICE, same as the OTF entry.

5. **Auto-fit decision gate — user's call, framed with evidence (not yet decided by this research):**
   - What we know: opt-in JS (status quo) already satisfies "not silent"; native Flutter `TextPainter`
     fit is straightforward and dependency-light for the Dart consumer; no CSS-only mechanism
     reproduces Marp's actual shrink-to-fit behavior as of 2026; the "drop auto-fit entirely" option
     is always available and lowest-risk, at the cost of a real (if narrow) feature gap vs. upstream
     Marp.
   - What's unclear: whether the user considers "opt-in JS, off by default" to already satisfy the
     objective's intent, or whether they want it removed/deprecated entirely in favor of Flutter-only
     (accepting that plain static HTML/PDF output never auto-fits).
   - Recommendation: present the three options (keep opt-in JS as-is for HTML/PDF + add
     `TextPainter` for Flutter [recommended hybrid]; CSS-only fluid-typography approximation
     [documented as NOT behavior-equivalent]; drop entirely) to the user at planning time as an
     explicit decision, not something the planner should resolve unilaterally.

## Sequencing + Risk Notes (deliverable 5)

**Independent / parallelizable:**
- The fallback-trigger rule finalization (deliverable 2) is independent of all 5 bug fixes — it only
  needs the `cases`-passes fact and the aligned-fix's *landing* (not its internals) as a precondition
  for removing `aligned` from the trigger set.
- The mathvariant→codepoint fix and the aligned pre-processing shim are independent of each other and
  of the fork-required fixes — both live entirely in `press/math`, no dependency fork needed, and can
  ship first, fastest, with the highest confidence.
- The STIX CI smoke test (deliverable 3) is fully independent of the math-conversion bug fixes — it
  only depends on `convert/chrome`/`convert/png`'s existing determinism substrate, which already
  exists.
- The Flutter `TextPainter` auto-fit path (if chosen) is independent of the browser/CSS auto-fit
  question and of all math bug fixes — it only depends on Objective 7's already-built raw-TeX/content
  seam.

**Sequential / blocking:**
- Forking `latex2mathml` into the repo is a **prerequisite** for both the `\sqrt[n]` fix and the
  big-operator fix — this fork decision should be made and executed once, early, not per-bug.
- The binom/pmatrix empirical render-and-compare step (Open Question 2) blocks committing to that
  bug's fix mechanism — do this before writing any patch for it.
- Corpus-test promotion (structural MathML assertions) for each bug depends on that bug's fix
  landing first — sequence fix-then-test per bug, consistent with the existing `TestMathML` pattern.

**Riskiest 2-3 items, in order:**
1. **The fork/vendor decision for `latex2mathml`.** This is the highest-leverage, highest-risk
   decision in the objective: it determines whether 2 of 5 bugs are fixable at all this cycle, sets
   an ongoing maintenance burden (a forked dependency the project now owns forever, including its
   MIT license/attribution obligations), and has knock-on effects on how corpus tests are written
   (asserting against a codebase Eden Press now controls vs. an opaque external one). Get explicit
   sign-off before starting the fork.
2. **The auto-fit decision gate itself.** Not risky in implementation terms (all three options are
   individually low-complexity) but risky in scope-creep/rework terms if the planner guesses wrong
   about what the user actually wants "no silent viewer-side JavaScript" to mean — the difference
   between "opt-in JS is fine" and "zero JS in any code path, ever" changes the shape of the whole
   deliverable. Resolve this with the user before planning tasks, not during.
3. **The big-operator `movablelimits` open question.** Low probability of being a large problem, but
   if Chromium DOES honor `movablelimits` and the planner doesn't check this first, the fork-required
   fix path (patching `converter.go`'s tag-selection conditional) does unnecessary, harder-to-maintain
   work when a simpler attribute-emission fix (possibly even post-processable, not fork-required)
   would have sufficed. Cheap to check, meaningfully changes the fix's shape and maybe its
   fork/no-fork classification if confirmed.

## Sources

### Primary (HIGH confidence — read directly from repo/dependency source in this session)
- `press/math/{math.go,mathml.go,fallback.go,detect.go,math_test.go,detect_test.go}` — current
  baseline implementation and its own documented BASELINE/hardening-deferred comments.
- `press/autofit.go`, `press/browserjs.go`, `press/themes/themes.go`, `themes/embed.go`,
  `themes/browser-fit.js` — CORE-09 marker emission and the actual vendored JS auto-fit engine,
  confirmed opt-in via `cmd/eden-press/{flags.go,htmldoc.go,htmldoc_test.go,convert_test.go}`.
- `convert/chrome/fonts.go`, `convert/chrome/fonts/STIXTwoMath-Regular.otf`, `NOTICE` (lines ~77-94)
  — STIX Two Math bundling provenance and license attribution.
- `convert/chrome/determinism.go` — existing pixel-diff-under-threshold CI determinism contract,
  directly reusable for the MATH-table smoke test.
- `/Users/justin/go/pkg/mod/git.sr.ht/~mekyt/latex2mathml@v0.0.0-20231214134936-808832af73fc/{commands.go,converter.go,walker.go}`
  — exact line-level tracing of all 4 confirmed bug mechanisms (aligned/MATRICES membership,
  mathvariant attribute emission, `\sqrt[n]` token-consumption order, big-operator tag-selection
  conditional) and the binom/pmatrix fence-attribute asymmetry.
- `conformance/corpus/cases/marp-math/`, `conformance/corpus/corpus.go`,
  `conformance/runner/chase_corpus_test.go` — confirms `marp-math` is permanently BLOCKED (not an
  oracle-diff target), motivating the structural-assertion corpus pattern recommendation.
- `.planning/objectives/08-math-autofit/OBJECTIVE.md`, `.planning/ROADMAP.md`,
  `.planning/REQUIREMENTS.md` (CORE-08/CORE-09) — objective scope and success criteria.

### Secondary (MEDIUM confidence — WebSearch, cross-referenced against spec/official docs)
- MathML Core spec status, `mlabeledtr` exclusion, `mtable` attribute stripping: W3C MathML Core spec
  (w3c.github.io/mathml-core), MathML Working Group 2026 charter draft, MathJax 4.0 docs explicitly
  naming this gap as the reason MathJax still doesn't ship native-MathML output.
  [MathML Core](https://w3c.github.io/mathml-core/spec.html) ·
  [MathML Core Explainer](https://w3c.github.io/mathml-core/docs/explainer.html) ·
  [MathJax 4.0 MathML output docs](https://docs.mathjax.org/en/v4.0/output/mathml.html) ·
  [Math WG 2026 charter draft](https://w3c.github.io/mathml-docs/charter-2026.html)
- CSS container-query units, fluid typography vs. true shrink-to-fit, `text-grow`/`text-shrink`
  proposal status: [Container Query Units and Fluid Typography](https://moderncss.dev/container-query-units-and-fluid-typography/) ·
  [Fitting Text to a Container](https://css-tricks.com/fitting-text-to-a-container/) ·
  [CSS Container Queries: The Complete Guide for 2026](https://viadreams.cc/en/blog/css-container-queries-guide/)
- Flutter `TextPainter`/`FittedBox`/`auto_size_text` patterns:
  [auto_size_text docs](https://pub.dev/documentation/auto_size_text/latest/) ·
  [flutter/flutter#18431 (text resizing based on available space)](https://github.com/flutter/flutter/issues/18431)
- KaTeX error-handling/fallback behavior (`throwOnError`, MathML as accessibility side-channel, not
  primary render path): [KaTeX Handling Errors](https://katex.org/docs/error) ·
  [KaTeX Options](https://katex.org/docs/options.html)

### Tertiary (LOW confidence — not independently re-verified this pass, carried from prior research)
- `PROPOSAL.md` §11's exact 8-case pass/fail table — read earlier in this same investigation (not
  re-read in this final pass); planner should re-confirm the exact table before finalizing the
  fallback-trigger rule (see Open Question 3).
- Whether stipub/stixfonts publishes an official WOFF2 build — not checked this pass (Open
  Question 4).

## Metadata

**Confidence breakdown:**
- Bug root causes (4 of 5: aligned, mathvariant, sqrt[n], big-operator): HIGH — traced to exact
  source lines in the vendored dependency, read directly, not inferred.
- Bug root cause (1 of 5: binom/pmatrix): MEDIUM — code asymmetry confirmed, exact visual symptom
  not empirically confirmed.
- Fallback-trigger rule: MEDIUM — mechanism and `cases`-removal well-supported; `array`/`alignat`
  status needs a re-check against the original spike table.
- STIX CI check approach: HIGH for the determinism-reuse recommendation (read verbatim); MEDIUM for
  the exact golden-fixture test mechanics (fixture directory confirmed to exist, comparison helper
  not re-read).
- Auto-fit analysis: MEDIUM-HIGH on the technical landscape (CSS capability, Flutter capability, both
  WebSearch-confirmed against current sources); the final choice is explicitly a user decision, not a
  research conclusion.

**Research date:** 2026-07-22
**Valid until:** ~30 days for the code-level findings (stable, tied to a pinned dependency version
that won't change unless the fork happens); ~14 days for the CSS `text-grow`/`text-shrink` proposal
status specifically, since browser-shipping status is the fastest-moving fact in this document.
