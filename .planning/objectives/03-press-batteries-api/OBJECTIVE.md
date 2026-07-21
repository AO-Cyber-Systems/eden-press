---
work: feature
requirements: [CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, CORE-07, CORE-08, CORE-09, API-01, API-02, API-03]
depends_on: [2]
---

# press/ Batteries + Public API

## Goal

Deliver the Marp-Core-equivalent batteries and the stable, importable `press.Render()` API —
the point at which Eden Press becomes embeddable in any Go service with zero Chrome dependency,
and the gate every downstream consumer (CLI, exporters, Dart) waits on.

## Success Criteria (what must be TRUE)

1. `press.Render(md, opts)` returns `{HTML, CSS, Model, Comments, Meta}` for a deck exercising
   all 3 bundled themes (default/gaia/uncover, embedded verbatim via `go:embed` with preserved
   MIT headers), `size`/`math` global directives, GFM tables + strikethrough (rendered as `<s>`
   to match Marp) + hard-breaks, heading slugs, native shortcode/unicode emoji, chroma-highlighted
   code styled correctly by the bundled themes' `.hljs-*`-shaped CSS, and math (`$…$`/`$$…$$`)
   rendered as MathML with construct-detection routing heavy constructs (`\tag`, `\label`,
   complex `aligned`) to the `codeberg.org/go-latex/latex` SVG/PNG fallback, plus emitted
   auto-fit markers (`<!--fit-->`).
2. `go list -deps ./press/...` contains no `chromedp` — enforced as an automated CI check.
3. HTML sanitization matches Marp's `xss` allow-list *behaviorally* (strip-vs-escape semantics
   documented and deliberately chosen) via an adversarial round-trip test suite, explicitly
   including the GFM disallowed-raw-HTML-tag filter, and the always-on directive/comment-parsing
   code path is validated as its own trust boundary.
4. `Options`/`Output` types are documented and stable enough that a consumer only ever imports
   `press/` — never reaches into `chase/`/`profiles/` directly — to render a complete deck; this
   is the explicit gate at which Objective 7 (Dart binding) may begin.

## Requirements

CORE-01..09, API-01..03 (see .planning/REQUIREMENTS.md)

---
*Created: 2026-07-21 (/devflow:build 3)*
