---
objective: 03-press-batteries-api
trd: "08"
subsystem: sanitize
tags: [bluemonday, xss, html-sanitize, gfm-tagfilter, security]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "03-01's press/ skeleton + bluemonday v1.0.27 provisioned in go.mod; press.Options.Sanitize *bluemonday.Policy contract (nil = built-in always-on policy)"
provides:
  - "press/sanitize package: Policy() *bluemonday.Policy — the blank-slate, always-on Marp-v4-parity allow-list"
  - "press/sanitize.Sanitize(html string) string — the RECOMMENDED full pipeline (Policy().Sanitize + SVG case restoration); 03-09 MUST call this, not bare Policy().Sanitize()"
  - "press/sanitize.GFMDisallowedTags — the 9-tag hand-filter list (script/iframe/style/textarea/title/xmp/noembed/noframes/plaintext) passed to p.SkipElementsContent"
  - "23 tests (5 rule tests + 12 preservation tests + 6 adversarial tests) proving XSS neutralization and battery-output survival over the FINAL HTML string"
affects: ["03-09 (press.Render composition — applies sanitize.Sanitize as the absolute last step)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Blank-slate bluemonday.NewPolicy(): nothing survives unless explicitly AllowElements/AllowAttrs'd, so style and on* handlers require ZERO special-case code to exclude — they are simply never allow-listed"
    - "p.SkipElementsContent(GFMDisallowedTags...) explicitly re-asserts bluemonday's own hardcoded 6-tag skip-content set PLUS the 3 tags it does NOT cover (textarea/xmp/plaintext), for uniform strip (not partial-strip-partial-escape) across all 9 GFM-disallowed tags"
    - "p.AllowElements(...) alone does not preserve a zero-attribute element instance — bluemonday's allowNoAttrs gate requires an explicit p.AllowNoAttrs().OnElements(...) call too (discovered via RED on bare MathML leaves like <mi>x</mi>)"
    - "golang.org/x/net/html's Tokenizer.Token() unconditionally lowercases both tag names (TagName()) and attribute keys (TagAttr()) — harmless for ordinary HTML but breaks SVG's case-sensitive foreign-content model (foreignObject, viewBox); mitigated with a documented post-Sanitize regex case-restoration pass, NOT a bluemonday policy option (none exists)"
    - "Go regexp.ReplaceAllString($1foo) is parsed as a reference to submatch NAMED '1foo' (silently empty) — must use ${1}foo to disambiguate; caught via RED test output showing the tag name entirely erased"
    - "Adversarial fixtures must stay HTML-tokenizer-valid: an unterminated raw-text element (e.g. <script>...<\\/script> with an escaped slash) makes the HTML5 tokenizer swallow everything after it as script content — that's correct browser behavior, not a sanitize bug, so obfuscation must use well-formed open/close tags (case variation) instead"

key-files:
  created:
    - press/sanitize/policy.go
    - press/sanitize/tagfilter.go
    - press/sanitize/policy_test.go
    - press/sanitize/adversarial_test.go
  modified: []

key-decisions:
  - "Strip, not escape: bluemonday's default posture removes disallowed tags entirely; Marp's JS xss library instead escapes them into visible encoded text. This is a deliberate, tested deviation (TestStripVsEscape) — CORE-05's bar is behavioral XSS neutralization, not byte-parity with the JS library's strip-vs-escape choice."
  - "style attribute excluded ENTIRELY, with no partial allowance — bluemonday has no CSS-value sanitizer. This is a KNOWN, DELIBERATE functional tradeoff: it silently degrades two already-shipped chase/markdown features when their output later passes through this sanitize pass in 03-09 — the ![bg] background-image feature (<figure style=\"background-image:url(...)\">) and the advanced-background split/color-inherit mechanism (style=\"--marpit-advanced-background-split:...;\" / style=\"color:...;\"). Proven via TestPreserveAdvancedBackground (structure survives, style attr does not). Not silently patched around; flagged here for 03-09/hardening to react to."
  - "bluemonday's underlying golang.org/x/net/html Tokenizer unconditionally lowercases tag names AND attribute keys (Token()/TagName()/TagAttr() all call lower()) — a real, previously-undocumented finding discovered via a scratch spike. Left unmitigated, Policy().Sanitize() alone would rewrite <foreignObject>/viewBox to <foreignobject>/viewbox, which browsers' foreign-content (SVG) parsing does NOT recognize, silently breaking Marpit's entire inline-SVG rendering mode. Mitigation: sanitize.Sanitize(html) wraps Policy().Sanitize() with a documented, tested regex case-restoration pass. 03-09 MUST call Sanitize(), not bare Policy().Sanitize() — called out prominently since must_haves/key_links literally name Policy() as what 03-09 applies; Sanitize() is the correct superset entry point."
  - "MathML leaf elements (mi/mo/mn/mrow/...) commonly appear with ZERO attributes (e.g. <mi>x</mi>) — bluemonday requires an explicit AllowNoAttrs().OnElements(...) call for the tag to survive at all in that case, AllowElements() alone is insufficient. Added and proven via TestPreserveMathML."

patterns-established:
  - "Adversarial round-trip testing over the FINAL HTML string (never a mocked directive layer): preservation tests and neutralization tests share the same Policy()/Sanitize() entry points sibling TRD 03-09 will actually call."

requirements-completed: [CORE-05]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 2
  tdd_evidence: true
  test_pairing: true

# Metrics
duration: 15min
completed: 2026-07-21
---

# Objective 3 TRD 08: HTML Sanitize Policy (CORE-05) Summary

**A blank-slate bluemonday allow-list behaviorally matching Marp's v4-era `xss` policy — strips (not escapes) the 9 GFM-disallowed raw-HTML tags, blocks `javascript:`/`on*`/`style` vectors, and (via a documented post-pass) preserves every battery's output including the case-sensitive inline-SVG `<foreignObject>`/`viewBox` pair — proven by a 23-test adversarial + preservation round-trip suite.**

## Performance

- **Duration:** ~15 min (Task 1 commit 10:16:54 -> Task 2 commit 10:19:39 local, plus prior investigation/spike time not reflected in commit deltas)
- **Started:** 2026-07-21T14:07:00Z (session start)
- **Completed:** 2026-07-21T14:20:17Z
- **Tasks:** 2/2 complete
- **Files modified:** 4 (all newly created)

## Accomplishments
- `press/sanitize.Policy() *bluemonday.Policy`: blank-slate allow-list covering Marp v4-era structural/inline elements, the Marpit inline-SVG container chain, MathML (`git.sr.ht/~mekyt/latex2mathml`'s real element/attribute set), twemoji `<img>`, chroma `.hljs-*` spans (via global `class`), advanced-background/pagination data attrs, and the CORE-09 fit-marker hedge (class OR `data-auto-scaling` attribute) — `style` and `on*` handlers are never allow-listed anywhere (zero special-case code needed; bluemonday's blank-slate default strips them).
- `press/sanitize.GFMDisallowedTags` (`tagfilter.go`): the 9 raw-HTML tags goldmark's GFM extension does not filter, explicitly passed to `p.SkipElementsContent` for uniform strip across all 9 (bluemonday's own hardcoded skip-content set only covers 6 of them).
- **Critical finding + mitigation:** discovered (via a disposable Go spike, not assumed) that bluemonday's underlying `golang.org/x/net/html` Tokenizer unconditionally lowercases tag names and attribute keys — `<foreignObject>`/`viewBox` become `<foreignobject>`/`viewbox` under bare `Policy().Sanitize()`, which real browsers' SVG foreign-content parsing does NOT recognize, silently breaking Marpit's inline-SVG rendering mode. Added `press/sanitize.Sanitize(html string) string` — the recommended full pipeline (`Policy().Sanitize` + a documented regex case-restoration pass) — and proved it with `TestPreserveInlineSVGCaseRestoration`. **03-09 must call `sanitize.Sanitize()`, not bare `Policy().Sanitize()`.**
- 23 hand-built tests (no generated/LLM test data): 5 policy-rule tests (Task 1) + 12 battery-preservation tests + 6 adversarial-neutralization tests (Task 2), all targeting the sanitized HTML string end to end.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Marp-parity policy + GFM tagfilter | `go test ./press/sanitize/ -run 'TestTagFilter\|TestStyleAttr\|TestURLScheme\|TestOnHandler\|TestStripVsEscape' -v && gofmt -l press/sanitize/` | 0 | PASS |
| 2: Battery-preservation + adversarial suite | `go test ./press/sanitize/ -run 'TestPreserve\|TestAdversarial' -v && go vet ./press/sanitize/...` | 0 | PASS |

