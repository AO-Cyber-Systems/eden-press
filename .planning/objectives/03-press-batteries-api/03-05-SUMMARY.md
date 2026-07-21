---
objective: 03-press-batteries-api
trd: "05"
subsystem: press-batteries
tags: [chroma, goldmark-highlighting, syntax-highlighting, css-remap, core-07]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "03-01's press.Options{HighlightStyle,NoHighlight} frozen API-03 surface + chase/markdown.NewEngine(extra ...goldmark.Option) seam + chroma/v2 and goldmark-highlighting/v2 provisioned in go.mod; 03-02's go:embed'd themes.DefaultCSS/GaiaCSS/UncoverCSS (the real compiled theme CSS this TRD's remap table is derived from)"
provides:
  - "highlightOption(style string) goldmark.Option — reuses goldmark-highlighting/v2's NewHighlighting as the fenced-code NodeRenderer with chromahtml.WithClasses(true) wired through WithFormatOptions; the goldmark.Option 03-09 folds into pressExtraOpts"
  - "remapHLJS(html string) string — bounded post-format string pass rewriting chroma's short-code span classes (kd, s2, nv, ...) to the .hljs-* names the bundled themes' CSS targets, derived from themes/{default,gaia,uncover}.css, leaving unmapped classes intact"
  - "hljsClassRemap map[string]string — the grounded chroma-short-code -> .hljs-* table, every value proven present in the acquired theme CSS via TestRemapGrounded"
affects: [03-08-sanitize-allowlist, 03-09-capstone-render]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reuse over hand-roll: goldmark-highlighting/v2's NewHighlighting supplies the entire fenced-code NodeRenderer (language lookup, chroma.Coalesce, CSSWriter hook); this TRD's only bespoke code is the .hljs-* remap table + post-pass"
    - "chromahtml.WithClasses(true) MUST be injected via highlighting.WithFormatOptions(...) — it is a chroma HTML-formatter option, not a top-level goldmark-highlighting option"
    - "Ground a derived table against ACQUIRED data (grep over the real compiled CSS / the embedded Go string constants), never memory/recall — TestRemapGrounded asserts every hljsClassRemap value appears in themes.DefaultCSS+GaiaCSS+UncoverCSS"
    - "Post-format string pass over class=\"...\" attribute values (regexp capture + strings.Fields token rewrite), applied strictly AFTER rendering — never a second parse, preserving the one-parse-two-sinks invariant (chase.Render/03-04)"
    - "chroma's chosen style has ZERO effect on which CSS class a token receives under WithClasses(true) — class resolution is purely a function of chroma.StandardTypes (token type), confirmed against chroma/v2@v2.27.0's formatters/html source; style is safe to wire through unconditionally without needing to assert on colors"

key-files:
  created:
    - press/highlight.go
    - press/highlight_remap.go
    - press/highlight_test.go
  modified: []

key-decisions:
  - "hljsClassRemap intentionally leaves chroma's Punctuation (\"p\"), plain-identifier (\"nx\"), whitespace (\"w\") short codes, and chroma's own structural wrapper classes (\"chroma\", \"line\", \"cl\", ...) UNMAPPED — remapHLJS passes any class not in the table through untouched (research anti-pattern: never drop a token's class outright)."
  - "chroma.StringInterpol (\"si\") maps to \"hljs-subst\" rather than \"hljs-string\" — mirrors the real distinction highlight.js itself draws between a string's literal text and a nested substitution expression; \"subst\" is one of the acquired .hljs-* names (present in themes/gaia.css)."
  - "The TRD's illustrative grounding regex (`grep -o '\\.hljs-[a-z-]+' themes/*.css`) truncates real selectors containing an underscore (`.hljs-built_in` -> `.hljs-built`) — the production grounding test (TestRemapGrounded) and its `hljsSelectorPattern` use the corrected `[a-zA-Z_-]+` character class instead. Documented as a deviation (Rule 1 — bug in the TRD's own illustrative example, not in intent); the underlying acquisition principle (derive from real CSS, not memory) is unchanged and, if anything, more faithfully honored by the fix."

patterns-established:
  - "Grounding tests for any derived lookup table: assert every table VALUE appears in the acquired source-of-truth data (here: the union of all three bundled themes' compiled CSS), not just that the table compiles."

