# Objective 0: Conformance Corpus, Acceptance Gate & Attribution Bootstrap - Research

**Researched:** 2026-07-20
**Domain:** Markdown/CSS conformance testing infrastructure (Go) + OSS licensing/attribution mechanics
**Confidence:** MEDIUM-HIGH (spec-suite and attribution mechanics are HIGH; the CSS-AST diff comparator is explicitly LOW/novel — see below)

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CONF-01 | Language-neutral golden corpus of Markdown→HTML/CSS cases, seeded from Marp's own Jest snapshot fixtures (MIT) | **Corrected finding**: marpit/marp-core have exactly one literal Jest `.snap` file (`test/__snapshots__/marp.ts.snap` in marp-core, 76 lines, KaTeX font-URL rewriting only — not general-purpose). The real reusable substance is inline Jest `describe/it` assertions in `test/markdown/*.js` (marpit) and `test/marp.ts`, `test/math/*.ts`, `test/size/*.ts` (marp-core). See "Don't Hand-Roll" and "Architecture Patterns" for the extraction mechanism (render via a real npm harness, don't hand-transcribe). |
| CONF-02 | Test runner renders each case and compares DOM-normalized HTML (ignores cosmetic `<br>`/`<hr>`/whitespace/attr-order) | Prior-art spike (`md-spike/main.go`) already implements this over `golang.org/x/net/html`. See "Code Examples" — lift and harden. |
| CONF-03 | CSS-AST diff comparator for theme-CSS output (new tooling, not a DOM diff) | `tdewolff/parse/v2/css` is a grammar-token stream, not a typed AST — confirmed no off-the-shelf Go AST diff library exists. See "Architecture Patterns > Pattern 3" for the recommended build-your-own-normalized-model approach, and "Open Questions" for the spike-first recommendation. **Flagged highest-uncertainty item in this objective.** |
| CONF-04 | Full CommonMark + GFM spec sweep (not just the 32-case spike), acceptance gate for every engine objective | Exact sources, URLs, extraction commands, and JSON schemas documented in "Architecture Patterns > Pattern 2" and "Code Examples". goldmark's own `commonmark_test.go`/`testutil` package is a directly reusable reference pattern. |
| LIC-01 | LICENSE (MIT) for Eden Press | Standard MIT text provided in "Code Examples". Copyright holder needs a decision (see "Open Questions"). |
| LIC-02 | NOTICE/CREDITS crediting Marpit, Marp Core, Marp CLI + deps (goldmark, chroma, latex2mathml, go-latex/latex) with licenses | Verified exact upstream copyright lines for all 3 Marp repos; convention documented in "Architecture Patterns > Pattern 4". |
| LIC-03 | Per-file MIT headers preserving original Marp copyright on verbatim-reused assets (3 themes + browser fit/polyfill script) | Exact vendored file identities confirmed (`marp-core/themes/{default,gaia,uncover}.scss`; browser script is a **build artifact**, not a raw source file — see "Common Pitfalls > Pitfall 2"). `google/addlicense` tool identified for enforcement. |
| LIC-04 | README acknowledgment + "not affiliated/endorsed" disclaimer | Wording pattern in "Architecture Patterns > Pattern 4". |

</phase_requirements>

## Summary

This objective has two largely independent halves that should be built in parallel-ish but sequenced by risk: (1) a conformance-testing harness (corpus + DOM-diff runner + CSS-diff runner + spec sweep) and (2) attribution/licensing paperwork. The attribution half is low-risk, well-precedented, and can be done in an afternoon. The conformance half has one genuinely novel piece — the CSS-AST diff comparator — that has no direct prior art (not in the existing spikes, not as an off-the-shelf Go library) and should be spiked in isolation before being wired into the acceptance gate.