## Task Commits

Each task was committed atomically via `df-tools.cjs commit` (never raw `git commit`):

1. **Task 1: Marp-parity policy + GFM tagfilter** - `f0ed5cb` (feat)
2. **Task 2: Battery-preservation + adversarial round-trip suite** - `cd6a3fa` (test)

_Both tasks are `tdd="true"`; RED (compile failure against undefined `Policy`/`GFMDisallowedTags` for Task 1; 3 genuine assertion failures for Task 2) confirmed before each GREEN — see TDD Evidence below._

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./press/sanitize/...` (and `go build ./...`) | 0 | PASS |
| vet | `go vet ./press/sanitize/...` (and `go vet ./...`) | 0 | PASS |
| test | `go test ./press/sanitize/...` (and `go test ./...`) | 0 | PASS (all 17 repo packages green) |
| gofmt | `gofmt -l press/sanitize/` (and whole tree, excluding pre-existing unlicensed corpus fixtures) | 0 (no output) | PASS |
| Obj-1 corpus/cssdiff | `go test ./conformance/...` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate -v` | 0 | PASS |
| no-chromedp invariant | `go list -deps ./press/... \| grep -c chromedp` | — | PASS (count = 0) |
| go.mod/go.sum untouched | `git diff --stat go.mod go.sum` | 0 | PASS (empty diff) |

