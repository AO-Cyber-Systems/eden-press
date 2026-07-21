# Objective 5: convert/pdf + convert/png (chromedp raster export) - Research

**Researched:** 2026-07-21
**Domain:** Headless-Chrome-driven raster export (PDF via `Page.printToPDF`, PNG/JPEG via per-slide screenshot) from a pre-rendered `press.Output{HTML, CSS}` — the sole Chrome-touching boundary in Eden Press
**Confidence:** HIGH overall (chromedp v0.16.0 / cdproto/page / cdproto/emulation APIs verified directly against pkg.go.dev; Chrome-for-Testing/headless-shell discovery mechanics verified against official Google/chromedp repos). MEDIUM on font-loading-race and DPR-determinism specifics (community-reported, cross-checked 2+ sources, no single canonical doc). This document extends — does not repeat — `.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS}.md`, all dated 2026-07-20.

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| EXP-01 | PDF via `chromedp` `Page.printToPDF` (invoked inside an `ActionFunc`), fixed viewport, pinned timezone/locale, animations disabled → deterministic | §1 (ActionFunc/PrintToPDFParams pattern), §2 (determinism recipe), §5 (HTML loading), §7 (riskiest items — PDF-only Chrome-version regressions) |
| EXP-02 | PNG/JPEG per-slide via chromedp screenshots | §1 (Screenshot pattern), §6 (per-slide splitting against `chase/model.Document.Sections` + `profile.Profile.Container`) |
| EXP-04 | Robust headless-Chrome discovery (`--browser-path`/`CHROME_PATH` env → known paths → documented pinned-download) + MATH-font provisioning (bundle official STIX Two Math OTF) | §3 (discovery chain), §4 (CI-without-Chrome test design), §8 (STIX Two Math provisioning) |
</phase_requirements>

## Summary

Objective 5 is a pure **consumer** of Objective 3's `press.Output{HTML, CSS, Model, Meta, Comments}` — it never touches `chase/`, `press/` internals, or the goldmark AST. Everything chromedp needs is already decided by prior research (`STACK.md` §6, `ARCHITECTURE.md` Pattern 6, `PITFALLS.md` Pitfalls 4/6/11): chromedp v0.16.0, `Page.printToPDF` via `cdproto/page` inside an `ActionFunc` (not a first-class `chromedp.Action`), one long-lived browser process per worker with one child `chromedp.NewContext` (tab) per render. What this document adds is the **exact API surface** (verified against pkg.go.dev, 2026-07-21) needed to write the TRDs: `PrintToPDFParams`'s full builder-method set (no `WithFormat("A4")` — paper size is inches via `WithPaperWidth/Height` or, better, CSS `@page` + `WithPreferCSSPageSize(true)`), `cdproto/emulation`'s `SetTimezoneOverride`/`SetLocaleOverride`/`SetEmulatedMedia`/`SetDeviceMetricsOverride` signatures for the determinism recipe, and `page.SetDocumentContent` (not a data: URL, not a temp file) as the correct way to load `press.Output` into Chrome without size-limit truncation or local-file-access security posture.

The riskiest parts of this objective are **not** the chromedp API itself (that surface is small, well-documented, and stable) — they are (1) proving the Chrome-discovery fallback chain actually works in a container with zero system Chrome, which is invisible until the first clean CI/production deploy; (2) the two independently-documented, PDF-path-**only** Chrome regressions (`foreignObject`-in-PDF since Chrome 108, the print-pipeline/LPAC-sandbox regression around Chrome 125) that mean a routine Chrome version bump can silently break EXP-01 while EXP-02 (screenshots) keeps working; and (3) accepting that "deterministic" for a headless-Chrome pipeline means pixel-diff-under-threshold, not byte-identical — Chrome's own headless-dev team has publicly acknowledged PRNG-based non-determinism with no committed fix date.

