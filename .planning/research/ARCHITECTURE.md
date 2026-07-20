# Architecture Research

**Domain:** Document-generation framework (Markdown → structured model + HTML/CSS + rasterized exports), Go core + Dart/Flutter client, Marp-compatible
**Researched:** 2026-07-20
**Confidence:** MEDIUM-HIGH (goldmark parser/renderer/ast APIs and chromedp APIs verified against official pkg.go.dev docs; Marpit's actual pipeline verified against its published source/docs; Dart FFI/WASM patterns are MEDIUM — verified against Flutter's own docs plus multiple independent tutorials, but no single canonical "Go+Flutter" official guide exists, and the tdewolff/parse/v2/css finding is a **correction** to the CSS-AST framing in PROPOSAL.md §4.2/§5.1 — see Pattern 5)

## Standard Architecture

### System Overview

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  cmd/eden-press  (Layer 3b — CLI, cobra)                                     │
│  render · watch · serve+live-reload · preview · convert --pdf/--pptx/--images│
└───────────────────────────────┬────────────────────────────────────────────┬─┘
                                 │ imports                        imports     │
┌────────────────────────────────▼───────────────────┐   ┌────────────────────▼─┐
│  press/  (Layer 2 — batteries + PUBLIC API)         │   │  convert/ (Layer 3a) │
│  press.Render(md, Options) → Output{HTML,CSS,Model} │   │  ONLY pkg touching   │
│  themes(embed) · emoji · chroma · latex2mathml      │   │  chromedp / OOXML    │
│  bluemonday sanitize · browser/fit asset            │──▶│  pdf.go png.go       │
│  ZERO Chrome dependency                             │   │  pptx/ (native OOXML,│
└───────────────────────────────┬──────────────────────┘  │  no chromedp)        │
                                 │ imports                 └───────────────────────┘
