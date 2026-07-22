---
objective: 08-math-autofit
job: "01"
subsystem: math-converter-fork
tags: [go, latex2mathml, mathml, vendor-fork, go-mod-replace, licensing, foundation]

# Dependency graph
requires:
  - objective: 03-press-batteries-api
    provides: "git.sr.ht/~mekyt/latex2mathml pinned in go.mod (03-01) + press/math/mathml.go's l2m.Convert import site — the exact import this fork redirects without changing it"
provides:
  - "internal/latex2mathml/ — the in-repo VERBATIM fork of git.sr.ht/~mekyt/latex2mathml @v0.0.0-20231214134936-808832af73fc (module path preserved), the single converter every downstream math TRD patches or reasons against"
  - "root go.mod `replace git.sr.ht/~mekyt/latex2mathml => ./internal/latex2mathml` — the SOLE fork seam; redirects the existing import path so press/math import sites stay byte-identical. This TRD OWNS go.mod for Objective 8; 08-02..08-07 never edit it"
  - "addlicense -ignore 'internal/latex2mathml/**' in ci.yml — the fork is verbatim third-party MIT (mekyt), not AO-Cyber-stamped; matches the themes/** + convert/chrome/fonts/** precedent"
  - "NOTICE latex2mathml (MIT, Copyright (c) 2023 mekyt) + go-latex/latex (BSD-3-Clause) license placeholders resolved with confirmed licenses + fork provenance"
affects:
  - "08-02 (big-op limits + \\sqrt[n] radicand) patches internal/latex2mathml/converter.go + walker.go directly — no go.mod / import-path churn"
  - "08-03 (binom/pmatrix fence, aligned->mtable, mathvariant->codepoint) patches internal/latex2mathml/converter.go"
  - "08-04 fallback-rule reasons against the in-repo converter (assumes the aligned fix landed)"

# Tech tracking
tech-stack:
  added:
    - "internal/latex2mathml/ (in-repo fork of git.sr.ht/~mekyt/latex2mathml @v0.0.0-20231214134936-808832af73fc — a full replace-target Go module, module path git.sr.ht/~mekyt/latex2mathml PRESERVED)"
    - "go.mod replace directive (git.sr.ht/~mekyt/latex2mathml => ./internal/latex2mathml) — additive, hand-authored, no go mod tidy"
  patterns:
    - "Local-path go.mod replace as a fork seam: because the replace redirects the EXISTING import path (not a new one), every press/math import site and the frozen press.Options.MathMode contract are byte-for-byte untouched — the fork is invisible to callers, so downstream patch TRDs edit converter.go/walker.go with zero import ripple"
    - "Verbatim third-party vendor = own-license-preserved + addlicense-ignored: mekyt's MIT LICENSE kept unmodified, the .go files keep their upstream header/absence (never AO-Cyber-stamped), and internal/latex2mathml/** joins the addlicense -ignore set exactly like themes/** and convert/chrome/fonts/**"
    - "go.mod ownership isolation (03-01 precedent): the vendor + replace + licensing land in ONE foundation TRD so the sequenced patch TRDs (08-02/08-03) touch only converter.go/walker.go/test files, never go.mod — no parallel-worktree go.mod conflicts"
    - "Behavior-identical landing proof: a verbatim copy (10 files md5-identical to the module cache) means go build/vet/test + conformance are green with NO math-output diff vs the external dep — the fork adds the mechanism and nothing else; the 5 fixes are 08-02/08-03's job"

key-files:
  created:
    - internal/latex2mathml/go.mod
    - internal/latex2mathml/go.sum
    - internal/latex2mathml/LICENSE
    - internal/latex2mathml/README.md
    - internal/latex2mathml/commands.go
    - internal/latex2mathml/converter.go
    - internal/latex2mathml/walker.go
    - internal/latex2mathml/tokenizer.go
    - internal/latex2mathml/symbols_parser.go
    - internal/latex2mathml/unimathsymbols.txt
  modified:
    - go.mod
    - .github/workflows/ci.yml
    - NOTICE

key-decisions:
  - "example/ OMITTED from the vendored copy. Upstream ships example/example.go (package main, imports only latex2mathml). It is not needed by the library, is absent from the TRD file_tree/files_modified, and lives under a nested module boundary so root `go build ./...` never descends into it anyway. Copying the 10 library-source + LICENSE/README/go.mod/go.sum files suffices; example/ omitted per the TRD anti-pattern to avoid bloat"
  - "go.sum shows NO diff. A local-path replace target is read from disk and needs no module-hash verification, so `go build ./...` did not add go.sum lines. The pre-existing latex2mathml version-hash lines remain (untouched — removing them would require the forbidden go mod tidy) and are harmless. github.com/neruyzo/etree (the fork's only dep) was already pinned at go.mod:41 at the identical pseudo-version, so it resolved through the existing go.sum with zero new require lines"
  - "latex2mathml NOTICE URL corrected from https://github.com/roniemartinez/latex2mathml (the roniemartinez Python project the pre-math placeholder cited) to https://git.sr.ht/~mekyt/latex2mathml — the sourcehut Go port is the module Eden Press actually depends on and forks"
  - "go-latex/latex resolved to BSD-3-Clause (Copyright (c)2020 The go-latex Authors), verified against its own LICENSE (retain-copyright / reproduce-in-binary / no-endorsement conditions). It is an ordinary go.mod dependency (v0.3.0), NOT vendored/forked, so it gets a NOTICE entry but NO addlicense -ignore"

