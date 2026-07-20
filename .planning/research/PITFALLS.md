# Pitfalls Research

**Domain:** Go/Dart Marp-compatible document-generation framework (Markdown parser fidelity, CSS theme scoping, native MathML, OOXML export, Go↔WASM/FFI multi-binding, headless-Chrome raster export, untrusted-Markdown sanitization)
**Researched:** 2026-07-20
**Confidence:** HIGH for verifiable library/spec facts (goldmark internals, CommonMark/GFM spec edge cases, OOXML EMU units, unioffice licensing, gomobile/TinyGo behavior, Chromium MathML gaps, bluemonday semantics — all confirmed against official docs/source/spec); MEDIUM for community-reported flakiness patterns (chromedp/Docker issues, foreignObject regressions) verified across 2+ independent sources; builds directly on the two completed spikes in `PROPOSAL.md` §11–12 rather than repeating them.

This file assumes the reader has read `PROPOSAL.md` (especially §9, §11, §12, §14) and `.planning/PROJECT.md`. Every pitfall below is scoped to go **beyond** what the spikes already retired.

---

## Critical Pitfalls

### Pitfall 1: Residual CommonMark/GFM corner cases beyond the 32-case spike

**What goes wrong:**
The parser spike (§12) proved 31/32 structural match on a *targeted* corpus — a strong indicator, not proof of full parity. CommonMark has 600+ spec examples and GFM adds ~70 more; the untested surface concentrates in exactly the areas both the spec authors and implementers call "genuinely hard": (1) **tab expansion** — tabs are column-relative (`4 - (columns % 4)`), not literal 4-space substitution, and the interaction with blockquote markers (`>` + tab → 3 "virtual" spaces, one consumed by the delimiter) is a documented source of implementation divergence; (2) **raw-HTML block boundary rules** — an HTML block consumes everything until the next blank line, so a fenced-looking construct can silently become part of an HTML block, and starting a block with a non-block-level tag requires it to be alone on its line; (3) **tight vs. loose list determination** — CommonMark's own spec authors flag this as ambiguous (partially-loose lists, marker-type changes mid-list); (4) **GFM autolink extension edge cases** — trailing-punctuation stripping, parenthesis-balancing (`)`-count heuristic), and semicolon/entity-reference lookahead are all subtle, and GitHub's *actual* website behavior is documented to diverge from GFM's own written spec in places; (5) **GFM disallowed-raw-HTML tag filtering** (`<title>`, `<textarea>`, `<style>`, `<xmp>`, `<iframe>`, `<noembed>`, `<noframes>`, `<script>`, `<plaintext>`) — goldmark's GFM extension does **not** implement this filter itself, so it must be handled in the sanitization layer (ties to Pitfall 12), not assumed to be "part of GFM support."

**Why it happens:**
The spike deliberately weighted 32 cases toward Marp-relevant constructs and known traps, and got a clean result — creating a temptation to treat the base-parser risk as fully closed. It isn't; it's *downgraded*, exactly as §9 says. Tab handling and raw-HTML boundaries in particular are rarely hit by hand-written test corpora because authors don't naturally write tab-indented Markdown or boundary-straddling HTML — they show up in real-world/generated content instead.

**How to avoid:**
Objective 0 must run the **full** CommonMark `spec.txt` (extract via the `commonmark-spec` JSON format, ~600 examples) and the GFM `spec.txt` (672 combined examples, reference `github/cmark-gfm`) through the same DOM-normalized diff harness the spike used — not just Marp-relevant subsets. Explicitly add a tab-handling sub-corpus (tab-indented code, tab after `>`, tab inside list markers) since the standard suite under-weights it relative to its real-world divergence rate. Track pass rate per spec *section*, not just aggregate — a 97% aggregate can hide a 0% pass rate on one gnarly section (tabs, or loose-list determination) that happens to be small in the corpus.

**Warning signs:**
Aggregate conformance percentage looks high (>95%) but per-section breakdown shows 0-2 failing sections clustered in tabs/HTML-blocks/lists; Marpit-authored decks in the wild that use tab-indentation or HTML fragments straddling blank lines render differently than upstream Marp.

**Objective to address:**
Objective 0 (conformance corpus + runner) for the full-spec sweep; Objective 1 (Marpit-in-Go) for any per-rule custom parser escalation the sweep surfaces.

---

### Pitfall 2: goldmark's AST-transformer model doesn't map 1:1 onto markdown-it's mutable-ruler model for Marpit's actual plugins

**What goes wrong:**
The parser spike tested the **base parser only** (§12's own caveat). Marpit's real value-add — the slide-splitter, directive resolver, and `![bg]` background-image syntax — are **markdown-it core-ruler hooks that mutate the token stream in place** before/during rendering. goldmark's extension model is fundamentally different: block parsers run in priority order (lower number = earlier; defaults range 100–1000, e.g. `ThematicBreakParser`=200, before `ParagraphParser`=1000), inline parsers similarly prioritized, and `ASTTransformer`s run **last**, after the whole tree exists. Reimplementing "split on `---`, but only when not consumed as a setext heading" and "hijack `![bg ...](url)` image syntax into a non-rendering directive node" requires either (a) a custom `BlockParser`/`InlineParser` registered at a priority number *below* the built-in one it must pre-empt (exactly the pattern goldmark's own `TaskCheckBoxParser` uses to out-prioritize `LinkParser` for `![bg]`), or (b) an `ASTTransformer` rewrite pass after full parsing. Picking the wrong mechanism causes silent double-parsing or nodes that never fire because a lower-priority default parser already consumed the input.

**Why it happens:**
It's easy to read "goldmark is extensible" and assume porting a markdown-it plugin is a mechanical translation. It isn't — markdown-it plugins mutate a *stream* mid-flight; goldmark extensions participate in a *priority-ordered pipeline* with a hard last-pass-only stage. The two Marpit constructs most likely to trip this: the slide-splitter (must out-prioritize or coexist with `ThematicBreakParser`, and the spike already proved `---`-after-paragraph must be special-cased as setext regardless of parser) and `![bg]` (must out-prioritize the default image/link inline parser, mirroring the documented `TaskCheckBoxParser`-vs-`LinkParser` precedent).

**How to avoid:**
Design each Marpit port as an explicit priority decision up front: slide-splitting as a low-priority-number custom `BlockParser` (pre-empting `ThematicBreakParser`); `![bg ...]` as a low-priority-number custom `InlineParser` (pre-empting `LinkParser`, same pattern as the built-in task-checkbox extension); directive resolution (front-matter + HTML-comment carry-forward state) as an `ASTTransformer` since it needs the complete tree to resolve global/local/spot carry-forward correctly. Write this decision down per-plugin in the objective-1 design doc before coding, and unit-test each in isolation against a corpus of priority-collision cases (e.g., `![bg fit](x.png)` next to a real image, `---` mid-code-fence).

**Warning signs:**
A custom parser "never fires" in testing (priority too high/late); background-image syntax renders as a literal `<img>` instead of a directive (LinkParser won the race); slide-splitting works everywhere except immediately after specific block types (list items, blockquotes) where a different default parser claims the line first.

**Objective to address:**
Objective 1 (Marpit-in-Go) — this is the core engineering risk of that objective, larger than the base-parser-parity question the spike already answered.

---

### Pitfall 3: CSS theme-scoping engine — tdewolff/parse maturity gaps and Marpit's dual-syntax `@import-theme`

