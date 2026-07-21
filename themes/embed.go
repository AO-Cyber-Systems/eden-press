// Copyright (c) 2026 AO Cyber Systems — SPDX-License-Identifier: MIT
//
// Package themes go:embeds the three official Marp themes (default, gaia,
// uncover) and the browser fit/auto-scaling helper VERBATIM (CORE-01,
// LIC-03). The embedded .css/.js are marp-core v4.4.0's OWN compiled output —
// extracted through the real npm oracle by tools/corpus-gen/extract-themes.mjs
// (research riskiest-item #1: marp-core ships no precompiled per-theme CSS, only
// Sass) — and carry Marp's OWN /*! @theme … */ (css) / MIT (js) header, never an
// Eden header (an added leading comment would displace the /*! @theme */ block
// chase/theme/meta.go parses). They are excluded from the AO-Cyber addlicense
// -check via .github/workflows/ci.yml's -ignore 'themes/**'.
//
// This package is the embed holder ONLY. The name-keyed *theme.ThemeSet builder
// lives in press/themes, which imports these strings — go:embed must be
// co-located with the asset files (patterns cannot contain ".."), so the embed
// directives live here in themes/ alongside default.css/gaia.css/uncover.css/
// browser-fit.js rather than under press/themes/.
package themes

import _ "embed"

// DefaultCSS is the verbatim compiled CSS of Marp Core v4.4.0's `default`
// theme, its /*! @theme default … */ metadata/copyright block as the leading
// comment (chase/theme.Load requires it there). Inlines github-markdown-css
// (MIT, sindresorhus) — see NOTICE.
//
//go:embed default.css
var DefaultCSS string

// GaiaCSS is the verbatim compiled CSS of Marp Core v4.4.0's `gaia` theme.
//
//go:embed gaia.css
var GaiaCSS string

// UncoverCSS is the verbatim compiled CSS of Marp Core v4.4.0's `uncover` theme.
//
//go:embed uncover.css
var UncoverCSS string

// BrowserFitJS is the verbatim Marp Core v4.4.0 browser fit/auto-scaling helper
// (lib/browser.js), carrying Marp's MIT header (year 2018). Exposed for a
// viewer-side consumer (CORE-09 emits the markers it reads).
//
//go:embed browser-fit.js
var BrowserFitJS string
