# Objective 3: press/ Batteries + Public API - Research

**Researched:** 2026-07-21
**Domain:** goldmark extension wiring (Go), Marp-Core-equivalent Markdown batteries, embeddable public API design
**Confidence:** MEDIUM-HIGH (mechanism-level findings are source-verified HIGH; a few acquisition/API-surface details are flagged MEDIUM/LOW and listed in Open Questions)

This document extends `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS,SUMMARY}.md` with Objective-3-specific implementation detail. It does not repeat those docs' base recommendations (chroma v2.27.0, bluemonday v1.0.27, latex2mathml fork-and-own, etc.) except where new facts change or sharpen them.

## Summary

Objective 3 wraps `chase.Render`'s internal one-parse-two-sinks composer with batteries and public ergonomics. The critical architectural fact this research adds: **`chase.Render` cannot be reused as-is** — it calls `chase/markdown.Parse(md)`, which is hardwired to a package-level `defaultEngine` singleton with no per-call extension point. `chase/markdown.NewEngine(extra ...goldmark.Option)` *does* already accept extra options, but neither `Parse` nor `RenderDoc` accept a caller-supplied engine. **Recommendation: add one small additive export to `chase/markdown` (`ParseWithEngine`) and have `press.Render` build its own engine via `NewEngine(pressExtraOpts...)`, call `ParseWithEngine`, then render directly via `engine.Renderer().Render(...)` (already a public accessor) — bypassing `chase.Render` entirely rather than trying to parameterize it.** This is additive to chase/markdown, not a rewrite, and every existing chase.go caller is unaffected.

Three library discoveries materially change the shape of CORE-06/CORE-07 from "hand-roll from scratch" to "wire an existing official goldmark extension + supply the bounded reconciliation layer it doesn't cover":

- **`github.com/yuin/goldmark-emoji` v1.0.6** (official yuin package, MIT, already in the local module cache) supplies `:shortcode:` parsing/rendering with a built-in `Twemoji` rendering method and configurable CDN template — this is most of CORE-06's shortcode half, for free.
- **`github.com/yuin/goldmark-highlighting/v2`** (pseudo-version `v2.0.0-20230729083705-37449abec8cc`, also cached locally) wires chroma as a `renderer.NodeRenderer` for `ast.KindFencedCodeBlock` with line-highlight/style/CSS-writer plumbing already built — this is CORE-07's wiring, for free; the hljs-class remap is still bespoke.
- **`codeberg.org/go-latex/latex`'s `drawtex` package has NO SVG canvas** — only `drawimg` (raster) and `drawpdf`. CORE-08's "SVG/PNG fallback" framing needs correcting to a PNG-only baseline (see Pitfalls).

**Primary recommendation:** Build `press/` as a genuinely new composition (own engine, own two-phase call), not a wrapper around `chase.Render`; reuse `goldmark-emoji` and `goldmark-highlighting/v2` for wiring instead of hand-rolling parsers/renderers; budget CORE-08's fallback as PNG-only for this objective.

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CORE-01 | Bundle 3 official Marp themes (default/gaia/uncover) verbatim via `go:embed`, MIT headers preserved | Battery Wiring §Theme Registration; Standard Stack (marp-core v4.4.0 source paths, license header template); Pitfall "SCSS vs compiled CSS" and "github-markdown-css sub-dependency" |
| CORE-02 | `size`/`math` GLOBAL directives (Marp-Core level, above Marpit) | Architecture Patterns §Directive Plumbing — precise gap identified: `buildMeta` already passes `size`/`math` through unfiltered; `CoerceGlobal` needs two new cases so comment-form directives classify correctly |
| CORE-03 | GFM tables + strikethrough as `<s>` not `<del>`, hard breaks→`<br>` | Battery Wiring §CORE-03 override (source-verified goldmark priority mechanism); tables/hard-breaks already satisfied by existing `extension.GFM`+`WithHardWraps()` — no new work |
| CORE-04 | Heading slug ids h1-h6 | Already satisfied — `parser.WithAutoHeadingID()` baked into `NewEngine`; confirmed consumed by `chase/model/build.go`'s `headingSlug`. No battery needed; verify only |
| CORE-05 | HTML allow-list sanitization matching Marp's `xss` behaviorally; GFM disallowed-tag hand-filter; directive/comment trust boundary | Common Pitfalls §Sanitize-last, §Trust boundary re-scoped; Don't Hand-Roll (bluemonday) |
| CORE-06 | Emoji shortcodes + unicode → twemoji, no JS | Don't Hand-Roll (goldmark-emoji reuse + bespoke unicode-literal trigger); Standard Stack |
| CORE-07 | chroma v2 syntax highlighting w/ bounded hljs-class reconciliation | Don't Hand-Roll (goldmark-highlighting/v2 reuse); Battery Wiring §Chroma class remap |
| CORE-08 | Math via forked latex2mathml→MathML, construct-detection routes to go-latex/latex SVG/PNG fallback (baseline only) | Standard Stack (module paths corrected); Common Pitfalls §No-SVG-canvas; construct-detection predicate proposal |
| CORE-09 | Auto-fit markers (`# <!--fit-->`, code/math shrink), `@auto-scaling` theme-CSS-only | Standard Stack (marp-core `src/auto-scaling/` file map); scope note — this is a marker/attribute-materialization battery, not a rendering battery |
| API-01 | `press.Render(md, opts) → {HTML, CSS, Model, Comments, Meta}` | Architecture Patterns §Options/Output proposal; `Comments` field data source resolved (`model.Section.Notes`, already built by `chase/model.Build`) |
| API-02 | `press/` must not import `chromedp`, CI-enforced | Architecture Patterns §API-02 CI gate |
| API-03 | Stable Options/Output types (themes, math mode, highlight, inline-SVG, profile) | Architecture Patterns §Options/Output proposal |
</phase_requirements>

