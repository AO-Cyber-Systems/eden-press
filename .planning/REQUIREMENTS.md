# Requirements: Eden Press

**Defined:** 2026-07-20
**Core Value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — emitting the document as structured data, not just HTML.

> Full design rationale, the two feasibility spikes, and stack decisions live in `PROPOSAL.md`; research corrections in `.planning/research/`. v1 = Marp-compatible slide decks; the architecture is format-agnostic (output-profile abstraction) so other document types follow as profiles, not rewrites.

## v1 Requirements

### Conformance (acceptance gate — precedes all engine work)

- [ ] **CONF-01**: A language-neutral golden corpus of Markdown→HTML/CSS cases exists, seeded from Marp's own Jest snapshot fixtures (MIT)
- [ ] **CONF-02**: A test runner renders each case and compares **DOM-normalized** HTML (ignores cosmetic `<br>`/`<hr>`/whitespace/attr-order)
- [ ] **CONF-03**: A **CSS-AST diff** comparator exists for theme-CSS output (new tooling; not a DOM diff)
- [ ] **CONF-04**: The corpus covers the full CommonMark + GFM spec sweep (not only the 32-case parser spike), and is the acceptance gate cited by every engine objective

### Markdown & directives (`chase/markdown`, `chase/directive`)

- [ ] **PARSE-01**: Parse Marpit Markdown via goldmark two-phase `Parse()`+`Render()` (never `Convert()`), so the finalized AST is available to downstream sinks
- [ ] **PARSE-02**: Resolve the directive system — global, local (carry-forward), and spot (`_`-prefixed) — via `parser.Context` state
- [ ] **PARSE-03**: Support both directive syntaxes: YAML front-matter (deck-level) and HTML-comment (`<!-- key: value -->`) directives
- [ ] **PARSE-04**: Implement the Marpit directive set — `theme`, `style`, `headingDivider`, `paginate`, `header`, `footer`, `class`, `color`, `backgroundColor`, `backgroundImage`, `backgroundPosition/Repeat/Size/Split`
- [ ] **PARSE-05**: Slide splitting on thematic breaks (`---`, incl. the setext-H2 trap) and `headingDivider`, wrapping each slide in `<section>` inside a `.marpit` container
- [ ] **PARSE-06**: Background-image syntax `![bg …](url)` (fit/split/position modifiers) → CSS backgrounds or the advanced-background SVG layer
- [ ] **PARSE-07**: Inline-SVG slide mode (`<svg><foreignObject><section>…`) — the blocking dependency for advanced backgrounds and auto-fit

### Theme-CSS engine (`chase/theme`)

- [ ] **THEME-01**: Build an in-memory `Stylesheet{Meta,Rules,Atoms}` model from `tdewolff/parse` css token stream (it is a token stream, not a node-AST)
- [ ] **THEME-02**: Parse theme metadata comments — `@theme` (required), `@size`, `@auto-scaling`
- [ ] **THEME-03**: The scoping pipeline, in verified order: nesting down-level → `:root` remap/specificity-fix → selector-scope to container → `@import`/`@import-theme` resolve → render-time pagination + advanced-background injection
- [ ] **THEME-04**: A dedicated, independently tested **selector-rewriter** subsystem (no Go equivalent exists)

### Document model & profiles (`chase/model`, `chase/profile`) — first-class from day one

- [ ] **MODEL-01**: A structured document model (slides, outline, speaker notes, metadata) derived from the finalized AST
- [ ] **MODEL-02**: Emit the model as JSON alongside rendered HTML+CSS from the same source pass
- [ ] **MODEL-03**: An output-**profile** interface separating shared core (parse/directives/model) from profile-specific rendering (container/layout/pagination/export)
- [ ] **MODEL-04**: A `slides` profile (Marp-compatible) implemented against that interface as profile #1

### Batteries (`press/` — Marp-Core equivalent)

