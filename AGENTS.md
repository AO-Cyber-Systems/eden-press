# AGENTS.md — eden-press agent interface

`eden-press` is a pure-Go CLI: it converts a Marp-compatible Markdown deck to
static output with **no browser, no Node, and no JavaScript runtime**. This
document is the scriptable contract an AI agent (or any non-interactive
caller) needs to inspect and export a deck without reading the Go source.

The whole surface described here is CI-enforced chromedp-free:
`scripts/check-no-chromedp.sh` scans `./cmd/...`'s transitive dependency
closure (alongside `./press/...`, `./chase/...`, `./profiles/...`,
`./bind/...`) and fails the build if `chromedp` ever appears in it.

## Formats (`--format`)

`--format` is a persistent flag on every convert invocation (the bare
`eden-press <in.md>` default, and the explicit `convert` subcommand). It
selects what `emitFormat` emits; `--output`/`-o` (registered on `convert`
and `watch`) selects the destination — a file path, or stdout when unset.

| `--format` | Output | Notes |
|---|---|---|
| `html` (default) | Standalone, zero-JS `<!doctype html>` document | `--auto-fit-script` splices one `<script>` for browser auto-fit; otherwise zero `<script>` tags |
| `json` | The full render `press.Output`, as a lowercase JSON envelope (below) | For programmatic inspection — no browser needed |
| `pptx` | An editable `.pptx` (OOXML zip), built directly from the docmodel | Binary — **always use `-o out.pptx`**, never rely on stdout in a script that also wants to read text |

```bash
# Inspect a deck's structure (defaults to stdout; pipe into jq/etc.):
eden-press deck.md --format json | jq '.model.outline'

# Export an editable PowerPoint deck, zero Chrome:
eden-press deck.md -o slides.pptx --format pptx

# Export via the explicit "convert" subcommand (identical to the bare default):
eden-press convert deck.md --output slides.pptx --format pptx
```

PNG/PDF export is intentionally **not** part of this binary (it requires
Chrome); that lives in a separate export path. `preview`/`watch`/`serve`
behavior is unaffected by `--format` beyond what's documented above.

## JSON envelope schema (`--format json`)

```json
{
  "html": "<section>...</section>",
  "css": "/* packed theme CSS */",
  "model": {
    "meta": { "directives": { "theme": "default", "marp": "true" } },
    "sections": [
      {
        "id": 1,
        "attrs": { "class": "lead" },
        "notes": ["speaker note text"],
        "blocks": [
          { "kind": "heading", "text": "Title", "level": 1 },
          { "kind": "paragraph", "text": "Body text." },
          { "kind": "code", "text": "fmt.Println(1)", "language": "go" },
          { "kind": "math", "text": "E=mc^2", "display": true },
          { "kind": "list", "ordered": false, "items": [
            { "text": "item one", "level": 0 }
          ] }
        ]
      }
    ],
    "outline": [
      { "sectionId": 1, "level": 1, "text": "Title", "slug": "title" }
    ]
  },
  "comments": ["speaker note text"],
  "meta": { "directives": { "theme": "default", "marp": "true" } }
}
```

Field notes an agent must get right:

- **A code block's raw source, and a math block's raw TeX, both live in
  `block.text`** — there are **no** `source` or `tex` keys. `language`
  accompanies a `code` block (e.g. `"go"`, `""` for an unlabeled fence);
  `display` accompanies a `math` block (`true` for `$$...$$`, `false` for
  `$...$`).
- `meta` (top-level) is a convenience alias identical to `model.meta` — read
  either.
- Every field is `omitempty`: absent when unused (e.g. a paragraph block has
  no `language`/`display`/`items`; a section with no speaker notes omits
  `notes` entirely). Do not assume a key is always present — check for it.
- `model` is exactly `chase/model.Document`'s own schema-v2 JSON shape,
  reused verbatim; `cmd/eden-press` itself never imports that package
  directly (it re-marshals the already-typed value through `any`), but the
  wire shape is stable and versioned independently of Marp/goldmark churn.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Render/runtime error (bad input path, render failure, JSON marshal failure, pptx export failure) |
| `2` | Usage/flag error (unknown `--format` value, bad/unregistered flag) |

## Error envelope (`--format json`, on failure)

When `--format json` is active, any failure prints a JSON object to
**stderr** (never stdout) instead of a plain-text message, and the process
exits with the matching code above:

```json
{ "error": { "code": 1, "message": "resolveInput: read file \"deck.md\": open deck.md: no such file or directory" } }
```

For `--format html`/`--format pptx`, a failure prints the same message as
plain text to stderr (no JSON wrapping) with the same exit-code contract.

## Copy-paste examples

```bash
# Pull just the outline (table of contents) as JSON:
eden-press deck.md --format json | jq '.model.outline'

# Export an editable pptx and check the exit code:
eden-press deck.md -o slides.pptx --format pptx && echo ok || echo "failed: exit $?"

# Script-friendly failure handling — write JSON to a file, check exit status:
eden-press deck.md --format json > out.json || echo "failed: exit $?"

# A bad path fails fast with a parseable error, never a stack trace:
eden-press /no/such.md --format json
# stderr: {"error":{"code":1,"message":"resolveInput: ..."}}
# exit code: 1
```

No browser, no Node, no JavaScript runtime is ever invoked by any of the
above — `html` and `json` are pure `press/`, and `pptx` is pure stdlib OOXML
(`convert/pptx`). This is a CI-enforced invariant
(`scripts/check-no-chromedp.sh` covers `./cmd/...`), not just a convention.
