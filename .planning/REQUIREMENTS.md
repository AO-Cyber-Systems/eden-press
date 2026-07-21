# Requirements: Eden Press

**Defined:** 2026-07-20
**Core Value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — emitting the document as structured data, not just HTML.

> Full design rationale, the two feasibility spikes, and stack decisions live in `PROPOSAL.md`; research corrections in `.planning/research/`. v1 = Marp-compatible slide decks; the architecture is format-agnostic (output-profile abstraction) so other document types follow as profiles, not rewrites.

## v1 Requirements

### Conformance (acceptance gate — precedes all engine work)

- [ ] **CONF-01**: A language-neutral golden corpus of Markdown→HTML/CSS cases exists, seeded from Marp's own Jest snapshot fixtures (MIT)
- [x] **CONF-02**: A test runner renders each case and compares **DOM-normalized** HTML (ignores cosmetic `<br>`/`<hr>`/whitespace/attr-order)
- [ ] **CONF-03**: A **CSS-AST diff** comparator exists for theme-CSS output (new tooling; not a DOM diff)
- [ ] **CONF-04**: The corpus covers the full CommonMark + GFM spec sweep (not only the 32-case parser spike), and is the acceptance gate cited by every engine objective

### Markdown & directives (`chase/markdown`, `chase/directive`)

- [x] **PARSE-01**: Parse Marpit Markdown via goldmark two-phase `Parse()`+`Render()` (never `Convert()`), so the finalized AST is available to downstream sinks
- [ ] **PARSE-02**: Resolve the directive system — global, local (carry-forward), and spot (`_`-prefixed) — via `parser.Context` state
- [ ] **PARSE-03**: Support both directive syntaxes: YAML front-matter (deck-level) and HTML-comment (`<!-- key: value -->`) directives
- [ ] **PARSE-04**: Implement the Marpit directive set — `theme`, `style`, `headingDivider`, `paginate`, `header`, `footer`, `class`, `color`, `backgroundColor`, `backgroundImage`, `backgroundPosition/Repeat/Size/Split`
- [ ] **PARSE-05**: Slide splitting on thematic breaks (`---`, incl. the setext-H2 trap) and `headingDivider`, wrapping each slide in `<section>` inside a `.marpit` container
- [ ] **PARSE-06**: Background-image syntax `![bg …](url)` (fit/split/position modifiers) → CSS backgrounds or the advanced-background SVG layer
- [ ] **PARSE-07**: Inline-SVG slide mode (`<svg><foreignObject><section>…`) — the blocking dependency for advanced backgrounds and auto-fit

### Theme-CSS engine (`chase/theme`)

- [ ] **THEME-01**: Build an in-memory `Stylesheet{Meta,Rules,Atoms}` model from `tdewolff/parse` css token stream (it is a token stream, not a node-AST)
- [ ] **THEME-02**: Parse theme metadata comments — `@theme` (required), `@size`, `@auto-scaling`
- [x] **THEME-03**: The scoping pipeline, in verified order: nesting down-level → `:root` remap/specificity-fix → selector-scope to container → `@import`/`@import-theme` resolve → render-time pagination + advanced-background injection
- [ ] **THEME-04**: A dedicated, independently tested **selector-rewriter** subsystem (no Go equivalent exists)

### Document model & profiles (`chase/model`, `chase/profile`) — first-class from day one

- [x] **MODEL-01**: A structured document model (slides, outline, speaker notes, metadata) derived from the finalized AST
- [x] **MODEL-02**: Emit the model as JSON alongside rendered HTML+CSS from the same source pass
- [x] **MODEL-03**: An output-**profile** interface separating shared core (parse/directives/model) from profile-specific rendering (container/layout/pagination/export)
- [x] **MODEL-04**: A `slides` profile (Marp-compatible) implemented against that interface as profile #1

### Batteries (`press/` — Marp-Core equivalent)

