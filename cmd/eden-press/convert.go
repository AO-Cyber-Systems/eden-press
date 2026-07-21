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

	"github.com/spf13/cobra"
)

// newConvertCmd registers the explicit "convert" subcommand -- the same
// operation the root command performs by default, but addressable by name
// so `eden-press convert -- <file>` can disambiguate a positional argument
// from a subcommand name (Pitfall 1; see root.go).
func newConvertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert [flags] <in.md|->",
		Short: "Convert a Markdown deck to static HTML (the default mode)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(cmd, args)
		},
	}
	return cmd
}

// runConvert is a STUB -- filled by TRD 04-03 (htmldoc bare-style zero-JS
// assembly + convert pipeline, CLI-01). It must compile so every downstream
// wave can build against a working command tree.
func runConvert(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("convert: not implemented (04-03)")
}