## TDD Evidence

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED (Task 1) | `go test ./press/sanitize/ -run 'TestTagFilter\|TestStyleAttr\|TestURLScheme\|TestOnHandler\|TestStripVsEscape' -v` | 1 (compile failure: undefined `Policy`) | FAIL (correct) |
| GREEN (Task 1) | same command | 0 (9+1+3+1+1 = all subtests pass) | PASS (correct) |
| RED (Task 2) | `go test ./press/sanitize/ -run 'TestPreserve\|TestAdversarial' -v` | 1 (3 genuine failures: bare MathML leaves stripped, `$1foreignObject` regexp bug erased the tag name, malformed obfuscated `<script>` swallowed the rest of the fixture) | FAIL (correct) |
| GREEN (Task 2) | same command | 0 (18 tests / all subtests pass) | PASS (correct) |

## Post-TRD Verification

- **Auto-fix cycles used:** 2 — (1) Task 1: none needed, policy compiled and passed on first implementation pass; (2) Task 2: one cycle fixing three co-discovered issues before commit — missing `AllowNoAttrs` for bare MathML leaves, a Go-regexp `$1`-vs-`${1}` submatch-naming bug in the case-restoration pass, and a test-fixture bug (unterminated raw-text `<script>` obfuscation swallowing trailing content, which is correct HTML5 tokenizer behavior, not a policy defect).
- **Must-haves verified:** 4/4 (all `must_haves.truths` from 03-08-TRD.md frontmatter — Policy()/Sanitize() behavioral parity + documented strip-vs-escape; GFM disallowed tags proven neutralized; battery output (emoji/MathML/inline-SVG/fit) proven preserved; adversarial round-trip suite proven over the final HTML string).
- **Gate failures:** None remaining (all fixed within Task 2's own TDD RED->GREEN loop, before commit).

## Files Created/Modified
- `press/sanitize/tagfilter.go` - `GFMDisallowedTags` (9-tag list) with strip-vs-escape rationale documented inline
- `press/sanitize/policy.go` - `Policy()` (the allow-list) + `Sanitize()` (recommended pipeline with SVG case restoration)
- `press/sanitize/policy_test.go` - Task-1 rule tests: `TestTagFilter`, `TestStyleAttr`, `TestURLScheme`, `TestOnHandler`, `TestStripVsEscape`
- `press/sanitize/adversarial_test.go` - Task-2 preservation tests (`TestPreserve*`, 12 cases incl. `TestPreserveInlineSVGCaseRestoration`) + adversarial tests (`TestAdversarial*`, 6 cases)

## Decisions Made
- See key-decisions in frontmatter (strip-vs-escape; `style` exclusion tradeoff; SVG case-restoration mitigation; MathML `AllowNoAttrs` requirement).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Bare (zero-attribute) MathML leaf elements were stripped despite being allow-listed**
- **Found during:** Task 2, RED phase (`TestPreserveMathML`)
- **Issue:** `p.AllowElements("mi","mo","mn","mrow",...)` alone does not preserve an element instance that carries zero attributes at parse time (e.g. `<mi>x</mi>`) — bluemonday's `sanitize()` loop requires `p.allowNoAttrs(tag)` to return true for any zero-attribute tag, which only happens via an explicit `AllowNoAttrs().OnElements(...)` call or bluemonday's own small hardcoded default set (which does not include MathML tags).
- **Fix:** Added `p.AllowNoAttrs().OnElements(mathElements...)` in `policy.go`, alongside the existing attribute policy for MathML elements that DO carry attributes.
- **Files modified:** press/sanitize/policy.go
- **Verification:** `TestPreserveMathML` passes (full `<mrow><mi>x</mi><mo>+</mo><mn>1</mn></mrow>` structure survives).
- **Committed in:** cd6a3fa (Task 2 commit)

**2. [Rule 1 - Bug] Go regexp `$1foreignObject` silently erased the restored tag name**
- **Found during:** Task 2, RED phase (`TestPreserveInlineSVGCaseRestoration`)
- **Issue:** `reForeignObjectTag.ReplaceAllString(out, "<$1foreignObject")` — Go's `regexp.ReplaceAllString` greedily parses `$1foreignObject` as a reference to a submatch NAMED `1foreignObject` (not submatch `1` followed by literal text), which does not exist, so it silently substitutes empty string. The tag name was completely erased (`<foreignobject width=...>` became `< width=...>`).
- **Fix:** Used the braced form `${1}foreignObject` to disambiguate the submatch-index boundary from the following literal text; documented the gotcha inline in `policy.go`.
- **Files modified:** press/sanitize/policy.go
- **Verification:** `TestPreserveInlineSVGCaseRestoration` passes (`<foreignObject>...</foreignObject>` and `viewBox="..."` both correctly restored).
- **Committed in:** cd6a3fa (Task 2 commit)

**3. [Rule 1 - Bug, in test fixture] Backslash-obfuscated `<script>` close tag made the HTML tokenizer swallow the rest of the adversarial fixture**
- **Found during:** Task 2, RED phase (`TestAdversarialNestedObfuscatedPayload`)
- **Issue:** The original fixture used `<script>var x=1;<\/script>` (a JS-string-escaping trick) as "obfuscation". HTML5 treats `<script>` as a raw-text element: the tokenizer scans for a literal, unescaped `</script` and nothing else terminates it. A backslash before the slash does NOT close the tag, so the tokenizer (correctly, matching real browser behavior) consumed everything after `<script>` — including the legitimate `<img>`, `<a>`, `<div>/<iframe>`, and `<p>safe text</p>` that followed — as script content, which was then stripped along with the script itself.
- **Fix:** This is a test-fixture bug, not a sanitize-policy bug (a real browser would do the same thing). Replaced the backslash trick with well-formed, properly-terminated open/close tags using case-variation obfuscation instead (`<ScRiPt type="text/javascript">...</ScRiPt>`, `<IfRaMe>...</IfRaMe>`), preserving the adversarial intent (case-insensitivity of the tag-name match) without breaking raw-text tokenization.
- **Files modified:** press/sanitize/adversarial_test.go
- **Verification:** `TestAdversarialNestedObfuscatedPayload` passes: all injected vectors neutralized, all legitimate sibling content (`class="fit"`, the data-URI `src`, `alt="ok"`, `"safe text"`) preserved.
- **Committed in:** cd6a3fa (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 1 - policy bugs; 1 Rule 1 - test-fixture bug; all fixed within Task 2's own TDD RED->GREEN loop, before commit)
**Impact on plan:** All three are corrections discovered by the adversarial/preservation suite doing exactly what it's meant to do. None change CORE-05's scope. The SVG case-restoration finding (deviation 2, paired with the underlying root-cause finding in key-decisions) is the most consequential: it surfaces a real, previously-undocumented bluemonday+SVG interaction that would have silently broken Marpit's inline-SVG rendering mode in 03-09 if `Policy().Sanitize()` had been used bare, per the TRD's own `key_links` literal phrasing. `Sanitize()` is the corrected, recommended entry point — flagged prominently here for 03-09's executor.

## Known Limitations / Deliberate Tradeoffs

1. **`style` attribute excluded entirely.** bluemonday has no CSS-value sanitizer, so `style` is never allow-listed anywhere in `Policy()`. This is safe but degrades two already-shipped chase/markdown features once their HTML passes through this sanitize pass in 03-09: the `![bg]` background-image feature (`<figure style="background-image:url(...)">` loses its `style`, so the image no longer renders) and the advanced-background split/color-inherit mechanism (`style="--marpit-advanced-background-split:...;"` / `style="color:...;"` similarly stripped). The surrounding structure (figure/figcaption, the three-layer advanced-background container/direction attrs) survives — only the inline CSS itself is lost. Proven by `TestPreserveAdvancedBackground`. This is a known, accepted tradeoff per the TRD's explicit, repeated anti-pattern ("never allow style") — not silently patched around. A future hardening objective could address this with a real CSS-value sanitizer (out of CORE-05's scope).
2. **bluemonday lowercases all tag/attribute names; mitigated for the two cases this project actually needs (`foreignObject` element, `viewBox` attribute) via `Sanitize()`'s regex case-restoration pass.** If a future battery introduces another case-sensitive SVG/MathML name, it must be added to the `reForeignObjectTag`/`reViewBoxAttr` (or an added sibling regex) in `policy.go` and proven with a preservation test — the mechanism is general but the concrete regex list is intentionally scoped to today's known battery output, not a general SVG-camelCase table.
3. **Marp v4 allow-list byte-parity is not exhaustive.** Per the TRD's own `error_recovery` guidance, this policy targets behavioral XSS neutralization plus proven preservation of every documented battery output shape, not byte-for-byte parity with Marp's JS `xss` library's exact tag/attribute enumeration. Any additional legitimate tag/attribute a future battery needs should be added with an accompanying preservation test (the established pattern here).

## Issues Encountered
None beyond the three auto-fixed deviations above, all resolved within the TDD cycle before commit.

## User Setup Required
None - no external service configuration required.

## Next Objective Readiness
- `press/sanitize.Sanitize(html string) string` is ready for 03-09 to apply as the ABSOLUTE LAST step over the fully-composed `Output.HTML` string — **03-09 must call `Sanitize()`, not bare `Policy().Sanitize()`**, to correctly preserve the inline-SVG container chain's case-sensitive `foreignObject`/`viewBox` naming.
- `press/sanitize.Policy()` remains available standalone (satisfies the TRD's literal must-have symbol) for any caller that wants the raw bluemonday policy without the SVG case-restoration wrapper, though 03-09 should not use it bare.
- `press/sanitize.GFMDisallowedTags` is exported and reusable if a sibling package ever needs the same 9-tag list independent of the policy.
- The three Known Limitations above are actionable inputs for 03-09's integration and any post-Objective-3 hardening pass.

## Self-Check: PASSED

All claimed files confirmed present on disk; both task commit hashes confirmed present in `git log --oneline --all`.

- FOUND: press/sanitize/policy.go
- FOUND: press/sanitize/tagfilter.go
- FOUND: press/sanitize/policy_test.go
- FOUND: press/sanitize/adversarial_test.go
- FOUND commit: f0ed5cb (Task 1)
- FOUND commit: cd6a3fa (Task 2)

---
*Objective: 03-press-batteries-api*
*Completed: 2026-07-21*
