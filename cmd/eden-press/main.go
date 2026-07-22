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
	"errors"
	"fmt"
	"os"
)

// main builds the root command and executes it, translating a non-nil error
// into a non-zero, DOCUMENTED process exit code (04.1-01's stable
// exit-code contract: 0 ok, 1 render/runtime error, 2 usage/flag error).
//
// main is the SOLE exit-code sink and the SOLE plain-text error printer:
// root.go sets SilenceErrors:true, so cobra itself never prints an error,
// and cliFail (format.go) never prints in plain text -- it only pre-prints
// the JSON error envelope (and marks printed=true) when --format json is
// active. Every failure is therefore printed exactly once, on exactly one
// path.
func main() {
	if err := newRootCmd().Execute(); err != nil {
		var ce *cliError
		if errors.As(err, &ce) {
			if !ce.printed {
				fmt.Fprintln(os.Stderr, ce.Error())
			}
			os.Exit(ce.code)
		}

		// A non-cliError failure (should not normally occur once every
		// runConvert/emitFormat return path is wrapped via cliFail, but
		// kept as a safety net) falls back to the historical exit(1) plain
		// print.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
