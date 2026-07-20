# Feature Research

**Domain:** Markdown-to-document(slide-deck) generation framework — Marp-compatible, Go/Dart, zero-JS backend
**Researched:** 2026-07-20
**Confidence:** HIGH (table stakes — verified against official Marp docs/source, this session) / MEDIUM-HIGH (differentiators — synthesized from PROPOSAL.md §13, cross-checked against ecosystem gaps) / HIGH (anti-features — explicit project constraints)

**Sources used this session:** `github.com/marp-team/marpit` (README + `docs/directives.md`, `docs/image-syntax.md`, `docs/theme-css.md` raw source), `github.com/marp-team/marp-core` (README + `themes/README.md`), `github.com/marp-team/marp-cli` (README), plus WebSearch cross-checks on auto-scaling, code-line-highlighting, sanitization, and stdin/URL input (marp-team GitHub Discussions/Issues). Context7 was not available in this environment; official-source WebFetch + WebSearch verification was used instead, per the priority order Context7 → Official Docs → WebSearch.

---

## Feature Landscape

### Table Stakes — Marp Compatibility (must reproduce, or it isn't "Marp-compatible")

These are organized by upstream layer since that maps directly to PROPOSAL.md's phasing (Objectives 1–5). Each row is verified against the actual Marpit/Marp Core/Marp CLI docs or source, not assumed from training data.

#### A. Marpit layer (the structural framework — no themes, no batteries)

