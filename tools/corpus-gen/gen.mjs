// Copyright (c) 2026 AO Cyber Systems — SPDX-License-Identifier: MIT
//
// Marp golden-corpus generator (CONF-01). Renders representative Markdown through
// the REAL @marp-team/marp-core (the npm oracle) and writes each result as a
// language-neutral golden case: conformance/corpus/cases/<id>/{input.md,
// options.json,expected.html[,expected.css]}. options.json marks requires_engine
// = "marp-core", so in Objective 0 (only the goldmark baseline exists) the corpus
// runner marks these PENDING rather than FAILING — the golden corpus is the gate
// that exists BEFORE the engine that satisfies it. Regenerate: `npm ci && node gen.mjs`.

import { Marp } from '@marp-team/marp-core'
import { mkdirSync, writeFileSync, rmSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const OUT = join(here, '..', '..', 'conformance', 'corpus', 'cases')

// Representative Marpit + Marp Core behaviors. `css:true` writes the golden theme
// CSS too (only where the case specifically exercises theme/size output — the
// expected.css file is optional, keeping HTML-focused cases lean).
const cases = [
  { id: 'marp-basic', md: `# Title\n\nA paragraph with **bold** and *italic*.` },
  { id: 'marp-slide-split', md: `# Slide 1\n\nfirst\n\n---\n\n# Slide 2\n\nsecond` },
  { id: 'marp-heading-divider', md: `---\nheadingDivider: 2\n---\n\n# Deck\n\n## Slide A\n\n## Slide B` },
  { id: 'marp-paginate', md: `---\npaginate: true\n---\n\n# One\n\n---\n\n# Two` },
  { id: 'marp-header-footer', md: `---\nheader: 'Eden Press'\nfooter: 'CONFIDENTIAL'\n---\n\n# Slide` },
  { id: 'marp-class-spot', md: `<!-- _class: lead -->\n\n# Lead slide\n\n---\n\n# Normal slide` },
  { id: 'marp-bg-image', md: `![bg](https://example.com/bg.jpg)\n\n# Over a background` },
  { id: 'marp-bg-split', md: `![bg left](https://example.com/l.jpg)\n\n# Split layout` },
  { id: 'marp-bg-color', md: `<!-- backgroundColor: black -->\n<!-- color: white -->\n\n# Inverted` },
  { id: 'marp-theme-gaia', md: `---\ntheme: gaia\n---\n\n# Gaia`, css: true },
  { id: 'marp-theme-uncover', md: `---\ntheme: uncover\n---\n\n# Uncover`, css: true },
  { id: 'marp-size-4-3', md: `---\nsize: 4:3\n---\n\n# Four by three`, css: true },
  { id: 'marp-math', md: `Inline $E = mc^2$ and display:\n\n$$\\int_0^\\infty e^{-x^2}\\,dx = \\frac{\\sqrt{\\pi}}{2}$$` },
  { id: 'marp-emoji', md: `Shortcode :smile: and unicode 🎉 emoji.` },
  { id: 'marp-code-highlight', md: '```go\nfunc main() {\n\tprintln("hi")\n}\n```' },
  { id: 'marp-gfm-table', md: `| L | C | R |\n|:--|:-:|--:|\n| a | b | c |` },
  { id: 'marp-strikethrough', md: `~~gone~~ and text` },
  { id: 'marp-fit-heading', md: `# <!--fit--> Big fitted heading` },
]

rmSync(OUT, { recursive: true, force: true })
mkdirSync(OUT, { recursive: true })

let n = 0
for (const c of cases) {
  const marp = new Marp({ script: false }) // omit the viewer-side JS blob for clean golden HTML
  const { html, css } = marp.render(c.md)
  const dir = join(OUT, c.id)
  mkdirSync(dir, { recursive: true })
  writeFileSync(join(dir, 'input.md'), c.md.endsWith('\n') ? c.md : c.md + '\n')
  writeFileSync(join(dir, 'options.json'), JSON.stringify({ requires_engine: 'marp-core' }, null, 2) + '\n')
  writeFileSync(join(dir, 'expected.html'), html.trim() + '\n')
  if (c.css) writeFileSync(join(dir, 'expected.css'), css.trim() + '\n')
  n++
}
console.log(`generated ${n} Marp golden cases -> ${OUT}`)