**Primary recommendation:** Build `convert/chrome` (session + discovery) first, `convert/pdf` and `convert/png` as siblings against it (both testable against hand-written static HTML fixtures, decoupled from `press/` until an integration/capstone TRD), fold determinism (timezone/locale/animation/viewport/font) into both via a shared helper rather than duplicating it, and treat the CI-without-Chrome discovery test and the Chrome-version-pin/PDF-re-validation process as first-class TRD deliverables, not follow-up cleanup.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/chromedp/chromedp` | v0.16.0 (MIT) | Drives headless Chrome via CDP; `ExecAllocator`/`NewContext`/`Screenshot`/`EmulateViewport` | Already pinned in `STACK.md` §6; 2,179+ importers; no viable alternative sought (Chrome itself is the accepted external) |
| `github.com/chromedp/cdproto/page` | pinned to chromedp's own `go.mod` (no independent version) | `page.PrintToPDF()`, `page.SetDocumentContent()`, `page.GetFrameTree()` | The only way to reach `Page.printToPDF` — not a curated `chromedp.Action`, must be called via `ActionFunc` (this is chromedp's own documented pattern for any CDP method returning multiple values, per `STACK.md` §6) |
| `github.com/chromedp/cdproto/emulation` | same | `emulation.SetTimezoneOverride`, `SetLocaleOverride`, `SetEmulatedMedia`, `SetDeviceMetricsOverride` | The determinism primitives chromedp itself does not wrap as first-class actions (confirmed gap, `STACK.md` §6) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/chromedp/cdproto/cdp` | same | `cdp.NodeID`/`cdp.Node` types for `ByNodeID`/`ScreenshotNodes` | Only if per-slide screenshot moves off simple CSS-selector targeting (see §6 open question) |
| `go:embed` (stdlib) | Go 1.25 floor (already project-wide) | Embed the STIX Two Math font file into the `convert` binary for `@font-face` data-URI injection | §8's recommended default font-provisioning path |
| `net/http` + `encoding/json` (stdlib) | — | Fetch Chrome-for-Testing's `known-good-versions-with-downloads.json` / `last-known-good-versions-with-downloads.json` for an optional pinned-Chrome-download helper | §3's "documented pinned-download" tier — no ready-made Go client library exists (verified, see Sources) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|-----------|
| `page.SetDocumentContent` to load HTML | `data:text/html,...` URL via `chromedp.Navigate` | **Rejected as primary.** Documented truncation bug: users report Chrome silently trims the data URL after a length threshold, producing blank/incomplete pages — a real risk for a multi-hundred-slide deck's self-contained HTML+CSS. |
| `page.SetDocumentContent` to load HTML | Write `Output.HTML` to a temp file, navigate `file://` | **Rejected as primary.** Extra filesystem I/O + cleanup, and pulls in the exact "local file access" security posture Marp CLI itself gates behind an explicit `--allow-local-files` flag (a real upstream precedent, per `FEATURES.md` §C) — avoiding it entirely keeps `convert/` from ever needing a "trust this path" flag. Legitimate fallback only if a deck's assets are so large that in-memory `SetDocumentContent` becomes impractical. |
| Chrome-for-Testing JSON-API download helper (hand-written) | `chromedp/headless-shell` Docker image, version-pinned via tag | **Recommended as the primary CI/reproducibility mechanism** (simpler, zero new Go code) — reserve the JSON-API downloader for a nice-to-have `eden-press chrome install` CLI convenience (STACK.md §6's own recommendation), not a hard EXP-04 requirement. |
| CDP-level `Emulation.setEmulatedMedia(prefers-reduced-motion)` for animation-disable | Direct CSS-injection (`*, *::before, *::after { animation: none !important; transition: none !important; }`) appended to `Output.CSS` before feeding Chrome | **CSS injection recommended as primary** — `convert/` already fully controls the CSS being fed to Chrome (it's a self-contained document per `ARCHITECTURE.md` Pattern 6), so a guaranteed override is simpler and more portable than relying on the page's own CSS honoring `@media (prefers-reduced-motion: reduce)` (which Marp/Marpit theme CSS does not currently do). Keep `SetEmulatedMedia` too, as defense-in-depth, since it's a one-line CDP call. |

**Installation:** No new `go.mod` entries beyond what `STACK.md` already lists (`github.com/chromedp/chromedp v0.16.0` — not yet in `go.mod`, since Objective 3 deliberately excludes it; this objective is what adds it, scoped to `convert/...` only).

## Architecture Patterns

### Recommended Project Structure

```
convert/
├── chrome/               # Session mgmt + discovery — the ONLY package that
│                          #   resolves an ExecPath/allocator; everyone else
│                          #   consumes a ready context.
│   ├── discover.go        #   --browser-path / CHROME_PATH / known-path /
│                          #   pinned-download fallback chain (§3)
│   ├── allocator.go        #   ExecAllocator options: NoSandbox, UserDataDir,
│                          #   Flag(...), one-browser-many-tabs pattern
│   └── determinism.go      #   shared helper: EmulateViewport + SetTimezoneOverride
│                          #   + SetLocaleOverride + SetEmulatedMedia + CSS-injection
│                          #   animation-disable + font @font-face injection (§2, §8)
├── pdf/
│   └── pdf.go               #   ToPDF(out press.Output, opts) — ActionFunc wrapping
│                          #   page.PrintToPDF, CSS @page-size injection (§1, §7)
├── png/
│   └── png.go               #   ToImages(out press.Output, opts) — per-Section
│                          #   chromedp.Screenshot loop (§1, §6)
└── convert.go              # shared Options/Format types, doc.go boundary note
```

`convert/` (and only `convert/`) imports `chromedp`/`cdproto`; `press/`, `chase/`, `profiles/` never import `convert/` (one-directional, per `ARCHITECTURE.md`'s Internal Boundaries table). `scripts/check-no-chromedp.sh`'s `TREES` array is `("./press/..." "./chase/..." "./profiles/...")` — it does **not** scan `convert/`, so adding chromedp under `convert/` requires **zero changes to the gate script**. The gate only breaks if something under `press/chase/profiles` imports `convert/` (which would also be a Go import cycle, since `convert/pdf` and `convert/png` import `press` for the `Output` type) or directly imports chromedp itself. Confirmed by reading `scripts/check-no-chromedp.sh` and `Makefile`'s `check-no-chromedp` target directly (both already exist, wired into `.github/workflows/ci.yml`).

### Pattern 1: `Page.printToPDF` via `ActionFunc` + `cdproto/page` builder

**What:** `page.PrintToPDF()` returns a `*page.PrintToPDFParams` with a chained `With*` builder API; `.Do(ctx)` returns `(data []byte, stream io.StreamHandle, err error)`. Because `Do` returns multiple values, it is not consumable as a plain `chromedp.Action` in `chromedp.Run(...)` — it must be wrapped in `chromedp.ActionFunc(func(ctx context.Context) error { data, _, err := page.PrintToPDF().With...().Do(ctx); ...; return err })`. This is the exact mechanism `STACK.md` §6 already flagged as "not a gap, just how chromedp is meant to be used beyond its curated helper set."

**Verified full `PrintToPDFParams` builder-method set** (source: pkg.go.dev, 2026-07-21): `WithLandscape(bool)`, `WithDisplayHeaderFooter(bool)`, `WithPrintBackground(bool)`, `WithScale(float64)`, `WithPaperWidth(float64)`/`WithPaperHeight(float64)` (inches), `WithMarginTop/Bottom/Left/Right(float64)` (inches), `WithPageRanges(string)`, `WithHeaderTemplate(string)`/`WithFooterTemplate(string)` (only take effect when `WithDisplayHeaderFooter(true)`; support the special classes `pageNumber`/`totalPages`/`date`/`title`/`url`), `WithPreferCSSPageSize(bool)`, `WithTransferMode(page.PrintToPDFTransferMode)` (`ReturnAsBase64` default vs `ReturnAsStream`), `WithGenerateTaggedPDF(bool)` (v2 EXP2-01 territory, not v1), `WithGenerateDocumentOutline(bool)`.

**No `WithFormat("A4")` convenience exists.** For a Marp-style fixed-size deck (16:9 = 1280×720px, 4:3 = 960×720px per `profiles/slides/slides.go`'s `Sizes()`), paper size should be driven by CSS, not raw inches: emit a `@page { size: <width-in>in <height-in>in; margin: 0; }` rule (px→in at 96px/in — 1280×720px = 13.333in×7.5in) into the CSS `convert/pdf` composes around `Output.CSS`, then call `WithPreferCSSPageSize(true)` with all four margins at `0`. This is a synthesized recommendation (MEDIUM confidence — not sourced from a specific Marp-CLI internals citation, but a direct, well-established application of `PreferCSSPageSize`'s documented CDP semantics, consistent with how Puppeteer's identically-named `preferCSSPageSize` option is universally used for slide/fixed-size exports) — flag for TRD-time confirmation against a rendered PDF's actual page dimensions.

**`WithPrintBackground(true)` is required**, not optional — Marp theme CSS backgrounds/colors are real content, and Chrome's print pipeline omits background graphics by default (a general Chrome/print-CSS default, not chromedp-specific).

### Pattern 2: Determinism recipe (fixed viewport, timezone/locale, animations, DPR)

Ordered application, all before content is loaded (set once per tab, persists across navigation):

1. **Viewport** — `chromedp.EmulateViewport(width, height int64, opts ...EmulateViewportOption)`, with `chromedp.EmulateScale(1.0)` pinned explicitly (not left to host DPI). Width/height should come from the resolved `profile.SizeTable` entry (`theme.Size{WidthPx, HeightPx}` — 1280×720 default). This underlies `Emulation.setDeviceMetricsOverride` (also directly callable as `emulation.SetDeviceMetricsOverride(width, height int64, deviceScaleFactor float64, mobile bool)` with `.WithScale(float64)` if finer control is needed than the chromedp convenience wrapper gives).
2. **Timezone** — `emulation.SetTimezoneOverride(timezoneID string).Do(ctx)` (single required string arg, e.g. `"UTC"`; no builder needed). Pin to a project-fixed zone (UTC recommended) rather than leaving it host-dependent.
3. **Locale** — `emulation.SetLocaleOverride().WithLocale(locale string).Do(ctx)` (e.g. `"en-US"`) — affects any `Intl`-driven formatting Chrome's own header/footer print templates use (`<span class="date">`), and any future custom-directive JS/CSS counters.
4. **Animations/transitions disabled** — primary: append `*, *::before, *::after { animation: none !important; transition: none !important; scroll-behavior: auto !important; }` to the composed CSS `convert/` feeds Chrome (see Pattern 1's alternatives-considered rationale). Secondary/defense-in-depth: `emulation.SetEmulatedMedia().WithMedia("screen").WithFeatures([]*emulation.MediaFeature{{Name: "prefers-reduced-motion", Value: "reduce"}}).Do(ctx)`.
5. **Font-load race (NEW finding, not in existing `PITFALLS.md`)** — if STIX Two Math (or any font) is injected via a `@font-face` **data-URI** (§8's recommended default), Chrome must still parse/apply it before layout is final — a screenshot/PDF captured immediately after `SetDocumentContent` risks a FOUT-style race distinct from Pitfall 6's "font not installed at all" tofu. No chromedp built-in exists for this; the documented general pattern (cross-referenced against Puppeteer's own `document.fonts.ready` idiom, which chromedp's `Evaluate`/polling actions can replicate) is to explicitly wait on the browser's `document.fonts.ready` promise resolving before capture. Flag as MEDIUM confidence — a real, logically-necessary risk given data-URI font injection, but not independently confirmed against a chromedp-specific worked example this session.
6. **`--force-device-scale-factor` flag** — set as a redundant `ExecAllocator` `Flag("force-device-scale-factor", "1")` alongside the CDP-level `EmulateScale(1.0)`. MEDIUM confidence, community-reported (not official chromedp docs): a documented `window.devicePixelRatio` floating-point artifact (`1.0000000149011612`) and DPR-doubling-on-retina-hosts issue has been independently reported multiple times; pinning both the launch flag and the CDP override is the safer default given the reports don't agree on which layer authoritatively wins.
7. **Font-render/rasterization stability** — pin the exact Chrome/`headless-shell` build **and container image** (extends Pitfall 11's existing recommendation) since FreeType/subpixel-hinting and font-package versions differ by OS/distro; do not rely on host system fonts for anything Chrome renders — bundle fonts explicitly (§8) so glyph shapes are identical across dev/CI/prod. Chrome's own headless-dev team has publicly acknowledged **PRNG-based non-determinism with no committed deterministic-mode ship date** — set the objective's acceptance bar as pixel-diff-under-threshold, not byte-identical output (a scope correction relative to how "byte-reproducible output" reads in `ARCHITECTURE.md`/`STACK.md` for this specific export path — the pure-Go `press/` render path can still target byte-identical; the Chrome-driven raster path cannot, and this should be stated explicitly in the objective's own success criteria).

### Pattern 3: Chrome discovery fallback chain

**Order** (matches `STACK.md` §6's recommendation, `PITFALLS.md` Pitfall 11's marp-cli precedent, and Lighthouse/chrome-launcher's `CHROME_PATH` convention users already expect):

1. **`--browser-path` CLI flag** (or equivalent `Options` field on `convert/`'s public API) — explicit, highest-precedence override, passed straight to `chromedp.ExecPath(path)`.
2. **`CHROME_PATH` env var** — **important correction:** `CHROME_PATH` is **not a chromedp convention** at all (verified: it originates with Lighthouse/`chrome-launcher`). chromedp has no built-in env-var lookup. `convert/chrome`'s discovery code must read `os.Getenv("CHROME_PATH")` itself and pass the result to `chromedp.ExecPath` — this is custom glue eden-press owns, chosen specifically because it matches a convention users of Chrome-automation tooling already know (marp-cli/Lighthouse-adjacent), not because chromedp provides it.
3. **Known install paths / auto-PATH search** — when no `ExecPath` option is given, chromedp's own `ExecAllocator` falls back to auto-discovering a Chrome-shaped binary; the historical name list (from `chromedp/chromedp/runner`, a separate, not-required-for-this subpackage) is `google-chrome`, `chromium-browser`, `chromium`, `google-chrome-beta`, `google-chrome-unstable`, with `/usr/bin/google-chrome` as the terminal default path. Treat this tier as "let chromedp try its own default," not something eden-press re-implements.
4. **Documented pinned-download** — no ready-made Go library performs this (confirmed by this session's search — the two real options are (a) pin a `chromedp/headless-shell:<version-tag>` Docker image, the simplest and recommended default for CI/reproducibility, or (b) fetch a specific version from Chrome for Testing's JSON API — `known-good-versions-with-downloads.json` (any historical version, useful for bisecting) or `last-known-good-versions-with-downloads.json` (latest per channel) at `googlechromelabs.github.io/chrome-for-testing/` — extract the platform zip, note the CfT Linux archive is **browser-only** and the executable inside is named `chrome`, not `google-chrome`. Google has stated the JSON schema/URL format is intended to stay stable long-term. Recommend documenting (b) as the mechanism behind an optional `eden-press chrome install` convenience command (per `STACK.md` §6) — EXP-04 only requires this tier be **documented**, not necessarily automated in v1.

### Anti-Patterns to Avoid

- **Spawning a fresh Chrome process per export request.** Confirmed cost: ~100s of ms + ~150–300MB RSS per instance (`ARCHITECTURE.md` Scaling Considerations). Pool one `NewExecAllocator`/browser per worker, one `NewContext` (tab) per render — `chromedp.NewContext(parentCtx)` where `parentCtx` already has a live `Browser` creates a new tab on the same browser rather than a new process; confirmed via `chromedp.FromContext(ctx).Browser`/`.Target` equality checks.
- **Using a data: URL for `press.Output.HTML`.** Documented truncation bug for large HTML strings — silently produces blank/incomplete pages, exactly the kind of failure that would show up only on a large real-world deck, not a small test fixture.
- **Assuming `CHROME_PATH` is a chromedp built-in.** It is not; must be read and wired manually (Pattern 3, tier 2).
- **Trusting `--font-render-hinting`-class flags as the primary font-determinism fix.** These flags are deprecated/removed across recent Chrome versions (uncertain exact removal version — LOW confidence, flag for TRD-time verification against the actually-pinned Chrome build); pin the container image and bundle fonts explicitly instead (Pattern 2 item 7, §8).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| CDP protocol bindings (message framing, domain/method dispatch) | A custom CDP client over the raw WebSocket protocol | `chromedp` + `cdproto` (already the pinned stack) | Exactly the problem chromedp exists to solve; no reason to drop to raw CDP given `cdproto/page`/`cdproto/emulation` already expose everything this objective needs |
| Chrome-version-pinned, reproducible download infrastructure | A bespoke Chrome-build mirror/cache service | `chromedp/headless-shell` Docker image (tag-pinned) as primary; Chrome for Testing's official JSON API as the documented secondary/CLI-convenience mechanism | Google explicitly built Chrome for Testing to solve exactly this reproducibility problem; chromedp's own maintainers publish `headless-shell` specifically "for use with the Go chromedp package" |
| "Wait for the page to be visually settled" logic | A custom sleep/poll-DOM-mutation heuristic | `document.fonts.ready`-style explicit wait (Pattern 2 item 5) + chromedp's own `WaitReady`/`WaitVisible` query actions for DOM-existence waits | Sleeping a fixed duration is both slow (worst-case padding) and unreliable (best-case race) — an explicit promise/condition wait is the standard idiom across Puppeteer/Playwright/chromedp-adjacent tooling |
| Math-font subsetting/hosting | Pulling STIX Two Math from a font CDN (Google Fonts or similar) | Vendor the font file directly from `stipub/stixfonts`'s own GitHub releases, `go:embed`, inject as a data-URI `@font-face` | Google Fonts' subsetting/chunking has been independently reported (twice, across two research sessions) to strip the OpenType MATH table, silently reintroducing tofu — this is a repeat-confirmed pitfall, not a one-off report |

**Key insight:** every piece of this objective that looks like "write a small helper" (CDP client, Chrome downloader, visual-settle waiter) already has an actively-maintained, purpose-built answer; the actual engineering here is **composition and determinism discipline** (ordering the CDP calls correctly, pinning every source of variance), not new protocol/infrastructure code.

## Common Pitfalls

*(Extends `PITFALLS.md` Pitfalls 4, 6, 11 — cited inline; does not repeat their full detail.)*

### Pitfall A: PDF-path-only Chrome-version regressions (extends Pitfall 4 + Pitfall 11)
**What goes wrong:** Two independently documented Chrome regressions affect `Page.printToPDF` specifically, **not** screenshot capture: (1) a Chrome ≥108 regression in SVG rendering specifically for PDF output generated from web pages (directly relevant since Marpit's inline-SVG `foreignObject` mode is a first-class Eden Press feature per `ARCHITECTURE.md`), and (2) a print-pipeline/LPAC-sandbox-change regression tied to a Chrome ~v125-class update that causes PDF generation timeouts while `--screenshot` is unaffected.
**Why it happens:** PDF generation and screenshot capture use structurally different Chrome subsystems (print compositor vs. paint/compositor pipeline); a Chrome bump can regress one without the other.
**How to avoid:** Pin the exact Chrome/`headless-shell` version used in CI and any reference deployment; treat any deliberate version bump as requiring **full PDF-path re-validation** (render the inline-SVG conformance fixtures + a representative deck, diff against golden output) as an explicit, blocking step — never assume a screenshot-path pass implies a PDF-path pass.
**Objective to address:** Objective 5 (this one) for the re-validation *process*; cross-reference `PITFALLS.md` Pitfalls 4 and 11 for the underlying regressions.

### Pitfall B: Font-load race with data-URI `@font-face` injection (new this session — see Pattern 2 item 5)
**What goes wrong:** A screenshot/PDF captured immediately after `SetDocumentContent` may fire before Chrome finishes parsing/applying a data-URI-embedded font, producing fallback-font (not tofu-box, but wrong-glyph) rendering intermittently.
**How to avoid:** Explicit `document.fonts.ready` wait before every capture, not a fixed sleep.
**Confidence:** MEDIUM — logically necessary given the data-URI font strategy, not independently confirmed against a chromedp-specific worked example.

### Pitfall C: "Byte-reproducible" is the wrong acceptance bar for the Chrome-driven path
**What goes wrong:** `ARCHITECTURE.md`/`STACK.md` frame "deterministic/reproducible output" as a project-wide differentiator; taken literally for `convert/pdf`/`convert/png`, this promises something Chrome itself cannot currently guarantee (publicly acknowledged PRNG-based non-determinism, no committed fix date).
**How to avoid:** State the objective's own acceptance criterion as pixel-diff-under-threshold (or PDF-structural-diff, e.g. same page count/dimensions/text-layer content, not byte-for-byte) explicitly in the TRD, distinct from the pure-Go `press/` path's legitimate byte-identical claim.
**Objective to address:** Objective 5 — a scope-precision fix, not a new engineering task.

## Code Examples

No runnable code is included per this research pass's scope (implementation-ready API surface only — signatures/params below feed the planner/TRD writer directly, not a working program).

| Operation | Verified signature |
|-----------|----------------------|
| PDF export entry point | `page.PrintToPDF() *page.PrintToPDFParams`, chained `With*` builders, `.Do(ctx context.Context) (data []byte, stream io.StreamHandle, err error)` — called inside `chromedp.ActionFunc` |
| Load HTML without navigation/data-URL | `page.GetFrameTree().Do(ctx) (*page.FrameTree, error)` → `page.SetDocumentContent(frameID cdp.FrameID, html string).Do(ctx) error` — requires a prior `chromedp.Navigate("about:blank")` to establish a top-level frame |
| Per-element screenshot | `chromedp.Screenshot(sel any, picbuf *[]byte, opts ...QueryOption) QueryAction` (default `ByQuery` = CSS `querySelector`) |
| Fixed viewport | `chromedp.EmulateViewport(width, height int64, opts ...EmulateViewportOption) EmulateAction`, with `chromedp.EmulateScale(scale float64)` |
| Timezone pin | `emulation.SetTimezoneOverride(timezoneID string).Do(ctx) error` |
| Locale pin | `emulation.SetLocaleOverride().WithLocale(locale string).Do(ctx) error` |
| Reduced-motion (defense-in-depth) | `emulation.SetEmulatedMedia().WithMedia(media string).WithFeatures(features []*emulation.MediaFeature).Do(ctx) error` |
| Browser+tab pooling | `chromedp.NewExecAllocator(parent context.Context, opts ...ExecAllocatorOption) (context.Context, context.CancelFunc)`; `chromedp.NewContext(parent context.Context, opts ...ContextOption) (context.Context, context.CancelFunc)` — child inherits parent's already-allocated `Browser`, gets a new `Target` (tab) |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| `CHROME_PATH` env var as a universal automation-tooling convention | Tool-specific: Lighthouse/chrome-launcher use `CHROME_PATH`; chromedp has no env convention at all (must be wired manually); `chromote` (R) uses `CHROMOTE_CHROME` | Long-standing fragmentation, reconfirmed this session | eden-press's own discovery code, not chromedp, owns the `CHROME_PATH` convention decision |
| Puppeteer-bundled Chromium as the automation-standard binary | Chrome for Testing (official Google channel, since Chrome 115) as the version-locked, dedicated-to-automation distribution | Chrome 115+ | The correct source for pinned/reproducible downloads; superseded ad-hoc "download whatever Puppeteer bundles" patterns |
| `--font-render-hinting=none` / similar flags as the standard font-determinism fix | Uncertain current status across recent Chrome — flagged LOW confidence, verify at TRD time against the pinned build | Ongoing Chrome flag churn | Do not hard-code assumed-working flags without testing against the actually-pinned Chrome/`headless-shell` version |

**Deprecated/outdated:**
- Data-URL HTML loading for anything beyond trivial fixture sizes — superseded by `page.SetDocumentContent`.
- Per-request fresh Chrome process spawning — superseded by the pool/reuse pattern (already documented in `ARCHITECTURE.md` Pattern 6, reconfirmed here).

## Open Questions

1. **Does `press.Output` guarantee self-contained assets (no relative-path `![bg]` images) today?**
   - What we know: `ARCHITECTURE.md` Pattern 6 states the export paths "consume `Output.HTML`+`Output.CSS` (a self-contained document: no relative asset URLs Chrome would need to fetch)" as a design intent.
   - What's unclear: whether Objective 3's actual implementation enforces/verifies this (e.g., does `press/` inline background images as data URIs, or does that remain the deck author's/`convert/`'s responsibility?).
   - Recommendation: confirm against `press/` source (`press/press.go`, background-image handling) before finalizing `convert/`'s "no network fetch" contract; if not guaranteed, `convert/` needs its own asset-inlining pre-pass, which would be new scope for this objective's TRDs.

2. **Does `ScreenshotNodes` capture one stitched image per call, or is it meant for one-node-at-a-time use?**
   - What we know: verified signature is `chromedp.ScreenshotNodes(nodes []*cdp.Node, scale float64, picbuf *[]byte) Action` — takes a slice of nodes into one `picbuf`.
   - What's unclear: whether this is for capturing multiple disjoint regions into one composite image (unlikely to be what "N separate PNG files, one per slide" needs) versus a batched-but-separable output.
   - Recommendation: default to the simpler, individually-verified `chromedp.Screenshot(sel, &buf, chromedp.ByQuery)` in a loop over `Model.Sections` (one call, one file, one slide) unless a TRD-time spike shows a meaningful throughput win from `ScreenshotNodes`.

3. **Exact selector for per-slide capture under inline-SVG mode.**
   - What we know: `profiles/slides.Container(inlineSVG)` returns `"div.marpit > svg > foreignObject"` (SVG mode) or `"div.marpit"` (plain mode); the actual per-slide element is `<section>` (per `UnitElement()`), repeated once per `Model.Sections` entry, in document order.
   - What's unclear: whether Chrome's element-screenshot bounding-box resolution behaves identically for a `<section>` nested inside `svg>foreignObject` vs. a plain `div>section` — `PITFALLS.md` Pitfall 4 already flags `foreignObject` as a documented Chrome fragility surface, specifically worse in PDF export; its screenshot-path behavior is not separately confirmed.
   - Recommendation: budget an explicit inline-SVG-mode screenshot smoke test in the TRD, not just a PDF-mode one.

## Sources

### Primary (HIGH confidence)
- pkg.go.dev/github.com/chromedp/cdproto/page — `PrintToPDFParams` full builder-method list, `Do` return signature, `SetDocumentContent`/`GetFrameTree` signatures (fetched 2026-07-21)
- pkg.go.dev/github.com/chromedp/cdproto/emulation — `SetTimezoneOverride`, `SetLocaleOverride`, `SetEmulatedMedia`, `SetDeviceMetricsOverride` signatures (fetched 2026-07-21)
- pkg.go.dev/github.com/chromedp/chromedp — `Screenshot`/`FullScreenshot`/`ScreenshotScale`/`ScreenshotNodes`/`CaptureScreenshot` signatures, `EmulateViewport`+options, `DefaultExecAllocatorOptions` flag list, `ExecPath`/`NoSandbox`/`UserDataDir`/`Flag` options, `NewExecAllocator`/`NewContext` pooling semantics (fetched 2026-07-21)
- `github.com/chromedp/docker-headless-shell` (GitHub) — image purpose, version-tag pinning, non-root/`--user nobody`+seccomp pattern, `--init` for zombie-process reaping
- `github.com/GoogleChromeLabs/chrome-for-testing` + `googlechromelabs.github.io/chrome-for-testing/` — JSON API endpoint list (`known-good-versions[-with-downloads].json`, `last-known-good-versions[-with-downloads].json`), URL-format stability commitment, CfT Linux archive is browser-only with executable named `chrome`
- `github.com/stipub/stixfonts` (GitHub, official STIX Fonts project repo) + its `OFL.txt` — OFL-1.1 license, "TM Math" reserved font name, OTF-native format, WOFF2 web-font builds available in the same repo
- This repo: `scripts/check-no-chromedp.sh`, `Makefile`, `.github/workflows/ci.yml`, `press/options.go` (`Output`/`Options` shape), `chase/model/document.go` (`Document.Sections` shape), `profiles/slides/slides.go` (`Sizes()`/`Container()`/`UnitElement()`) — read directly, ground truth for this objective's integration surface

### Secondary (MEDIUM confidence)
- `chromedp/chromedp` GitHub issues #703, #827 — `page.SetDocumentContent` frame-tree-first pattern and the `about:blank`-navigate-first requirement (verified across 2 independent issue threads)
- WebSearch cross-check: `--force-device-scale-factor` + `window.devicePixelRatio` floating-point artifact and DPR-doubling-on-retina reports (multiple independent sources, no single official doc)
- WebSearch cross-check: Chrome headless-dev group thread acknowledging PRNG-based non-determinism, no committed deterministic-mode date
- WebSearch cross-check: `CHROME_PATH` as a Lighthouse/chrome-launcher (not chromedp) convention; `chromote`'s `CHROMOTE_CHROME` as a divergent tool-specific convention

### Tertiary (LOW confidence)
- Exact current-Chrome status of `--font-render-hinting`-class flags — not independently confirmed this session; verify against whichever Chrome/`headless-shell` build is actually pinned at TRD/implementation time
- Font-load-race (`document.fonts.ready`) mitigation for data-URI `@font-face` — logically derived, not confirmed against a chromedp-specific worked example

## Metadata

**Confidence breakdown:**
- Standard stack (chromedp/cdproto API surface): HIGH — every signature verified directly against pkg.go.dev on the research date
- Architecture (package layout, discovery chain, HTML-loading mechanism): HIGH — cross-checked against this repo's actual `check-no-chromedp.sh`/`ARCHITECTURE.md` boundary plus 2+ independent GitHub-issue confirmations for `SetDocumentContent`
- Pitfalls/determinism specifics (DPR artifacts, font-load race, PRNG non-determinism): MEDIUM — community-reported, cross-checked but not single-official-source-confirmed; flagged inline throughout

**Research date:** 2026-07-21
**Valid until:** ~14 days (shorter than the usual 30-day default) — this objective's correctness is unusually coupled to the exact Chrome/`headless-shell` version in use (two independently documented PDF-path-only regression precedents already exist), so re-verify the pinned Chrome version and re-run the PDF-path conformance fixtures before TRD execution if this research is more than 2 weeks old.

---
*Research for: Eden Press Objective 5 (convert/pdf + convert/png, chromedp raster export)*
*Researched: 2026-07-21*
