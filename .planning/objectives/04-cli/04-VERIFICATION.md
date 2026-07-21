---
status: passed
objective: 4
verified: 2026-07-21
score: 6/6 requirements, 6/6 CLI modes functionally verified
gaps: []
---

# Objective 4 Verification — CLI (cmd/eden-press)

**Verdict: PASSED.** Verified against the actual merged codebase on `main` (all 8 job commits merged: `04-01` through `04-08`, reconciled at `3e69060`). Every static gate — `gofmt`, `go build ./...`, `go vet ./...`, `go test ./cmd/... ./...` (fresh, whole repo, 0 failures), `bash scripts/check-no-chromedp.sh`, `bash scripts/check-cli-imports.sh` — passes clean. Beyond static analysis, every one of the 6 CLI modes (convert, watch, serve, preview, theme/theme-set, config) was **actually executed** against a compiled `eden-press` binary (or `go run`) in this session: piped stdin, custom themes, `.marprc.{yaml,json,toml}` config files with precedence, a live fsnotify watch-and-rebuild cycle, a bound HTTP server on port **8091** (never 8080) with a real directory-traversal probe, and the preview seam's unit tests.

## Gate results (run fresh, this session)

| Gate | Command | Result |
|------|---------|--------|
| Format | `gofmt -l cmd/ scripts/` | Empty output — exit 0 |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (whole repo) | `go test ./cmd/... ./...` | All 25 packages `ok` (incl. `cmd/eden-press`, `cmd/eden-press/reload`), 0 failures |
| No-chromedp CI gate | `bash scripts/check-no-chromedp.sh` | `PASS: no chromedp in the press/chase/profiles dependency closure.` exit 0 |
| CLI-imports boundary gate | `bash scripts/check-cli-imports.sh` | `PASS: cmd/eden-press imports only press/ (no direct chase/ profiles/ chromedp).` exit 0 |
| CLI-imports wired into Makefile | `grep check-cli-imports Makefile` | `.PHONY` target present, `check-cli-imports: bash scripts/check-cli-imports.sh` |
| CLI-imports wired into CI | `grep check-cli-imports .github/workflows/ci.yml` | `run: make check-cli-imports` step present (line 85), beside the existing no-chromedp step |
| addlicense | `addlicense -l mit -s -c "AO Cyber Systems" -check cmd/eden-press/ scripts/check-cli-imports.sh` | exit 0, no missing headers |
| Anti-pattern scan | `grep -rn TODO\|FIXME\|not implemented cmd/eden-press/` | 3 hits, all benign doc-comments in `flags.go` referencing which downstream TRD implements a default (all now merged) — no actual stubs |

## Requirements coverage