- [ ] **CORE-01**: Bundle the three official themes (default/gaia/uncover) **verbatim** via `go:embed`, with preserved MIT headers
- [ ] **CORE-02**: `size` and `math` global directives (Marp-Core-level, not Marpit)
- [ ] **CORE-03**: GFM tables + strikethrough (config `<s>` to match Marp, not goldmark's `<del>`) + line-break→`<br>`
- [ ] **CORE-04**: Heading slug `id`s on `h1`–`h6`
- [ ] **CORE-05**: HTML allow-list sanitization matching Marp's `xss` policy (behavioral parity: strip-vs-escape semantics; hand-filter the GFM disallowed tags `<script>`/`<iframe>`/… that goldmark does not filter)
- [ ] **CORE-06**: Emoji — shortcodes + unicode → twemoji, native (shortcode table + regex), no JS
- [ ] **CORE-07**: Syntax highlighting via `chroma`, with a bounded chroma-class ↔ highlight.js-class reconciliation so the bundled themes style code correctly
- [ ] **CORE-08**: Math (`$…$`, `$$…$$`) via vendored/forked `latex2mathml` → native MathML; construct-detection (`\tag`/`\label`/complex `aligned`) auto-routes to the `codeberg.org/go-latex/latex` SVG/PNG fallback
- [ ] **CORE-09**: Auto-fit markers (`# <!--fit-->`, code/math shrink) emitted for the viewer-side helper; `@auto-scaling` honored as theme-CSS-only

### Library API (`press/`)

- [ ] **API-01**: `press.Render(md, opts) → {HTML, CSS, Model, Comments, Meta}` with **no Chrome dependency** (pure Go, no browser for HTML/structured output)
- [ ] **API-02**: The `press/` package must not import `chromedp` — enforced by CI (`go list -deps ./press/... | grep chromedp` is empty)
- [ ] **API-03**: Stable, documented options (themes, math mode, highlight, inline-SVG, profile) and output types

### CLI (`cmd/eden-press`)

- [ ] **CLI-01**: `eden-press <in.md>` → HTML output (default `bare`-style zero-JS static HTML)
- [ ] **CLI-02**: Watch mode (`fsnotify`) — rebuild on change
- [ ] **CLI-03**: Server mode with live-reload (serve local files, convert on request)
- [ ] **CLI-04**: Preview (open output in a browser)
- [ ] **CLI-05**: `--theme` / `--theme-set` loading
- [ ] **CLI-06**: Config file loading (YAML/JSON/TOML via koanf) + stdin input (`-`)

### Export (`convert/`)

- [ ] **EXP-01**: PDF via `chromedp` `Page.printToPDF`
- [ ] **EXP-02**: PNG/JPEG per-slide via `chromedp` screenshots
- [ ] **EXP-03**: PPTX via a **hand-rolled OOXML writer** (`archive/zip` + `encoding/xml`; reject `unioffice` — licensing + network check-in), consuming the docmodel directly (no Chrome), targeting **editable** text boxes
- [ ] **EXP-04**: Robust headless-Chrome discovery (`--browser-path`, known paths) and a MATH-font provisioning note (bundle official STIX Two Math OTF)

### Dart / Flutter client (`bindings/`)

- [ ] **DART-01**: A single Go C-ABI core with a JSON-in/JSON-out boundary (not mirrored structs)
- [ ] **DART-02**: Native binding via `dart:ffi` — Android `-buildmode=c-shared` (`.so`), iOS `-buildmode=c-archive` (`.a`); **no `gomobile bind`**
- [ ] **DART-03**: Web binding via `GOOS=js/wasm` + `wasm_exec.js` loader
- [ ] **DART-04**: JS-free Dart rendering surface — math via `flutter_math_fork`, highlight via `highlighting`/`flutter_highlighting`
- [ ] **DART-05**: Bindings pass a shared subset of the conformance corpus

### Licensing & attribution (early, first-class)

- [ ] **LIC-01**: `LICENSE` (MIT) for Eden Press
- [ ] **LIC-02**: `NOTICE`/`CREDITS` crediting Marpit, Marp Core, Marp CLI + deps (goldmark, chroma, latex2mathml, go-latex/latex) with licenses
- [ ] **LIC-03**: Per-file MIT headers preserving the original Marp copyright on verbatim-reused assets (3 themes + browser fit/polyfill script)
- [ ] **LIC-04**: README acknowledgment ("inspired by & Markdown-compatible with Marp") + explicit "not affiliated with / endorsed by the Marp team"

## v2 Requirements

Deferred differentiators (architecture in v1 must not preclude them).

### Profiles
- **PROF-01**: Paged-document profile (A4/Letter, running headers, page numbers, TOC)
- **PROF-02**: Single-page article profile
- **PROF-03**: EPUB profile

### Theming & branding
- **BRAND-01**: Design-token theming (JSON/TOML brand tokens → generated theme CSS) for multi-tenant branding

### Trust & reproducibility
- **SAFE-01**: Capability-gated hardened renderer for untrusted Markdown (no implicit network/asset fetch)
- **REPRO-01**: Byte-reproducible/deterministic output + content-hashed incremental rebuild

### Export quality
- **EXP2-01**: Tagged / accessible PDF (PDF/A-capable)

### Markdown extras
- **HL-01**: Code line-highlighting (`{1-3}`) — native differentiator over upstream (community-plugin-only in Marp)

### Platform
- **GM2-01**: Migrate to goldmark v2 (semantic-AST) once GA — aligns with the document-model design

## Out of Scope

| Feature | Reason |
|---------|--------|
| goja / embedded JS interpreter / Node / npm at runtime | Hard constraint — a JS-free backend is the entire point |
| Reactive/JS presenter components (Slidev/`bespoke` navigation) | Eden Press is server-side/static; viewer-side JS is out of scope |
| WYSIWYG editor / end-user app | That is Eden Docs' role; Eden Press is a library + CLI |
| Forking Marp source | Clean-room reimplementation of the dialect + theme format; reuse only MIT assets, with attribution |
| Reimplementing TeX layout or a Markdown grammar from scratch | Use native libs (goldmark, latex2mathml) instead |
| True remote-URL input | Marp CLI itself has none — stdin + server mode only |

## Traceability

Populated during roadmap creation (each requirement maps to exactly one objective).

**Coverage:**
- v1 requirements: 44 total
- Mapped to objectives: (pending roadmap)
- Unmapped: (pending roadmap)

---
*Requirements defined: 2026-07-20*
*Last updated: 2026-07-20 after initialization*
