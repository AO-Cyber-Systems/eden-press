---
objective: 03-press-batteries-api
job: "04"
subsystem: press
tags: [goldmark, emoji, twemoji, inline-parser, ast, no-js]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "03-01's press.Options/Output frozen API surface, chase/markdown.NewEngine(extra ...goldmark.Option) composition seam, and the pre-provisioned github.com/yuin/goldmark-emoji v1.0.6 indirect dependency + deps_spike_test.go compat proof"
provides:
  - "press/emoji.go: emojiOption()/emojiOptionWithTwemoji(cfg) goldmark.Option wiring github.com/yuin/goldmark-emoji's emoji.New(WithRenderingMethod(Twemoji), WithTwemojiTemplate(...)) for the shortcode half (\":smile:\")"
  - "press.TwemojiOptions{Base,Ext} + DefaultTwemojiOptions() — Go-level base/ext configurability matching Marp's twemoji base/ext contract, for 03-09 to wire into press.Options later"
  - "press/emoji_unicode.go: unicodeEmojiParser (halfspace-triggered parser.InlineParser) + unicodeEmojiExtender, mapping literal unicode emoji runes typed in prose to the SAME east.Emoji AST node goldmark-emoji's own renderer already handles (no second NodeRenderer, no hand-rolled <img>, no hand-rolled shortcode table)"