**What goes wrong:**
Marpit's PostCSS chain does five things Eden Press must port faithfully onto `tdewolff/parse/v2/css`: (1) parse `/* @theme name */` + `@size` + `@auto-scaling` **metadata comments** (not real at-rules — a custom convention, easy to miss since they look like ordinary CSS comments to a naive parser); (2) resolve **both** `@import` (standard CSS, theme-set relative) **and** `@import-theme` (Marpit's own at-rule, which exists *specifically* because Sass/SCSS preprocessors delete plain `@import` rules they don't recognize before Marpit ever sees them) — supporting only one breaks any theme authored for Sass compatibility; (3) scope selectors into the `.marpit` container while correctly handling the `:root` vs. `section` **specificity override** (`:root` in Marpit context means "this slide's section," but has *higher* CSS specificity than `section`, so a theme mixing both must resolve exactly as upstream does); (4) inject the "advanced background" pseudo-structure (a companion SVG background layer, not just a CSS `background-image`) whose selector/pseudo-element conventions are undocumented outside the Marpit source; (5) CSS nesting and `:is()`/`:where()` down-leveling — **tdewolff/parse's nesting support is recent and still evolving** (nested-ruleset parsing landed incrementally across v2.8.5–v2.8.8, with a changelog entry as late as Feb 2026 titled "fix declaration errors"), and `:is()`/`:where()` are lexed as generic `FunctionToken`s rather than given first-class selector-AST treatment — meaning **the selector rewriter must hand-walk function-token arguments itself**, there's no built-in "give me the compound selectors inside `:is()`" API the way `postcss-selector-parser` provides upstream.

**Why it happens:**
PROPOSAL §4.2 correctly identifies tdewolff/parse as the port target and correctly proposes *dropping* the down-level passes (targeting modern Chrome only) — but that decision only removes the need to *transform* `:is/:where/nesting` for older engines; it does **not** remove the need to *correctly parse selectors containing them* in order to scope-prefix each compound selector for the `.marpit` container. A theme author who nests rules or uses `:is()` inside a theme still needs correct selector-prefixing, and that's exactly the immature part of the dependency.

**How to avoid:**
Pin to the latest tdewolff/parse/v2 release and write a dedicated selector-prefixing test corpus covering: plain selectors, comma-lists, `:is(...)`/`:where(...)` with nested compound selectors, native CSS nesting (`&` references), and `:root` mixed with `section` in the same theme. Budget an explicit "selector rewriter" component as its own testable unit (not folded silently into the theme-loading code), since it's the one piece with no 1:1 upstream library equivalent (`postcss-selector-parser` has no direct Go port — this is a custom-write, not a port). Extract Marpit's own theme-related snapshot fixtures (ties to Pitfall 13) as the acceptance corpus for this component specifically, since `@import-theme`/advanced-background conventions are undocumented outside Marpit's source and tests.

**Warning signs:**
A theme using `:is()` or nesting scopes incorrectly (selector prefix applied to the wrong compound, or applied twice); `@import-theme` works for plain CSS but silently fails/is-a-no-op for themes originally authored in Sass; advanced-background rendering (the SVG layer) differs pixel-for-pixel from upstream on any theme beyond the 3 bundled ones.

**Objective to address:**
Objective 1 (Marpit-in-Go), specifically the theme-CSS scoper sub-component. Flag for **deeper per-objective research** before implementation — the selector-rewriter has no upstream Go analogue and needs its own design pass, not just "use tdewolff and go."

---

### Pitfall 4: Inline-SVG `foreignObject` rendering fragility in Chrome

**What goes wrong:**
Marpit's inline-SVG mode wraps every slide as `<svg viewBox="..."><foreignObject><section>...</section></foreignObject></svg>` for CSS-only pixel-perfect scaling — but `foreignObject` in Chromium/Blink has a documented history of being "quite broken in many ways" per a Chromium engineer's own admission. Three concrete failure classes: (1) **stacking-context changes** — since Chrome 61 (SVG2 spec adoption), `<svg>` and `<foreignObject>` became stacking contexts, breaking z-index/layering assumptions that worked pre-61; (2) **PDF-export-specific regressions** — a documented Chrome ≥108 regression affects SVG rendering specifically in *PDF output* generated from web pages (i.e., exactly Eden Press's `chromedp`/`Page.printToPDF` export path), distinct from on-screen rendering; (3) **silent non-rendering from missing attributes** — a `foreignObject` without explicit `width`/`height`, or inner HTML missing the `xmlns="http://www.w3.org/1999/xhtml"` namespace, silently fails to render rather than erroring, which is exactly the kind of defect a manual QA pass across dozens of slides would miss.

**Why it happens:**
This mode is inherited wholesale from Marpit as "the" pixel-perfect scaling mechanism, and PROPOSAL §2.1/§5.1 treats it as a straightforward container/renderer port. It isn't fully — it depends on a genuinely fragile browser primitive whose PDF-export behavior specifically has regressed across Chrome versions, which directly intersects Pitfall 11 (Chrome version pinning).

**How to avoid:**
Treat inline-SVG mode as a first-class conformance-corpus target (render + rasterize, not just render-to-HTML-string), since string-level HTML matching won't catch a `foreignObject` that renders empty in the actual browser. Always emit explicit `width`/`height` on generated `foreignObject` elements and the XHTML namespace on inner markup. Pin the Chrome version used for the PDF/PNG export test matrix and re-test on Chrome upgrades specifically for this feature (not just general regression testing) — the v108 regression is a documented precedent that a routine browser bump can silently break this exact code path.

**Warning signs:**
Slides render fine as raw HTML in a normal browser tab but come out blank/cropped specifically in PDF export; layering/z-index of background vs. content differs between preview and export; a Chrome version bump changes SVG-mode output with no code change on the Eden Press side.

**Objective to address:**
Objective 1 (container/SVG renderer) for correct emission; Objective 4 (CLI/chromedp PDF+PNG export) for the export-specific regression risk — cross-reference both.

---

### Pitfall 5: `latex2mathml` converter-hardening is real engineering, not "8 small patches"

**What goes wrong:**
The math spike (§11) is genuinely good news — 100% of defects are converter bugs, not engine limits — but PROPOSAL's own characterization ("small," "small table," "medium" for `aligned`) risks under-scoping objective 7. Concretely: big-operator limit stacking (`msubsup`→`munderover`) requires detecting *display-mode* context correctly (the same operator needs different markup inline vs. display); the `\binom`/`pmatrix` shared-fence bug and `\sqrt[n]` argument parsing are genuine parser-logic fixes inside a single-author third-party AST, not config flags; the `aligned`→`mtable` fix requires synthesizing `columnalign="right left"` semantics from `&`/`\\` tokens that the converter currently emits as literal text — this is writing a small sub-parser, not patching a constant; and the `mathvariant`-is-dead discovery (MathML Core dropped non-`normal` `mathvariant` — confirmed independently as a known MathML Core Chromium limitation) means the fix is a **letter+variant → Unicode math-alphanumeric codepoint mapping table** covering all of `\mathbb`/`\mathbf`/`\mathcal`/`\mathfrak`/`\mathscr` × `A-Za-z0-9`, which is a meaningfully sized lookup table to build and verify, not "a small table" in the trivial sense.

**Why it happens:**
The spike's framing ("bounded pure-Go fixes") is accurate in the *"not an engine limitation"* sense but easy to over-read as *"quick."* Since `latex2mathml` is single-author and will be forked/vendored (per §11's own honest caveat), every fix is also now Eden Press's ongoing maintenance burden, including any future upstream drift in the library.

**How to avoid:**
Budget objective 7's converter-hardening pass with real estimation (each of the 5 root-cause classes as a separately-sized task, not one lump), and decide up front: fork-and-patch vs. vendor-and-patch vs. upstream-contribute-first. Since forking creates permanent maintenance debt, attempt an upstream PR first for at least the codepoint-mapping and fence-sharing bugs (self-contained, reviewable fixes), falling back to a fork only where upstream is unresponsive. Build the corrected-MathML test cases from the spike (`corrected.png`, `variant.png` artifacts) into the permanent regression corpus so future `latex2mathml` version bumps can't silently regress a fixed case.

**Warning signs:**
Objective 7 estimated as "small" in the roadmap; no fork/vendor decision made explicit before work starts; corrected-math test cases exist only as spike scratch artifacts and aren't promoted into the permanent conformance corpus.

**Objective to address:**
Objective 7 (auto-fit resolution + math-fidelity tuning) — explicitly re-scope this objective's estimate upward from the spike's own framing.

---

### Pitfall 6: Headless-Chrome MATH font availability (tofu) and Chromium's structural MathML-Core gaps