## Battery Wiring Order

**Confidence: HIGH** (source-verified against goldmark v1.8.4 and the current chase/ tree; file:line citations below).

### The composition gap and its fix

Traced call chain: `chase.Render` (`chase/chase.go:78`) → `markdown.Parse(md)` (`chase/markdown/seam.go:103`) → `defaultEngine.Parser().Parse(...)` where `defaultEngine = NewEngine()` (`seam.go:77`, no extra options). `RenderDoc(doc, source)` (`chase/markdown/renderdoc.go:49`) is *also* hardwired to `defaultEngine`. Neither takes a caller-supplied engine. `NewEngine(extra ...goldmark.Option)` (`seam.go:62`) already has the hook press/ needs — it's just never exercised with anything but zero args.

**Recommended fix (additive, non-breaking):** add `markdown.ParseWithEngine(md string, engine goldmark.Markdown) (*ast.Document, parser.Context)` to `chase/markdown` — a copy of `Parse`'s pre-seed logic (`SvgOptionsKey`, `frontMatterHeadingDividerLevels`) parameterized by `engine` instead of `defaultEngine`. `press.Render` then:
1. Builds `engine := markdown.NewEngine(pressExtraOpts...)` where `pressExtraOpts` bundles the battery `goldmark.Option`s below.
2. Calls `doc, pc := markdown.ParseWithEngine(md, engine)` — the ONE parse.
3. Renders HTML directly: `engine.Renderer().Render(&buf, source, doc)` (no new chase/markdown export needed — `Renderer()` is already public on any `goldmark.Markdown`).
4. Builds `model.Build(doc, source, pc)` — sink 2, unchanged, engine-agnostic (it only walks `*ast.Document`/`parser.Context`, never touches the engine).
5. Runs bluemonday sanitize as a **post-render string pass over the HTML from step 3** — last, always, never on the CSS or the Model.
6. Packs CSS via a `theme.ThemeSet` seeded with the go:embed'd named themes (see below), keyed by `opts.Theme` or the front-matter `theme` directive.

This avoids touching `chase.Render`/`chase.go` at all — every existing caller of `chase.Render` is unaffected; press/ is a sibling composition, not a wrapper.

### Per-battery attachment point

| Battery | Mechanism | Priority / notes |
|---|---|---|
| GFM tables, hard breaks | Already in `NewEngine`'s baked-in `extension.GFM` + `ghtml.WithHardWraps()` (`seam.go:64,66`) | No new wiring — verify only |
| Heading slugs | Already in `NewEngine`'s baked-in `parser.WithAutoHeadingID()` (`seam.go:65`) | No new wiring — verify only |
| Strikethrough `<s>` not `<del>` | Custom `renderer.NodeRenderer` registering `ast.KindStrikethrough` (from `github.com/yuin/goldmark/extension/ast`) via `renderer.WithNodeRenderers(util.Prioritized(r, N))`, **N < 500** | See mechanism proof below |
| Emoji (shortcode) | `goldmark-emoji`'s `emoji.New(emoji.WithRenderingMethod(emoji.Twemoji), emoji.WithTwemojiTemplate(...))` as a `goldmark.Extender` (registers InlineParser prio 999, NodeRenderer prio 200 internally) | Added via `goldmark.WithExtensions(...)` in `pressExtraOpts` |
| Emoji (literal unicode) | Bespoke small InlineParser, reverse-mapping unicode rune sequences → the *same* `east.Emoji` AST node `goldmark-emoji`'s renderer already knows how to render | See Don't Hand-Roll |
| Chroma highlighting | `highlighting.NewHighlighting(...)` (`goldmark-highlighting/v2`) as a `goldmark.Extender` (NodeRenderer prio 200 for `ast.KindFencedCodeBlock`) | No collision — chase/markdown's own NodeRenderer never registers `KindFencedCodeBlock` |
| Math (`$...$`/`$$...$$`) | Bespoke InlineParser (trigger `$`) + custom AST node + custom NodeRenderer, feeding forked latex2mathml or the PNG fallback | New battery, no reusable library found for the goldmark-integration layer itself |
| Sanitize | Post-render string pass (bluemonday), NOT a goldmark option at all | Must run last, see above |
| Directive/comment trust boundary | Already isolated — `chase/markdown`'s comment/directive parse path (`comment.go`) runs *before* any battery; batteries only ever see already-materialized HTML attributes (already `util.EscapeHTML`'d, confirmed in `render.go`'s `writeAttrs`/`renderSection`) or the finalized HTML string (sanitize's job) | No new trust boundary work needed beyond CORE-05's sanitize pass |

### CORE-03 exact override mechanism (source-verified)

Confirmed directly against `goldmark@v1.8.4`'s source (not training-data recall):

