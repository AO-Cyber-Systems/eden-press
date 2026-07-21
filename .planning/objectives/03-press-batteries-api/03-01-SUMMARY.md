---
objective: 03-press-batteries-api
job: "01"
subsystem: press-public-api
tags: [go, goldmark, chroma, bluemonday, emoji, api-surface, additive-seam, tdd, spike]

# Dependency graph
requires:
  - objective: 02-model-profile
    provides: "chase/markdown.Parse + NewEngine (the two-phase seam this TRD parameterizes), chase.Output shape (HTML/CSS/Model/Meta mirrored by press.Output), chase/model.Document + Meta + Section.Notes (source of press.Output.Model/Meta/Comments)"
provides:
  - "chase/markdown.ParseWithEngine(md string, engine goldmark.Markdown) (*ast.Document, parser.Context) — the ADDITIVE, engine-parameterized twin of Parse; identical SvgOptionsKey + resolved-HeadingDividerKey pre-seed, parses via a CALLER-supplied engine. The one-parse composition seam press.Render (03-09) drives; chase.Render/seam.go untouched"
  - "press.Options — the frozen API-03 input surface (Theme/Profile/InlineSVG/MathMode/NoHighlight/HighlightStyle/Sanitize) whose ZERO VALUE is the Marp-Core-matching default (NoHighlight inverted => highlighting ON; Sanitize nil => built-in always-on policy)"
  - "press.Output — the frozen API-03 output contract (HTML/CSS/Model *model.Document/Meta model.Meta/Comments []string) Objective 7's Dart binding serializes over"
  - "go.mod OWNED for Objective 3: all six battery deps provisioned additively so wave-2 battery TRDs only import, never edit go.mod"
  - "goldmark-emoji v1.0.6 ↔ goldmark v1.8.4 build-compat PROVEN (research riskiest-item #3 closed) — CORE-06 (03-04) may reuse goldmark-emoji"
  - "press/ package skeleton with the no-chromedp invariant enforced from the first file (go list -deps ./press/... has 0 chromedp)"
affects:
  - "Every wave-2 battery TRD (03-02..03-08) imports press.Options/press.Output and the six deps without touching go.mod"
  - "03-09 compose TRD calls ParseWithEngine(md, NewEngine(pressExtraOpts...)) and defines press.Render returning press.Output"
  - "03-06 (math) consumes codeberg.org/go-latex/latex — provisioned here (v0.3.0), NOT deferred"

# Tech tracking
tech-stack:
  added:
    - "github.com/alecthomas/chroma/v2 v2.27.0 (research-recommended; cached locally, so pinned above the v2.25.0 fallback — has html.WithClasses, HighlightLines, WriteCSS)"
    - "github.com/yuin/goldmark-emoji v1.0.6"
    - "github.com/yuin/goldmark-highlighting/v2 v2.0.0-20230729083705-37449abec8cc"
    - "github.com/microcosm-cc/bluemonday v1.0.27"
    - "git.sr.ht/~mekyt/latex2mathml v0.0.0-20231214134936-808832af73fc"
    - "codeberg.org/go-latex/latex v0.3.0 (network-provisioned this session — NOT deferred; see Deviations)"
    - "(transitive, additive via go get: dlclark/regexp2/v2, aymerick/douceur, gorilla/css, neruyzo/etree)"
  patterns:
    - "Additive engine-parameterized seam: ParseWithEngine is a byte-for-byte copy of seam.go's Parse except engine.Parser().Parse(...) replaces defaultEngine.Parser().Parse(...) — a NEW file, zero edits to Parse/Render/RenderDoc/NewEngine, so every existing chase caller stays byte-identical (git diff on seam.go/chase.go/renderdoc.go is empty)"
    - "Zero-value-is-safe-default API shaping: press.Options{} is a valid Marp-matching config. NoHighlight is INVERTED (false => highlighting ON) and Sanitize is nil-defaulted (nil => built-in always-on policy, NOT off) so the zero value is never the unsafe/off value"
    - "Compile-fence test: keyed composite literals naming EVERY Options/Output field lock the API-03 surface — a rename/removal fails the build, protecting Objective 7's binding from silent churn"
    - "Spike-gate-before-dependence: the goldmark-emoji↔goldmark v1.8.4 compat risk (research riskiest-item #3) is closed by a passing build+render test HERE, before CORE-06 commits to reusing the library"
    - "Central dep ownership: go.mod is provisioned once in the wave-1 foundation TRD so the parallel wave-2 battery worktrees never conflict on go.mod/go.sum (go get, NEVER go mod tidy)"

