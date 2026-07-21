---
status: passed
objective: 3
verified: 2026-07-21
score: 12/12 requirements, 4/4 success criteria
---

# Objective 3 Verification — press/ Batteries + Public API

**Verdict: PASSED.** Verified against the actual merged codebase on `main` (all 9 job commits merged: `03-01` through `03-09`, reconciled at `66e0ce3`). Every gate — `gofmt`, `go build ./...`, `go vet ./...`, `go test ./...` (fresh, `-count=1`, all 19 packages), `bash scripts/check-no-chromedp.sh` (PASS, exit 0), the Obj-1 conformance suite (`go test ./conformance/...`), and the Obj-2 `TestGrepGate` — passes clean. All three named capstone tests pass: `TestCapstoneAllThemesEveryBattery` (subtests default/gaia/uncover), `TestOneParseInvariant`, `TestCapstonePressOnlyConsumer` (confirmed external `press_test` package importing only `press/`).

## Gate results (run fresh, this session)

| Gate | Command | Result |
|------|---------|--------|
| Format | `gofmt -l press chase profiles` | Empty output — exit 0 |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all pkgs) | `go test ./... -count=1` | 19 packages, all `ok` (0 failures) |
| No-chromedp CI gate | `bash scripts/check-no-chromedp.sh` | `PASS: no chromedp in the press/chase/profiles dependency closure.` exit 0 |
| No-chromedp raw grep | `go list -deps ./press/... \| grep -i chromedp` | empty match, grep exit 1 (nothing found) |
| Obj-1 conformance | `go test ./conformance/... -count=1` | all `ok` (corpus, cssdiff, htmldiff, report, runner) |
| Obj-2 grep-gate | `go test ./profiles/slides/... -run TestGrepGate -v` | `--- PASS: TestGrepGate` |
| Capstone (all 3 themes) | `go test ./press/... -run TestCapstoneAllThemesEveryBattery -v` | PASS, subtests default/gaia/uncover all PASS |
| One-parse invariant | `go test ./press/... -run TestOneParseInvariant -v` | PASS |
| Press-only consumer gate | `go test ./press/... -run TestCapstonePressOnlyConsumer -v` | PASS; confirmed `package press_test` importing only `github.com/AO-Cyber-Systems/eden-press/press` |

## Requirements coverage