Concretely: goldmark (the chosen Markdown engine) already ships its own CommonMark-spec-test harness (`commonmark_test.go` + `testutil` package) that binds a JSON array of `{markdown, html, example, section}` objects to a Go `testing.T` loop — this is a directly reusable reference pattern (not an importable dependency, since `testutil` is designed for goldmark's own internal use, but its shape should be mirrored). CommonMark's `spec.txt`→`spec.json` dump script is the standard, official extraction mechanism (confirmed: 657 examples in the current 0.31.2 spec). GFM's spec has forked into two live, actively-synced files in `github/cmark-gfm` (`test/spec.txt`, a CommonMark mirror, 650 examples; `test/extensions.txt`, GFM-only additions, 30 examples) — the historical combined "~672-example" spec used by the public gfm.github.io site is a frozen 2019 snapshot (tag `0.29.0.gfm.13`) and should NOT be used as the authoritative source going forward.

The `test/__snapshots__/marp.ts.snap` file is the *only* literal Jest snapshot artifact in either Marp repo, and it covers KaTeX font-URL rewriting only — not general Markdown/CSS output. CONF-01's "Jest snapshot fixtures" premise should be read as "Marp's Jest **test suite**" (inline assertions), not literal `.snap` files. The reusable extraction method is exactly the pattern already proven in the prior-art spike's `ref.mjs`: a small Node.js harness that installs the real npm package and renders real inputs to get ground-truth output — not hand-transcription from JS assertion code.

**Primary recommendation:** Build the DOM-diff runner and CommonMark/GFM spec sweep first (low risk, direct extension of proven prior art and a well-documented upstream pattern), stand up LICENSE/NOTICE/headers/README in parallel (near-zero risk, half a day), and treat the CSS-AST diff comparator as its own 1-2 day timeboxed spike with an explicit negative-test exit criterion before folding it into the gate.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `golang.org/x/net/html` | latest (stdlib-adjacent, no semver floor concerns) | HTML5 fragment parsing for DOM-normalized diffing | Already proven in the prior-art spike; the only credible pure-Go HTML5 parser (implements the WHATWG tree-construction algorithm) |
| `github.com/tdewolff/parse/v2` + `.../css` | v2.8.13 (actively maintained; pushed 2026-07-14) | CSS tokenizing/grammar-stream parsing for both the theme-CSS engine and the CSS-AST diff comparator | Already the project's chosen CSS tool (per PROJECT.md constraints); reusing it for the diff comparator keeps parsing-edge-case behavior consistent with what the engine itself produces |
| `github.com/yuin/goldmark` | v1.8.4 (stable; **not** v2, which is `v2.0.0-beta.5` as of 2026-07-20 and still beta) | Markdown/CommonMark/GFM parser under test | Chosen engine per PROJECT.md; ships its own reusable spec-test pattern (see below) |
| Go (toolchain) | 1.25+ floor (installed locally: 1.26.4; current stable: 1.26.5) | `go:embed` for corpus/spec fixtures, `testing.T` | STACK.md-confirmed floor; `go:embed` has been stdlib since 1.16, no risk |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/addlicense` | latest | Per-file SPDX/MIT header insertion + CI verification (`-check` mode) | LIC-03 enforcement — run once to stamp headers, then `-check` in CI to prevent drift |
| Node.js + npm (`@marp-team/marpit`, `@marp-team/marp-core`) | pinned to the exact tag being mined (currently marpit v3.2.2, marp-core v4.4.0) | One-off, throwaway harness to render real Marp output as ground truth for the golden corpus | Build-time-only tooling, never a runtime or CLI dependency — mirrors the existing `ref.mjs` spike pattern; delete/isolate after corpus generation, or keep in a `tools/corpus-gen/` dir excluded from the Go module's dependency graph |
| Python 3 (stdlib only) | any 3.x | Running CommonMark's official `test/spec_tests.py --dump-tests` extraction script | One-time (or CI-scheduled) extraction step, not a build dependency |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `tdewolff/parse/v2/css` for the diff comparator | `aymerick/douceur` (wraps gorilla/css tokenizer, exposes an actual `Stylesheet`/`Rule`/`Declaration` object model) | douceur's object model is more diff-friendly out of the box, but the repo is stale (last push 2022-09, though not archived) and introduces a second CSS parser with potentially different edge-case behavior than the engine's own tdewolff-based parser — rejected in favor of consistency with the already-chosen engine dependency |
| A hand-rolled Go CSS-AST diff | JS-world `compare-stylesheet` (built on csstree/csso) as a **design-pattern reference only**, not a dependency | Confirms the right normalization shape (parse → normalize to `{selector, property, value}` tuples → compare), but is JS tooling and out of scope for a JS-free backend; use only to validate the Go design, not to import |
| Manual "check Marp's GitHub releases page" reminder | Scheduled GitHub Action using `gh release view --repo <marp-repo> --json tagName` | Manual reminders silently rot (Pitfall 13); a scheduled workflow with a pinned-version comparison + `gh issue create` is a real mechanical trigger |

**Installation:**
```bash
go get golang.org/x/net/html
go get github.com/tdewolff/parse/v2
go install github.com/google/addlicense@latest
```

## Architecture Patterns

### Recommended Project Structure

```
eden-press/
├── go.mod                              # module github.com/AO-Cyber-Systems/eden-press ; go 1.25
├── LICENSE                             # MIT, Eden Press copyright (LIC-01)
├── NOTICE                              # third-party credits (LIC-02)
├── README.md                           # acknowledgment + not-affiliated disclaimer (LIC-04)
├── themes/
│   ├── default.scss                    # verbatim from marp-core, MIT header preserved (LIC-03)
│   ├── gaia.scss
│   ├── uncover.scss
│   └── browser-fit.js                  # vendored build artifact, MIT header preserved (LIC-03)
├── conformance/                        # the acceptance gate — imported by every engine test
│   ├── corpus/
│   │   ├── cases/                      # golden Markdown->HTML/CSS cases (CONF-01)
│   │   │   └── <case-id>/
│   │   │       ├── input.md
│   │   │       ├── options.json        # Marpit/Marp Core constructor options used to render this case
│   │   │       ├── expected.html
│   │   │       └── expected.css        # only present for theme-CSS cases
│   │   └── extract/                    # one-off Node.js harness (throwaway/build-time only)
│   │       ├── package.json            # pins @marp-team/marpit + @marp-team/marp-core to exact tags
│   │       └── generate.mjs            # mines test/*.js assertions -> renders real output -> writes cases/
│   ├── spec/
│   │   ├── commonmark/spec.json        # vendored, generated via commonmark-spec's dump script
│   │   └── gfm/{spec.json,extensions.json}
│   ├── htmldiff/                       # CONF-02: DOM-normalized HTML diff (lift from md-spike)
│   │   ├── normalize.go
│   │   └── normalize_test.go           # includes the negative test (broken <pre> case)
│   ├── cssdiff/                        # CONF-03: CSS-AST diff (new, spike first)
│   │   ├── model.go                    # normalized Stylesheet/Rule/Declaration types
│   │   ├── build.go                    # tdewolff/parse/v2/css grammar stream -> model
│   │   ├── diff.go
│   │   └── diff_test.go                # includes the negative test (broken theme case)
│   └── runner/
│       ├── corpus_test.go              # go test entrypoint over corpus/cases
│       └── spec_test.go                # go test entrypoint over spec/{commonmark,gfm}
└── .github/workflows/
    ├── ci.yml                          # runs `go test ./conformance/...` as a required check
    └── upstream-drift.yml              # scheduled: compares pinned Marp versions vs latest releases (Q6)
```

This mirrors PROPOSAL.md §5.1/§6's sketch and keeps `conformance/` as a normal Go package tree so later engine objectives import `conformance/htmldiff` and `conformance/cssdiff` directly as test helpers, and `go test ./conformance/...` is itself the CI-required acceptance gate — no bespoke standalone-binary runner needed (unlike the prior-art spike, which was a `main.go` because it was a throwaway spike, not a shippable test suite).

### Pattern 1: DOM-normalized HTML diff (CONF-02) — lift and harden the spike

**What:** Parse both the actual and expected HTML fragments with `golang.org/x/net/html`'s fragment parser (using a `<body>` context node so bare block content works), then walk both trees emitting a canonical token stream: sort attributes alphabetically, collapse inter-block whitespace via `strings.Fields`, but preserve `<pre>`/`<code>` text **verbatim** (whitespace-significant). Compare the two canonical strings.

**When to use:** Every corpus case's `expected.html` comparison, and every CommonMark/GFM spec example.

**Example (from the prior-art spike — to be hardened, not reinvented):**
```go
// Source: prior-art spike, /private/tmp/.../scratchpad/md-spike/main.go (our own code, no external attribution needed)
func normalize(fragment string) (string, error) {
	nodes, err := html.ParseFragment(strings.NewReader(fragment), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, n := range nodes {
		walk(&sb, n, false)
	}
	return sb.String(), nil
}
```
**Hardening gaps to close vs. the spike** (found by re-reading it against CONF-02's exact wording "ignores cosmetic `<br>`/`<hr>`/whitespace/attr-order"): the spike's `walk()` already handles attr-order and whitespace; explicitly confirm/add void-element-syntax normalization (`<br>` vs `<br/>` vs `<br />`) as its own test case, since `x/net/html`'s tokenizer already collapses these at parse time but this should be a named, asserted behavior, not an incidental one.

### Pattern 2: CommonMark + GFM spec sweep (CONF-04)

**What:** Vendor pre-generated `spec.json` files (don't fetch/parse spec.txt at test time — goldmark itself vendors a static `_test/spec.json`, confirmed 140KB, committed to their repo rather than regenerated per test run). Bind each JSON example to a Go test case via a struct matching CommonMark's dump schema, run through goldmark, DOM-diff the output, and track pass/fail keyed by the `section` field for per-section reporting (directly satisfies CONF-04's "tracked per spec section").

**Sources and exact extraction commands:**
- **CommonMark**: `github.com/commonmark/commonmark-spec`, current tag `0.31.2`. Extract via the repo's own documented mechanism: `python3 test/spec_tests.py --dump-tests < spec.txt > spec.json`. Verified: 657 `example` blocks in the current `spec.txt`. JSON schema per example: `{"markdown": ..., "html": ..., "example": <int>, "start_line": <int>, "end_line": <int>, "section": "<string>"}`.
- **GFM**: `github/cmark-gfm`, live `master` branch. Two files, both extracted via the same `spec_tests.py --dump-tests` mechanism (cmark-gfm forked commonmark-spec's own tooling):
  - `test/spec.txt` — mirrors upstream CommonMark (kept current via the project's own `make update-spec`), 650 examples measured on 2026-07-20.
  - `test/extensions.txt` — GFM-only additions (Tables, Strikethrough, Autolinks, disallowed-raw-HTML tag filter, Footnotes, front-matter/Interop, Task lists), 30 examples measured.
  - **Note on the "~672" figure** cited in prior project research (PITFALLS.md/SUMMARY.md): that number traces to the frozen historical tag `0.29.0.gfm.13` (version 0.29, published 2019; basis of the publicly rendered spec at `https://github.github.com/gfm/`), which I measured at 650 combined examples — not 672. I could not find a primary source that produces exactly 672. **Recommendation: use the live split `master` files (650 + 30 = 680 total, actively synced with upstream CommonMark) as the authoritative source, not the frozen 2019 tag**, and treat "~672" as an approximation to be superseded by the actual vendored count once generated.

**Reference pattern (goldmark's own spec runner — mirror the shape, don't import `testutil` directly since it's designed as goldmark's internal package):**
```go
// Source: https://github.com/yuin/goldmark/blob/master/commonmark_test.go (MIT, for pattern reference only)
type commonmarkSpecTestCase struct {
	Markdown  string `json:"markdown"`
	HTML      string `json:"html"`
	Example   int    `json:"example"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Section   string `json:"section"`
}
// goldmark vendors a static _test/spec.json (committed, 140KB) generated once via
// commonmark-spec's dump script — regenerate on CommonMark version bumps, not per-test-run.
```
Eden Press's own runner should follow this exact shape but route the actual/expected comparison through `conformance/htmldiff` (DOM-normalized) rather than goldmark's own byte-exact `bytes.Equal` (goldmark's `testutil.DoTestCase` uses raw `bytes.TrimSpace` equality — appropriate for goldmark validating itself against a byte-precise spec fixture, but Eden Press is validating a **different** engine against the same fixtures and should tolerate the same cosmetic variance CONF-02 already allows for the corpus).

### Pattern 3: CSS-AST diff comparator (CONF-03) — build order for the spike

**What:** `tdewolff/parse/v2/css` exposes a **grammar-token stream** (`AtRuleGrammar`, `BeginRulesetGrammar`, `DeclarationGrammar`, `TokenGrammar`, etc. via `p.Next()`/`p.Values()`) — confirmed via official docs and current source, there is no typed/navigable AST comparable to the sibling `js` package's `ast.go`. This means Eden Press must build its own thin `Stylesheet`/`Rule`/`Declaration` model on top of the grammar stream before any diffing is possible.

**Recommended build order (spike, 1-2 days, timeboxed):**
1. Walk `p.Next()` and materialize a normalized model: `type Stylesheet struct { Rules []Rule }`, `type Rule struct { Selector string; Declarations []Declaration; AtRule string }`, `type Declaration struct { Property, Value string; Important bool }`.
2. Normalize *within* each node only (not across the tree): lowercase hex colors, collapse redundant whitespace in selectors/values, strip comments, normalize quote style — but do **not** reorder rules or declarations. This is a deliberate departure from the JS-world `compare-stylesheet` pattern (which explicitly ignores rule order for general-purpose stylesheet equivalence testing) — for Eden Press's use case (comparing the *same* theme-CSS engine's output across a code change, not comparing two independently-authored stylesheets), declaration/rule order is itself part of what's being verified, since CSS cascade semantics depend on it and an accidental reorder is exactly the kind of regression this gate should catch.
3. Diff the two normalized `Stylesheet` values structurally (a straightforward recursive struct-field diff is sufficient — no line-based diff algorithm needed since both sides are already fully parsed).
4. **Exit criterion for the spike**: prove it on a negative test — take one of the 3 vendored themes' expected CSS, intentionally break it (e.g., swap a property value, drop an `!important`, change a selector's specificity-relevant class), and confirm the comparator reports a diff. This is the exact validation ROADMAP.md's success criteria calls for.

**Example (illustrative model shape, not yet implemented — this is the spike's starting point):**
```go
// Source: this project (novel — no direct upstream/Context7 precedent for a Go CSS-AST diff)
type Declaration struct {
	Property  string
	Value     string
	Important bool
}
type Rule struct {
	Selector     string
	Declarations []Declaration
}
type Stylesheet struct {
	Rules []Rule
	// AtRules for @media/@keyframes etc. — extend once theme-CSS engine's actual
	// at-rule usage (pagination, auto-scaling breakpoints) is known from later objectives
}
```

### Anti-Patterns to Avoid
- **Treating the CSS-AST diff comparator as a trivial extension of the HTML diff:** it is not — HTML attribute order is safely ignorable, CSS declaration/rule order is cascade-significant and must be preserved through normalization, not discarded.
- **Fetching Marp's build artifacts (e.g., the browser fit script) via raw GitHub URLs:** `browser.js` at marp-core's repo root is a one-line re-export stub (`module.exports = require('./lib/browser.cjs')`); `lib/` is gitignored (confirmed via `.gitignore`) and only exists as a rollup/terser build output published to npm. Fetching the raw GitHub path gets you the stub, not the actual script.
- **Hand-transcribing expected HTML/CSS from Jest assertion source code:** error-prone and not truly "the real upstream output." Render through the actual npm package instead (mirrors the existing `ref.mjs` spike pattern).
- **Using the frozen `0.29.0.gfm.13` GFM spec as the CONF-04 source:** it's a 2019 snapshot; the live `github/cmark-gfm` master `test/spec.txt` + `test/extensions.txt` split is actively maintained and already naturally section-tagged.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| HTML5 fragment parsing for the DOM diff | A custom HTML tokenizer/tree-builder | `golang.org/x/net/html` | Implements the actual WHATWG tree-construction algorithm; already proven in the prior-art spike |
| CommonMark/GFM spec extraction | Hand-copying examples out of the rendered spec pages | `spec_tests.py --dump-tests` (official, both commonmark-spec and cmark-gfm ship it) | Produces exact, versioned, line-numbered, section-tagged JSON — matches goldmark's own vendoring approach |
| Per-file MIT/SPDX header stamping + drift detection | A custom regex-based header injector/checker script | `google/addlicense` (`-l mit -s -c "..." -check`) | Handles comment-syntax detection per file extension, idempotent re-runs, CI check mode, in active use across many Go projects |
| Golden-fixture generation from Marp's JS test suite | Manually transcribing expected HTML/CSS from Jest `expect()` assertions | A small throwaway Node.js harness installing the real npm package and rendering real inputs (mirrors `ref.mjs`) | Guarantees the golden output is what upstream Marp *actually* produces, not what a human believes the assertion implies |

**Key insight:** Every piece of this objective that looks like "just write a small script" has a real upstream artifact or tool that already does it correctly (spec dump scripts, addlicense, x/net/html) — the only place genuinely without prior art is the CSS-AST diff model itself, and that should be scoped and timeboxed accordingly rather than assumed to be "just like the HTML one."

## Common Pitfalls

### Pitfall 1: Marp's "Jest snapshot fixtures" aren't what CONF-01's name suggests
**What goes wrong:** Assuming there's a directory of `.snap` golden files to bulk-copy.
**Why it happens:** Marpit/marp-core use Jest for `describe/it`-style behavioral assertions (via cheerio DOM queries), not golden-file snapshot testing, except for one narrow case.
**How to avoid:** Confirmed via `gh api search/code -f q="filename:.snap"`: the only literal snapshot file is `marp-core/test/__snapshots__/marp.ts.snap` (KaTeX font-URL rewriting, 3 cases, irrelevant to general HTML/CSS conformance). Treat CONF-01 extraction as: read `marpit/test/markdown/*.js`, `marpit/test/postcss/*.js`, `marp-core/test/marp.ts`, `marp-core/test/math/*.ts`, `marp-core/test/size/*.ts` to identify interesting `(markdown, options)` pairs, then render them through a real npm harness to capture ground truth.
**Warning signs:** If a build step tries to `curl`/`gh api` a `__snapshots__` directory expecting dozens of files, it will find effectively none.

### Pitfall 2: The "browser fit/polyfill script" is a build artifact, not a source file
**What goes wrong:** Fetching `marp-core/browser.js` or `src/browser.ts` from GitHub and treating it as the shippable asset.
**Why it happens:** The root `browser.js` is a 1-line CJS re-export stub; the real bundle (`lib/browser.cjs`) is produced by marp-core's own rollup+terser build and is gitignored (confirmed in `.gitignore`: `lib/`).
**How to avoid:** Either (a) `npm pack @marp-team/marp-core@<pinned-version>` and extract `lib/browser.js` from the tarball, or (b) clone marp-core at the pinned tag and run its own `npm ci && npm run build`. Vendor whichever exact built artifact is produced — this is what "verbatim-reused" should mean for LIC-03, and it needs its own MIT header (the file itself won't have one, since it's minified build output).
**Warning signs:** A vendored "browser fit script" that's one line long, or that `require()`s a path that doesn't exist in this repo.

### Pitfall 3: CSS declaration/rule order is semantically significant — don't normalize it away
**What goes wrong:** Copying the JS-world `compare-stylesheet` pattern (order-independent stylesheet equivalence) wholesale into the CSS-AST diff comparator.
**Why it happens:** It's the closest existing prior art and its normalization approach (ignore rule order, compare as a set) looks like the "obvious" generalization of the HTML attr-order-insensitive approach.
**How to avoid:** For Eden Press's actual use case — verifying the theme-CSS engine's own output hasn't regressed — order changes are often exactly the bug class this gate exists to catch (an accidental selector-scoping reorder can silently change which rule wins the cascade). Normalize whitespace/quote-style/hex-case within nodes; preserve rule and declaration order in the comparison.
**Warning signs:** A negative test that reorders two conflicting declarations (e.g., swaps which of two same-property declarations comes last) passes when it should fail.

### Pitfall 4: Upstream-drift tracking without a mechanical trigger silently becomes "never"
**What goes wrong:** Documenting "check Marp's releases periodically" as a process note with no automation.
**Why it happens:** It's the path of least resistance during initial setup, and Marp ships frequently (confirmed: marpit v3.2.2 on 2026-07-04, marp-core v4.4.0 and marp-cli v4.5.0 both on 2026-07-17 — all within the last two weeks of this research date), so manual tracking falls behind fast.
**How to avoid:** A scheduled GitHub Action (see "Architecture Patterns" workflow file and "Code Examples") comparing `gh release view --repo marp-team/<repo> --json tagName` against a version pinned in a checked-in file, opening/deduplicating a GitHub issue on drift.
**Warning signs:** No `.github/workflows/*drift*.yml` exists, or a "pinned versions" file exists but nothing reads it in CI.

## Code Examples

### CommonMark spec extraction (Q2)
```bash
# Source: commonmark/commonmark-spec Makefile (`spec.json` target)
git clone --branch 0.31.2 --depth 1 https://github.com/commonmark/commonmark-spec.git
cd commonmark-spec
python3 test/spec_tests.py --dump-tests < spec.txt > spec.json
# -> JSON array of {markdown, html, example, start_line, end_line, section}
```

### GFM spec extraction (Q2)
```bash
# Source: github/cmark-gfm, live master (same spec_tests.py mechanism, forked from commonmark-spec)
git clone --depth 1 https://github.com/github/cmark-gfm.git
cd cmark-gfm/test
python3 spec_tests.py --dump-tests < spec.txt > spec.json          # 650 examples (CommonMark mirror)
python3 spec_tests.py --dump-tests < extensions.txt > extensions.json  # 30 examples (GFM-only)
```

### Upstream-drift CI (Q6)
```yaml
# Source: pattern adapted from common gh-CLI drift-check conventions (no single canonical
# marketplace action fits 3 repos + issue-dedup cleanly; hand-rolled with `gh` is standard practice)
name: Check Marp upstream drift
on:
  schedule:
    - cron: '0 6 * * 1'   # weekly, Monday 06:00 UTC
  workflow_dispatch:
permissions:
  contents: read
  issues: write
jobs:
  check:
    runs-on: ubuntu-latest
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    strategy:
      matrix:
        repo: [marp-team/marpit, marp-team/marp-core, marp-team/marp-cli]
    steps:
      - uses: actions/checkout@v4
      - name: Compare pinned vs latest
        run: |
          LATEST=$(gh release view --repo "${{ matrix.repo }}" --json tagName --jq '.tagName')
          PINNED=$(grep "${{ matrix.repo }}" UPSTREAM-VERSIONS.txt | cut -d' ' -f2)
          if [ "$LATEST" != "$PINNED" ]; then
            TITLE="Upstream drift: ${{ matrix.repo }} released $LATEST (pinned: $PINNED)"
            if [ -z "$(gh issue list --search "$TITLE in:title" --json number --jq '.[].number')" ]; then
              gh issue create --title "$TITLE" --label upstream-drift \
                --body "Conformance corpus was mined from ${{ matrix.repo }} @ $PINNED. Latest is $LATEST. Review for spec/theme changes."
            fi
          fi
```

### Per-file MIT header stamping + CI check (LIC-03)
```bash
# Source: google/addlicense README (https://github.com/google/addlicense)
addlicense -l mit -s -c "Marp team (marp-team@marp.app)" -v themes/default.scss themes/gaia.scss themes/uncover.scss themes/browser-fit.js
# CI verification (fails the build if a vendored file is missing its header):
addlicense -l mit -s -check themes/
```

### Root MIT LICENSE (LIC-01)
```
MIT License

Copyright (c) 2026 AO Cyber Systems

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
(Copyright holder "AO Cyber Systems" is a recommendation, not a verified decision — see Open Questions.)

### NOTICE/CREDITS convention (LIC-02)
```
# Source: convention pattern, common in mature MIT-licensed Go projects that vendor
# third-party MIT/BSD assets (e.g., Kubernetes-style NOTICE, simplified for MIT-only deps)

Eden Press
Copyright (c) 2026 AO Cyber Systems

This product includes software developed as part of the following
third-party projects. Eden Press is a clean-room reimplementation and is
NOT affiliated with, endorsed by, or sponsored by the Marp team.

--------------------------------------------------------------------------------
Marpit (https://github.com/marp-team/marpit)
Copyright (c) 2018- Marp team (marp-team@marp.app)
License: MIT

Marp Core (https://github.com/marp-team/marp-core)
Copyright (c) 2018 Marp team (marp-team@marp.app)
License: MIT
  - Bundled themes (default, gaia, uncover) and the browser fit/polyfill script
    are used verbatim under their original MIT license and copyright; see
    per-file headers in themes/.

Marp CLI (https://github.com/marp-team/marp-cli)
Copyright (c) 2018 Marp team (marp-team@marp.app)
License: MIT

--------------------------------------------------------------------------------
goldmark (https://github.com/yuin/goldmark) — MIT
chroma (https://github.com/alecthomas/chroma) — MIT
latex2mathml (https://github.com/... ) — [verify exact license in Objective covering math]
go-latex/latex (https://github.com/go-latex/latex) — [verify exact license in Objective covering math]
tdewolff/parse (https://github.com/tdewolff/parse) — MIT
```
(Note: license verification for latex2mathml and go-latex/latex is explicitly out of this objective's scope per the requirement text listing them under LIC-02's credit list; STACK.md/prior research already vetted the math stack — cross-reference rather than re-verify here.)

### README acknowledgment wording (LIC-04)
```markdown
## Acknowledgments

Eden Press is inspired by and Markdown-compatible with [Marp](https://marp.app)
(Marpit, Marp Core, Marp CLI), implemented clean-room in Go with zero JavaScript
in the backend. Eden Press is **not affiliated with, endorsed by, or sponsored
by the Marp team**. See [NOTICE](./NOTICE) for full third-party credits.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| Combined GFM spec as one file (`spec.txt` with everything, e.g. the 2019 `0.29.0.gfm.13` tag basis of gfm.github.io) | Split `test/spec.txt` (CommonMark-synced) + `test/extensions.txt` (GFM-only) on `cmark-gfm` master | Ongoing (cmark-gfm actively syncs `spec.txt` against upstream CommonMark via `make update-spec`) | The public gfm.github.io rendering is stale relative to master; use the live split files for CONF-04, not the public page or its underlying frozen tag |
| goldmark v1.x | goldmark v2 in active beta (`v2.0.0-beta.5`, confirmed 2026-07-20) | Ongoing beta series | Do not adopt v2 yet — PROJECT.md/STACK.md correctly pin v1.8.4 |

**Deprecated/outdated:**
- The "~672 GFM examples" figure floating in prior project research documents traces to the frozen 2019 combined spec (measured 650) — supersede with the live split-file count (650 + 30 = 680) once actually vendored and counted at extraction time.

## Open Questions

1. **What is the exact legal copyright-holder name for Eden Press's own LICENSE/NOTICE?**
   - What we know: PROJECT.md frontmatter ties the repo to GitHub org `AO-Cyber-Systems`; the user's global brand context refers to "AO Cyber Systems" as the company name.
   - What's unclear: whether the copyright line should read "AO Cyber Systems", a specific legal entity name, or an individual maintainer — this wasn't specified in PROPOSAL.md/PROJECT.md/ROADMAP.md.
   - Recommendation: use "AO Cyber Systems" (matches org branding) unless the user specifies a different legal entity; this is a one-line fix if wrong, non-blocking.

2. **Should the CSS-AST diff comparator support `@media`/`@keyframes` at-rules in v1, or defer until the pagination/auto-scaling theme-CSS objective lands?**
   - What we know: the 3 vendored themes use `@theme`/`@auto-scaling`/`@size` custom comment-based directives (parsed separately by the theme-CSS engine, not standard CSS at-rules) plus likely some standard `@media print` usage for PDF export.
   - What's unclear: the exact at-rule surface the theme-CSS engine objective will need, since that engine doesn't exist yet (this is Objective 0, dependency-free).
   - Recommendation: build the CSS-AST diff's initial model (`Stylesheet`/`Rule`/`Declaration`) without at-rule support, and extend it when the theme-CSS engine objective's actual output shape is known — don't speculatively build at-rule handling now.

3. **Exact vendored `browser-fit.js` provenance path — npm tarball vs. from-source build?**
   - What we know: both are viable (see Pitfall 2); `npm pack` is simpler and matches exactly what Marp CLI itself ships to end users.
   - What's unclear: whether a from-source build is preferred for supply-chain-transparency reasons (Eden's stated privacy/sovereignty ethos might favor "build from source, pin the commit" over "trust npm's published tarball").
   - Recommendation: `npm pack @marp-team/marp-core@<pinned-tag>` for speed; note the tarball's own SHA/integrity hash in NOTICE for traceability. Revisit if a stricter supply-chain policy is set elsewhere in the project.

## Sources

### Primary (HIGH confidence)
- `github.com/commonmark/commonmark-spec` (tag `0.31.2`) — `spec.txt`, `test/spec_tests.py`, verified example count via direct fetch + grep
- `github.com/github/cmark-gfm` (master + tag `0.29.0.gfm.13`) — `test/spec.txt`, `test/extensions.txt`, verified example counts via direct fetch + grep
- `github.com/yuin/goldmark` (master) — `commonmark_test.go`, `testutil/testutil.go`, `_test/spec.json`, `Makefile` — fetched and read in full
- `github.com/marp-team/marpit` (main) — `LICENSE`, `package.json`, root/`test/` directory listing via `gh api`
- `github.com/marp-team/marp-core` (main) — `LICENSE`, `themes/*.scss`, `browser.js`, `.gitignore`, `rollup.config.mjs`, `test/__snapshots__/marp.ts.snap` — fetched and read in full
- `pkg.go.dev/github.com/tdewolff/parse/v2/css` — official docs, confirmed grammar-stream (not AST) API surface
- `github.com/google/addlicense` — README, confirmed flags/usage
- `gh api repos/{marpit,marp-core,marp-cli}/releases/latest` — live version check, 2026-07-20

### Secondary (MEDIUM confidence)
- `compare-stylesheet` (npm/GitHub) — WebSearch-surfaced, used only as a design-pattern reference for the CSS diff normalization shape, not as a dependency
- `aymerick/douceur` maintenance status (`pushed_at: 2022-09-11`, not archived) — confirmed via `gh api`, used to justify rejecting it as an alternative

### Tertiary (LOW confidence)
- The generic "scheduled GitHub Action checks upstream release" YAML pattern — WebSearch-derived template, adapted but not tested end-to-end in this research pass; recommend a dry run (`workflow_dispatch`) before relying on the schedule

## Metadata

**Confidence breakdown:**
- Standard stack (HTML diff, spec sweep, addlicense): HIGH — all verified against live upstream sources and existing proven prior art
- CSS-AST diff comparator design: LOW/novel by design — no direct prior art exists in Go; documented as a timeboxed spike, not a settled pattern
- Attribution mechanics (LICENSE/NOTICE/headers/README): HIGH — exact upstream copyright text verified directly from all 3 Marp repos
- Upstream-drift CI: MEDIUM — mechanism is sound and uses verified `gh` commands, but the exact YAML wasn't executed/tested in this research pass

**Research date:** 2026-07-20
**Valid until:** ~30 days for spec-suite/attribution findings (stable); ~7-14 days for exact Marp version numbers cited (marpit/marp-core/marp-cli all shipped within the prior two weeks — re-check pinned versions at implementation time, not just at planning time)
