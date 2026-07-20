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

package runner

// CONF-01: the Marp golden corpus. Cases under conformance/corpus/cases/ were
// generated (tools/corpus-gen) by rendering representative Markdown through the
// REAL @marp-team/marp-core — the golden HTML/CSS an Eden Press marp-core-equivalent
// engine must reproduce. Each case declares requires_engine="marp-core"; in
// Objective 0 that engine does not exist yet, so the runner marks these PENDING
// (report.AddPending) rather than FAILING. The gate is "the golden corpus exists,
// loads, and the runner reports it" — the engine that satisfies it lands in
// Objective 3, which reruns this exact corpus for real.

import (
	"path/filepath"
	"testing"

	"github.com/AO-Cyber-Systems/eden-press/conformance/corpus"
	"github.com/AO-Cyber-Systems/eden-press/conformance/htmldiff"
	"github.com/AO-Cyber-Systems/eden-press/conformance/report"
)

func TestMarpCorpus(t *testing.T) {
	root := filepath.Join("..", "corpus", "cases")
	cases, err := corpus.LoadCases(root)
	if err != nil {
		t.Fatalf("load corpus %q: %v (regenerate via tools/corpus-gen: npm ci && node gen.mjs)", root, err)
	}
	if len(cases) == 0 {
		t.Fatal("Marp golden corpus is empty — regenerate via tools/corpus-gen")
	}

	rep := report.New()
	renderBaseline := GoldmarkRenderFunc(NewGoldmark())
	var pending, evaluated int

	for _, c := range cases {
		// Every case is well-formed golden data.
		if c.InputMD == "" {
			t.Errorf("case %q: empty input.md", c.ID)
		}
		if c.ExpectedHTML == "" {
			t.Errorf("case %q: empty expected.html", c.ID)
		}

		switch c.RequiresEngine {
		case "marp-core", "marpit":
			// The Marp-equivalent engine is not built until Objective 3. Track the
			// case as pending the required engine (neither pass nor fail).
			rep.AddPending("marp/" + c.ID)
			pending++
		case "commonmark", "":
			actual, rerr := renderBaseline(c.InputMD, nil)
			eq := false
			if rerr == nil {
				eq, _ = htmldiff.Equal(c.ExpectedHTML, actual)
			}
			rep.Add("base/"+c.ID, eq)
			evaluated++
		default:
			t.Errorf("case %q: unknown requires_engine %q", c.ID, c.RequiresEngine)
		}
	}

	t.Logf("Marp golden corpus: %d cases (%d pending marp-core engine, %d evaluated on goldmark baseline)",
		len(cases), pending, evaluated)
	t.Logf("per-case report:\n%s", rep.Render())

	// Gate: the golden corpus exists, loads, and every Marp case is tracked as
	// pending the not-yet-built engine. CONF-01 is "corpus exists + runner reports",
	// not "the engine passes it" — that assertion flips on in Objective 3.
	if pending == 0 {
		t.Error("no Marp cases were pending the marp-core engine — corpus/runner wiring drift")
	}
}