**What goes wrong:**
Two distinct, compounding risks for the export path specifically (as opposed to the spike's dev-machine environment, which already had `STIXTwoMath.otf` present): (1) **Tofu risk** — a server/CI/container Chrome instance with no OpenType MATH-table font installed renders MathML as boxes ("tofu"), and the *specific font source matters*: even STIX Two Math served via Google Fonts' CSS has been reported to render incorrectly compared to the STIX-fonts-project's own WOFF2/OTF files, apparently because Google Fonts' subsetting/chunking strips MATH-table data. (2) **Structural Chromium gaps beyond font availability** — Chromium's MathML support is *MathML Core only* (a deliberately reduced subset restored to Chromium in 2023 after being pulled in 2013), and MathML Core specifically **omits `<mlabeledtr>`** (the element numbered/labeled equations need) and **most `<mtable>` attributes** (used to implement LaTeX alignment environments) — these are not bugs to fix, they are missing platform features, meaning `\tag`/numbered-equation support and complex multi-condition alignments are permanently out of reach for the native-MathML path in Chromium specifically (Firefox has stronger native support; irrelevant here since export is Chrome-only, but relevant if Eden Press's HTML output is ever viewed in Firefox/Safari too).

**Why it happens:**
The spike's environment already had the right font pre-installed and didn't probe numbered equations/complex alignment (its own honest caveat, §11) — so both gaps are real but currently *invisible* in local dev, and will only surface in a CI/production Chrome image or on math-heavy real-world decks.

**How to avoid:**
Bundle the STIX-fonts-project's own OTF/WOFF2 files (not a Google-Fonts-served copy) directly with any server-side Chrome image/container Eden Press ships or documents (OFL-licensed, bundling explicitly permitted, fonts must stay unmodified — ties to Pitfall 14's attribution requirements); verify MATH-table presence in CI as an explicit smoke test (render a known formula, screenshot/pixel-diff against a golden reference) rather than assuming "font installed" == "renders correctly." For `\tag`/numbered-equations and complex `aligned`-with-conditions, treat these as a **hard boundary**, not a converter bug to eventually fix — document them as "always route to the `go-latex/latex` SVG/PNG fallback" rather than attempting native MathML, and detect them at conversion time (e.g., presence of `\tag`, `\label`, or `align`-with-more-than-2-columns) to auto-select the fallback path per §9's open sub-decision on fallback threshold.

**Warning signs:**
Math renders correctly in local dev but as tofu in CI/Docker/production; numbered-equation or complex-alignment decks silently drop labels/misalign in the native path with no error; a font-availability regression is discovered only when a customer reports broken math in production.

**Objective to address:**
Objective 7 (math-fidelity tuning) for the fallback-routing logic; Objective 4/5 (CLI, Chrome discovery/bundling) for shipping/documenting the bundled font requirement in any reference deployment.

---

### Pitfall 7: Native editable PPTX (OOXML) — EMU units, real text boxes, and the group-shape coordinate trap

**What goes wrong:**
Marp CLI's own PPTX export is image-per-slide (a screenshot dropped into a slide shape) — the "editable PPTX" differentiator (§13, item 6) means building **real** `<p:sp>` text-box shapes with actual text runs, which is qualitatively harder than image placement: (1) **EMU is the only unit** — 914,400 EMU/inch, 360,000 EMU/cm, chosen specifically so inch/mm/pixel conversions stay integer — every position/size value in `<a:off>`/`<a:ext>` (and text-margin values like `marL`/`indent`) must be computed in this unit, and mixing up raw-EMU vs. unit-suffixed values (`2in` is valid in some contexts) is an easy off-by-360000 or off-by-914400 bug; (2) **group-shape coordinate trap** — child shapes inside a `<p:grpSp>` use a *separate* `chOff`/`chExt` coordinate space, not the parent's, which "trips many people up" per OOXML documentation — relevant the moment Eden Press groups elements (e.g., a background + text box as one logical unit per slide); (3) **placeholder vs. free-shape positioning** — if any content maps to a layout placeholder rather than a free-form text box, position/size can be *inherited from the slide layout* and silently ignored if also specified on the shape itself; (4) **size specified twice for any embedded image** (once for the drawing canvas extent, once for the embedded picture) — a real trap if backgrounds/images coexist with text boxes on the same slide.

**Why it happens:**
"Editable PPTX" sounds like "write XML instead of a screenshot," but OOXML's coordinate model, placeholder-inheritance model, and group-shape model are each independent sources of subtle positioning bugs that only manifest visually (PowerPoint silently accepts malformed-but-valid values and just renders them wrong) — there's no compiler error for "your text box is 1000x too small."

**How to avoid:**
Build a small, well-tested EMU conversion utility as its own unit (`Inches()`, `Points()`, `Pixels()` → EMU, and back) rather than inlining `* 914400` arithmetic at call sites. Write positioning conformance tests that open generated PPTX files in an automated PowerPoint/LibreOffice headless check (or at minimum, parse the generated XML back and assert EMU values against expected input dimensions) rather than only eyeballing rendered output. Decide explicitly, per slide element, whether it's a layout placeholder (inherits position) or a free shape (must always specify `xfrm`) — don't let this be implicit. If any content is grouped, write the `chOff`/`chExt` handling as a dedicated tested code path, not an extension of single-shape logic.

**Warning signs:**
Generated PPTX opens in PowerPoint with elements in the wrong position/size that "look close but not exact"; grouped elements are positioned correctly individually but wrong once grouped; positions drift specifically on non-default slide sizes (16:9 vs 4:3) — a sign EMU/unit conversion, not raw layout logic, is the bug.

**Objective to address:**
Objective 5 (PPTX + polish).

---

### Pitfall 8: The `unioffice` AGPLv3 licensing trap — no mature permissive Go alternative exists

**What goes wrong:**
`unioffice` (by UniDoc) is the dominant, most-discoverable pure-Go OOXML library and would be the obvious first search result for "create PPTX in Go" — but it is **dual-licensed AGPLv3 / commercial**, and the current upstream repository has moved further toward a **paid, license-key-gated commercial product** (metered API key required even for the "free" tier in newer versions). AGPLv3's copyleft terms would require Eden Press's *consuming* code to also be released under AGPL-compatible terms unless a commercial license is purchased — directly incompatible with Eden Press's MIT positioning and its "library-first, embeddable in any Go service" value proposition (a proprietary Eden-Biz/AOCore consumer embedding an AGPL dependency inherits AGPL's network-copyleft obligations). Research turned up **no mature MIT/Apache-licensed pure-Go alternative for PPTX *creation*** as of this research — community forks of unioffice (e.g. `5andr0/unioffice`) carry the identical AGPLv3 terms, they are not independent permissively-licensed implementations.

**Why it happens:**
PROPOSAL §5.1/§13 mentions "native OOXML PPTX (+ optional `soffice` editable)" without naming a library — an implementer defaulting to "just use unioffice, it's the standard Go OOXML lib" would silently introduce a license-incompatible dependency, discovered only at a legal/compliance review far later.

**How to avoid:**
Do **not** depend on `unioffice` (or any of its forks) at all. Since OOXML/PPTX is fundamentally "a ZIP of XML files following a documented schema" (confirmed by the officeopenxml.com reference used above, and mirrored by how `python-pptx` — MIT-licensed — implements the format from first principles rather than a commercial SDK), write Eden Press's own minimal OOXML writer: a Go `archive/zip` wrapper + Go `text/template` or `encoding/xml`-driven slide/shape/text-run templates, informed by the OOXML spec directly (officeopenxml.com and ECMA-376 are the authoritative free references) rather than any third-party SDK. This is more upfront work than "import a library" but is the only path consistent with Eden Press's MIT/embeddable positioning, and it's a bounded, well-documented file format — precisely the kind of "reimplement the format, don't reimplement TeX layout" case the project's own out-of-scope list (PROJECT.md) implicitly endorses.

**Warning signs:**
Any `go.mod` entry for `unioffice` or a fork of it; a PR that adds PPTX generation without an explicit licensing note in the objective's design doc; legal/compliance flags the dependency post-hoc (should be caught pre-hoc instead).

**Objective to address:**
Objective 5 (PPTX + polish) — this is a **blocking decision to make before objective 5's design doc is written**, not a mid-implementation discovery. Flag for deeper research at objective-5-planning time to confirm no new permissive library has emerged since this research date.

