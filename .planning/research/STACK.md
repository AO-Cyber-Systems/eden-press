# Stack Research — Pressure Test

**Domain:** Go (+ Dart/Flutter) Marp-compatible document-generation framework, zero-JS backend
**Researched:** 2026-07-20
**Confidence:** HIGH overall (every library below checked against pkg.go.dev, official GitHub/Codeberg repos, and pub.dev on the research date). A few items are MEDIUM/LOW and flagged inline — mostly maturity/activity judgment calls on single-maintainer libraries, not factual gaps.

**Scope note:** The stack in `PROPOSAL.md` / `PROJECT.md` is already decided. This document does **not** re-pick a stack — it pressure-tests each already-chosen library (does it exist, current version, can it actually do the job, what breaks, what's the fallback) and flags three corrections the roadmap should account for.

---

## Executive Verdict

| # | Item | Verdict |
|---|------|---------|
| 1 | goldmark + extension mechanism | **HOLDS.** Exactly the extension surface (block/inline parsers, ASTTransformer, NodeRenderer) needed. v1.8.4 stable; v2 is a **live beta** (see §1) — stay on v1 for v1 ship. |
| 2 | tdewolff/parse css | **HOLDS**, but confirm it is a token/grammar stream, not an AST — the selector-rewriter is 100% custom code Eden Press must write (already implied by PROPOSAL §5.1, now confirmed as necessary, not optional). |
| 3 | chroma | **HOLDS.** v2.27.0 stable; v3 is alpha, stay on v2. Line-highlight and class-based CSS both confirmed as first-class features. |
| 4 | latex2mathml | **HOLDS WITH CAVEATS.** MIT, single-author, dormant since ~Dec 2023, no tagged release — fork/vendor is mandatory, exactly as the proposal's own spike concluded. |
| 5 | go-latex/latex | **HOLDS AT A CORRECTED ADDRESS.** `github.com/go-latex/latex` is archived; the maintained module is now `codeberg.org/go-latex/latex`. |
| 6 | chromedp | **HOLDS.** PrintToPDF requires dropping to `cdproto/page` inside an `ActionFunc` (normal chromedp usage, not a gap). No pure-Go alternative exists or should be sought — Chrome is an accepted external. |
| 7 | cobra vs urfave/cli | **cobra HOLDS.** Best fit for a Marp-CLI-shaped multi-command tool needing shell completion + man pages. |
| 8 | fsnotify | **HOLDS**, with real, documented gotchas (no recursion, atomic-save renames) that must be designed around, not discovered later. |
| 9 | koanf vs viper | **koanf HOLDS.** Confirmed lighter, modular, case-preserving — a better fit for a single static binary than viper. |
| 10 | unioffice / native PPTX | **Hand-rolled OOXML CONFIRMED as the only viable path.** unioffice is fully commercial now (not just AGPL-vs-commercial as the proposal assumed) — even its "free" tier needs a live license-key check-in, which conflicts with Eden Press's own no-implicit-network security goal. |
| 11 | bluemonday | **HOLDS.** Full custom-policy API confirmed sufficient to mirror Marp's `xss` allow-list rule-by-rule. |
| 12 | go:embed | **HOLDS** (stdlib, no risk). |
| 13 | Dart/Flutter FFI | **HOLDS, WITH A CORRECTION:** drop `gomobile bind` from the toolchain — use plain `go build -buildmode=c-shared`/`c-archive` (cgo) directly, which is what `dart:ffi` actually needs. `gomobile bind` is a different, heavier, less-maintained tool. |
| 14 | Go→WASM | **HOLDS**, with confirmed current mechanics (`GOOS=js GOARCH=wasm`, `wasm_exec.js` now lives in `lib/wasm/` not `misc/wasm/`) and concrete size numbers. |
| 15 | flutter_math_fork | **HOLDS.** Real KaTeX-parser Dart port, Apache-2.0, functional but single-maintainer (same "unmaintained-original fork" pattern as latex2mathml). |
| 16 | pub `highlight` | **REPLACE.** Named package is 5 years stale. Use its maintained successor **`highlighting` + `flutter_highlighting`** instead. |

---

## 1. goldmark (Markdown base + extensions)

**Version:** `github.com/yuin/goldmark` **v1.8.4** (MIT, released 2026-07-12) — the actively maintained, GA line.

**v2 status (important, time-sensitive finding):** `github.com/yuin/goldmark/v2` exists as a **real, active beta** — `v2.0.0-beta.1` through `beta.5`, all tagged in the two weeks immediately before this research (2026-07-02 through 2026-07-12). This is not a stale artifact; the author is actively cutting betas right now. v2's redesign is notable for Eden Press specifically because it targets exactly what PROPOSAL §13.3 wants long-term (structured, semantic AST; position tracking; renderer generic over non-HTML output targets; a builder API for programmatic AST construction). v2 is **not** recommended for the v1 build: third-party extensions haven't ported yet, and the goldmark team's own docs call the v2 API "almost stable... some parts may change." **Recommendation:** ship v1.8.4 now; track v2 GA as a candidate migration once Eden Press's own structured-AST/output-as-data work (PROPOSAL §13.3) matures — the two roadmaps may converge well.

**Extension mechanism (confirmed exactly matches the pressure-test ask):**
- `parser.WithBlockParsers` / `parser.WithInlineParsers` — register custom `parser.BlockParser` / `parser.InlineParser` (needed for slide-splitting on `---`, background-image `![bg]` syntax, directive comment scanning).
- `parser.WithASTTransformers` (`parser.ASTTransformer`) — whole-document passes (needed for directive carry-forward state, inline-SVG wrapping).
- `renderer.NodeRenderer` + `renderer.WithNodeRenderers(util.Prioritized(r, priority))` — custom render functions per AST node kind, with priority-based override of built-in renderers.
- Extensions bundle into the `goldmark.Extender` interface (`Extend(Markdown)`), the standard way to package a reusable plugin (mirrors Marpit's per-feature plugin shape 1:1).

**GFM (tables, strikethrough):** `extension.GFM` bundles `Table + Strikethrough + Linkify + TaskList`. Table support is native and complete.

**`<s>` vs `<del>` — confirmed directly against source (not just the spike):** `extension/strikethrough.go`'s `StrikethroughHTMLRenderer` emits **`<del>`** by default (GFM-spec-correct; Marp/markdown-it emits `<s>`). This is a one-file override:

```go
type sRenderer struct{}
func (r *sRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
    reg.Register(extast.KindStrikethrough, r.renderStrike)
}
func (r *sRenderer) renderStrike(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
    if entering { w.WriteString("<s>") } else { w.WriteString("</s>") }
    return ast.WalkContinue, nil
}
md := goldmark.New(goldmark.WithRendererOptions(
    renderer.WithNodeRenderers(util.Prioritized(&sRenderer{}, 100)),
))
```
This confirms the spike's finding (§12 of PROPOSAL) was correct and trivially fixable — no architectural risk here.

**Frontmatter:** there is **no** yuin-official "frontmatter" package for goldmark v1 — `goldmark-meta` (yuin-maintained, MIT) is the closest official option, and it is **YAML-only metadata**, last released **v1.1.0 in Feb 2022** (4+ years stale, but simple/stable code — low risk). Its own **v2.0.0 targets goldmark/v2 beta**, not v1 — **do not pin `goldmark-meta@v2.0.0` against `goldmark@v1.8.4`**; pin `goldmark-meta@v1.1.0`. Alternative, more actively maintained: **`github.com/abhinav/goldmark-frontmatter` v0.3.0** (BSD-3-Clause, released 2025-11-17), which supports YAML **and** TOML delimiters, custom formats, and typed struct decoding — worth using instead of goldmark-meta since Marpit-compatible YAML front-matter is a subset of what it already does and it's the more recently touched codebase.

**Verdict:** HOLDS. No replacement needed. Only decision: goldmark-meta v1.1.0 vs abhinav/goldmark-frontmatter v0.3.0 for the frontmatter block (lean toward abhinav's — more recent, more format-flexible).

---

## 2. tdewolff/parse/v2/css (theme-CSS engine)

**Version:** `github.com/tdewolff/parse/v2` **v2.8.13** (MIT, released 2026-05-27, 66 known importers).

**Confirmed shape — this is the load-bearing finding:** the `css` subpackage is a **CSS3 lexer + streaming grammar parser**, not a navigable DOM-like AST. `Lexer.Next()` yields raw tokens (`IdentToken`, `AtKeywordToken`, `HashToken`, etc.); `Parser.Next()` yields grammar events (`AtRuleGrammar`, `QualifiedRuleGrammar`, `DeclarationGrammar`, `BeginRulesetGrammar`/`EndRulesetGrammar`) with associated token slices via `p.Values()`. **Selectors arrive as raw tokens — there is no typed selector object** (no combinator/specificity/pseudo-class structure). At-rules including `@import`/`@media`/`@page` are reachable (general `AtRuleGrammar` handling; `@import` specifically is matched via the `AtKeywordToken` text, since it isn't in the package's perfect-hash table of "well-known" at-rule keywords). There is **no built-in serializer** — round-tripping means concatenating the original token bytes and manually re-inserting `{ } : ;`, which the package's own doc example demonstrates but does not automate.

**Practical consequence for Eden Press:** the "selector rewriter to scope theme CSS to `.marpit`/`section`" is **entirely custom code** Eden Press must write on top of tdewolff's grammar stream (a small selector-AST + rewrite-and-reserialize layer). This matches PROPOSAL §5.1's `marpit/theme/` package plan — the plan is correct, but it should be budgeted as its own real subsystem, not "wire up a library," since tdewolff gives you tokens/events, not a selector object you can mutate and print.

**Alternatives evaluated (per the pressure-test's explicit ask):**
- **`gorilla/css/scanner`** — a *pure tokenizer*, one level below tdewolff (no grammar/parse-tree at all — the docs say outright "it doesn't perform lexical analysis... intended to be used by a lexer or parser"). v1.0.1, last released 2023-10-18, low activity. **Worse fit** — you'd build the grammar layer yourself on top of an even more primitive base.
- **`vanng822/css`** — does expose a rule-list with a `Selector.Text()` and `Styles` per rule (closer to a real, if thin, AST) and is the CSS engine behind the actively-maintained `vanng822/go-premailer` (last published 2026-03-27). However the `vanng822/css` repo itself shows only **34 stars**, a **2021** last tagged release, and self-describes as "Basic css parser" — no confirmed nesting support, and `@import`/`@media` support exists but is untested for Marpit's more advanced needs (advanced backgrounds, pagination injection). **Legitimate fallback**, not the primary pick: tdewolff wins on maintenance activity and CSS-Syntax-Level-3 spec fidelity; only reach for vanng822/css if tdewolff's token-stream model proves too painful to build a rewriter on top of.

**Verdict:** HOLDS as the pick, specifically *because* it's the actively-maintained, spec-correct option — but go in expecting to write a real selector-AST/rewriter component, not a thin wrapper.

---

## 3. chroma (syntax highlighting)

**Version:** `github.com/alecthomas/chroma/v2` **v2.27.0** (2026-06-17). License: **MIT** for the code, plus **OFL-1.1** for one bundled font file (`formatters/svg/font_liberation_mono.go` — only relevant if the SVG formatter's embedded font is used; the HTML formatter path Eden Press needs doesn't touch it).

**v3 status:** `chroma/v3` exists only as **`v3.0.0-alpha.5`** (2026-07-08) — breaking API (custom `Iterator` type replaced by stdlib `iter.Seq[Token]`; `EOF` sentinel removed; needs Go 1.23+ for range-over-func). **Not production-ready.** Stay on v2.

**Confirmed capabilities (all present, first-class):**
- `formatters/html` `WithClasses(true)` → emits CSS classes instead of inline styles, using Pygments-style short class names via a `StandardTypes` map (`k`=Keyword, `n`=Name, `s`=LiteralString, `c`=Comment, `ln`=LineNumbers, `hl`=LineHighlight, `bg`=Background, `chroma`=PreWrapper). `ClassPrefix(prefix)` namespaces them. `formatter.WriteCSS(w, style)` emits the matching stylesheet for a given `chroma/styles` theme.
- `HighlightLines(ranges)` — directly implements Marp's `` ```lang {1-3} `` line-highlight syntax; highlighted lines get the `hl` class.
- Also available: `TabWidth`, `WithLineNumbers`, `WithLinkableLineNumbers`, `LineNumbersInTable`.

**Confirmed gotcha (real, not hypothetical — no off-the-shelf fix exists):** chroma's class taxonomy (`k`, `n`, `s`, `c`, ...) has **no relationship** to highlight.js's `.hljs-keyword`, `.hljs-string`, `.hljs-comment` naming that Marp's three themes' embedded CSS actually targets. There is no existing library or chroma feature that reconciles this — it is bespoke, one-time work. Two options, both bounded:
  (a) Write a custom `chroma.Style` + a formatter class-map so chroma emits `.hljs-*`-shaped classnames — preserves the three theme CSS files **verbatim** (matches PROPOSAL §2.2's "copy verbatim" plan for themes).
  (b) Regenerate the highlight-related CSS rules inside the three themes to target chroma's native class names instead.
  **Recommend (a)** — it's strictly less work and keeps the "reuse MIT theme assets unmodified" attribution story clean.

**Verdict:** HOLDS.

---

## 4–5. Math: latex2mathml + go-latex/latex

**`git.sr.ht/~mekyt/latex2mathml`:**
- **License: MIT**, confirmed directly on pkg.go.dev.
- **Version: pseudo-version only** (`v0.0.0-20231214134936-808832af73fc`) — **no tagged release ever**, last commit **2023-12-14** (≈2.5 years stale as of this research). Single dependency: a small `etree` XML-tree library. Single author, `imported by: 0` on the Go module proxy (i.e., no other known Go project depends on it) — this is as niche as libraries get.
- Construct coverage (confirmed via its exported constant/API inventory): fraction variants (`\frac/\dfrac/\tfrac/\cfrac/\binom/\over/\atop`), roots (`\sqrt/\root`), accents, `\text`/`\mathbb`/`\mathcal`/`\mathfrak`, matrices/environments (`\matrix/\pmatrix/\bmatrix/\cases/\align*`), spacing, sizing/delimiters, and standard functions (`\sin/\log/\lim/\int/\sum`). This matches — and independently corroborates — the breadth PROPOSAL's own §11 spike exercised. The spike's finding (8/20 constructs render wrong, but all 8 are converter bugs in already-declared features, not missing capabilities) is consistent with what this research found: a real but small, single-maintainer, dormant codebase that will need forking and patching, exactly as PROPOSAL already concluded. **Treat this as a vendored/forked dependency Eden Press owns and patches, not a `go get`-and-trust-upstream library.**
- No actively-maintained pure-Go alternative was found anywhere in this research. All alternatives surfaced are non-Go: the upstream Python original (`roniemartinez/latex2mathml`, MIT, actively maintained, Python 3.14-ready) and non-Go ports (Haskell `texmath`, Perl `LaTeXML`, JS `Mathoid`/`TeXZilla`, Java `SnuggleTeX`). If the Go fork ever becomes unsalvageable, the only realistic fallback within the zero-JS-*runtime* constraint is shelling to the Python original at **build time** (not runtime) as a last resort — this should stay a documented escape hatch, not a plan.

**`go-latex/latex` — corrected location (real finding, update the module path anywhere it's referenced):**
- `github.com/go-latex/latex` is **archived** (read-only since **2025-03-04**) with an explicit pointer to its new home.
- The maintained module is now **`codeberg.org/go-latex/latex`** — actively developed (commits as recent as **2026-05-18**, including a "bump latin-modern, fpdf, x/image and Go-1.25" dependency pass), **BSD-3-Clause**, 3 tagged releases. It renders LaTeX math to **PNG** in pure Go via `mtex.Render` (a `cmd/mtex-render` example exists); the README explicitly scopes it as *not* a full TeX typesetting system, positioning itself like MathJax/matplotlib's mathtext — exactly the "SVG/PNG fallback for math-dense decks" role PROPOSAL assigns it.
- **Action item for the roadmap:** any `go.mod`/import referencing `github.com/go-latex/latex` must be written as `codeberg.org/go-latex/latex` — the GitHub path will not receive further updates.

**Verdict:** latex2mathml HOLDS WITH CAVEATS (fork+patch required — budget it as engineering, not integration). go-latex/latex HOLDS at the corrected Codeberg address.

---

## 6. chromedp (headless Chrome / raster export)

**Version:** `github.com/chromedp/chromedp` **v0.16.0** (MIT, released 2026-07-14, 2,179+ importers).

**`Page.printToPDF`:** confirmed **not** a first-class `chromedp.Action` — the documented, supported pattern is to call `cdproto/page`'s `page.PrintToPDF()` inside a `chromedp.ActionFunc` (this is literally chromedp's own documented FAQ answer for any CDP method returning multiple values). This isn't a gap or a risk — it's just how chromedp is meant to be used for anything beyond its curated helper set. Write the PDF exporter expecting this shape, not a single high-level `chromedp.PrintToPDF(...)` call.

**Screenshots:** first-class actions exist for every shape Eden Press needs — `Screenshot`/`ScreenshotScale` (single element, for per-slide raster export), `ScreenshotNodes` (multiple nodes at once), `CaptureScreenshot` (viewport), `FullScreenshot` (full page). Arbitrary-region clipping needs `page.CaptureScreenshot().WithClip(...)` via `ActionFunc`, same pattern as PDF.

**Determinism:** `chromedp.DefaultExecAllocatorOptions` already sets several determinism-relevant Chrome flags out of the box (`disable-background-timer-throttling`, `force-color-profile=srgb`, `disable-features=...,Translate,...`, `metrics-recording-only`). There is **no dedicated chromedp helper** for disabling CSS animations or overriding timezone/locale — those require explicit `Emulation.setTimezoneOverride`/`setLocaleOverride` CDP calls (again via `ActionFunc`) or raw Chrome flags. Since "byte-reproducible output" is one of Eden Press's own stated differentiators (PROPOSAL §13.6), **budget explicit determinism engineering** (fixed viewport via `EmulateViewport`/`WindowSize`, explicit timezone/locale pinning, animation disabling) as real work in the export objective — chromedp gets you most of the way, not all of it.

**Chrome discovery/pinning (resolves PROPOSAL's open decision #3):** chromedp's `ExecAllocator` supports explicit `ExecPath(path)` or auto-detection; it has **no built-in "download a pinned Chrome build" feature**. The actual mechanism for reproducible, pinned Chrome downloads is Google's own **Chrome for Testing** distribution channel (the same one Selenium/Puppeteer rely on for version-locked, dedicated-to-automation Chrome+driver builds, official since Chrome 115). **Recommendation:** default to system Chrome via `ExecPath`/auto-detect/`--browser-path` (matches upstream Marp CLI behavior), and offer an optional command (e.g. `eden-press chrome install`) that fetches a pinned build from the Chrome for Testing JSON API for reproducible CI/archival exports.

**Verdict:** HOLDS. Chrome itself remains an accepted external binary dependency, same as upstream — there is no pure-Go alternative to seek here, and none should be sought.

---

## 7. CLI: cobra vs urfave/cli

**cobra** `github.com/spf13/cobra` **v1.10.2** (Apache-2.0, 2025-12-03, ~196k importers; powers kubectl, Hugo, gh). Confirmed: pflag-based POSIX flags, nested subcommands, persistent/local/cascading flags, flag-grouping (mutually-exclusive/required-together/one-required), full shell-completion generation (bash/zsh/fish/powershell), man-page generation, optional viper integration.

**urfave/cli** `github.com/urfave/cli/v3` **v3.10.1**. Confirmed: declarative struct-based command tree, zero non-stdlib dependencies, alias/prefix-match subcommands, shell completion, compound short flags.

**Verdict:** **cobra HOLDS.** It's the closer structural match to Marp CLI's own `yargs`-based surface (subcommands for convert/watch/serve/preview, `--pdf`/`--pptx`/`--images` flags, config-file loading), and its built-in shell-completion + man-page generation directly reduce CLI-polish work the roadmap would otherwise need to hand-build. **urfave/cli v3 is a legitimate documented fallback** if a zero-non-stdlib-dependency CLI later becomes a priority (e.g. for a minimal-footprint embedded mode) — not needed for v1.

---

## 8. fsnotify (watch mode)

**Version:** `github.com/fsnotify/fsnotify` **v1.10.1** (BSD-3-Clause, 2026-05-04, requires **Go 1.23+**).

**Confirmed real limitations to design around (not blockers, but silent-bug traps if ignored):**
- **Not recursive.** Every directory to watch must be `Add()`-ed individually; a recursive watcher has been an open, unresolved roadmap item for years (issue #18). Eden Press's `--watch` must walk the deck's directory tree itself.
- **Atomic-save breaks single-file watches.** Many editors (Vim, etc.) write to a temp file then rename over the original — the watch on the original inode is lost. **Correct pattern:** watch the *parent directory*, filter events by `Event.Name`, never watch the markdown file directly.
- **Cross-platform event-semantics differ:** Linux (inotify) delays a `REMOVE` event until all file descriptors close (a `CHMOD` fires first); Windows excludes `Chmod` entirely and doesn't clear watches on rename; a directory `Write` event means "contents changed" on kqueue/Windows but only "file content changed" on Linux inotify — portable watch code should filter directory-level `Write` events explicitly.
- Platform coverage is solid (inotify/Linux, kqueue/BSD+macOS, ReadDirectoryChangesW/Windows, FEN/illumos) but does **not** work over NFS/SMB/FUSE — worth a doc note if decks might live on a network mount.

**Verdict:** HOLDS — standard, correct choice, no viable pure-Go alternative with comparable platform coverage — but budget explicit directory-walk + atomic-save-safe filtering in the watch objective rather than assuming naive single-file watching works.

---

## 9. Config: koanf vs viper

**koanf** `github.com/knadh/koanf/v2` **v2.3.5** (MIT, 2026-05-30, 832 importers). Confirmed: `Provider`/`Parser` interfaces fully decoupled (file/env/HTTP/S3/Vault providers; json/yaml/toml/toml-v2/dotenv/hcl/hjson/huml/nestedtext parsers), each installed as a separate module so the core stays small; successive `Load()` calls merge recursively; **preserves key case** (critical — viper force-lowercases keys, which silently breaks case-sensitive JSON/YAML/TOML/HCL). koanf's own docs frame it explicitly as "a cleaner, lighter alternative to spf13/viper... far fewer dependencies," citing viper's forced lowercasing, dependency bloat, hardcoded source-ordering, and `Get()` leaking mutable slice/map references as concrete defects it avoids.

**Verdict:** **koanf HOLDS** over viper specifically because Eden Press ships a single static binary and wants exactly YAML+JSON+TOML for `.marprc.{yml,json,toml}` with minimal added dependency weight — koanf's pay-for-what-you-use module design is a direct match. No JS-config loading exists in either (consistent with PROPOSAL's own decision to drop `.marprc.js` support).

---

## 10. Native OOXML PPTX generation (the license trap, confirmed and worse than assumed)

**unioffice, checked directly against its current official README (`github.com/unidoc/unioffice`):** this is **not** the "AGPL-or-commercial" dual-license situation the proposal hedged on — the **current** official repository states plainly: *"This software package (unioffice) is a commercial product and requires a license code to operate,"* governed by the UniDoc EULA. There is **no AGPL-licensed, freely-operable current version.** Even the no-cost path requires signing up for a "Metered License API Key" at cloud.unidoc.io — which implies a live licensing check-in at runtime. **This directly conflicts with Eden Press's own stated differentiator of "no implicit network/asset fetch" for safe rendering of untrusted content** (PROPOSAL §13.5) — using unioffice would mean shipping a renderer that phones home by design. Old AGPL-era snapshots survive only as frozen, unmaintained community forks/mirrors (`unioffice-free`, assorted GitHub forks, the pre-commercial `gooxml` mirrors) — not something to build a product on.

**Surveyed the open-source landscape for a full from-scratch pure-Go PPTX generator:** none exists that is both (a) actively maintained and (b) license-free. The realistic options are: frozen forks of the pre-commercial codebase (stale since ~2020), or narrow template-substitution tools (e.g. `moipa-cn/pptx` — replaces text/images inside an *existing* .pptx, doesn't generate slides from scratch) — neither is a substitute for "build a deck's OOXML from an AST."

**Verdict:** **hand-rolling the OOXML zip is confirmed as the only viable path**, not just PROPOSAL's fallback option. This is real, self-contained engineering (matches PROPOSAL §5.1's `convert/pptx.go`): OOXML/PPTX is a documented ECMA-376 format — `[Content_Types].xml` + `ppt/presentation.xml` + `ppt/slides/slideN.xml` + `_rels` relationship files, all plain XML, zipped — buildable entirely with Go stdlib `archive/zip` + `encoding/xml`, no third-party dependency required at all. If a reference implementation is wanted while building the schema-derived Go structs, the pre-commercial `gooxml`/`unioffice` source (last MIT/AGPL-era snapshot, pre-~2020) can be read for schema shape — but should not be imported as a live dependency given the current licensing regime.

---

## 11. bluemonday (HTML sanitization)

**Version:** `github.com/microcosm-cc/bluemonday` **v1.0.27** (BSD-3-Clause, released 2024-07-04 — no release in ~2 years, but this reads as feature-complete stability for a narrowly-scoped allow-list sanitizer, not neglect; no CVEs surfaced in this research).

**Confirmed API is sufficient to mirror Marp's `xss`-package allow-list rule-for-rule:** blank-slate `NewPolicy()`, `AllowElements(...)`/`AllowElementsMatching(regex)`, `AllowAttrs(...).OnElements(...)` / `.Globally()`, URL-scheme control via `RequireParseableURLs(true)` + `AllowURLSchemes(...)`/`AllowURLSchemesMatching(regex)`/`AllowURLSchemeWithCustomPolicy(scheme, fn)`, plus a convenience `AllowStandardURLs()`. `style`/`script` are stripped by default unless explicitly (and inadvisably) unlocked via `AllowUnsafe(true)`.

**Verdict:** HOLDS. Building the Marp-parity policy is a config/test-writing task, not a capability risk.

---

## 12. go:embed

Stdlib since Go 1.16 — no version pressure-test needed. Recommend a project floor of **Go 1.25+** (current toolchain: **Go 1.26.5**, released 2026-07-07; the prior line **1.25.12** is also still security-supported), which comfortably satisfies every other library's minimum (fsnotify needs 1.23+; chroma v3-alpha's `iter.Seq` needs 1.23+ if ever adopted). This matches the Go 1.26 (arm64) environment PROPOSAL's own spikes already used.

---

## 13–14. Dart/Flutter binding stack

### 13a. dart:ffi — correction: drop `gomobile`, use plain cgo build modes

The pressure-test brief names "dart:ffi + gomobile (Android .so via c-shared, iOS .a via c-archive)" as one item — but these are actually **two different tools that shouldn't be combined**:

- **`golang.org/x/mobile/cmd/gomobile bind`**, checked directly: it does **not** output raw `.so`/`.a` files as its deliverable. It outputs an Android **AAR** (bundling Java/Kotlin JNI stub classes it generates, plus the compiled shared library) and an Apple **XCFramework** (with generated Objective-C headers) — a full cross-language *binding generator*, aimed at apps that want idiomatic Java/Swift APIs, not at a raw C-ABI FFI consumer. It's also confirmed **minimally maintained**: only pseudo-versioned (no tagged v1 ever, despite existing for over a decade), and the Go issue tracker shows a long, still-open pattern of module/toolchain-friction bugs (ambiguous-path failures, broken internal bind tests, Gradle/Go-modules integration issues) recurring across years.
- **The actually-correct, simpler, and better-maintained pattern for `dart:ffi`:** plain `go build -buildmode=c-shared -o libpress.so` (Android, cross-compiled per-ABI with `CGO_ENABLED=1` + the Android NDK toolchain) and `go build -buildmode=c-archive -o libpress.a` (iOS, linked into an Xcode framework target). Both are **core Go toolchain build modes** (not an `x/` sub-repo), actively maintained as part of the compiler itself, and are the pattern most production "Flutter + native-language core" integrations actually use. `dart:ffi`'s `DynamicLibrary.open()` (Android `.so`) / `DynamicLibrary.process()` (iOS, statically linked into the app binary) load the result directly — no generated JNI/ObjC stub layer required, since `dart:ffi` talks the C ABI natively.

**Verdict:** keep the proposal's actual outcome (native `.so` on Android / static `.a` on iOS, consumed via `dart:ffi`) but **remove `gomobile` from the toolchain description entirely** — it solves a problem (idiomatic per-platform language bindings) Eden Press doesn't have, while adding a less-maintained dependency.

### 13b/14. Go → WASM (Flutter Web)

Confirmed current mechanics: `GOOS=js GOARCH=wasm go build -o main.wasm`, loaded via `wasm_exec.js` — **note the file moved from `misc/wasm/` to `lib/wasm/` in recent Go versions** (`$(go env GOROOT)/lib/wasm/wasm_exec.js`); the Go-compiler major version and the `wasm_exec.js` version **must match exactly**. `GOOS=wasip1` (Go 1.21+) is a **different, non-browser target** (the WASI syscall API for standalone runtimes like Wazero/Wasmtime/Node's `wasi` module) and cannot be used for DOM/JS interop — `js/wasm` remains the only browser path, confirming PROPOSAL's plan as correct.

**Size (now with concrete numbers, not just an estimate):** a Go/Wasm binary floors around **~2MB+ uncompressed** (matches PROPOSAL's own estimate), but compresses to roughly **500–660KB with gzip or Brotli** — that's the number that actually matters for web-embed latency, and it should be stated as the target in any web-delivery objective. **TinyGo** can shrink Wasm output by **10–20x**, but is explicitly **not a drop-in Go compiler** — partial stdlib and reflection support is the single biggest source of breakage, and goldmark's extension/YAML-frontmatter dependency chain plausibly touches enough reflection-adjacent code that a TinyGo build is a real risk, not a free win. **Recommendation:** treat TinyGo as a stretch optimization to spike later (objective 6/7 territory), not a default assumption for the initial WASM build.

---

## 15. flutter_math_fork (native Dart math rendering)

**Version:** **0.7.4** (Apache-2.0, pub.dev shows "published 14 months ago" relative to this research date, i.e. ~2025-05). Confirmed: it is itself a fork created because the original `flutter_math` package went unmaintained (the same "fork of an abandoned original" shape recurring across this stack — see latex2mathml). Its TeX parser is a **direct Dart port of the actual KaTeX parser**, aiming for "maximum compatibility and fidelity" with KaTeX — exactly the fidelity bar needed to match the engine's MathML rendering path. Supports Android/iOS/Linux/macOS/web/Windows.

**Verdict:** HOLDS. Single-maintainer risk noted but no better-maintained pure-Dart, KaTeX-fidelity alternative surfaced in this research.

---

## 16. Dart syntax highlighting — REPLACE `highlight` with `highlighting`

The pub package literally named **`highlight`** (pd4d10) is confirmed **stale**: version **0.7.0**, published **~5 years ago**, flagged with an "unverified uploader" notice on pub.dev. **Its actual maintained successor is `highlighting`** (pure Dart, MIT, current **v0.9.0+11.8.0**) — described by its own docs as an automated, periodically-refreshed port of highlight.js (currently tracking highlight.js **11.8.0**, ~190+ languages), paired with a companion Flutter widget package **`flutter_highlighting`** (a drop-in `HighlightView` widget). There's an explicit migration guide from the old `highlight` package to `highlighting` v0.9, confirming this is the recognized successor, not a competing fork.

**Alternative also worth naming:** **`syntax_highlight`** — TextMate-grammar-based (VSCode-style fidelity), actively maintained, but a much smaller curated language list (~15 languages: css/dart/go/html/java/javascript/json/kotlin/python/rust/sql/swift/typescript/yaml/etc.) vs `highlighting`'s 190+. Use `syntax_highlight` only if VSCode-grade fidelity for a known-small language set matters more than breadth; `highlighting`'s coverage is the better match for a general document tool that needs to highlight whatever fenced-code-block language an author writes.

**Verdict:** **swap `highlight` → `highlighting` + `flutter_highlighting`** in the stack decision — this is a straightforward correction, not a tradeoff.

---

## Core Technologies (verified versions, at a glance)

| Technology | Version (verified 2026-07-20) | Purpose | Confidence |
|------------|-------------------------------|---------|------------|
| Go | 1.26.5 (patch), 1.25.12 also supported | Language/toolchain | HIGH |
| goldmark | v1.8.4 (MIT) | Markdown → AST | HIGH |
| goldmark-meta | v1.1.0 (MIT) — **not** v2.0.0 (that targets goldmark/v2 beta) | YAML front matter | HIGH |
| tdewolff/parse/v2 (css) | v2.8.13 (MIT) | CSS tokenizer/grammar stream | HIGH |
| chroma/v2 | v2.27.0 (MIT + OFL-1.1 font) | Syntax highlighting | HIGH |
| latex2mathml (mekyt) | pseudo-version, no tag (MIT) | LaTeX → MathML | HIGH (maturity), MEDIUM (long-term viability judgment) |
| go-latex/latex | `codeberg.org/go-latex/latex`, active (BSD-3) | LaTeX → PNG fallback | HIGH |
| chromedp | v0.16.0 (MIT) | Headless Chrome CDP driver | HIGH |
| cobra | v1.10.2 (Apache-2.0) | CLI framework | HIGH |
| fsnotify | v1.10.1 (BSD-3) | File watching | HIGH |
| koanf/v2 | v2.3.5 (MIT) | Config (YAML/JSON/TOML) | HIGH |
| bluemonday | v1.0.27 (BSD-3) | HTML sanitization | HIGH |
| flutter_math_fork | 0.7.4 (Apache-2.0) | Native Dart math rendering | HIGH |
| highlighting + flutter_highlighting | v0.9.0+11.8.0 (MIT) | Native Dart syntax highlighting | HIGH |
| Flutter / Dart SDK | Flutter 3.44.6 / Dart 3.12.2 (stable) | Client framework | HIGH |

## Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `abhinav/goldmark-frontmatter` | v0.3.0 (BSD-3) | YAML/TOML frontmatter, typed decode | Preferred over goldmark-meta if TOML or more active maintenance matters |
| `gopkg.in/yaml.v3` | current | Directive YAML parsing (Marpit-level, not frontmatter) | Parsing `<!-- key: value -->` and front-matter payload after goldmark-meta/frontmatter extracts the block |
| `nhooyr.io/websocket` or stdlib + `net/http` | current | Server live-reload | CLI `--server` mode |
| `archive/zip` + `encoding/xml` (stdlib) | Go 1.25+ | Hand-rolled OOXML PPTX | PPTX export, in place of unioffice |

## Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| Chrome for Testing (googlechromelabs.github.io/chrome-for-testing) | Pinned, version-locked Chrome downloads | Use for reproducible CI/archival export; default to system Chrome otherwise |
| `go build -buildmode=c-shared` / `c-archive` | Build Android `.so` / iOS `.a` for dart:ffi | Do **not** use `gomobile bind` for this |
| TinyGo | Optional Wasm-size optimization | Spike only — partial stdlib/reflect support is a real risk with goldmark's dependency chain |

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| goldmark v1.8.4 | goldmark v2 (beta) | Once v2 reaches GA and third-party extensions port — especially if Eden Press's structured-AST/JSON-AST output work matures in parallel |
| goldmark-meta v1.1.0 | abhinav/goldmark-frontmatter v0.3.0 | If TOML frontmatter or more recent upstream maintenance is wanted (recommended default, in fact) |
| tdewolff/parse/v2/css | vanng822/css | If a ready selector/rule object (vs. raw tokens) is worth trading for a much smaller, less-active dependency |
| cobra | urfave/cli v3 | If a zero-non-stdlib-dependency CLI becomes a priority over cobra's shell-completion/man-page tooling |
| koanf | viper | Only if tighter native cobra↔viper glue is wanted and the larger dependency footprint / forced key-lowercasing is acceptable |
| `go build -buildmode=c-shared/c-archive` | `gomobile bind` | Only if idiomatic generated Java/Kotlin or Swift/ObjC language bindings are wanted for direct native-app consumption (not the Eden Press use case) |
| `highlighting` + `flutter_highlighting` | `syntax_highlight` | If VSCode-grade TextMate fidelity for a small curated language set outweighs highlight.js-level language breadth |
| latex2mathml (fork+patch) | Shell to Python `roniemartinez/latex2mathml` at build time | Only as a last resort if the Go fork proves unsalvageable — reintroduces a non-Go dependency, so build-time only, never runtime |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `github.com/go-latex/latex` (GitHub) | Archived (read-only since 2025-03-04); no further updates will land there | `codeberg.org/go-latex/latex` (same lineage, actively maintained) |
| `golang.org/x/mobile/cmd/gomobile bind` | Generates unneeded Java/ObjC binding layers for a pure `dart:ffi` consumer; minimally maintained (no tagged release ever, recurring unresolved toolchain issues) | `go build -buildmode=c-shared` (Android) / `-buildmode=c-archive` (iOS), loaded via `dart:ffi` directly |
| pub `highlight` (pd4d10) | Stale ~5 years, unverified uploader flag on pub.dev | pub `highlighting` + `flutter_highlighting` |
| `github.com/unidoc/unioffice` (current) | Commercial product requiring a license-key/API check-in even on the "free" tier — conflicts with Eden Press's own no-implicit-network security goal | Hand-rolled OOXML zip via stdlib `archive/zip` + `encoding/xml` |
| `chroma/v3` | Alpha only (`v3.0.0-alpha.5`), breaking iterator API, not production-ready | `chroma/v2` (v2.27.0) |
| `goldmark/v2` | Active beta, API "almost stable... some parts may change," extension ecosystem not yet ported | `goldmark` v1 (v1.8.4) — revisit at GA |
| `gorilla/css/scanner` | Pure tokenizer only, no grammar/parse-tree layer at all; low activity (last release 2023) | `tdewolff/parse/v2/css` |
| `goldmark-meta@v2.0.0` paired with `goldmark@v1.8.4` | Version mismatch — goldmark-meta's v2 line targets goldmark/v2 beta, not v1 | `goldmark-meta@v1.1.0` (or `abhinav/goldmark-frontmatter@v0.3.0`) |

## Stack Patterns by Variant

**If targeting v1 ship (Marp-compatible slides, zero JS):**
- goldmark v1.8.4 + goldmark-meta v1.1.0 (or abhinav/goldmark-frontmatter) + tdewolff/parse css + chroma v2 + latex2mathml (forked) + go-latex/latex (Codeberg) + chromedp + cobra + fsnotify + koanf + hand-rolled OOXML + bluemonday.
- Because this is exactly the set the proposal already committed to, pressure-tested and confirmed viable, with three corrections applied (go-latex/latex address, gomobile removal, highlight→highlighting swap).

**If/when goldmark v2 reaches GA (future migration candidate):**
- Re-evaluate the whole Markdown layer against goldmark v2's semantic AST, position tracking, and non-HTML renderer generics — this lines up unusually well with Eden Press's own "output as data" / profile-abstraction differentiators (PROPOSAL §13.1, §13.3). Not a v1 blocker; a deliberate later objective.

**If Wasm bundle size becomes a real product blocker (Flutter Web):**
- Spike TinyGo against the actual `press` package (not just a toy program) before committing — partial stdlib/reflect support is the risk, and goldmark's own dependency chain is the thing to test against, not a synthetic benchmark.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `goldmark@v1.8.4` | `goldmark-meta@v1.1.0` | Do **not** pair with `goldmark-meta@v2.0.0` (targets goldmark/v2 beta) |
| `goldmark@v1.8.4` | `extension.GFM` (same module) | No extra pin |
| `chroma/v2@v2.27.0` | Go 1.21+ | Stay off `chroma/v3` alpha line until it tags a stable v3.0.0 |
| Go 1.25+/1.26.x | `fsnotify@v1.10.1` (needs Go 1.23+), `tdewolff/parse/v2@v2.8.13`, `koanf/v2@v2.3.5` | All comfortably satisfied by the recommended Go floor |
| `flutter_math_fork@0.7.4` / `highlighting@0.9.0` | Flutter 3.44.x / Dart 3.12.x (current stable) | Verify against the exact Flutter pin chosen; neither states a hard SDK ceiling in its pubspec metadata as checked |
| `chromedp@v0.16.0` | Any recent stable Chrome/Chromium, or a Chrome for Testing pinned build | No hard version coupling documented beyond normal CDP protocol drift |
| Android `.so` (`-buildmode=c-shared`) / iOS `.a` (`-buildmode=c-archive`) | `dart:ffi` `DynamicLibrary.open()` / `.process()` | Standard cgo build modes — no `gomobile` dependency needed |

## Sources

- pkg.go.dev/github.com/yuin/goldmark, `/v2`, `/extension`, `/ast` — version, extension API, GFM/strikethrough default tag, v2 beta status
- github.com/yuin/goldmark/releases — confirmed v1.8.4 latest stable, v2.0.0-beta.1–beta.5 as active pre-releases (2026-07-02–07-12)
- raw.githubusercontent.com/yuin/goldmark/master/README.md — cross-check, no v2 mention in README (confirms v2 is not yet the documented default)
- pkg.go.dev/github.com/yuin/goldmark-meta, versions tab — v1.0.0/v1.1.0 vs v2.0.0 compatibility split
- github.com/abhinav/goldmark-frontmatter — version, license, YAML/TOML support, maintenance date
- pkg.go.dev/github.com/tdewolff/parse/v2/css — token/grammar-stream model, selector/at-rule/round-trip behavior
- pkg.go.dev/github.com/gorilla/css/scanner — tokenizer-only confirmation
- github.com/vanng822/css, pkg.go.dev/github.com/vanng822/go-premailer — alternative CSS engine, activity signals
- pkg.go.dev/github.com/alecthomas/chroma/v2, `/v3` — version, HTML formatter options, line-highlight, v3 alpha status, license (MIT+OFL-1.1 via `?tab=licenses`)
- git.sr.ht/~mekyt/latex2mathml, pkg.go.dev listing — license, pseudo-version, construct inventory, dormancy
- github.com/go-latex/latex (archived notice) + codeberg.org/go-latex/latex — corrected canonical location, license, activity, PNG rendering confirmation
- pkg.go.dev/github.com/chromedp/chromedp — version, PrintToPDF pattern, screenshot actions, determinism flags, Chrome discovery
- WebSearch: Chrome for Testing / pinned reproducible Chrome — confirms the pinning mechanism referenced in PROPOSAL's open decision #3
- pkg.go.dev/github.com/spf13/cobra, github.com/urfave/cli/v3 — versions, feature comparison
- pkg.go.dev/github.com/fsnotify/fsnotify — version, platform support, documented limitations (non-recursive, atomic-save, cross-platform semantics)
- pkg.go.dev/github.com/knadh/koanf/v2 — version, parser/provider architecture, explicit viper comparison from koanf's own docs
- github.com/unidoc/unioffice — direct README quote confirming current commercial/license-key requirement (no free/AGPL operable tier)
- WebSearch: pure-Go OOXML/PPTX alternatives — survey confirming no viable actively-maintained free alternative exists
- pkg.go.dev/github.com/microcosm-cc/bluemonday — version, custom-policy API
- WebSearch: Go release history (go.dev/doc/devel/release, go.dev/blog/go1.26) — Go 1.26.5 / 1.25.12 current supported versions
- pkg.go.dev/golang.org/x/mobile/cmd/gomobile + WebSearch on gomobile issue-tracker activity — AAR/XCFramework output shape, minimal-maintenance status
- go.dev/wiki/WebAssembly — `GOOS=js GOARCH=wasm`, `wasm_exec.js` location (`lib/wasm/`), `GOOS=wasip1` scope, binary-size figures
- WebSearch: TinyGo vs standard Go for Wasm — size reduction figures and stdlib/reflection caveats
- pub.dev/packages/flutter_math_fork — version, license, maintenance status, KaTeX-port confirmation
- pub.dev/packages/highlight, pub.dev/packages/highlighting + WebSearch — staleness of `highlight`, confirmed successor `highlighting`/`flutter_highlighting`, alternative `syntax_highlight`
- WebSearch: Flutter/Dart stable release — Flutter 3.44.6 / Dart 3.12.2 current stable (2026-07-09)

---
*Stack research for: Eden Press (Go/Dart Marp-compatible document-generation framework)*
*Researched: 2026-07-20*