requirements-completed: []

# Verification evidence
verification:
  gates_defined: 5
  gates_passed: 5
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: false
  blockers: none

# Metrics
duration: ~5min
completed: 2026-07-22
---

# Objective 08 TRD 01: Fork latex2mathml — in-repo vendor + go.mod replace foundation Summary

**latex2mathml is now an in-repo fork.** `git.sr.ht/~mekyt/latex2mathml @v0.0.0-20231214134936-808832af73fc` is vendored VERBATIM (10 files, every one md5-identical to the module cache) into `internal/latex2mathml/` with its module path preserved, and a single `replace git.sr.ht/~mekyt/latex2mathml => ./internal/latex2mathml` directive in the root go.mod redirects the existing import — so `press/math/mathml.go`'s `l2m "git.sr.ht/~mekyt/latex2mathml"` import and `l2m.Convert(...)` call are **byte-for-byte unchanged** and the whole tree builds against the fork. The copy is behavior-identical (this TRD is the fork MECHANISM and NOTHING else — the five converter root-cause patches land in 08-02/08-03 against this in-repo copy). The fork's MIT license (Copyright (c) 2023 mekyt) is preserved verbatim + addlicense-ignored, and the NOTICE latex2mathml + go-latex/latex license placeholders the objective inherited are resolved with confirmed licenses and fork provenance. This is the prerequisite that makes the two structurally-unfixable-from-outside bugs (`\sqrt[n]` radicand loss, big-operator limit stacking) patchable at all.

## What was built

### Task 1 — vendor latex2mathml verbatim + go.mod replace (commit 66fc2cf)
- Copied 10 files from the local module cache into `internal/latex2mathml/`: `go.mod`, `go.sum`, `LICENSE`, `README.md`, `commands.go`, `converter.go`, `walker.go`, `tokenizer.go`, `symbols_parser.go`, `unimathsymbols.txt`. All 10 verified **md5-identical** to the source (verbatim copy — no truncation, no re-save, no line-ending drift).
- `internal/latex2mathml/go.mod` keeps `module git.sr.ht/~mekyt/latex2mathml` UNCHANGED so the replace target matches the import path. Its `require github.com/neruyzo/etree v0.0.0-20230816193247-70b7b06b18ad` is the identical pseudo-version already pinned at root go.mod:41.
- Appended the single `replace git.sr.ht/~mekyt/latex2mathml => ./internal/latex2mathml` directive (with an explanatory comment block) to the root go.mod by hand — NO `go mod tidy`. `go build ./...` reconciled additively (go.sum needed no new lines; a local replace target needs no hash entry).
- `example/` omitted (see key-decisions). `press/math/mathml.go` NOT touched — `git diff --exit-code press/math/mathml.go` is empty; the replace does all the redirection.

### Task 2 — license the fork + resolve NOTICE placeholders (commit 1d6a38f)
- `internal/latex2mathml/LICENSE` confirmed the upstream MIT text VERBATIM (`Copyright (c) 2023 mekyt`) — preserved unmodified from Task 1, never AO-Cyber-stamped.
- Added `-ignore 'internal/latex2mathml/**'` to the addlicense invocation in `.github/workflows/ci.yml` (alongside the existing `themes/**` + `convert/chrome/fonts/**` verbatim-asset ignores) and extended the explaining comment. Verified load-bearing: WITHOUT the ignore, addlicense flags the 5 fork `.go` files (exit 1); WITH it, exit 0.
- Resolved the NOTICE `latex2mathml` placeholder: `MIT, Copyright (c) 2023 mekyt`, upstream `git.sr.ht/~mekyt/latex2mathml @v0.0.0-20231214134936-808832af73fc` (URL corrected from the roniemartinez Python project to the sourcehut Go module), + fork provenance (vendored verbatim into internal/latex2mathml/, redirected via go.mod replace, patched by Objective 8 for the 5 converter root-causes, redistributed under its own MIT terms).
- Resolved the sibling `go-latex/latex` placeholder: `BSD-3-Clause, Copyright (c)2020 The go-latex Authors` (v0.3.0, ordinary go.mod dep — not vendored, so no addlicense -ignore).

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: vendor + replace | `gofmt -l internal/latex2mathml/*.go && go build ./... && go vet ./... && go test ./press/math/... ./... && go test ./conformance/... && bash scripts/check-no-chromedp.sh && bash scripts/check-cli-imports.sh && go test ./profiles/slides/ -run TestGrepGate && git diff --exit-code press/math/mathml.go` | 0 | PASS |
| 2: license + NOTICE | `addlicense … -ignore 'internal/latex2mathml/**' -check . && grep -q "Copyright (c) 2023 mekyt" internal/latex2mathml/LICENSE && ! grep -q "verify exact license in the math objective" NOTICE` | 0 | PASS |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| gofmt | `gofmt -l internal/latex2mathml/*.go` (+ press/chase/profiles) | 0 | PASS (empty — vendored files are gofmt-clean) |
| build | `go build ./...` | 0 | PASS (replace resolves to in-repo copy) |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok; press/math TestMathML/TestFallback/TestMathRender green — no math diff) |
| conformance | `go test ./conformance/...` | 0 | PASS (corpus, cssdiff, htmldiff, report, runner ok) |
| no-chromedp | `bash scripts/check-no-chromedp.sh` | 0 | PASS (fork adds no chromedp) |
| cli-imports | `bash scripts/check-cli-imports.sh` | 0 | PASS |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS (no regression) |
| addlicense | `addlicense … -ignore 'internal/latex2mathml/**' -check .` | 0 | PASS (fork ignored; mekyt MIT preserved) |
| import-site untouched | `git diff --exit-code press/math/mathml.go` | 0 | PASS (byte-identical) |
| go.mod diff | `git diff go.mod` | — | PASS (ONLY the replace directive + comment; go.sum diff empty — no other churn) |

