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

// Command gencases writes conformance/corpus/xlsxtyping/cases.json from the
// xlsxtyping package's Cases and StructuralCases.
//
// It is invoked by the //go:generate line in corpus.go, which runs with the
// package directory as its working directory, so the output lands beside
// corpus.go. Run it as `go generate ./conformance/corpus/xlsxtyping`.
//
// cases.json is never hand-edited: TestCorpusJSONIsCurrent fails when the
// committed bytes differ from what this program would write.
package main

import (
	"fmt"
	"os"

	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus/xlsxtyping"
)

const outFile = "cases.json"

func main() {
	b, err := xlsxtyping.CasesJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gencases: marshal: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outFile, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gencases: write %s: %v\n", outFile, err)
		os.Exit(1)
	}
	fmt.Printf("gencases: wrote %s (%d bytes, %d cell cases, %d structural cases)\n",
		outFile, len(b), len(xlsxtyping.Cases), len(xlsxtyping.StructuralCases))
}
