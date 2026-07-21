# State Archive

Append-only log. Written by df-tools `add-decision` and `record-metric`.
STATE.md stays lean; this file grows over time.

## Decisions

- *()*
- [Objective 00-conformance-corpus-attribution]: go.mod/go.sum owned by 00-01 as single source of truth; downstream additive-only (go mod download, never go mod tidy)
- [Objective 00-conformance-corpus-attribution]: Eden headers = 2026 AO Cyber Systems; verbatim Marp assets preserve 2018 marp-team copyright; addlicense two-template enforcement
- [Objective 01-chase-framework]: chase/theme Pack pipeline: Pass{Name,Run}+RunPasses shared runner reused at both Tier-1 Load and Tier-2 Pack; own repadding serializers (declarationText) added since Stylesheet.String()/Declaration.String() don't repad function-argument commas in values
- [Objective 01-chase-framework]: Only @import-theme is recursively resolved (cycle-safe via per-branch copied visited map); plain @import atoms left unresolved (no filesystem/network layer in scope); the :marpit-container-alone AdvancedBackgroundCSS gap rule is deliberately left un-scoped rather than modifying the locked chase/theme/selector package

## Performance Metrics

| Objective | Duration | Tasks | Files |
|-----------|----------|-------|-------|
| Objective 00-conformance-corpus-attribution P01 | 6min | 3 tasks | 19 files |
| Objective 01-chase-framework P04 | 22min | 3 tasks | 11 files |

