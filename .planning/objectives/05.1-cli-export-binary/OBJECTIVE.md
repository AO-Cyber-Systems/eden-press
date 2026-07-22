---
work: feature
inserted: true
parent_objective: 5
requirements: []
depends_on: [3, 4, 5]
---

# CLI Raster Export Binary (eden-press-export)  [INSERTED 5.1]

## Goal
Provide a turnkey PNG/PDF raster export CLI as a SEPARATE Chrome-permitting binary
`cmd/eden-press-export`, keeping the core `eden-press` binary chromedp-free (user-chosen
"Separate export binary"). Wires the already-built Objective-5 exporters to a command.

## Deliverables
1. **`cmd/eden-press-export`** (standalone `package main`): `eden-press-export <deck.md> -o out.pdf --format pdf`
   and `--format png` (per-slide PNG/JPEG → slide-numbered paths or a dir). Flow:
   resolveInput → press.Render(md,opts) → `chrome.New(convert.Options{...})` →
   `pdf.ToPDF(sess, out, pdf.Options{})` / `png.ToImages(sess, out, png.Options{})` → write bytes.
   REUSE convert/pdf, convert/png, convert/chrome, press/. Mirror cmd/eden-press cobra/flag/input
   ergonomics (small duplicated input+flag helper is fine). Chrome flags: `--browser-path`/CHROME_PATH.
   Clear error + non-zero exit when Chrome can't be discovered.
2. **Re-scope `scripts/check-no-chromedp.sh`**: change the `./cmd/...` TREES entry (added in 4.1) to
   `./cmd/eden-press/...` so the gate still proves the CORE cli is chromedp-free but does NOT flag
   the new export binary (which legitimately imports chromedp). Core trees (press/chase/profiles/bind/
   cmd/eden-press) stay 0; cmd/eden-press-export is the ONLY cmd allowed chromedp.
3. **CI**: extend the export CI job (05-05) to also `go build ./cmd/eden-press-export` and, when the
   pinned headless-shell is present, run a pdf+png smoke. Live-Chrome tests SKIP-guarded (no system
   Chrome in sandbox).
4. **Docs**: update AGENTS.md (+ convert/EXPORT.md if apt) — the separate export binary, its formats,
   and that the core CLI stays Chrome-free.

## Success Criteria
1. `eden-press-export deck.md -o out.pdf --format pdf` produces a `%PDF-` file; `--format png`
   produces per-slide PNG(s) — via press.Render + convert/{chrome,pdf,png}, SKIP-guarded when no Chrome.
2. `go list -deps ./cmd/eden-press/... \| grep -c chromedp` == 0 (core stays clean); the export binary
   builds and is the only cmd with chromedp. check-no-chromedp.sh green on all core trees.
3. Core `eden-press` binary, preview, and the 4.1 json/pptx formats are UNCHANGED.
4. All standing gates green (gofmt, build incl new binary, vet, test, check-no-chromedp re-scoped,
   check-cli-imports on core PASS, Obj-1 corpus/cssdiff, Obj-2 grep-gate); no go.mod churn.

## Non-goals
- No change to the core CLI, preview, watch, serve, or json/pptx. No new rasterization code (reuse convert/).

---
*Created: 2026-07-22 (/devflow:build — separate export binary, user-scoped)*
