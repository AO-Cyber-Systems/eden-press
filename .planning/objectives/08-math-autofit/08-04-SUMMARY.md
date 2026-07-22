---
objective: 08-math-autofit
job: "04"
subsystem: math-fallback-trigger-detector
tags: [go, mathml, regexp, fallback-trigger, corpus-test, tdd, structural-ceiling, latex2mathml, empirical-verification]

# Dependency graph
requires:
  - objective: 08-math-autofit
    job: "03"
    provides: "internal/latex2mathml \\aligned MATRICES registration (aligned->mtable) -- the HARD landed precondition this TRD's trigger-shrink depends on (research Pitfall 1: never remove a trigger before its fix lands)."
provides:
  - "press/math/detect.go: fallbackRE FINALIZED to \\tag\\b|\\label\\b|\\begin\\{(?:align|alignat\\*?)\\}  -- the permanent Chromium MathML-Core structural ceiling (no <mlabeledtr>) plus the two amsmath numbered-alignment forms (\\align, \\alignat/\\alignat*) still empirically broken in this fork. cases/aligned/array/align* removed (render natively)."
  - "press/math/detect_test.go: TestNeedsFallback positives/negatives finalized to the above; NEW TestFallbackRouting corpus/table test asserting the trigger set -> PNG and every now-supported construct (incl. the 8 PROPOSAL §11 spike cases) -> native MathML, structurally (criterion 2)."
  - "press/math/math_test.go TestMathRender: end-to-end routing re-proved through the real goldmark pipeline -- aligned now -> <math> (no <img>); \\tag (still the ceiling) -> fallback <img> (no <math>)."
  - "press/capstone_test.go: Objective-3's capstone fixture (deviation fix) swapped its 'routes to fallback' example from \\begin{aligned} (now stale, renders native) to \\tag{1} (still genuinely routes to fallback)."
affects:
  - "Objective 8 is now FULLY COMPLETE (7/7 TRDs) -- this was the last wave, closing success criterion 2 (concrete, corpus-tested fallback-trigger rule reflecting the permanent structural ceiling, not converter bugs). The whole roadmap (9 objectives, 0-8) has no remaining planned work."
  - "Any future converter patch that adds MATRICES registration + {n} column-count parsing for \\align (un-starred)/\\alignat/\\alignat* should REMOVE them from fallbackRE and add a TestSpikeCorpus-style regression case FIRST (research Pitfall 1 applies symmetrically to future shrinks)."

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Empirical-first regex finalization: PROPOSAL.md §11's 20-case battery never actually exercised \\array or \\alignat (only the 8 later-fixed spike cases + \\cases were in that table) -- so 'read §11' alone was insufficient. A throwaway probe test (l2m.Convert called directly, deleted before the real TDD commits) against the LIVE post-08-03 converter was the actual authoritative source for the array/alignat/align decision, exactly as the TRD's own error_recovery anticipated ('or a quick native-render check against the now-patched converter')."
    - "Regex alternation with a shared literal suffix, verified not assumed: \\begin\\{(?:align|alignat\\*?)\\} relies on Go's RE2 alternation correctly falling through to the longer 'alignat' branch when 'align' + immediate closing brace fails to match (e.g. \\begin{alignat} does NOT spuriously match on the 'align' arm alone) -- confirmed with a second throwaway regex-only probe before committing to the final pattern, not reasoned about in the abstract."

key-files:
  created:
    - .planning/objectives/08-math-autofit/08-04-SUMMARY.md
  modified:
    - press/math/detect.go
    - press/math/detect_test.go
    - press/math/math_test.go
    - press/capstone_test.go

