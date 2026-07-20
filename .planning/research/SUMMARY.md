# Project Research Summary

**Project:** Eden Press
**Domain:** Go (+ Dart/Flutter client) Marp-compatible document-generation framework — zero-JavaScript backend, library-first
**Researched:** 2026-07-20
**Confidence:** HIGH overall, with acknowledged MEDIUM pockets (Dart/Flutter FFI+WASM toolchain mechanics; CSS-AST diff tooling, which is unproven engineering, not a spike-validated approach)

## Executive Summary

Eden Press is a clean-room, Go-native reimplementation of the Marpit -> Marp Core -> Marp CLI stack (Markdown-to-slide-deck generation), engineered from the ground up to have **zero JavaScript in the backend** and to be **library-first** (`press.Render()` embeddable in any Go service, not a Node CLI wrapper). Two feasibility spikes already de-risked the two scariest unknowns -- goldmark achieves 97% structural parity with markdown-it on a targeted 32-case corpus, and pure-Go `latex2mathml`->MathML rendering is 100% converter-bugs-not-engine-limits on a 20-case math corpus. All four research passes (stack, features, architecture, pitfalls) independently converge on the same conclusion: **the stack decisions in PROPOSAL.md hold**, but every "port this library" task the proposal describes as integration work is actually real, standalone engineering -- a CSS selector-rewriter with no upstream Go analogue, a directive/slide-splitting engine that must be re-architected (not translated) onto goldmark's priority-ordered pipeline instead of markdown-it's mutable-ruler model, and a hand-rolled OOXML writer because the only mature Go PPTX library (`unioffice`) is commercial-licensed in a way that conflicts with Eden Press's own no-implicit-network security ethos.

The recommended approach is a five-package `chase/` framework layer (`markdown`, `directive`, `theme`, `model`, `profile`) sitting under a single public `press/` facade, with a hard, CI-enforced wall keeping headless Chrome (`chromedp`) out of everything except `convert/`. Two architectural decisions are load-bearing and must be made in Objective 0/1, not deferred: (1) `chase/model` (the structured JSON document model) and `chase/profile` (the output-profile interface enabling paged-docs/article/EPUB later) must be first-class packages from day one -- folding them into "Marpit-in-Go" as an afterthought is the single most expensive mistake this project could make, since retrofitting either after the slides profile hardens means a rewrite, not an extension; and (2) `press.Render` must call goldmark's two-phase `Parse()`+`Render()` explicitly, never the one-shot `Convert()` convenience wrapper, because only the explicit path exposes the finalized AST needed to build the docmodel from the same tree that produces HTML.

The primary risks, in order of how badly PROPOSAL.md's own framing under-states them: the base-parser-parity spike (97%) only tested CommonMark's surface, not the full 600+/672-example spec (tabs, raw-HTML block boundaries, loose-list determination are all untested gaps); the real Marpit-port risk is architectural (goldmark's fixed, priority-ordered pipeline vs. markdown-it's mutable in-flight token stream), not parser-choice; math "converter-hardening" is five separately-sized root-cause classes plus a hard, permanent Chromium-MathML-Core structural ceiling (no `<mlabeledtr>`, limited `<mtable>` attributes) that must auto-route to an SVG/PNG fallback rather than be "fixed"; and HTML-sanitization parity with Marp's `xss` library is a **behavioral** semantics problem (bluemonday strips, `xss` escapes; GFM's disallowed-tag filter isn't automatic in goldmark) that only adversarial round-trip testing catches, not a static allow-list comparison. All of these are bounded and well-understood risks with documented mitigations -- none invalidate the stack or architecture -- but the roadmap must budget them as real engineering, not "wire up a library."

## Key Findings

### Recommended Stack