| REQ | Source TRD | Description | Evidence | Status |
|-----|-----------|-------------|----------|--------|
| CORE-01 | 03-02 | 3 bundled themes via go:embed, MIT/Marp headers preserved | `themes/embed.go` (`//go:embed default.css/gaia.css/uncover.css`); `themes/default.css`, `gaia.css`, `uncover.css` each carry Marp's own leading `/*! @theme … */` block (verbatim, not an Eden header); `.github/workflows/ci.yml` addlicense `-ignore 'themes/**'`; `press/themes/themes_test.go` (`TestEmbeddedThemesLoad`) | ✅ |
| CORE-02 | 03-07 | `size`/`math` global directives | `chase/directive/directives.go` `CoerceGlobal` — explicit `case "size"` / `case "math"` passthrough cases (lines 60-74), mirroring the existing style/lang pattern | ✅ |
| CORE-03 | 03-03 | GFM tables + strikethrough as `<s>` + hard-breaks | `press/strikethrough.go` (`sRenderer` registers `extast.KindStrikethrough` at priority 100, below goldmark's own 500, so `<s>` wins over `<del>`); `press/gfm_verify_test.go` `TestGFMTableRenders`, `TestHardWrapRendersBr` | ✅ |
| CORE-04 | 03-03 | Heading slugs h1-h6 | `press/gfm_verify_test.go` `TestSlugHeadingID`, `TestSlugH6`, `TestSlugDedup` (dedup suffix `-1` verified) | ✅ |
| CORE-05 | 03-08 | bluemonday sanitize, Marp xss parity, GFM disallowed-tag filter, always-on | `press/sanitize/policy.go` `Policy()` (blank-slate bluemonday allow-list) + `Sanitize()` (SVG case-restoration wrapper); `press/sanitize/tagfilter.go` `GFMDisallowedTags` (all 9: script/iframe/style/textarea/title/xmp/noembed/noframes/plaintext) via `SkipElementsContent`; `press/sanitize/adversarial_test.go` — 18 test funcs (11 "Preserve*" battery-output-survives tests + 7 "Adversarial*" XSS-neutralization tests incl. nested-obfuscated-payload, javascript: URI, on-handler variants); strip-vs-escape deviation explicitly documented in code comment | ✅ |
| CORE-06 | 03-04 | Emoji shortcode + unicode → twemoji, no JS | `press/emoji.go` (reuses `goldmark-emoji`'s `emoji.New(emoji.WithRenderingMethod(emoji.Twemoji), ...)`); `press/emoji_unicode.go` (`unicodeEmojiParser`/`unicodeEmojiExtender`, bespoke InlineParser reverse-indexing `definition.Github()` into a rune→`*definition.Emoji` map, emits the same `east.Emoji` node); `press/emoji_test.go` | ✅ |
| CORE-07 | 03-05 | chroma highlight + chroma→hljs remap | `press/highlight.go` (reuses `goldmark-highlighting/v2.NewHighlighting` + `chromahtml.WithClasses(true)`); `press/highlight_remap.go` `hljsClassRemap` table + `remapHLJS()` bounded post-format string pass; `press/highlight_test.go` `TestRemapGrounded` proves every `.hljs-*` name is grep-acquired from `themes/*.css`, not hand-typed | ✅ |
| CORE-08 | 03-06 | math `$…$`/`$$…$$` → MathML + construct-detection → PNG fallback | `press/math/math.go` (bespoke `$`-trigger `mathInlineParser` + `mathNode` + `mathRenderer`); `press/math/detect.go` `fallbackRE` (`\tag`/`\label`/`\begin{aligned\|align\|alignat\|cases\|array}`) + `needsFallback()`; `press/math/fallback.go` `renderFallbackIMG`/`safeRasterPNG` (go-latex `drawtex/drawimg`, base64 PNG data-URI, panic-recovering). BASELINE scopes (documented, in-scope-as-specified per objective spec, not gaps): PNG-only (drawtex has no SVG canvas) and go-latex's raster cannot render the `aligned`-family it is routed FOR — both explicitly noted in code comments as Objective-8 hardening targets | ✅ |
| CORE-09 | 03-07 | auto-fit markers | `press/autofit.go` (`autofitTransformer` sets `data-auto-scaling="fit"` on a `# <!--fit-->` heading; `autofitShrinkNode`/`wrapWithShrinkMarker` wraps code/math blocks in `<div class="marp-fit-shrink">`); `press/autofit_test.go` | ✅ |
| API-01 | 03-09 | `press.Render(md,opts)→{HTML,CSS,Model,Comments,Meta}`, no Chrome | `press/press.go` `Render()` — one `parseWithEngine` call (verified countable via `TestOneParseInvariant`), forks to HTML sink + `model.Build` sink, sanitizes last, packs CSS, returns `Output{HTML,CSS,Model,Meta,Comments}`; never calls/modifies `chase/chase.go` | ✅ |
| API-02 | 03-09 | `go list -deps ./press/...` has no chromedp — CI-enforced | `scripts/check-no-chromedp.sh` (checks `./press/...`, `./chase/...`, `./profiles/...`); wired into `.github/workflows/ci.yml` step "Check no chromedp (API-02)" running `make check-no-chromedp`; `Makefile` `check-no-chromedp` target; confirmed exit 0 this session | ✅ |
| API-03 | 03-01 | Stable, documented Options/Output | `press/options.go` `Options` struct (Theme/Profile/InlineSVG/MathMode/NoHighlight/HighlightStyle/Sanitize, zero-value = Marp-Core defaults, documented field-by-field) + `Output` struct (HTML/CSS/Model/Meta/Comments); `press/doc.go` package doc; `press/options_test.go` `TestOptionsZeroValueIsMarpDefault`, `TestOutputZeroValueFields`, `TestOptionsOutputCompileFence` | ✅ |

**12/12 requirement IDs have a concrete, tested artifact on disk. Zero orphans** — every ID in `.planning/REQUIREMENTS.md`'s Objective-3 mapping (CORE-01..09, API-01..03) is claimed by exactly one TRD's frontmatter `requirements:` field (03-01 through 03-09) and traced above.

## Success criteria

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | `press.Render` returns `{HTML,CSS,Model,Comments,Meta}` for a deck exercising all 3 themes + every battery (tables, strikethrough-as-`<s>`, hard-breaks, slugs, emoji, chroma+hljs, math+MathML+PNG-fallback-routing, autofit markers) | ✅ | `TestCapstoneAllThemesEveryBattery` — deck fixture (`press/capstone_test.go`) exercises every battery in one deck, subtests pass for default/gaia/uncover |
| 2 | `go list -deps ./press/...` contains no chromedp — CI-enforced | ✅ | `scripts/check-no-chromedp.sh` + `.github/workflows/ci.yml` step, confirmed exit 0 |
| 3 | HTML sanitization matches Marp's xss allow-list behaviorally (strip-vs-escape documented), GFM disallowed-tag filter, adversarial round-trip suite, directive/comment path as its own trust boundary | ✅ | `press/sanitize/policy.go` + `tagfilter.go` + `adversarial_test.go` (18 tests); `press/autofit_test.go`'s `TestAdversarialCommentNotUsedAsFitMarker` (in sanitize suite) validates the comment-parsing trust boundary specifically |
| 4 | Options/Output documented and stable enough a consumer imports ONLY `press/` (Objective-7-begin gate) | ✅ | `TestCapstonePressOnlyConsumer` — external `press_test` package compiles and passes importing only `github.com/AO-Cyber-Systems/eden-press/press` |

## Notes (non-blocking)

- **Documented BASELINE scopes, not gaps** (per objective spec): CORE-08's math fallback is PNG-only (go-latex's `drawtex` has no SVG canvas — the requirement's "SVG/PNG" framing is corrected in-code) and go-latex's raster cannot yet render the `aligned`-family construct it is routed to as a fallback FOR. Both are explicitly called out in `press/math/fallback.go`/`detect.go` comments and in `03-06-SUMMARY.md` as Objective 8's hardening job (KaTeX-parity + final fallback rule). Treated as in-scope-as-specified.
- **`.planning/REQUIREMENTS.md` checkbox drift (tracking-only, not a code gap):** the requirements checklist (lines 43-57) still shows `[ ]` (Pending) for CORE-02, CORE-05, CORE-07, CORE-09, API-01, and API-02, and the traceability table (lines 155-166) still lists them "Pending" — despite all six being fully implemented, tested, and merged (commits `4775168`, `159fb4f`, `459dfc3`, `da08cbf`). The `docs(03-XX): complete ... TRD` reconciliation commits for jobs 03-05, 03-07, 03-08, and 03-09 did not update `REQUIREMENTS.md`'s checkboxes/table the way 03-01 through 03-04 and 03-06's docs commits did. This is a documentation-reconciliation gap in the tracking artifacts, not a functional gap — the code, tests, and gates are all green as detailed in the table above. Recommend a follow-up `docs` commit to flip these six checkboxes and traceability rows to `[x]`/"Complete" so `REQUIREMENTS.md` matches reality.

## Gaps

None. All 12 requirements verified present with passing tests; all gates green; all 4 success criteria met.

---

*Verified: 2026-07-21*
*Verifier: Claude (verifier)*