| REQ | Source TRD | Description | Evidence | Status |
|-----|-----------|-------------|-----------|--------|
| CLI-01 | 04-03 | `eden-press <in.md>` → default bare-style zero-JS static HTML | `cmd/eden-press/htmldoc.go` `assembleHTML()` (script-injection seam, zero `<script>` unless requested); `cmd/eden-press/convert.go` `runConvert()`. Tests: `TestAssembleHTMLZeroJSGolden`, `TestRunConvertFileToStdout`, `TestRunConvertStdinToStdout`, `TestRunConvertOutputFile`, `TestRunConvertAutoFitScript`. **Functional:** `printf '# Hi\n\nWorld' \| go run ./cmd/eden-press -` emitted a full `<!doctype html>…<style>…</style>…<body><section>…Hi…World…</section></body></html>` document; `grep -c "<script" out.html` = 0; `--auto-fit-script` flag adds exactly 1 `<script>` | ✅ |
| CLI-02 | 04-06 | Watch mode via fsnotify — rebuild on change | `cmd/eden-press/watch.go` `runWatch()` (parent-dir watch, name-filtered, debounced 300ms, atomic-save-safe); `cmd/eden-press/reload/` (stdlib SSE Hub, `client.js` embedded EventSource snippet). Tests: `TestDebounced`, `TestIsBackupOrSwap`, `TestEventTriggersRebuild` (8 subtests incl. atomic-rename/Chmod/backup-swap/sibling-file), `TestRunWatchRejectsStdin`, `TestRebuildOnceInjectsReloadClientOnly`, `TestRunWatchRebuildsOnAtomicSave`, `TestHubBroadcastDeliversReloadEvent`. **Functional:** ran `eden-press watch deck.md` in background; initial build produced `deck.html` containing "Hi"; edited `deck.md` to add "Updated"; `deck.html` was rewritten with "Updated" within ~1.2s; output contained exactly 1 `<script>` (`EventSource` reload snippet); log printed a non-8080 loopback reload URL (`http://127.0.0.1:55048/`); `eden-press watch -` errored cleanly ("cannot watch stdin") | ✅ |
| CLI-03 | 04-07 | Server mode with live-reload (serve local files, convert on request) | `cmd/eden-press/serve.go` `runServe()`/`safeJoin()` (traversal guard) reusing the 04-06 reload Hub + 04-03 assembleHTML. Tests: `TestServeConvertsMarkdownOnRequest`, `TestServeRejectsDirectoryTraversal` (2 subtests incl. URL-encoded `%2f`), `TestSafeJoinContainsResultUnderRoot` (6 subtests), `TestServeStaticFile`, `TestServeReloadEndpoint`, `TestServeIgnoresFormatSwitchQuery`, `TestResolveServeAddrDefaultsToNon8080Port`. **Functional (bound on port 8091, never 8080):** `eden-press serve --port 8091 serveroot`; `curl :8091/deck.md` returned rendered HTML containing "Served"; `curl :8091/plain.txt` returned the raw static file bytes; `curl :8091/../../../../tmp/outsideroot/secret.txt` → 404 (sentinel file never served); URL-encoded `..%2f..%2f..%2f..%2ftmp%2foutsideroot%2fsecret.txt` → 400; `curl -i :8091/__reload` → `200 text/event-stream`; `:8091/deck.md?pdf` byte-identical to `:8091/deck.md` (no format-switching, confirmed by `diff`) | ✅ |
| CLI-04 | 04-08 | Preview (open output in default browser) | `cmd/eden-press/preview.go` `runPreview()` + injectable `var openURL = browser.OpenURL` seam (`github.com/pkg/browser`, no chromedp). Tests: `TestPreviewOpensRenderedFile` (asserts `openURL` called with a `file://…eden-press-*.html` path whose contents are the standalone doc), `TestPreviewRejectsStdin`. Both PASS. **Note:** a literal OS browser window opening was not exercised (see Human Verification) — the seam, temp-file write, and URL target are all code- and test-verified | ✅ |
| CLI-05 | 04-01 + 04-05 | `--theme` / `--theme-set` loading | `press/options.go` `Options.ThemeCSS []string` (additive field, 04-01) threaded through `press/press.go` `packThemeCSS` via `theme.Load`+`ts.Add`; `cmd/eden-press/themeset.go` `themeCSS(cmd)` (reads `--theme-set` file paths → raw CSS text); `cmd/eden-press/options.go` `buildOptions` wires `themeCSS`'s result into `Options.ThemeCSS`. Tests: `TestThemeCSSMultiFile`, `TestThemeSetEndToEnd`, `TestThemePassThroughBundled`, `TestThemeCSSMissingFile`, `TestThemeSetMalformedErrorsAtRender`, plus press-side `TestThemeCSSAdditive`/`TestBrowserFitJSReexport`. **Functional:** a hand-written `brand.css` (`/* @theme brand */ section { color: #d4a853; }`) selected via front-matter `theme: brand` + `--theme-set brand.css` produced `Output.CSS` containing `color: #d4a853`; `--theme gaia` vs `--theme default` produced measurably different CSS (105 vs 259 lines, confirmed via `diff`) | ✅ |
| CLI-06 | 04-02 + 04-04 | Config file (YAML/JSON/TOML via koanf) + stdin input (`-`) | `cmd/eden-press/config.go` `loadConfigSources` (file[ext-routed]→env→posflag-LAST, Pitfall-5 instance guard) + `discoverConfigPath`/`parserFor`; `cmd/eden-press/input.go` `resolveInput`/`resolveInputFrom` (stdin `-` vs file). Tests: `TestConfigFileToOptions`, `TestPrecedenceFlagOverFile`, `TestPrecedenceEnvOverFileFlagOverEnv`, `TestExtRouting` (json/toml/unsupported-ext subtests), `TestConfigFlagOverridesDiscovery`, `TestPitfall5GuardThroughFullChain`, `TestResolveInputStdin`, `TestResolveInputFile`, `TestResolveInputMissingFile`. **Functional:** built binary + ran in a scratch dir: `.marprc.yaml` (`theme: gaia`) alone → output byte-identical to `--theme gaia` explicit flag; `--theme uncover` (flag) overrode the YAML file's `gaia`; a `.marprc.json` (`theme: uncover`) matched the `--theme uncover` flag output; a `.marprc.toml` (`theme = "gaia"`) matched the YAML case — YAML/JSON/TOML all parse and apply correctly with flags-over-file precedence confirmed by `diff` | ✅ |