| Feature | Why Expected (source) | Complexity | Notes |
|---------|------------------------|------------|-------|
| **Directive system — Global scope**: `theme`, `style`, `headingDivider`, `lang` | Whole-deck settings; only last value wins if repeated. `$`-prefix aliasing removed as of v1.4.0 — **do not implement the old `$theme` syntax** as current. [marpit/docs/directives.md] | MEDIUM | `size` and `math` are **NOT** Marpit directives — they are Marp-Core-layer custom global directives registered via `customDirectives.global`. Get this layering right or the port misattributes ownership. |
| **Directive system — Local scope**: `paginate`, `header`, `footer`, `class`, `backgroundColor`, `backgroundImage`, `backgroundPosition`, `backgroundRepeat`, `backgroundSize`, `color` | Applies to "this slide and all following" — carry-forward state machine per slide. [marpit/docs/directives.md] | HIGH | This carry-forward state is the trickiest part: it's not per-slide, it's a running reduction over the slide sequence. `paginate` has 4 distinct values (`true`/`false`/`hold`/`skip`) with independent show/increment semantics — not a boolean. |
| **Directive system — Spot scope**: any local directive prefixed `_` (e.g. `_class`, `_paginate`, `_backgroundColor`, `_color`) | Applies to current slide only, does not carry forward. [marpit/docs/directives.md] | LOW (once local directives exist) | Mechanically: same directive table, parsed with an `_` strip + "apply, don't carry" flag. |
| **Two directive syntaxes**: YAML front-matter (deck-level, must be first content between `---` rulers) + HTML-comment (`<!-- key: value -->`, inline) | Both parse as YAML; front-matter sets initial global+local state, comments update it mid-deck. [marpit/docs/directives.md] | MEDIUM | HTML-comment directives are **excluded** from the `comments` output (speaker-notes channel) — don't conflate directive comments with note comments. `looseYAML` mode (lenient parsing) exists as a constructor option — decide day one whether to support it. |
| **Custom directives extension point** (`customDirectives.global` / `customDirectives.local`) | This is *how* Marp Core adds `size`/`math` on top of Marpit — not a special case, a public extension mechanism. [marpit/docs/directives.md] | MEDIUM | Worth porting as a first-class extension point (not hardcoding `size`/`math` into the framework layer) — it's also the seam where Eden Press's own future directives would attach. |
| **Slide splitting**: `---` thematic-break rule + optional `headingDivider` (level 1–6 or array of levels) | Core structural transform — every slide is a markdown-it *core ruler* pass that turns thematic breaks (and optionally headings) into `<section>` boundaries. [marpit/docs/directives.md; PROPOSAL §2.1, §4.1] | HIGH | **Verified trap (spike §12, PROPOSAL):** `---` immediately after a paragraph is ambiguous with CommonMark setext-heading syntax (`Text\n---` → `<h2>Text</h2>`). Marpit's splitter must special-case this independent of parser choice — goldmark and markdown-it agree on the setext interpretation, so the *splitter* (not the base parser) must resolve the ambiguity correctly, matching Marp's actual (non-obvious) precedence rules. |
| **Background image syntax `![bg]`**: sizing keywords (`cover` default, `contain`/`fit` alias, `auto`, `x%` percentage, `width`/`w`, `height`/`h`) | `bg` in alt text triggers background-image handling instead of inline `<img>`. [marpit/docs/image-syntax.md] | MEDIUM | Without inline-SVG mode, only the **last** `![bg]` per slide wins (simple CSS background-image swap). |
| **Advanced backgrounds** (requires inline-SVG mode): multiple backgrounds (horizontal row by default, `vertical` keyword to stack), split backgrounds (`left`/`right` + `:NN%` size — reserves half the slide, shrinks content to the opposite side; last keyword wins if both used), image filters (`blur`, `brightness`, `contrast`, `drop-shadow`, `grayscale`, `hue-rotate`, `invert`, `opacity`, `saturate`, `sepia`, combinable) | Deckset-inspired advanced layout; genuinely one of Marp's more distinctive features. [marpit/docs/image-syntax.md] | HIGH | Depends on inline-SVG mode (below) being implemented first — hard dependency, not parallel work. |
| **Inline-SVG slide mode** (experimental upstream, but load-bearing for advanced backgrounds + fit): wraps each slide in `<svg viewBox><foreignObject><section>…</section></foreignObject></svg>` | Enables pixel-perfect CSS scaling and is the container advanced backgrounds/fit render into. [marpit README; marpit-svg-polyfill sub-project for Safari] | HIGH | Upstream ships a **separate polyfill script** (`marpit-svg-polyfill`) for Safari-class browsers lacking full `foreignObject` support — a verbatim-reusable MIT asset, not something to reimplement. |
| **Theme CSS format**: `/* @theme name */` metadata (**required** — Marpit refuses an unnamed theme; use `/*! */` to survive CSS minification/Sass-compression), `width`/`height` on `:root`/`section` in absolute units (`cm`/`in`/`mm`/`pc`/`pt`/`px`/`Q`; default `1280×720`; **fixed per theme, cannot be overridden by inline style/class/custom-property**), `@import` (compiled-CSS import; theme must already be registered in the ThemeSet), `@import-theme` (Sass-safe alternative; both resolve anywhere at CSS root; `@import` processed before `@import-theme`) | This is the theme-authoring contract every custom theme (and all 3 bundled themes) must satisfy. [marpit/docs/theme-css.md] | HIGH | `:root` in a Marpit theme means **the slide `<section>`**, not `<html>` — higher specificity than a bare `section` selector, and `rem` recalculates relative to the section, not the page. This is a "theme author must know this or nothing looks right" semantic — port it exactly. |
| **`@size [name] [width] [height]`** (Marp-Core-layer metadata, not base Marpit) | Defines named size presets a theme exposes to the `size` global directive; `@size <name> false` disables an inherited preset from an imported theme. [marp-core themes/README.md] | MEDIUM | Belongs in the Core layer per the directive-ownership note above — flagged here because it's part of "theme CSS format" as commonly understood, but structurally it's a Marp-Core convention layered on Marpit's `@import`/`@import-theme`. |
| **`@auto-scaling [true \| fittingHeader,math,code]`** (Marp-Core-layer metadata) | Theme opts into fit/auto-shrink behavior; **never settable from Markdown** — deliberately theme-author-only so document authors can't break a theme's layout guarantees. Requires inline-SVG mode; horizontal-only (vertical overflow is not corrected). [marp-core themes/README.md; verified via WebSearch cross-check, marp-team/marp-core#72 discussion] | MEDIUM | **Correction to a common assumption:** there is **no `auto-scaling` Markdown directive** — only the theme-CSS metadata. Confirmed both by docs and by an explicit marp-team maintainer clarification (people who look for a front-matter `auto-scaling:` key are looking for something that has never existed). |
| **Pagination CSS hook**: `section::after`/`:root::after` styled via `content` that **must** include `attr(data-marpit-pagination)` (else Marpit silently ignores the whole `content` declaration); `attr(data-marpit-pagination-total)` available for "N of M" styles | Load-bearing, easy-to-miss validation rule. [marpit/docs/theme-css.md] | LOW-MEDIUM | Reproduce the "must contain this exact `attr()` or the whole declaration is dropped" validation — it's a real upstream footgun worth matching (or improving with a clearer error, an Eden Press UX opportunity noted under differentiators). |
| **Header/footer directives**: support Markdown formatting + inline images; **cannot** use `![bg]` syntax inside header/footer due to parse-order constraints; no default CSS position (theme must place them, typically `position: absolute`) | [marpit/docs/directives.md] | LOW | Simple content substitution + a documented limitation to preserve, not fix silently (fixing it would be a compatibility break). |
| **Scoped inline `<style>`**: a `<style>` block inside the Markdown is parsed in theme-CSS context and merged into emitted CSS (stripped from rendered HTML); `<style scoped>` limits it to the current slide page | [marpit/docs/theme-css.md] | MEDIUM | Requires the CSS engine (theme scoper) to accept ad-hoc per-document CSS fragments, not just theme files — same code path, different trust boundary (relevant to the "safe for untrusted content" differentiator). |
| **CSS auto-scoping to `.marpit` container** — Marpit rewrites theme selectors to be scoped without the theme author doing anything | [marpit/docs/theme-css.md; PROPOSAL §4.2] | HIGH | This is the CSS-AST transform PROPOSAL §4.2 calls "the #2 risk" — selector rewriting over `tdewolff/parse/v2/css`. |

#### B. Marp Core layer (batteries on top of Marpit)

| Feature | Why Expected (source) | Complexity | Notes |
|---------|------------------------|------------|-------|
| **3 bundled themes: Default, Gaia, Uncover** | The only themes most users ever touch; each is verbatim MIT CSS, not logic. Default = GitHub-markdown-styled, always vertically centered, uses `github-markdown-css`-derived CSS variables (`--color-fg-default`, `--color-canvas-default`, …). Gaia = classic yhatt/marp look, left-top aligned by default, `lead` class (centers — title slides) and `gaia` class (alt color scheme), variables `--color-background/-foreground/-highlight/-dimmed`. Uncover = reveal.js-inspired minimal design, richest variable set (`--color-background-code`, `--color-background-paginate`, `--color-highlight-hover`, `--color-highlight-heading`, `--color-header`, `--color-header-shadow`, …). All three ship "auto-scaling ready." [marp-core themes/README.md] | LOW (copy) / MEDIUM (variable-driven customization layer) | **Copy verbatim + `go:embed`**, per PROPOSAL — but document each theme's CSS-variable surface explicitly; it's the de facto "theming API" users already customize via CSS-variable overrides today, and Eden Press's own token layer (differentiator #4) should map onto these exact variable names for drop-in compatibility. |
| **Universal `invert` class + 4:3 size preset** (`<!-- size: 4:3 -->` → 960×720) | Common to all 3 themes. [marp-core themes/README.md] | LOW | |
| **GFM tables + strikethrough** | Marp-Core addition — **confirmed NOT part of base Marpit** (Marpit's own README/docs make no GFM claim; Marp Core's README explicitly lists these as Marp-Core extensions "based on GitHub Flavored Markdown"). | LOW (goldmark has both natively) | Layering correction versus a naive reading of "Marp = GFM Markdown" — keep GFM support in the `core` package, not `marpit`, to match upstream's actual boundary and keep Marpit importable standalone. |
| **Line breaks → `<br>`** | Paragraph soft-breaks render as hard breaks — a Marp-Core Markdown-option choice (goldmark: `WithHardWraps`), not universal CommonMark default. [marp-core README] | LOW | Spike-verified 31/32 goldmark/markdown-it parity used exactly this config (`breaks:true` ~ `WithHardWraps`) — already validated (PROPOSAL §12). |
| **Heading slugs**: auto `id` on `h1`–`h6`, GitHub-compatible, on by default; configurable (`slug: true/false/custom function`, `slugifier` + `postSlugify` hooks, dedup logic) | [marp-core README] | LOW-MEDIUM | The "avoid duplicate IDs" postprocessing is a small but real detail (`heading`, `heading-1`, `heading-2` pattern) — verify against goldmark's own auto-heading-id extension rather than assuming identical dedup behavior. |
| **HTML allow-list sanitization** tied to markdown-it's `html` option: `false`(only Marpit-required HTML)/`true` (everything)/curated default allow-list (**relaxed significantly in Marp Core v4** to permit common layout tags/attributes — this was a deliberate response to "forgetting to enable HTML" being a top user complaint) / custom object (tags+attributes incl. per-attribute sanitizer functions); `<style>` and HTML comments are **always** parsed regardless of setting | [marp-core README; WebSearch cross-check of marp-core PR #74 / #533 / v4 allowlist relaxation] | HIGH (security-sensitive) | **Must pin to the current (v4-era) allow-list, not an older/stricter one** — this changed materially between major versions. `bluemonday` policy needs an explicit allow-list matching the *current* upstream list, tested case-by-case (per PROPOSAL §9 risk: "sanitization parity — security-sensitive"). |
| **Emoji**: shortcodes (`:smile:`) and raw Unicode emoji → twemoji SVG (or PNG); `emoji.shortcode`: `false`/`true`(→plain Unicode)/`"twemoji"`(default); `emoji.unicode`: `"twemoji"`(default)/`false`; twemoji sub-options `base` (CDN/local path) and `ext` (`svg`/`png`) | [marp-core README] | MEDIUM | Native port = shortcode table + Unicode-range regex + asset resolution, exactly as PROPOSAL describes; the `base`/`ext` override knobs are part of the contract to preserve (self-hosting twemoji assets is a real deployment need, esp. for the "safe for untrusted / air-gapped" differentiator). |
| **Math**: `$...$` inline / `$$...$$` block (Pandoc-style delimiters, not raw LaTeX `\(\)`/`\[\]`); engines are **MathJax (default)** or KaTeX (opt-in, faster for math-heavy decks); `math` global directive (Core-layer, not Marpit) overrides per-document but **cannot re-enable a math engine disabled at the constructor level**; sub-options `lib`, `katexOption`, `katexFontPath` | [marp-core README] | HIGH | **Important correction to PROPOSAL's framing:** upstream's *default* engine is **MathJax, not KaTeX** — PROPOSAL's math-fidelity spike (§11) benchmarked against KaTeX-quality output, which is the right *quality bar* (KaTeX and MathJax render the same TeX-to-visual fidelity for common constructs) but Eden Press should decide explicitly whether "Marp-compatible" math means matching MathJax's default behavior/rendering quirks or documents this as an intentional deviation (native MathML instead of either JS engine). Document this as a compatibility note, not silently diverge. |
| **Auto-scaling / fit** (theme-gated via `@auto-scaling`): `# <!--fit-->` fitting headers; code-block and KaTeX-block auto-shrink to avoid right-edge overflow; MathJax blocks **always** auto-shrink regardless of the metadata flag; horizontal-only, depends on inline-SVG | [marp-core README + themes/README.md] | HIGH (inherently browser-side — see PROPOSAL §3) | Confirmed by WebSearch cross-check: this is real, documented, and theme-gated, not a folk feature. The "MathJax always scales even without opting in" exception is a specific behavior to preserve/replicate for compatibility, or explicitly deviate from and document. |
| **Code syntax highlighting** via highlight.js | [marp-core README implies "automatic syntax highlighting"; general ecosystem knowledge] | MEDIUM | `chroma` per PROPOSAL. Chroma token-class taxonomy ≠ highlight.js `.hljs-*` classes — the 3 bundled themes' code-block CSS needs a one-time regeneration/remap (PROPOSAL §9 already flags this). |
| ~~"Line-highlight" code syntax (`{1-3}`)~~ | **Correction — verified this session, do not treat as table stakes.** Line-based highlighting is **not built into marp-core**; it requires the community `markdown-it-highlight-lines` plugin (via a custom engine/config) or a hand-rolled `marpit.highlighter` override, and even upstream discussion frames it as a "long-requested, not-yet-built-in" feature (tracked in marp-core issues #168, #296). Full-width highlight styling additionally requires **disabling** `@auto-scaling` for code in a custom theme — the two features actively interact. | N/A | **Move to Differentiators, not Table Stakes.** Reproducing this is not required for Marp compatibility (it isn't a Marp Core feature); *offering it natively* (where upstream users need a plugin) is a legitimate, low-controversy differentiator. |
| **CSS container queries enabled by default** (inherited from Marpit) | [marp-core README, listed alongside inline-SVG/loose-YAML as Marpit-inherited defaults] | LOW (flag) / MEDIUM (use) | Directly relevant to PROPOSAL §3's "CSS-only fit via `cqw` units" plan for the auto-fit open decision — confirms the primitive is already part of the rendering model, not something to newly introduce. |

#### C. Marp CLI layer (the tool)

| Feature | Why Expected (source) | Complexity | Notes |
|---------|------------------------|------------|-------|
| **HTML output** (default conversion target) | Baseline. [marp-cli README] | LOW | |
| **PDF export**: `--pdf` / `.pdf` extension; `--pdf-notes` (speaker notes as PDF annotations); `--pdf-outlines` (bookmarks; `.pages`/`.headings` sub-modes) | [marp-cli README] | MEDIUM (bookmarks/annotations logic) + HIGH (Chrome driving, shared with PNG/PPTX) | `chromedp` → `Page.printToPDF`, per PROPOSAL. The outline/bookmark and notes-as-annotation features are additional PDF-structure work beyond "just print the HTML." |
| **PPTX export**: image-per-slide by default; **experimental `--pptx-editable`** mode requiring a LibreOffice Impress (`soffice`) round-trip | [marp-cli README] | HIGH | Confirms PROPOSAL's read: upstream's "editable" PPTX is not native OOXML text boxes, it's LibreOffice-mediated. Eden Press's differentiator (#5 below, native OOXML text boxes without a LibreOffice dependency) is a genuine improvement over upstream's own "editable" mode, not merely parity. |
| **PNG/JPEG export**: `--images [png\|jpeg]` (all slides), `--image` (title slide only), `--image-scale`, `--jpeg-quality` (default 85) | [marp-cli README] | MEDIUM | Per-slide `chromedp` clip/element screenshot, per PROPOSAL. |
| **Presenter notes as plain text**: `--notes` / `.txt` output | [marp-cli README] | LOW | Straightforward extraction of the `comments` channel Marpit already separates out. |
| **Watch mode** (`--watch`/`-w`): reconvert on change, auto-refresh an open browser tab | [marp-cli README] | MEDIUM | `fsnotify`, per PROPOSAL. |
| **Server mode** (`--server`/`-s`): on-demand HTTP conversion of a served directory; format selected via query string (`?pdf`, `?pptx`, `?png`, `?jpeg`, `?txt`); `index.md`/`PITCHME.md` default-deck convention; port via `PORT` env | [marp-cli README] | MEDIUM-HIGH | This *is* effectively "URL input" in upstream's actual model — **verified correction:** Marp CLI does **not** accept an arbitrary remote URL as a direct input argument; "URL input" in practice means (a) server mode's own on-demand HTTP conversion, or (b) users piping `curl url \| marp -` through stdin. Scope the "stdin/URL input" requirement in PROJECT.md accordingly — stdin (`-`) is real and simple; direct fetch-a-remote-URL-as-input is not an upstream feature to match. |
| **Preview window** (`--preview`/`-p`): opens a live preview, auto-enables watch; unavailable in Docker/headless contexts | [marp-cli README] | MEDIUM | Desktop-only concern; less relevant to a server-embeddable Go library but real for CLI parity. |
| **Concurrency**: parallel conversion of multiple inputs (default concurrency 5, tunable via `--parallel`/`-P`, disable via `--no-parallel`) | [marp-cli README] | LOW (Go goroutines make this cheap) | |
| **Input types**: single file, multiple files, directories + globs, `--input-dir`/`-I` (preserve directory structure on output), **stdin via `-`** | [marp-cli README] | LOW-MEDIUM | |
| **Theme options**: `--theme` (built-in name or custom CSS file — custom themes via CLI flag don't require the `@theme` metadata comment), `--theme-set` (register multiple theme files/directories for the Markdown `theme:` directive to select from) | [marp-cli README] | MEDIUM | The "`--theme` via CLI doesn't require `@theme` metadata but `theme:` directive selection does" asymmetry is a real upstream nuance to preserve or consciously simplify. |
| **Config file support**: `marp.config.{js,mjs,cjs,ts}`, `.marprc` (JSON/YAML), `marp` key in `package.json`; `--config-file`/`-c`; `--no-config` to skip discovery | [marp-cli README] | MEDIUM | Go equivalent: YAML/JSON/TOML via `koanf`/`viper` per PROPOSAL, explicitly **dropping** JS-based config (`.js`/`.ts`/`package.json`-embedded) since that would reintroduce a JS/Node dependency for config alone — a deliberate, justified compatibility gap to document, not an oversight. |
| **Metadata directives/flags**: `--title`, `--description`, `--author`, `--keywords`, `--url` (canonical), `--og-image`; CLI flags override in-document directives | [marp-cli README] | LOW | |
| **Two output templates**: `bespoke` (default — keyboard/on-screen navigation, fullscreen, presenter view, overview/grid view, optional progress bar, View-Transition-API slide transitions) vs `bare` (minimal, "supports zero-JS decks when using the Marpit engine") | [marp-cli README] | MEDIUM (bare) / HIGH (bespoke, if reproduced) | **Load-bearing distinction for the JS-free mandate:** `bare` is upstream's *own* acknowledgment that a fully static, JS-free HTML output is a first-class supported mode — this is Eden Press's natural default template, not a deviation. `bespoke`'s presenter/navigation features are viewer-side JS (not backend JS) shipped as a bundled script; treat as an optional, clearly-scoped-later template (see Anti-Features note below), not v1 table stakes. |
| **Pluggable engine** (`--engine`): swap the underlying Marpit-based converter (npm module, class, or functional file), with plugin injection via a `marp` getter (e.g., markdown-it plugins) | [marp-cli README] | MEDIUM (design-time), not urgent to port 1:1 | Go analogue = the `core.Options{}` constructor already sketched in PROPOSAL §5.2; less about literally supporting swappable *JS* engine files (N/A, JS-free) and more about preserving the *extensibility contract* (custom directives, custom highlighter, custom sanitizer). |
| **Browser discovery/selection**: `--browser` (chrome/edge/firefox/auto + fallback list), `--browser-path`, `--browser-protocol` (CDP default or WebDriver-BiDi), `--browser-timeout` (30s default) | [marp-cli README] | MEDIUM | `chromedp` needs equivalent discovery logic; PROPOSAL's open decision #3 (system-first vs. bundled/pinned Chromium) sits here. |
| **Local-file access guard**: browser-based conversions block local file access by default; `--allow-local-files` opts in with an explicit security-risk callout | [marp-cli README] | LOW-MEDIUM | Directly reinforces the "safe for untrusted content" differentiator — upstream already treats this as a security boundary worth a flag; Eden Press's capability-gated renderer generalizes this same instinct. |
| **Node.js embeddable API** (`marpCli()` returns a Promise; `CLIError`/`CLIErrorCode`; `waitForObservation()`) | [marp-cli README] | N/A (JS-specific) | Not a port target — Eden Press's answer to "embeddable API" is the **Go library itself** (`press.Render()`), which is a stronger position than Marp CLI's Node-API wrapper (PROPOSAL differentiator #2). |

---

### Differentiators (Eden Press's Reason to Exist Beyond a Port — PROPOSAL §13)

| Feature | Value Proposition | Complexity | v1 or Later | Notes |
|---------|--------------------|------------|:---:|-------|
| **Output-profile abstraction** (one Markdown+directive model → pluggable output profiles: slides now, paged docs/reports/EPUB/single-page articles later) | This is the identity: "a press, not a slide tool." Without this abstraction shaping v1's internals, later profiles become rewrites instead of extensions — the single highest-leverage architectural decision in the whole project. | MEDIUM (as an architecture decision made early) / HIGH (if retrofitted later) | **v1 — architecturally, not functionally.** Ship only the slides profile, but design the `render` boundary (directive resolution, theme/token system, document model) so "slides" is one profile implementation, not baked-in. | Directly maps to PROPOSAL §13 item 1 and the phasing note "lay the profile abstraction... so #1–#6 are natural extensions, not rewrites." |
| **Library-first, server-native rendering** (`press.Render()` importable in any Go binary/service; zero Node, zero browser required for HTML/structured output; Chrome opt-in only for raster export) | Marp is fundamentally a Node CLI wrapping a JS library; Eden Press flips this — the library is the product, the CLI is a thin consumer. This is what makes Eden Press embeddable in Eden-Biz/AOCore without spinning up a sidecar process. | LOW-MEDIUM (mostly a packaging/API-boundary discipline, not new algorithms) | **v1.** Already the stated architecture (PROPOSAL §5.2: engine has zero Chrome dependency; only `convert` touches Chrome). | This differentiator is nearly free if the module layout in PROPOSAL §5.1 is followed — the risk is scope creep pulling Chrome/CLI concerns into the engine package. |
| **Structured document-model / AST output** (JSON AST, outline, speaker notes, metadata — alongside HTML/CSS) | Because Eden Press owns the parse tree (unlike a black-box Node CLI), it can expose the deck as **data**: programmatic manipulation, search/indexing, accessibility trees, translation pipelines, LLM/AOCore ingestion. This is a genuinely un-Marp-like capability — upstream has no public AST/JSON export. | MEDIUM | **v1 — as an API-shape decision**, even if only a subset of consumers (translation, LLM ingestion) are wired up later. Since the engine already builds an AST (goldmark) internally, exposing a stable serialized form is incremental once the internal model is designed for it. | Depends on: markdown parsing (table stakes, above) + directive resolution (table stakes) — the AST output is a *view* over data those steps already compute, so sequence this after the parser/directive engine, not before. |
| **Design-token theming** (JSON/TOML brand tokens → generated theme CSS; multi-tenant brand support) | Marp themes are hand-written raw CSS per deck/org — fine for individual authors, painful for a SaaS serving many tenants' brand identities. A token layer generates conformant `@theme`-format CSS programmatically. | MEDIUM-HIGH | **Later (v1.x/v2).** Explicitly a "layered differentiator" in PROPOSAL §13 ("build on the above"). | **Hard dependency:** requires the theme-CSS engine (table stakes, PROPOSAL §4.2) to exist first — token→CSS generation targets the same `@theme`/`@size`/`@auto-scaling` contract, and should map onto the bundled themes' existing CSS-variable surface (see Marp-Core theme table above) for a smooth migration path. |
| **Safe-for-untrusted-content rendering** (capability-gated: no implicit network/asset fetch, deterministic, strict allow-list sanitization) | Enables a multi-tenant SaaS to render arbitrary user-submitted Markdown without SSRF/exfiltration/XSS risk — an explicit Eden/AOCyber-ethos requirement, not just a nice-to-have. | MEDIUM-HIGH | **Partially v1, hardens later.** The HTML allow-list sanitizer (table stakes) must exist for v1 regardless of trust model; the *capability-gating* (blocking implicit local-file/network access during render — note upstream's own `--allow-local-files` security flag above) is the additional layer, and should be designed in from day one even if not fully hardened until a dedicated security pass. | Directly informed by Marp CLI's own `--allow-local-files` guard (verified table stakes above) — this differentiator is "upstream's opt-in safety flag, made the mandatory default." |
| **Deterministic / reproducible output** (byte-reproducible rendering, git-diffable decks, content-hashed incremental rebuilds) | A natural consequence of an all-Go, no-JS-runtime, no-headless-browser (for HTML) pipeline — genuinely hard for Marp to guarantee given Node/npm dependency-resolution variance and any JS-engine nondeterminism. | LOW-MEDIUM (mostly "don't introduce nondeterminism," e.g. map iteration order, timestamp injection, font-fallback variance) | **v1-adjacent** — cheap to preserve if disciplined from the start (deterministic map ordering, no wall-clock in output unless explicit, pinned embedded assets), expensive to retrofit if determinism-breaking shortcuts creep in early (e.g., unordered directive-merge maps). | Worth a lint/test rule (render same input twice, byte-diff) added early, not deferred — this is a "test for it from commit one" differentiator, not a late feature. |
| **Editable PPTX (real OOXML text boxes, not image-per-slide)** | Upstream's own "editable" PPTX mode requires a LibreOffice round-trip (verified table stakes above) — Eden Press doing this **natively** in Go (build the OOXML zip directly with real text runs) is a genuine quality upgrade over both Marp's default (image-per-slide) and Marp's "editable" mode (LibreOffice dependency). | HIGH | **Later** (PROPOSAL Objective 5: "PPTX + polish"). | Requires the structured document model (differentiator above) to exist first — mapping AST nodes to OOXML text-box runs needs a stable intermediate representation, not ad-hoc HTML scraping. |
| **Tagged / accessible, PDF/A-capable output** | Enterprise/government/archival buyers often have hard accessibility (tagged PDF, reading order) or archival (PDF/A) compliance requirements Marp CLI's plain `printToPDF` doesn't attempt. | HIGH | **Later.** Not in PROPOSAL's Objective 0–7 phasing at all yet — flag for a dedicated future objective once core export (Objective 4–5) ships. | Depends on Chrome-based PDF export (table stakes) existing first as a baseline, plus the structured document model for tag-tree generation — this is a "third pass over PDF export," not a variant of the first. |

---

### Anti-Features (Deliberately NOT Building)

| Feature | Why It Seems Appealing | Why Problematic (project-specific) | Alternative |
|---------|------------------------|--------------------------------------|-------------|
| **`goja` (or any embedded JS interpreter) / Node / npm at runtime** | Would make porting Marp's actual JS leaf libraries (highlight.js, KaTeX, `xss`, `pptxgenjs`) trivial — just run the real thing. | Directly defeats the project's core motivation (a JS-free backend, single static Go binary, embeddable with no sidecar process). Also reintroduces npm supply-chain surface into a security-conscious (AOCyber-ethos) stack. Verified upstream deps that *would* need this if ported as-is: `highlight.js`, `katex`/`mathjax-full`, `xss`, `pptxgenjs`, `puppeteer-core` (Chrome driving only — that part is unavoidable but is a binary, not JS-in-process). | Native Go/pure-Go replacements per PROPOSAL: `chroma` (highlight), `latex2mathml`→MathML + `go-latex/latex` SVG/PNG fallback (math), native emoji table, native OOXML builder (PPTX), `chromedp` (drives the Chrome *binary*, not JS-in-process). |
| **Interactive/reactive slide components (Slidev/Vue-style: live component islands, client-side reactivity, `<script setup>`-style embedded app logic)** | Slidev-class tools are popular precisely because they let authors embed live Vue/React components inside slides — richer than static Marp decks. | Directly contradicts "server-side/static, templating not client reactivity" (PROJECT.md Out of Scope) and the zero-JS mandate — a reactive component model requires a client-side JS framework runtime by definition. Also a fundamentally different product category (an app-authoring tool) from "a press" (a document-generation pipeline). | If interactivity is ever wanted, it belongs in a downstream consumer (e.g., an Eden-UI viewer layer) built as an explicit, separately-scoped client, never inside the Eden Press rendering core. |
| **A WYSIWYG editor / end-user authoring app** | Natural "complete the product" instinct — Marp has community WYSIWYG-adjacent tools (marp-vscode, web editors); users often expect an editor alongside a renderer. | That's explicitly Eden Docs' role (the Collabora-fork productivity suite), not Eden Press's. Building an editor here duplicates effort and blurs the "library + CLI, not an editor" identity (PROJECT.md). | Eden Docs (or any Markdown-aware editor) consumes Eden Press as a rendering backend; Eden Press stays headless. |
| **Forking Marp's actual source (JS codebases)** | Fastest path to 100% behavioral fidelity — literally reuse the reference implementation's logic/tests. | Contradicts the "clean-room reimplementation" positioning (PROPOSAL §14) and reintroduces the exact JS runtime dependency the whole project exists to remove. Also raises different licensing/attribution obligations than "inspired-by, MIT-compatible-native-port" framing. | Clean-room native reimplementation validated against a conformance corpus **extracted from** (not executed from) Marp's own MIT-licensed Jest snapshot fixtures (PROPOSAL §6) — same fidelity discipline, zero runtime coupling. |
| **Reproducing `bespoke` template's presenter/navigation JS bundle as v1 table stakes** | It's "the default Marp CLI experience" — keyboard navigation, fullscreen, presenter view, overview grid, slide transitions — so it's tempting to treat as required parity. | It's viewer-side JS shipped to the browser, not backend JS — technically compatible with "zero-JS **backend**," but out of step with a v1 that's proving the JS-free *rendering* story, and it's a large, separable surface (full presenter UI) that upstream itself treats as swappable (`--template bare` exists precisely because not everyone wants it). | Ship **`bare`**-equivalent static HTML as the v1 default (upstream's own zero-JS-capable template) and treat a bespoke-equivalent navigation layer as an explicit, later, clearly-scoped viewer feature — not a hidden dependency of "HTML export." |
| **Line-level code-highlight (`{1-3}`) treated as a compatibility must-have** | It "feels like" a Marp feature because it's commonly seen in Marp decks in the wild via the community plugin. | Verified this session: **not actually part of marp-core** — it's a third-party markdown-it plugin (`markdown-it-highlight-lines`) or a hand-rolled highlighter override upstream itself has never shipped natively (tracked as an open feature request, marp-core #168/#296). Treating it as table stakes would mean chasing fidelity with a *plugin*, not "Marp." | Offer it as a differentiator (native, built-in, no plugin required) rather than a compatibility requirement — see Marp-Core-layer table correction above. |
| **Direct remote-URL-as-input** (`eden-press https://example.com/deck.md`) | "Marp CLI supports URL input" is a plausible-sounding assumption, and server mode's HTTP-conversion behavior can look like it from a distance. | Verified this session: Marp CLI's actual input surface is file/dir/glob/stdin plus **server mode** (serving local files, converted on HTTP request) — not "fetch an arbitrary remote URL as the document source." Building true remote-fetch input isn't reproducing a real upstream feature, and it reintroduces the exact implicit-network-fetch risk the "safe for untrusted content" differentiator is designed to close off. | Support stdin (`-`) for true parity; let users `curl \| eden-press -` if they want remote content, matching upstream's actual (not assumed) behavior. Server mode (serve a directory, convert on request) is the real "URL-shaped" feature to port. |

---

## Feature Dependencies

```
Markdown parsing (goldmark base, spike-validated 31/32) [table stakes: foundation]
    └──requires──> Directive system (global/local/spot, front-matter + comment syntax)
                       └──requires──> Slide splitting (---, headingDivider) [needs directive state for headingDivider level]
                       └──requires──> Custom-directive extension point
                                          └──enables──> Core-layer directives (size, math)

Slide splitting
    └──requires──> Container/section rendering (.marpit wrapper)
                       └──requires──> Inline-SVG mode
                                          └──requires──> Advanced backgrounds (multi/split/filters)
                                          └──requires──> Auto-scaling / fit (theme-gated)
                                          └──requires──> marpit-svg-polyfill asset (Safari compat)

Theme-CSS engine (@theme/@size/@auto-scaling parsing, selector scoping, :root mapping, @import/@import-theme)
    └──requires──> CSS-AST transform (tdewolff/parse/v2/css)
    └──enables──> 3 bundled themes (embed verbatim)
    └──enables──> Pagination CSS hook (attr(data-marpit-pagination) validation)
    └──enables──> Design-token theming (differentiator) ──requires──> theme-CSS engine + bundled-theme CSS-variable surface

Document/AST model (structured output)
    └──requires──> Markdown parsing + Directive resolution (both already computed; AST output is a serialization view)
    └──enables──> Editable PPTX (differentiator) ──requires──> stable AST → OOXML text-run mapping
    └──enables──> Tagged/accessible PDF (differentiator) ──requires──> AST → PDF tag-tree mapping

HTML allow-list sanitization
    └──enables──> Safe-for-untrusted-content rendering (differentiator, adds capability-gating on top)

Importable Go API (press.Render) [library-first differentiator]
    └──requires──> Engine layers above, with zero Chrome dependency
    └──enables──> CLI (thin consumer) + Chrome-based convert package (PDF/PNG/PPTX raster export)

Output-profile abstraction (differentiator, architectural)
    └──shapes──> ALL of the above — slides is profile #1; retrofitting this after v1 ships is the expensive path

Line-highlight code feature (differentiator, NOT table stakes) ──conflicts-with──> @auto-scaling code-shrink
    (upstream note: full-width line highlights require *disabling* auto-scaling for code in the active theme — the two features actively fight each other, same interaction to preserve if both are offered)
```

### Dependency Notes

- **Slide splitting requires the directive system, not just the base parser:** `headingDivider` (a directive value) changes *where* splitting happens, so the directive-resolution pass and the slide-splitting pass are mutually entangled, not strictly sequential — plan them as one engineering unit (matches PROPOSAL's Objective 1 bundling them together).
- **Advanced backgrounds and auto-scaling both hard-require inline-SVG mode.** Do not schedule either as parallelizable with the inline-SVG container work — both are literally inert without it (verified: "will not work if you disable Marpit's inlineSVG mode" is explicit upstream language for auto-scaling; advanced backgrounds require it structurally for the `foreignObject` layer).
- **Design-token theming, editable PPTX, and tagged PDF (differentiators) all require table-stakes engine pieces to exist first** — the theme-CSS engine, the AST/document model, and Chrome-based raster export respectively. None of these differentiators can be pulled forward ahead of their table-stakes dependency without doing throwaway work.
- **Line-highlight code feature conflicts with auto-scaling's code-shrink behavior** — this is a genuine upstream-documented interaction (disable `@auto-scaling` for `code` to get full-width highlights), not a hypothetical; if Eden Press offers both, the theme system needs an explicit way to express "auto-scaling on for math, off for code" per-feature-keyword granularity (which `@auto-scaling`'s comma-separated keyword design already supports — `fittingHeader,math` without `code`).
- **The output-profile abstraction is a dependency of everything else in the sense that matters most: it doesn't gate any single table-stakes feature, but retrofitting it after the slides profile's internals harden is the single most expensive mistake this project could make** — per PROPOSAL, this shapes v1 architecture even though only one profile ships in v1.

---

## MVP Definition

### Launch With (v1) — Marp-Compatible Slides, Zero-JS Backend

Matches PROPOSAL §8 Objectives 0–5 (the phasing this research should feed directly into the roadmap):

- [ ] Conformance corpus + AST-diff runner extracted from Marp's own Jest fixtures (Objective 0) — **the acceptance gate for everything below, must exist first**
- [ ] Markdown parsing (goldmark + GFM tables/strikethrough/hard-breaks, spike-validated) — table stakes
- [ ] Directive system: global/local/spot, front-matter + HTML-comment syntax, custom-directive extension point — table stakes
- [ ] Slide splitting (`---`, `headingDivider`) with the setext-ambiguity handled correctly — table stakes
- [ ] `.marpit` container + inline-SVG mode + `marpit-svg-polyfill`-equivalent (or documented Safari gap) — table stakes
- [ ] Background image syntax (`![bg]` sizing) + advanced backgrounds (multi/split/filters) — table stakes
- [ ] Theme-CSS engine (`@theme`/`@size`/`@auto-scaling` parsing, `:root` mapping, selector scoping, `@import`/`@import-theme`, pagination hook) — table stakes
- [ ] 3 bundled themes embedded verbatim (Default/Gaia/Uncover) with attribution — table stakes
- [ ] Heading slugs, HTML allow-list sanitization (current v4-era allow-list), native emoji — table stakes
- [ ] `chroma` highlighting (with theme CSS class remap) — table stakes
- [ ] `latex2mathml`→MathML math with the converter-hardening fixes from the math spike (Objective 7's bugs, pulled forward since they're bounded/known) — table stakes, quality caveat documented
- [ ] Fit markers emitted (theme-gated `@auto-scaling`); actual fit behavior resolved per PROPOSAL's open decision #1 (native Flutter / CSS-only / drop for browser-HTML) — table stakes, mechanism TBD
- [ ] Importable Go API (`press.Render`) as the primary interface — differentiator, near-free if module boundaries are respected
- [ ] Structured document-model / AST output (even a v1-minimal shape: outline + notes + metadata) — differentiator, cheap once the AST exists internally
- [ ] Output-profile abstraction **as an architectural boundary**, slides as the only shipping profile — differentiator, architecture-only for v1
- [ ] CLI: HTML/`bare`-template output, watch, server (with query-string format selection), preview, config file (YAML/JSON/TOML, no JS config) — table stakes
- [ ] PDF + PNG/JPEG export via `chromedp` — table stakes
- [ ] HTML allow-list sanitization = a *default*, not opt-in — baseline for the safe-for-untrusted-content differentiator
- [ ] Attribution shipped day one (LICENSE, NOTICE/CREDITS, per-file MIT headers on the 3 themes + polyfill/browser assets) — non-negotiable, low cost, do early

### Add After Validation (v1.x)

- [ ] PPTX export, starting with image-per-slide parity, then **native editable OOXML text boxes** (surpassing upstream's LibreOffice-mediated "editable" mode) — Objective 5
- [ ] Dart/Flutter client via Go→WASM + FFI, JS-free (`flutter_math_fork`, pure-Dart `highlight`, native `TextPainter` fit) — Objective 6
- [ ] Design-token theming (JSON/TOML brand tokens → generated theme CSS), mapped onto the bundled themes' existing CSS-variable surface for easy adoption
- [ ] Native line-highlight code feature (built-in, no plugin needed) — an easy, low-risk differentiator once `chroma` integration is solid
- [ ] Bundled/pinned Chromium download option (vs. system-Chrome-only) — resolves PROPOSAL's open decision #3 once real deployment friction is observed

### Future Consideration (v2+)

- [ ] Additional output profiles: paged documents/reports (A4/Letter, running headers, page numbers, TOC), single-page articles, EPUB — the actual payoff of the profile abstraction, deferred until the slides profile has proven the architecture in production
- [ ] Tagged/accessible, PDF/A-capable export — enterprise/gov/archival buyers, not needed to validate the core product
- [ ] Capability-gated hardening pass for fully untrusted multi-tenant rendering (beyond the v1 sanitization baseline) — defer until an actual multi-tenant SaaS consumer (Eden-Biz) needs it, so the threat model is real rather than speculative
- [ ] Bespoke-equivalent presenter/navigation viewer (keyboard nav, fullscreen, overview grid, transitions) as an explicit opt-in viewer layer — only if user feedback shows the static `bare`-style output is insufficient for live presenting use cases

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|----------------------|----------|
| Conformance corpus + runner | HIGH (de-risks everything else) | MEDIUM | P1 |
| Markdown parsing + directive system + slide splitting | HIGH (nothing works without it) | HIGH | P1 |
| Theme-CSS engine + 3 bundled themes | HIGH (no visual output without it) | HIGH | P1 |
| Inline-SVG mode + advanced backgrounds | MEDIUM-HIGH (a distinctive Marp feature; some decks don't use it) | HIGH | P1 |
| Sanitization + emoji + slugs + GFM | MEDIUM (expected, low glamor) | LOW-MEDIUM | P1 |
| Highlighting (chroma) | MEDIUM-HIGH (very common in decks) | MEDIUM | P1 |
| Math (latex2mathml→MathML) | MEDIUM (math-heavy decks are a minority but vocal) | HIGH (converter-hardening pass) | P1 (with documented quality caveat) |
| Importable Go API + structured AST output | HIGH (the actual product thesis) | LOW-MEDIUM | P1 |
| Output-profile abstraction (architecture only) | HIGH (long-term identity) | MEDIUM (if designed early) / HIGH (if retrofitted) | P1 |
| CLI (HTML/watch/server/preview/config) | HIGH (parity expectation from Marp users) | MEDIUM | P1 |
| PDF/PNG/JPEG export | HIGH (the most common export need) | HIGH (Chrome integration) | P1 |
| PPTX export (image-per-slide) | MEDIUM | MEDIUM | P2 |
| PPTX editable (native OOXML) | MEDIUM-HIGH (a genuine upgrade over upstream) | HIGH | P2 |
| Dart/Flutter client (WASM+FFI) | MEDIUM (stated goal, but not needed to validate the Go core) | HIGH | P2 |
| Design-token theming | MEDIUM-HIGH (the multi-tenant SaaS story) | MEDIUM-HIGH | P2 |
| Native line-highlight code feature | LOW-MEDIUM (nice, not required) | LOW | P3 |
| Additional output profiles (paged/EPUB) | HIGH long-term, LOW near-term (no v1 users need it yet) | HIGH | P3 |
| Tagged/accessible PDF/A | LOW-MEDIUM (narrow buyer segment) | HIGH | P3 |
| Bespoke-equivalent presenter viewer | LOW (out of scope per zero-JS-forward posture) | HIGH | P3 |

**Priority key:** P1 = must have for launch · P2 = should have, add when possible · P3 = nice to have, future consideration

---

## Competitor / Reference-Implementation Feature Analysis

| Feature | Marp (upstream JS) | Eden Press Approach |
|---------|---------------------|----------------------|
| Runtime | Node.js (v18+), npm dependency tree, headless Chrome for raster export | Single static Go binary; goldmark/chroma/latex2mathml native libs; Chrome only for raster export (same external as Marp) |
| Highlighting | highlight.js (JS tokenizer) | `chroma` (pure Go), remapped theme CSS classes |
| Math | MathJax (default) or KaTeX (opt-in), both JS layout engines | `latex2mathml`→browser-native MathML, pure-Go SVG/PNG fallback for heavy math |
| Line-highlight code | Not built-in — community plugin (`markdown-it-highlight-lines`) or custom engine | Native built-in (differentiator), deferred to v1.x |
| Embeddability | Node CLI + a Node-API wrapper (`marpCli()`) | Native Go library (`press.Render()`) as the primary surface, CLI is a thin consumer |
| Output as data | HTML/CSS only; no public AST/JSON export | Structured document-model (AST/outline/notes/metadata) as a first-class output |
| Theming | Hand-written raw theme CSS per author/org | Same raw-CSS compatibility retained, plus an optional design-token→CSS generation layer for multi-tenant brands (v1.x+) |
| PPTX editable mode | LibreOffice (`soffice`)-mediated round-trip, experimental | Native OOXML text-box generation, no LibreOffice dependency (v1.x) |
| Untrusted-content safety | `--allow-local-files` opt-in guard; standard allow-list sanitizer | Capability-gated by default (no implicit network/asset fetch) + same-class sanitizer, positioned as multi-tenant-SaaS-safe from day one |
| Config | JS-capable config files (`.js`/`.ts`/`package.json`) | YAML/JSON/TOML only — JS config intentionally dropped to preserve the zero-JS mandate |
| Output profiles | Slides only (a slide-deck tool, full stop) | Slides in v1; architected for paged docs/reports/EPUB as later profiles (the core identity differentiator) |

---

## Sources

- `github.com/marp-team/marpit` — README (fetched), `docs/directives.md` (raw source, fetched), `docs/image-syntax.md` (raw source, fetched), `docs/theme-css.md` (raw source, fetched) — HIGH confidence, primary/official source.
- `github.com/marp-team/marp-core` — README (fetched), `themes/README.md` (fetched) — HIGH confidence, primary/official source.
- `github.com/marp-team/marp-cli` — README (fetched) — HIGH confidence, primary/official source.
- WebSearch cross-checks (MEDIUM confidence, verified against maintainer statements / linked official discussions, not training data alone): `@auto-scaling` directive scope and non-existence of a Markdown-level auto-scaling directive (marp-team/marp-core Discussion #186, Issue #72); line-highlight (`{1-3}`) not being a native marp-core feature (marp-team/marp-core Issues #168, #296, and community plugin `markdown-it-highlight-lines`); HTML sanitizer default allow-list and its v4 relaxation (marp-team/marp-core PR #74, #533); Marp CLI's actual input surface (no direct remote-URL input; server mode + stdin are the real mechanisms).
- `/Users/justin/dev/eden-press/PROPOSAL.md` — primary project design source; §2–§6 (architecture review, hard-problem deep-dives), §8 (phasing/objectives), §11–§12 (math and parser feasibility spikes, both run and verified 2026-07-20), §13 (differentiation rationale), §14 (attribution plan).
- `/Users/justin/dev/eden-press/.planning/PROJECT.md` — Requirements (Active/Out of Scope), Constraints, Key Decisions.
- Context7 was unavailable in this environment (no MCP tool present in the toolset provided); WebFetch of official GitHub-hosted docs/READMEs was used as the primary verification channel per the source-priority order (Context7 → Official Docs → WebSearch), with WebSearch reserved for cross-checking specific behavioral claims (auto-scaling, sanitizer, line-highlight, input types) against maintainer statements and linked official sources rather than raw docs pages (the marpit.marp.app site is a client-rendered SPA that returned only its page title to WebFetch — raw GitHub-hosted markdown source was used instead, which is the same underlying content).

---
*Feature research for: Eden Press (Go/Dart Marp-compatible document-generation framework)*
*Researched: 2026-07-20*
