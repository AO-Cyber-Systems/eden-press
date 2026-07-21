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
	"io"
	"os"
)

// inputSource identifies where resolveInput read its markdown from, so
// callers can apply their own source policy without resolveInput enforcing
// it itself. In particular, watch/serve (04-06/04-07) reject a stdin
// source (research Pitfall 8: a filesystem watch or a repeatable HTTP
// convert-on-request has nothing to re-read from a consumed, one-shot
// stdin pipe) -- resolveInput only makes that source OBSERVABLE.
type inputSource int

const (
	// fileSource is the zero value: resolveInput read a named file from
	// disk.
	fileSource inputSource = iota
	// stdinSource: resolveInput read all of stdin (arg was "-").
	stdinSource
)

// String renders the source for error messages and test failure output.
func (s inputSource) String() string {
	switch s {
	case stdinSource:
		return "stdin"
	case fileSource:
		return "file"
	default:
		return "unknown"
	}
}

// resolveInput resolves arg into markdown text: "-" reads all of the
// process's real stdin; any other value is a file path read from disk. It
// reports which source was used (see inputSource) so a caller can apply
// its own policy on top.
func resolveInput(arg string) (md string, source inputSource, err error) {
	return resolveInputFrom(arg, os.Stdin)
}

// resolveInputFrom is resolveInput's testable core: stdin is injected as
// an io.Reader (a bytes/strings.Reader in tests) so no test ever touches
// the real process stdin.
func resolveInputFrom(arg string, stdin io.Reader) (string, inputSource, error) {
	if arg == "-" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", stdinSource, fmt.Errorf("resolveInput: read stdin: %w", err)
		}
		return string(b), stdinSource, nil
	}

	b, err := os.ReadFile(arg)
	if err != nil {
		return "", fileSource, fmt.Errorf("resolveInput: read file %q: %w", arg, err)
	}
	return string(b), fileSource, nil
}