key-decisions:
  - "PROPOSAL §11 RE-CONFIRMED (research Open Q3), then EXTENDED empirically because §11's 20-case battery has NO row for \\array or \\alignat: (a) \\cases = KaTeX-quality as-shipped (§11, unchanged) -> REMOVED from trigger. (b) \\array -- not in §11's battery at all; a direct l2m.Convert() probe against the live (post-08-03) converter shows a correct <mtable> with proper <mtr>/<mtd> rows/columns (\\array has been a MATRICES-registered environment all along) -> REMOVED from trigger. (c) \\alignat{n} / \\alignat*{n} -- also not in §11's battery; the SAME probe shows BOTH forms emit a stray <mn>n</mn> (the column-count argument leaks through unconsumed), a literal unparseable <mi>&</mi> per separator, and NO <mtable> wrapper at all -- \\alignat is not in the converter's MATRICES list, so it falls to the generic convertGroup fallthrough. Genuinely hits the structural ceiling (unmodeled {n}-argument syntax) exactly as the TRD's error_recovery anticipated -> KEPT in trigger, both starred and un-starred forms."
  - "[Deviation, Rule 1 - bug in the TRD's own assumption] Plain \\begin{align} (un-starred) is NOT fixed by 08-03 and must stay in the trigger -- contradicting this TRD's own codebase_examples/task text, which assumed 'align' was fixed alongside 'aligned'. Root cause: internal/latex2mathml/commands.go's ALIGN constant is literally `\\align*` (the STARRED amsmath sibling), not `\\align`. 08-03 registered \\aligned as MATRICES-behaviour-identical to the EXISTING \\align* entry -- it never touched plain, un-starred \\align. A direct probe confirms plain \\align still emits the literal <mi>&</mi> + <mspace linebreak> fallthrough (byte-identical to the pre-08-03 \\aligned bug). This is not merely a leftover converter bug of no consequence: amsmath's un-starred \\align is the NUMBERED alignment environment (auto equation numbers per row), so it shares \\tag/\\label's <mlabeledtr> gap by design -- even a future converter fix would still need <mlabeledtr> support Chromium lacks. KEPT in trigger; \\aligned and \\align* (both unnumbered, both empirically confirmed correct) are the ones removed."
  - "[Deviation, Rule 1 - bug, downstream] press/capstone_test.go (Objective 3) hardcoded \\begin{aligned} as its 'this routes to the fallback' battery fixture. That assumption is now false (aligned renders native, correctly, as designed). Fixed by swapping the fixture to \\tag{1}, which still genuinely triggers the fallback -- the capstone's fingerprint battery (`math-fallback`) needed a construct that ACTUALLY still hits the ceiling, and \\tag is the cleanest permanent example."
  - "Kept the regex's word-boundary + exact-brace-name structure (`\\{(?:align|alignat\\*?)\\}`) rather than switching to a broader amsmath-environment name list: this scopes the alignat*/align* handling precisely (verified with a throwaway regex-only probe) without reopening the whole alternation to unrelated environments."

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 9
  gates_passed: 9
  auto_fix_cycles: 1
  tdd_evidence: true
  test_pairing: true
  blockers: none

# Metrics
duration: ~35min
completed: 2026-07-22
---

# Objective 08 TRD 04: Finalize the MathML fallback-trigger detector to the permanent structural ceiling Summary

**`press/math/detect.go`'s `fallbackRE` is FINALIZED to `\tag\b|\label\b|\begin\{(?:align|alignat\*?)\}` — the permanent Chromium MathML-Core structural ceiling plus the two amsmath numbered-alignment forms this fork's converter still cannot render. `cases`, `aligned`, `array`, and `align*` are REMOVED from the trigger (all confirmed rendering natively at KaTeX quality); `\tag`/`\label` (no `<mlabeledtr>` in MathML Core) and `\align`(un-starred)/`\alignat`/`\alignat*` (numbered-equation semantics sharing the same `<mlabeledtr>` gap, PLUS an empirically-confirmed unmodeled `{n}`-column-count argument for alignat) are KEPT. Two facts this TRD had to discover empirically rather than assume: PROPOSAL §11's 20-case battery never tested `\array` or `\alignat` at all (so "re-read §11" alone was insufficient — a direct converter probe was required, exactly as the TRD's own error_recovery anticipated), and plain un-starred `\begin{align}` was NEVER actually fixed by 08-03 (only `\aligned` and the pre-existing `\align*` work — the `ALIGN` constant in the fork is literally `\align*`), contradicting this TRD's own assumption that "align" was fixed alongside "aligned". Routing is corpus-tested (not manually inspected) via the new `TestFallbackRouting` table plus the finalized `TestNeedsFallback` positives/negatives, both driven end-to-end through `renderHTML` in `TestMathRender`. This is the LAST TRD of Objective 8 — the objective (and the entire 9-objective roadmap) is now fully complete.**