- [x] **CORE-01**: Bundle the three official themes (default/gaia/uncover) **verbatim** via `go:embed`, with preserved MIT headers
- [ ] **CORE-02**: `size` and `math` global directives (Marp-Core-level, not Marpit)
- [x] **CORE-03**: GFM tables + strikethrough (config `<s>` to match Marp, not goldmark's `<del>`) + line-break→`<br>`
- [x] **CORE-04**: Heading slug `id`s on `h1`–`h6`
- [ ] **CORE-05**: HTML allow-list sanitization matching Marp's `xss` policy (behavioral parity: strip-vs-escape semantics; hand-filter the GFM disallowed tags `<script>`/`<iframe>`/… that goldmark does not filter)
- [x] **CORE-06**: Emoji — shortcodes + unicode → twemoji, native (shortcode table + regex), no JS
- [ ] **CORE-07**: Syntax highlighting via `chroma`, with a bounded chroma-class ↔ highlight.js-class reconciliation so the bundled themes style code correctly
- [x] **CORE-08**: Math (`$…$`, `$$…$$`) via vendored/forked `latex2mathml` → native MathML; construct-detection (`\tag`/`\label`/complex `aligned`) auto-routes to the `codeberg.org/go-latex/latex` SVG/PNG fallback
- [ ] **CORE-09**: Auto-fit markers (`# <!--fit-->`, code/math shrink) emitted for the viewer-side helper; `@auto-scaling` honored as theme-CSS-only

### Library API (`press/`)

- [ ] **API-01**: `press.Render(md, opts) → {HTML, CSS, Model, Comments, Meta}` with **no Chrome dependency** (pure Go, no browser for HTML/structured output)
- [ ] **API-02**: The `press/` package must not import `chromedp` — enforced by CI (`go list -deps ./press/... | grep chromedp` is empty)
- [x] **API-03**: Stable, documented options (themes, math mode, highlight, inline-SVG, profile) and output types

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

- [x] **LIC-01**: `LICENSE` (MIT) for Eden Press
- [x] **LIC-02**: `NOTICE`/`CREDITS` crediting Marpit, Marp Core, Marp CLI + deps (goldmark, chroma, latex2mathml, go-latex/latex) with licenses
- [x] **LIC-03**: Per-file MIT headers preserving the original Marp copyright on verbatim-reused assets (3 themes + browser fit/polyfill script)
- [x] **LIC-04**: README acknowledgment ("inspired by & Markdown-compatible with Marp") + explicit "not affiliated with / endorsed by the Marp team"

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

Each v1 requirement maps to exactly one objective in `.planning/ROADMAP.md`.

| Requirement | Objective | Status |
|-------------|-----------|--------|
| CONF-01 | Objective 0 — Conformance Corpus, Acceptance Gate & Attribution Bootstrap | Pending |
| CONF-02 | Objective 0 | Complete |
| CONF-03 | Objective 0 | Pending |
| CONF-04 | Objective 0 | Pending |
| LIC-01 | Objective 0 | Complete |
| LIC-02 | Objective 0 | Complete |
| LIC-03 | Objective 0 (enforcement mechanism) + Objective 3 (per-file Marp headers applied, with CORE-01) | Complete |
| LIC-04 | Objective 0 | Complete |
| PARSE-01 | Objective 1 — chase/markdown + chase/directive + chase/theme | Complete |
| PARSE-02 | Objective 1 | Pending |
| PARSE-03 | Objective 1 | Pending |
| PARSE-04 | Objective 1 | Pending |
| PARSE-05 | Objective 1 | Pending |
| PARSE-06 | Objective 1 | Pending |
| PARSE-07 | Objective 1 | Pending |
| THEME-01 | Objective 1 | Pending |
| THEME-02 | Objective 1 | Pending |
| THEME-03 | Objective 1 | Complete |
| THEME-04 | Objective 1 | Pending |
| MODEL-01 | Objective 2 — chase/model + chase/profile + profiles/slides | Complete |
| MODEL-02 | Objective 2 | Complete |
| MODEL-03 | Objective 2 | Complete |
| MODEL-04 | Objective 2 | Complete |
| CORE-01 | Objective 3 — press/ Batteries + Public API | Complete |
| CORE-02 | Objective 3 | Complete |
| CORE-03 | Objective 3 | Complete |
| CORE-04 | Objective 3 | Complete |
| CORE-05 | Objective 3 | Complete |
| CORE-06 | Objective 3 | Complete |
| CORE-07 | Objective 3 | Complete |
| CORE-08 | Objective 3 (hardened further in Objective 8) | Complete |
| CORE-09 | Objective 3 (hardened further in Objective 8) | Complete |
| API-01 | Objective 3 | Complete |
| API-02 | Objective 3 | Complete |
| API-03 | Objective 3 | Complete |
| CLI-01 | Objective 4 — CLI (cmd/eden-press) | Pending |
| CLI-02 | Objective 4 | Pending |
| CLI-03 | Objective 4 | Pending |
| CLI-04 | Objective 4 | Pending |
| CLI-05 | Objective 4 | Pending |
| CLI-06 | Objective 4 | Pending |
| EXP-01 | Objective 5 — convert/pdf + convert/png (chromedp) | Pending |
| EXP-02 | Objective 5 | Pending |
| EXP-04 | Objective 5 | Pending |
| EXP-03 | Objective 6 — convert/pptx (native OOXML) | Pending |
| DART-01 | Objective 7 — Dart/Flutter Binding | Pending |
| DART-02 | Objective 7 | Pending |
| DART-03 | Objective 7 | Pending |
| DART-04 | Objective 7 | Pending |
| DART-05 | Objective 7 | Pending |

Objective 8 (Math-Fidelity Hardening + Auto-Fit Resolution) owns no new v1 requirement ID — it
hardens CORE-08 and CORE-09 (delivered in Objective 3) to production quality.

LIC-03 spans two objectives like CORE-08/09: Objective 0 delivers the ENFORCEMENT MECHANISM
(the `addlicense -check` CI gate + the Marp-copyright-preserving per-file header template in
CONTRIBUTING.md + the per-PR NOTICE/header checklist). The actual per-file Marp-preserving MIT
headers on the 3 vendored themes (default/gaia/uncover) + the browser-fit/polyfill script are
STAMPED in Objective 3, alongside CORE-01 (the verbatim `go:embed` of those assets) — the
mechanism exists day-one; the per-file headers land when the assets they annotate arrive.

**Coverage:**
- v1 requirements: 50 total (corrected from the initial 44 placeholder after full extraction)
- Mapped to objectives: 50/50 ✓
- Unmapped: 0 ✓
- v2 requirements (PROF-01..03, BRAND-01, SAFE-01, REPRO-01, EXP2-01, HL-01, GM2-01): intentionally not mapped — deferred, architecture must not preclude them

---
*Requirements defined: 2026-07-20*
*Last updated: 2026-07-20 — LIC-03 traceability annotated as an Obj-0 (mechanism) + Obj-3 (per-file headers) split, mirroring CORE-08/09; 100% v1 coverage unchanged*