- `extension/strikethrough.go`: `Extend()` registers `NewStrikethroughParser()` as an `InlineParser` at priority 500 **and** `NewStrikethroughHTMLRenderer()` as a `NodeRenderer` at priority 500, both for `ast.KindStrikethrough`. The default renderer emits `<del>...</del>`.
- `renderer/renderer.go`'s `Render()`: sorts `NodeRenderers` **ascending** by priority (`r.config.NodeRenderers.Sort()`, confirmed ascending via `util/util.go`'s `PrioritizedSlice.Sort()` doc comment), then iterates the sorted slice **in reverse** (`for i := l-1; i >= 0; i--`), calling `RegisterFuncs` on each. `Register()` (used by every `NodeRenderer.RegisterFuncs`) does a plain map write: `r.nodeRendererFuncsTmp[kind] = v`.
- Net effect: the LAST `RegisterFuncs` call for a given `ast.NodeKind` wins (last map write wins), and because iteration goes from highest index (= highest priority number, since ascending-sorted) down to index 0 (= lowest priority number), **the LOWEST priority number's registration is always the last call, and therefore always wins.**

**Conclusion:** a custom `NodeRenderer` registering `ast.KindStrikethrough` via `renderer.WithNodeRenderers(util.Prioritized(customRenderer, N))` with **N < 500** overrides `<del>` with `<s>`. This is the exact same mechanism (not just an analogous pattern) by which `chase/markdown`'s own `nodeRenderer` (registered at priority 0, `chase/markdown/markdown.go`) already overrides goldmark's `renderer.DefaultRenderer` (priority 1000) for `Document`/`Section`/comment kinds. No collision: chase/markdown's own `RegisterFuncs` (`chase/markdown/render.go:55-66`) never registers `ast.KindStrikethrough`. Priority 0 (matching the existing convention) or any number below 500 is safe.

**Sources:** `/Users/justin/go/pkg/mod/github.com/yuin/goldmark@v1.8.4/extension/strikethrough.go`, `.../renderer/renderer.go`, `.../util/util.go` (local module cache, read directly).

## Standard Stack

*Extends STACK.md — only new/corrected entries below.*

### New/confirmed packages

| Library | Version | Purpose | Notes |
|---|---|---|---|
| `github.com/yuin/goldmark-emoji` | v1.0.6 (MIT) | `:shortcode:` emoji parse+render, Twemoji rendering method built in | go.mod requires goldmark v1.7.10; project is on v1.8.4 — verify compatibility with `go build`/`go test` before committing (MEDIUM confidence on cross-version compatibility, not yet build-tested) |
| `github.com/yuin/goldmark-highlighting/v2` | pseudo-version `v2.0.0-20230729083705-37449abec8cc` (MIT, Yusuke Inuzuka) | Wires chroma as `ast.KindFencedCodeBlock` NodeRenderer, line-highlight/style/CSS-writer plumbing | go.mod pins `chroma/v2 v2.2.0`; Eden Press's own `go.mod` `require`-ing chroma v2.27.0 will win under Go's MVS — no `replace` needed, just add the direct require |
| `@marp-team/marp-core` themes | tag `v4.4.0`, `themes/{default,gaia,uncover}.scss` | Source of the 3 bundled themes | **Raw source, not directly embeddable** — see Pitfalls |
| `codeberg.org/go-latex/latex` | latest (BSD-3) | Math fallback rendering | `drawtex/drawimg` (raster/PNG) and `drawtex/drawpdf` (PDF) only — **no SVG canvas subpackage exists** |

### Theme source verification (CORE-01)

Fetched directly from `github.com/marp-team/marp-core` at tag `v4.4.0`:
- `themes/default.scss`, `themes/gaia.scss`, `themes/uncover.scss` are the 3 theme sources (confirmed file listing).
- `default.scss`'s leading comment block (fetched verbatim):
  ```
  /* stylelint-disable no-descending-specificity -- ... */

  /*!
   * Marp default theme.
   *
   * @theme default
   * @author Yuki Hattori
   *
   * @auto-scaling true
   * @size 16:9 1280px 720px
   * @size 4:3 960px 720px
   */
  ```
  This confirms `@theme <name>` / `@size <name> <W>px <H>px` / `@auto-scaling <value>` metadata syntax matches `chase/theme/meta.go`'s `ParseMeta`/`metaLineRE`/`sizeLineRE` **exactly**, byte-for-byte (HIGH confidence — direct comparison).
