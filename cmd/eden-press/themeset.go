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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// themeCSS resolves --theme-set (a repeatable StringSlice flag, also
// settable via config once 04-04 layers a file/env provider ahead of
// posflag) into raw custom-theme CSS TEXT: one entry per path, in the order
// the paths resolve through cfg. buildOptions assigns the returned slice
// directly to press.Options.ThemeCSS (04-01's additive field) -- press.Render
// registers each entry via the same chase/theme.Load + ThemeSet.Add path the
// 3 embedded themes use, so a custom theme becomes selectable by its own
// leading `/* @theme <name> */` name through --theme <name>.
//
// This function reads BYTES ONLY: it does not parse or validate the CSS, or
// the theme's own `@theme` metadata -- that is press.Render's job (via
// chase/theme.Load), so a malformed file surfaces press.Render's own clear
// "load custom theme CSS" error at render time, not a CLI-side one. An
// unreadable path is the one error this function DOES own, wrapped as
// "theme-set: read <path>: <err>" so the failure names the offending file.
//
// An empty/unset --theme-set (cfg.Strings returns an empty, non-nil slice
// when the key is absent) returns (nil, nil): a true no-op, matching
// Options.ThemeCSS's nil zero value and packThemeCSS's `range nil` no-op.
func themeCSS(cmd *cobra.Command) ([]string, error) {
	paths := cfg.Strings("theme-set")
	if len(paths) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("theme-set: read %s: %w", p, err)
		}
		out = append(out, string(b))
	}
	return out, nil
}