## Empirical baseline (captured before finalizing the regex — Task 1)

PROPOSAL.md §11 (~lines 289-317) re-opened and re-confirmed: its 20-case battery table has rows for `cases` (✅ KaTeX-quality) and `aligned` (❌, fixed by 08-03) — but **no row at all** for `array` or `alignat`. Per the TRD's own instruction ("§11, or a quick native-render check against the now-patched converter"), a throwaway probe (`l2m.Convert` called directly against the live, post-08-03 converter; deleted before any real commit) was run:

| Input | Live converter output (post-08-03) | Diagnosis |
|---|---|---|
| `\begin{array}{cc} a & b \\ c & d \end{array}` | `<mtable><mtr><mtd>a</mtd><mtd>b</mtd></mtr>…` | Correct `<mtable>`, proper rows/columns — `\array` has been MATRICES-registered all along |
| `\begin{alignat}{2} a &= b & c &= d \end{alignat}` | `<mrow><mn>2</mn><mi>a</mi><mi>&</mi><mo>=</mo>…` (no `<mtable>`) | Column-count `{2}` leaks as a stray `<mn>`; literal unparseable `<mi>&</mi>`; NO table wrapper — `\alignat` is NOT in the converter's `MATRICES` list |
| `\begin{alignat*}{2} a &= b & c &= d \end{alignat*}` | identical broken shape to the un-starred form | Same gap, starred and un-starred |
| `\begin{align} a &= b \\ c &= d \end{align}` (un-starred) | `<mrow><mi>a</mi><mi>&</mi><mo>=</mo><mi>b</mi><mspace linebreak/>…` | **Still broken** — identical to the pre-08-03 `aligned` bug. `commands.go`'s `ALIGN` constant is literally `` `\align*` ``; 08-03 registered `\aligned` as behaviour-identical to that EXISTING `\align*` entry, never touching plain un-starred `\align` |
| `\begin{align*} a &= b \\ c &= d \end{align*}` (starred) | `<mtable columnspacing="0em 2em"><mtr><mtd columnalign="right">…` | Correct — this is the pre-existing `ALIGN = \align*` entry |
| `\begin{aligned} a &= b \\ c &= d \end{aligned}` | `<mtable>` right/left split | Correct (08-03 fix, unchanged) |
| `a = b \tag{1}` | `<mi>a</mi><mo>=</mo><mi>b</mi><mi>\tag</mi><mrow><mn>1</mn></mrow>` | `\tag` not a recognized token at all — confirmed permanent |

