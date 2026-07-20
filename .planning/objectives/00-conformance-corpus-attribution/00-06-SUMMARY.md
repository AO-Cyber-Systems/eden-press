# 00-06 Summary — CSS-AST diff comparator (CONF-03)

**Requirement:** CONF-03 · **Wave:** 3 · **Status:** Complete

## What was built

The **CSS-AST diff comparator** built on the 00-03 spike's normalized `Stylesheet` model — the CSS
analogue of `conformance/htmldiff.Equal`. It is the theme-CSS half of the acceptance gate: cosmetic
formatting is ignored, every semantically meaningful difference is caught.

- **`conformance/cssdiff/diff.go`** — `Equal(expected, actual string) (bool, string)`: parses both
  via `cssdiff.Parse` and compares the models, returning a human-readable unified-style diff on
  mismatch. Comparison semantics:
  - **Format-insensitive** — whitespace, indentation, and comments are erased by parsing.
  - **Order-sensitive** — rule order and in-rule declaration order are preserved and significant
    (the CSS cascade is order-dependent; Marpit output is deterministic, so a reorder is a real change).
  - **Value / selector / `!important`-sensitive** — the cascade-meaningful bits are all compared.
- **`conformance/cssdiff/diff_test.go`** — the CONF-03 negative-test gate (all green):
  - broken-theme: **changed value**, **dropped `!important`**, **changed selector**, added declaration,
    added rule → all correctly reported as NOT equal.
  - cascade-significant **rule reorder** and **same-property declaration reorder** → NOT equal.
  - positive controls: identical CSS and a whitespace/comment-only reformat → equal.

## key-files.created
- conformance/cssdiff/diff.go
- conformance/cssdiff/diff_test.go

## Task Evidence
| Check | Command | Result |
|---|---|---|
| Comparator tests | `go test ./conformance/cssdiff/ -v` | PASS — 9 comparator + all 00-03 spike tests |
| Build / full test / vet | `go build ./...` · `go test ./...` · `go vet ./conformance/cssdiff/` | all exit 0 |
| License | `addlicense -l mit -s -c "AO Cyber Systems" -check` (both new files) | exit 0 |
| Negative gate | broken-theme (value/`!important`/selector) + reorder | all report `Equal == false` |

## Deviations
None. The comparator reuses the 00-03 spike's `Parse`/`Stylesheet` verbatim (spike → comparator
sequencing held); no changes to the model were needed.

## Self-Check: PASSED
Both files present and committed (`7129322`); comparator + negative tests green; build/test/vet/addlicense all pass. CONF-03 (CSS-AST diff comparator that catches an intentionally-broken theme) satisfied.
