# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-20)

**Core value:** Render Marp-compatible documents from Markdown inside a Go service or CLI with no JavaScript runtime, no Node, and no browser for HTML/structured output — while emitting the document as structured data, not just HTML.
**Current focus:** Objective 3 COMPLETE (9/9 TRDs); Objectives 4/5/6/7 unblocked and executing in parallel worktreams — this worktree executed Objective 6 (convert/pptx) TRD 06-03

## Current Position

Objective: 3 of 9 (press/ Batteries + Public API) — COMPLETE (9/9 TRDs); Objectives 0-2 complete. Objective 6 (convert/pptx) now 2/5 TRDs complete (this worktree: 06-02, 06-03) — parallel workstream, pending orchestrator reconcile at merge.
Job: Objective 6, TRD 03 of 5 (wave 2, depends_on 06-02 in-worktree). This worktree's task: 06-03 deterministic OPC zip packager + complete static part graph + trivial-deck structural openability at 16:9/4:3 (EXP-03, partial). Pending orchestrator reconcile at merge.
Status: 06-03-TRD executed on this worktree — package.go (fixed-Modified/zip.Store/explicit-order deterministic zip assembler), contenttypes.go ([Content_Types].xml Default+Override manifest builder/parser), parts_static.go (the complete minimal OPC part graph: _rels/.rels, docProps, presentation.xml+rels, presProps/viewProps/tableStyles, theme1 with 12-color clrScheme + exactly-3-entry fmtScheme lists, slideMaster1+rels with the mandatory 12-attr clrMap, slideLayout1+rels), and openable_test.go (the reusable structural openability asserter: content-types coverage + full .rels Target/r:id resolution closure, proven on a trivial hardcoded title-box deck at BOTH 16:9 and 4:3). 3 task commits (all GREEN on first attempt, 0 auto-fix cycles during the tasks themselves) plus 1 post-task gofmt-alignment fix (Rule 1 - Bug, found during the final full-repo gate sweep). All gates green: gofmt, go build ./..., go vet ./..., go test ./... (whole repo), check-no-chromedp.sh, isolation gate (convert/pptx at 0 references from press/chase/profiles), addlicense; go.mod/go.sum untouched (stdlib only). Objective 6 at 2/5 TRDs — 06-04 (ToPPTX writer) and 06-05 (notes + comprehensive verification) now consume this packaging foundation.
Last activity: 2026-07-21 — 06-03-TRD executed (Objective 6, wave 2): convert/pptx's OPC packaging layer built test-first for Tasks 1-2 (RED->GREEN, 0 auto-fix cycles) and standard-verified for Task 3 — Task 1 (deterministic zip + Content-Types, commit eb2ada9), Task 2 (complete static part set incl. clrMap/fmtScheme, commit 5a98584), Task 3 (trivial-deck structural openability at 16:9/4:3, commit b209398), plus a post-task gofmt-alignment fix (commit 95fe34c); SUMMARY committed at 022762c.

Progress: [██████████] 100% (27/27 TRDs across Objectives 0-3, fully complete); Objective 6 (convert/pptx): 2/5 TRDs complete on this worktree (06-02, 06-03)

## Accumulated Context

> Decisions and performance metrics are logged in STATE_ARCHIVE.md.

### Pending Todos

None yet.

### Blockers/Concerns

- Decision gate open: `chase/*` internal vs. exported Go (resolve during Objective 2 planning).
- Decision gate re-confirmed at 06-02 execution: hand-rolled OOXML vs. any newly-emerged permissive Go PPTX lib — 06-RESEARCH.md (dated 2026-07-21) re-verified no new permissive Go PPTX library has emerged; `unioffice`/forks remain rejected (AGPLv3/commercial license-key). No new dependency added.
- Decision gate open: standard Go vs. TinyGo for the WASM target (resolve before Objective 7's WASM-specific code is written — functional risk on reflection-heavy JSON/YAML paths, not just size).
- Decision gate open: concrete MathML fallback-trigger rule + final auto-fit mechanism (resolve during Objective 8 planning).
- CSS-AST diff tooling (Objective 0) is genuinely new/unproven engineering — no spike precedent exists; budget accordingly at planning time.

## Session Continuity

Last session: 2026-07-21 (06-03-TRD execution — Objective 6 wave 2: convert/pptx deterministic OPC packager + static part graph + trivial-deck openability)
Stopped at: Completed 06-03-TRD.md (EXP-03, partial): built convert/pptx's OPC packaging layer atop 06-02's EMU/slide-size foundation. `package.go`: `buildZip([]part) ([]byte, error)` -- fixed `FileHeader.Modified` (1980-01-01), `zip.Store`, explicit ordered `[]part` (never a map) -- byte-identical rebuilds. `contenttypes.go`: `buildContentTypesXML` ([Content_Types].xml Default+Override manifest) + `parseContentTypesXML`/`.covers` (coverage-checking reader). `parts_static.go`: every invariant part in the minimal OPC graph (_rels/.rels, docProps/core+app, presentation.xml+rels, presProps/viewProps/tableStyles, theme1 with full 12-color clrScheme + exactly-3-entry-per-list fmtScheme, slideMaster1+rels with the mandatory 12-attr `<p:clrMap>`, slideLayout1+rels), plus N-slide-ready `slideRef`/`presentationXML`/`presentationRelsXML` plumbing sizing `<p:sldSz>`/`<p:notesSz>` from 06-02's `SlideSize16x9`/`SlideSize4x3`/`NotesSize`. `openable_test.go`: the reusable structural openability asserter (`assertStructurallyOpenable`) -- content-types coverage + full `.rels` Target/r:id-resolution closure -- proven on a trivial hardcoded title-box deck (test-only `buildTrivialDeck`) at BOTH 16:9 and 4:3 (criterion 4), plus a SKIP-guarded optional LibreOffice-headless smoke. 3 task commits (eb2ada9, 5a98584, b209398; Tasks 1-2 TDD RED->GREEN with 0 auto-fix cycles, Task 3 standard/non-TDD per its TRD-declared type, GREEN on first attempt) plus 1 post-task gofmt-alignment fix (95fe34c, Rule 1 - Bug, found during the final full-repo gate sweep, whitespace-only). Gates: gofmt, go build ./..., go vet ./..., go test ./... (whole repo, all green), check-no-chromedp.sh PASS, isolation gate (`go list -deps ./press/... ./chase/... ./profiles/...` contains 0 convert/pptx references) PASS, addlicense PASS, go.mod/go.sum diff empty (zero new deps, stdlib only). SUMMARY committed at 022762c. This worktree ran as Objective 6's wave 2 (depends_on 06-02, already merged in-worktree). 06-04 (ToPPTX writer) and 06-05 (notes + comprehensive verification) are unblocked to consume this packaging foundation. Objective 6 now 2/5 TRDs complete (this worktree: 06-02, 06-03) — pending orchestrator reconcile at merge.
Resume file: None
