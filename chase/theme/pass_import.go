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

package theme

import (
	"fmt"
	"strings"
)

// pass_import.go implements Tier-2's recursive @import-theme resolution
// (RESEARCH's Tier-2 order item 5, "importReplace(this)... RECURSIVE
// @import/@import-theme resolve, cycle throws").
//
// Scope-narrowing (deliberate, documented, see this TRD's error_recovery
// and anti_patterns "DO NOT attempt @import RESOLUTION" — that note was
// parse.go's own scope in 01-03; THIS file is exactly the layer where
// resolution now happens): only "@import-theme" atoms — Marpit's
// theme-REGISTRY-based import, resolved by looking a theme NAME up in the
// ThemeSet — are recursively resolved here. A plain "@import" atom (a
// generic external-stylesheet URL import) is left as a recorded,
// unresolved AtRule in the output: this package has no filesystem/network
// resolution layer, and adding one is out of this TRD's scope.
//
// Cycle detection uses a per-BRANCH `visited` set (a name -> bool map),
// COPIED (not mutated in place) at the start of every recursive call: two
// independent, non-overlapping branches that both (non-cyclically)
// import the same theme resolve it independently without a false-positive
// cycle error; only a theme that imports itself, directly or through a
// chain, errors.

// resolveImportTheme resolves name's @import-theme atoms recursively
// against ts, returning the fully-inlined Stylesheet (imported themes'
// Rules/Atoms first, in import order, followed by name's own Rules —
// matching real CSS's cascade-order convention that @import statements
// must precede a stylesheet's own rules). A theme that (directly or
// transitively) imports itself returns an error rather than recursing
// forever.
func resolveImportTheme(ts *ThemeSet, name string, visited map[string]bool) (Stylesheet, error) {
	if visited[name] {
		return Stylesheet{}, fmt.Errorf("theme: circular @import-theme detected: %q", name)
	}
	th, ok := ts.Get(name)
	if !ok {
		return Stylesheet{}, fmt.Errorf("theme: import-theme: unknown theme %q", name)
	}

	nextVisited := make(map[string]bool, len(visited)+1)
	for k, v := range visited {
		nextVisited[k] = v
	}
	nextVisited[name] = true

	var result Stylesheet
	result.Meta = th.Sheet.Meta

	for _, atom := range th.Sheet.Atoms {
		if atom.Name != "import-theme" {
			result.Atoms = append(result.Atoms, atom)
			continue
		}
		imported := unquoteImportName(atom.Prelude)
		importedSheet, err := resolveImportTheme(ts, imported, nextVisited)
		if err != nil {
			return Stylesheet{}, err
		}
		result.Rules = append(result.Rules, importedSheet.Rules...)
		result.Atoms = append(result.Atoms, importedSheet.Atoms...)
	}

	result.Rules = append(result.Rules, th.Sheet.Rules...)
	return result, nil
}

// unquoteImportName strips a single layer of matching double/single
// quotes from an @import-theme prelude (e.g. `"gaia"` -> `gaia`) — the
// AtRule.Prelude text is the raw, still-quoted StringToken data (see
// parse.go's newAtRule/tokensText).
func unquoteImportName(prelude string) string {
	s := strings.TrimSpace(prelude)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
