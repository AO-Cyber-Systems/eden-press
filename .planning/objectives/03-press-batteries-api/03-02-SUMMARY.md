---
objective: 03-press-batteries-api
trd: "02"
subsystem: themes
tags: [go-embed, marp-themes, verbatim-vendor, npm-oracle, css-attribution, core-01]

# Dependency graph
requires:
  - objective: 02-model-profile
    provides: "chase/theme.Load/NewThemeSet/ThemeSet.Pack (Tier-1 Load + Tier-2 Pack pipeline); chase/theme.Size; profiles/slides.Profile supplying UnitElement/Scaffold/Sizes; conformance/cssdiff.Parse normalized CSS-AST comparator"
provides:
  - "themes/{default,gaia,uncover}.css — marp-core v4.4.0's OWN fully-compiled per-theme CSS, extracted verbatim through the real npm oracle, each with its /*! @theme … */ block as the leading comment"
  - "themes/browser-fit.js — marp-core's shipped lib/browser.js verbatim (Marp MIT header, y2018) for CORE-09's viewer-side auto-scaling consumer"
  - "themes (repo-root) Go package: go:embed holder exposing DefaultCSS/GaiaCSS/UncoverCSS/BrowserFitJS string constants"
  - "press/themes.ThemeSet(unit, scaffoldCSS, advancedBackgroundCSS, sizeFallback) (*theme.ThemeSet, error) — name-keyed set (default/gaia/uncover) built via the exact NewThemeSet+Load+Add flow chase/chase.go's packCSS uses; press/themes.Names(); press/themes.BrowserFitJS()"
  - "tools/corpus-gen/extract-themes.mjs — reproducible npm-oracle theme extraction (npm script 'extract-themes')"
affects: [03-05-chroma-hljs-remap, 03-09-press-render]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "go:embed patterns cannot contain '..', so the embed directives live in themes/embed.go (package themes, co-located with the asset files at repo root); press/themes imports that holder (aliased `assets`) rather than embedding across a parent boundary"
    - "Verbatim vendored CSS keeps Marp's OWN /*! @theme */ block as the file's LEADING comment (never AO-Cyber addlicense-stamped — an added header would displace the block chase/theme/meta.go's leadingComment() parses); the whole themes/** tree is -ignore'd in the CI addlicense -check"
    - "Extraction hoists marp-core's single per-theme /*! @theme … */ comment to the front byte-for-byte (default inlines github-markdown-css rules first; gaia emits a leading @charset) — the only transform, leaving every CSS rule/value untouched"
    - "cssdiff 'gate SHAPE' = format-insensitive shared-rule OVERLAP against the marp-core-rendered corpus expected.css, NOT full-document Equal (expected.css layers render-time passes press/themes doesn't apply at this objective); the .hljs-* highlight palette is the sharp byte-parity signal"

key-files:
  created:
    - tools/corpus-gen/extract-themes.mjs
    - themes/default.css
    - themes/gaia.css
    - themes/uncover.css
    - themes/browser-fit.js
    - themes/embed.go
    - press/themes/themes.go
    - press/themes/themes_test.go
  modified:
    - tools/corpus-gen/package.json
    - tools/corpus-gen/package-lock.json
    - NOTICE
    - .github/workflows/ci.yml

key-decisions:
  - "The compiled per-theme CSS accessor is marp.themeSet.get(<name>).css (confirmed by probing the marp-core v4.4.0 ThemeSet surface: themes()/get/getThemeProp). This yields the THEME-level compiled CSS (selectors still `section`/`h1`), NOT the render-time composite — exactly what chase/theme.Load expects. Retires research riskiest-item #1 (marp-core ships no precompiled per-theme .css)."
  - "embed.go lives in themes/ (package themes) not press/themes/, because go:embed cannot reach across '..' to the repo-root assets. The TRD's file_tree sketched press/themes/embed.go; its action text anticipated exactly this and recommended the themes/ co-location, which is what shipped. Documented as a deviation."
  - "The Objective-0 cssdiff comparison is implemented as a shared-rule OVERLAP gate (>=50% of packed rules byte-identical to marp-core's expected.css; >=90% of .hljs-* palette rules), NOT full cssdiff.Equal. Full document equality is not achievable at this objective by design — see Deviations."