## Deviations from Plan

### Recorded (no user permission needed — additive, TRD-anticipated)

**1. [Decision — TRD-sanctioned] `example/` omitted from the vendored copy**
- **Found during:** Task 1. Upstream ships `example/example.go` (package main, imports only latex2mathml).
- **Resolution:** Omitted, exactly as the TRD anti-pattern permits ("drop example/ if it introduces churn; note the omission in SUMMARY"). It is absent from the TRD file_tree/files_modified, unneeded by the library, and under a nested module boundary so root `go build ./...` never descends into it regardless. The 10 library-source + LICENSE/README/go.mod/go.sum files are the complete replace-target module.

**2. [Observation] go.sum has an empty diff**
- **Found during:** Task 1 `go build ./...`. A local-path replace target is read from disk and needs no module-hash verification, so no go.sum lines were added. The fork's only dependency (github.com/neruyzo/etree) was already pinned at the identical pseudo-version (go.mod:41), resolving through the existing go.sum. This is the expected "no other churn" outcome — not a missing step.

No other deviations — the vendor + replace + licensing landed exactly as the TRD specified. No converter bug was patched (that is 08-02/08-03's job); `go test ./...` proves behavior is UNCHANGED from the external dep.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0
- Must-haves verified: 5/5
  1. latex2mathml vendored VERBATIM into internal/latex2mathml/ (module path preserved) + go.mod replace redirects the existing import — press/math/mathml.go byte-for-byte unchanged, whole tree builds against the fork. VERIFIED (md5-identical copy; empty git diff on mathml.go; go build 0).
  2. Fork is BEHAVIOR-IDENTICAL at landing — go build/vet/test + conformance green with the vendored copy; this TRD adds the mechanism and nothing else. VERIFIED (no math diff; all tests ok).
  3. Upstream MIT LICENSE (Copyright (c) 2023 mekyt) preserved untouched; internal/latex2mathml/** added to the addlicense -ignore set; addlicense -check green. VERIFIED (grep confirms copyright; ignore proven load-bearing).
  4. NOTICE latex2mathml placeholder replaced with confirmed MIT + upstream pin + fork provenance; go-latex/latex resolved to BSD-3-Clause. VERIFIED (placeholder string gone).
  5. No other dependency churn — go.mod gains ONLY the replace directive; go mod tidy never run; etree resolves through existing go.sum with no new require. VERIFIED (go.mod diff = replace only; go.sum diff empty).
- Gate failures: None
- Blockers: None

## Commits

- `66fc2cf` feat(08-01): vendor latex2mathml as in-repo fork via go.mod replace
- `1d6a38f` chore(08-01): license the vendored latex2mathml fork + resolve NOTICE placeholders

## Self-Check: PASSED

- Files verified on disk (11/11): `internal/latex2mathml/{go.mod,go.sum,LICENSE,README.md,commands.go,converter.go,walker.go,tokenizer.go,symbols_parser.go,unimathsymbols.txt}` (all md5-identical to the module cache) + `.planning/objectives/08-math-autofit/08-01-SUMMARY.md` — all FOUND.
- Commits verified in `git log` (2/2): `66fc2cf`, `1d6a38f` — both FOUND.
- Both TRD `<verify>` gates PASS; whole-repo `go build`/`go vet`/`go test`/`gofmt -l` clean; conformance + no-chromedp + cli-imports + Obj-2 grep-gate green; addlicense -check green (fork ignored, proven load-bearing); `press/math/mathml.go` byte-identical; `go.mod` diff = replace directive only; `go.sum` diff empty (no other churn).
- No BLOCKER, no auth gate, 0 auto-fix cycles. Behavior identical to the external dep — the 5 converter patches are deferred to 08-02/08-03 by design.