- Browser fit/auto-scaling logic (CORE-09's client-side counterpart) lives in `src/auto-scaling/{code-block.ts, fitting-header.ts, index.ts, utils.ts}` plus `src/observer.ts`/`src/browser.ts` at the same tag (file map only — internals not read; MEDIUM confidence, directional only).
- Per-file license header template (already locked, `CONTRIBUTING.md`): preserve original Marp copyright, **year 2018**, `Marp team (marp-team@marp.app)`, via `addlicense -l mit -s -c "Marp team (marp-team@marp.app)" -y 2018 -v themes/default.scss themes/gaia.scss themes/uncover.scss themes/browser-fit.js`. Do not relabel 2026/AO Cyber Systems.

## Architecture Patterns

### Theme registration / `chase/theme.Pack` interplay (CORE-01, deliverable #2)

`chase/theme.ThemeSet` (`chase/theme/pack.go`) already supports exactly what CORE-01 needs, unused today: `NewThemeSet(unit, scaffoldCSS, advancedBackgroundCSS)` auto-registers only the reserved scaffold identity; `(*ThemeSet).Add(th *Theme)` registers any additional named theme (`Add` explicitly documents that re-adding even `ScaffoldThemeName` is allowed); `(*ThemeSet).Get(name)` and `.Pack(name, opts)` operate on any registered name. `theme.Load(cssText, unit, sizeFallback)` (`chase/theme/theme.go:59`) parses a raw theme CSS string (requiring `@theme`) into a `*Theme` ready for `Add`.

**Recommended press/ flow:** at package init or lazily, parse the three `go:embed`'d theme CSS strings via `theme.Load(css, profile.UnitElement(), profile.Sizes().ByName)` once each, `Add` them to a `ThemeSet` built the same way `chase.go`'s `packCSS` already does (`theme.NewThemeSet(p.UnitElement(), scaffoldCSS, advancedBackgroundCSS)`), then `Pack(selectedName, theme.PackOptions{InlineSVG: inlineSVG})` where `selectedName` resolves from (in order) `Options.Theme` override → front-matter `theme` directive (`Meta.Directives["theme"]`) → `theme.ScaffoldThemeName` fallback (today's chase.go behavior, i.e. "no theme selected" renders bare scaffold CSS only — matches Marp's own no-theme-selected behavior of falling back to `default`... **verify**: Marp Core's actual behavior is to fall back to the `default` theme when none is specified, not to a bare unstyled scaffold — press/ should probably default `Options.Theme` to `"default"` rather than leaving it empty, an intentional deviation from chase.go's internal-only fallback. Flag for planner as a concrete decision point, not yet resolved here.)

`@import`/`@import-theme` interplay: already fully handled by `pack.go`'s `resolveImportTheme` pass (recursive, cycle-detected) — a go:embed'd theme using `@import-theme "default"` to extend one of the other two bundled themes will resolve correctly against the same `ThemeSet` as long as all three are `Add`'d before any is `Pack`'d. No new work needed here.

### CORE-01 acquisition gap: SCSS is not the artifact to embed

Confirmed via `unpkg`/`app.unpkg.com` browsing of `@marp-team/marp-core@4.4.0`: the published npm package's `lib/` directory contains only bundled JS (`browser.cjs.js`, `browser.js`, `marp.js`, 207kB) — **no standalone compiled theme `.css` files ship in this version.** The GitHub source's `themes/*.scss` files are Sass source using `@use 'sass:meta'; @include meta.load-css('pkg:github-markdown-css/github-markdown.css')` — a Sass `pkg:` importer feature requiring Dart Sass tooling and the `github-markdown-css` npm package; Go cannot process this directly, and hand-compiling it separately risks producing CSS that doesn't byte-match what marp-core actually renders (defeating the "verbatim" requirement in CORE-01's own text).

