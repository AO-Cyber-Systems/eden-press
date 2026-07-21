# Objective 4: CLI (cmd/eden-press) - Research

**Researched:** 2026-07-21
**Domain:** Go CLI engineering — command-tree design (cobra), file-watching (fsnotify), config loading (koanf), zero-JS-backend live-reload, cross-platform browser launch — as a thin consumer of the frozen `press/` library.
**Confidence:** HIGH overall. Stack choices, the live-reload mechanism, the sanitizer/script boundary, and the cobra default-command pattern are HIGH (verified against upstream source + official docs). Exact watch scope/debounce interval and the `--theme-set` API shape are flagged MEDIUM/LOW and listed as Open Questions — they are genuine unresolved design points, not gaps in research effort.

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| CLI-01 | `eden-press <in.md>` → default `bare`-style zero-JS static HTML | Architecture Pattern 1 (command tree) + Pattern 2 (bare HTML assembly from `press.Output`) |
| CLI-02 | Watch mode (`fsnotify`) — rebuild on change | Architecture Pattern 3 (scoped, non-recursive-by-default watch design) + Pitfalls 2, 6, 8 |
| CLI-03 | Server mode with live-reload (serve local files, convert on request) | Architecture Pattern 4 (SSE reload channel) + Pattern 5 (serve-and-convert-on-request + traversal guard) |
| CLI-04 | Preview (open output in a browser) | Standard Stack → `github.com/pkg/browser` (verified source, no headless-Chrome dependency) |
| CLI-05 | `--theme`/`--theme-set` loading | Architecture Pattern 2 (theme resolution) + **Open Question 1 (blocking-risk finding: `press.Options` has no custom-theme-CSS hook today)** |
| CLI-06 | Config file (YAML/JSON/TOML via koanf) + stdin (`-`) input | Architecture Pattern 6 (koanf precedence chain + format-by-extension) + Pitfall 9 (stdin/watch interaction) |

</phase_requirements>

## Summary

Objective 4 is a thin `cmd/eden-press` binary wired against the now-frozen `press.Render(md string, opts press.Options) (press.Output, error)` API. Nothing about this objective requires new parsing/theme/battery engineering — the risk surface is entirely in CLI plumbing: command-tree ergonomics (cobra), a correctly-scoped file watcher (fsnotify has real, documented sharp edges), a JS-free-by-default HTML document assembled from `Output.HTML`+`Output.CSS`, and a live-reload channel that does not compromise the "no JS runtime in the backend" thesis (the reload helper is a small viewer-side script, not a backend JS engine — same distinction Marp CLI itself draws between `bare` and `bespoke`).

Direct inspection of Marp CLI's own source (`marp-team/marp-cli`, fetched via GitHub API for this research) resolves several of the open design questions definitively: the `bare` template's `block script` is **empty** — it carries **zero** script by default, and the live-reload `<script>` is injected as a **separate, additive layer** only when a watch/serve session actually exists (`watchJs` is `false` otherwise). This is the exact behavior Eden Press should replicate, and it's now confirmed rather than assumed.

The single most important finding of this research pass is a **concrete gap, not a research gap**: `press.Options` (frozen in Objective 3, `press/options.go`) has `Theme string` (a *name* resolved against the embedded 3-theme `ThemeSet`) but **no field for injecting caller-supplied theme CSS**. `--theme-set` (CLI-05) cannot be implemented as a CLI-only feature without either (a) a small, additive `press.Options` field — explicitly permitted by that struct's own doc comment ("add a field only when a named consumer needs it") — or (b) violating the Objective-3 CLI-must-import-only-`press/` boundary by reaching into `chase/theme` directly. This must be resolved as a planning decision before CLI-05 tasks are written, not discovered mid-implementation.

