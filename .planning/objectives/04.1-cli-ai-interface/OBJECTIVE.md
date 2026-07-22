---
work: feature
inserted: true
parent_objective: 4
requirements: []
depends_on: [3, 4, 6]
---

# CLI Agent Interface (AI-usability, Chrome-free)  [INSERTED 4.1]

## Goal
Make `eden-press` a complete, scriptable, agent-facing interface WITHOUT pulling Chrome into
the CLI. Close the Objective-4↔library wiring gap for the Chrome-free surfaces so an AI agent
can inspect and export a deck programmatically, zero browser.

## Scope (locked by user: "AI-core, Chrome-free")
- `preview` stays UNTOUCHED (default OS browser; no Chrome).
- The main `eden-press` binary MUST stay chromedp-free. PNG/PDF are explicitly OUT (deferred to
  a separate export path/binary).

## Deliverables
1. **`--format json`** (default stays `html`): emit the FULL `press.Output` as JSON —
   `{html, css, model:{meta, sections:[{id, attrs, notes, blocks:[{kind,...; code→source+language;
   math→tex+display}]}], outline}, comments, meta}` — so an agent inspects deck structure/content
   programmatically with no browser. Pure `press/`; no Chrome.
2. **`--format pptx`**: wire the CHROME-FREE `convert/pptx.ToPPTX(press.Output.Model, opts)` exporter
   into the CLI (stdlib OOXML, zero chromedp). `-o out.pptx`.
3. **Machine-readable errors**: JSON error envelope on stderr when `--format json` is active +
   stable, documented exit codes so agents can script it.
4. **AGENTS.md** at repo root documenting the agent interface (formats, JSON schema shape, exit
   codes, examples).
5. **Relax `scripts/check-cli-imports.sh`**: permit the CLI to import the chrome-free
   `convert/pptx` (which transitively imports `chase/model`) while STILL forbidding any `chromedp`
   in `cmd/` — i.e. "press/ + chrome-free convert subpackages, never chromedp." Keep
   `check-no-chromedp.sh` green (cmd/eden-press = 0 chromedp).

## Success Criteria
1. `eden-press deck.md --format json` emits valid JSON of the full press.Output incl. per-slide
   Blocks (code source+language, math tex+display), outline, comments — proven by a test that
   parses the JSON and asserts structure.
2. `eden-press deck.md -o out.pptx --format pptx` produces a valid .pptx via convert/pptx with
   NO chromedp in cmd/eden-press's dependency closure.
3. Non-zero, documented exit codes + a JSON error envelope on failure under `--format json`.
4. AGENTS.md documents the interface; all standing gates stay green (gofmt, build/vet/test,
   check-no-chromedp = 0 in press/chase/profiles/bind AND cmd/, check-cli-imports pass, Obj-1
   corpus/cssdiff, Obj-2 grep-gate).

## Non-goals
- No PNG/PDF wiring; no chromedp in cmd/; no change to preview/watch/serve behavior.

---
*Created: 2026-07-22 (/devflow:build — AI-usability wiring, user-scoped)*
