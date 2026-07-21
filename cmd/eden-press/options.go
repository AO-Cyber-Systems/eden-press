// Copyright (c) 2026 AO Cyber Systems
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// SPDX-License-Identifier: MIT

package main

import (
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// cfg is the single package-level koanf instance every mode resolves its
// options through. It is populated by applyConfig (the root command's
// PersistentPreRunE, so it runs before any subcommand's RunE) and read by
// buildOptions -- the one place flag/config-file/env precedence (04-04)
// layers in without any downstream TRD editing this mapping.
var cfg = koanf.New(".")

// applyConfig loads every configured source into cfg via loadConfigSources
// (config.go). It is the root command's PersistentPreRunE, so it runs
// exactly once per invocation, before the resolved subcommand's RunE reads
// cfg through buildOptions.
func applyConfig(cmd *cobra.Command) error {
	return loadConfigSources(cfg, cmd)
}

// buildOptions is the single options-resolution point every mode calls: it
// reads RESOLVED values through cfg (populated by applyConfig, so
// config-file/env precedence added by 04-04 layers in without editing this
// mapping) and maps them onto a press.Options. Unset flags are passed
// through as their Go zero values VERBATIM -- press.Render owns the
// ""->front-matter->"default" (etc.) fallback chain; buildOptions must
// never re-implement it (anti-pattern, research Pattern 1).
//
// themeCSS(cmd) (filled by 04-05 in themeset.go) is called here so this is
// the one call site every downstream mode shares; its result (tcss) is
// assigned verbatim to press.Options.ThemeCSS (04-01's additive field) --
// buildOptions does no further interpretation of it, matching the "CLI reads
// files, press/ registers them" boundary 04-01/04-05 establish.
func buildOptions(cmd *cobra.Command) (press.Options, error) {
	tcss, err := themeCSS(cmd)
	if err != nil {
		return press.Options{}, err
	}

	return press.Options{
		Theme:          cfg.String("theme"),
		ThemeCSS:       tcss,
		Profile:        cfg.String("profile"),
		MathMode:       cfg.String("math"),
		NoHighlight:    cfg.Bool("no-highlight"),
		HighlightStyle: cfg.String("highlight-style"),
		InlineSVG:      cfg.Bool("inline-svg"),
	}, nil
}
