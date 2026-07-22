// Copyright (c) 2026 AO Cyber Systems — SPDX-License-Identifier: MIT
//
// Package themes go:embeds the three official Marp themes (default, gaia,
// uncover) VERBATIM (CORE-01, LIC-03). The embedded .css files are
// marp-core v4.4.0's OWN compiled output — extracted through the real npm
// oracle by tools/corpus-gen/extract-themes.mjs (research riskiest-item #1:
// marp-core ships no precompiled per-theme CSS, only Sass) — and carry
// Marp's OWN /*! @theme … */ header, never an Eden header (an added leading
// comment would displace the /*! @theme */ block chase/theme/meta.go
// parses). They are excluded from the AO-Cyber addlicense -check via
// .github/workflows/ci.yml's -ignore 'themes/**'.
//
// There is no viewer-side auto-fit/auto-scaling helper embedded here
// (Objective 8, 08-06): auto-fit is Flutter-only via the Dart binding's
// native TextPainter fit (08-07); the assembled HTML never ships that
// script.
//
// This package is the embed holder ONLY. The name-keyed *theme.ThemeSet builder
// lives in press/themes, which imports these strings — go:embed must be
// co-located with the asset files (patterns cannot contain ".."), so the embed
// directives live here in themes/ alongside default.css/gaia.css/uncover.css
// rather than under press/themes/.
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
