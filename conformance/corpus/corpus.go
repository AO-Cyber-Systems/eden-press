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

// Package corpus defines the golden-corpus case format and loader used by the
// Eden Press conformance harness.
//
// # On-disk schema
//
// A corpus is a directory whose immediate subdirectories are cases. Each case
// directory is named by its ID and contains:
//
//	<root>/
//	  <id>/
//	    input.md       (REQUIRED) the Markdown source fed to the engine
//	    options.json   (REQUIRED) a JSON object of render options; may carry the
//	                              optional "requires_engine" field (see below)
//	    expected.html  (REQUIRED) the golden HTML the engine must structurally match
//	    expected.css   (OPTIONAL) the golden CSS (theme output); absent for
//	                              HTML-only cases
//
// options.json is decoded into Case.Options (map[string]any). The reserved field
// "requires_engine" ("commonmark" | "marpit" | "marp-core") lets a runner mark a
// case that needs an engine not yet built as PENDING rather than FAILING — the
// corpus exists as the acceptance gate before any engine does. It is optional;
// when omitted, Case.RequiresEngine is the empty string.
//
// LoadCases returns cases sorted by ID for deterministic iteration.
package corpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Case is a single golden-corpus fixture loaded from disk.
type Case struct {
	ID             string         // case directory name
	InputMD        string         // input.md content
	Options        map[string]any // decoded options.json
	RequiresEngine string         // "" | "commonmark" | "marpit" | "marp-core"
	ExpectedHTML   string         // expected.html content
	ExpectedCSS    string         // expected.css content, or "" when absent
	Dir            string         // absolute-or-given path to the case directory
}

const (
	fileInputMD      = "input.md"
	fileOptionsJSON  = "options.json"
	fileExpectedHTML = "expected.html"
	fileExpectedCSS  = "expected.css"
)

// LoadCases walks root's immediate subdirectories and loads each as a Case. Every
// case must contain input.md, options.json, and expected.html; expected.css is
// optional. A missing required file or malformed options.json yields a clear
// error identifying the case. Cases are returned sorted by ID.
func LoadCases(root string) ([]Case, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("corpus: read root %q: %w", root, err)
	}
	var cases []Case
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		c, err := loadCase(id, filepath.Join(root, id))
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	return cases, nil
}

// loadCase reads a single case directory into a Case.
func loadCase(id, dir string) (Case, error) {
	inputMD, err := readRequired(id, dir, fileInputMD)
	if err != nil {
		return Case{}, err
	}
	optsRaw, err := readRequired(id, dir, fileOptionsJSON)
	if err != nil {
		return Case{}, err
	}
	var opts map[string]any
	if err := json.Unmarshal([]byte(optsRaw), &opts); err != nil {
		return Case{}, fmt.Errorf("corpus: case %q: parse %s: %w", id, fileOptionsJSON, err)
	}
	expectedHTML, err := readRequired(id, dir, fileExpectedHTML)
	if err != nil {
		return Case{}, err
	}
	expectedCSS, err := readOptional(dir, fileExpectedCSS)
	if err != nil {
		return Case{}, fmt.Errorf("corpus: case %q: read %s: %w", id, fileExpectedCSS, err)
	}

	// Pull requires_engine defensively: a missing or non-string value is "".
	requiresEngine, _ := opts["requires_engine"].(string)

	return Case{
		ID:             id,
		InputMD:        inputMD,
		Options:        opts,
		RequiresEngine: requiresEngine,
		ExpectedHTML:   expectedHTML,
		ExpectedCSS:    expectedCSS,
		Dir:            dir,
	}, nil
}

// readRequired reads a mandatory case file, returning a clear error if missing.
func readRequired(id, dir, name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", fmt.Errorf("corpus: case %q: missing/unreadable required file %s: %w", id, name, err)
	}
	return string(b), nil
}

// readOptional reads an optional case file, returning "" (no error) when absent.
func readOptional(dir, name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}