**6/6 requirement IDs have a concrete, tested artifact on disk AND were functionally exercised. Zero orphans** — every ID in `.planning/REQUIREMENTS.md`'s Objective-4 mapping (CLI-01..06) is claimed by exactly one TRD's frontmatter `requirements:` field (04-03→CLI-01, 04-06→CLI-02, 04-07→CLI-03, 04-08→CLI-04, 04-01+04-05→CLI-05, 04-02+04-04→CLI-06) and traced above. 04-01 and 04-02 (`requirements: []`) are correctly foundational/shared-skeleton TRDs whose work is claimed by the downstream TRDs that consume their seams.

## CLI-imports boundary (mechanical enforcement)

`scripts/check-cli-imports.sh` checks `cmd/eden-press`'s OWN direct imports (`.Imports`, not transitive `-deps`) and fails if any directly import `chase/`, `profiles/`, or `chromedp`. Confirmed:
- `bash scripts/check-cli-imports.sh` → PASS this session.
- `grep check-cli-imports Makefile` → `.PHONY` entry + target present (`Makefile:23`, `:48-49`).
- `grep check-cli-imports .github/workflows/ci.yml` → wired as a CI step (`ci.yml:85`, `run: make check-cli-imports`), beside the pre-existing `check-no-chromedp` step.
- `go list -f '{{join .Imports "\n"}}' ./cmd/eden-press/...` shows only `press/` (+ subpackages), `cmd/eden-press/reload`, cobra/koanf/fsnotify/pkg-browser, and stdlib — no direct `chase/`, `profiles/`, or `chromedp`.

## Anti-patterns found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/eden-press/flags.go` | 53, 60, 67 | Doc-comment phrase "not implemented here" | ℹ️ Info | Benign — these are flag-registration doc comments pointing to the (now-merged) downstream TRD that implements the default-value resolution (e.g., "`-o` default resolves stdout — 04-03's default, not implemented here" in the flags-registration file). No actual stub code; `runConvert`/`runWatch`/`runServe`/`runPreview` are all fully implemented, tested, and functionally verified above. |

No blockers. No placeholder/`return nil`/console-log-only implementations found anywhere in `cmd/eden-press/`.

## Notes (non-blocking)

- **`.planning/REQUIREMENTS.md` checkbox drift (tracking-only, not a code gap):** the top checklist (lines 61-66) still shows `[ ]` (Pending) for CLI-02, CLI-05, and CLI-06, while the traceability table further down the same file (lines 167-172) correctly lists all of CLI-01 through CLI-06 as "Complete" mapped to Objective 4. This is the same documentation-reconciliation pattern flagged in the Objective-3 verification (checkbox section lagging the traceability table / merged code) — not a functional gap. All 6 requirements are implemented, tested, and functionally verified above. Recommend a follow-up `docs` commit to flip the top-checklist boxes to `[x]` to match the traceability table and the actual code.
- **fsnotify cross-platform scope:** per 04-06-TRD's own documented scope, the atomic-save-safe watch behavior was authored against, and this session's functional check ran on, macOS. Linux/Windows fsnotify semantics are noted in the TRD as tracked-but-not-re-verified-per-OS; this is a stated v1 scope boundary, not a regression.
- **go.mod `// indirect` markers on directly-imported CLI deps** (cobra, fsnotify, koanf, pkg/browser): these carry `// indirect` comments in `go.mod` even though `cmd/eden-press` imports them directly. This is a cosmetic staleness from `go get`-only provisioning (04-02's TRD explicitly forbids `go mod tidy` to avoid touching other objectives' dependency trees) — `go build`/`go vet`/`go test` all pass clean regardless, and the comment has zero effect on resolution or the build graph. Not a functional gap.

## Human Verification (optional, non-blocking)

### 1. Preview opens a real browser window

**Test:** Run `eden-press preview <deck>.md` on a desktop with a default browser configured.
**Expected:** The OS default browser opens a new tab/window showing the rendered standalone HTML deck.
**Why human:** `TestPreviewOpensRenderedFile`/`TestPreviewRejectsStdin` prove the `openURL` seam is called with the correct `file://` target and that the temp file's contents are the correct standalone doc — but actually spawning and visually confirming a browser window is a manual, environment-dependent step outside what an automated agent should trigger unprompted.

## Gaps

None. All 6 requirements verified present with passing tests AND functional runtime evidence; all static gates green; the CLI-imports boundary gate is mechanically wired into Makefile + CI; no blocker anti-patterns found.

---

*Verified: 2026-07-21*
*Verifier: Claude (verifier)*