key-files:
  created:
    - chase/markdown/parse_engine.go
    - chase/markdown/parse_engine_test.go
    - press/doc.go
    - press/options.go
    - press/options_test.go
    - press/deps_spike_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "chroma pinned at v2.27.0 (research-recommended), NOT the v2.25.0 fallback: the local module cache actually carries v2.27.0 (the TRD's 'cache tops out at v2.25.0' assumption was stale), so the higher research-recommended version was used. v2.27.0 has every feature the objective needs (html.WithClasses, HighlightLines, WriteCSS)"
  - "codeberg.org/go-latex/latex was PROVISIONED (v0.3.0), not deferred: the network was reachable this session (proxy.golang.org 200), so `go get codeberg.org/go-latex/latex@latest` succeeded and resolved v0.3.0. The TRD's offline-defer BLOCKER path (defer to 03-06) was therefore NOT taken — 03-06 finds its dep already present"
  - "ParseWithEngine parity proven by byte-identical rendered HTML, not an os.Stdout doc.Dump capture: rendering both the ParseWithEngine tree (via the caller engine) and the Parse tree (via defaultEngine) and asserting byte-equality is a stronger, capture-free faithfulness proof and also demonstrates the caller-engine render path 03-09 will use"
  - "New deps land as `// indirect` in go.mod (nothing in non-test code imports chroma/highlighting/latex/latex2mathml yet, and go mod tidy is FORBIDDEN so the annotation is not rewritten). This is cosmetic — go build/vet/test are all green with them present; wave-2 TRDs that import them will drop the annotation via their own go get if needed"

requirements-completed: [API-03]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: true
  test_pairing: true
  blockers: none

# Metrics
duration: ~8min
completed: 2026-07-21
---

# Objective 03 TRD 01: press/ Foundations — ParseWithEngine seam + frozen API-03 surface + battery deps Summary