affects: [03-09-options-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reuse-first extension composition: the shortcode half is 100% github.com/yuin/goldmark-emoji (a goldmark.Extender) with zero bespoke rendering code; only the unicode-literal half required new code, and even that reuses goldmark-emoji's east.Emoji AST node + its emojiHTMLRenderer, contributing only a parser.InlineParser that resolves runes to a *definition.Emoji and emits east.NewEmoji(...)"
    - "Halfspace trigger idiom for non-punctuation inline matches: registering Trigger() on raw UTF-8 lead bytes (all >= 0x80) never fires mid-line because goldmark's parser.go scan loop only consults registered InlineParsers when isPunct(c) || isSpace(c) || i==0 (util.IsPunct's generated table is 0 for all bytes >= 128) — the correct mechanism, mirrored exactly from goldmark's own extension/linkify.go linkifyParser, is Trigger() []byte{' '}, peeking past the boundary char and preserving it via ast.MergeOrAppendTextSegment(parent, segment.WithStop(segment.Start+1)) before block.Advance"
    - "Seed-list reverse index for unicode-literal lookup: definition.Emojis exposes only Get/Add/Clone (no enumeration method), so unicodeEmojiIndex is built by calling definition.Github().Get(name) against a curated list of ~46 common shortnames, guaranteeing pointer/data identity with what the shortcode path resolves — documented as a baseline (not exhaustive ZWJ/skin-tone) mapping per the TRD's own error_recovery guidance"
    - "Longest-rune-sequence match (via unicode/utf8.DecodeRune, tried longest-to-shortest) to correctly handle multi-rune Github entries (e.g. \"heart\" = base codepoint + U+FE0F variation selector) without special-casing"

key-files:
  created:
    - press/emoji.go
    - press/emoji_unicode.go
    - press/emoji_test.go
  modified: []

key-decisions:
  - "Trigger() is registered on a single literal space byte (the 'halfspace' idiom), NOT on the raw UTF-8 lead bytes of emoji runes as the TRD's error_recovery text suggested — verified via direct reading of goldmark v1.8.4's parser.go scan loop and util.IsPunct's generated table that non-ASCII lead bytes (all >= 0x80) never satisfy isPunct(c) || isSpace(c), so a lead-byte-registered InlineParser would only ever fire at true line-start (i==0), never mid-paragraph. Confirmed as the intended pattern by cross-referencing goldmark's own extension/linkify.go, which solves the identical word-boundary problem the same way."
  - "unicodeEmojiExtender{} is added to emojiOptionWithTwemoji in Task 2 (editing press/emoji.go, not listed in Task 2's <files> tag) rather than kept as a separate wiring step, per the task's own <action> text: \"Register it via a small unicodeEmojiExtender added in emojiOption()\" — Task 1 intentionally ships emoji.go without this reference so it stands alone and compiles/tests green before unicodeEmojiExtender exists."
  - "Reverse index is a curated ~46-entry seed list (common prose emoji: smilies, gestures, hearts, symbols) rather than an exhaustive walk of definition.Github()'s ~1870 entries, because the Emojis interface exposes no enumeration method — documented as a baseline limitation per the TRD's own guidance not to block CORE-06 on exhaustive ZWJ/skin-tone coverage."

requirements-completed: [CORE-06]

# Verification evidence
verification:
  gates_defined: 8
  gates_passed: 8
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 24min
completed: 2026-07-21
---

# Objective 03 TRD 04: Native Emoji (Shortcode + Unicode-Literal to Twemoji, No JS) Summary

**CORE-06 native emoji: the shortcode half (`:smile:`) is 100% reused from `github.com/yuin/goldmark-emoji` v1.0.6 as a `goldmark.Extender`; the unicode-literal half (typed 😄) is a small bespoke halfspace-triggered `parser.InlineParser` that resolves runes to the same `*definition.Emoji` and emits the identical `east.Emoji` AST node goldmark-emoji's own renderer already handles — proven by both paths producing byte-identical `<img>` tags, with zero JavaScript.**

## Performance

- **Duration:** 24 min (Task 1 commit 10:19:30 -> Task 2 commit 10:21:13 -> gates/summary, local time)
- **Started:** 2026-07-21T14:19:30Z (approx, first commit)
- **Completed:** 2026-07-21 (this SUMMARY commit)
- **Tasks:** 2/2 complete
- **Files modified:** 3 (all newly created)

## Accomplishments

- **Task 1 (shortcode half, commit `6f81008`):** `press/emoji.go` wires `github.com/yuin/goldmark-emoji`'s `emoji.New(emoji.WithRenderingMethod(emoji.Twemoji), emoji.WithTwemojiTemplate(cfg.twemojiTemplate()))` as a `goldmark.Extender`, reused verbatim — zero hand-rolled rendering logic. `press.TwemojiOptions{Base, Ext}` + `DefaultTwemojiOptions()` give Go-level configurability of the twemoji CDN base/extension (matching Marp's own base/ext contract), ready for 03-09 to surface through `press.Options`. `TestEmojiShortcode` and `TestEmojiBaseExt` prove `:smile:` renders a twemoji `<img class="emoji" ...>` with a fully custom base/ext when configured.
- **Task 2 (unicode-literal half, commit `e052a0b`):** `press/emoji_unicode.go` adds ONLY the piece goldmark-emoji lacks: a rune-sequence-keyed reverse index (`buildUnicodeEmojiIndex`, seeded from `definition.Github().Get(name)` since the `Emojis` interface has no enumeration method) feeding `unicodeEmojiParser`, a halfspace-triggered (`Trigger() []byte{' '}`) `parser.InlineParser` registered via `unicodeEmojiExtender`. It performs a longest-rune-sequence match (`longestUnicodeEmojiMatch`, via `utf8.DecodeRune`) to correctly handle multi-rune entries (e.g. "heart" = 2 runes), then emits `east.NewEmoji(shortName, entry)` — the exact same AST node type goldmark-emoji's shortcode path produces, so the SAME `emojiHTMLRenderer` renders both. `TestEmojiUnicode` and `TestEmojiMixed` prove literal 😄/🎉 and `:smile:`/`:tada:` render byte-identical `<img>` tags in the same document, in the correct surrounding-text order.
- **Architectural correction found via investigation, not trial-and-error:** the TRD's error_recovery text suggested triggering on raw UTF-8 lead bytes of emoji runes; reading goldmark's actual scan loop (`parser/parser.go`, gated by `isPunct(c) || isSpace(c) || i==0`) showed this would never fire mid-paragraph (non-ASCII lead bytes are all >= 0x80, never punct/space per `util.IsPunct`'s generated table). Adopted the halfspace idiom instead, matching goldmark's own `extension/linkify.go` pattern exactly — caught and corrected before any code was written.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: shortcode half (goldmark-emoji reuse) | `go test ./press/ -run TestEmojiShortcode\|TestEmojiBaseExt -v` | 0 | PASS |
| 2: unicode-literal half (bespoke InlineParser) | `go test ./press/ -run TestEmoji -v` (all 5 emoji tests) | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: wire goldmark-emoji Twemoji shortcode rendering** - `6f81008` (feat)
2. **Task 2: unicode-literal emoji InlineParser** - `e052a0b` (feat)

_Note: TRD is `type: tdd`; RED (compile failure against undefined symbols: `emojiOption`/`TwemojiOptions` for Task 1, `unicodeEmojiParser`/`unicodeEmojiExtender`/`buildUnicodeEmojiIndex` for Task 2) confirmed before each GREEN implementation — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l press/emoji.go press/emoji_unicode.go press/emoji_test.go` | 0 (no output) | PASS |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test | `go test ./...` (16 packages) | 0 | PASS |
| addlicense (MIT header) | `addlicense -check press/emoji.go press/emoji_unicode.go press/emoji_test.go` | 0 | PASS |
| Obj-1 corpus/cssdiff | `go test ./conformance/... -v` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS |
| no-chromedp invariant | `go list -deps ./press/... \| grep -c chromedp` -> 0 | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./press/... -v` | 1 (compile failure: undefined `emojiOption`, `TwemojiOptions`, `DefaultTwemojiOptions`) | FAIL (correct) |
| GREEN (Task 1) | `go test ./press/ -run TestEmojiShortcode\|TestEmojiBaseExt -v` | 0 | PASS (correct) |
| RED (Task 2) | `go test ./press/... -v` | 1 (compile failure: undefined `unicodeEmojiParser`, `unicodeEmojiExtender`, `buildUnicodeEmojiIndex`) | FAIL (correct) |
| GREEN (Task 2) | `go test ./press/ -run TestEmoji -v` | 0 (all 5 emoji tests, first attempt, zero debug iterations) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (both RED->GREEN cycles passed on the first implementation attempt; no post-commit debugging required)
- **Must-haves verified:** CORE-06's core claim confirmed by test — `:smile:`/`:tada:` (shortcode) and literal 😄/🎉 (unicode) render byte-identical `<img class="emoji" ...twemoji...>` tags via `TestEmojiMixed` and `TestEmojiUnicode`, with zero JavaScript in the output (no `<script>`, no client-side lib)
- **Gate failures:** None

## Files Created/Modified

- `press/emoji.go` - `TwemojiOptions{Base,Ext}`, `DefaultTwemojiOptions()`, `(TwemojiOptions).twemojiTemplate()`, `emojiOption()`, `emojiOptionWithTwemoji(cfg)` — shortcode half, wraps `emoji.New(...)` verbatim
- `press/emoji_unicode.go` - `unicodeEmojiShortnames` (seed list), `buildUnicodeEmojiIndex()`, `longestUnicodeEmojiMatch()`, `unicodeEmojiParser` (`parser.InlineParser`), `unicodeEmojiExtender` (`goldmark.Extender`) — unicode-literal half
- `press/emoji_test.go` - `renderEmoji`/`renderEmojiWithTwemoji`/`extractImgTag(From)` test helpers + `TestEmojiShortcode`, `TestEmojiBaseExt`, `TestEmojiUnicode`, `TestEmojiMixed`, `TestEmojiReverseIndex`

## Decisions Made

- Halfspace trigger (`Trigger() []byte{' '}`) instead of raw lead-byte triggers — see key-decisions above.
- `unicodeEmojiExtender{}` wired into `emojiOptionWithTwemoji` during Task 2's edit of `press/emoji.go`, per the task's own action text (Task 1 intentionally ships without it so it's self-contained and green on its own).
- Seed-list (not exhaustive) reverse index of ~46 common shortnames — documented baseline limitation, consistent with the TRD's own error_recovery guidance not to block CORE-06 on exhaustive ZWJ/skin-tone coverage.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TRD's error_recovery-suggested Trigger() mechanism (raw emoji lead bytes) would never fire mid-paragraph**
- **Found during:** Task 2, pre-implementation design/investigation (before any code written)
- **Issue:** The TRD's embedded guidance suggested basing `Trigger()` on "the set of distinct leading bytes across `definition.Github()`'s unicode entries" (raw UTF-8 lead bytes, e.g. 0xF0, 0xE2). Direct reading of goldmark v1.8.4's `parser/parser.go` scan loop showed inline parsers are only consulted when `isPunct(c) || isSpace(c) || i==0` is true, and `util.IsPunct`'s generated table returns 0 (false) for every byte >= 128 — meaning a lead-byte trigger would only ever match at true line-start, never for an emoji typed mid-sentence.
- **Fix:** Adopted the halfspace idiom instead: `Trigger() []byte{' '}`, peeking past the leading space/line-start boundary in `Parse()`, preserving the consumed space as its own text segment via `ast.MergeOrAppendTextSegment(parent, segment.WithStop(segment.Start+1))` before advancing past the matched emoji — the identical pattern goldmark's own `extension/linkify.go` `linkifyParser` uses for the same underlying reason.
- **Files modified:** press/emoji_unicode.go (as originally written; no rework needed since the correct approach was identified before implementation)
- **Verification:** `TestEmojiUnicode` and `TestEmojiMixed` confirm mid-sentence unicode emoji (`"Hello \U0001F604 world"`, `"hi :tada: \U0001F389 there"`) resolve correctly with surrounding text preserved in order.
- **Commit:** e052a0b (Task 2 commit — correct approach implemented directly, no separate fix commit needed)

**2. [Rule 2 - Missing functionality] `definition.Emojis` has no enumeration method; TRD's sketch implied walking all entries**
- **Found during:** Task 2, pre-implementation design
- **Issue:** The TRD's codebase-example sketch implied building a reverse index by iterating `definition.Github()`'s full unicode entry set, but the `Emojis` interface exposes only `Get(shortName)`, `Add(Emojis)`, `Clone() Emojis` — no enumeration/range method exists over the ~1870 generated entries.
- **Fix:** Built the index from a curated seed list of ~46 common prose shortnames (smilies, gestures, hearts, checks, symbols), resolved individually via `definition.Github().Get(name)` — guarantees pointer/data identity with what the shortcode parser resolves, and is explicitly documented in code comments as a baseline (not exhaustive) mapping, per the TRD's own error_recovery note that ZWJ/skin-tone long-tail coverage should not block CORE-06.
- **Files modified:** press/emoji_unicode.go
- **Verification:** `TestEmojiReverseIndex` confirms the index's `smile` entry is pointer-identical to `definition.Github().Get("smile")`'s result.
- **Commit:** e052a0b (Task 2 commit)

---

**Total deviations:** 2 (1 Rule 1 - Bug caught pre-implementation via investigation, 1 Rule 2 - missing functionality worked around with a documented baseline approach). Both resolved within Task 2's own design/TDD cycle, before the task commit — no scope creep, no change to CORE-06's shipped behavior or public surface.

## Issues Encountered

None beyond the two deviations above, both resolved before commit with zero post-commit fix cycles.

## User Setup Required

None - no external service configuration required. Default twemoji CDN base (`https://cdn.jsdelivr.net/gh/twitter/twemoji@latest/assets/72x72/`) is used unless `TwemojiOptions` is overridden (future 03-09 wiring point).

## Next Objective Readiness

- `press.TwemojiOptions`/`DefaultTwemojiOptions()`/`emojiOptionWithTwemoji(cfg)` are ready for 03-09 to surface through `press.Options` (self-hosted/air-gapped twemoji base/ext override), matching Marp's own base/ext contract per `.planning/research/FEATURES.md`.
- `unicodeEmojiExtender`'s seed-list approach is easily extendable (append more shortnames to `unicodeEmojiShortnames`) if a later objective needs broader unicode-literal coverage — no architectural change required.
- Remaining wave-2 battery TRDs (03-03, 03-05..03-08) are independent of this work and unblocked.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: press/emoji.go
- FOUND: press/emoji_unicode.go
- FOUND: press/emoji_test.go
- FOUND: .planning/objectives/03-press-batteries-api/03-04-SUMMARY.md
- FOUND commit: 6f81008 (Task 1)
- FOUND commit: e052a0b (Task 2)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