┌────────────────────────────────▼────────────────────────────────────────────┐
│  chase/  (Layer 1 — framework, NO themes, ~Marpit)                          │
│  ┌───────────┐ ┌───────────┐ ┌─────────┐ ┌────────┐ ┌─────────────────────┐ │
│  │ markdown/ │ │ directive/│ │ theme/  │ │ model/ │ │ profile/            │ │
│  │ goldmark  │ │ global/   │ │ Stylesh-│ │ docmod.│ │ Profile interface + │ │
│  │ Extenders │ │ local/    │ │ eet     │ │ (JSON  │ │ registry — slides/  │ │
│  │ (directive│ │ spot      │ │ model + │ │ tree,  │ │ paged/article/epub  │ │
│  │ syntax,   │ │ state     │ │ scoping │ │ shared │ │ plug in HERE only   │ │
│  │ boundary  │ │ machine,  │ │ passes  │ │ sink)  │ └─────────────────────┘ │
│  │ transform,│ │ no        │ │ over    │ └────────┘                        │
│  │ ![bg],    │ │ goldmark  │ │ tdewolff│                                   │
│  │ inline-   │ │ import    │ │ tokens  │                                   │
│  │ SVG)      │ │           │ │         │                                   │
│  └───────────┘ └───────────┘ └─────────┘                                   │
└──────────────────────────────────────────────────────────────────────────────┘
                                 ▲
                                 │ implements profile.Profile
                    ┌────────────┴────────────┐
                    │ profiles/slides (v1)     │  profiles/paged, article, epub (later — zero chase/* changes)
                    └──────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────┐
│  bind/  — Dart binding surface (depends only on press/'s stable API)         │
│  bind/capi  → C-ABI shim: PressRender(char*,char*) char*  +  PressFree      │
│    builds 3 ways: -buildmode=c-shared (.so, Android/NDK)                    │
│                    -buildmode=c-archive (.a, iOS, static-link)              │
│                    GOOS=js GOARCH=wasm (Web, loaded via wasm_exec.js)       │
│  bind/dart   → Flutter package: dart:ffi loader (native) / JS-interop       │
│                loader (web) — same JSON-in/JSON-out surface either way     │
└──────────────────────────────────────────────────────────────────────────────┘

conformance/  — golden corpus (from Marp's own Jest fixtures) + Go test runner
              — later: same corpus run against the compiled capi/wasm artifact
              — gates every layer above AND the Dart binding, not just chase/
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `chase/markdown` | goldmark Extenders: directive front-matter/comment syntax, slide/unit-boundary AST transform, `![bg]` image parser, inline-SVG container transform+render | goldmark `parser.BlockParser`/`InlineParser`/`ASTTransformer` + `renderer.NodeRenderer`, one `goldmark.Extender` per concern |
| `chase/directive` | Global/local/spot directive resolution + carry-forward state, **profile-agnostic** | Pure Go state machine, no goldmark import — called from a `parser.ASTTransformer` in `chase/markdown` via `parser.Context` |
| `chase/theme` | Theme CSS scoping: metadata parse, `:root`→section remap, selector scoping/prefixing, `@import` resolution, pagination/background rule injection | Own mutable `Stylesheet` model built once from `tdewolff/parse/v2/css` token/grammar stream (see Pattern 5) |
| `chase/model` | Structured document model (docmodel): `Document{Meta, Sections, Outline}` — the JSON-serializable sink | Direct recursive walk of the **finalized** goldmark AST (post-transform), materializing `text.Segment`s into owned strings |
| `chase/profile` | `Profile` interface + registry: what counts as a unit boundary, how units are wrapped/paginated/sized | Small interface package; profiles register themselves by name |
| `profiles/slides` | v1 output profile: Marp-compatible `.marpit`/`<section>`/inline-SVG, 16:9-style sizes, `paginate` semantics | Implements `profile.Profile`; imports `chase/*` only |
| `press` | Batteries + **the public Go API** (`press.Render`) | Embeds themes (`go:embed`), wires chroma/latex2mathml/emoji/bluemonday, selects a profile, calls `chase` |
| `convert` | Raster/OOXML export — **the only package that imports chromedp** | `chromedp` for PDF/PNG (drives Chrome via CDP); hand-built OOXML zip for PPTX (no Chrome) |
| `cmd/eden-press` | CLI argument parsing, watch/serve/preview | `cobra` + `fsnotify` + a websocket lib for live-reload |
| `bind/capi` + `bind/dart` | Single Go core exposed to Dart via FFI (native) and WASM (web) | `//export`'d C functions, JSON-in/JSON-out; `dart:ffi` / `dart:js_interop` loaders |
| `conformance` | Acceptance gate for every layer and every binding | Golden `input.md`/`expected.html`/`expected.css` cases + DOM/CSS-AST-normalized diff runner |

## Recommended Project Structure

```
github.com/AO-Cyber-Systems/eden-press/
├── chase/                    # Layer 1: framework — NO themes, NO batteries (~Marpit)
│   ├── markdown/             #   goldmark Extenders (directives, boundary, bg-image, inline-svg)
│   ├── directive/            #   global/local/spot resolution engine (pure, no goldmark dep)
│   ├── theme/                #   Stylesheet model + scoping passes (tdewolff/parse/v2/css)
│   ├── model/                #   docmodel: Document/Section/Block/Meta (JSON-serializable)
│   ├── profile/              #   Profile interface + registry
│   └── chase.go              #   internal Parse+Render entrypoint (NOT the public API)
├── profiles/
│   ├── slides/                #   v1: Marp-compatible profile.Profile implementation
│   ├── paged/                 #   later: paged-doc profile (A4/Letter, running headers, TOC)
│   ├── article/               #   later: single-page article profile
│   └── epub/                  #   later: EPUB packaging profile
├── press/                     # Layer 2: batteries + PUBLIC API (~Marp Core)
│   ├── themes/                #   default/gaia/uncover CSS, go:embed, per-file MIT headers
│   ├── emoji/  highlight/  math/  sanitize/  browser/
│   └── press.go                #   press.Render(md, Options) → Output{HTML, CSS, Model, Meta}
├── convert/                    # Layer 3a: exporters (importable, no CLI) — Chrome boundary
│   ├── chrome/                 #   chromedp session mgmt + browser discovery
│   ├── pdf.go  png.go
│   └── pptx/                   #   native OOXML from chase/model — NO chromedp
├── cmd/
│   └── eden-press/              # Layer 3b: the CLI (cobra)
├── bind/                        # Dart binding surface
│   ├── capi/                    #   C-ABI shim (builds .so / .a / .wasm from one source)
│   └── dart/                    #   Flutter package (pub): FFI + WASM loaders, same surface
└── conformance/                 # golden corpus + runner(s)
```

### Structure Rationale

- **`chase/` has five sibling packages, not one monolith**, because the task's own differentiators demand it: `profile/` and `model/` must be first-class from day one, not bolted on after v1 ships — that is the only way "paged/article/EPUB later" and "JSON output" stay additive. PROPOSAL.md §5.1's module layout buries the structured-model idea in prose (§13) but doesn't give it a package in the tree; this structure fixes that gap.
- **`chase/directive` has zero goldmark import.** The global/local/spot carry-forward state machine is pure logic (a slide/unit's resolved directive set given the previous unit's state + this unit's front-matter/comment tokens) — it doesn't need an AST to exist and is trivially unit-testable without parsing anything. `chase/markdown`'s `ASTTransformer` is a thin adapter that feeds parsed directive tokens into `chase/directive` and writes results back onto AST nodes via `parser.Context`.
- **`chase/theme` has zero dependency on `chase/markdown`.** CSS scoping is a standalone transform on CSS text (Marpit's own `ThemeSet`/`theme.js` works this way — it doesn't touch the Markdown AST at all). This lets the two hardest problems (§4.1/§4.2 in PROPOSAL.md) be built and tested in parallel by different objectives/agents.
- **`press/` is the only package most consumers import**, matching PROJECT.md's stated public surface (`press.Render(md)`). Everything under `chase/` and `profiles/` is effectively internal-but-exported Go (documented, stable-ish, but not the "front door") — consider an `internal/` prefix on `chase/` only if you want the compiler to enforce that consumers can't reach around `press/`; skip it if you want `profiles/paged` etc. to be independently importable by advanced consumers later.
- **`convert/` never appears as an import inside `chase/`, `profiles/`, or `press/`.** This is the differentiator-#2 boundary made structural, not just a docs promise — `go list -deps ./press/... | grep chromedp` should always return nothing. Consider a `go vet`/CI check that fails the build if it ever does.
- **`convert/pptx` depends on `chase/model`, not on rendered HTML/CSS and not on chromedp.** This is a genuinely different dependency edge from `convert/pdf`/`convert/png` — see Pattern 4 for why, and why it lets editable-PPTX export skip the Chrome dependency entirely (a real export-quality edge over Marp CLI, not just a Go-vs-Node one).
- **`bind/` depends only on `press/`'s public `Output`/`Options` types**, never on `chase/` internals — the FFI/WASM boundary is JSON-in/JSON-out (see Pattern 6), so it only needs the stable surface, keeping it buildable/versioned independently of engine internals.

## Architectural Patterns

### Pattern 1: Layering — `chase` (framework) → `press` (batteries+API) → `convert`+`cmd` (CLI-facing)

**What:** Three layers mirroring Marpit→Marp Core→Marp CLI, but as **two Go-idiomatic layers plus a hard external-dependency wall**, not three peer packages: `chase/*` (multiple sub-packages, no themes, no Chrome), `press` (single public package: batteries + facade), `convert`+`cmd/eden-press` (consumers of `press`, one of which — `convert` — is the sole Chrome-touching code in the whole module).

**When to use:** Any time a v1 profile/theme set should not gate the reusability of the parsing/directive/theme-scoping core, and any time "library with no browser dependency" is a hard requirement (both true here).

**Trade-offs:** A flat `press/parse`, `press/theme`, `press/model`, `press/profile` layout (dropping the `chase` name entirely) is a legitimate, slightly-less-discoverable alternative — PROPOSAL.md itself hedges this ("flat `press/...` layout is a fine alternative if the metaphor reads too obscure"). **Recommendation: keep the two-name split (`chase`/`press`)** — the printing-press metaphor is already the project's identity (Eden Press, chase = the frame that locks the type), and the *boundaries* below matter far more than whichever name wins; renaming `chase/*` to `press/*` later is a mechanical `mv` + import-path rewrite, not a redesign. Do **not** resurrect "loom" — that was the discarded "Eden Weave" naming (see PROJECT.md Key Decisions) and mixing metaphors mid-module would be confusing.

**Example:**
```go
// press/press.go — the public facade. Selects a profile, wires batteries, calls chase.
package press

func Render(md []byte, opts Options) (Output, error) {
    p := opts.profileOrDefault(profiles.Slides) // profile.Profile
    doc, pc, err := chase.Parse(md, opts.chaseOptions(p))
    if err != nil { return Output{}, err }
    html, css, err := chase.RenderHTML(doc, pc, p, opts.themeSet())
    if err != nil { return Output{}, err }
    model := chasemodel.Build(doc, md, pc) // same finalized AST, second sink
    return Output{HTML: html, CSS: css, Model: model, Meta: model.Meta}, nil
}
```

### Pattern 2: Output-profile abstraction (differentiator #1)

**What:** A `chase/profile.Profile` interface owns everything that differs between "Marp slides," "paged report," "single-page article," and "EPUB" — and nothing else. Parsing, directive syntax/resolution, and the theme-CSS scoping *passes* are profile-agnostic; a profile only supplies the tables/predicates those passes consult.

**Shared (never touched by adding a profile):**
- goldmark parsing + the Markdown dialect (directive comment/front-matter syntax, `![bg]` image syntax, GFM)
- `chase/directive`'s global/local/spot carry-forward state machine
- `chase/theme`'s scoping passes themselves (meta-parse → `:root`-remap → selector-scope → `@import`-resolve → inject)
- `chase/model`'s `Document`/`Section`/`Block` schema (a "Section" is what a profile happens to call a slide or a page)
- sanitization, emoji, highlighting, math batteries in `press/`

**Profile-specific (the interface surface):**
```go
package profile

type Profile interface {
    ID() string
    // Boundary reports whether this AST node starts a new Unit (thematic break
    // for slides; \pagebreak or a configured heading level for paged docs;
    // never, for a single-flow article).
    Boundary(n ast.Node, pc parser.Context) bool
    // Wrap supplies the physical container(s): .marpit/<section>/<svg> for
    // slides; <div class="page"> + @page CSS boxes for paged docs.
    Wrap(units []Unit, meta Meta) Container
    // Sizes resolves a `size`/`@size` directive/theme-meta value to concrete
    // dimensions (16:9/4:3 for slides; A4/Letter/Legal for paged docs).
    Sizes() SizeTable
    // Pagination describes how running numbers get injected into theme CSS —
    // `section::after` counters for slides vs `@page { @bottom-center }` for
    // paged docs.
    Pagination() PaginationRules
    // Directives lets a profile add profile-only directives (e.g. paged's
    // `toc:`/`runningHeader:`) into the shared directive engine's schema.
    Directives() []directive.Spec
}
```

**Trade-offs:** This buys profile-2/3/4 for the cost of one interface and a registry (`profile.Register(slides.New())`) — cheap insurance given the roadmap explicitly wants paged/article/EPUB to be additive objectives, not rewrites. The risk is over-abstracting for a v1 that only needs one profile; mitigate by writing the interface *from* what `profiles/slides` actually needs (bottom-up), not by speculatively generalizing before a second profile exists to validate the shape.

### Pattern 3: Structured document model — goldmark AST is transient, docmodel is the product

**What:** goldmark's `ast.Node` tree is a **parse-time working structure**, not a stable public artifact: it references source-buffer segments (`text.Segment{Start,End,Padding}`) rather than owning copies of text, has no built-in JSON marshaling, and its shape is an internal implementation detail of whichever goldmark version you're on. `chase/model` defines Eden Press's **own** versioned, JSON-serializable tree (`Document{Meta, Outline, Sections []Section{ID, Directives, Blocks, Notes, HTML}}`) built by a **direct recursive walk of the finalized AST**, materializing text via `Segment.Value(source)` into owned strings.

**The load-bearing implementation detail:** `press.Render` must call goldmark's **two-phase API** — `md.Parser().Parse(reader, parser.WithContext(pc))` then `md.Renderer().Render(w, source, doc)` — **not** the one-shot `goldmark.Convert()` convenience wrapper. `Convert()` never exposes the finalized `ast.Node` to the caller, so there is nowhere to hook a docmodel builder if you use it. Calling `Parse` and `Render` explicitly gives `press.Render` the same finalized AST (after all `ASTTransformer`s — directive resolution, unit-boundary splitting, background-image processing — have run) as input to **two independent sinks**: the HTML renderer (`renderer.NodeRenderer`s) and the docmodel builder (a plain recursive function, not itself a goldmark extension point). This is the literal mechanism behind differentiator #3 ("emit JSON and rendered HTML+CSS from the same source") — one parse, two consumers of one finalized tree.

**When to use:** Any time a caller (Eden-Biz, AOCore, an LLM ingestion pipeline) needs the outline/notes/metadata *without* parsing your HTML back out of your HTML.

**Trade-offs:** Building the docmodel is one more O(nodes) walk per render (cheap — same order as rendering itself) plus a schema you now own and must version (`docmodel.SchemaVersion`) independently of upstream Marp/goldmark churn.

**Example:**
```go
// chase/model/build.go
func Build(doc ast.Node, source []byte, pc parser.Context) *Document {
    d := &Document{}
    for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
        if isUnitBoundary(n) { // profile-tagged during the boundary ASTTransformer
            d.Sections = append(d.Sections, buildSection(n, source, pc))
        }
    }
    d.Outline = buildOutline(d.Sections)
    return d
}
```

### Pattern 4: goldmark extension architecture — where each concern hooks in

**What (verified against `pkg.go.dev/github.com/yuin/goldmark`, `/parser`, `/renderer`, HIGH confidence):**

| Concern | Hook | Notes |
|---|---|---|
| Directive front-matter (`---\nkey: val\n---`) | `parser.BlockParser` with `Trigger() []byte` returning `[]byte{'-'}`, low priority number (runs before `ParagraphParser`'s default 1000) | Parses YAML into a custom `ast.BaseBlock` node holding raw directive tokens |
| Inline HTML-comment directives (`<!-- paginate: true -->`) | Either goldmark's existing `RawHTMLParser`/HTML-block handling (comments already parse as HTML) **plus** a `parser.ASTTransformer` that scans for comment nodes matching `key: val` and attaches them to `parser.Context` | Avoid writing a new inline parser if default HTML-comment parsing already produces inspectable nodes — check first |
| Unit-boundary splitting (`---` thematic break → new slide/page) | `parser.ASTTransformer`, run **after** default parsing so `ast.ThematicBreak` nodes already exist; the transformer regroups siblings into synthetic `Unit`/`Section` container nodes | Registered via `parser.WithASTTransformers(util.Prioritized(t, N))` — priority controls transformer *ordering* relative to other custom transformers, not vs. the block/inline parse phase (that always runs first per goldmark's fixed 3-phase pipeline) |
| `![bg]` background-image syntax | `parser.InlineParser` with `Trigger() []byte{'!'}`, or reuse the default image parser's output and post-process alt-text (`bg`, `bg fit`, `bg right:40%`) via an `ASTTransformer` | The alt-text mini-grammar (`bg [fit\|cover\|contain] [left\|right[:pct]]`) is custom parsing regardless of hook choice |
| Inline-SVG container | `renderer.NodeRenderer` registered for the synthetic `Unit` node kind — write `<svg viewBox>...<foreignObject>` on `entering=true`, close on `entering=false` | The `entering bool` on `NodeRendererFunc` is exactly the open/close-tag mechanism needed to wrap groups of block children |
| Directive carry-forward state (global/local/spot) | `parser.Context.Set/Get/ComputeIfAbsent` with a `parser.NewContextKey()`-allocated key | This is goldmark's sanctioned mechanism for state that must survive across nodes within one parse; `chase/directive`'s state machine lives behind this key |

**Trade-offs:** goldmark's fixed pipeline (parse block → parse inline/delimiters → run `ASTTransformer`s, once, in one direction) means anything needing a *second* pass over the tree (e.g., "does this deck have any math at all, to decide whether to inject the MathML font `@font-face`") must be a separate `ASTTransformer` at higher priority number (runs later) than the transformers that produce the nodes it inspects — order your custom transformers deliberately, don't rely on registration order alone.

### Pattern 5: Theme-CSS scoping pipeline — CORRECTION to "CSS AST" framing

**What was assumed vs. verified:** PROPOSAL.md (§4.2, §5.1) frames the port as "CSS-AST transform over `tdewolff/parse/v2/css`." **Verified against official docs (pkg.go.dev): `tdewolff/parse/v2/css` is a streaming lexer/grammar parser (`GrammarType` + `Token` via repeated `Next()`/`Values()` calls), not a mutable, node-based tree** — it has no `NodeCSS` type, no parent/child pointers, no in-place mutation API. This matters architecturally: eden-press needs to build **its own** small in-memory intermediate model on top of the token stream before it can run Marpit's multi-pass pipeline (you cannot re-run a stream twice without buffering it into something).

**The pipeline, ordered (matches Marpit's actual `theme.js` — verified against its published source/docs):**
1. **Meta parse** — extract `/* @theme name */`, `@size`, `@auto-scaling` from leading comments (a single pass over the token stream, populating a `ThemeMeta` struct).
2. **Nesting down-level** *(optional — proposal recommends dropping this pass; target modern CSS nesting natively since export is Chrome and HTML target is modern engines)*.
3. **`:root` remap + specificity fix-up** — rewrite `:root` selectors to the profile's unit selector (`section`) while bumping specificity so `:root` still wins over a plain `section` rule if a theme mixes both (Marpit does this via a dedicated "increasing specificity" pseudo-class pass — a real, verified nuance, not a guess).
4. **Selector scoping/prefixing** — prefix/scope all selectors to the container (`.marpit`/profile root) so a theme's CSS can't leak into host-page styles when embedded.
5. **`@import`/`@import-theme` resolution** — resolve across the `ThemeSet` (multiple themes referencing each other).
6. **Render-time injection** (not baked into the static theme, computed per-render from directive state) — pagination rule (`section::after { content: attr(data-...-pagination) }` or profile-specific `@page` margin-box equivalent) and advanced-background rules (from `![bg]` directive state).

**Recommended intermediate model:**
```go
// chase/theme/stylesheet.go
type Stylesheet struct {
    Meta  ThemeMeta          // @theme, @size, @auto-scaling
    Rules []Rule             // ordered; built once from css.NewParser grammar events
    Atoms []AtRule           // @import, @media, etc.
}
type Rule struct {
    Selectors []Selector     // parsed selector list — needed to detect :root, do scoping
    Decls     []Declaration
}
// Each pass is a pure function: func(s *Stylesheet) — passes 1-5 run once per
// theme load; pass 6 runs once per render (depends on directive state).
// A final Print(*Stylesheet) string re-serializes to CSS text.
```

**Trade-offs:** This is one more package's worth of design work than "just transform the AST" implied — but it's the only sound way to run five-plus ordered, stateful passes over CSS with a token-stream library. (Alternative: `tdewolff/parse`'s sibling `css` printer/minifier utilities can help with re-serialization, but the rule-list model itself still has to be hand-rolled.)

### Pattern 6: Library API surface — the Chrome boundary (differentiator #2)

**What:** `press.Render()` is pure, deterministic, and has **zero transitive dependency on chromedp or any browser** — verify this continuously with `go list -deps ./press/... | grep -i chromedp` returning nothing, ideally as a CI check. `convert/` is the **only** package in the module that imports `chromedp`; everything it needs (finished, self-contained HTML+CSS, or the structured `Model` for OOXML) comes from `press.Output`, never from touching `chase/` internals directly.

**Two different export paths, two different dependency edges (a nuance not explicit in PROPOSAL.md):**
- `convert.ToPDF` / `convert.ToImages` — consume `Output.HTML`+`Output.CSS` (a **self-contained** document: no relative asset URLs Chrome would need to fetch), drive `chromedp` (`page.PrintToPDF`, `CaptureScreenshot`/`FullScreenshot`/`Screenshot`). Serve via a temp-file `file://` navigation (keeps the "no implicit network" ethos from differentiator #5 — no open port, no server) rather than a loopback HTTP server, unless/until asset volume makes inlining impractical.
- `convert.ToPPTX` (editable mode) — consumes `Output.Model` (the structured docmodel: text runs, images, positions) directly and emits OOXML DrawingML shapes. **No Chrome at all.** This is a real export-quality/architecture win over Marp CLI (which always needs Chrome or `soffice` for PPTX) — keep an optional `convert/legacy` image-per-slide fallback (chromedp screenshot per unit + zip) for parity if the native OOXML mapping can't cover a construct, but the native path should be the default.

**Batch/concurrency detail (verified against chromedp docs):** allocate **one** `chromedp.NewExecAllocator`/browser per worker process, then create **one child `chromedp.NewContext` (tab) per document render** — child contexts sharing a parent's already-allocated `Browser` get their own `Target` (tab) rather than spawning a new Chrome process each time. This is the concrete mechanism for the "batch export fleet" scaling tier below.

**Trade-offs:** Isolating Chrome this cleanly means `press/` (and everything under `chase/`) can be embedded in any Go service (Eden-Biz, AOCore) with **no external process at all** for HTML/JSON output — Chrome is opt-in, pay-as-you-go, only for teams that need rasterized output.

### Pattern 7: Dart binding architecture — one Go core, FFI + WASM, JSON at the boundary

**What (verified against Flutter's own docs + multiple cross-checked tutorials, MEDIUM confidence — no single canonical Go+Flutter reference exists, but the pattern is consistent across all sources):** a single Go package (`bind/capi`) with `//export`-annotated C functions compiles three ways from one source:

| Target | Build | Loaded via |
|---|---|---|
| Android | `CGO_ENABLED=1 GOOS=android GOARCH=arm64 go build -buildmode=c-shared` (NDK toolchain, one `.so` per ABI into `jniLibs/`) | `dart:ffi` `DynamicLibrary.open('libpress.so')` |
| iOS | `CGO_ENABLED=1 GOOS=ios go build -buildmode=c-archive` (static `.a` + header) | `dart:ffi` `DynamicLibrary.executable()`/`.process()` — statically linked into the app binary; **use the legacy `plugin_ffi` template**, not the newer `package_ffi` build-hooks path, specifically because static linking on iOS/macOS is what that legacy template still supports (per current Flutter docs) |
| Web | `GOOS=js GOARCH=wasm go build` | Go's own `wasm_exec.js` loader (a JS shim **Go ships**, not JS eden-press authors — same "external binary, not our code" framing as Chrome) registers Go functions as globals via `syscall/js`; Dart calls them through `dart:js_interop`/`package:web` (**not** `dart:html`/`package:js` — those don't compile if the Flutter app itself targets Wasm) |

**The API surface is deliberately minimal — JSON in, JSON out, plus an explicit free:**
```c
// bind/capi — the entire cross-language contract
char* PressRender(const char* markdownUTF8, const char* optionsJSON); // caller must PressFree() the result
void  PressFree(char* ptr);
```
This sidesteps the FFI marshalling constraint found consistently across research: dart:ffi's usable type set at the boundary is small (numbers, pointers, structs of those) — passing a rich `Output{HTML, CSS, Model, Meta}` means serializing to one JSON string once, not trying to mirror Go structs as Dart FFI structs. **Memory-management contract:** Go allocates the returned string with `C.CString` (or `js.CopyBytesToJS`-equivalent framing on Wasm); Dart is responsible for calling `PressFree` after copying the bytes out — treat "forgot to free" as a known leak-risk pitfall (see PITFALLS.md), not a hypothetical.

**Dart surface (`bind/dart`, a pub package):**
```dart
class Press {
  Future<PressOutput> render(String markdown, PressOptions opts) async {
    final json = await _platform.callRender(markdown, jsonEncode(opts)); // FFI or JS-interop under the hood
    return PressOutput.fromJson(jsonDecode(json));
  }
}
```
Client-side math/highlight/fit stay JS-free per PROPOSAL.md §7: `flutter_math_fork` (native KaTeX-layout port) or native MathML rendering, the pure-Dart `highlight` package, `TextPainter`-measured auto-fit.

**Shared conformance:** the `conformance/` corpus should gain a second runner that shells out to (or loads) the compiled `capi`/`wasm` artifact through the *same* `PressRender` JSON entrypoint the Dart code uses — this is what catches an FFI/WASM-boundary regression that a Go-only test run would miss (a serialization bug, a build-flag difference, a stale `wasm_exec.js`).

**Trade-offs:** Three build targets from one source is real cross-compile/toolchain setup cost (NDK, `-buildmode=c-archive` header wrangling, Wasm size ~2MB+ gzipped) — but it is one conformance pass instead of two engines, which PROPOSAL.md correctly identifies as the expensive alternative to avoid.

## Data Flow

### Render Flow (library path — no Chrome)

```
markdown []byte
    │
    ▼
chase/markdown extensions (goldmark Extenders: directive syntax, ![bg], inline-svg)
    │  parser.BlockParser / InlineParser register custom nodes
    ▼
goldmark block+inline parse  →  ast.Node tree (unfinalized)
    │
    ▼
parser.ASTTransformers run (priority-ordered):
  1. directive resolution (chase/directive state machine via parser.Context)
  2. unit-boundary regrouping (thematic breaks → Unit/Section nodes, profile.Boundary())
  3. background-image alt-text processing
    │
    ▼
ast.Node tree (FINALIZED — this is the fork point, Pattern 3)
    │
    ├──────────────────────────────┐
    ▼                              ▼
renderer.NodeRenderer walk    chase/model.Build() direct recursive walk
  → HTML string                 → Document{Meta, Outline, Sections} (docmodel)
    │                              │
    ▼                              ▼
chase/theme scoping pipeline   (Model attached to Output)
  (Pattern 5, passes 1-6)
    │
    ▼
press.Output{HTML, CSS, Model, Meta}   ← returned to caller, ZERO Chrome dependency
```

### Export Flow (convert/ path — Chrome or native OOXML)

```
press.Output{HTML, CSS, Model}
    │
    ├── HTML+CSS ──▶ temp file (self-contained, no relative asset fetch)
    │                    │
    │                    ▼
    │              chromedp: one child context (tab) per render,
    │              shared long-lived browser per worker process
    │                    │
    │           ┌────────┴────────┐
    │           ▼                 ▼
    │      page.PrintToPDF   CaptureScreenshot/FullScreenshot
    │        → PDF bytes        → PNG/JPEG bytes
    │
    └── Model ──▶ convert/pptx: walk Sections/Blocks → OOXML DrawingML shapes
                     → .pptx zip (editable text boxes)   [NO Chrome]
```

### Key Data Flows

1. **One parse, two sinks (differentiator #3):** the finalized AST feeds both the HTML renderer and the docmodel builder — this is what makes "JSON alongside HTML" a fork in the pipeline rather than a second parse or a reverse-engineering of HTML back into structure.
2. **PPTX bypasses Chrome entirely** by consuming the docmodel instead of rendered HTML — a structurally different (shorter, cheaper, more reliable) path than PDF/PNG, and a genuine differentiator over Marp CLI's screenshot-per-slide/soffice approach.
3. **Dart/WASM/FFI reuses the exact same `press.Render` code path** — the C-ABI shim is a thin JSON marshaling wrapper around `press.Render`, not a reimplementation, so this "flow" has no divergence to maintain in lockstep beyond the conformance corpus's boundary runner (Pattern 7).

## Scaling Considerations

This is a library + CLI, not a horizontally-scaled multi-user web app — "scale" here means single-render performance, concurrent embedding in a host service (Eden-Biz/AOCore), and batch-export throughput.

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single CLI invocation / small deck | No concerns — in-process `press.Render`; a fresh Chrome process per CLI run for `--pdf`/`--images` is fine (this is what upstream `marp-cli` does too). |
| Embedded in a multi-tenant service (Eden-Biz), moderate concurrency | `press.Render` must be safe to call concurrently across goroutines with **no shared mutable package-level state** — each call gets its own `parser.Context`; a shared `goldmark.Markdown` instance (built once, configured, read-only after `New()`) is safe to reuse across goroutines since all mutable state lives in the per-call `Context`, not the `Markdown` value. Verify this explicitly in tests (race detector on concurrent `press.Render` calls) before shipping it as a claimed property. For Chrome export at this tier: pool a small number of long-lived browser processes (one `NewExecAllocator` per worker), one tab/context per request — never spawn-a-browser-per-request (first real bottleneck: Chrome process spawn latency ~100s of ms + ~150-300MB RSS per instance). |
| Batch/SaaS-scale export fleet | Move `convert/` (the Chrome-touching tier) into its own worker pool/queue, decoupled from the pure-Go `press.Render` tier — this is exactly why the Pattern-6 boundary matters operationally, not just architecturally: the stateless, dependency-free `press/` tier scales trivially (add goroutines/replicas), while the Chrome tier scales as a bounded pool sized to available CPU/RAM, independently. |

### Scaling Priorities

1. **First bottleneck: Chrome process lifecycle**, the moment any export path is used behind more than a handful of concurrent requests. Fix: reuse-browser/one-tab-per-render pattern (Pattern 6), never per-request process spawn.
2. **Second bottleneck: large single decks** (hundreds of slides/pages) — the docmodel walk and CSS scoping passes are O(n) and should be fine, but the inline-SVG-per-unit rendering mode and any per-unit `chromedp.Screenshot` calls (PNG export) are the first place to profile if a very large deck is slow.

## Anti-Patterns

### Anti-Pattern 1: Baking "slide" (or 16:9, or `<section>`) into `chase/theme` or `chase/model`

**What people do:** Name things `Slide`, hardcode `section` as the unit selector inside the scoping engine, assume a 16:9/4:3 size table is universal.
**Why it's wrong:** This is precisely the "1-1 Marp port" trap the task is designed to avoid — it means adding the paged-doc profile later requires touching the framework core, not just registering a new `profile.Profile`, defeating differentiator #1's entire purpose.
**Do this instead:** Name the shared concept `Unit`/`Section` in `chase/model` and `chase/theme`; let `profiles/slides` be the thing that calls a unit a "slide," sets 16:9 as a default size, and wires `section` as its container selector via `Profile.Wrap()`/`Profile.Sizes()`.

### Anti-Pattern 2: Using goldmark's one-shot `Convert()` in `press.Render`

**What people do:** Call the convenience `goldmark.Convert(source, writer)` because it's the README's first example.
**Why it's wrong:** It never exposes the finalized `ast.Node` to the caller — there is no hook point left to build the docmodel from the same tree that produced the HTML, forcing either a second parse (fidelity-risk, perf cost) or reverse-engineering structure out of rendered HTML (fragile, lossy).
**Do this instead:** Call `md.Parser().Parse(reader, parser.WithContext(pc))` then `md.Renderer().Render(w, source, doc)` explicitly, and build the docmodel from the same `doc` in between (Pattern 3).

### Anti-Pattern 3: Letting `convert` (or chromedp) leak into `chase`/`press` imports

**What people do:** "Just for now," reference `chromedp` from inside a theme or math package to do some measurement/rendering trick (e.g., for auto-fit).
**Why it's wrong:** Silently breaks the "library with zero Chrome dependency" promise (differentiator #2) — a consumer embedding `press` in a server for pure JSON/HTML output suddenly pulls in a browser-automation dependency transitively.
**Do this instead:** Anything that needs a real browser (auto-fit measurement, rasterization) belongs in `convert/`, gated behind an explicit opt-in by the caller. Enforce with a CI check on `go list -deps`.

### Anti-Pattern 4: Building the CSS scoping engine as ad-hoc string concatenation over `tdewolff/parse/v2/css`'s token stream

**What people do:** Since `tdewolff/parse/v2/css` gives you a stream, not a tree, it's tempting to do all six scoping passes in one single-pass "read token, maybe rewrite, write token" loop.
**Why it's wrong:** Marpit's real pipeline is multi-pass and order-dependent (meta must be extracted before `:root`-remap; import resolution must happen after both; pagination injection needs to know the final selector shape) — a single-pass approach can't express "look at the whole stylesheet, then decide" transforms like specificity fix-up or cross-theme `@import` resolution.
**Do this instead:** Buffer the token stream into an owned `Stylesheet{Meta, Rules, Atoms}` model once (Pattern 5), run each pass as `func(*Stylesheet)`, serialize once at the end.

### Anti-Pattern 5: Treating "pure-Dart reimplementation" as the default Dart strategy

**What people do:** Start a second Markdown/CSS engine in Dart (using `package:markdown` + `csslib`) because it avoids toolchain complexity (NDK, cgo, Wasm size).
**Why it's wrong:** PROPOSAL.md already identifies this as the expensive path — two engines to keep in fidelity-lockstep against a moving Marp upstream, doubling the conformance-corpus maintenance burden indefinitely.
**Do this instead:** Default to Go→FFI+WASM (Pattern 7); reserve a pure-Dart "lite" subset as an explicitly-scoped fallback *only if* WASM size/perf is proven unacceptable for a specific Flutter target, and validate it against a documented subset of the corpus rather than silently drifting.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Chrome/Chromium | `convert/chrome`: discovery via env var, known install paths, `--browser-path` flag; drive via `chromedp` (`NewExecAllocator`+`NewContext`) | Binary dependency only — same external as upstream `marp-cli`; no JS authored in our code. Reuse one browser process, one tab per render (Pattern 6). |
| LibreOffice (`soffice`) | Optional, `convert/legacy` only | Only needed if you keep an editable-PPTX-via-conversion fallback; the native OOXML path (Pattern 6) doesn't need it — treat as legacy/parity, not core. |
| STIX Two Math / Latin Modern Math (font) | Bundled asset or Chrome-environment font install | Required for native MathML to render correctly (not tofu) in headless Chrome during PDF export — an operational/deployment dependency, not a code dependency. |
| pub.dev: `flutter_math_fork`, `highlight` | Dart client-side dependencies | Keep the Flutter client JS-free per PROPOSAL.md §7; these replace KaTeX/highlight.js on the client the same way chroma/latex2mathml replace them server-side. |
| Eden-Biz / AOCore (internal consumers) | Import `press` as a Go module dependency; call `press.Render()` in-process | No network boundary needed for HTML/JSON output — this is the entire point of the library-first design (differentiator #2). |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `chase/markdown` ↔ `chase/directive` | Go function calls via `parser.Context` | `directive` has no goldmark import — a deliberate one-way, testable-in-isolation boundary. |
| `chase/theme` ↔ everything else in `chase/` | None (by design) | Theme scoping is a standalone CSS-text transform; no shared types with the Markdown/AST side except the `ThemeMeta`/`SizeTable` values a `profile.Profile` passes in. |
| `profiles/*` ↔ `chase/*` | Implements `profile.Profile`, imports `chase/model`+`chase/theme` types | One-directional: `chase/*` never imports `profiles/*` (no import cycle; registry pattern in `press/` wires them). |
| `press` ↔ `convert` | `press.Output` value (HTML+CSS+Model), passed by value/pointer, no shared mutable state | `convert` imports `press`'s public types; `press` never imports `convert`. |
| Go core ↔ Dart | `bind/capi`'s JSON-in/JSON-out C-ABI; FFI (native) or JS-interop-over-Wasm (web) | The only cross-language boundary in the system; kept intentionally narrow (two functions) to sidestep FFI marshalling constraints (Pattern 7). |

## Build Order / Dependency Graph (feeds roadmap objective ordering)

This translates PROPOSAL.md §8's phasing into explicit package-dependency edges, so objectives can be sequenced by what must exist before what — not just by rough size estimate.

```
0. conformance/ corpus + normalizer         (no deps — pure data + a Go test harness)
        │
        ▼
1a. chase/directive   (pure logic, no goldmark)         ─┐  parallel-buildable;
1b. chase/theme       (Stylesheet model + passes,          different objectives/
    depends only on tdewolff/parse/v2/css)               ─┘  agents can own these
        │
        ▼
2. chase/markdown     (goldmark Extenders: directives,
   boundary-transform, ![bg], inline-svg — wires 1a into
   parser.Context; independent of 1b)
        │
        ▼
3. chase/model        (docmodel builder — walks the AST
   chase/markdown produces; needs Unit/Section boundary
   nodes to exist first)
        │
        ▼
4. chase/profile      (Profile interface — can actually be
   written in parallel with 1-3 since it's mostly an
   interface definition; VALIDATE it by building it
   bottom-up from what profiles/slides needs, not before)
        │
        ▼
5. profiles/slides    (v1 Profile impl — needs 1-4 all
   present; this is where "Marp-compatible" gets proven
   against the conformance corpus from step 0)
        │
        ▼
6. press/             (batteries: themes go:embed, emoji,
   chroma, latex2mathml, bluemonday + press.Render facade
   — needs 5; this is the "Importable Go API" milestone
   and the point at which Dart binding work can start)
        │
        ├──────────────────────────────┐
        ▼                              ▼
7. convert/pdf.go+png.go            8. convert/pptx (native
   (chromedp; needs only press's       OOXML; needs only
   Output.HTML/CSS — can be built      chase/model's schema
   and tested against ANY static       — can be built and
   HTML, decoupled from the rest       tested independently
   of the engine)                      of chromedp entirely)
        │                              │
        └──────────────┬───────────────┘
                        ▼
9. cmd/eden-press       (CLI; needs press/ + convert/*)
                        │
                        ▼
10. bind/capi + bind/dart  (needs ONLY press/'s stable
    Options/Output types — explicitly gated on step 6
    being API-stable, not on 7-9)
                        │
                        ▼
11. conformance/ boundary runner (extends step 0's corpus
    to exercise the compiled capi/wasm artifact through
    the same JSON entrypoint bind/dart uses)
                        │
                        ▼
12. Auto-fit + math-fidelity hardening (cross-cutting —
    touches press/math (converter bug fixes per PROPOSAL.md
    §11), profiles/slides (fit markers), bind/dart (native
    TextPainter fit) — genuinely last because it depends on
    the rest existing to have something to fit/tune)
```

**Key ordering implications for the roadmap:**
- Steps **1a/1b** and, loosely, **4** can be parallelized across objectives/agents — they have no edges between them.
- Step **5** (v1 profile) is the true gate for "does this look like Marp yet" and should consume the conformance corpus from step 0 as its acceptance test, exactly as PROPOSAL.md's phasing intends.
- Steps **7** and **8** are siblings, not sequential — PPTX-native doesn't need Chrome and can be an independent objective from PDF/PNG once step 6 lands.
- Step **10** (Dart) should be gated on **6** (API stability), not on **9** (CLI) — PROPOSAL.md's own phasing already gets this right (Dart depends only on "Importable Go API"); this graph makes the *reason* explicit (Dart only touches `press`'s public types, never `chase/`, `convert/`, or `cmd/`).
- Step **12** is last by necessity (nothing to tune until the rest renders something), not by low priority — don't let "small size estimate" cause it to be scheduled too early and then blocked.

## Sources

- PROPOSAL.md (this repo) — primary design source; architecture review of Marpit/Marp Core/Marp CLI, JS-free stack decisions, two completed feasibility spikes (§11 math, §12 parser parity), differentiation rationale (§13)
- [goldmark (yuin/goldmark)](https://pkg.go.dev/github.com/yuin/goldmark) — `Markdown`/`Extender`/`New` exact signatures, 3-phase pipeline diagram (HIGH confidence, official docs)
- [goldmark/parser](https://pkg.go.dev/github.com/yuin/goldmark/parser) — `ASTTransformer`, `BlockParser`, `InlineParser`, `Context`, priority/registration options (HIGH confidence, official docs)
- [goldmark/renderer](https://pkg.go.dev/github.com/yuin/goldmark/renderer) — `NodeRenderer`, `NodeRendererFunc`, `entering` bool mechanism (HIGH confidence, official docs)
- [goldmark/ast (github.com/yuin/goldmark/blob/master/ast/ast.go)](https://github.com/yuin/goldmark/blob/master/ast/ast.go) + [pkg.go.dev/.../ast](https://pkg.go.dev/github.com/yuin/goldmark/ast) — `Walk`, `WalkStatus`, segment-based text model (HIGH confidence)
- [Marpit (marp-team/marpit)](https://github.com/marp-team/marpit) + [Marpit DeepWiki](https://deepwiki.com/marp-team/marpit) + [theme.js source](https://marpit-api.marp.app/theme.js.html) — actual pipeline order (meta → nesting → root-replace → section-size → import-parse), `:root` specificity issue ([#232](https://github.com/marp-team/marpit/issues/232)), pagination via `section::after` (MEDIUM-HIGH — DeepWiki-summarized source, cross-checked against multiple Marpit doc pages)
- [tdewolff/parse/v2/css](https://pkg.go.dev/github.com/tdewolff/parse/v2/css) — confirms streaming `GrammarType`/`Token` model, **no** node-based AST/`NodeCSS` type (HIGH confidence, official docs; this is a correction to PROPOSAL.md's "CSS AST" framing, see Pattern 5)
- [chromedp](https://pkg.go.dev/github.com/chromedp/chromedp) — `NewExecAllocator`/`NewContext` browser/tab model, `Run`, `CaptureScreenshot`/`FullScreenshot`/`Screenshot`, multi-tab-one-browser batch pattern, `page.PrintToPDF` via `cdproto/page` (HIGH confidence, official docs)
- [Flutter: Binding to native iOS code using dart:ffi](https://docs.flutter.dev/platform-integration/ios/c-interop) + [Android C interop](https://docs.flutter.dev/platform-integration/android/c-interop) — c-shared/c-archive platform build strategy, legacy `plugin_ffi` vs newer `package_ffi`+build-hooks for static linking (MEDIUM confidence, official Flutter docs, cross-checked against multiple independent tutorials: openprivacy.ca, dev.to)
- [Dart: WebAssembly (Wasm) compilation](https://dart.dev/web/wasm) + [JavaScript interoperability](https://dart.dev/interop/js-interop) — `dart:js_interop`/`package:web` requirement when the Flutter app itself targets Wasm, vs legacy `dart:html`/`package:js` (MEDIUM-HIGH confidence, official Dart docs)
- Go `syscall/js` + `wasm_exec.js` pattern — cross-checked across multiple independent tutorials (aaron-powell.com, blog.lazyhacker.com); no single official "Go+Flutter Wasm" doc exists, so held at MEDIUM confidence
- FFI marshalling constraints (JSON-at-the-boundary, memory-free contract) — consistent across multiple independent Go+Flutter FFI tutorials (dev.to ×2, roothex200.hashnode.dev, Medium) — MEDIUM confidence (pattern consensus, not a single authoritative source)

---
*Architecture research for: Eden Press (Go/Dart document-generation framework, Marp-compatible)*
*Researched: 2026-07-20*