STACK.md pressure-tested (not re-selected) every library PROPOSAL.md already committed to, and every item **holds**, with five corrections the roadmap must carry forward. goldmark v1.8.4 is the Markdown base -- its extension mechanism (`BlockParser`/`InlineParser`/`ASTTransformer`/`NodeRenderer`) exactly matches what directive-parsing, slide-splitting, and inline-SVG rendering need; goldmark v2 exists only as an active beta (do not adopt for v1, but it converges unusually well with Eden Press's own structured-AST ambitions -- track it as a deliberate future migration once v2 reaches GA). `tdewolff/parse/v2/css` is confirmed to be a **token/grammar stream, not a navigable AST** -- the "CSS-AST transform" language in PROPOSAL.md SS4.2/SS5.1 is a misnomer; the selector-rewriter that scopes theme CSS to `.marpit`/`section` is 100% custom code Eden Press must design, build, and test as its own subsystem. `chroma` v2 handles highlighting but its class taxonomy has zero relationship to highlight.js's `.hljs-*` classes the bundled themes target -- a one-time custom `chroma.Style`/formatter class-map is required. `latex2mathml` (MIT, single-author, dormant) must be forked-and-owned, not `go get`-and-trusted; `github.com/go-latex/latex` is **archived** -- the maintained module moved to `codeberg.org/go-latex/latex` and every reference must use that path. `unioffice` is now a fully commercial product requiring a live license-key check-in even on its "free" tier -- this directly conflicts with the no-implicit-network security goal, confirming hand-rolled OOXML (`archive/zip`+`encoding/xml`, stdlib only) as the only viable PPTX path, not merely a fallback. On the Dart/Flutter side: drop `gomobile bind` entirely (it generates unneeded Java/ObjC binding layers and is minimally maintained) -- use plain `go build -buildmode=c-shared`/`c-archive`, which is what `dart:ffi` actually consumes; and replace the stale pub `highlight` package with its actively-maintained successor `highlighting`+`flutter_highlighting`.

**Core technologies:**
- **goldmark v1.8.4** -- Markdown parsing/AST; explicit `Parse()`+`Render()` two-phase call (not `Convert()`) is load-bearing for exposing the AST to the docmodel builder.
- **tdewolff/parse/v2/css** -- CSS tokenizer/grammar stream (not an AST); the selector-rewriter built on top of it is real, standalone engineering, budgeted as its own component.
- **chroma v2** -- syntax highlighting; needs a one-time class-name remap to `.hljs-*` to keep the 3 bundled themes' CSS verbatim.
- **latex2mathml (forked) + codeberg.org/go-latex/latex** -- LaTeX->MathML with SVG/PNG fallback for constructs Chromium's MathML Core can't render at all.
- **chromedp v0.16.0** -- headless-Chrome driver for PDF/PNG export only; `Page.PrintToPDF` requires `cdproto/page` inside an `ActionFunc` (normal usage, not a gap).
- **cobra + fsnotify + koanf** -- CLI, watch, config; fsnotify needs explicit directory-walk (non-recursive) and atomic-save-safe filtering designed in, not discovered later.
- **bluemonday** -- HTML sanitization; API-sufficient, but matching Marp's `xss` policy is a behavioral/semantic exercise, not a tag-list copy.
- **Hand-rolled OOXML (stdlib archive/zip + encoding/xml)** -- PPTX generation; `unioffice` is explicitly rejected (commercial license-key requirement).
- **Go->WASM (`GOOS=js GOARCH=wasm`) + dart:ffi (c-shared/c-archive)** -- Dart/Flutter binding; TinyGo is a stretch optimization to spike later, not a v1 default (reflection/stdlib compatibility risk with goldmark's dependency chain).

### Expected Features

FEATURES.md verified the full Marpit -> Marp Core -> Marp CLI feature surface directly against upstream docs/source (not assumed from training data) and found several corrections worth carrying into planning: `size`/`math`/`@auto-scaling` are Marp-**Core**-layer directives, not Marpit-layer, and must be routed through a `customDirectives` extension point, not hardcoded into the framework; there is **no** Markdown-level `auto-scaling` directive (theme-CSS-metadata only); line-highlight code syntax (`{1-3}`) is **not** actually part of marp-core (a community plugin upstream never shipped natively) -- this moves from table stakes to a legitimate differentiator; and Marp CLI has no true remote-URL-as-input feature (only stdin `-` and server mode), so that scope item should be corrected in requirements.

**Must have (table stakes):** full directive system (global/local/spot, front-matter + comment syntax, custom-directive extension point); slide splitting (`---`/`headingDivider`) with the setext-heading ambiguity resolved; `.marpit` container + inline-SVG mode + advanced backgrounds (multi/split/filters -- hard-depend on inline-SVG, not parallelizable with it); theme-CSS engine (`@theme`/`@size`/`@auto-scaling`, `:root` remap, selector scoping, `@import`/`@import-theme`, pagination hook); 3 bundled themes embedded verbatim with attribution; GFM tables/strikethrough/hard-breaks, heading slugs, native emoji, current (v4-era) HTML allow-list sanitization; chroma highlighting; latex2mathml math with the spike's converter-hardening fixes; CLI (HTML/`bare` template, watch, server, preview, YAML/JSON/TOML config -- no JS config); PDF+PNG/JPEG export via chromedp.

**Should have (differentiators):** output-profile abstraction (architectural, v1 -- slides is profile #1, everything else stays additive); library-first `press.Render()` API (near-free if module boundaries hold); structured document-model/JSON-AST output alongside HTML+CSS (v1, as an API-shape decision); deterministic/reproducible output (cheap if disciplined from commit one, expensive to retrofit).

**Defer (v1.x/v2+):** editable native-OOXML PPTX text boxes (a genuine quality upgrade over Marp's LibreOffice-mediated "editable" mode); design-token theming (multi-tenant brand support -- depends on the theme-CSS engine existing first); native line-highlight code feature; additional output profiles (paged docs/reports, article, EPUB -- the actual long-term payoff of the profile abstraction); tagged/accessible PDF/A export; bespoke-equivalent presenter/navigation viewer (explicitly out of step with the zero-JS-forward v1 posture).

### Architecture Approach

The recommended structure is a strict four-layer module tree -- `chase/` (framework: `markdown`, `directive`, `theme`, `model`, `profile` as five sibling packages, no themes/batteries), `profiles/slides` (the only shipping `profile.Profile` implementation in v1), `press/` (batteries + the single public API surface, `press.Render`), and `convert/`+`cmd/eden-press` (the only consumers of Chrome/CLI concerns). `chase/directive` has zero goldmark import (pure carry-forward state machine, independently unit-testable); `chase/theme` has zero dependency on `chase/markdown` (CSS scoping is a standalone text transform, matching Marpit's own `theme.js`) -- these two can be built in parallel by different objectives/agents. `chase/model` is a direct recursive walk of the **finalized** goldmark AST into Eden Press's own versioned, JSON-serializable `Document{Meta,Sections,Outline}` tree -- a second sink off the same parse, not a second parse or HTML-reverse-engineering. `convert/` is structurally isolated as the only package permitted to import `chromedp` (enforceable via a `go list -deps ./press/... | grep chromedp` CI check) -- and within `convert/`, `convert/pptx` consumes `chase/model` directly and needs **no Chrome at all**, making it a sibling objective to the chromedp-based PDF/PNG exporter, not a downstream dependent of it. The Dart/Flutter binding (`bind/capi`+`bind/dart`) is a thin JSON-in/JSON-out C-ABI wrapper around `press.Render`, gated only on `press/`'s API stability -- not on the CLI or convert/ layers.

**Major components:**
1. `chase/directive` + `chase/theme` -- pure-logic carry-forward state machine and CSS-scoping engine; parallel-buildable, no shared dependency.
2. `chase/markdown` + `chase/model` + `chase/profile` -- goldmark Extenders wiring directives into the AST, the structured docmodel builder (direct AST walk, must exist as a first-class package from day one), and the `Profile` interface/registry (validated bottom-up from what `profiles/slides` actually needs).
3. `press/` -- the public facade (themes, emoji, chroma, latex2mathml, bluemonday, `press.Render`) and the only package most consumers import; **zero transitive dependency on chromedp**, verified continuously.
4. `convert/pdf.go`+`png.go` (chromedp) and `convert/pptx` (native OOXML) -- two structurally different export paths with different dependency edges, buildable and tested independently of each other.
5. `bind/capi`+`bind/dart` -- one Go core exposed via C-ABI, compiled three ways (`c-shared` Android, `c-archive` iOS, `GOOS=js/wasm` Web), consuming only `press/`'s stable `Options`/`Output` JSON contract.

### Critical Pitfalls

1. **Base-parser-parity spike (97% on 32 cases) is a downgrade of risk, not a retirement of it** -- the untested surface (tab expansion, raw-HTML block boundaries, loose-list determination, GFM autolink edge cases, the GFM disallowed-tag filter goldmark doesn't implement) concentrates exactly where implementations diverge in practice. Avoid by running the full CommonMark (600+) + GFM (672) spec sweep in Objective 0, tracked per spec section, not aggregate percentage.
2. **goldmark's priority-ordered pipeline does not map 1:1 onto markdown-it's mutable-ruler model** -- this is the real engineering risk of the Marpit-in-Go port, larger than the parser-parity question. Slide-splitting and `![bg]` each need an explicit, up-front priority decision (out-prioritize a built-in parser, mirroring goldmark's own `TaskCheckBoxParser`-vs-`LinkParser` precedent) or an `ASTTransformer` rewrite pass -- picking the wrong mechanism causes silent double-parsing or dead-on-arrival custom parsers.
3. **The CSS selector-rewriter has no upstream Go analogue and is harder than "port to tdewolff"** -- `tdewolff/parse/v2/css`'s nesting support is recent/still-evolving and `:is()`/`:where()` lex as generic function tokens with no selector-AST API; Marpit's own pipeline is five-plus ordered, stateful passes (meta parse -> `:root` remap+specificity fix-up -> selector scoping -> `@import`/`@import-theme` resolution -> render-time injection) that cannot be expressed as a single-pass token rewrite. Budget an owned `Stylesheet{Meta,Rules,Atoms}` intermediate model as its own tested component.
4. **`unioffice`'s AGPLv3/commercial licensing is a blocking decision, not a mid-implementation discovery** -- no mature MIT/Apache pure-Go PPTX-creation alternative exists; the only viable path is a hand-rolled OOXML writer (stdlib `archive/zip`+`encoding/xml`), which must be decided before Objective 5's design doc is written, not after code exists.
5. **Math and sanitization "parity" are both harder than they look**: `latex2mathml` converter-hardening is five separately-sized root-cause classes plus a **permanent** Chromium MathML-Core structural ceiling (no `<mlabeledtr>`, limited `<mtable>`) requiring auto-routed SVG/PNG fallback detection, not a fix; bluemonday-vs-`xss` parity is a strip-vs-escape behavioral difference plus an always-on directive/comment parsing path that is itself an untrusted-input trust boundary -- both require adversarial/round-trip testing, not static list comparisons.

## Implications for Roadmap

Research from all four dimensions converges on the same objective skeleton already implied by PROPOSAL.md's own phasing (Objectives 0-7) -- but the architecture research adds package-level dependency detail that changes how each objective must be scoped internally. Suggested structure:

### Objective 0: Conformance Corpus + Acceptance Gate
**Rationale:** Must exist before any implementation begins -- every other objective (including the Dart binding, much later) is validated against it. This is the single point where "looks done" and "is done" diverge across the whole project.
**Delivers:** Golden corpus extracted from Marp's own MIT-licensed Jest fixtures; a DOM-normalized HTML diff runner (extending the spike's approach, with normalization rules scoped as an explicit allow-list, not a general whitespace-ignore); a **new, unproven** CSS-AST diff runner (the HTML-diff approach doesn't extend to CSS automatically -- build and validate this as its own exercise); a scheduled upstream-drift-check mechanism (CI job diffing against Marp's latest tag, not a manual reminder); the full CommonMark (600+)/GFM (672) spec sweep, tracked per-section; and initial NOTICE/CREDITS + a per-PR "new vendored asset" checklist process.
**Addresses:** No user-facing feature; the acceptance gate for everything below.
**Avoids:** Pitfall 1 (residual CommonMark/GFM gaps), Pitfall 13 (normalization false-negatives, unproven CSS-diff, drift-tracking), Pitfall 14 (attribution completeness process).

### Objective 1: `chase/` Framework Core + `profiles/slides`
**Rationale:** The framework layer. Architecture research is explicit that this must ship as **five sibling packages, not one monolith** -- burying `model`/`profile` in prose (as PROPOSAL.md currently does) instead of giving them a package is the gap this objective must close. `chase/directive` (pure state machine, no goldmark import) and `chase/theme` (Stylesheet model over tdewolff's token stream) have no dependency on each other and can be built/tested in parallel; `chase/markdown` (goldmark Extenders: directive syntax, slide-splitting, `![bg]`, inline-SVG) depends on `chase/directive`; `chase/model` (the structured docmodel) depends on `chase/markdown`'s finalized AST and must be designed in from day one, not retrofitted; `chase/profile` (the `Profile` interface) should be validated bottom-up from what `profiles/slides` actually needs, not speculatively generalized before a second profile exists. `profiles/slides` is where "does this look like Marp yet" gets proven against Objective 0's corpus.
**Delivers:** Marp-compatible parsing, directive resolution, slide-splitting, inline-SVG/advanced-backgrounds, theme-CSS scoping engine, structured JSON document model, and the profile abstraction as an architectural boundary (slides is the only shipping implementation).
**Uses:** goldmark v1.8.4 via the explicit `Parse()`+`Render()` two-phase call (never `Convert()` -- this is the mechanism that lets the docmodel and the HTML renderer share one finalized AST); `tdewolff/parse/v2/css` token stream plus a hand-built `Stylesheet` intermediate model.
**Avoids:** Pitfall 2 (goldmark-pipeline-vs-markdown-it-ruler mismatch -- write an explicit priority decision per Marpit construct), Pitfall 3 (CSS selector-rewriter maturity gaps), Pitfall 4 (inline-SVG `foreignObject` fragility -- treat as a rasterized, not just HTML-string, conformance target).
**Decision gate:** whether `chase/*` stays exported Go (documented, importable by advanced consumers) or gets an `internal/` prefix to force all traffic through `press/` -- decide explicitly, don't default silently.

### Objective 2: `press/` Batteries + Public API
**Rationale:** Needs Objective 1 in place. This is where the "Importable Go API" milestone lands and the point at which Dart binding work (Objective 6) can start, gated on this objective's API stability alone.
**Delivers:** 3 bundled themes (`go:embed`, verbatim, per-file MIT headers preserved); GFM tables/strikethrough/hard-breaks; heading slugs; native emoji; bluemonday sanitization matching Marp's `xss` allow-list **semantically** (strip-vs-escape behavior documented, GFM disallowed-tag filter explicitly implemented, directive/comment path treated as its own trust boundary); chroma highlighting with a custom `.hljs-*`-shaped class remap; latex2mathml->MathML math with the spike's bounded converter fixes.
**Addresses:** Table-stakes batteries features (sanitization, emoji, highlighting, GFM, heading slugs, math baseline).
**Avoids:** Pitfall 12 (bluemonday/`xss` semantic parity -- adversarial round-trip tests, not tag-list comparison).

### Objective 3: CLI (`cmd/eden-press`)
**Rationale:** A thin consumer of `press/`; standard Go CLI patterns apply, low research risk.
**Delivers:** cobra-based `convert`/`watch`/`serve`/`preview` subcommands; fsnotify watch mode with explicit directory-walk (non-recursive by design) and atomic-save-safe filtering (watch the parent directory, filter by `Event.Name`); koanf-based YAML/JSON/TOML config (JS config intentionally dropped).
**Uses:** cobra, fsnotify, koanf -- all HIGH-confidence, well-documented libraries.

### Objective 4: `convert/pdf`+`convert/png` (chromedp raster export)
**Rationale:** Structurally isolated as the only code touching Chrome -- enforce with a CI `go list -deps` check so `press/`/`chase/`/`profiles/` never regress into a transitive chromedp dependency. Sibling to Objective 5, not a prerequisite for it.
**Delivers:** PDF export via `cdproto/page.PrintToPDF` inside an `ActionFunc` (not a first-class chromedp action -- expected usage, not a gap); PNG/JPEG per-unit screenshot export; explicit determinism engineering (fixed viewport, timezone/locale pinning, animation disabling -- chromedp's defaults get you partway, not all the way); Chrome discovery (system-first, with an optional Chrome-for-Testing pinned-download command); CI hardened against container-specific Chrome flakiness.
**Avoids:** Pitfall 4 (foreignObject fragility specifically in the PDF-export path -- a documented Chrome >=108 regression class), Pitfall 11 (shm/sandbox/root/user-data-dir CI flakiness -- adopt `chromedp/headless-shell`, pin the Chrome version, re-test PDF export specifically on any version bump).

### Objective 5: `convert/pptx` (native OOXML) -- sibling to Objective 4
**Rationale:** Needs no Chrome; consumes `chase/model` directly. Because it has a fundamentally different (shorter, cheaper, Chrome-free) dependency edge than PDF/PNG, it should be scheduled and staffed as an independent objective, not a follow-on to Objective 4.
**Delivers:** Hand-rolled OOXML zip (stdlib `archive/zip`+`encoding/xml` only); editable text-box PPTX -- a genuine quality upgrade over Marp CLI's own image-per-slide default and LibreOffice-mediated "editable" mode.
**Avoids:** Pitfall 7 (EMU-unit and group-shape `chOff`/`chExt` coordinate traps -- build a dedicated EMU conversion utility and programmatic positional tests), Pitfall 8 (the `unioffice` AGPLv3/commercial licensing trap -- this decision must be made and documented **before** the objective's design doc is written).

### Objective 6: Dart/Flutter Binding (`bind/capi`+`bind/dart`)
**Rationale:** Gated on Objective 2's API stability alone (touches only `press/`'s public `Options`/`Output` types), not on the CLI or convert/ layers -- can proceed in parallel with Objectives 3-5 once Objective 2 lands.
**Delivers:** C-ABI shim (`PressRender`/`PressFree`, JSON-in/JSON-out) built three ways (`c-shared` for Android/NDK, `c-archive` for iOS, `GOOS=js GOARCH=wasm` for Web); Flutter package with `dart:ffi`/`dart:js_interop` loaders; a conformance-corpus boundary-runner extension that exercises the compiled artifact through the same JSON entrypoint.
**Decision gate (must resolve before writing WASM-specific code):** standard Go (full stdlib/reflection compatibility, ~2MB+/500-660KB-gzipped binary) vs. TinyGo (4-20x smaller, but partial reflection/`encoding/json` support that plausibly breaks Eden Press's own JSON-AST output and YAML front-matter parsing) -- this has functional consequences, not just a performance tradeoff, and cannot be deferred to a later "optimize if needed" pass.
**Avoids:** Pitfall 9 (Android NDK vs. iOS `c-archive`+`lipo` are two independently-toolchained pipelines; a confirmed Go toolchain bug panics iOS builds on Apple Silicon if Android NDK is merely present -- test both independently in CI), Pitfall 10 (Go->WASM vs. TinyGo reflection incompatibility and non-interchangeable `wasm_exec.js` versions).

### Objective 7: Math-Fidelity Hardening + Auto-Fit Resolution
**Rationale:** Last by necessity, not low priority -- there is nothing to tune until the rest of the pipeline renders something. Cross-cutting across `press/math`, `profiles/slides` (fit markers), and `bind/dart` (native fit).
**Delivers:** The five separately-sized `latex2mathml` converter-hardening fixes (big-operator display-mode detection, `\binom`/`pmatrix` fence-sharing, `\sqrt[n]` argument parsing, `aligned`->`mtable` sub-parsing, `mathvariant`->Unicode-codepoint mapping table) -- budgeted as real engineering per PROPOSAL's own honest framing, not "8 small patches"; an explicit fallback-trigger detector (auto-route `\tag`/`\label`/complex multi-column `aligned` constructs to the SVG/PNG path, since Chromium's MathML Core structurally lacks `<mlabeledtr>` and most `<mtable>` attributes -- this is a permanent platform ceiling, not a bug to fix); STIX Two Math font bundled from the STIX-fonts-project's own OTF/WOFF2 releases (never Google Fonts' CDN, which has been reported to strip MATH-table data); resolution of the auto-fit mechanism (native Flutter / CSS-only `cqw` units / drop for browser-HTML).
**Avoids:** Pitfall 5 (converter-hardening under-scoping), Pitfall 6 (font tofu + structural Chromium-MathML gaps).

### Objective Ordering Rationale

- **Objective 0 is non-negotiable-first**: every downstream objective, including Objective 6 (Dart, much later), is validated against its corpus -- this is the literal acceptance gate, not just good practice.
- **Objective 1's internal parallelism (`chase/directive` || `chase/theme`) is a genuine scheduling opportunity**: these two packages share no dependency edge and can be assigned to different objectives/agents simultaneously, unlike the rest of the graph which is largely linear.
- **Objectives 4 and 5 are siblings, not sequential** -- this is the single most consequential ordering correction architecture research surfaces relative to a naive reading of PROPOSAL.md's phasing (which lists PPTX as "Objective 5," implying "after PDF/PNG"). PPTX-native has a shorter, Chrome-free dependency chain and should not wait on Chrome-export work completing.
- **Objective 6 is gated on Objective 2 (API stability), not Objective 3 (CLI)** -- the Dart binding only ever touches `press/`'s public types, so tying it to CLI completion in the roadmap would be an artificial, avoidable delay.
- **This ordering directly avoids the anti-pattern architecture research flags most strongly**: baking "slide"/`section`/16:9 assumptions into `chase/theme` or `chase/model` instead of naming things `Unit`/`Section` and letting `profiles/slides` supply the slide-specific defaults -- get the naming and interface shape right in Objective 1, since fixing it after Objective 5 (a second profile) exists is the expensive path.

### Decision Gates to Resolve During Planning

These three decisions recur across the research and should be flagged explicitly at the relevant objective's planning stage rather than left implicit:
- **Standard Go vs. TinyGo for the WASM target** (Objective 6, before any WASM-specific code is written) -- functional risk (reflection-dependent JSON/YAML paths), not just a size/perf tradeoff.
- **MathML-quality fallback trigger** (Objective 7) -- which specific TeX constructs (confirmed candidates: `\tag`, `\label`, complex multi-column `aligned`) auto-route to the SVG/PNG fallback rather than attempting native MathML; this needs a concrete, testable detection rule, not a vague heuristic.
- **`chase/*` internal vs. exported Go** (Objective 1) -- whether to enforce "only `press/` is the front door" via an `internal/` prefix, or leave `chase/*`/`profiles/*` independently importable for advanced consumers.

### Research Flags

Objectives likely needing deeper research during planning (`/devflow:research-objective`):
- **Objective 1** -- the CSS selector-rewriter (`chase/theme`) has no upstream Go analogue (`postcss-selector-parser` has no Go port); Pitfalls research explicitly flags this as needing its own design pass before implementation, not "use tdewolff and go."
- **Objective 5** -- the `unioffice`-vs-hand-rolled-OOXML licensing decision and EMU/group-shape coordinate model need a confirmed-current check (no new permissive Go PPTX library may have emerged) before the design doc is written.
- **Objective 6** -- Dart/Flutter FFI+WASM patterns are architecture research's own MEDIUM-confidence area (no single canonical Go+Flutter guide exists); the standard-Go-vs-TinyGo decision and the Apple-Silicon/NDK toolchain interaction both warrant a focused spike.
- **Objective 7** -- the fallback-trigger detection rule and the full scope of the five math-hardening root-cause classes need explicit per-class estimation, not a lump "small" sizing.

Objectives with standard, well-documented patterns (research-objective likely unnecessary):
- **Objective 3** (CLI) -- cobra/fsnotify/koanf are HIGH-confidence, widely-used libraries with clear documented gotchas already surfaced (fsnotify's non-recursion and atomic-save behavior).
- **Objective 4** (chromedp PDF/PNG) -- the API patterns are HIGH-confidence and well-documented; the CI-hardening checklist (headless-shell, shm/sandbox/user-data-dir, version pinning) is already fully specified in PITFALLS.md and can be applied directly.
- **Objective 2** (press/ batteries) -- bluemonday/chroma/emoji APIs are HIGH-confidence; the main residual risk (sanitization semantic parity) has a concrete testing strategy already specified (adversarial round-trip diffing), not an open unknown.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Every library checked directly against pkg.go.dev/GitHub/Codeberg/pub.dev on the research date; only maturity/activity judgment calls (single-maintainer libraries) are softer, and those are flagged inline rather than hidden. |
| Features | HIGH (table stakes, anti-features) / MEDIUM-HIGH (differentiators) | Table stakes and anti-features verified against official Marpit/Marp Core/Marp CLI docs+source directly; differentiators are synthesized from PROPOSAL.md's own rationale, cross-checked against real ecosystem gaps (e.g., Marp's own lack of a public AST export). |
| Architecture | MEDIUM-HIGH | goldmark/chromedp APIs verified HIGH against official docs; the CSS-token-stream correction is HIGH-confidence (directly checked, and corrects PROPOSAL.md's own "CSS AST" framing); Dart FFI/WASM patterns are MEDIUM -- consistent across multiple independent tutorials but no single canonical Go+Flutter reference exists. |
| Pitfalls | HIGH (library/spec facts) / MEDIUM (community-reported flakiness) | goldmark internals, CommonMark/GFM spec edge cases, OOXML EMU units, unioffice licensing, gomobile/TinyGo behavior, and Chromium MathML gaps are all confirmed against official docs/source/spec; chromedp/Docker flakiness and foreignObject regression reports are MEDIUM, verified across 2+ independent sources each. |

**Overall confidence:** HIGH, with two acknowledged MEDIUM pockets (Dart/Flutter binding toolchain mechanics; the CSS-AST diff tooling, which is genuinely new engineering with no spike precedent).

### Gaps to Address

- **CSS-AST diff tooling is unproven** -- the parser spike validated DOM-normalized HTML diffing; the analogous CSS-level diff (for theme-scoping conformance) has never been built or spike-tested. Address in Objective 0 as its own validation exercise, not an assumed extension of the HTML approach.
- **Only 32 of 600+/672 CommonMark/GFM spec examples have been run** -- the base-parser-parity number (97%) is real but covers a deliberately curated, Marp-relevant subset. Address by running the full spec sweep in Objective 0 before treating base-parser risk as retired.
- **No canonical single Go+Flutter FFI/WASM reference exists** -- the binding pattern is a cross-tutorial consensus (MEDIUM confidence), not a single authoritative source. Address with an early, focused spike in Objective 6 before committing to the full three-target build pipeline.
- **The MathML fallback-trigger detection rule is not yet concretely specified** -- candidate constructs (`\tag`, `\label`, complex `aligned`) are named, but the exact detection logic and threshold need to be nailed down during Objective 7 planning, not left as "auto-route when it seems necessary."
- **goldmark v2's GA timeline is unknown** -- it's an active beta as of this research (5 betas cut in the two weeks before research date). Not a v1 blocker, but the roadmap should carry an explicit note to re-evaluate the whole Markdown layer against v2 once it reaches GA, since it converges well with Eden Press's own structured-AST differentiator.
- **No reference implementation exists for the hand-rolled OOXML writer** beyond reading pre-commercial `gooxml`/`unioffice` source for schema shape (not importable) and the officeopenxml.com/ECMA-376 spec directly -- Objective 5 should budget this as first-principles engineering, informed but not accelerated by any existing Go library.

## Sources

### Primary (HIGH confidence)
- pkg.go.dev -- goldmark, goldmark/parser, goldmark/renderer, goldmark/ast, goldmark-meta, tdewolff/parse/v2/css, chroma/v2, chromedp, cobra, fsnotify, koanf/v2, bluemonday, microcosm-cc/bluemonday, golang.org/x/mobile/cmd/gomobile
- GitHub/Codeberg official repositories -- yuin/goldmark (releases, README), tdewolff/parse, alecthomas/chroma, git.sr.ht/~mekyt/latex2mathml, codeberg.org/go-latex/latex (corrected canonical location), chromedp/chromedp, unidoc/unioffice (README licensing confirmation), fsnotify/fsnotify, knadh/koanf
- marp-team/marpit, marp-team/marp-core, marp-team/marp-cli -- README + docs/directives.md, docs/image-syntax.md, docs/theme-css.md, themes/README.md (raw source, fetched directly)
- pub.dev -- flutter_math_fork, highlight, highlighting, flutter_highlighting
- go.dev -- WebAssembly wiki, release history (Go 1.25/1.26), Flutter/Dart official docs (c-interop, Wasm compilation, JS interop)
- officeopenxml.com / ECMA-376 -- OOXML DrawingML shape/size/EMU-unit reference
- CommonMark Spec (spec.commonmark.org), github/cmark-gfm -- spec example counts and reference implementation

### Secondary (MEDIUM confidence)
- Chromium bug trackers and blink-dev discussion -- foreignObject stacking-context history, Chrome >=108 PDF-export SVG regression, Chrome v125-class print-pipeline sandbox regression
- chromedp/chromedp issue #297, marp-cli PRs #292/#80, issue #475 -- Chrome discovery/root-execution/Docker flakiness patterns
- golang/go issue #47296 -- confirmed Apple-Silicon + Android-NDK-present iOS build panic
- Multiple independent Go+Flutter FFI/WASM tutorials (dev.to, openprivacy.ca, roothex200.hashnode.dev, Medium) -- cross-checked pattern consensus, no single canonical source
- MathML Core / Igalia restoration coverage, MathJax 4.0 docs, google/fonts issue #3773 -- Chromium MathML structural gaps and font-subsetting risk

### Tertiary (LOW confidence)
- None flagged as standalone low-confidence findings -- all research claims were corroborated against at least one official source or 2+ independent secondary sources per the pitfalls methodology.

---
*Research completed: 2026-07-20*
*Ready for roadmap: yes*
