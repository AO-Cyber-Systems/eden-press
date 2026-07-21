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

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/AO-Cyber-Systems/eden-press/press"
)

// newPreviewCmd registers the "preview" subcommand.
func newPreviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview [flags] <in.md>",
		Short: "Render a Markdown deck and open it in the default browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreview(cmd, args)
		},
	}
	return cmd
}

// openURL is a package var so tests assert the target without launching a
// real browser -- the ONLY seam preview needs (CLI-04, research
// Don't-Hand-Roll): github.com/pkg/browser owns the cross-platform
// open/xdg-open/ShellExecute fallback chain, so runPreview never shells out
// by hand and never depends on a headless Chrome (no chromedp anywhere in
// this file).
var openURL = func(url string) error { return browser.OpenURL(url) }

// runPreview is CLI-04's capstone: convert the input to a standalone HTML
// document -- reusing 04-03's resolveInput -> buildOptions -> press.Render
// -> assembleHTML chain verbatim -- write it to a temp file, and open it in
// the user's default browser through the openURL seam. Like watch (04-06),
// preview needs a real file path to (re-)open, so stdin ("-") is rejected
// up front, before any conversion work happens.
func runPreview(cmd *cobra.Command, args []string) error {
	arg := "-"
	if len(args) == 1 {
		arg = args[0]
	}
	if arg == "-" {
		return fmt.Errorf("preview: cannot preview stdin (\"-\"); pass a file path")
	}

	md, _, err := resolveInput(arg)
	if err != nil {
		return err
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		return err
	}

	out, err := press.Render(md, opts)
	if err != nil {
		return err
	}

	doc := assembleHTML(out, htmlDocOptions{AutoFitScript: cfg.Bool("auto-fit-script")})

	f, err := os.CreateTemp("", "eden-press-*.html")
	if err != nil {
		return fmt.Errorf("preview: create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(doc); err != nil {
		return fmt.Errorf("preview: write temp file %q: %w", f.Name(), err)
	}

	return openURL("file://" + f.Name())
}