requirements-completed: [CORE-01]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: true

# Metrics
duration: 14min
completed: 2026-07-21
---

# Objective 3 TRD 02: Bundled Marp Themes (go:embed) Summary

**The three official Marp themes (default/gaia/uncover) + the browser fit helper are bundled VERBATIM via go:embed — marp-core v4.4.0's own fully-compiled CSS pulled through the real npm oracle (never hand-recompiled Sass), each preserving its /*! @theme */ header — and wired into a name-keyed chase/theme.ThemeSet that Loads + Packs every theme to scoped CSS, ready for Options.Theme lookup in 03-09.**

## Performance

- **Duration:** ~14 min
- **Started:** 2026-07-21T13:35:19Z
- **Completed:** 2026-07-21T13:49:21Z
- **Tasks:** 3/3 complete
- **Files:** 8 created, 4 modified

## Accomplishments
- **Acquisition spike (research riskiest-item #1 retired):** `tools/corpus-gen/extract-themes.mjs` drives the real `@marp-team/marp-core` v4.4.0 and pulls each built-in theme's fully-compiled CSS off `marp.themeSet.get(<name>).css`, writing `themes/{default,gaia,uncover}.css` VERBATIM (byte-parity with marp-core — the CSS never touches a Go/Sass recompile). Reproducible via `cd tools/corpus-gen && npm ci && node extract-themes.mjs` (npm script `extract-themes`).
- **Leading-comment hoist:** each theme has exactly ONE comment (its `/*! @theme … */` metadata/copyright block); the extractor moves that single block to the file front byte-for-byte so `chase/theme/meta.go`'s `leadingComment()` (reads only the FIRST comment) parses it. `default` (github-markdown-css inlined first) and `gaia` (leading `@charset`) needed the hoist; `uncover` already led with it. No CSS rule/value altered.
- **browser-fit.js:** marp-core's shipped `lib/browser.js` vendored verbatim (the marp-auto-scaling web component + SVG polyfill), stamped with Marp's MIT header (`-c "Marp team (marp-team@marp.app)" -y 2018`) — the `.js` has no `leadingComment` conflict.
- **go:embed + name-keyed ThemeSet:** `themes/embed.go` (package `themes`) exposes `DefaultCSS`/`GaiaCSS`/`UncoverCSS`/`BrowserFitJS`; `press/themes.ThemeSet(...)` builds a `*theme.ThemeSet` via the exact `NewThemeSet`+`Load`+`Add` flow `chase/chase.go`'s `packCSS` uses, registered under each `@theme` name, profile-agnostic (unit/scaffold/advanced-bg/size-fallback all caller-supplied — no `press` import, no cycle). `Names()` + `BrowserFitJS()` accessors added.
- **Attribution:** `NOTICE` marks the reserved themes block LANDED (provenance = marp-core tag v4.4.0 + `extract-themes.mjs`), adds the **github-markdown-css** (MIT, Sindre Sorhus) fourth-asset entry inlined into `default.css`, and records `browser-fit.js` provenance; CI `addlicense -check` now `-ignore 'themes/**'` so the verbatim Marp-headered assets pass.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Extract compiled theme CSS + vendor browser-fit.js | `ls -l themes/*.css themes/browser-fit.js && head -8 themes/default.css \| grep -q '@theme default' && grep -o '\.hljs-[a-z-]*' themes/default.css \| sort -u \| head` | 0 | PASS |
| 2: go:embed + name-keyed ThemeSet | `go test ./press/themes/... -v && go build ./press/... && gofmt -l press/themes/` | 0 | PASS |
| 3: NOTICE + CI attribution | `addlicense … -ignore 'themes/**' -check . && grep -q 'github-markdown-css' NOTICE` | 0 | PASS |

## press/themes Test Results (7/7 PASS)

| Test | Proves |
|---|---|
| `TestEmbeddedThemesLoad` | each embedded CSS leads with `/*!` and `theme.Load`s to the right `@theme` name |
| `TestLeadingCommentGate` | negative control — a stray comment displacing the `/*! @theme */` block makes `Load` FAIL (leading-comment placement is load-bearing) |
| `TestThemeSetRegistersAllByName` | `ThemeSet` registers default/gaia/uncover + reserved scaffold, keyed by name |
| `TestEveryThemePacksNonEmpty` | each theme Packs to non-empty CSS scoped to `div.marpit > svg > foreignObject > section`, CONF-03-parseable |
| `TestCorpusSharedRuleGate` | shared-rule overlap vs marp-core expected.css: gaia 64/84 rules (hljs 20/21), uncover 52/68 rules (hljs 15/15) |
| `TestNames` | bundled name set + order (default first) |
| `TestBrowserFitJS` | helper embedded, carries Marp 2018 header, no AO-Cyber header |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt (whole repo) | `gofmt -l .` | 0 (no output) | PASS |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (incl. Obj-1 corpus/cssdiff) | `go test ./...` | 0 | PASS |
| addlicense (themes/** ignored) | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -check .` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS |
| no-browser dep in press | `go list -deps ./press/... \| grep -c chromedp` | 0 (clean) | PASS |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (no bugs required inline fixing during task execution)
- **Must-haves verified:** 5/5 (all `must_haves.truths` from 03-02-TRD.md)
- **Gate failures:** None

## Task Commits

Each task committed atomically via `df-tools.cjs commit` (never raw `git`):

1. **Task 1: extract compiled Marp themes + vendor browser-fit.js** — `b623195` (feat)
2. **Task 2: go:embed themes + name-keyed ThemeSet builder** — `0b29b86` (feat)
3. **Task 3: attribute vendored Marp theme assets (NOTICE + CI themes ignore)** — `2289ca1` (chore)

## Files Created/Modified
- `tools/corpus-gen/extract-themes.mjs` — npm-oracle theme extractor (probe accessor + hoist leading comment + vendor browser-fit.js)
- `tools/corpus-gen/package.json` / `package-lock.json` — bump `@marp-team/marp-core` to `^4.4.0`, add `extract-themes` script
- `themes/default.css`, `themes/gaia.css`, `themes/uncover.css` — verbatim compiled marp-core v4.4.0 themes
- `themes/browser-fit.js` — verbatim marp-core `lib/browser.js` (Marp MIT header)
- `themes/embed.go` — go:embed holder (package `themes`)
- `press/themes/themes.go` — `ThemeSet`/`Names`/`BrowserFitJS` builder
- `press/themes/themes_test.go` — 7 tests (Load/Pack/registration/leading-comment gate/corpus overlap)
- `NOTICE` — themes LANDED provenance + github-markdown-css fourth asset
- `.github/workflows/ci.yml` — `-ignore 'themes/**'` in addlicense check

## Decisions Made
- Compiled-CSS accessor = `marp.themeSet.get(<name>).css` (theme-level compiled CSS, pre-render-time-passes) — the correct artifact for `chase/theme.Load`.
- `embed.go` co-located in `themes/` (go:embed's no-`..` rule) rather than `press/themes/` as the file_tree sketched — the TRD action text recommended exactly this.
- cssdiff gate = shared-rule overlap, not full `Equal` (see Deviation 2).

## Deviations from Plan

### 1. [Layout] embed.go lives in themes/, not press/themes/
- **Why:** Go's `//go:embed` patterns cannot contain `..`, so a `press/themes/embed.go` could not reach the repo-root `themes/*.css` assets. The TRD frontmatter `files_modified` listed `press/themes/embed.go`, but the Task-2 `<action>` explicitly anticipated this ("go:embed reads paths relative to the .go file … Simplest: put the embed directives in a small `themes/embed.go` (`package themes` at repo root alongside the assets)"). Shipped the recommended co-located layout.
- **Impact:** `themes/embed.go` (an Eden-authored `.go` file) falls under the `themes/**` addlicense `-ignore`, so its AO-Cyber header isn't CI-enforced. It carries the correct AO-Cyber MIT header regardless; the ignore is a harmless superset. No functional impact.

### 2. [Scope note] cssdiff gate is shared-rule OVERLAP, not full-document Equal
- **Why:** `must_haves` truth 3 asks the packed themes to pass "the Objective-0 CSS-AST diff gate SHAPE (cssdiff.Equal … where a corpus case exists)." Empirically, full `cssdiff.Equal(Pack(name), expected.css)` does NOT hold, for well-understood, by-design reasons: the corpus `expected.css` is marp-core's FULL `render()` output, which layers render-time passes `press/themes` + `chase/theme.Pack` (a Marpit-level pipeline fed the slides profile's Marpit scaffold) does not apply at this objective — `marp-h1`/`:is()` custom-element expansion of heading selectors, emoji-CSS injection, highlight-base injection, and a fuller scaffold. Those belong to `press.Render` / later batteries (e.g. 03-05's chroma→hljs remap, an emoji battery), NOT this verbatim-theme-bundling TRD.
- **What shipped instead:** `TestCorpusSharedRuleGate` asserts a format-insensitive shared-rule OVERLAP under the CONF-03 `cssdiff` model — ≥50% of packed rules byte-identical to marp-core's expected.css, and ≥90% of the `.hljs-*` highlight palette (03-05's ground truth, which carries no heading/emoji render rewrite and IS byte-parity). Measured: gaia 64/84 rules + 20/21 hljs; uncover 52/68 rules + 15/15 hljs. This is the "gate SHAPE" the TRD hedges for and the intended bar (coordinator-confirmed: "a format-insensitive shared-rule overlap check is the intended bar … do NOT block the TRD on byte-parity").
- **Impact:** None on CORE-01 scope. The themes are still verbatim/byte-parity by CONSTRUCTION (extracted from marp-core's own compiled output). Byte-parity of the extracted asset is proven by the extraction mechanism itself, not by the render-composite diff.

---
**Total deviations:** 2 (1 layout, 1 documented scope note) — neither changes CORE-01's scope or the shipped artifacts. No auto-fixes (Rules 1-3) were needed.

## Issues Encountered
None beyond the two documented deviations. The extraction accessor was found on the first probe; all gates were green on first full run.

## User Setup Required
None. Re-extraction (dev-only) needs Node + `cd tools/corpus-gen && npm ci && node extract-themes.mjs`, then re-stamp `browser-fit.js` via `addlicense -l mit -s -c "Marp team (marp-team@marp.app)" -y 2018 themes/browser-fit.js`.

## Next Objective/TRD Readiness
- **03-05** (chroma→hljs remap) can `grep -o '\.hljs-[a-z-]+' themes/default.css` for its ground-truth selector set — the compiled `default.css` is now on disk with the full `.hljs-*` palette.
- **03-09** (press.Render) can call `press/themes.ThemeSet(...)` with the active profile's primitives and `Pack(opts.Theme→front-matter→"default", …)` against a fully-populated, name-keyed set.
- **CORE-09** (viewer) can consume `press/themes.BrowserFitJS()`.

## Self-Check: PASSED

All claimed files confirmed present on disk; all three task commit hashes confirmed in `git log`.

- FOUND: tools/corpus-gen/extract-themes.mjs
- FOUND: themes/default.css, themes/gaia.css, themes/uncover.css, themes/browser-fit.js
- FOUND: themes/embed.go
- FOUND: press/themes/themes.go, press/themes/themes_test.go
- FOUND: NOTICE (github-markdown-css entry), .github/workflows/ci.yml (themes/** ignore)
- FOUND: .planning/objectives/03-press-batteries-api/03-02-SUMMARY.md
- FOUND commit: b623195 (Task 1), 0b29b86 (Task 2), 2289ca1 (Task 3)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
