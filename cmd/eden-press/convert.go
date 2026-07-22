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

	"github.com/AO-Cyber-Systems/eden-press/press"
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
	registerConvertFlags(cmd)
	return cmd
}

// runConvert is the CLI-01 capstone: the full convert pipeline
// (resolveInput -> buildOptions -> press.Render -> assembleHTML ->
// writeOutput). It backs BOTH the root command's default action and the
// explicit "convert" subcommand (root.go's RunE calls this same function),
// so watch (04-06) and serve (04-07) reuse the identical
// resolveInput->Render->assembleHTML chain rather than re-implementing it.
//
// The default arg (no positional given) is "-" (stdin) -- the documented
// default pairing: `cat deck.md | eden-press` and `eden-press -` both read
// stdin and write stdout. A positional path reads that file instead.
//
// Input is read via resolveInputFrom(arg, cmd.InOrStdin()) rather than
// resolveInput(arg) -- resolveInput hardcodes the process's real os.Stdin
// (see input.go), which is correct for main()'s real invocation (cobra's
// cmd.InOrStdin() falls back to os.Stdin when no reader was injected) but
// would bypass a test's cmd.SetIn(reader). Using cmd.InOrStdin() keeps
// both paths correct through the SAME resolveInputFrom core.
func runConvert(cmd *cobra.Command, args []string) error {
	arg := "-"
	if len(args) == 1 {
		arg = args[0]
	}

	md, _, err := resolveInputFrom(arg, cmd.InOrStdin())
	if err != nil {
		return cliFail(cmd, exitRender, err)
	}

	opts, err := buildOptions(cmd)
	if err != nil {
		return cliFail(cmd, exitRender, err)
	}

	out, err := press.Render(md, opts)
	if err != nil {
		return cliFail(cmd, exitRender, err)
	}

	return emitFormat(cmd, out)
}

// writeOutput writes the assembled document to the --output/-o path if
// set, or to cmd.OutOrStdout() otherwise -- the default pairing (convert's
// own default output is stdout; watch's default of "<input-stem>.html"
// belongs to 04-06, not here). Using cmd.OutOrStdout() (rather than
// os.Stdout directly) is load-bearing for testability: a test's
// cmd.SetOut(buf) captures exactly what a real invocation prints.
//
// runConvert backs BOTH the root command's default action (which has no
// --output flag of its own; only the "convert" subcommand registers it via
// registerConvertFlags) and the "convert" subcommand itself, so --output
// is looked up defensively: cmd.Flags().Lookup, not cmd.Flags().GetString,
// so an invocation through root's bare default (no --output registered at
// all) falls through to stdout instead of erroring on an undefined flag.
func writeOutput(cmd *cobra.Command, doc string) error {
	path := ""
	if f := cmd.Flags().Lookup("output"); f != nil {
		path = f.Value.String()
	}

	if path == "" {
		_, err := fmt.Fprint(cmd.OutOrStdout(), doc)
		return err
	}

	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("writeOutput: write file %q: %w", path, err)
	}
	return nil
}