**The two foundations every other Objective-3 TRD stands on now exist: (1) `chase/markdown.ParseWithEngine` — an ADDITIVE, engine-parameterized twin of `Parse` (identical `SvgOptionsKey` + resolved `HeadingDividerKey` pre-seed, but parses through a caller-supplied engine) that leaves `chase.Render`/`seam.go` byte-for-byte untouched; and (2) the `press/` package with the frozen `Options`/`Output` API-03 types (zero value = Marp-matching default) plus all six battery dependencies provisioned into `go.mod`. The goldmark-emoji v1.0.6 ↔ goldmark v1.8.4 compat spike (research riskiest-item #3) is closed with a passing test, and `codeberg.org/go-latex/latex` was provisioned online (v0.3.0) — no BLOCKER, no deferral.**

## What was built

### Task 1 — six battery deps + goldmark-emoji compat spike (commit 8af93ad)
- `go get` (additive, NEVER `go mod tidy`) provisioned: chroma/v2 v2.27.0, goldmark-emoji v1.0.6, goldmark-highlighting/v2, bluemonday v1.0.27, latex2mathml, and codeberg.org/go-latex/latex v0.3.0.
- `press/doc.go`: package documentation for the API-03 public package; states the sibling-composition-not-wrapper relationship to `chase.Render` and the no-chromedp invariant.
- `press/deps_spike_test.go` → `TestGoldmarkEmojiCompat`: builds `goldmark.New(WithExtensions(emoji.New(WithRenderingMethod(Twemoji))))`, converts `:smile:`, asserts a twemoji `<img class="emoji" ... twemoji ...>` (not the literal `:smile:`). Proves v1.0.6 builds AND runs against goldmark v1.8.4 — riskiest-item #3 closed BEFORE CORE-06 depends on it. It built + ran green (no API break; the go.mod goldmark v1.7.10 floor is a version bump only, expected under Go MVS).

### Task 2 — additive ParseWithEngine seam (commit 9b74483)
- `chase/markdown/parse_engine.go` → `ParseWithEngine(md string, engine goldmark.Markdown) (*ast.Document, parser.Context)`: verbatim copy of `seam.go`'s `Parse` pre-seed (`SvgOptionsKey{Enabled:true}` + `frontMatterHeadingDividerLevels`→`HeadingDividerKey`), reusing the already-package-private helper, with `engine.Parser().Parse(...)` the ONLY substitution. Returns the finalized AST + context inspectable between phases; the caller renders via `engine.Renderer().Render(&buf, source, doc)`.
- `chase/markdown/parse_engine_test.go` (Test-list cases 1–3): parity-with-Parse (same Section count + byte-identical rendered HTML), pre-seed parity (SvgOptionsKey enabled; HeadingDividerKey `[]int{1,2}` for `headingDivider: 2`, identical to Parse), and additive-non-breaking (ParseWithEngine render == existing `Render(md, nil)`).
- `chase.go`/`seam.go`/`renderdoc.go` untouched (`git diff --stat` empty).

### Task 3 — frozen press.Options / press.Output (TDD, commit 60a12a4)
- `press/options.go`: `Options{Theme, Profile, InlineSVG, MathMode, NoHighlight, HighlightStyle, Sanitize *bluemonday.Policy}` and `Output{HTML, CSS, Model *model.Document, Meta model.Meta, Comments []string}`, each field documented with its zero-value meaning.
- `press/options_test.go` (RED→GREEN): zero-value default table (`Theme→""→"default"`, `MathMode→""→"mathml"`, `NoHighlight=false→ON`, `Sanitize=nil→built-in policy`), an `Output` zero-value/`[]string` Comments check, and a keyed-literal compile-fence naming every field to lock the surface.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: deps + emoji spike | `go build ./... && go test ./press/ -run TestGoldmarkEmojiCompat -v && go vet ./...` | 0 | PASS |
| 2: ParseWithEngine | `go test ./chase/markdown/... -run ParseWithEngine -v && go test ./chase/... && gofmt -l chase/markdown/parse_engine.go` | 0 | PASS |
| 3: Options/Output | `go build ./press/... && go test ./press/ -run 'TestOptions\|TestOutput' -v && go vet ./press/...` | 0 | PASS |

## TDD Evidence (Task 3)

| Phase | Command | Exit Code | Expected |
|---|---|---|---|
| RED | `go test ./press/ -run 'TestOptions\|TestOutput'` | 1 | FAIL — `undefined: Options` / `undefined: Output` (correct) |
| GREEN | `go test ./press/ -run 'TestOptions\|TestOutput' -v` | 0 | PASS — 3/3 (correct) |
| REFACTOR | (none needed — implementation is minimal/documented) | — | — |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| build | `go build ./...` | 0 | PASS |
| vet | `go vet ./...` | 0 | PASS |
| test (whole repo) | `go test ./...` | 0 | PASS (all packages ok) |
| gofmt | `gofmt -l .` | 0 | PASS (empty output) |
| addlicense (Go source) | `find . -name '*.go' \| xargs addlicense ... -check` | 0 | PASS (all .go licensed) |
| emoji compat spike | `go test ./press/ -run TestGoldmarkEmojiCompat` | 0 | PASS (riskiest-item #3 closed) |
| Obj-1 corpus/cssdiff | `go test ./conformance/...` | 0 | PASS (corpus, cssdiff, htmldiff, runner ok) |
| Obj-2 grep-gate | `go test ./profiles/slides/ -run TestGrepGate` | 0 | PASS |
| additive seam | `git diff --stat chase/markdown/seam.go chase/chase.go chase/markdown/renderdoc.go` | 0 | PASS (empty — untouched) |
| no-chromedp invariant | `go list -deps ./press/... \| grep -c chromedp` | — | PASS (count = 0) |

Note: `addlicense -check .` over the WHOLE tree exits 1, but ONLY because pre-existing corpus test-data fixtures (`conformance/corpus/cases/*/expected.{html,css}`, tracked on main since 9488227) carry no license header. Every `.go` source file — including all six files created here — passes the license gate. This is pre-existing data-fixture noise, not a regression from this TRD.

## Deviations from Plan

### Auto-resolved / recorded

**1. [Rule 3 - Blocking check resolved favorably] codeberg.org/go-latex/latex provisioned online, NOT deferred**
- **Found during:** Task 1. The TRD's error_recovery anticipated this module being absent from the local cache (it was) and instructed deferring it to 03-06 with a BLOCKER if the network was unreachable.
- **Resolution:** The network WAS reachable this session (proxy.golang.org returned 200; the module list showed v0.1.0–v0.3.0). `go get codeberg.org/go-latex/latex@latest` succeeded and resolved **v0.3.0**. All six deps are in go.mod; 03-06 finds its dep already present. **No BLOCKER.**

**2. [Decision] chroma pinned at v2.27.0 (research-recommended), not the v2.25.0 fallback**
- **Found during:** Task 1. The TRD assumed the local cache "tops out at v2.25.0" and said to fall back to v2.25.0 if v2.27.0 could not fetch.
- **Resolution:** The cache actually carried v2.27.0 (assumption stale), so the research-recommended v2.27.0 was pinned. It satisfies every feature the objective needs. Documented in key-decisions.

No other deviations — `ParseWithEngine`, `press.Options`/`press.Output`, and the emoji spike landed exactly as the TRD specified.

## Authentication gates

None encountered.

## Post-TRD Verification

- Auto-fix cycles used: 0
- Must-haves verified: 5/5 (ParseWithEngine additive+pre-seed-faithful; press.Options{} zero-value = Marp default; press.Output stable contract; six deps additive with build green; goldmark-emoji compat proven)
- Gate failures: None (whole-tree addlicense exit-1 is pre-existing non-Go fixture noise, explained above)
- Blockers: None (go-latex provisioned online)

## Commits

- `8af93ad` chore(03-01): provision six battery deps + close goldmark-emoji compat spike
- `9b74483` feat(03-01): add chase/markdown.ParseWithEngine (additive one-parse seam)
- `60a12a4` feat(03-01): define frozen press.Options / press.Output API-03 surface

## Self-Check: PASSED

- Files verified on disk (7/7): `chase/markdown/parse_engine.go`, `chase/markdown/parse_engine_test.go`, `press/doc.go`, `press/options.go`, `press/options_test.go`, `press/deps_spike_test.go`, `.planning/objectives/03-press-batteries-api/03-01-SUMMARY.md` — all FOUND.
- Commits verified in `git log` (3/3): `8af93ad`, `9b74483`, `60a12a4` — all FOUND.
- All 3 TRD `<verify>` gates PASS; whole-repo `go build`/`go vet`/`go test`/`gofmt -l` clean; Go-source addlicense clean; Obj-1 conformance + Obj-2 grep-gate green; emoji compat spike PASS; seam additive (empty diff); no-chromedp count 0.
- No BLOCKER: all six deps (incl. codeberg.org/go-latex/latex v0.3.0) provisioned online.