**Recommendation:** extend the already-proven `tools/corpus-gen` Node/npm infrastructure (which already runs the real, pinned `@marp-team/marp-core@4.4.0` via `npm ci` to generate the golden conformance corpus — see NOTICE) to also dump each theme's fully-compiled CSS text via marp-core's own public API, guaranteeing byte-parity with the real renderer rather than an independently-recompiled artifact. The exact accessor (likely something under the `Marp`/`Theme`/`ThemeSet` class surface exported from `@marp-team/marp-core`'s `types/`) was not confirmed in this pass — **flagged as an Open Question**, resolvable with a short Node spike (`node -e "const {Marp}=require('@marp-team/marp-core'); console.log(Object.keys(new Marp()))"` or reading `types/*.d.ts`) before task execution.

**Attribution consequence:** the compiled `default.css` will inline `github-markdown-css` (MIT, `sindresorhus/github-markdown-css`) content — this is a *fourth* vendored-asset entry `NOTICE` needs beyond Marp's own three projects, not currently listed. Flag for the CORE-01 task's NOTICE-update checklist step.

### Directive plumbing (CORE-02, deliverable minor)

Traced precisely: `chase/model/build.go`'s `buildMeta` already materializes **every** front-matter key (including `size`/`math`) onto `Meta.Directives` unconditionally — its own doc comment states this is deliberate, citing `"size"` by name as an example of a key `chase/directive.CoerceGlobal` doesn't yet recognize but that should still survive onto the model. So **front-matter-level `size`/`math` already reach `press.Render`'s `Output.Meta` today, with zero new work.**

The actual gap: `chase/directive.CoerceGlobal` (`chase/directive/directives.go:54`) only recognizes `theme`/`headingDivider`/`style`/`lang`. `chase/model/build.go`'s `isRecognizedDirectiveKey` (used by `isNote` to classify an HTML-comment as a directive vs. a speaker note) calls `CoerceGlobal`/`CoerceLocal`/`SpotKey+CoerceLocal` — so a **comment-form** `<!-- size: 4:3 -->` or `<!-- math: mathml -->` directive would today be misclassified as a presenter note rather than a recognized directive. **CORE-02's concrete, minimal fix: add `case "size"` and `case "math"` to `CoerceGlobal`'s switch**, mirroring the existing `style`/`lang` pattern (pass raw value through, `isKnown = true`). No `chase/theme` or `chase/model` change is required beyond this.

### `press.Render` Options / Output proposal (API-01/API-03, deliverable #7)

Presented as field tables (not code, per instructions) — the planner should treat these as a strong starting proposal, not frozen.

**Options** (all fields have safe zero-value defaults so `press.Render(md, press.Options{})` works):

| Field | Type | Zero-value behavior | Notes |
|---|---|---|---|
| `Theme` | `string` | `""` → resolves to front-matter `theme:` directive, then `"default"` | See theme-fallback open decision above |
| `Profile` | `string` | `""` → `profile.Default()` (today: `"slides"`) | Matches existing `chase/profile` registry lookup-by-ID |
| `InlineSVG` | `bool` | `false` → non-SVG container chain | Mirrors `chase.go`'s existing `svgEnabled`/`PackOptions.InlineSVG`; consider deriving from front matter like `chase/markdown.Parse` does today for `SvgOptionsKey`, rather than defaulting statically |
| `MathMode` | enum-like `string` (`"mathml"` \| `"off"`, extend later) | `""` → `"mathml"` | Baseline: MathML-only + PNG fallback; no MathJax/KaTeX parity attempted (Objective 8 territory) |
| `Highlight` | `bool` | `false` (zero value) → **must decide**: does `false` mean "off" or "on by default like Marp"? Marp Core highlights fenced code by default. Recommend inverted field name `NoHighlight bool` (zero value = highlighting ON, matching Marp default) to avoid a footgun zero-value | Flag for planner |
| `HighlightStyle` | `string` (chroma style name) | `""` → a fixed default style whose token classes are pre-verified against the reconciliation table | |
| `Sanitize` | `bool` or a `*bluemonday.Policy` override | `false`/`nil` → built-in default policy always applied (CORE-05 is not optional) | Recommend NOT making sanitize skippable via zero value — security-sensitive default should be "always on" |

**Output** (extends `chase.Output`, adding `Comments`):

| Field | Type | Source |
|---|---|---|
| `HTML` | `string` | Battery-composed render, post-sanitize |
| `CSS` | `string` | `ThemeSet.Pack` output |
| `Model` | `*model.Document` | `model.Build` — unchanged |
| `Meta` | `model.Meta` | `Model.Meta` — unchanged |
| `Comments` | `[]string` or `map[int][]string` (section-keyed) | **Already exists as `model.Section.Notes` per-section** (`chase/model/document.go:100`, populated by `chase/model/build.go`'s `isNote`-gated walk). Simplest: `Output.Comments` is a flattened `[]string` across all sections in document order (mirrors `Output.Meta` being a convenience alias for `Model.Meta`); a consumer needing per-section notes already has `Model.Sections[i].Notes`. **No new AST-walk work needed — this is a pure aggregation of already-built data.** |

### API-02 CI gate

`REQUIREMENTS.md`/`OBJECTIVE.md` already specify the exact check: `go list -deps ./press/... | grep chromedp` must be empty. Concrete mechanism: a dedicated CI step (or a `go test` in `press/` itself using `os/exec` to shell out to `go list -deps ./...` and asserting no line matches `chromedp`) added to `.github/workflows/ci.yml` alongside the existing `addlicense -check` step (`CONTRIBUTING.md` already documents this file as the CI entrypoint). Recommend a `Makefile`/script target (`make check-no-chromedp`) so the same check runs identically in CI and locally. Confidence: HIGH — this is a simple, deterministic shell check with no library dependency.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| `:shortcode:` emoji parsing + Twemoji rendering | A custom InlineParser + shortcode table + regex from scratch | `github.com/yuin/goldmark-emoji` v1.0.6, `emoji.New(emoji.WithRenderingMethod(emoji.Twemoji), emoji.WithTwemojiTemplate(...))` | Official yuin package, MIT, already handles trigger detection (`:`), shortcode table (`definition.Github()`), and has a first-class `Twemoji` rendering method with configurable CDN template built in — exactly CORE-06's shortcode half |
| Literal-unicode-emoji → twemoji (the *other* half of CORE-06) | A second full parser+renderer pair | A **small** custom InlineParser that reverse-indexes `definition.Github()`'s unicode-bearing entries (`Emoji.IsUnicode()`) into a rune-sequence→`*definition.Emoji` map, then emits the *same* `east.NewEmoji(...)` AST node goldmark-emoji's own `emojiHTMLRenderer` already renders | goldmark-emoji's parser only triggers on `:` — it does not detect literal unicode emoji runes typed directly in prose. Reusing its AST node + renderer means only the trigger/lookup logic is new, not the render path |
| Chroma↔goldmark wiring (fenced-code-block NodeRenderer, line-highlight/style/CSS-writer plumbing, `{hl_lines=...}` attribute parsing) | A from-scratch `renderer.NodeRenderer` for `ast.KindFencedCodeBlock` | `github.com/yuin/goldmark-highlighting/v2`'s `highlighting.NewHighlighting(...)` | Already implements language lookup (`lexers.Get`/`lexers.Analyse`), `chroma.Coalesce`, `WithFormatOptions` passthrough to `chromahtml.Option`, per-code-block attribute overrides (`hl_lines`, `hl_style`, `nohl`, `linenos`, `linenostart`), and a `CSSWriter` hook for style-sheet extraction — CORE-07's wiring, not its class-remap, is what this buys |
| Sanitization allow-list engine | Hand-rolled tag/attribute walker | bluemonday (already STACK.md's pick) | Unchanged from STACK.md — restated here only to note it runs as a **post-render string pass**, never a goldmark option |

**Key insight:** the chroma-class↔hljs-class reconciliation (CORE-07's actual remaining bespoke work) and the literal-unicode-emoji trigger (CORE-06's actual remaining bespoke work) are now both *small, bounded* problems layered on top of official/reused wiring — not full custom extensions. This meaningfully de-risks both requirements versus treating them as from-scratch builds.

### CORE-07 chroma↔hljs remap mechanism (deliverable #4)

`goldmark-highlighting/v2`'s `Config.FormatOptions []chromahtml.Option` (set via its own `WithFormatOptions(...)`) is the injection point for `chromahtml.WithClasses(true)` (STACK.md already recommends this). Chroma's own class names are its short `TokenType` codes (e.g. `kd`, `s2`, `nv` — from `chroma.StandardTypes`), not `.hljs-*` names, and neither `chromahtml.Option` nor `goldmark-highlighting` expose a way to substitute chroma's own class-name table per-type. Two mechanisms were identified; recommend the first as more surgical:

1. **Post-format string/regex replace pass** over the `<span class="...">` HTML chroma's formatter already emits (bounded, deterministic map from each known chroma short-code to its `.hljs-*` equivalent — STACK.md's already-recorded "option (a)").
2. Bypass `chromahtml.Formatter` and iterate the `chroma.Iterator` tokens directly in a custom `WrapperRenderer`/render path, emitting `hljs-*` classes per-token — more control, more code, not needed if (1) suffices.

Bundled theme CSS (marp-core's) targets `.hljs-*` selectors for its syntax-highlighting rules (consistent with FEATURES.md's already-recorded chroma/highlight.js class-mismatch finding) — this was not independently re-verified against the actual compiled theme CSS in this pass (that CSS wasn't yet extracted per the acquisition gap above); treat as MEDIUM confidence pending the theme-CSS acquisition spike.

## Common Pitfalls

### Pitfall: CORE-08's "SVG/PNG fallback" is only PNG in practice

**What goes wrong:** planning or building against the assumption that `codeberg.org/go-latex/latex` produces SVG output for the fallback path.
**Why it happens:** the requirement text says "SVG/PNG fallback"; the library's own naming (`drawtex`) sounds general-purpose.
**How to avoid:** confirmed via `pkg.go.dev` — `drawtex` has exactly two canvas subpackages, `drawtex/drawimg` (raster image) and `drawtex/drawpdf` (PDF). **No SVG canvas exists.** Baseline CORE-08 fallback should render via `drawimg` to a raster image, then embed it as a base64 data-URI `<img>` tag (avoids a filesystem/asset-serving side channel, keeps `press.Render` a pure function). Treat true SVG output as a future enhancement requiring a hand-written `drawtex.Canvas` implementation — out of scope for this objective's "baseline" framing, likely Objective 8 (or never, if raster is judged sufficient).
**Warning signs:** any task description or acceptance criterion for CORE-08 that says "renders SVG" without a corresponding custom-Canvas task.

### Pitfall: marp-core theme SCSS is not embeddable CSS as-is

**What goes wrong:** `go:embed`-ing `themes/*.scss` verbatim and expecting it to work as plain CSS.
**Why it happens:** the files are Sass source (`@use`, `meta.load-css('pkg:...')`), not compiled CSS; `default.scss` also inlines a third-party stylesheet (`github-markdown-css`) via a Sass package importer at build time.
**How to avoid:** vendor the *compiled* CSS text (via a Node/marp-core spike, extending `tools/corpus-gen`'s existing infra — see Architecture Patterns), not the raw `.scss`. Also: the raw source's very first comment is a `stylelint-disable` directive-comment, *before* the real `/*! @theme ... */` metadata block — if any raw-scss-adjacent artifact is ever fed through `chase/theme/meta.go`'s `leadingComment()` (which grabs only the *first* `CommentGrammar` event), it will capture the wrong comment and metadata parsing will fail. The compiled CSS output should not have this issue (stylelint-disable comments are a linter-only concern, not typically preserved through Sass compilation) but this should be explicitly checked once the compiled artifact is in hand.
**Warning signs:** `theme.ParseMeta` returning "missing required @theme metadata" on an embedded theme file.

### Pitfall: sanitize must be the LAST step, over the full outer HTML string

**What goes wrong:** running bluemonday over an intermediate render (e.g. per-node, or before emoji/chroma batteries run) misses content those batteries inject, or double-processes/mangles chroma's `<span class="hljs-...">` structural markup.
**Why it happens:** it's tempting to fold sanitize in as "just another NodeRenderer," but bluemonday's API operates on a complete HTML string/`io.Writer`, not per-AST-node.
**How to avoid:** sanitize runs exactly once, as a plain Go function call over the fully-rendered `Output.HTML` string, after every NodeRenderer (including chroma/emoji/strikethrough-override) has already produced its markup and after `<div class="marpit">` wrapping is complete. This is a `press/` composition-order fact, not a goldmark-option fact — restated here because deliverable #1 explicitly asked for it and it's easy to get backwards.
**Warning signs:** sanitized output missing chroma's highlighting spans, or emoji/twemoji `<img>` tags stripped because the allow-list wasn't updated for them.

### Pitfall: the "directive/comment trust boundary" is narrower than it sounds

**What goes wrong:** assuming CORE-05's adversarial round-trip test needs to cover the directive *parsing* path itself as an XSS vector.
**Why it happens:** PITFALLS.md (project-level) flags "the always-on directive/comment parse path is its own trust boundary," which sounds like it needs its own sanitizer.
**How to avoid:** traced precisely — every directive-derived value that reaches HTML output today goes through `chase/markdown/render.go`'s `writeAttrs`/`renderSection`, which already call `util.EscapeHTML` on every attribute value. The comment/directive parse path's real risk is (a) values that end up in a `href`/`src`-like attribute where escaping alone doesn't stop a `javascript:` URI scheme (bluemonday's URL-scheme policy is the actual backstop, not a directive-layer change), and (b) the GFM disallowed-tag list goldmark's GFM extension doesn't filter (`<script>`, `<iframe>`, `<style>`, `<textarea>`, `<title>`, `<xmp>`, `<noembed>`, `<noframes>`, `<plaintext>` — already enumerated in PITFALLS.md). Both are sanitize-layer concerns, not directive-layer ones. The adversarial test suite should target the final `Output.HTML` string, exercising both vectors, not the directive parser in isolation.
**Warning signs:** writing sanitizer tests that mock/bypass the directive layer instead of running full `press.Render`.

### Pitfall: `model.Section.Notes` already exists — don't rebuild comment/note extraction

**What goes wrong:** implementing a new AST walk in `press/` to extract speaker notes for `Output.Comments`, duplicating `chase/model.Build`'s existing `isNote`-gated logic.
**Why it happens:** `chase/markdown/render.go`'s `renderComment` returns `ast.WalkSkipChildren` unconditionally, making it look like *all* comment content — directive or note — is discarded with no distinction retained anywhere.
**How to avoid:** `chase/model.Build` already walks the same doc a second time (structurally, not a second *parse*) and populates `Section.Notes []string` via `isNote`, which correctly distinguishes a recognized directive comment (already absorbed into `Section.Attrs`) from a genuine presenter note. `Output.Comments` should be a pure aggregation of `Model.Sections[*].Notes`, not new extraction logic.
**Warning signs:** a new `ast.Walk` over `doc` inside `press/` duplicating `isNote`/`isRecognizedDirectiveKey`.

## Sequencing / Parallelism

**Independent, parallelizable batteries** (each touches a distinct NodeKind/priority slot, no shared state beyond the shared engine-construction option list):
- Emoji (shortcode + unicode-literal)
- Chroma highlighting
- Strikethrough `<s>` override
- Heading-slug verification (likely zero new code, just a test)

**Must-land-before-compose (sequential dependencies):**
1. `markdown.ParseWithEngine` export (chase/markdown change) — everything else in press/ depends on having an engine-parameterized parse/render seam at all.
2. Theme go:embed + compiled-CSS acquisition spike — CORE-01 blocks the "exercise all 3 themes" success criterion and should start early given the acquisition-mechanism uncertainty flagged above.
3. Math battery (CORE-08) — the construct-detection predicate and the PNG-only fallback correction should be decided before implementation starts, since they change the shape of the math NodeRenderer.
4. Sanitize (CORE-05) must land after every HTML-producing battery is wired (it's the literal last step), but its *policy construction* (allow-list design, GFM tag list, adversarial test corpus) can be designed in parallel with the others.
5. Options/Output struct finalization (API-01/API-03) gates Objective 7 (Dart binding) — should be locked once, not iterated per-battery, to avoid API churn the binding would otherwise have to chase.

**Riskiest items (flag for planner):**
1. **CORE-01's compiled-CSS acquisition mechanism** — the exact marp-core JS API to extract byte-parity compiled theme CSS is unconfirmed (Open Question below); if no clean API exists, this becomes a larger side-quest (Sass toolchain replication) than a single task should absorb.
2. **CORE-08's math battery** — has no reusable goldmark-integration library at all (unlike emoji/chroma); it's the one battery that's still a genuine from-scratch InlineParser+NodeRenderer build, on top of an already-flagged-as-risky upstream (dormant `latex2mathml` fork, per STACK.md/PITFALLS.md Pitfall 5).
3. **goldmark-emoji's `go 1.22`/goldmark v1.7.10 requirement vs. this repo's goldmark v1.8.4** — not build-tested in this research pass; if there's an actual API break between those two goldmark versions (not just a version-floor bump), the "reuse, don't hand-roll" recommendation for CORE-06 could partially collapse. Should be the very first thing verified when work starts (a five-minute `go get` + build spike).

## Open Questions

1. **Exact marp-core API for compiled theme CSS extraction**
   - What we know: the npm package doesn't ship precompiled per-theme `.css` files at v4.4.0; the project already has proven Node/npm infra (`tools/corpus-gen`) driving real marp-core for corpus generation.
   - What's unclear: the exact JS accessor (likely on the `Marp`/theme-set class exported from `@marp-team/marp-core`) to pull a single theme's fully-compiled CSS text.
   - Recommendation: a short Node spike (`require('@marp-team/marp-core')`, inspect exports/`types/*.d.ts`) before the CORE-01 task begins; budget it as its own small task, not a sub-step of "embed the themes."

2. **Theme fallback-when-unspecified behavior**
   - What we know: `chase.go`'s internal `packCSS` only ever packs the bare scaffold theme (no named-theme selection at all, by design — that's Objective 3's job to add).
   - What's unclear: whether `press.Render` should default `Options.Theme` to `"default"` (matching real Marp behavior when no theme is specified) or leave it empty/scaffold-only (matching today's internal-only behavior).
   - Recommendation: default to `"default"` — matches user-facing Marp compatibility expectations; document as an explicit, deliberate choice.

3. **hljs-class selector confirmation in the actual compiled theme CSS**
   - What we know: FEATURES.md already asserts a chroma/highlight.js class-name mismatch exists.
   - What's unclear: the *exact* set of `.hljs-*` selectors marp-core's bundled themes reference — blocked on the same CSS-acquisition spike as Open Question 1.
   - Recommendation: derive the remap table from the acquired compiled CSS directly (`grep -o '\.hljs-[a-z-]+'`), not from memory/assumption.

4. **Construct-detection predicate for CORE-08's fallback trigger**
   - What we know: PITFALLS.md's Pitfall 6 identifies the structural gap (Chromium MathML Core lacks `<mlabeledtr>`/full `<mtable>` support; `\tag`/`\label`/complex `aligned` environments are the trigger classes).
   - What's unclear: no existing test corpus was found to validate a specific predicate against.
   - Recommendation (new synthesis, MEDIUM confidence): a minimal pre-scan regex over the raw LaTeX source (before attempting MathML conversion) matching any of `\tag`, `\label`, or `\begin{aligned|align|alignat|cases|array}` — cheap, bounded, and directly targets the known-unsupported constructs rather than trying to detect failure after the fact from a partial MathML conversion.

## Sources

### Primary (HIGH confidence — direct source/repo reads)
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark@v1.8.4/extension/strikethrough.go`, `renderer/renderer.go`, `util/util.go` — NodeRenderer priority-override mechanism
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark-emoji@v1.0.6/{emoji.go,README.md,LICENSE,definition/definition.go}` — full API surface, MIT license text
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark-highlighting/v2@v2.0.0-20230729083705-37449abec8cc/highlighting.go` — full API surface
- `chase/chase.go`, `chase/markdown/{seam.go,render.go,renderdoc.go,markdown.go,comment.go}`, `chase/model/{document.go,build.go}`, `chase/theme/{pack.go,theme.go,meta.go}`, `chase/profile/{profile.go,registry.go}`, `profiles/slides/slides.go`, `chase/directive/{directives.go,frontmatter.go}`, `CONTRIBUTING.md`, `NOTICE`, `UPSTREAM-VERSIONS.txt`, `themes/README.md`, `go.mod` — all read directly from the repo, this session and prior
- `https://github.com/marp-team/marp-core` at tag `v4.4.0` (`themes/` directory listing, `raw.githubusercontent.com/.../themes/default.scss` verbatim header, `src/` and `src/auto-scaling/` directory listings) — WebFetch, direct repo content
- `https://pkg.go.dev/codeberg.org/go-latex/latex/drawtex` — subpackage listing confirming no SVG canvas

### Secondary (MEDIUM confidence)
- `app.unpkg.com` package-content browsing for `@marp-team/marp-core@4.4.0` (`lib/`, top-level `files`) — confirms no precompiled theme CSS ships in the npm package, but couldn't reach the actual `Marp`/theme API surface (blocked by unpkg not rendering `.d.ts` contents in this tool)
- WebSearch cross-referencing `roniemartinez/latex2mathml` (original Python), `git.sr.ht/~mekyt/latex2mathml` (the Go port STACK.md already recommends forking) — confirms these are the same lineage, `~mekyt` is the correct Go module to fork, `NOTICE`'s current `github.com/roniemartinez/latex2mathml` reference is the *Python* upstream this Go port itself credits, not a competing Go module — no conflict, `NOTICE` should eventually cite both

### Tertiary (LOW confidence, flagged for validation)
- goldmark-emoji's goldmark v1.7.10 floor vs. this repo's v1.8.4 — not build-tested, flagged as riskiest-item #3 above
- Bundled-theme `.hljs-*` selector list — asserted by FEATURES.md, not independently re-derived from actual compiled CSS in this pass (blocked on the CSS-acquisition open question)

## Metadata

**Confidence breakdown:**
- Battery wiring order / CORE-03 mechanism: HIGH — direct source verification, no assumptions
- goldmark-emoji / goldmark-highlighting reuse recommendation: HIGH on API surface, MEDIUM on cross-version build compatibility (untested)
- CORE-01 theme acquisition: HIGH on "what's wrong with embedding raw SCSS," MEDIUM on "exact fix" (spike still needed)
- CORE-08 math/fallback: HIGH on the SVG-canvas-doesn't-exist correction, MEDIUM on the construct-detection predicate (new synthesis, not yet corpus-tested)
- Options/Output API proposal: MEDIUM — a strong starting point, explicitly flagged as not frozen (esp. the `Highlight`/`NoHighlight` zero-value footgun and theme-fallback decision)

**Research date:** 2026-07-21
**Valid until:** 30 days for the mechanism-level (goldmark source) findings; 7-14 days for anything pinned to a specific marp-core npm/GitHub snapshot, since upstream drift tooling (`upstream-drift.yml`) may re-pin versions
