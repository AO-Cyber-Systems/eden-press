---
status: passed
objective: 2
verified: 2026-07-21
score: 4/4 requirements, 4/4 success criteria
---

# Objective 2 Verification — chase/model + chase/profile + profiles/slides

**Verdict: PASSED.** Verified against the actual codebase (whole-repo CI gate run). `go build ./...`, `go vet ./...`, `go test ./...` (all 13 packages), `addlicense -check`, and every prior gate (Obj-1 `TestChaseCorpus` + `pack_conformance_test.go` cssdiff, Obj-2 `TestGrepGate`) all pass.

## Success criteria / requirements

| REQ | Criterion | Evidence | Status |
|-----|-----------|----------|--------|
| MODEL-01 | `Document{Meta,Sections,Outline}` built by direct walk of the SAME finalized AST that produces HTML (not a 2nd parse) | `chase/model` (document.go/build.go); test proves HTML byte-identical before/after `model.Build` | ✅ |
| MODEL-02 | One entrypoint returns HTML+CSS+Model from a SINGLE parse pass | `chase/chase.go` `chase.Render` → `Output{HTML,CSS,Model,Meta}`; `TestOneParseTwoSinksHTMLUnaffectedByModelBuild` + 4-corpus-deck validation | ✅ |
| MODEL-03 | `chase/profile` exposes a `Profile` interface + registry, validated bottom-up (no speculative superset) | `chase/profile` (profile.go/registry.go); Register/Get/Default tested; `Boundary()/Directives()` deferred | ✅ |
| MODEL-04 | `profiles/slides` the only Profile impl (Marp behavior); no slide-specific naming/`section`/size constants in chase/model+theme | `profiles/slides`; `chase/theme` params threaded from profile; `TestGrepGate` (CI-enforceable) green | ✅ |

## Decision gate resolved
- **`chase/*` packages are EXPORTED** (no `internal/` prefix) — documented in `chase/profile/doc.go`. Per the library-first thesis (advanced consumers / future profiles can import chase directly).

## Notes
- The de-hardcode (MODEL-04) was a behavior-preserving value relocation; the safety net is the byte-identical cssdiff + corpus gates staying green (they do). A `TestGrepGate` test enforces the purity going forward; any narrowly-justified residual literal is whitelisted there.
- The output-profile abstraction (`chase/profile` + `profiles/slides`) + structured model (`chase/model`) are now first-class packages — differentiators #1 and #3 are load-bearing, so future profiles (paged/EPUB) are additive, not a rewrite.

**4/4 jobs complete with SUMMARYs + self-checks. Objective 2 achieves its goal.**