---

### Pitfall 9: Dart FFI cross-compile pain — Android NDK, iOS c-archive static linking, and a known Apple-Silicon toolchain bug

**What goes wrong:**
The Dart-binding plan (§7, confirmed viable) uses `dart:ffi` with platform-specific build modes: Android needs a `.so` built with `-buildmode=c-shared`, `CGO_ENABLED=1`, `GOOS=android`, and a `CC` pointing at the NDK's per-architecture clang (e.g. `aarch64-linux-android21-clang`) — requiring the Android NDK toolchain to be correctly installed and version-pinned (API-level suffix on the clang binary matters); iOS needs a **static** `.a` via `-buildmode=c-archive`, `GOOS=darwin`/`ios`, `CGO_ENABLED=1`, an SDK-specific clang wrapper, and `lipo` to combine per-architecture slices into a fat binary — a meaningfully different build pipeline from Android's, not a parameter swap. A **documented, confirmed Go toolchain bug** (`golang/go#47296`) causes `gomobile bind` to **panic on Apple Silicon (arm64) Macs when building for iOS if the Android NDK is merely installed** — the iOS build path incorrectly invokes Android-NDK-detection logic regardless of target, meaning a dev machine set up for *both* platforms (the common case) can break iOS builds purely by having Android tooling present.

**Why it happens:**
§7 correctly identifies this as "the standard Flutter FFI-plugin pattern," which is true in the sense that it's well-trodden — but "well-trodden" doesn't mean "friction-free." The two build modes are genuinely different compilation strategies (shared vs. static linking) with independent toolchain requirements, and the M1-NDK-panic bug is exactly the kind of environment-specific landmine that works fine on one dev's Intel Mac and breaks on another's M-series Mac.

**How to avoid:**
Document the exact toolchain versions (NDK version, Xcode version, Go version) that were verified working, and pin them in CI. If building on Apple Silicon and hitting the NDK-panic bug, the known workaround is isolating iOS builds from any Android-NDK-environment-variable presence (build in a container/environment without `ANDROID_HOME`/NDK set, or patch around the known issue) — don't assume "gomobile bind" will just work cross-platform on first try on an M-series machine with both SDKs installed. Treat Android and iOS as two independently-tested build pipelines in CI (separate jobs, separate toolchain containers), not one "mobile build" step.

**Warning signs:**
iOS builds panic specifically on Apple Silicon CI runners or dev machines that also have Android SDK/NDK installed, with no code change; a build that works on one team member's Intel Mac fails identically-configured on another's M-series Mac; `gomobile bind` errors reference NDK detection while targeting `ios`.

**Objective to address:**
Objective 6 (Dart binding).

---

### Pitfall 10: Go→WASM binary size/latency vs. TinyGo's reflection incompatibility — the "one conformance pass" promise is at risk

**What goes wrong:**
Standard Go-compiled WASM links the **entire Go runtime** into the binary regardless of what's used — even a single `fmt.Println` can balloon the output because `fmt` pulls in reflection, which pulls in type metadata, which pulls in a large slice of the runtime. Real-world reports put a minimal Go WASM binary at ~2MB gzipped-down-to-~500KB, consistent with PROPOSAL §7's own "~2MB+ gzipped" estimate — acceptable for an installed app, a real concern for web-embed latency (first-load cost on every visit unless cached). **TinyGo** is the standard mitigation (reported 4×–20× size reduction, e.g. one case from 2MB to 86KB) — but it is a **separate LLVM-based compiler**, not a flag on the standard Go toolchain, with a materially different runtime: **limited reflection support** (the single biggest source of breakage), **partial-to-absent support for `encoding/json`** (which relies heavily on reflection) and `net/http`, a **cooperative rather than preemptive goroutine scheduler**, and — critically for a multi-binding project — **its own incompatible `wasm_exec.js`** (TinyGo's JS glue code is a modified fork of Go's own `misc/wasm/wasm_exec.js` and the two are not interchangeable; using the wrong one produces import errors). Since Eden Press's core API explicitly emits a **structured JSON document model** (PROJECT.md: "structured document-model output... alongside HTML+CSS") and likely uses `gopkg.in/yaml.v3` (reflection-based) for front-matter parsing, both are exactly the kind of reflection-heavy code TinyGo struggles with.

**Why it happens:**
§7 frames the WASM-size tradeoff as "acceptable... evaluate for web-embed latency," treating it as a scale/perf decision to make later — but the *real* fork in the road is standard-Go-WASM vs. TinyGo-WASM, and that choice has functional consequences (does JSON-AST output even work under TinyGo?) that can't be deferred to a later "optimize if needed" pass without risking a rewrite of the JSON-emission and YAML-parsing paths.

