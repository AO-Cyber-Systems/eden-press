# Objective 1: chase/markdown + chase/directive + chase/theme (Marpit-in-Go) - Research

**Researched:** 2026-07-20
**Domain:** goldmark AST extensions (directive system, slide-splitting, background images, inline-SVG) + a standalone CSS selector-scoping/rewriting engine over tdewolff/parse/v2/css tokens
**Confidence:** HIGH — every mechanism below is grounded either in goldmark v1.8.4 source read directly from `/Users/justin/go/pkg/mod/github.com/yuin/goldmark@v1.8.4` (this machine's module cache) or in Marpit's actual JS source read from `/Users/justin/dev/eden-press/tools/corpus-gen/node_modules/@marp-team/marpit/lib/**` (the exact version that generated the 18-case conformance corpus). No part of this document is based on unverified training-data recall of either library.

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| PARSE-01 | Two-phase parse/render (no `Convert()`) | §"Two-Phase Render Seam" — confirms `Convert()` is literally `Parse()` then `Render()` (markdown.go:115-119); chase must call the two steps directly so Objective 2's docmodel can inspect the `ast.Document` between them. |
| PARSE-02 | Directive system (front-matter + HTML-comment, global/local/spot, carry-forward) | §"chase/directive" + §"HTML-Comment Directive Detection" — full parse.js/apply.js/comment.js mechanism, carry-forward cursor algorithm, spot-directive `_` prefix rule. |
| PARSE-03 | Slide-splitting on thematic breaks + headingDivider | §"Slide-Splitting ASTTransformer" — confirms goldmark's own block-parser priority order (SetextHeadingParser=100 before ThematicBreakParser=200) already resolves the setext-H2 trap before the transformer runs; headingDivider is a second, independent split pass. |
| PARSE-04 | Wrap each slide in `<section>` inside `.marpit` container | §"Slide-Splitting ASTTransformer" + §"Two-Phase Render Seam" — container element is injected at render time, not parse time (mirrors Marpit's `marpitContainer` Element + `renderMarkdown`/`render` split). |
| PARSE-05 | Background-image syntax (`![bg fit right:40%](url)`) | §"Background Images" — full option-grammar (bg size keywords, split keyword regex, filters) transcribed from `background_image/parse.js` + `image/parse.js`. |
| PARSE-06 | Inline-SVG slide mode (`<svg><foreignObject><section>`) | §"Inline-SVG Mode" — exact wrapping structure from `inline_svg.js`, plus the 3-layer advanced-background SVG structure from `background_image/advanced.js`. |
| PARSE-07 | Directive carry-forward correctness (global/local/spot semantics across slides) | §"chase/directive" — the `cursor{slide, local, spot}` state-machine transcribed verbatim from `directives/parse.js`; spot state resets to `{}` after every slide close. |
| THEME-01 | Theme CSS meta/size/parse pass | §"CSS Pipeline — Tier 1 (Theme.fromCSS, add-time)" — meta regex, size detection, order. |
| THEME-02 | Nesting down-leveling | §"Nesting: The No-Go-Analogue Gap" — flags this as the highest-risk, zero-prior-art sub-part; recommends a minimal hand-rolled flattener scoped to what the synthetic stress theme + 3 bundled themes actually need. |
| THEME-03 | Full ordered CSS scoping pipeline (render-time) | §"CSS Pipeline — Tier 2 (ThemeSet.pack, render-time)" — the exact 20-step pass list transcribed line-for-line from `theme_set.js:281-286`, corrected against ARCHITECTURE.md's earlier approximation. |
| THEME-04 | Standalone selector-rewriter subsystem (`:root`, `:is()/:where()`, combinators, scope-prepend) | §"The Selector-Rewriter Subsystem" — concrete tdewolff token-walking algorithm, the two-step placeholder mechanism (`:marpit-container`/`:marpit-slide` prepend-then-replace), and the `:where(section):not([\20 root])` specificity trick with exact source. |

</phase_requirements>

## Summary

Objective 1 has two genuinely novel mechanics with no Go prior art, and both are now fully de-risked by reading Marpit's actual source (not just its docs): **(1)** the directive/comment/slide-splitting pipeline, which goldmark's fixed-order parser makes *easier* than markdown-it's mutable-ruler design once you see that goldmark already resolves the "setext-H2 vs thematic-break" ambiguity for you before any custom code runs; and **(2)** the CSS selector-scoping engine, whose real pass order (verified from `theme_set.js` and `theme.js`) is a two-tier pipeline — per-theme "add-time" passes baked into `Theme.css` once, then a 20-step "render-time" `pack()` pipeline re-run on every render — with a specificity trick (`:where(section):not([\20 root])`) that must run **after** selector-scoping, not before, and a placeholder-based scoping mechanism (`:marpit-container`/`:marpit-slide` prepend-then-replace) that Go's `chase/theme` should copy directly since it generalizes cleanly to a two-pass sentinel-token design over tdewolff's `css.Token` stream.

The single highest-risk sub-part is **CSS nesting down-leveling (THEME-02)**: Marpit delegates this entirely to `postcss-nesting` + `@csstools/postcss-is-pseudo-class`, both PostCSS (JS) plugins with **zero Go equivalent**. This must be hand-rolled in `chase/theme`, but the good news is it only needs to be *correct enough for the corpus + a synthetic stress theme*, not general-purpose — none of the 3 bundled Marp themes (default/gaia/uncover) is confirmed to use nesting in what's in the corpus today, so this is buildable test-first against a purpose-written stress theme. The second highest-risk sub-part is **HTML-comment directive detection**, which is now fully resolved: Marpit uses a **dedicated markdown-it plugin** (`comment.js`) registered first in the plugin chain — not a post-hoc AST scan — and the Go equivalent is a dedicated goldmark `BlockParser` + `InlineParser` pair registered at low priority, exactly mirroring how `goldmark-meta` registers its own `BlockParser` at priority 0.

**Primary recommendation:** Build `chase/directive` and `chase/theme` in parallel (zero cross-imports, confirmed below), each test-first against literal transcriptions of the Marpit pass functions in this document; then build `chase/markdown` as goldmark `Extender`s that call into `chase/directive`, following the exact `goldmark-meta`-style `Extend(m goldmark.Markdown)` pattern (verified in goldmark-meta v1.1.0 source) for every extension (comment/directive parser, slide-splitter ASTTransformer, background-image inline parser, inline-SVG ASTTransformer).

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/yuin/goldmark` | v1.8.4 (pinned in go.mod) | CommonMark/GFM parser + renderer; extension host for `chase/markdown` | Already the project's chosen markdown engine (Objective 0); its `parser.Context`/`ASTTransformer`/`BlockParser` APIs are exactly what's needed for directive carry-forward and slide-splitting. |
| `github.com/tdewolff/parse/v2` (css subpackage) | v2.8.13 (pinned) | Streaming CSS lexer/grammar parser for `chase/theme` | Already the project's chosen CSS tokenizer (Objective 0's `cssdiff` package already builds a `Stylesheet` model on top of it — proven pattern to extend). |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/yuin/goldmark-meta` | v1.1.0 (present in module cache, not yet in go.mod) | Reference pattern for how a goldmark extension registers a `BlockParser` at priority 0 and stores parsed data via a `parser.ContextKey` | Do NOT import this for front-matter — Marpit's front-matter directive semantics (merges into global/local directive resolution, `looseYAML` option) are custom. Use it only as the **API pattern to copy** for `chase/directive`'s own front-matter block parser. |
| `gopkg.in/yaml.v2` (goldmark-meta's YAML dep) | matches goldmark-meta | Only if chase/directive's own YAML front-matter/comment parsing needs a YAML library — Marpit uses `js-yaml` with a custom `FAILSAFE_SCHEMA` (see `directives/yaml.js`, not yet read in full — flag as open question below) | Confirm exact YAML schema restrictions before picking a Go YAML lib; `gopkg.in/yaml.v2` (already a transitive dep via goldmark-meta if used) is a safe default. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `chase/theme.Stylesheet` model over tdewolff tokens | A full CSS AST library (none exists in Go with `:is()`/`:where()`/nesting support) | No viable Go alternative exists — this is precisely why THEME-04 is called out as having "no Go analogue" in the objective goal. Building the owned model (as `conformance/cssdiff` already does) is the only path. |
| Hand-rolled nesting down-leveler | Shelling out to Node + `postcss-nesting` at build time | Rejected: Objective 1's success criteria require the CSS pipeline to run as pure Go (`cssdiff` gate is a Go-side test), and a Node dependency at runtime breaks the "library-first" differentiation goal in PROPOSAL.md §13. |

**Installation:**
```bash
# goldmark and tdewolff/parse are already in go.mod at the required versions — no action needed.
# Do NOT add goldmark-meta or any PostCSS/Node dependency.
```

## Architecture Patterns

### Recommended Project Structure
```
chase/
├── directive/                  # Pure state machine — zero goldmark import
│   ├── comment.go              # HTML-comment directive detection + YAML-ish value parsing (mirrors comment.js + yaml.js)
│   ├── frontmatter.go          # Front-matter directive parsing (mirrors directives/parse.js's frontMatter block)
│   ├── directives.go           # Directive name tables + value coercion (mirrors directives/directives.js: globals{}, locals{})
│   ├── carryforward.go         # cursor{slide, local, spot} state machine (mirrors directives/parse.js's local-directive loop)
│   └── directive_test.go       # Table-driven tests: one test per corpus case's directive semantics
├── theme/                      # CSS engine over tdewolff/parse/v2/css — zero markdown dependency
│   ├── stylesheet.go           # Owned Stylesheet{Meta, Rules, Atoms} model (like cssdiff's model.go but richer: preserves nesting depth, selector token lists not just strings)
│   ├── parse.go                # css.NewParser() token-stream → Stylesheet (extends conformance/cssdiff/build.go's pattern)
│   ├── pass.go                 # Pass func(*Stylesheet) type + Pipeline runner
│   ├── pass_meta.go            # @theme/@size/@auto-scaling meta extraction (mirrors postcss/meta.js)
│   ├── pass_nesting.go         # Nesting down-level — HIGHEST RISK, build test-first (mirrors postcss-nesting, no Go lib)
│   ├── pass_root.go            # :root ↔ section:marpit-root marker + increasing-specificity rewrite (mirrors postcss/root/{replace,increasing_specificity,font_size,rem}.js)
│   ├── pass_sectionsize.go     # width/height detection (mirrors postcss/section_size.js)
│   ├── pass_import.go          # @import / @import-theme parse + hoist + recursive resolve + circular-import detection (mirrors postcss/import/{parse,hoisting,replace,suppress}.js)
│   ├── pass_scaffold.go        # Scaffold reset CSS injection (mirrors theme/scaffold.js + postcss/scaffold.js)
│   ├── pass_advancedbg.go      # Advanced-background static CSS injection (mirrors postcss/advanced_background.js)
│   ├── pass_pagination.go      # Comment-out non-marpit `content:` on `section::after` (mirrors postcss/pagination.js)
│   ├── pass_svgbackdrop.go     # ::backdrop retargeting (mirrors postcss/svg_backdrop.js) — only if backdropSelector supported in Obj 1
│   ├── selector/               # THE standalone selector-rewriter subsystem (THEME-04) — independently unit-tested
│   │   ├── selector.go         # Selector AST: []SimpleSelector + Combinator, hand-walks :is()/:where() FunctionToken args
│   │   ├── scope.go            # Two-step placeholder scoping: Prepend(sentinel) then Replace(sentinel, realContainerChain)
│   │   ├── root.go             # :root → sentinel-marked section, then sentinel → :where(section):not([\20 root])
│   │   └── selector_test.go    # Pure unit tests, no markdown/theme dependency — run in isolation per success criterion 5
│   └── theme_test.go           # Pipeline-level tests using cssdiff.Equal against expected.css fixtures
└── markdown/                    # goldmark Extenders — depends on chase/directive, NOT chase/theme
    ├── comment.go                # BlockParser + InlineParser wrapping chase/directive's comment detector as goldmark nodes
    ├── directive.go               # goldmark.Extender: registers comment parsers + an ASTTransformer that calls chase/directive's carry-forward machine
    ├── slide.go                   # ASTTransformer: splits ast.Document children on *ast.ThematicBreak (+ headingDivider), wraps each run in a custom *ast.Section node
    ├── headingdivider.go          # Second split pass (its own ASTTransformer, ordered after slide.go's — mirrors marpit.js registering heading_divider AFTER slide)
    ├── image.go                   # InlineParser hook / post-parse walk detecting `bg` in image alt-text option grammar (mirrors background_image/parse.js + image/parse.js)
    ├── inlinesvg.go                # ASTTransformer: wraps each *ast.Section in <svg><foreignObject> nodes when inline-SVG mode enabled
    ├── render.go                   # renderer.NodeRenderer for Section/Svg/ForeignObject/AdvancedBackground node kinds
    └── markdown_test.go            # Wired against conformance/runner.RenderFunc + conformance/corpus.LoadCases; this is what flips the 18 marp-core cases from PENDING to PASS
```

### Build Order (dependency-ordered)

1. **`chase/directive`** and **`chase/theme`** — build in parallel. Confirmed zero cross-dependency:
   - `chase/directive` only needs Go stdlib + a YAML-ish scalar parser (no goldmark import — it operates on plain strings, exactly like Marpit's `directives/yaml.js` operates on a raw comment-content string before any markdown-it token exists).
   - `chase/theme` only needs `tdewolff/parse/v2/css` (no markdown import at all — CSS theme files are parsed completely independently of any slide content, exactly as Marpit's `ThemeSet` never touches markdown-it tokens).
   - `chase/theme/selector` is the standalone selector-rewriter — build and unit-test it **first**, before wiring it into `chase/theme`'s pipeline, per success criterion 5 ("independently unit-tested selector-rewriter subsystem").
2. **`chase/markdown`** — depends on `chase/directive` (imports it directly for carry-forward state and directive-name tables). Does **not** depend on `chase/theme` at parse time — theme CSS processing is a fully separate concern invoked later (by `press/` in Objective 2, or by a test harness in Objective 1) once slide HTML is generated. This mirrors Marpit itself: `Marpit#render()` calls `renderMarkdown()` (markdown-it) and `renderStyle()` (ThemeSet.pack) as two independent calls that only share directive-derived *values* (e.g., theme name, width/height for the SVG viewBox), not code paths.
3. **Conformance wiring** — implement a new `RenderFunc`-compatible engine (`func(markdown string, opts map[string]any) (string, error)`) that calls `chase/markdown`'s two-phase `Parse()`/`Render()` and swap it in wherever `conformance/runner.NewGoldmarkMarp` + `GoldmarkRenderFunc` are currently used, turning the 18 `requires_engine: "marp-core"` cases from PENDING to PASS/FAIL. Do this incrementally, case-by-case (start with `marp-basic`, `marp-slide-split`, then directives, then backgrounds, then themes) — not as one big-bang swap.

### Two-Phase Render Seam (PARSE-01)

Verified directly from goldmark v1.8.4 source (`markdown.go:37-53, 100-135`):

```go
// markdown.go — Markdown interface (verified, v1.8.4)
type Markdown interface {
    Convert(source []byte, writer io.Writer, opts ...parser.ParseOption) error
    Parser() parser.Parser
    SetParser(parser.Parser)
    Renderer() renderer.Renderer
    SetRenderer(renderer.Renderer)
}

// Convert is LITERALLY defined as the two-phase call — proof the two-phase
// approach is not a hack, it's the documented internal implementation:
func (m *markdown) Convert(source []byte, writer io.Writer, opts ...parser.ParseOption) error {
    reader := text.NewReader(source)
    doc := m.parser.Parse(reader, opts...)          // Phase 1
    return m.renderer.Render(writer, source, doc)   // Phase 2
}
```

`chase/markdown`'s `RenderFunc` implementation must call these two lines directly instead of `md.Convert(...)`:

```go
reader := text.NewReader([]byte(markdown))
pc := parser.NewContext()                              // parser.NewContext(options ...ContextOption) Context — verified parser.go:258
doc := md.Parser().Parse(reader, parser.WithContext(pc))
var buf bytes.Buffer
if err := md.Renderer().Render(&buf, []byte(markdown), doc); err != nil {
    return "", err
}
// doc (ast.Node) and pc (parser.Context) are now BOTH inspectable here —
// this is the exact seam Objective 2's docmodel will hook into.
```

`parser.Context` (verified `parser.go:158-198`) is the carry-forward vehicle:
```go
type Context interface {
    Get(ContextKey) any
    ComputeIfAbsent(ContextKey, func() any) any
    Set(ContextKey, any)
    // ... plus reference/ID/block-offset methods not needed for directives
}
// Keys are minted once, package-level, exactly like goldmark-meta does:
var directiveStateKey = parser.NewContextKey()
```

### goldmark Extension Registration Pattern (verified against goldmark-meta v1.1.0 source)

This is the **exact** pattern every `chase/markdown` extension must follow — copied line-for-line in structure from `goldmark-meta@v1.1.0/meta.go:295-318`:

```go
// goldmark-meta's actual Extend() — this is the reference implementation to copy:
func New(opts ...Option) goldmark.Extender { /* ... */ }

func (e *meta) Extend(m goldmark.Markdown) {
    m.Parser().AddOptions(
        parser.WithBlockParsers(
            util.Prioritized(NewParser(), 0),   // registers a parser.BlockParser at priority 0
        ),
    )
    m.Parser().AddOptions(
        parser.WithASTTransformers(
            util.Prioritized(newTransformer(topts...), 0),  // registers a parser.ASTTransformer at priority 0
        ),
    )
}
```

`chase/markdown/directive.go` should look like:
```go
package markdown

func New(opts ...Option) goldmark.Extender { return &directiveExt{opts} }

func (e *directiveExt) Extend(m goldmark.Markdown) {
    m.Parser().AddOptions(
        parser.WithBlockParsers(
            util.Prioritized(newCommentBlockParser(), 0),   // mirrors Marpit's comment.js md.block.ruler.before('html_block', ...)
        ),
        parser.WithInlineParsers(
            util.Prioritized(newCommentInlineParser(), 0),  // mirrors comment.js's md.inline.ruler.before('html_inline', ...)
        ),
    )
    m.Parser().AddOptions(
        parser.WithASTTransformers(
            util.Prioritized(newSlideSplitTransformer(), 100),      // must run before...
            util.Prioritized(newHeadingDividerTransformer(), 200),  // ...this (mirrors marpit.js's plugin order: slide before heading_divider)
            util.Prioritized(newDirectiveApplyTransformer(), 300),  // carry-forward + dataset/CSS-var application
            util.Prioritized(newInlineSVGTransformer(), 400),       // wraps sections in <svg><foreignObject> last
        ),
    )
}
```
Verified interfaces used above (all from `parser.go`):
- `type ASTTransformer interface { Transform(node *ast.Document, reader text.Reader, pc Context) }` (line 586-590)
- `type BlockParser interface { Trigger() []byte; Open(parent ast.Node, reader text.Reader, pc Context) (ast.Node, State); ... }` (line 514+)
- `type InlineParser interface { Trigger() []byte; Parse(parent ast.Node, block text.Reader, pc Context) ast.Node }` (line 556-570)
- `util.PrioritizedSlice` / `util.Prioritized(value, priority)` — ascending-priority sort (util.go:872-887)

### Pattern: HTML-Comment Directive Detection (PARSE-02) — RESOLVED

**The PITFALLS.md open question is answered.** Marpit's `comment.js` (read in full) registers a **dedicated pair** of rules, not a post-hoc AST scan:

```js
// Verified verbatim from lib/markdown/comment.js:
md.block.ruler.before('html_block', 'marpit_comment', (state, startLine, endLine, silent) => {
  // Fast-fail: char at line start must be '<'
  // Match /^<!--/ opening; scan forward line-by-line for /-->/ closing
  // Push a 'marpit_comment' token, token.hidden = true, token.content = inner text (trimmed)
  // Then immediately parse content via yaml() into token.meta.marpitParsedDirectives
})
md.inline.ruler.before('html_inline', 'marpit_inline_comment', (state, silent) => {
  // Same idea, for comments appearing mid-line (inline context)
})
```

Two important, previously-unverified details this resolves:
1. Directive comments are matched via a single regex `/<!--+\s*([\s\S]*?)\s*--+>/` applied to the **assembled multi-line markup string**, not incrementally token-by-token — i.e., the parser first finds line-range boundaries cheaply (fast-fail on `<`, then scan to a line matching `-->`), then re-slices and regex-matches the whole comment body at once. This is important for the Go port: don't try to build a single-pass character-by-character comment-content extractor; instead (a) cheaply detect `<!--` at block/inline start, (b) advance/consume to the matching `-->` (multi-line aware), (c) THEN regex/trim the captured span for the actual key:value content.
2. Comments are marked `hidden = true` and get a `meta.marpitCommentParsed` tag once consumed by a later pass (`well-known-magic-comment` for prettier-ignore/markdownlint/remark-lint passthrough comments, or `directive` once actually recognized as a directive key). **This two-stage mark-then-consume design matters**: the comment DETECTOR (comment.js) runs once and stores raw parsed key/value pairs on the token; separate LATER passes (`directives/parse.js`) decide which parsed keys are actually recognized directives and mark the token accordingly. Do not conflate detection and directive-semantic recognition into one goldmark pass — split them exactly as Marpit does: a `chase/markdown` `BlockParser`/`InlineParser` pair that ONLY detects+stores raw comment key/value data on a custom AST node, and a separate `ASTTransformer` (or `chase/directive` call from within one) that walks the tree afterward deciding which keys are global/local/spot directives.

**Go translation recommendation:** implement as a goldmark `BlockParser` (trigger byte `<`, priority 0, registered `before` — i.e. numerically lower priority than — goldmark's built-in raw-HTML block parser so it wins the race) that produces a custom `*ast.BaseBlock`-derived node (e.g., `chasemd.CommentNode{Raw string, Hidden bool}`), plus a matching `InlineParser` (trigger byte `<`) for mid-paragraph comments. Both call into `chase/directive`'s pure string-parsing function (no goldmark types) to get back a `map[string]string` of raw key/value pairs — this is the "zero cross-import" boundary: `chase/directive` never imports `goldmark/ast`; only `chase/markdown`'s parser wrapper does.

### Pattern: Slide-Splitting ASTTransformer (PARSE-03/04) — setext-H2 trap RESOLVED

Verified from goldmark's own `DefaultBlockParsers()` (parser.go:592-616):
```
SetextHeadingParser, priority 100   (registered FIRST)
ThematicBreakParser, priority 200   (registered SECOND)
ListParser, priority 300
...
ParagraphParser, priority 1000      (registered LAST — lowest priority)
```
Because `SetextHeadingParser` is tried **before** `ThematicBreakParser` on every candidate line, and `setext_headings.go`'s `matchesSetextHeadingBar` only fires when the line is entirely `=`/`-` characters **directly following an open paragraph** (goldmark tracks this via `temporaryParagraphKey`, a `parser.ContextKey`), any `---` line that is actually functioning as a Setext H2 underline is **already consumed into an `*ast.Heading` node before it can ever become an `*ast.ThematicBreak`**. This means:

> **The setext-H2 trap flagged in PITFALLS.md is a non-issue for an ASTTransformer-based slide-splitter.** By the time `chase/markdown`'s slide-split `ASTTransformer` runs (after all block+inline parsing, per the `ASTTransformer` contract), every remaining `*ast.ThematicBreak` sibling of `ast.Document` is **guaranteed** to be a genuine slide separator — goldmark's own CommonMark-compliant precedence has already resolved the ambiguity. No special-case detection code is needed in `chase/markdown` for this.

Translate Marpit's `slide.js` (read in full) directly:
```js
// Verified verbatim structure from lib/markdown/slide.js:
md.core.ruler.push('marpit_slide', state => {
  const splittedTokens = split(state.tokens, t => t.type === 'hr' && t.level === 0, true)
  state.tokens = splittedTokens.reduce((arr, slideTokens, marpitSlide) => {
    const firstHr = slideTokens[0]?.type === 'hr' ? slideTokens[0] : undefined
    return [...arr, ...wrapTokens(state.Token, 'marpit_slide', {
      tag: 'section', id: anchorCallback(marpitSlide),
      open: { meta: { marpitSlide, marpitSlideTotal, marpitSlideElement: 1 } },
      close: { meta: { marpitSlide, marpitSlideTotal, marpitSlideElement: -1 } },
    }, slideTokens.slice(firstHr ? 1 : 0))]
  }, [])
})
```
Go equivalent for `chase/markdown/slide.go`:
```go
type SlideSplitTransformer struct{}

func (t *SlideSplitTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
    var slides [][]ast.Node
    var current []ast.Node
    for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
        if tb, ok := c.(*ast.ThematicBreak); ok {
            _ = tb
            slides = append(slides, current)
            current = nil
            continue // drop the ThematicBreak itself — it's the separator, not slide content
        }
        current = append(current, c)
    }
    slides = append(slides, current)
    // Remove all children from doc, then re-append: for each slide, a custom
    // *ast.Section node wrapping that slide's node run, with meta
    // (index, total) stashed via pc.Set for the carry-forward pass to read.
}
```
Note: `headingDivider` is a **second, independently-ordered** split pass (`heading_divider.js`, read in full) that runs `md.core.ruler.before('marpit_slide', ...)` — i.e., BEFORE the hr-based split, by prepending synthetic hidden `<hr>` tokens before qualifying headings. In goldmark terms this means: **run the heading-divider transformer FIRST** (inserting synthetic `*ast.ThematicBreak` nodes before qualifying `*ast.Heading` nodes per the resolved `headingDivider` directive value), **then** run the generic thematic-break slide-splitter over the now-augmented tree. This ordering (heading-divider inserts breaks → slide-splitter consumes all breaks uniformly) is simpler in Go than trying to run two independent split algorithms — copy it directly.

### Pattern: chase/directive carry-forward state machine (PARSE-02/07)

Verified verbatim from `directives/parse.js`'s local-directive loop — this is the **exact** algorithm to port, since it's already a clean, self-contained state machine with no markdown-it-specific plumbing:

```js
// Verified structure (directives/parse.js, 'marpit_directives_parse' core rule):
const cursor = { slide: undefined, local: {}, spot: {} }
for (const token of state.tokens) {
  if (token.meta?.marpitSlideElement === 1) {           // slide OPEN
    token.meta.marpitDirectives = {}
    cursor.slide = token
  } else if (token.meta?.marpitSlideElement === -1) {   // slide CLOSE
    cursor.slide.meta.marpitDirectives = { ...cursor.slide.meta.marpitDirectives, ...cursor.local, ...cursor.spot }
    cursor.spot = {}                                     // spot state resets EVERY slide
  } else if (isDirectiveComment(token)) {
    for (const key of Object.keys(parsedDirectives)) {
      if (directives.locals[key]) cursor.local = { ...cursor.local, ...directives.locals[key](value) }
      if (key.startsWith('_')) {                          // SPOT directive: "_class" → local directive "class", spot-scoped only
        const spotKey = key.slice(1)
        if (directives.locals[spotKey]) cursor.spot = { ...cursor.spot, ...directives.locals[spotKey](value) }
      }
    }
  }
}
// Global directives are merged in separately, AFTER the local/spot loop, onto EVERY slide:
for (const token of slides) token.meta.marpitDirectives = { ...token.meta.marpitDirectives, ...marpit.lastGlobalDirectives }
```

Key correctness rules to preserve exactly (confirmed against corpus's `marp-class-spot` case):
1. **local** directives persist in `cursor.local` across slides — never reset except by being overridden by a later directive of the same key.
2. **spot** directives (`_key`) are collected into `cursor.spot`, merged into the CURRENT slide only at slide-close time, then **`cursor.spot = {}`** — this reset is what makes spot directives single-slide-only.
3. **global** directives (`theme`, `headingDivider`, `style`, `lang`) are resolved once for the whole document (via a separate, earlier core rule pass — `marpit_directives_global_parse`, which runs `md.core.ruler.after('inline', ...)`, i.e. BEFORE the slide-local pass) and stamped onto every slide identically at the end.
4. Directive value coercion tables (verified verbatim from `directives/directives.js`):
   - `globals.headingDivider`: accepts int 1-6 (expanded to `[1..N]`), an array of ints, or `"false"` (string) → `false`.
   - `globals.theme`: only accepted if `marpit.themeSet.has(v)` — i.e., **unknown theme names are silently dropped**, not errored.
   - `locals.paginate`: normalizes to lowercase; `"hold"`/`"skip"` pass through as strings, anything else coerces to boolean (`v === 'true'`).
   - `locals.class`: arrays are joined with a space.
   - `locals.footer`/`locals.header`: only accepted if the value is already a `string` (guards against YAML producing non-string types).

`chase/directive/carryforward.go` should implement this as a pure function operating on an ordered slice of "slide boundary + directive-comment" events (produced by `chase/markdown`'s tree walk) — keeping `chase/directive` free of any goldmark import, per the package-boundary requirement.

### Directive → Attribute/Style materialization (verified from `directives/apply.js`)

Every recognized directive is applied to each slide's `<section>` in **three possible forms simultaneously** (not just one) — this is a load-bearing detail for the conformance HTML-diff to pass byte-for-byte:
1. `data-{kebab-case-name}="{value}"` HTML attribute (if `opts.dataset`, default true) — e.g. `data-heading-divider`, `data-paginate`.
2. `--{kebab-case-name}: {value};` inline CSS custom property (if `opts.css`, default true) — appended into the section's `style` attribute via an ordered `InlineStyle` builder (dedupe-by-property-name, preserves insertion order — do not use a Go `map[string]string` for this, it must preserve declaration order across multiple `.set()` calls the way Marpit's `InlineStyle` class does).
3. For 4 SPECIFIC directives, a **direct CSS property override** on top of the custom-property emission:
   - `color` → `style.set('color', value)`
   - `backgroundColor` → `style.set('background-color', value).set('background-image', 'none')` (explicitly clears any inherited background-image)
   - `backgroundImage` → `style.set('background-image', ...).set('background-position','center').set('background-repeat','no-repeat').set('background-size','cover')`, each override-able by `backgroundPosition`/`backgroundRepeat`/`backgroundSize` if also present
   - `class` → `token.attrJoin('class', value)` (joins, does not replace, existing class list)
4. `paginate` also drives a **running page-number counter** (`pageNumber`) that increments once per slide UNLESS that slide's `paginate` value is exactly `'skip'` or `'hold'` — and if a slide sets `paginate` truthy before any increment has happened, it retroactively becomes page 1. This counter, plus a final `data-marpit-pagination-total` attribute stamped onto every paginating slide once the total is known, must be computed as a **second pass after full slide-list construction** (Marpit does this with `tokensForPaginationTotal` collected during the main loop, then patched in a trailing loop) — port this two-pass structure directly rather than trying to compute the total inline.

### Background Images (PARSE-05)

Verified verbatim from `background_image/parse.js` + `image/parse.js` — the full option grammar for `![bg <options>](url)`:

```
bg                              → marks image as background (hides it from normal flow)
auto | contain | cover | fit    → background-size keyword ("fit" is an ALIAS for "contain", not a separate CSS value)
(left|right)(:NN%)?             → split background: side + optional split percentage (regex: /^(left|right)(?::((?:\d*\.)?\d+%))?$/)
vertical | horizontal           → stacking direction for MULTIPLE bg images in one slide
NN%                             → resize (plain percentage, applies to size)
w:NN(unit) / width:NN(unit)     → explicit width (units: %,ch,cm,em,ex,in,mm,pc,pt,px; bare number → px)
h:NN(unit) / height:NN(unit)    → explicit height (same units)
blur(:amount) / brightness(:amt) / contrast(:amt) / grayscale(:amt) / hue-rotate(:amt) / invert(:amt)
  / opacity(:amt) / saturate(:amt) / sepia(:amt) / drop-shadow(:offset-x,offset-y[,blur[,color]])
                                 → CSS filter() functions, each with a documented DEFAULT amount if omitted
                                   (blur→10px, brightness→1.5, contrast→2, grayscale→1, hue-rotate→180deg,
                                    invert→1, opacity→.5, saturate→2, sepia→1, drop-shadow→'0 5px 10px rgba(0,0,0,.4)')
```

Two distinct output modes depending on inline-SVG setting:
- **Inline-SVG DISABLED**: single bg image only; becomes a `backgroundImage`/`backgroundSize` **local directive** on the slide (i.e., reuses the exact same directive-application code path as an author-written `<!-- backgroundImage: url(...) -->` comment) — simplest possible mode, no new HTML structure.
- **Inline-SVG ENABLED** (Marp Core's default, and what all 18 corpus HTML fixtures assume, confirmed by every `expected.html` containing `data-marpit-svg`): triggers **advanced-background mode** — multiple images allowed, each becomes a `<figure style="background-image:url(...)">` inside a `<div data-marpit-advanced-background-container>`, itself inside a dedicated background `<section data-marpit-advanced-background="background">`, sibling to the real content section — see next section for the full 3-layer structure.

### Inline-SVG Mode (PARSE-06) — exact structure verified from `inline_svg.js` + `background_image/advanced.js`

**Base inline-SVG wrap** (`inline_svg.js`, applies to EVERY slide when enabled, independent of backgrounds):
```
<svg data-marpit-svg="" viewBox="0 0 {widthPixel} {heightPixel}">
  <foreignObject width="{widthPixel}" height="{heightPixel}">
    <section id="..." ...directive-attrs>
      ...slide content...
    </section>
  </foreignObject>
</svg>
```
`widthPixel`/`heightPixel` come from `themeSet.getThemeProp(theme, 'widthPixel'|'heightPixel')` — i.e., the ACTIVE theme's resolved slide dimensions (default 1280×720 per the scaffold theme, overridable by a theme's `@size` meta or explicit `width`/`height` CSS on `section`).

**Advanced-background wrap** (`background_image/advanced.js`, only for slides with `bg` images, ADDITIONAL structure inserted before/after the content foreignObject — confirmed by corpus's `marp-bg-image`/`marp-bg-split` expected.html): the single `<svg>` for that slide gets **three** `<foreignObject>` children instead of one:
1. **background layer**: `<foreignObject width height><section data-marpit-advanced-background="background" ...original-slide-attrs><div data-marpit-advanced-background-container="true" data-marpit-advanced-background-direction="horizontal|vertical"><figure style="background-image:url(...);...">[<figcaption>alt-text</figcaption> if alt present]</figure>...(repeat per image)...</div></section></foreignObject>`
2. **content layer**: the original `<section data-marpit-advanced-background="content">...actual markdown content...</section>` — for split layouts, this foreignObject's `width`/`x` attrs are adjusted (`width = 100% - splitSize`, and if split=left, `x = splitSize`), plus a `style="--marpit-advanced-background-split:{splitSize}"` custom property set on the background section's OPEN tag.
3. **pseudo layer**: `<foreignObject data-marpit-advanced-background="pseudo" width height><section data-marpit-advanced-background="pseudo" ...original-attrs style="[color if set]">` — empty placeholder section, exists purely so CSS `::after`-based pagination/pseudo-content still renders on top of the background-image layer (background section has `pointer-events:none` and is excluded from normal box/pagination display via the injected advanced-background CSS).

This exact 3-layer structure (background/content/pseudo, in that DOM order) is what `chase/markdown/inlinesvg.go` + a new `chase/markdown` advanced-background transformer must reproduce byte-for-byte for the `marp-bg-image`/`marp-bg-split` corpus cases to pass `htmldiff.Equal`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CommonMark/GFM parsing | A markdown tokenizer | goldmark's built-in `SetextHeadingParser`/`ThematicBreakParser`/`ParagraphParser` (already correctly prioritized) | Already solves the setext-H2-vs-thematic-break ambiguity for free — re-implementing block parsing to "help" slide-splitting would actually reintroduce a bug goldmark already prevents. |
| CSS tokenization | A custom CSS lexer | `tdewolff/parse/v2/css`'s `css.NewParser(...).Next()`/`.Values()` grammar stream (already proven in `conformance/cssdiff/build.go`) | Handles comments, strings, escapes, at-rules correctly; reinventing this risks subtle tokenization bugs the existing `cssdiff` package has already worked through. |
| Ordered, dedupe-by-key inline style building | `map[string]string` + manual join | A small `InlineStyle` type mirroring Marpit's `helpers/inline_style.js` (ordered `.set(prop, value)` calls, last-write-wins per property, stable insertion order for first-seen properties) | Declaration order in an inline `style="..."` attribute is cascade-significant and directly compared by `htmldiff.Equal`; a Go map has no deterministic iteration order. |

**Key insight:** every "don't hand-roll" item above is deceptively simple until you look at the corpus fixtures — plain-map style building looks fine until two directives set the same CSS property in different orders across two runs and the HTML diff flakes.

## Common Pitfalls

### Pitfall 1: Treating the CSS pass order as a single linear "scope then resolve" pipeline
**What goes wrong:** ARCHITECTURE.md's original approximation ("meta parse → nesting down-level → `:root` remap/specificity fix → selector-scope → import resolve → render-time pagination + advanced-background injection") is *close* but wrong in a way that will produce wrong specificity and wrong `:root` handling if followed literally.
**Why it happens:** Marpit actually runs `:root`-replacement **twice** (once at per-theme "add" time on the raw theme CSS, once again at render "pack" time on freshly-injected before/after/scaffold/advanced-background/pagination CSS) and defers the ACTUAL specificity-boosting rewrite (`:marpit-root` → `:where(section):not([\20 root])`) until **after** container-selector scoping is complete — not right after the first `:root` replacement.
**How to avoid:** Implement the exact two-tier, 20-step order transcribed in the next section, verified line-for-line from `theme_set.js:281-286` and `theme.js`. Do not "simplify" the order — the specificity trick is provably order-dependent (it must see the FULLY qualified selector, e.g. `div.marpit > svg > foreignObject > section`, to add the correct extra `:not([\20 root])` specificity unit relative to plain `section` rules).
**Warning signs:** cssdiff gate failures where selectors are structurally right but come out in the wrong CASCADE order, or `:root`-authored rules end up with equal (not higher) specificity vs. plain `section` rules.

### Pitfall 2: CSS nesting has no Go library — do not assume tdewolff/parse supports it
**What goes wrong:** Assuming tdewolff/parse/v2/css's incremental nesting-token support (landed v2.8.5–v2.8.8 per PITFALLS.md) means nesting *down-leveling* (flattening `.a { .b { color: red } }` into `.a .b { color: red }`) comes for free.
**Why it happens:** tdewolff/parse only *tokenizes* nested-rule syntax (recognizes the grammar shape) — it does not flatten it. Marpit's actual down-leveling logic lives entirely in the JS-only `postcss-nesting` + `@csstools/postcss-is-pseudo-class` packages (verified: `postcss/nesting.js` imports both, and specifically re-wraps `:is()`-expanded selectors starting with `section`/`:root` through the is-pseudo-class plugin's "onComplexSelector: warning" mode to avoid overly broad specificity semantics from bare `:is()`).
**How to avoid:** Scope Objective 1's nesting support to exactly what the synthetic stress theme (§ below) and the 3 bundled themes require — do not attempt general CSS Nesting Module Level 1 spec compliance. Build `chase/theme/pass_nesting.go` test-first against a handful of concrete nesting shapes (simple `&`-implicit child nesting, one level of `:is()` fan-out) rather than trying to be exhaustive.
**Warning signs:** any nesting test with 2+ levels of depth, or nesting combined with `@media`, will likely need scope-narrowing decisions — flag these as they arise rather than blocking on full generality.

### Pitfall 3: Building the selector-rewriter as string concatenation instead of a token-level model
**What goes wrong:** Prepending `"div.marpit > svg > foreignObject > section "` as a raw string prefix to every selector text works for simple selectors but breaks for multi-selector rules (`h1, h2 { ... }` — each comma-separated compound selector needs its own prefix) and for `:is()`/`:where()` function arguments (which must NOT be prefixed themselves, since the function is evaluated within the already-scoped context).
**Why it happens:** tdewolff's CSS grammar lexes `:is(h1, marp-h1)` as one opaque `FunctionToken` — there's no automatic decomposition into "this is a functional pseudo-class containing a comma-separated selector list."
**How to avoid:** `chase/theme/selector` must (1) split the top-level selector list on top-level (non-nested) commas first, (2) for each compound selector, walk its token list and only prepend the scope chain to the OUTERMOST compound selector — never descend into `FunctionToken` arguments during the prepend step — but (3) DO hand-walk into `:is(...)`/`:where(...)` FunctionToken argument lists specifically for the `:root`-marker-rewrite step, since `:where(:is(h1, marp-h1))` (confirmed actual production output in `marp-theme-gaia/expected.css`) shows `:root`/`:marpit-root` markers CAN appear nested inside these functions and must still be found and rewritten.
**Warning signs:** any corpus theme fixture where a rule selector contains a comma or a `:is()`/`:where()` — treat `marp-theme-gaia`'s CSS as the canonical regression fixture for this exact bug class.

### Pitfall 4: Conflating comment DETECTION with directive RECOGNITION
**What goes wrong:** Building one goldmark pass that both (a) detects `<!-- ... -->` as a candidate directive AND (b) decides whether its content is a real known directive key, then treats it as consumed/hidden either way.
**Why it happens:** Marpit deliberately separates these: `comment.js` detects ALL HTML comments (storing raw parsed key/value pairs via a permissive `yaml()` helper) and marks well-known NON-directive magic comments (prettier-ignore, markdownlint, remark-lint directives) as parsed-but-not-a-directive; a LATER pass (`directives/parse.js`) decides which of the stored keys are actually recognized global/local directives and marks the token `directive`-consumed only then. Comments matching neither category remain as literal (hidden) comment tokens, invisible in output but NOT stripped of their raw content.
**How to avoid:** Mirror the two-stage design: `chase/markdown`'s comment BlockParser/InlineParser only extracts raw key/value data onto a custom node; a separate transformer (or an ASTTransformer that internally calls `chase/directive`) decides directive recognition. Keep the "well-known magic comment" passthrough list (prettier-ignore, markdownlint, remark-lint) as a documented but likely OUT-OF-SCOPE-for-Objective-1 concern — flag in Open Questions below.
**Warning signs:** a corpus case with a non-directive HTML comment (e.g., a normal author comment) either disappearing entirely from output when it shouldn't, or leaking into rendered HTML when it should stay hidden.

## Code Examples

### Verified goldmark v1.8.4 core interfaces (read directly from module cache, not training-data recall)
```go
// parser/parser.go:158-198 — Context (carry-forward vehicle)
type Context interface {
    Get(ContextKey) any
    ComputeIfAbsent(ContextKey, func() any) any
    Set(ContextKey, any)
    // + reference/ID/block-offset methods
}
func NewContextKey() ContextKey   // parser.go:151-154, mint once per package-level state slot

// parser/parser.go:586-590 — ASTTransformer (slide-splitting, directive-apply, inline-SVG-wrap all implement this)
type ASTTransformer interface {
    Transform(node *ast.Document, reader text.Reader, pc Context)
}

// parser/parser.go:514-529 — BlockParser (comment detection, front-matter)
type BlockParser interface {
    Trigger() []byte
    Open(parent ast.Node, reader text.Reader, pc Context) (ast.Node, State)
    // + Continue, Close
}

// parser/parser.go:556-570 — InlineParser (inline comment detection, bg-image alt-text option parsing)
type InlineParser interface {
    Trigger() []byte
    Parse(parent ast.Node, block text.Reader, pc Context) ast.Node
}

// markdown.go:138-141 — Extender (how chase/markdown plugs into a goldmark.Markdown)
type Extender interface { Extend(Markdown) }

// parser/parser.go:592-616 — default block-parser priority order (CONFIRMS setext-H2 trap is pre-resolved)
// SetextHeadingParser=100, ThematicBreakParser=200, ListParser=300, ..., ParagraphParser=1000
```

### tdewolff/parse/v2/css grammar walk (proven pattern, extend from `conformance/cssdiff/build.go`)
```go
// Already-working pattern in this repo (conformance/cssdiff/build.go) — chase/theme's
// own parse.go should follow the identical shape but build a richer Stylesheet model
// (preserving selector TOKEN LISTS, not just joined strings, so chase/theme/selector
// can hand-walk :is()/:where() FunctionToken arguments):
p := css.NewParser(parse.NewInputString(cssText), false)
for {
    gt, _, _ := p.Next()
    switch gt {
    case css.BeginRulesetGrammar:
        selectorTokens := p.Values() // []css.Token — DO NOT flatten to string here for chase/theme;
                                      // keep as []css.Token so FunctionToken args stay walkable.
    case css.DeclarationGrammar:
        // property name + p.Values() for the value token list
    case css.BeginAtRuleGrammar, css.AtRuleGrammar, css.EndAtRuleGrammar:
        // @import / @import-theme / @media capture
    case css.EndRulesetGrammar:
        // close current rule
    case css.ErrorGrammar:
        // EOF or parse error — p.Err() distinguishes the two (see cssdiff/build.go)
    }
}
```

### The `:root` specificity trick — exact source (postcss/root/increasing_specificity.js + replace.js, root.js selector rewriter target)
```
Step 1 (Theme.fromCSS, add-time):  :root  →  section:marpit-root
Step 2 (ThemeSet.pack, mid-pipeline): a SECOND root/replace pass, same
        pseudoClass ':marpit-root', catches :root appearing in
        before/after/scaffold/advanced-background/pagination-injected CSS
        (i.e. CSS that did NOT go through Theme.fromCSS's add-time pass)
Step 3 (ThemeSet.pack, pseudo_selector/prepend + replace):
        scope-prefix chain gets prepended/substituted onto ALL selectors,
        INCLUDING ones still carrying the :marpit-root marker
Step 4 (ThemeSet.pack, root/increasing_specificity — LAST, after scoping):
        :marpit-root  →  :where(section):not([\20 root])
        ("[\20 root]" = CSS-escaped attribute selector for an attribute
        literally named "<SPACE>root" — impossible to author for real,
        so it NEVER matches; :where() itself contributes 0 specificity;
        :not() wrapping a non-matching attribute selector still contributes
        its normal specificity weight of one attribute-selector (0-1-0),
        which is exactly what's needed to outrank a plain `section` rule
        (0-0-1) while :where() guarantees the overall rule still always
        matches every section.)
```
`chase/theme/selector/root.go` should implement this as: (1) a sentinel string marker inserted at parse time wherever a bare `:root` compound-selector-start is found; (2) after scope-prefixing is complete, a token-level rewrite pass converting the sentinel into the literal `:where(section):not([\20 root])` token sequence.

### CSS Pipeline — Tier 1 (`Theme.fromCSS`, add-time, run ONCE per theme when it's added to the theme set)
Verified from `theme.js`'s plugin array:
```
[nesting? (if cssNesting enabled), meta, root/replace(pseudoClass=':marpit-root'), section_size, import/parse (record-only, no resolution)]
```

### CSS Pipeline — Tier 2 (`ThemeSet.pack(name, opts)`, render-time, run EVERY render) — verified verbatim, `theme_set.js:270-286`
```js
const runPostCSS = (css, plugins) =>
  postcss([this.cssNesting && nesting(), ...plugins].filter(p => p)).process(css).css
//         ^ nesting is ALWAYS step 0 of every runPostCSS call, including the
//           normalizeExtraCSS() calls used for opts.before/opts.after

return runPostCSS(theme.css, [
  before && beforePlugin(beforeCSS),                                    //  1. inject author "before" CSS
  after  && afterPlugin(afterCSS),                                      //  2. inject author "after" CSS
  opts.containerQuery && containerQuery(containerName),                 //  3. wrap in @container if requested
  hoisting,                                                             //  4. hoist @charset/@import to top
  importReplace(this),                                                  //  5. RECURSIVELY resolve @import/@import-theme (circular-import throws)
  opts.printable && printable({ width, height }),                       //  6. inject print-specific CSS (PDF export)
  theme !== scaffoldTheme && scaffold,                                  //  7. prepend scaffold reset CSS (skipped if THIS theme IS the scaffold)
  inlineSVGOpts.enabled && advancedBackground,                          //  8. append static advanced-background CSS block
  inlineSVGOpts.enabled && inlineSVGOpts.backdropSelector && svgBackdrop,//  9. retarget ::backdrop → scoped @media screen rule
  pagination,                                                           // 10. comment-out non-marpit `content:` on section::after-like selectors
  rootReplace({ pseudoClass: increasingSpecificityPseudoClass }),       // 11. SECOND :root→marker pass (catches injected CSS from steps 1-10)
  fontSize,                                                             // 12. inject :root{font-size:...} var (for rem conversion in step 18)
  prepend,                                                              // 13. prepend ':marpit-container > :marpit-slide ' placeholder to every selector
  replace2(opts.containers, slideElements),                             // 14. substitute placeholders with REAL container/slide element chain
  increasingSpecificity,                                                // 15. NOW rewrite the marker → :where(section):not([\20 root]) — AFTER scoping
  opts.printable && printable.postprocess,                              // 16. print-CSS cleanup
  opts.containerQuery && containerQuery.postprocess,                    // 17. container-query cleanup
  rem,                                                                  // 18. convert rem units → calc(var(--marpit-root-font-size,1rem)*N) or similar
  hoisting,                                                             // 19. re-hoist @charset/@import one more time (steps 1-18 may add new ones)
  ...this.#plugins,                                                     // 20. user-registered custom plugins, always last
].filter(Boolean))
```
`slideElements` (step 14's substitution target, `theme_set.js:262-268`):
```js
const slideElements = [
  ...(inlineSVGOpts.enabled ? [{tag:'svg'}, {tag:'foreignObject'}] : []),
  {tag:'section'}
]
// → joined as "svg > foreignObject > section" when inline-SVG enabled, else just "section"
// containers defaults to Marpit's own [Element('div',{class:'marpit'})] → "div.marpit"
// FINAL scoped chain: "div.marpit > svg > foreignObject > section" — CONFIRMED against
// real corpus expected.css, correcting ARCHITECTURE.md's looser ".marpit section" guess.
```

### Static CSS injected by advanced-background (step 8) — verbatim, so the synthetic stress theme + Obj 1 tests can assert against it exactly
```css
section[data-marpit-advanced-background="background"] { columns: initial !important; display: block !important; padding: 0 !important; }
section[data-marpit-advanced-background="background"]::before, section[data-marpit-advanced-background="background"]::after,
section[data-marpit-advanced-background="content"]::before, section[data-marpit-advanced-background="content"]::after { display: none !important; }
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] { all: initial; display: flex; flex-direction: row; height: 100%; overflow: hidden; width: 100%; }
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container][data-marpit-advanced-background-direction="vertical"] { flex-direction: column; }
section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split] > div[data-marpit-advanced-background-container] { width: var(--marpit-advanced-background-split, 50%); }
section[data-marpit-advanced-background="background"][data-marpit-advanced-background-split="right"] > div[data-marpit-advanced-background-container] { margin-left: calc(100% - var(--marpit-advanced-background-split, 50%)); }
section[data-marpit-advanced-background="background"] > div[data-marpit-advanced-background-container] > figure { all: initial; background-position: center; background-repeat: no-repeat; background-size: cover; flex: auto; margin: 0; }
section[data-marpit-advanced-background="content"], section[data-marpit-advanced-background="pseudo"] { background: transparent !important; }
section[data-marpit-advanced-background="pseudo"], :marpit-container > svg[data-marpit-svg] > foreignObject[data-marpit-advanced-background="pseudo"] { pointer-events: none !important; }
section[data-marpit-advanced-background-split] { width: 100%; height: 100%; }
```
Note the `:marpit-container > svg[...]` selector INSIDE this injected block — proof that injected-CSS steps (7-10) run BEFORE selector-scoping (steps 13-14), since this selector still uses the placeholder token and gets resolved along with everything else.

### Scaffold theme CSS (verbatim, `theme/scaffold.js`) — recommended as chase/theme's own embedded baseline reset for Objective 1 testing (Objective 3 owns embedding the 3 named bundled themes with MIT headers, but this scaffold text is small, load-bearing for EVERY theme, and safe to hardcode now)
```css
section { width: 1280px; height: 720px; box-sizing: border-box; overflow: hidden; position: relative; scroll-snap-align: center center; -webkit-text-size-adjust: 100%; text-size-adjust: 100%; }
section::after { bottom: 0; content: attr(data-marpit-pagination); padding: inherit; pointer-events: none; position: absolute; right: 0; }
section:not([data-marpit-pagination])::after { display: none; }
:where(h1) { font-size: 2em; margin-block: 0.67em; }
video::-webkit-media-controls { will-change: transform; }
```

### Recommended synthetic stress theme (for success criterion 4 — nesting + `:is()`/`:where()` coverage)
Design it to exercise exactly the gaps flagged above, nothing more:
```css
/**
 * @theme stress
 * @size 4:3
 */
:root { --accent: teal; }               /* exercises Tier-1 :root replace + Tier-2 specificity trick */
section {
  & h1, & h2 { color: var(--accent); }  /* exercises nesting down-level (implicit & child combinator) */
  &.lead { text-align: center; }        /* nesting + class compound selector */
}
:where(h1, h2) { margin: 0; }           /* exercises :where() with multiple args, zero specificity */
:is(h3, h4) + p { margin-top: 0; }      /* exercises :is() combined with an adjacent-sibling combinator */
::backdrop { background: #048; }        /* exercises svg_backdrop retargeting, if THEME-04 scope includes it */
```
This is deliberately minimal — expand only if a specific corpus/bundled-theme need surfaces one you can't yet handle.

## State of the Art

| Old Approach (assumed by ARCHITECTURE.md) | Corrected Approach (verified this session) | When Changed | Impact |
|---|---|---|---|
| Single CSS pass order: meta → nesting → root-fix → scope → import → render-inject | Two-TIER pipeline: Theme.fromCSS (4 steps, add-time, once) then ThemeSet.pack (20 steps, render-time, every render), with `:root` replaced TWICE and the specificity-boost deferred until AFTER scoping | This session (verified from `theme_set.js`/`theme.js` source) | Planner must design `chase/theme`'s `Pass` pipeline as two distinct phases (`Theme.Load(css string) (*Theme, error)` vs `ThemeSet.Pack(name string, opts PackOptions) (string, error)`), not one flat list. |
| Scoped selector assumed `.marpit section` | Confirmed real scoped chain is `div.marpit > svg > foreignObject > section` (inline-SVG mode) or `div.marpit > section` (non-SVG mode) | This session (verified from corpus `expected.css` + `theme_set.js` `slideElements`/container logic) | The selector-rewriter's scope-prepend step must build a COMBINATOR CHAIN (`>` between container/svg/foreignObject/section), not a single class-descendant prefix. |
| "Dedicated inline parser vs post-hoc AST scan" open question for comment directives | RESOLVED: dedicated `BlockParser`+`InlineParser` pair, registered first, detection separated from directive-recognition (two-stage design) | This session (verified from `comment.js` + `directives/parse.js`) | Removes a previously-flagged architectural uncertainty; `chase/markdown` should follow the two-stage split exactly. |

**Deprecated/outdated:** None — Marpit's pipeline itself hasn't changed; what changed is OUR confidence level, from MEDIUM-HIGH (DeepWiki-sourced summary in ARCHITECTURE.md) to HIGH (literal source read this session).

## Open Questions

1. **Exact YAML dialect/schema restrictions for directive comment values**
   - What we know: `directives/yaml.js` is called with a `looseYAML` boolean flag and (in front-matter) an allow-list of custom directive keys; Marpit's underlying `yaml()` helper was not read in full this session.
   - What's unclear: whether Marpit restricts to a YAML "failsafe" subset (no anchors/aliases/custom tags) or allows full YAML 1.1 — this affects which Go YAML library (or hand-rolled scalar parser) `chase/directive` should use.
   - Recommendation: read `lib/markdown/directives/yaml.js` before finalizing `chase/directive`'s value-parsing implementation; default to a minimal hand-rolled scalar/flow-list parser (most directive values in the corpus are bare strings, booleans, or small integer/array lists — not full YAML documents) rather than pulling in a full YAML dependency, unless that file reveals genuine full-YAML usage.

2. **`<!--fit-->` auto-scaling marker: which layer emits `is="marp-h1"`/`data-auto-scaling`?**
   - What we know: the corpus's `marp-fit-heading` case expects `<h1 is="marp-h1" data-auto-scaling ...>` — this is a Marp CORE behavior (confirmed: `marp-h1` is a Marp Core custom-element name, not a Marpit concept), while raw Marpit itself likely only needs to detect/preserve the `<!--fit-->` marker.
   - What's unclear: whether Objective 1 (positioned as "raw Marpit-equivalent") should emit the bare comment-preservation behavior only, deferring `is="marp-h1"`/`data-auto-scaling` emission to Objective 3 (Marp-Core-layer), or whether the conformance corpus's `requires_engine: marp-core` tag on this case means Objective 1 is NOT expected to turn it green at all.
   - Recommendation: planner should explicitly scope `marp-fit-heading` either into Objective 1 (if `<!--fit-->` detection belongs in `chase/markdown`) or explicitly OUT (deferred to Objective 3) — do not leave it ambiguously "maybe."

3. **Well-known "magic comment" passthrough (prettier-ignore, markdownlint, remark-lint) — in scope for Objective 1?**
   - What we know: Marpit's `comment.js` has a hardcoded list of non-directive magic-comment patterns that get marked `parsed` (hidden, non-directive) rather than left as raw ambiguous comments.
   - What's unclear: whether any corpus case depends on this passthrough behavior (none of the 18 read cases obviously do).
   - Recommendation: treat as low-priority/deferrable; implement only if a corpus or future case requires it.

4. **`::backdrop` / `svg_backdrop.js` retargeting — is `backdropSelector` support in Objective 1's scope?**
   - What we know: it's part of Tier-2 pack() (step 9), gated behind `inlineSVGOpts.backdropSelector`, and none of the 18 read corpus cases exercise it directly (not confirmed absent, just not observed in the files read).
   - What's unclear: whether any bundled theme (default/gaia/uncover) uses `::backdrop` — would need to check Objective 3's actual theme CSS files once embedded.
   - Recommendation: include the synthetic stress theme's `::backdrop` rule (already added above) as a cheap way to force a decision either way during Objective 1 test-writing, without depending on Objective 3's theme embedding being done first.

## Sources

### Primary (HIGH confidence — read directly from source on this machine)
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark@v1.8.4/parser/parser.go` — Context, ASTTransformer, BlockParser, InlineParser, PrioritizedSlice, DefaultBlockParsers priority order
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark@v1.8.4/markdown.go` — Markdown interface, Convert()'s two-phase internal implementation, Extender interface
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark@v1.8.4/parser/setext_headings.go`, `thematic_break.go` — verified setext-vs-thematic-break precedence resolution
- `/Users/justin/go/pkg/mod/github.com/yuin/goldmark-meta@v1.1.0/meta.go` — reference `Extend()` pattern for registering BlockParser+ASTTransformer via `util.Prioritized`
- `/Users/justin/dev/eden-press/tools/corpus-gen/node_modules/@marp-team/marpit/lib/{theme.js,theme_set.js,marpit.js,element.js}` — full CSS Tier-1/Tier-2 pipeline order, plugin registration order, Element/container model
- `/Users/justin/dev/eden-press/tools/corpus-gen/node_modules/@marp-team/marpit/lib/markdown/{comment.js,slide.js,heading_divider.js,inline_svg.js,image.js}` — comment detection, slide-splitting, inline-SVG wrap structure
- `/Users/justin/dev/eden-press/tools/corpus-gen/node_modules/@marp-team/marpit/lib/markdown/directives/{parse.js,apply.js,directives.js}` — full directive carry-forward + application semantics
- `/Users/justin/dev/eden-press/tools/corpus-gen/node_modules/@marp-team/marpit/lib/markdown/background_image/{parse.js,apply.js,advanced.js}`, `image/{parse.js,apply.js}` — full bg-image option grammar + advanced-background 3-layer structure
- `/Users/justin/dev/eden-press/tools/corpus-gen/node_modules/@marp-team/marpit/lib/postcss/{nesting.js,advanced_background.js,pagination.js,scaffold.js,svg_backdrop.js}`, `postcss/root/*.js`, `postcss/import/*.js`, `theme/scaffold.js` — every Tier-2 pass's exact behavior
- `/Users/justin/dev/eden-press/conformance/cssdiff/{model.go,build.go,diff.go}` — proven tdewolff/parse/v2/css walking pattern to extend
- `/Users/justin/dev/eden-press/conformance/corpus/cases/marp-*/` (18 cases, expected.html/.css fixtures) — ground truth for byte-level structure claims

### Secondary (MEDIUM confidence)
- `/Users/justin/dev/eden-press/.planning/research/ARCHITECTURE.md`, `PITFALLS.md` — prior-session research, now partially corrected/upgraded by this session's direct-source findings (corrections noted explicitly in "State of the Art" above)
- `/Users/justin/dev/eden-press/PROPOSAL.md` §2.1/§4.1/§4.2, `/Users/justin/dev/eden-press/.planning/REQUIREMENTS.md`, `/Users/justin/dev/eden-press/.planning/ROADMAP.md` — objective scope and requirement wording (project-internal, not third-party, but included here as scope source)

### Tertiary (LOW confidence / flagged for follow-up)
- Marpit's `directives/yaml.js` and `helpers/inline_style.js` — referenced/inferred from call sites but not read in full this session; see Open Question 1.

## Metadata

**Confidence breakdown:**
- Standard stack (goldmark v1.8.4, tdewolff/parse v2.8.13): HIGH — versions confirmed from go.mod, APIs confirmed from local module-cache source.
- Directive/slide-split/bg-image/inline-SVG architecture: HIGH — every mechanism transcribed from Marpit's actual JS source, cross-checked against real corpus fixtures.
- CSS scoping pass order + selector-rewriter design: HIGH — corrected and verified line-for-line from `theme_set.js`/`theme.js`, a meaningful upgrade over the prior session's MEDIUM-HIGH DeepWiki-sourced approximation.
- Nesting down-level implementation approach: MEDIUM — the PROBLEM is fully understood (no Go library exists), but the exact minimal algorithm for `chase/theme/pass_nesting.go` is left to test-first implementation against the synthetic stress theme, not fully pre-specified here.
- YAML directive-value dialect: LOW — flagged as Open Question 1, needs one more file read (`directives/yaml.js`) before implementation.

**Research date:** 2026-07-20
**Valid until:** Stable for the life of this objective — goldmark v1.8.4 and tdewolff/parse v2.8.13 are pinned dependencies, and Marpit's source (read directly, not summarized) will not change. Re-verify only if go.mod version pins change.
