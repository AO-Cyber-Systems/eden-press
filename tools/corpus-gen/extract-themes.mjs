// Copyright (c) 2026 AO Cyber Systems — SPDX-License-Identifier: MIT
//
// Marp theme-extraction spike (CORE-01, research riskiest-item #1). The npm
// package @marp-team/marp-core ships NO precompiled per-theme .css — only Sass
// source Go cannot process and that would not byte-match marp-core if
// hand-recompiled. So this sibling of gen.mjs drives the REAL marp-core oracle
// and pulls each built-in theme's already-COMPILED CSS text straight off
// `marp.themeSet.get(name).css` (the accessor confirmed by probing the 4.4.0
// ThemeSet surface: get/getThemeProp/themes()), writing themes/{default,gaia,
// uncover}.css VERBATIM for a Go go:embed. It also vendors the shipped browser
// fit/auto-scaling helper (lib/browser.js) as themes/browser-fit.js (CORE-09).
//
// Leading-comment hoist: chase/theme/meta.go's leadingComment() reads only the
// FIRST CSS comment, and ParseMeta REQUIRES the theme's `/*! @theme … */` block
// to be that leading comment. marp-core's compiled `css` does not always lead
// with it — `default` inlines github-markdown-css rules first, `gaia` emits a
// leading `@charset "UTF-8";`. Each compiled theme contains EXACTLY ONE comment
// (the `/*! @theme … */` metadata/copyright block and nothing else), so we move
// that single block to the front byte-for-byte, leaving every CSS rule/value
// untouched — the only transform applied, and the one the TRD's must_haves
// require ("preserving its `/*! @theme … */` block as its LEADING comment").
//
// Regenerate: `cd tools/corpus-gen && npm ci && node extract-themes.mjs`.

import { Marp } from '@marp-team/marp-core'
import { readFileSync, writeFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'

const here = dirname(fileURLToPath(import.meta.url))
const THEMES = join(here, '..', '..', 'themes')
const require = createRequire(import.meta.url)

// hoistThemeComment moves a compiled theme's single `/*! … @theme … */` block to
// the front so it becomes the file's LEADING comment (chase/theme meta.go
// requirement). No CSS rule or value is altered — only the metadata comment's
// position. Throws if the block is absent or does not carry @theme (a signal the
// extraction accessor changed and must be re-probed).
function hoistThemeComment(css, name) {
  const start = css.indexOf('/*!')
  if (start < 0) throw new Error(`${name}: no /*! comment found in compiled CSS`)
  const end = css.indexOf('*/', start) + 2
  const block = css.slice(start, end)
  if (!/@theme\s+/.test(block)) {
    throw new Error(`${name}: leading /*! block is not the @theme metadata block`)
  }
  if (start === 0) return css.endsWith('\n') ? css : css + '\n' // already leading
  const rest = (css.slice(0, start) + css.slice(end)).replace(/^\s+/, '').replace(/\s+$/, '')
  return `${block}\n${rest}\n`
}

const marp = new Marp()

// Sanity-probe the oracle: the three built-in themes must be present.
const builtins = new Set([...marp.themeSet.themes()].map((t) => t.name))
for (const name of ['default', 'gaia', 'uncover']) {
  if (!builtins.has(name)) {
    throw new Error(`marp-core no longer ships built-in theme ${name} (found: ${[...builtins]})`)
  }
}

let n = 0
for (const name of ['default', 'gaia', 'uncover']) {
  const compiled = marp.themeSet.get(name).css // fully-compiled per-theme CSS text
  const out = hoistThemeComment(compiled, name)
  // Verify the hoist result actually leads with the @theme block for THIS name.
  if (!out.startsWith('/*!') || !new RegExp(`@theme\\s+${name}\\b`).test(out.slice(0, out.indexOf('*/') + 2))) {
    throw new Error(`${name}: hoisted CSS does not lead with its @theme block`)
  }
  writeFileSync(join(THEMES, `${name}.css`), out)
  n++
  console.log(`wrote themes/${name}.css (${out.length} bytes, leading @theme ${name})`)
}

// Vendor the shipped browser fit/auto-scaling helper VERBATIM (CORE-09's
// viewer-side consumer). It is marp-core's own compiled lib/browser.js.
const browserFit = require.resolve('@marp-team/marp-core/lib/browser.js')
const fitSrc = readFileSync(browserFit)
writeFileSync(join(THEMES, 'browser-fit.js'), fitSrc)
console.log(`wrote themes/browser-fit.js (${fitSrc.length} bytes, verbatim marp-core lib/browser.js)`)

console.log(`extracted ${n} compiled themes + browser-fit.js -> ${THEMES}`)