**How to avoid:**
Decide **early** (objective 6 design, before writing WASM-specific code) whether the WASM target is standard Go (accept binary size, gain full stdlib/reflection compatibility) or TinyGo (accept a compatibility audit of every dependency — goldmark, yaml.v3, the JSON-AST emitter — against TinyGo's reflection/stdlib support before committing). If standard Go, budget Brotli compression (documented as outperforming gzip/Zopfli for this exact use case) and treat first-load latency as a measured, not assumed, acceptable cost. If TinyGo, pin its bundled `wasm_exec.js` version to the TinyGo compiler version exactly (mismatches produce import errors) and do **not** assume the same conformance-corpus test binary that runs under standard-Go CI will behave identically under TinyGo — reflection-dependent code paths (JSON marshaling, YAML front-matter) are the most likely to silently diverge, undermining the "one conformance pass across bindings" goal (§6/§7) specifically for the web target.

**Warning signs:**
JSON-AST output works in the native/CLI build but panics or silently produces wrong output only in the WASM build; front-matter YAML parsing breaks only under a TinyGo build; `wasm_exec.js` "undefined function" errors after a TinyGo version bump; first Flutter-Web load time isn't measured until late in the project.

**Objective to address:**
Objective 6 (Dart binding) — flag as needing a compatibility-audit sub-task if TinyGo is chosen, before broader WASM integration work begins.

---

### Pitfall 11: Headless-Chrome flakiness, determinism, and version-dependent regressions in CI

**What goes wrong:**
Multiple independent, compounding sources of non-determinism affect the `chromedp`-driven export path specifically in CI/container environments (as opposed to a developer's local machine, which is what the math spike ran on): (1) **shared-memory crashes** — Chrome's default `/dev/shm` allocation is frequently too small in containers, causing intermittent `BUS_ADRERR` crashes that look like random flakiness rather than a fixable root cause, unless `--disable-dev-shm-usage` and/or a larger `--shm-size` are explicitly set; (2) **sandbox privilege requirements** — Chrome's sandbox typically doesn't function in restricted CI containers without `--cap-add=SYS_ADMIN`, forcing a choice between `--no-sandbox` (common but reduces defense-in-depth, ties to Pitfall 12 for untrusted-Markdown scenarios) or a seccomp profile; (3) **running as root** — headless-shell running as root in a container has caused chromedp-specific startup crashes (`chromedp/chromedp#297`, a `WaitGroup` panic during allocator startup); (4) **version-dependent print-pipeline regressions** — a confirmed Chromium bug ties PDF generation timeouts specifically to a sandbox change in the LPAC print compositor landing around Chrome v125, affecting `--print-to-pdf` but *not* `--screenshot` — meaning a routine Chrome version bump can silently introduce PDF-specific (but not PNG-specific) flakiness with zero code change on Eden Press's side; (5) **cross-run contention** — concurrent headless instances sharing a user-data directory can interfere with each other, mitigated by a unique `--user-data-dir` per run.

**Why it happens:**
PROPOSAL §9 correctly flags "Chrome coupling... make browser discovery robust" but frames it primarily as a *discovery* problem (finding the binary) rather than a *runtime-stability* problem (keeping it from crashing/timing out once found) — these are two separate risk categories requiring two separate mitigations.

**How to avoid:**
Adopt the `chromedp/headless-shell` Docker image (purpose-built for the Go `chromedp` package) as the reference CI environment rather than a generic Chrome install, and bake in `--disable-dev-shm-usage`, a generous `--shm-size` (2G is a documented working value), non-root execution, and per-run unique `--user-data-dir` as default `chromedp` launch options — not opt-in flags a user has to discover. **Pin the Chrome/headless-shell version** used in CI and any reference deployment explicitly (rather than tracking `latest`), specifically because of the documented version-125-class PDF-only regression; re-validate PDF export (not just PNG) on any deliberate version bump. Separately from CI stability, implement Chrome discovery (Pitfall order: system Chrome → `CHROME_PATH` → bundled/downloaded pinned Chromium) as its own tested fallback chain, mirroring the precedent marp-cli itself established (chrome-launcher → Edge fallback → Puppeteer-bundled-binary-via-env-var pattern), per §9's open decision #3.

**Warning signs:**
Export tests pass locally but flake intermittently in CI with crash signatures rather than clear errors; PDF export specifically (not PNG) becomes flaky after an unrelated Chrome/base-image update; "works on my machine" reports that trace back to root-vs-non-root container execution.

**Objective to address:**
Objective 4 (CLI-in-Go, PDF+PNG via chromedp) for launch-flag defaults and stability; Objective 5 (Chrome discovery/bundling) for the version-pinning and fallback-chain design.

---

### Pitfall 12: bluemonday policy must match Marp's `xss` allow-list *semantically*, not just tag-for-tag

**What goes wrong:**
Matching Marp's HTML sanitization isn't just "list the same allowed tags in bluemonday" — the two libraries differ in **default behavior for disallowed content**, which changes what "matching" even means: bluemonday's default posture is to **strip** disallowed elements/attributes entirely, while the JS `xss` library's default is to **escape** disallowed tags into visible encoded text rather than removing them — a policy that's tag-list-identical between the two can still produce visibly different output (missing content vs. visible-as-text content) on the same malicious/unexpected input. Additional Marp-specific nuances that must be preserved exactly: (1) **HTML comments and `<style>` tags are *always* parsed for directives/theming regardless of the sanitization option** — meaning the directive/style-parsing code path is itself part of the trust boundary and must not accidentally become an injection vector (e.g., a crafted "directive" comment that's actually intended to smuggle unsanitized content past the allow-list check that runs on the rest of the document); (2) bluemonday has **no built-in CSS/style-attribute sanitization** — the `style` attribute must be excluded entirely (bluemonday's own docs advise against ever allowing it) even if upstream's `xss` policy technically permits some style properties, since matching that specific behavior would require hand-rolling CSS-value validation bluemonday doesn't provide; (3) the GFM disallowed-raw-HTML tag list (`<title>`, `<textarea>`, `<style>`, `<xmp>`, `<iframe>`, `<noembed>`, `<noframes>`, `<script>`, `<plaintext>`) is **not filtered by goldmark's GFM extension itself** (confirmed — this filtering is a GFM-spec "tagfilter" extension, not automatic parser behavior), so it must be explicitly implemented in the bluemonday policy layer, not assumed to come "for free" from using a GFM-compliant parser.

**Why it happens:**
"Match the allow-list" reads as a data problem (get the same list of tags/attributes) when it's actually a **behavioral-semantics** problem (strip vs. escape, what counts as "always parsed regardless of settings," what a GFM-compliant parser does vs. doesn't filter automatically) — the kind of gap that only shows up under adversarial/fuzz testing, not a manual side-by-side tag-list comparison.

**How to avoid:**
Write the security-parity test suite as **adversarial round-trip tests** (feed the same crafted malicious/edge-case Markdown through both Marp's actual `xss`-based sanitizer and the bluemonday policy, diff outputs) rather than a static tag-list comparison. Explicitly decide and document the strip-vs-escape behavior choice (bluemonday's native strip behavior is arguably *safer* by default than escape, but "safer and different" still isn't "matching," so document the deliberate deviation if kept). Explicitly implement the GFM tagfilter list in the bluemonday policy rather than relying on goldmark's GFM extension to provide it. Treat the "comments/style always parsed for directives" code path as untrusted input needing its own validation, independent of the general HTML sanitization pass — this is Eden Press's own security-sensitive logic, not something inherited safely from upstream.

**Warning signs:**
Security test suite is a static allow-list-comparison table rather than adversarial fuzzing; no explicit test for the GFM-disallowed-tag list; the directive-comment parser accepts arbitrary content without validation because "sanitization happens elsewhere in the pipeline."

**Objective to address:**
Objective 2 (Marp-Core-in-Go, sanitization sub-component) — and re-verified any time objective 5's "safe for untrusted content" differentiator (§13, item 5) is built out, since that's where the security bar is explicitly raised for multi-tenant SaaS use.

---

### Pitfall 13: Conformance-corpus construction — normalization false-negatives/positives and upstream-drift tracking

**What goes wrong:**
The spike's own DOM-normalization approach (parse both sides via `x/net/html`, compare structurally) is exactly right in principle but has two failure modes at scale: (1) **over-normalization masks real bugs** — the spike's own honest caveat notes the normalizer "intentionally ignores cosmetic whitespace/void-element/attr-order diffs," which is correct for `<br>` vs `<br/>` but would also mask a **genuinely whitespace-significant divergence** (rare, but CommonMark has some — e.g., inside `<pre>`/code spans, which the spike separately confirms are compared verbatim, correctly) if the normalization rules aren't scoped precisely to "provably cosmetic" contexts; a normalizer built too aggressively for convenience becomes a source of false-negative "matches" that hide real fidelity bugs. (2) **CSS-AST diffing has no equivalent proven approach yet** — the spike validated DOM normalization for HTML; the *CSS* side of the corpus (theme scoping, §6's `expected.css`) needs an analogous CSS-AST-level diff (ignoring property order, whitespace, but not ignoring actual selector/value differences) that hasn't been spike-tested at all — this is new ground, not an extension of proven work. (3) **Upstream drift tracking is a process problem, not just a technical one** — "re-import periodically" (§6) needs an actual trigger (a scheduled check against `marp-team/marpit`/`marp-core` releases, or a dependency-bot-style alert) or it silently becomes "re-import never," since there's no natural forcing function once v1 ships.

**Why it happens:**
The math and parser spikes proved out the *hardest technical unknowns* (does goldmark parse right, does MathML render right) — but the corpus *methodology* itself (how normalization is scoped, how CSS comparison works, how drift is tracked operationally) wasn't the spikes' target and remains comparatively under-specified relative to how load-bearing it is (§6: "the acceptance gate for every layer and every language binding").

**How to avoid:**
Scope HTML normalization rules explicitly and narrowly (an allow-list of "these specific differences are cosmetic," not a general "ignore whitespace" rule), and add negative test cases proving the normalizer *doesn't* mask known-real differences (e.g., verify it still flags a deliberately-broken `<pre>` content change). Build and validate the CSS-AST diff approach as its own spike-equivalent exercise early in objective 0 — don't assume it "just works like the HTML one." Set up an actual mechanical trigger for upstream-drift re-import (a scheduled CI job diffing against upstream's latest tag, filed as an issue/PR rather than a manual reminder) as part of objective 0's deliverable, not a "someday" process note.

**Warning signs:**
The corpus passes 100% but a manually-inspected export still shows a visible fidelity bug (normalizer too aggressive); no CSS-AST-diff tooling exists distinct from the HTML DOM-diff by the time theme-scoping work starts; six months post-v1, no one has re-run the corpus against a newer Marp release.

**Objective to address:**
Objective 0 (conformance corpus + runner) for normalization-rule rigor and CSS-diff tooling; ongoing process (not a single objective) for drift tracking, but the *mechanism* for it should ship as part of objective 0.

---

### Pitfall 14: Licensing/attribution completeness — per-file headers, OFL font notices, and "not affiliated" framing under actual reuse

**What goes wrong:**
§14's attribution plan is sound in principle but has concrete completeness risks once real assets are reused: (1) **per-file MIT headers must be preserved on the browser fit/polyfill script and all three themes verbatim** — easy to do correctly on initial import, easy to accidentally strip during a later refactor/reformat/minification pass if the header isn't treated as load-bearing content (e.g., a linter or minifier stripping "comments"); (2) **if STIX Two Math (or Latin Modern Math) is bundled** for the headless-Chrome MATH-font requirement (Pitfall 6), that's an **additional OFL-licensed asset requiring its own bundling notice** — distinct from the MIT-attribution plan §14 already covers for Marp assets, and easy to omit since it's a font dependency introduced to solve a rendering problem, not "a Marp asset" the existing NOTICE workflow already tracks; (3) **"not affiliated/endorsed" framing needs to be checked against trademark use, not just copyright** — using "Marp-compatible" in marketing copy/README is factually accurate and the proposal's framing is careful, but as the project gains visibility, any use of the Marp *logo*, or phrasing that could imply official partnership (vs. "inspired by / compatible with"), is a trademark question independent of the MIT copyright-attribution question already covered.

**Why it happens:**
§14 was written focused on the *known* verbatim-reused assets (3 themes + browser script) at proposal time — new verbatim/bundled assets introduced later for unrelated engineering reasons (a MATH font, for instance) don't automatically get routed through the same attribution checklist unless that checklist is a standing process, not a one-time NOTICE-file write.

**How to avoid:**
Add a NOTICE/CREDITS **process check**, not just a one-time file: any time a new third-party asset is vendored/bundled/embedded (fonts, scripts, themes), require an explicit NOTICE-file update as part of that PR's checklist — tie this to code review, not memory. Explicitly add the STIX Two Math (or whichever MATH font is bundled) OFL notice to NOTICE/CREDITS the moment font-bundling is implemented (Pitfall 6's objective), not deferred to a "licensing cleanup" pass later. Keep "not affiliated/endorsed" framing and any Marp-name usage under a light trademark-awareness review (no Marp logo use; "compatible with" not "official"; this is a documentation/marketing discipline, not a legal blocker) as the project's public-facing surface grows.

**Warning signs:**
A `go:embed`'d theme file or browser script has its header comment stripped by a formatter/minifier at some point after initial import; NOTICE/CREDITS references only the original 3 Marp-team assets months after a MATH font was bundled; marketing copy or a landing page uses Marp branding/logo rather than name-only textual reference.

**Objective to address:**
Cross-cutting — initial NOTICE/CREDITS setup belongs to objective 0/repo-setup; the *process* (checklist trigger on new vendored assets) should be established then and re-applied at objective 2 (themes/browser-script embed) and objective 7/wherever font-bundling is finalized (ties to Pitfall 6).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Skip the full CommonMark/GFM spec sweep, ship on the 32-case spike alone | Faster start on objective 1 | Silent parity gaps surface later as user bug reports instead of caught pre-emptively (Pitfall 1) | Never — the full sweep is cheap (an afternoon of harness work) relative to the risk it retires |
| Use `unioffice` "just to get PPTX working," clean up licensing later | PPTX ships faster in objective 5 | AGPL dependency baked into a codebase positioned as MIT/embeddable; ripping it out later touches every PPTX code path | Never — decide the OOXML-writer approach before any PPTX code is written (Pitfall 8) |
| Defer the CSS-AST diff tooling, reuse only HTML DOM-diff for the whole corpus | One diff engine to build, not two | Theme-scoping bugs (Pitfall 3) go uncaught since CSS output isn't actually being verified structurally | Only as a v0 stopgap for objective 0's first week; must be closed before objective 1's theme-scoper is considered "done" |
| Ship native-MathML-only, defer the SVG/PNG fallback decision | Simpler v1 math pipeline | Font-tofu and structural Chromium-MathML gaps (numbered eqns, complex alignment) become visible-in-production surprises instead of designed-around limitations | Acceptable only if fallback-trigger detection (Pitfall 6) ships in the same objective, even if the fallback renderer itself lands slightly later |
| Use standard Go (not TinyGo) for WASM "for now," decide on size optimization later | Full stdlib/reflection compatibility, no compatibility audit needed up front | If TinyGo is adopted later for size reasons, JSON/YAML/reflection-dependent code may need rewriting — a second migration, not a flag flip | Acceptable as the v1 default; make the TinyGo evaluation an explicit follow-up decision point, not silent technical debt |
| Bundle a font from Google Fonts' CDN/CSS rather than the font project's own release files | Simpler bundling (one `<link>` or CDN fetch) | Google Fonts' subsetting has been reported to strip MATH-table data, silently reintroducing the tofu problem the bundling was meant to solve | Never for the MATH font specifically — always use the STIX-fonts-project's own OTF/WOFF2 release |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| goldmark (parser base) | Treating extension-writing as "port the markdown-it plugin logic 1:1" | Design each plugin around goldmark's priority-ordered BlockParser/InlineParser/ASTTransformer pipeline explicitly (Pitfall 2); mirror the built-in TaskCheckBoxParser-vs-LinkParser precedent for any construct that must out-prioritize a default parser |
| tdewolff/parse/v2/css | Assuming it exposes a `postcss-selector-parser`-equivalent selector AST | It lexes `:is()`/`:where()`/nesting as generic function tokens; the selector-prefixing rewriter must be hand-written against `Values()`, not assumed to come from the library (Pitfall 3) |
| chromedp / headless Chrome | Launching with default flags and assuming CI behaves like local dev | Set `--disable-dev-shm-usage`, adequate `--shm-size`, non-root execution, unique `--user-data-dir`, and pin the Chrome/headless-shell version explicitly (Pitfall 11) |
| unioffice | Adding it as the PPTX dependency because it's the most-discoverable Go OOXML library | Don't depend on it at all (or any fork); write a minimal in-house OOXML writer against the ECMA-376/officeopenxml.com spec (Pitfall 8) |
| gomobile / dart:ffi | Assuming one build script covers both Android and iOS since "it's the standard pattern" | Treat Android (`c-shared` + NDK) and iOS (`c-archive` + `lipo`) as two independently-toolchained, independently-CI'd pipelines; watch for the confirmed M1+NDK-present iOS-build panic (Pitfall 9) |
| Go→WASM | Assuming `wasm_exec.js` is interchangeable between standard Go and TinyGo | Pin `wasm_exec.js` to the exact compiler (standard Go vs. TinyGo) and version producing the `.wasm` binary; they are not drop-in compatible (Pitfall 10) |
| bluemonday | Comparing tag/attribute allow-lists against Marp's `xss` config as if that's sufficient | Also match strip-vs-escape default behavior, the always-parsed-regardless-of-settings comment/style path, and the GFM disallowed-tag filter goldmark doesn't provide automatically (Pitfall 12) |
| STIX Two Math (or other MATH font) | Pulling the font from Google Fonts' CDN/CSS for convenience | Use the STIX-fonts-project's own OTF/WOFF2 release files directly; Google Fonts' subsetting has been reported to drop MATH-table data (Pitfall 6) |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Standard-Go WASM binary shipped uncompressed to a web client | Slow first paint on Flutter-Web load, especially on mobile networks | Serve Brotli-compressed `.wasm` (documented to outperform gzip/Zopfli for this exact case); measure real first-load latency instead of assuming "2MB is fine" | Immediately noticeable on any non-fast connection; becomes a real complaint once the web target has real users, not just dev-machine testing |
| Headless Chrome instances launched per-export without pooling/reuse | Export throughput bottlenecked by Chrome cold-start cost on every PDF/PNG/PPTX-screenshot request | Pool/reuse browser instances behind the `convert` package boundary (§5.2 already isolates Chrome dependency there) rather than spawning fresh per call | Becomes visible once Eden Press is embedded in a service handling concurrent export requests (e.g., Eden-Biz), not in single-shot CLI use |
| Full CommonMark+GFM spec sweep run synchronously on every CI push without caching | Conformance-corpus CI job becomes slow as the corpus grows (spec sweep + Marpit fixtures + math cases) | Parallelize per-case, cache parsed-AST fixtures, and gate on changed-file relevance where feasible | Once the corpus grows well beyond the 32-case spike toward the full 600+/672-example spec sweep (Pitfall 1) |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Assuming a GFM-compliant parser (goldmark) automatically filters disallowed raw-HTML tags | `<script>`, `<iframe>`, `<style>`-injection-adjacent tags pass through unfiltered in untrusted-content scenarios, since goldmark's GFM extension doesn't implement the tagfilter itself | Implement the GFM disallowed-tag list explicitly in the bluemonday policy layer, tested adversarially (Pitfall 12) |
| Running `--no-sandbox` Chrome as the default (not just a CI-only accommodation) when rendering **untrusted** user Markdown for export | Reduces Chrome's own defense-in-depth specifically in the scenario (multi-tenant SaaS, §13 differentiator 5) where it matters most | Reserve `--no-sandbox` for controlled CI/build environments; for untrusted-content export in a live service, prefer `--cap-add=SYS_ADMIN`-with-sandbox or a seccomp profile, and treat this as a deliberate security-review decision, not a default copy-pasted from CI setup (Pitfall 11) |
| Treating the "comments/style always parsed for directives regardless of sanitization settings" code path as safe because it's "just directives" | A crafted directive-comment or `<style>` block becomes an injection vector that bypasses the general HTML allow-list, since it's processed on a separate, always-on code path | Validate/sanitize the directive-parsing path independently, as its own trust boundary — don't assume general sanitization covers it (Pitfall 12) |
| Allowing the `style` HTML attribute because upstream's `xss` policy technically permits some CSS properties | bluemonday has no built-in CSS-value sanitization; naively mirroring upstream's attribute allow-list without the CSS-value filtering behind it reopens a CSS-injection vector upstream closed differently | Exclude the `style` attribute entirely in the bluemonday policy (bluemonday's own maintainers' guidance) even if this is a deliberate, documented deviation from exact upstream parity (Pitfall 12) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Native-MathML rendering silently degrades (tofu, missing labels, misaligned complex equations) with no visible warning to the document author | Author ships a deck/PDF with broken math and doesn't notice until a viewer reports it | Detect fallback-triggering constructs (`\tag`, `\label`, complex `aligned`) at render time and either auto-route to the SVG/PNG fallback or surface a clear build-time warning, rather than silent degradation (Pitfall 6) |
| PPTX export "works" (a file is produced) but text boxes are subtly mispositioned due to an EMU conversion bug | Deck opens in PowerPoint looking almost-but-not-quite right — the worst kind of bug because it doesn't fail loudly | Automated positional conformance tests against generated PPTX XML (Pitfall 7), not just "did a file get produced" |
| Theme authors hit an undocumented failure when their Sass-authored theme's `@import` gets silently dropped by the Sass compiler before Marpit/Eden Press ever sees it | Confusing "my theme just doesn't import" bug with no clear error, since the failure happens upstream of Eden Press entirely | Document `@import-theme` prominently as the required alternative for Sass-authored themes (mirroring Marpit's own docs), and consider a lint/warning if a theme file appears Sass-flavored but uses plain `@import` (Pitfall 3) |

## "Looks Done But Isn't" Checklist

- [ ] **Markdown parser parity:** Passing the 32-case spike corpus is not the same as passing the full CommonMark (600+) + GFM (672) spec sweep — verify the full sweep runs in objective 0, broken down by spec section, not just an aggregate percentage (Pitfall 1).
- [ ] **Theme-CSS scoping:** Rendering the 3 bundled themes correctly is not the same as correctly scoping an arbitrary user-authored theme using `:is()`/`:where()`/nesting/`@import-theme` — verify against a theme corpus beyond the 3 bundled ones (Pitfall 3).
- [ ] **Math rendering:** "Renders without crashing" (the spike's 20/20) is not the same as "renders correctly" — verify against the corrected-MathML regression cases *and* explicit fallback-routing for `\tag`/numbered-equations/complex-alignment, which the spike didn't exercise (Pitfall 5, 6).
- [ ] **PPTX export:** "A .pptx file is produced and opens" is not the same as "text boxes are positioned/sized correctly" — verify EMU-level positional accuracy programmatically, including grouped-shape `chOff`/`chExt` cases (Pitfall 7).
- [ ] **Dart binding:** "Builds successfully on my machine" is not the same as "builds on both Android and iOS in CI" — verify both build pipelines independently in CI, specifically on Apple-Silicon runners with Android tooling also present (the confirmed M1+NDK panic) (Pitfall 9).
- [ ] **Chrome-driven export:** "Passes locally" is not the same as "passes reliably in CI/containers" — verify under the actual CI container image with shm/sandbox/root constraints applied, not just a local dev machine (Pitfall 11).
- [ ] **Sanitization:** "Same tag allow-list as upstream" is not the same as "matches upstream's actual sanitization behavior" — verify via adversarial round-trip diffing, not a static list comparison (Pitfall 12).
- [ ] **Licensing:** "NOTICE file exists" is not the same as "every currently-bundled third-party asset (including ones added after the initial NOTICE write, like a MATH font) is credited" — verify NOTICE/CREDITS against the actual current set of vendored assets, not just the original three Marp assets (Pitfall 14).

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|-----------------|------------------|
| Residual CommonMark/GFM parity gap found post-ship | LOW–MEDIUM | Add the failing case to the conformance corpus as a regression test, implement a targeted custom parser/AST-transformer fix (per Pitfall 2's priority-based approach), re-run the full sweep |
| `unioffice`/AGPL dependency discovered already merged | HIGH | Requires excising the dependency from every PPTX code path and replacing with an in-house OOXML writer (Pitfall 8) — the exact reason to prevent this pre-emptively rather than recover from it |
| Theme-scoping bug found in a user-authored theme using modern CSS features | MEDIUM | Isolate as a selector-rewriter unit-test case, fix the hand-written `:is()`/`:where()`/nesting handling (Pitfall 3), add to the theme conformance corpus |
| MATH-font tofu discovered in a live deployment | LOW | Swap in the STIX-fonts-project's own OTF/WOFF2 files (not a CDN-served copy), verify via the render-and-pixel-diff smoke test (Pitfall 6) |
| CI flakiness traced to Chrome/shm/sandbox misconfiguration after months of "random" failures | LOW–MEDIUM | Adopt `chromedp/headless-shell` with the documented flag set (Pitfall 11), pin the Chrome version, and re-baseline any export-timing assumptions |
| A stripped MIT-header on a vendored asset (theme/browser script) discovered during a licensing audit | LOW | Restore the header from the original upstream source file (Marpit/Marp Core GitHub), add a lint rule or file-header-check to prevent recurrence (Pitfall 14) |
| Apple-Silicon iOS build panic discovered late in objective 6 | LOW | Known issue (`golang/go#47296`) — isolate iOS CI build environment from Android NDK presence, or patch around per the known workaround; not a novel debugging problem (Pitfall 9) |

## Pitfall-to-Objective Mapping

| Pitfall | Prevention Objective | Verification |
|---------|------------------------|-----------------|
| 1. Residual CommonMark/GFM corner cases | Objective 0 (corpus), Objective 1 (parser) | Full CommonMark (600+) + GFM (672) spec sweep passes, broken down per spec section |
| 2. goldmark AST-transformer vs. markdown-it mutable-ruler mismatch | Objective 1 (Marpit-in-Go) | Each Marpit plugin (slide-split, `![bg]`, directives) has an explicit, tested priority/mechanism decision; priority-collision test corpus passes |
| 3. CSS theme-scoping engine gaps (tdewolff nesting/`:is`/`:where`, `@import-theme`) | Objective 1 (theme-CSS scoper) | Selector-rewriter unit tests cover nesting/`:is()`/`:where()`/`:root`-vs-`section`; theme corpus beyond the 3 bundled themes passes |
| 4. Inline-SVG `foreignObject` fragility | Objective 1 (renderer), Objective 4 (chromedp export) | Rasterized (not just HTML-string) conformance tests for inline-SVG mode; Chrome-version-bump re-test specifically for PDF export |
| 5. `latex2mathml` converter-hardening scope | Objective 7 (math-fidelity tuning) | Corrected-MathML spike cases promoted into permanent regression corpus; fork/vendor/upstream-PR decision documented |
| 6. MATH-font tofu + Chromium MathML structural gaps | Objective 7 (fallback routing), Objective 4/5 (Chrome env) | CI smoke test renders known formula and pixel-diffs against golden reference; `\tag`/numbered-eq/complex-alignment auto-routes to SVG/PNG fallback |
| 7. PPTX OOXML EMU/positioning/group-shape complexity | Objective 5 (PPTX + polish) | Programmatic EMU-level positional assertions on generated PPTX XML, including grouped-shape cases |
| 8. `unioffice` AGPLv3 licensing trap | Objective 5 (PPTX + polish) — decide before design doc | No `unioffice`/fork dependency in `go.mod`; in-house OOXML writer documented against ECMA-376/officeopenxml.com |
| 9. Dart FFI cross-compile pain (NDK, c-archive, M1 panic) | Objective 6 (Dart binding) | Android and iOS build independently in CI on Apple-Silicon runners with Android tooling present |
| 10. Go→WASM size vs. TinyGo reflection incompatibility | Objective 6 (Dart binding) | Explicit standard-Go-vs-TinyGo decision documented; if TinyGo, JSON/YAML reflection-dependent paths compatibility-audited |
| 11. Headless-Chrome flakiness/determinism in CI | Objective 4 (CLI/chromedp), Objective 5 (Chrome discovery) | `chromedp/headless-shell`-based CI with shm/sandbox/root/user-data-dir defaults; Chrome version pinned and PDF-path re-tested on bumps |
| 12. bluemonday vs. Marp `xss` semantic parity | Objective 2 (sanitization) | Adversarial round-trip diff test suite; GFM disallowed-tag list explicitly implemented; directive/style path independently validated |
| 13. Conformance-corpus normalization/CSS-diff/drift-tracking | Objective 0 (corpus + runner) | Normalization allow-list documented with negative tests; CSS-AST diff tooling exists and is validated; scheduled upstream-drift-check job exists |
| 14. Licensing/attribution completeness (headers, font notices, trademark framing) | Objective 0 (setup), re-verified at Objective 2 and wherever font-bundling lands | NOTICE/CREDITS reflects every currently-vendored asset; header-preservation lint exists; no Marp logo/branding used in project marketing |

## Sources

- [goldmark GitHub repository](https://github.com/yuin/goldmark) — CommonMark 0.31.2 compliance, HTML-safety defaults
- [goldmark parser package docs](https://pkg.go.dev/github.com/yuin/goldmark/parser) — BlockParser/InlineParser/ASTTransformer priority model
- [goldmark parser.go source](https://github.com/yuin/goldmark/blob/master/parser/parser.go)
- [CommonMark Spec](https://spec.commonmark.org/) and [commonmark/commonmark-spec](https://github.com/commonmark/commonmark-spec) — official spec.txt, 600+ conformance examples, JSON extraction format
- [commonmark-spec npm package](https://www.npmjs.com/package/commonmark-spec) — pre-extracted golden-case JSON
- [github/cmark-gfm](https://github.com/github/cmark-gfm) — GFM reference implementation, 672 combined spec examples
- [CommonMark talk: Tab expansion with indented code](https://talk.commonmark.org/t/tab-expansion-with-indented-code/2442)
- [CommonMark talk: Tab-related issues](https://talk.commonmark.org/t/tab-related-issues/1831)
- [github/cmark-gfm issue #59: How to deal with tabs?](https://github.com/github/cmark-gfm/issues/59)
- [GFM Autolinks (extension) spec](https://gfm.xiniushu.com/Inlines/Autolinks%20extension.html)
- [micromark-extension-gfm-autolink-literal README](https://github.com/micromark/micromark-extension-gfm-autolink-literal/blob/main/readme.md) — documents GitHub's actual-vs-spec divergence
- [Marpit GitHub repository](https://github.com/marp-team/marpit) and [theme-css.md docs](https://github.com/marp-team/marpit/blob/main/docs/theme-css.md)
- [Marpit discussion #73: How can I use @import-theme?](https://github.com/orgs/marp-team/discussions/73)
- [Marpit issue #363: Unexpected behavior importing CSS in scoped style](https://github.com/marp-team/marpit/issues/363)
- [Marpit theme.js API source](https://marpit-api.marp.app/theme.js.html)
- [marp-core GitHub repository](https://github.com/marp-team/marp-core) and [PR #74: HTML sanitization refactor](https://github.com/marp-team/marp-core/pull/74)
- [tdewolff/parse GitHub repository](https://github.com/tdewolff/parse) and [css package docs](https://pkg.go.dev/github.com/tdewolff/parse/v2/css) — nesting support landed v2.8.5–v2.8.8
- [MathML Core / Igalia 2023 restoration coverage](https://www.gilesthomas.com/2025/02/mathml-fonts-on-chromium-based-browsers) — mlabeledtr/mtable gaps, stretchy-fence font issue
- [MathJax 4.0 MathML output docs](https://docs.mathjax.org/en/latest/output/mathml.html) — MathML Core limitations vs. MathJax/KaTeX
- [STIX fonts project GitHub](https://github.com/stipub/stixfonts) — OFL license, official OTF/WOFF2 source
- [google/fonts issue #3773: STIX Two Math rendering broken](https://github.com/google/fonts/issues/3773) — Google-Fonts-subsetting MATH-table concern
- [Plurimath: Licensing and availability of math fonts](https://www.plurimath.org/blog/2023-08-19-math-font-availability-licensing/)
- [officeopenxml.com: DrawingML Shapes — Location](http://officeopenxml.com/drwSp-location.php) and [Size](http://officeopenxml.com/drwSp-size.php) — EMU units, xfrm/off/ext
- [Points, inches and Emus (Lars Corneliussen)](https://startbigthinksmall.wordpress.com/2010/01/04/points-inches-and-emus-measuring-units-in-office-open-xml/)
- [unidoc/unioffice GitHub repository](https://github.com/unidoc/unioffice) — AGPLv3/commercial dual license, metered commercial model
- [unioffice pkg.go.dev](https://pkg.go.dev/github.com/Preciselyco/unioffice)
- [Go Wiki: Mobile](https://go.dev/wiki/Mobile) and [gomobile command docs](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile)
- [golang/go issue #47296: gomobile bind panics on M1 for iOS if NDK installed](https://github.com/golang/go/issues/47296)
- [Go Wiki: WebAssembly](https://go.dev/wiki/WebAssembly) and [TinyGo WebAssembly guide](https://tinygo.org/docs/guides/webassembly/wasm/)
- [Why Your Go Binary Is Too Fat for WebAssembly (TinyGo)](https://dev.to/alanwest/why-your-go-binary-is-too-fat-for-webassembly-and-how-tinygo-fixes-it-24l)
- [chromedp/headless-shell Docker image](https://hub.docker.com/r/chromedp/headless-shell/)
- [chromedp/chromedp issue #297: won't work out of the box running as root](https://github.com/chromedp/chromedp/issues/297)
- [Chromium issue 363225675: Chrome no longer able to generate PDF without --no-sandbox](https://issues.chromium.org/issues/363225675) — version-125-class print-pipeline regression
- [microcosm-cc/bluemonday GitHub repository](https://github.com/microcosm-cc/bluemonday) and [policy.go source](https://github.com/microcosm-cc/bluemonday/blob/main/policy.go)
- [marp-cli PR #292: Fallback to Microsoft Edge](https://github.com/marp-team/marp-cli/pull/292) and [PR #80: Docker Chrome crash workaround](https://github.com/marp-team/marp-cli/pull/80/files)
- [marp-cli issue #475: does not seem to find chromium](https://github.com/marp-team/marp-cli/issues/475)
- [Puppeteer installation docs](https://pptr.dev/guides/installation)
- [SVG foreignObject stacking context discussion (blink-dev)](https://groups.google.com/a/chromium.org/g/blink-dev/c/DHSUFGpZafc)
- [Google Chrome Community: SVG rendering issues in PDFs after Chrome 108](https://support.google.com/chrome/thread/217663507/svg-rendering-issues-in-pdfs-generated-from-web-pages-after-google-chrome-version-108)
- `PROPOSAL.md` §9, §11, §12, §14 (this repository) — the two completed feasibility spikes this research builds on
- `.planning/PROJECT.md` (this repository)

---
*Pitfalls research for: Eden Press (Go/Dart Marp-compatible document-generation framework)*
*Researched: 2026-07-20*