requirements-completed: [CORE-07]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 15min
completed: 2026-07-21
---

# Objective 3 TRD 05: Chroma Syntax Highlighting + CSS-Grounded .hljs Remap Summary

**CORE-07 syntax highlighting reusing goldmark-highlighting/v2 (chromahtml.WithClasses(true)) with a bounded chroma-short-class -> `.hljs-*` reconciliation table DERIVED from the actual compiled theme CSS (themes/default.css, gaia.css, uncover.css), not from memory.**

## Performance

- **Duration:** ~15 min (baseline `9328d2c` 09:56:27 -> Task 2 commit `cf3df93` 10:11:31, local time; both task commits below)
- **Started:** 2026-07-21T13:56:27Z
- **Completed:** 2026-07-21T14:11:31Z
- **Tasks:** 2/2 complete
- **Files modified:** 3 (all newly created)

## Accomplishments
- `highlightOption(style string) goldmark.Option`: reuses `github.com/yuin/goldmark-highlighting/v2`'s `NewHighlighting` as the fenced-code-block `NodeRenderer` (no hand-rolled renderer written), wiring `chromahtml.WithClasses(true)` through `highlighting.WithFormatOptions` so chroma emits `<span class="kd">`-style short classes instead of inline `style="..."` attributes; `style` (from `press.Options.HighlightStyle`) is forwarded via `highlighting.WithStyle` when non-empty, otherwise the library's own `github` default applies.
- `remapHLJS(html string) string` + `hljsClassRemap map[string]string`: a bounded post-format string pass (regex-captured `class="..."` attribute, token-by-token rewrite via `strings.Fields`/`strings.Join`) that rewrites every chroma short code present in the table to its `.hljs-*` counterpart, leaving unmapped classes (`p`, `nx`, `w`, chroma's own `chroma`/`line`/`cl` wrapper classes) untouched.
- The remap table is GROUNDED, not recalled: every value in `hljsClassRemap` is proven (via `TestRemapGrounded`) to appear in the real, acquired `.hljs-*` selector set extracted from `themes.DefaultCSS + themes.GaiaCSS + themes.UncoverCSS` (03-02's compiled `go:embed` output) — 36 distinct `.hljs-*` selectors acquired across the three bundled themes; the table maps chroma's ~45 relevant short codes onto the subset those themes actually style (keywords, names/types, literals/numbers, strings, operators, comments, and Generic diff-style subtypes).
- Confirmed empirically (via a disposable, never-committed spike test against the real `press/` package's already-resolved deps) that chroma's chosen style has zero effect on which class a token resolves to under `WithClasses(true)` — class resolution is purely `chroma.StandardTypes`-driven — which is why `TestHighlightStyleOmittable` tests the option's omittability/wiring rather than attempting to detect a style-dependent color difference that doesn't exist in classed output.
- One-parse invariant preserved: `remapHLJS` is a pure string->string rewrite over already-rendered HTML, run strictly after `engine.Convert`, never re-parsing (consistent with chase.Render/03-04's "one-parse-two-sinks").

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Wire goldmark-highlighting/v2 with class-based chroma output | `go test ./press/ -run 'TestHighlightClasses\|TestHighlightStyleOmittable' -v && gofmt -l press/highlight.go` | 0 | PASS |
| 2: Derive + apply the chroma->.hljs-* remap from the acquired theme CSS | `go test ./press/ -run 'TestHLJSRemapAppliesToKnownTokens\|TestRemapGrounded\|TestRemapUnknown' -v && grep -oh '\.hljs-[a-zA-Z_-]\+' themes/*.css \| sort -u \| head` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`), RED test then GREEN implementation:

1. **Task 1 RED — failing test for highlightOption class-based chroma wiring** - `79837f0` (test)
2. **Task 1 GREEN — wire goldmark-highlighting/v2 with class-based chroma output** - `2849ff9` (feat)
3. **Task 2 RED — failing tests for chroma-to-hljs remap (grounded/unknown)** - `eb0fee5` (test)
4. **Task 2 GREEN — derive + apply chroma-to-hljs remap from acquired theme CSS** - `cf3df93` (feat)

_Both tasks are `tdd="true"`; compile-failure RED (`undefined: highlightOption` / `undefined: remapHLJS`, `undefined: hljsClassRemap`) confirmed before each GREEN implementation — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` | 0 | PASS |
| gofmt | `gofmt -l press/highlight.go press/highlight_remap.go press/highlight_test.go` | 0 (no output) | PASS |
| Obj-1 corpus/cssdiff | `go test ./conformance/...` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/... -v` (includes `TestGrepGate`) | 0 | PASS |
| no-chromedp invariant | `go list -deps ./press/... \| grep -c chromedp` | 0 (count) | PASS |
| addlicense -check | `addlicense -check press/highlight.go press/highlight_remap.go press/highlight_test.go` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./press/ -run 'TestHighlightClasses\|TestHighlightStyleOmittable' -v` | 1 (compile failure: undefined `highlightOption`) | FAIL (correct) |
| GREEN (Task 1) | `go test ./press/ -run 'TestHighlightClasses\|TestHighlightStyleOmittable' -v` | 0 (both tests PASS) | PASS (correct) |
| RED (Task 2) | `go test ./press/ -run 'TestHLJSRemapAppliesToKnownTokens\|TestRemapGrounded\|TestRemapUnknown' -v` | 1 (compile failure: undefined `remapHLJS`/`hljsClassRemap`) | FAIL (correct) |
| GREEN (Task 2) | `go test ./press/ -run 'TestHLJSRemapAppliesToKnownTokens\|TestRemapGrounded\|TestRemapUnknown' -v` | 0 (all 3 tests PASS) | PASS (correct) |

## Grounding evidence (Open Question #3)

Corrected grounding command actually used (widened character class — see Deviations):

```sh
grep -oh '\.hljs-[a-zA-Z_-]\+' themes/*.css | sort -u
```

Yields 36 distinct selectors across the three bundled themes (union): `addition, attr, attribute, built_in, bullet, class, code, comment, deletion, doctag, emphasis, formula, keyword, link, literal, meta, name, number, operator, params, quote, regexp, section, selector-attr, selector-class, selector-id, selector-pseudo, selector-tag, string, strong, subst, symbol, tag, template-tag, template-variable, title, type, variable`. Every `hljsClassRemap` value maps into this set — proven programmatically by `TestRemapGrounded` against `themes.DefaultCSS + themes.GaiaCSS + themes.UncoverCSS` (the live `go:embed` constants, not the static file grep, so the grounding is re-verified on every `go test` run, not just once at authoring time).

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (both tasks' RED->GREEN cycles completed on the first implementation attempt; the only correction was to a TRD illustrative regex — see Deviations — not to implementation logic)
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 03-05-TRD.md frontmatter: reuse-not-hand-roll confirmed via `class="chroma"` wrapper assertion; grounded remap confirmed via `TestRemapGrounded`; post-format string-pass-only confirmed by `remapHLJS`'s implementation and doc comments; self-contained/omittable option confirmed via `TestHighlightStyleOmittable`'s bare-engine case)
- **Gate failures:** None

## Files Created/Modified
- `press/highlight.go` - `highlightOption(style string) goldmark.Option`, reusing `goldmark-highlighting/v2`'s `NewHighlighting` with `chromahtml.WithClasses(true)` via `WithFormatOptions`
- `press/highlight_remap.go` - `hljsClassRemap` (grounded chroma-short-code -> `.hljs-*` table), `classAttrPattern`, `remapHLJS(html string) string` (bounded post-format string pass)
- `press/highlight_test.go` - `TestHighlightClasses`, `TestHighlightStyleOmittable`, `TestHLJSRemapAppliesToKnownTokens`, `TestRemapGrounded`, `TestRemapUnknown` (Test-list cases 1-5) + `hljsSelectorPattern`/`goCodeFence` test fixtures

## Decisions Made
- `highlightOption("")` omits `highlighting.WithStyle` entirely rather than passing an explicit press-local default, deliberately reusing goldmark-highlighting's own `NewConfig()` default (`"github"`) — no new default style is invented.
- The remap operates on the `class="..."` HTML attribute via a single bounded regex + token-list rewrite, not a full HTML re-parse — kept deliberately minimal and reviewable, and safe under the one-parse-two-sinks invariant.
- Generic/diff-style chroma subtypes (`gd`, `gi`, `gh`, `gu`, `ge`, `gs`, `gp`) ARE included in the table (mapped to `hljs-deletion`/`hljs-addition`/`hljs-section`/`hljs-emphasis`/`hljs-strong`/`hljs-meta`) since those `.hljs-*` names are present in the acquired CSS and a diff-fenced block is a realistic input; chroma's own structural wrapper classes and plain-identifier/punctuation/whitespace codes are deliberately excluded (see key-decisions).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TRD's illustrative grounding regex truncates real underscore-containing selectors**
- **Found during:** Task 2, RED-phase design of `TestRemapGrounded`
- **Issue:** 03-05-TRD.md's `<codebase_examples>` and `<tasks>` verify steps specify `grep -o '\.hljs-[a-z-]\+' themes/*.css` for deriving the ground-truth `.hljs-*` set. That character class excludes the underscore, so the real, verbatim selector `.hljs-built_in` (present in both `themes/default.css` and `themes/uncover.css`) is truncated to `.hljs-built` in the grep output — a bug in the TRD's own illustrative command, not in the acquisition principle it expresses (derive from real CSS, never from memory).
- **Fix:** Used the corrected character class `[a-zA-Z_-]+` both in the production grounding test's `hljsSelectorPattern` (`press/highlight_test.go`) and when re-deriving the ground set for this SUMMARY — verified directly against the raw CSS that `.hljs-built_in` genuinely appears verbatim (not a chroma/grep artifact). `hljsClassRemap["nb"]` and `["bp"]` correctly map to `"hljs-built_in"` (the full, correct selector), matched against the corrected ground set.
- **Files modified:** press/highlight_test.go (test-only; no production logic changed)
- **Verification:** `TestRemapGrounded` passes; re-ran both the TRD's literal (truncating) grep and the corrected grep side-by-side (see Grounding evidence above) confirming the discrepancy and the fix.
- **Committed in:** `eb0fee5` (Task 2 RED commit, which introduced `hljsSelectorPattern`)

---

**Total deviations:** 1 auto-fixed (Rule 1 — bug in the TRD's own illustrative regex example; no change to implementation scope or CORE-07's requirements)
**Impact on plan:** None on scope. The grounding principle (derive from acquired CSS, not memory) is honored more faithfully by the fix, since the original literal command would have silently under-verified a real selector.

## Issues Encountered
None beyond the single auto-fixed deviation above, resolved within Task 2's TDD cycle before commit.

## User Setup Required
None — no external service configuration required. `chroma/v2` and `goldmark-highlighting/v2` were already provisioned into `go.mod` by 03-01; this TRD adds no new dependencies and does not touch `go.mod`.

## Next Objective Readiness
- `highlightOption(style string) goldmark.Option` is ready for 03-09's capstone to fold into `pressExtraOpts`, sourcing `style` from `press.Options.HighlightStyle` and omitting the option entirely when `press.Options.NoHighlight` is set (mirrors `TestHighlightStyleOmittable`'s bare-engine case).
- `remapHLJS(html string) string` is ready for 03-09 to apply as a post-render string pass over the final HTML output, before (or independent of) sanitization.
- The resulting `<span class="hljs-*">` markup is documented here for 03-08's sanitize allow-list: the classes emitted are exactly the acquired set in `hljsClassRemap`'s values, plus any unmapped chroma short codes / chroma structural wrapper classes (`chroma`, `line`, `cl`, ...) that pass through untouched — 03-08 should allow both the `.hljs-*` set and chroma's own wrapper classes on `<span>`/`<pre>`/`<code>`.

## Self-Check: PASSED

All claimed files confirmed present on disk; all four task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: press/highlight.go
- FOUND: press/highlight_remap.go
- FOUND: press/highlight_test.go
- FOUND: .planning/objectives/03-press-batteries-api/03-05-SUMMARY.md
- FOUND commit: 79837f0 (Task 1 RED)
- FOUND commit: 2849ff9 (Task 1 GREEN)
- FOUND commit: eb0fee5 (Task 2 RED)
- FOUND commit: cf3df93 (Task 2 GREEN)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
