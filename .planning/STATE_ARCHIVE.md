# State Archive

Append-only log. Written by df-tools `add-decision` and `record-metric`.
STATE.md stays lean; this file grows over time.

## Decisions

- *()*
- [Objective 00-conformance-corpus-attribution]: go.mod/go.sum owned by 00-01 as single source of truth; downstream additive-only (go mod download, never go mod tidy)
- [Objective 00-conformance-corpus-attribution]: Eden headers = 2026 AO Cyber Systems; verbatim Marp assets preserve 2018 marp-team copyright; addlicense two-template enforcement
- [Objective 01-chase-framework]: chase/theme Pack pipeline: Pass{Name,Run}+RunPasses shared runner reused at both Tier-1 Load and Tier-2 Pack; own repadding serializers (declarationText) added since Stylesheet.String()/Declaration.String() don't repad function-argument commas in values
- [Objective 01-chase-framework]: Only @import-theme is recursively resolved (cycle-safe via per-branch copied visited map); plain @import atoms left unresolved (no filesystem/network layer in scope); the :marpit-container-alone AdvancedBackgroundCSS gap rule is deliberately left un-scoped rather than modifying the locked chase/theme/selector package
- [Objective 01-chase-framework]: marp-gfm-table passes outright via goldmark's stock GFM extension -- no Objective-3 battery needed, asserted explicitly as a decision record
- [Objective 01-chase-framework]: headingDivider display-value materialization fixed (Rule 1) at apply.go's data-attribute layer only, leaving the 01-02-locked expanded-range CoerceGlobal contract untouched
- [Objective 01-chase-framework]: Rasterization proof strategy: human-verify checkpoint (screenshot) now for Objective 1; deterministic headless-Chrome pixel-diff deferred to Objective 5 where chromedp lives, keeping chase/press Chrome-free through Objectives 1-4
- [Objective 02-model-profile]: chase.Render composes markdown.Parse+RenderDoc+model.Build+profile-parameterized theme.Pack as the one-parse-two-sinks internal entrypoint (MODEL-02); Objective 2 complete
- [Objective 03-press-batteries-api]: CORE-07 chroma highlighting reuses goldmark-highlighting/v2 (chromahtml.WithClasses(true)) with the ONE bespoke piece being a chroma-short-class to .hljs-* remap table GROUNDED in the acquired themes/{default,gaia,uncover}.css (36 selectors), not recalled from memory; corrected the TRD's illustrative grounding regex ([a-z-]+ -> [a-zA-Z_-]+) to avoid truncating .hljs-built_in

## Performance Metrics

| Objective | Duration | Tasks | Files |
|-----------|----------|-------|-------|
| Objective 00-conformance-corpus-attribution P01 | 6min | 3 tasks | 19 files |
| Objective 01-chase-framework P04 | 22min | 3 tasks | 11 files |
| Objective 01-chase-framework P08 | 45min | 3 tasks | 6 files |
| Objective 02-model-profile P04 | 14min | 3 tasks | 4 files |
| Objective 03-press-batteries-api P01 | ~8min | 3 tasks | 7 files |
| Objective 03-press-batteries-api P05 | 15min | 2 tasks | 3 files |