**Decision recorded:** `array` → REMOVED (renders correctly, no §11 row needed since the live converter already handles it). `alignat`/`alignat*` → KEPT (genuinely unmodeled `{n}` syntax, exactly the "hits the ceiling" case the TRD's error_recovery anticipated). `align` (un-starred) → **KEPT, deviating from the TRD's own assumption** that it was fixed alongside `aligned` — it was not; see key-decisions above for the full justification (numbered-equation semantics share `\tag`'s `<mlabeledtr>` gap by design, independent of whatever a future converter patch might do about the raw ampersand bug).

## What was built

### Task 1 — RED: finalized positive/negative lists + routing corpus table (commit 5ca2c56)
- `TestNeedsFallback`: positives narrowed to `\tag`, `\label`, `\align` (un-starred), `\alignat`, `\alignat*`. Negatives gained `\aligned`, `\align*`, `\cases`, `\array` (moved from positives) alongside the existing common-math and word-boundary-guard cases.
- New `TestFallbackRouting`: a `{name, raw, expectFallback}` table covering the finalized trigger set as TRUE and every now-supported construct — `cases`/`aligned`/`align*`/`array` plus all 8 PROPOSAL §11 `TestSpikeCorpus` raw strings (`math_test.go`) — plus the `\tagged`/`\labelled` word-boundary guard, as FALSE.
- Confirmed RED against the prior over-broad `fallbackRE`: `aligned`/`cases`/`array` still matched (over-broad); `alignat*` did NOT yet match (the old rule's exact `{alignat}` brace match never covered the starred form).

### Task 2 — GREEN: finalized fallbackRE + end-to-end routing proof (commit 20584e9)
- `detect.go`: `fallbackRE` rewritten to `` \tag\b|\label\b|\begin\{(?:align|alignat\*?)\} ``. Doc comment rewritten to explain BOTH reasons a construct is in the set (permanent `<mlabeledtr>` gap vs. currently-unmodeled `alignat` column-count syntax) and cite the empirical evidence for every removal.
- `math_test.go` `TestMathRender`: inverted — `\begin{aligned}` now asserted to render `<math>` with no `<img>`; the fallback-routing assertion moved to `` $$a=b\tag{1}$$ `` (still-ceiling construct).
- `press/capstone_test.go` (Rule 1 auto-fix, downstream regression): Objective 3's capstone deck hardcoded `\begin{aligned}` as its "routes to fallback" battery fingerprint — now false. Swapped to `\tag{1}`; comment updated to explain why.
- Full `go test ./...` re-run clean after the capstone fix; all 8 `TestSpikeCorpus` cases, `TestNeedsFallback`, `TestFallbackRouting`, and `TestMathRender` all green.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: RED (finalized lists + corpus table) | `go test ./press/math/... -run 'TestNeedsFallback\|TestFallbackRouting' -v` | 1 (expected RED) | PASS (correctly RED) |
| 2: GREEN (fallbackRE + end-to-end) | `go test ./press/math/... -v && gofmt -l … && go build ./... && go vet ./... && go test ./... && go test ./conformance/... && bash scripts/check-no-chromedp.sh && bash scripts/check-cli-imports.sh && go test ./profiles/slides/ -run TestGrepGate && addlicense … -check .` | 0 | PASS |

## TDD Evidence

| Phase | Command | Exit | Expected |
|---|---|---|---|
| RED | `go test ./press/math/... -run 'TestNeedsFallback\|TestFallbackRouting' -v` | 1 | FAIL — `aligned`/`cases`/`array` still over-broadly routed to fallback; `alignat*` not yet caught (correct) |
| GREEN | `go test ./press/math/... -run 'TestNeedsFallback\|TestFallbackRouting' -v` | 0 | PASS — exact finalized trigger set, all 8 spike cases + cases/aligned/array/align* native (correct) |

## Auto-fix cycle (Rule 1)

**1. [Rule 1 - bug, downstream regression] `press/capstone_test.go` (Objective 3) broke when `fallbackRE` was tightened**
- **Found during:** the `go test ./...` whole-repo gate after Task 2's GREEN commit.
- **Issue:** `TestCapstoneAllThemesEveryBattery` failed on all 3 themes — `composed HTML missing math fallback routing (03-06) (want "math-fallback")`. The Objective-3 capstone deck used `` $$\begin{aligned} x &= 1 \end{aligned}$$ `` as its hardcoded "this construct routes to the PNG fallback" battery fixture; now that `aligned` correctly renders native MathML, the `math-fallback` fingerprint no longer appears anywhere in the composed HTML.
- **Fix:** swapped the fixture to `` $$x = 1 \tag{1}$$ `` — `\tag` still genuinely triggers the fallback (permanent ceiling) — and updated the surrounding doc comment to explain the swap and why `aligned` no longer qualifies as a fallback example.
- **Files modified:** `press/capstone_test.go`.
- **Commit:** 20584e9 (folded into Task 2's GREEN commit, since it was a direct, same-cause regression from the routing change).
- **Verification:** `go test ./press/...` green afterward; full `go test ./...` green.

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l press/math/detect.go press/math/detect_test.go press/math/math_test.go press/capstone_test.go` | 0 | PASS (empty) |
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok, no FAIL) |
| test (press/math -v) | `go test ./press/math/... -v` | 0 | PASS (every subtest, incl. TestSpikeCorpus 8/8) |
| conformance | `go test ./conformance/...` | 0 | PASS |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS |
| cli-imports | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -ignore 'conformance/corpus/cases/**' -ignore '**/node_modules/**' -ignore 'themes/**' -ignore 'convert/chrome/fonts/**' -ignore 'internal/latex2mathml/**' -check .` | 0 | PASS |

## Deviations from Plan

### Auto-fixed (Rule 1, no user permission required)

**1. Plain `\align` (un-starred) was NOT fixed by 08-03 — kept in the trigger, contradicting this TRD's own assumption**
- **Found during:** Task 1's empirical re-confirmation (before writing any test).
- **Issue:** the TRD's `codebase_examples` and task text assumed "align" (alongside "aligned") was fixed in 08-03 and should be removed. A direct probe shows plain, un-starred `\begin{align}` still emits the literal `<mi>&</mi>` + `<mspace linebreak>` fallthrough — byte-identical to the pre-08-03 `aligned` bug. Root cause: `internal/latex2mathml/commands.go`'s `ALIGN` constant is literally `` `\align*` `` (the amsmath STARRED/unnumbered sibling), a naming trap; 08-03 registered `\aligned` as behaviour-identical to that pre-existing `\align*` entry and never touched plain `\align`.
- **Fix:** kept `\align` (un-starred) in `fallbackRE`, alongside `\alignat`/`\alignat*`. Removed only `\aligned` and implicitly left `\align*` unmatched (both empirically confirmed correct).
- **Files modified:** `press/math/detect.go`, `press/math/detect_test.go` (positives/negatives + `TestFallbackRouting` reflect this).
- **Commit:** 5ca2c56 (test), 20584e9 (implementation).

**2. `press/capstone_test.go` fixture regression** — see "Auto-fix cycle (Rule 1)" above.

No other deviations — the array/alignat/cases decisions match the TRD's own anticipated error_recovery outcomes exactly.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 1 (the capstone fixture regression, Rule 1)
- Must-haves verified: 5/5 — (1) fallbackRE finalized to the structural ceiling, justified per-construct against PROPOSAL §11 + empirical evidence; (2) `cases` removed, `aligned`/`align`/`cases` all corpus-tested to their correct route; (3) routing corpus-tested via `TestFallbackRouting` (table, not manual); (4) end-to-end routing holds through `renderHTML` (`TestMathRender`); (5) PROPOSAL §11 re-confirmed as the first step, with `array`/`alignat`'s decision recorded here (empirically extended beyond §11's own table, which never covered them).
- Success criteria: the MathML fallback-trigger detector is a concrete, corpus-tested function routing exactly the permanent structural ceiling (+ the two still-broken numbered-alignment forms) to the PNG path; trigger set tightened to reflect post-08-03 converter reality.
- Gate failures: None (after the one Rule-1 auto-fix)
- Blockers: None

**Objective 8 status: 7/7 TRDs complete.** All four objective success criteria are now satisfied: (1) all 8 spike cases at KaTeX-parity, locked in `TestSpikeCorpus` (08-02/08-03); (2) the fallback-trigger detector finalized + corpus-tested (this TRD); (3) STIX Two Math bundled + CI MATH-table smoke (08-05); (4) auto-fit resolved Flutter-native, zero viewer-side JS (08-06/08-07). This closes the entire 9-objective roadmap (0-8) — no further objectives remain planned.

## Commits

- `5ca2c56` test(08-04): RED — finalized needsFallback positives/negatives + TestFallbackRouting corpus table
- `20584e9` feat(08-04): finalize fallbackRE to the permanent structural ceiling (GREEN)

## Self-Check: PASSED

- Files verified on disk (5/5): `press/math/detect.go`, `press/math/detect_test.go`, `press/math/math_test.go`, `press/capstone_test.go`, `.planning/objectives/08-math-autofit/08-04-SUMMARY.md` — all FOUND.
- Commits verified in `git log` (2/2): `5ca2c56`, `20584e9` — all FOUND.
- Full standing gate set (gofmt/build/vet/test/conformance/no-chromedp/cli-imports/grep-gate/addlicense) PASS; whole-repo `go test ./...` clean including the Rule-1 capstone fix.
- `TestFallbackRouting` + finalized `TestNeedsFallback` + `TestMathRender` all green; all 8 `TestSpikeCorpus` cases still locked and passing.
- No scratch/probe test files remain in `press/math` (two throwaway `l2m.Convert`/regexp probes used during Task 1's empirical research were deleted before any commit).