**Primary recommendation:** cobra root command with a catch-all default `Args: cobra.MaximumNArgs(1)` action (mirroring Hugo's `hugo` vs. `hugo server` split) for `convert`, plus explicit `watch`/`serve`/`preview` subcommands; koanf with `posflag`+`file`+`env` providers loaded in defaults→file→env→flags order; fsnotify scoped to the input file's (and any loaded theme file's) parent directory by default, never a blind whole-repo recursive walk; a stdlib-only Server-Sent-Events reload channel (no extra websocket dependency needed since the channel is strictly server→browser); `github.com/pkg/browser` for CLI-04. Resolve the `press.Options` theme-CSS-injection gap as an explicit Wave-0 task before any `--theme-set` task is planned.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `github.com/spf13/cobra` | v1.10.2 (Apache-2.0) | Command tree, POSIX flags via bundled `pflag` | Already vetted HOLD in `.planning/research/STACK.md`; structural match for Marp CLI's own subcommand surface; built-in shell-completion + man-page generation. |
| `github.com/fsnotify/fsnotify` | v1.10.1 (BSD-3-Clause, needs Go 1.23+) | Cross-platform filesystem events | Already vetted HOLD in STACK.md; no viable pure-Go alternative with comparable platform coverage (inotify/kqueue/ReadDirectoryChangesW/FEN). |
| `github.com/knadh/koanf/v2` | v2.3.5 (MIT) | Config loading/merging (YAML/JSON/TOML/env/flags) | Already vetted HOLD in STACK.md; case-preserving (unlike viper), pay-for-what-you-use parser/provider modules. |
| `github.com/knadh/koanf/parsers/{yaml,json,toml}` | matching v2 line | Format-specific parsers, installed as separate modules | koanf's core has zero built-in format assumption by design — this project needs all three for `.marprc.{yml,json,toml}`. |
| `github.com/knadh/koanf/providers/{file,posflag,env}` | matching v2 line | Load sources: file / cobra-compatible pflag / environment | `posflag.Provider(f *pflag.FlagSet, ".", k)` consumes `cmd.Flags()` directly — cobra's `*pflag.FlagSet` is exactly what `posflag` expects, no adapter needed. |
| `github.com/pkg/browser` | latest tagged (BSD-2-Clause-style; last commit 2024-01-02) | Cross-platform "open in default browser" | Verified by reading its actual source (see Sources): `open` on darwin, tries `xdg-open`/`x-www-browser`/`www-browser` in order on linux, `windows.ShellExecute` (via `golang.org/x/sys/windows`, Windows-only build tag) on windows. Zero non-stdlib dependency on darwin/linux; the one extra dependency (`x/sys`) is Windows-only and part of the extended stdlib ecosystem. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| stdlib `net/http` + `http.Flusher` (or `http.NewResponseController`) | stdlib | Server-Sent-Events reload channel | Recommended **instead of** a websocket library for CLI-02/CLI-03's live-reload signal — see Architecture Pattern 4. Zero added Go dependency. |
| stdlib `path/filepath` (`WalkDir`) | stdlib | Directory-tree enumeration for watch-scope setup | Pairs with fsnotify's `Add`/`AddWith` — fsnotify itself does not recurse (confirmed, see Pitfalls). |
| stdlib `io/fs`, `net/http` (`http.FileServer`/`http.ServeMux`) | stdlib | Static file serving in `serve` mode | No third-party static-file server needed; Go's stdlib is sufficient and this keeps the CLI's own dependency count low. |

### Alternatives Considered

| Instead of | Could use | Tradeoff |
|---|---|---|
| SSE (stdlib) for reload | `github.com/coder/websocket` (formerly `nhooyr.io/websocket`, actively maintained; `gorilla/websocket` is archived since late 2022 — do not adopt it new) | Websocket is bidirectional and is what Marp CLI itself uses, but the reload channel is strictly unidirectional (server→browser "reload" signal) — SSE's built-in `EventSource` auto-reconnect and zero-Go-dependency profile is a strictly better fit for this one need. Reconsider only if a future feature needs the browser to talk back to the CLI. |
| `github.com/pkg/browser` for CLI-04 | `github.com/toqueteos/webbrowser` (its own docs point users to `pkg/browser` as the more complete option), `github.com/skratchdot/open-golang` | Both are viable narrower alternatives; `pkg/browser`'s source is short enough to read in full during review (verified above) and has no surprising behavior. |
| Hand-rolled cobra "catch-all root + subcommands" | Force every invocation through an explicit `convert` subcommand (no bare-positional default) | Marp CLI's own UX (`marp deck.md`) and Hugo's (`hugo` alone builds) both default to a bare-positional invocation; matching that convention is lower-friction for users coming from either tool, at the cost of the documented cobra positional-arg/subcommand-name collision (Pitfall 1). |

**Installation** (representative, exact versions to be pinned via `go get` at task time):
`go get github.com/spf13/cobra@v1.10.2 github.com/fsnotify/fsnotify@v1.10.1 github.com/knadh/koanf/v2@v2.3.5 github.com/knadh/koanf/parsers/yaml github.com/knadh/koanf/parsers/json github.com/knadh/koanf/parsers/toml github.com/knadh/koanf/providers/file github.com/knadh/koanf/providers/posflag github.com/knadh/koanf/providers/env github.com/pkg/browser`

## Architecture Patterns

### Recommended Project Structure

```
cmd/eden-press/
├── main.go              # entrypoint: build root cmd, Execute()
├── root.go              # root command: shared/persistent flags, default (convert) action
├── convert.go            # explicit `convert` subcommand (same logic root falls through to)
├── watch.go              # `watch` subcommand
├── serve.go              # `serve` subcommand
├── preview.go            # `preview` subcommand
├── config.go             # koanf setup: defaults -> file -> env -> flags, format-by-extension
├── htmldoc.go            # bare-style standalone HTML document assembly from press.Output
├── reload/               # SSE notify server + tiny injected client script (viewer-side only)
│   ├── server.go
│   └── client.js         # ~10-line EventSource reload snippet, go:embed'd
└── input.go              # stdin ("-") vs file-path input resolution
```

This mirrors Objective 3's own package-per-concern discipline and matches ARCHITECTURE.md's existing top-level box for `cmd/eden-press` ("CLI argument parsing, watch/serve/preview").

### Pattern 1: cobra root-as-default + explicit subcommands

**What:** The root `cobra.Command` both (a) defines persistent/local flags shared by every mode and (b) has its own `RunE` that performs a **convert** when invoked with zero or one positional argument and no subcommand — `eden-press deck.md` "just works," matching Marp CLI's (`marp deck.md`) and Hugo's (`hugo`) own top-level ergonomics. `watch`, `serve`, and `preview` are registered as ordinary subcommands via `AddCommand` in `init()`/constructor functions. A `convert` subcommand is **also** registered explicitly (not just reachable via the root default) so a filename can never be mistaken for a missing/unknown subcommand and so shell completion has a stable name to offer.

**When to use:** Always, for this CLI shape — it is the documented, community-verified cobra idiom for "default action + real subcommands" (see Sources; confirmed via the cobra issue tracker's "catch-all default action" thread and via Hugo's real-world precedent).

**Key gotcha (see Pitfall 1):** cobra resolves an exact-match subcommand name **before** treating an argument as positional data. A markdown file literally named `watch`, `serve`, `preview`, or `convert` (with no extension, or whose whole first arg exactly matches a subcommand name) would be swallowed as a subcommand invocation, not a filename. Mitigate by documenting `eden-press convert -- watch.md` (`--` end-of-flags marker) or by requiring a recognizable extension.

**Flag surface → `press.Options` mapping:**

| CLI flag | Scope | Maps to | Note |
|---|---|---|---|
| `--theme <name>` | persistent | `Options.Theme` | `""` (unset) already resolves front-matter `theme:` → `"default"` inside `press.Render` itself — the CLI must NOT re-implement that fallback, just pass the flag value through verbatim (empty string when unset). |
| `--theme-set <path...>` | persistent | **no current `Options` field** | See Open Question 1 — blocking finding, needs a `press/` API decision first. |
| `--profile <name>` | persistent | `Options.Profile` | `""` resolves `profile.Default()` ("slides") already; pass through. |
| `--math <mathml\|off>` | persistent | `Options.MathMode` | `""` and `"mathml"` are equivalent per `pmath.Option`; `"off"` disables. |
| `--no-highlight` | persistent | `Options.NoHighlight` | Boolean, direct passthrough. |
| `--highlight-style <name>` | persistent | `Options.HighlightStyle` | `""` resolves chroma's pre-verified default style. |
| `--inline-svg` | persistent | `Options.InlineSVG` | **Advisory-only finding:** per `press/press.go`'s own doc comment, `packThemeCSS` ORs `opts.InlineSVG` with `svgEnabled(pc)`, and `pc`'s inline-SVG flag is set **unconditionally true** by the seam today ("the seam's ParseWithEngine unconditionally enables inline-SVG mode"). In effect, `Options.InlineSVG` is currently a no-op — the packed CSS is always inline-SVG-shaped. Expose the flag for forward-compatibility/documentation, but do not build CLI logic that assumes it can turn inline-SVG off yet. |
| (none — Go value type, not flag-representable) | — | `Options.Sanitize` | Deliberately **never** exposed at the CLI layer in v1: it's a `*bluemonday.Policy` Go object with no serializable CLI/config representation. The CLI always leaves it `nil` (built-in always-on policy) — this is a considered omission, not an oversight. |
| `--output, -o <path>` | per-subcommand | *(not a press.Option — CLI-only concern)* | Where the assembled standalone HTML document is written; default stdout for `convert`, default `<input-stem>.html` for `watch`. |
| `--config <path>` | persistent | *(koanf source, not a press.Option)* | Explicit override of the auto-discovered `.marprc.*` search. |

### Pattern 2: bare-style standalone HTML assembly from `press.Output`

**What:** `press.Render` returns `Output.HTML` (already `<div class="marpit">` + one `<svg><foreignObject><section>…</section></foreignObject></svg>` per slide — confirmed by reading `chase/markdown/render.go` and `profiles/slides/slides.go`'s `"div.marpit"` unit-container string) and `Output.CSS` (packed theme CSS). The CLI's `htmldoc.go` wraps these into a **complete, standalone HTML document**, directly mirroring Marp CLI's own `layout.pug`/`bare.pug` composition (fetched and read verbatim from `marp-team/marp-cli` for this research — see Sources): `<!doctype html>` → `<head>` with charset/viewport meta + `<style>{Output.CSS}</style>` → `<body>{Output.HTML}</body>`. **No script tag by default** — matching `bare.pug`'s `block script` being empty.

**Confirmed, not assumed:** `press/press_test.go` and `press/capstone_test.go` both assert `!strings.Contains(out.HTML, "<script")` — the bluemonday sanitize pass (CORE-05) **always** strips `<script>` from `Output.HTML` before it's ever returned. This means:
- Any live-reload script (Pattern 4) or the auto-fit `themes.BrowserFitJS()` helper **must be appended by the CLI's `htmldoc.go` layer, outside and after `Output.HTML`** — never fed back through `press.Render`/sanitize, and never assumed to survive if accidentally concatenated *before* sanitization runs.
- This is the concrete mechanism by which "zero-JS backend" and "an optional viewer-side reload/auto-fit helper" coexist without contradiction: the backend (`press/`) never emits or depends on JS; the CLI's document-assembly step is where an *additive, clearly-labeled* script is spliced in, only for `watch`/`serve` (reload) or if auto-fit support is explicitly wanted (see Open Question 2).

**Theme/theme-set loading:** `press/themes.Names()` (`["default","gaia","uncover"]`) is the enumerable set `--theme` may name today. `press/themes.BrowserFitJS()` returns the vendored Marp Core auto-fit helper script verbatim (with its original 2018 MIT header) — this is the exact asset to splice in per the paragraph above, if/when auto-fit is wired at the CLI layer.

### Pattern 3: fsnotify — scoped, non-recursive-by-default watch

**What:** fsnotify does **not** recurse (confirmed via CHANGELOG + README: "a recursive watcher is planned but not yet available," tracked as issue #18 for years) and does **not** survive an atomic-save rename on a directly-watched file (confirmed via README: editors write-to-temp-then-rename, "the watcher on the original file is now lost"). The correct, documented pattern (fsnotify's own `cmd/fsnotify/file.go` example, confirmed by direct read) is: **watch the parent directory**, then filter `Event.Name` down to the file(s) of interest.

**Scope recommendation:** default `eden-press watch <in.md>` to watching only (a) the input file's parent directory and (b) any explicitly-loaded custom theme file's parent directory — mirroring Marp CLI's own granularity (its `chokidar`-based watcher tracks a resolved *file list*, not an arbitrary project-wide recursive scan). Reserve a full `filepath.WalkDir`-based recursive watch (walk once at startup to `Add` every directory, then `Add` any newly-`Create`d directory encountered during the run) for a later, explicitly-scoped "batch/directory" mode — not the v1 single-deck default. This avoids the real inotify `fs.inotify.max_user_watches` exhaustion risk of watching an entire repo (a documented, concrete failure mode on large trees).

**Debounce:** editors reliably fire multiple raw events per logical save. The standard idiom is a `time.Timer`/`time.AfterFunc` reset on every incoming event, firing the rebuild only once the stream goes quiet — 300–500ms is the commonly cited interactive-tooling range (verified against multiple independent examples, see Sources); recommend 300ms as the starting default (rebuild + reload should feel instant, and eden-press's own render is pure-Go/no-Chrome so a rebuild is cheap).

**Filtering:** ignore `fsnotify.Chmod` (fires spuriously; Linux inotify in particular emits it ahead of a delayed `Remove`), ignore editor backup/swap files (`~`-suffixed, `.swp`), and ignore directory-level `Write` events specifically (their meaning differs across kqueue/Windows — "contents changed" — vs. Linux inotify — "a *file's* content changed" — so treat directory `Write` as a signal to re-scan matched filenames, not as a content-change event on its own).

### Pattern 4: live-reload without a JS framework — SSE, not WebSocket

**What:** Confirmed by reading Marp CLI's actual source (`src/templates/watch/watch.ts`, `src/watcher.ts`): its mechanism is a small `WebSocket` client (`ws.addEventListener('message', e => if (e.data === 'reload') location.reload())`) talking to a `WebSocketServer` the CLI itself starts (on an ephemeral port via `portfinder`, defaulting to search from 37717), with the client script only injected into the template (`watchJs`) when a watch/serve session is active.

**Recommendation for Eden Press:** replicate the same shape but with **Server-Sent Events** instead of WebSocket, since the channel is one-directional (server tells the browser "reload," full stop): a tiny stdlib `net/http` handler on a loopback-only port sets `Content-Type: text/event-stream`, holds the connection open, and on a filesystem-change signal writes `event: reload\ndata: reload\n\n` + `Flusher.Flush()` (or `http.NewResponseController(w).Flush()` on newer Go). The browser side is the built-in `EventSource` API (~5 lines: `new EventSource(url).addEventListener('reload', () => location.reload())`) — no bundled client library needed, and `EventSource` auto-reconnects on its own, which is exactly the reconnect-loop Marp CLI had to hand-write for its WebSocket client. This keeps the CLI's own dependency count at zero for the reload channel (stdlib only) versus adding a websocket library.

**This applies to both `watch` (no file server — inject the SSE client + endpoint even though the deck itself may be opened via `file://`, exactly as Marp CLI spins up its notifier even in "static" watch mode) and `serve` (the same reload channel, served from the same HTTP server, alongside static files)** — do not build two different reload mechanisms.

### Pattern 5: serve mode — static files + convert-on-request + traversal guard

**What:** confirmed by reading Marp CLI's `src/server.ts` in full: serve mode is (1) an `http.FileServer`-equivalent rooted at a directory, (2) an interception layer that recognizes markdown-extension requests, validates the resolved path stays inside the root directory (`path.resolve` + prefix check — defense against `../../etc/passwd`-style traversal even after URL-decoding), and (3) converts that one file via the render function **on every request** (no server-side caching layer in v1 — matches Marp CLI's own behavior) before responding with the assembled HTML document (Pattern 2) plus the SSE reload script (Pattern 4).

**Scope correction vs. Marp CLI:** Marp CLI's server also switches output format via query string (`?pdf`, `?png`, `?pptx`) because its `Converter` can already drive `chromedp`/soffice-equivalent exporters. **Eden Press's Objective-4 CLI must not do this in v1** — PDF/PNG/PPTX export is `convert/` package territory (EXP-01..04, Objectives 5/6), and the CLI's own Obj-3 boundary is "import `press/` only." CLI-03's own requirement wording ("serve local files, convert on request") is satisfied by HTML-only conversion; query-string format-switching is a natural post-Objective-5/6 extension point, not an Objective-4 deliverable — call this out explicitly in the plan so it isn't accidentally scope-crept in.

**Directory traversal guard (must-have, not optional):** resolve the server's root to an absolute path once at startup; for every request, `filepath.Join` + decode the requested path against that root, then verify (via `filepath.Rel` or a prefix check on the cleaned absolute path) that the result still lives under the root before opening/converting it. Marp CLI keeps this check even though its underlying framework (Express) already does some of this — treat it as defense-in-depth, not redundant.

### Pattern 6: koanf — multi-format config + explicit precedence

**What:** koanf imposes no built-in precedence — "whatever you load last wins" (confirmed via koanf's own README). The recommended, explicit chain for this CLI is: (1) `structs.Provider` for compiled-in defaults, (2) `file.Provider(path)` + the format-appropriate parser (chosen by a small extension→parser switch — koanf deliberately has **no** built-in extension-sniffing, this is called out in its own docs as a viper flaw it avoids by being explicit) for the discovered `.marprc.{yml,yaml,json,toml}`, (3) `env.Provider("EDEN_PRESS_", ".", …)` for environment overrides, (4) `posflag.Provider(cmd.Flags(), ".", k)` for CLI flags **last** (highest precedence) — critically, `posflag.Provider` must be given the koanf instance `k` itself as its third argument so it can tell "flag left at its default" apart from "flag explicitly set," and not let unset-flag defaults silently stomp values already loaded from file/env (confirmed via koanf's README + a documented GitHub issue about this exact caveat with the plain-`flag`-package `basicflag.Provider`, which lacks this safeguard — use `posflag`, not `basicflag`, specifically because cobra's flags are `pflag`-based already).

**Config file discovery:** no cosmiconfig-equivalent library is warranted for three fixed filenames — a short, explicit search (cwd `.marprc.yml`/`.marprc.yaml`/`.marprc.json`/`.marprc.toml`, in that order, first match wins, `--config` flag overrides the search entirely) is simpler and auditable. Document whether an XDG-style global config path is also wanted (Open Question 5) — Marp CLI itself only searches project-local locations, no global config.

**Stdin (`-`) input:** orthogonal to koanf entirely — this is the *markdown input source*, not config. When the positional input argument is exactly `"-"`, read all of stdin into the string passed to `press.Render`. This is incompatible with `watch` (there is no file to watch) — the CLI must explicitly reject or no-op `--watch` when input is `-` (Pitfall 9), and it changes the default output destination assumption for `convert` (stdin → stdout is the natural default pairing, matching Unix pipeline conventions and Marp CLI's own stdin/stdout behavior).

### Anti-Patterns to Avoid

- **Re-implementing recursive directory watching as a "smart" single fsnotify call.** It does not exist upstream (issue #18, years open) — don't spend implementation time trying to find a hidden recursive mode; budget the explicit walk+dynamic-`Add`-on-`Create` pattern from the start.
- **Feeding the live-reload or auto-fit `<script>` back through `press.Render`.** It will be stripped by the sanitizer (confirmed by the codebase's own tests) — assemble it only in the CLI's own `htmldoc.go`, after `Output.HTML` is already final.
- **Treating `Options.InlineSVG` as a functioning on/off toggle.** It's effectively always-true today per `packThemeCSS`'s own doc comment — don't build a `--no-inline-svg` feature that silently does nothing.
- **Query-string format-switching in `serve` mode for v1.** That's `convert/`'s job (Objective 5/6), not Objective 4's.
- **Defaulting the `serve` port to 8080.** Independent of this project's own dev-environment constraint (this repository's local verification must never bind/curl/reference `:8080` — use `8091` for any local verification of `serve` mode during this objective's own manual testing), 8080 is also simply an extremely common collision target for local dev tooling in general; pick a distinct default (e.g. an ephemeral/ freely-choosable port via a `--port` flag with a documented non-8080 default) and make it fully overridable.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Opening a URL/file in the OS default browser | Platform-specific `exec.Command("open"/"xdg-open"/"rundll32", …)` shell-out logic | `github.com/pkg/browser` | Its entire implementation is ~50 lines across 3 platform files (verified by direct read) — trivially small, but the linux fallback chain (`xdg-open`→`x-www-browser`→`www-browser`) and the Windows `ShellExecute` call are exactly the kind of "looks simple, has three OS-specific edge cases" surface not worth re-discovering. |
| YAML/JSON/TOML parsing | A hand-rolled format-sniffing parser | koanf's `parsers/{yaml,json,toml}` + `providers/file` | This is precisely koanf's reason for existing; don't write config parsing when the chosen library already owns it. |
| Reload-channel reconnect-on-drop logic | A hand-written retry/backoff loop for a websocket (or SSE) client | Browser-native `EventSource` (SSE) reconnect | `EventSource` reconnects automatically per spec — Marp CLI had to hand-write this exact retry loop *because* it chose WebSocket; SSE gets it for free. |
| HTML sanitization for the assembled document | A second sanitize pass on the final HTML document (theme CSS + `Output.HTML` + injected script) | Nothing — `press.Render` already sanitizes `Output.HTML`; the CLI's own injected `<script>`/`<style>` additions are CLI-authored, not user content, and must not be run back through any sanitizer (which would strip the `<script>` tag entirely, per the pitfall above). | Don't build a defense mechanism against your own trusted, static template code. |

**Key insight:** every "don't hand-roll" item in this objective is small in isolation (opening a browser, parsing a config file, retrying a dropped connection) — the risk is not effort, it's the accumulation of platform-specific/protocol-specific edge cases that existing, narrowly-scoped libraries (or browser-native APIs) already own correctly.

## Common Pitfalls

### Pitfall 1: cobra positional-arg vs. subcommand-name collision
**What goes wrong:** `eden-press watch` is ambiguous if a file literally named `watch` (no extension, or exactly matching a registered subcommand name) is the intended input.
**Why it happens:** cobra resolves an exact subcommand-name match before treating the first argument as positional data — this is documented cobra behavior, not a bug.
**How to avoid:** document `eden-press convert -- <file>` (`--` end-of-flags) as the disambiguation path; in practice this only bites files with no extension matching a subcommand name exactly, a narrow edge case worth a one-line doc note, not a redesign.
**Warning signs:** a bug report where a specific filename "doesn't work" but renaming it does.

### Pitfall 2: fsnotify atomic-save breaks single-file watches
**What goes wrong:** watching the markdown file's own path directly loses the watch the moment an editor does its usual write-temp-then-rename save.
**Why it happens:** the rename replaces the inode fsnotify was watching; the original watch has nothing left to watch.
**How to avoid:** always watch the **parent directory**, filter `Event.Name` (confirmed pattern from fsnotify's own example code).
**Warning signs:** watch mode works for the first edit, then silently stops rebuilding after a save in a specific editor (classically Vim).

### Pitfall 3: sanitizer strips `<script>` — auto-fit markers are inert without deliberate script injection
**What goes wrong:** `# <!--fit-->` headings and shrink-wrapped code/math blocks get their `data-auto-scaling="fit"`/`.marp-fit-shrink` markers emitted correctly (CORE-09), but nothing ever actually resizes them, because the viewer-side script that reads those markers (`themes.BrowserFitJS()`) is never included.
**Why it happens:** `press.Render`'s sanitize pass (CORE-05) strips any `<script>` unconditionally, and nothing in `press/` ever re-adds one — by design, since `press/` must stay JS-free.
**How to avoid:** this is a CLI-layer, document-assembly-time decision (Pattern 2/Open Question 2), not a `press/` bug — the CLI's `htmldoc.go` must explicitly choose whether/when to splice `themes.BrowserFitJS()` in, after sanitize has already run.
**Warning signs:** a deck with a `<!--fit-->` heading renders with the marker attribute present in the DOM (inspectable) but visually unscaled.

### Pitfall 4: `--theme-set` has no `press.Options` hook today (the blocking finding)
**What goes wrong:** planning CLI-05 tasks as CLI-only work will stall the moment implementation tries to pass a caller-supplied theme CSS string into `press.Render` — there is no field for it.
**Why it happens:** Objective 3 froze `press.Options` with exactly the fields a Marp-Core-parity render needed; custom/user themes were never a named consumer at that time.
**How to avoid:** treat this as an explicit, sequenced Wave-0 decision (see Sequencing & Risk below) — either add a small additive `Options` field (e.g., a slice/map of raw theme-CSS text, threaded into `packThemeCSS`'s `ThemeSet` build exactly the way the 3 embedded themes are added today) or document an accepted alternative. Do not let a task silently attempt to import `chase/theme` from the CLI to work around it — that violates the Objective-3 CLI-boundary gate.
**Warning signs:** a task's TRD references `chase/theme.Load` or `chase/theme.ThemeSet` directly from `cmd/eden-press` — an immediate boundary-violation signal.

### Pitfall 5: `posflag` needs the koanf instance to avoid default-value stomping
**What goes wrong:** using `basicflag.Provider` (stdlib `flag`) or calling `posflag.Provider` without passing `k` causes every flag's zero/default value to silently overwrite values already loaded from file/env, even when the user never touched that flag.
**Why it happens:** without the koanf instance, the provider can't distinguish "flag left at default" from "flag explicitly set to this value."
**How to avoid:** always use `posflag.Provider(cmd.Flags(), ".", k)` (three-argument form, koanf instance included) as the last-loaded source.
**Warning signs:** a config file's theme setting is mysteriously ignored the moment any flag (even an unrelated one) is present on the command line.

### Pitfall 6: unbounded recursive watch scope on large trees
**What goes wrong:** a naive "just walk everything under cwd and `Add` every directory" watch implementation can exhaust `fs.inotify.max_user_watches` on Linux, or blow through per-process file-descriptor limits on kqueue platforms, when pointed at a large monorepo.
**Why it happens:** fsnotify allocates a real OS-level watch (and, on some backends, an open file descriptor) per watched directory; there is no built-in ceiling or backpressure.
**How to avoid:** default to the narrow, per-file/per-theme-file scope (Pattern 3); only widen to a full recursive walk for an explicitly-opted-into "directory/batch" mode, and consider surfacing a clear error (not a silent OS-level failure) if the watch count would be excessive.
**Warning signs:** `too many open files` or `no space left on device` errors that have nothing to do with actual disk space.

### Pitfall 7: directory-traversal in serve mode
**What goes wrong:** a crafted request path (`../../../../etc/passwd`, URL-encoded variants) escapes the intended served directory.
**Why it happens:** naive path concatenation without resolving to an absolute path and checking containment.
**How to avoid:** resolve the server root to an absolute path once; for every request, decode + join + `filepath.Clean`, then verify the result is still prefixed by the root before any file I/O (Marp CLI keeps exactly this check as defense-in-depth even on top of its underlying framework's own protections).
**Warning signs:** a security scan / fuzzer flags path traversal; or, more mundanely, a request for a sibling directory's contents unexpectedly succeeds.

### Pitfall 8: stdin input combined with watch mode
**What goes wrong:** `cat deck.md | eden-press watch -` has no file for fsnotify to watch — a naive implementation would either error confusingly deep in the watch setup, or (worse) silently hang.
**Why it happens:** watch mode is fundamentally file-path-based; stdin is a one-shot stream with no path to re-read.
**How to avoid:** explicitly detect `input == "-"` at the CLI's argument-validation layer and reject `--watch`/the `watch` subcommand with a clear, early error message — don't let this surface from inside the fsnotify setup code.
**Warning signs:** a hang or an opaque fsnotify error when a user pipes into `watch`.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| `viper` for CLI config | `koanf` | Already resolved in `.planning/research/STACK.md` (case-preservation, dependency weight) | Carried forward unchanged into this objective — no new decision needed here. |
| `gorilla/websocket` for Go websocket needs | `github.com/coder/websocket` (formerly `nhooyr.io/websocket`, handed off to Coder Aug 2024) | gorilla archived late 2022; coder/websocket is the actively-maintained successor | Not directly adopted here (SSE is recommended instead, see Pattern 4), but relevant if a future feature needs bidirectional browser↔CLI communication. |
| Hand-rolled recursive-watch scripts (`howeyc/fsnotify`-era community patterns) | `filepath.WalkDir` at startup + dynamic `Add` on `Create` events, using the current `fsnotify/fsnotify` module's `Add`/`AddWith` | Standard idiom for years now, unchanged in the 1.9.0→1.10.1 changelog window checked for this research | Confirms there is still no built-in recursive mode as of the pinned version — plan around it, don't wait for it. |

**Deprecated/outdated:**
- `nhooyr.io/websocket` import path: functionally fine (no breaking API change) but the canonical home is now `github.com/coder/websocket` — moot here since SSE is recommended over either.
- `gorilla/websocket`: archived, do not adopt for new work.

## Sequencing & Risk (CLI TRD waves)

Mirroring Objective 3's wave discipline (`.planning/ROADMAP.md`'s 03-01..03-09 pattern):

**Wave 0 (blocking, must land before any `--theme-set` task is written):**
- Resolve the `press.Options` custom-theme-CSS gap (Pitfall 4 / Open Question 1) — a small, additive `press/options.go` decision + (if approved) a tiny `press/` TRD, coordinated as an explicit cross-objective dependency even though Objective 3 is marked complete/merged.

**Wave 1 (parallel, no interdependencies):**
- Command skeleton: cobra root + `convert`/`watch`/`serve`/`preview` subcommand stubs, persistent flag surface, flag→`Options` wiring (Pattern 1).
- koanf config loading: defaults→file→env→flags chain, `.marprc.*` discovery, stdin (`-`) input resolution (Pattern 6).
- `htmldoc.go`: bare-style standalone HTML assembly from `press.Output` (Pattern 2), with the script-injection point deliberately left as a seam (not wired yet) for Wave 2.

**Wave 2 (depends on Wave 1's command skeleton + htmldoc seam):**
- `convert` end-to-end (the CLI-01 capstone for the default path).
- `watch`: fsnotify scoped-watch design + debounce (Pattern 3) + the SSE reload server (Pattern 4), wired through the Wave-1 `htmldoc.go` script-injection seam.
- `serve`: static-file serving + convert-on-request + traversal guard (Pattern 5), reusing the exact same SSE reload plumbing `watch` built — not a second implementation.

**Wave 3 (capstone/integration):**
- `preview`: `pkg/browser` wired to both the `convert` output path and the `serve` URL.
- Integration test: exercise convert+watch+serve+preview+`--theme`+config-file+stdin in one pass; add a `go list -deps ./cmd/eden-press/... | grep -E 'chase|profiles'` CI gate mirroring API-02's `chromedp` gate, to keep the Objective-3 "CLI imports only `press/`" boundary enforced mechanically, not just by review discipline.

**The 3 riskiest items, ranked:**
1. **The `press.Options` theme-CSS gap (Pitfall 4).** This is a cross-package coordination risk, not a pure-CLI risk — it requires touching a package Objective 3 already shipped and marked complete. Getting the shape of that addition right (and getting buy-in that it's in-scope for Objective 4 to make a small additive change there) is the single highest-leverage decision in this whole objective.
2. **fsnotify cross-platform event-semantics correctness (Pitfall 2, 6).** inotify/kqueue/ReadDirectoryChangesW genuinely differ (delayed `Remove`, directory-`Write` meaning, no `Chmod` on Windows) — "passes on my dev machine" is not evidence it works everywhere; this needs real per-OS verification (CI matrix or at minimum documented manual verification on macOS+Linux+Windows), not just unit tests against a single backend.
3. **The zero-JS-vs-auto-fit tension at the HTML-assembly boundary (Pitfall 3/Open Question 2).** This is a genuine product decision with visible UX consequences (a heading marked `<!--fit-->` either silently doesn't scale, or the "zero-JS" claim gets an asterisk) — it must be decided explicitly in planning, with the decision documented in the CLI's own README/help text, not left to whichever engineer happens to implement `htmldoc.go` first.

## Open Questions

1. **What is the exact `press.Options` shape for custom/`--theme-set` themes?**
   - What we know: the mechanism that would consume it already exists inside `press/` (`chase/theme.Load(cssText, unit, sizeFallback)` + `ThemeSet.Add`, exactly how the 3 embedded themes are registered today, per `press/themes/themes.go`).
   - What's unclear: whether the new field should be raw CSS text (slice/map), file paths (requiring `press/` to do its own file I/O — likely undesirable, breaks the "pure function over strings" shape of `Render`), or a pre-built object of some exported `press/`-level type.
   - Recommendation: raw CSS text (caller reads the files, `press/` never touches the filesystem) is the shape most consistent with `Render`'s existing "pure function of `(md, opts)`" contract. Resolve as a Wave-0 task with an explicit small TRD.

2. **Should the auto-fit viewer script be injected by default in `convert` output, or opt-in only?**
   - What we know: without it, `<!--fit-->` markers are inert; with it, the "zero-JS" default output claim needs a footnote (viewer-side only, not backend, but still a script tag in the shipped HTML).
   - What's unclear: which framing the project wants to lead with in its own marketing/README (this echoes the exact `bare` vs `bespoke` tension Marp CLI itself encodes as a template choice, not an accident).
   - Recommendation: default `convert`/`watch`/`serve` output to **not** inject it (true zero-script bare HTML, matching CLI-01's literal wording "zero-JS static HTML"), and add an explicit opt-in flag (e.g. `--auto-fit-script`) for users who want the marker-driven scaling to actually function. Document the tradeoff plainly rather than silently picking one.

3. **Exact debounce interval and watch-scope default.**
   - What we know: 300–500ms is the commonly-cited interactive-tooling range; narrow (input-file-dir-only) scope is recommended over broad recursive scope for v1.
   - What's unclear: whether real usage (large decks with many linked image assets in subdirectories) will demand recursive scope sooner than expected.
   - Recommendation: ship the narrow default, but design the internal watch-scope resolver so widening to recursive is a config change, not a rewrite.

4. **Default `serve` port.**
   - What we know: it must not be 8080 (both as a general good-practice default and, specifically, as a hard rule for this project's own local dev/verification environment, which permanently reserves `:8080` for another app and designates `8091` for local verification needs).
   - What's unclear: the actual shipped product default (this is a product/UX choice for the eden-press maintainers, not an environment constraint) — pick something distinct and clearly document it as overridable via `--port`/env.
   - Recommendation: pick any non-8080 default (e.g., an explicit fixed value in a safe ephemeral-adjacent range) and make it trivially overridable; separately, ensure any manual/local verification performed *during this objective's own development* binds to `8091`, never `8080`, per the standing environment rule.

5. **Global (XDG-style) config search path, in addition to project-local `.marprc.*`?**
   - What we know: Marp CLI itself only searches project-local locations, no global config path.
   - What's unclear: whether Eden Press wants to differ here (e.g., a `~/.config/eden-press/config.{yml,json,toml}` fallback for shared team defaults).
   - Recommendation: start with project-local-only (matches upstream, minimal surface); treat a global config path as a clearly-scoped v1.x addition if requested later, not a silent v1 inclusion.

## Sources

### Primary (HIGH confidence)
- `github.com/marp-team/marp-cli` — fetched directly via `gh api repos/marp-team/marp-cli/contents/...` (not a cache/summary): `src/templates/layout.pug`, `src/templates/bare/bare.pug`, `src/templates/bare/bare.scss`, `src/templates/index.ts`, `src/templates/watch/watch.ts`, `src/watcher.ts` (`Watcher`/`WatchNotifier` classes), `src/server.ts` (`Server` class), `src/preview.ts`. Confirms: `bare` template's empty `block script`; the `watchJs`-injected-only-when-active mechanism; the WebSocket-based reload protocol; the query-string format-selection and directory-traversal-guard patterns in serve mode.
- `github.com/pkg/browser` — fetched `browser.go`, `browser_darwin.go`, `browser_linux.go`, `browser_windows.go`, `LICENSE`, `go.mod` directly; confirmed last-commit date via GitHub API (`2024-01-02`).
- `github.com/fsnotify/fsnotify` — official README + CHANGELOG.md (fetched raw), confirming platform-support table, Go 1.23+ requirement, non-recursive limitation (issue #18), and the parent-directory-watch atomic-save pattern; `cmd/fsnotify/file.go` example (fetched) confirming the filter-by-`Event.Name` idiom.
- `github.com/knadh/koanf` — official README (fetched raw), confirming provider/parser list, load-order-is-precedence design, and the `posflag`-needs-koanf-instance caveat.
- Direct repo inspection (this codebase): `press/press.go`, `press/options.go`, `press/themes/themes.go`, `press/autofit.go`, `press/doc.go`, `press/sanitize/*.go` (grep-confirmed), `press/press_test.go`/`capstone_test.go` (confirmed the `!strings.Contains(out.HTML, "<script")` assertion), `chase/markdown/render.go` (`<div class="marpit">` container), `profiles/slides/slides.go` (`"div.marpit"` unit string), `chase/profile/profile.go` (`Profile` interface shape), `go.mod` (module path, `go 1.25.0`, no cobra/fsnotify/koanf dependency present yet — confirms greenfield).
- `.planning/research/STACK.md`, `FEATURES.md`, `ARCHITECTURE.md`, `PITFALLS.md` (this project's own prior research pass) — cobra/fsnotify/koanf version/verdict table, the `bare`-vs-`bespoke` zero-JS distinction, the ARCHITECTURE.md box for `cmd/eden-press`.

### Secondary (MEDIUM confidence)
- WebSearch on cobra's "catch-all default root command + subcommands" idiom, cross-checked against the cobra GitHub issue tracker's own quoted code (`requireSubcommand`/catch-all `Run` patterns) and Hugo's well-known real-world precedent (not independently re-verified against Hugo's own source in this pass).
- WebSearch on Go SSE patterns (`http.Flusher`/`http.NewResponseController`, `EventSource` client) — pattern is standard/textbook Go, cross-checked across multiple independent tutorials, not a single-source claim.
- WebSearch confirming `coder/websocket` as the maintained successor to `nhooyr.io/websocket`, and `gorilla/websocket`'s archived status — corroborated by the library's own blog post + GitHub deprecation notice content quoted in the search result.
- WebSearch on fsnotify debounce interval conventions (300ms/500ms examples from named real projects) — multiple independent examples, no single authoritative "the" interval; treated as a starting recommendation, not a hard requirement.

### Tertiary (LOW confidence)
- None presented as fact in this document — items with residual uncertainty are captured in Open Questions instead of stated as findings.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — cobra/fsnotify/koanf already vetted in prior project research; `pkg/browser` verified by direct source read in this pass; SSE-vs-websocket recommendation is a reasoned design choice built on HIGH-confidence primitives (stdlib `net/http`, browser-native `EventSource`), not itself an external-library claim.
- Architecture (command tree, watch scope, serve/reload, HTML assembly): HIGH for the mechanisms (all confirmed against real upstream source or this codebase's own code/tests) — MEDIUM for the exact scope/interval defaults (Open Questions 3, 4), which are genuinely undecided design parameters, not unresearched facts.
- Pitfalls: HIGH — every pitfall in this document is either confirmed against official docs/changelogs, against this codebase's own source/tests, or against a directly-read upstream implementation (Marp CLI, `pkg/browser`).
- The `--theme-set`/`press.Options` gap: HIGH confidence that the gap exists (verified by reading the full current `press/options.go`) — LOW/undetermined on the exact resolution shape, correctly captured as Open Question 1 rather than asserted.

**Research date:** 2026-07-21
**Valid until:** ~30 days for the stack/architecture guidance (stable Go-ecosystem libraries); the `press.Options` gap finding is valid until Wave 0 resolves it (at which point this document's Pattern 2/Pitfall 4/Open Question 1 should be updated to reflect the actual shipped `Options` shape).
