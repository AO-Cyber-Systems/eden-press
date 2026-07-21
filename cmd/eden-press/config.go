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
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

// loadConfigSources is the config-source pipeline applyConfig runs through.
// It loads, in this exact order (load-order IS precedence in koanf -- last
// write wins), a project-local config FILE (YAML/JSON/TOML, chosen by
// extension via parserFor -- koanf deliberately does NOT sniff formats),
// then ENV (EDEN_PRESS_* prefix), then POSFLAG (command-line flags) LAST,
// establishing:
//
//	flags > env > file > compiled defaults
//
// Pitfall 5 (research): posflag.Provider is called with the THREE-arg form,
// passing the live koanf instance k as the third argument. This lets
// posflag see which keys are ALREADY set (by the file or env loads above)
// and skip merging a flag's unchanged DEFAULT value over them -- an unset
// flag's zero-value default must never stomp a file/env-provided value.
func loadConfigSources(k *koanf.Koanf, cmd *cobra.Command) error {
	// 1. file (lowest of the three sources loaded here, above compiled
	// defaults) -- explicit --config override, else project-local discovery.
	if path := discoverConfigPath(cmd); path != "" {
		p, err := parserFor(path)
		if err != nil {
			return err
		}
		if err := k.Load(file.Provider(path), p); err != nil {
			return fmt.Errorf("config: load %s: %w", path, err)
		}
	}

	// 2. env -- EDEN_PRESS_THEME -> "theme", EDEN_PRESS_NO_HIGHLIGHT ->
	// "no-highlight", EDEN_PRESS_HIGHLIGHT_STYLE -> "highlight-style": the
	// mapper produces the SAME key namespace (dash-separated, lowercase)
	// that buildOptions reads through cfg, and that the flag names use.
	if err := k.Load(env.Provider("EDEN_PRESS_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "EDEN_PRESS_")), "_", "-")
	}), nil); err != nil {
		return fmt.Errorf("config: load env: %w", err)
	}

	// 3. posflag LAST (highest precedence) -- instance k passed so unset
	// flags don't stomp the file/env values just loaded above (Pitfall 5).
	if err := k.Load(posflag.Provider(cmd.Flags(), ".", k), nil); err != nil {
		return fmt.Errorf("config: load flags: %w", err)
	}
	return nil
}

// parserFor picks the koanf parser for a config file by its extension.
// koanf has no built-in format sniffing (a viper flaw it deliberately
// avoids), so this explicit switch is the only routing mechanism.
func parserFor(path string) (koanf.Parser, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yml", ".yaml":
		return yaml.Parser(), nil
	case ".json":
		return json.Parser(), nil
	case ".toml":
		return toml.Parser(), nil
	default:
		return nil, fmt.Errorf("config: unsupported extension %q (want .yml/.yaml/.json/.toml)", filepath.Ext(path))
	}
}

// discoverConfigPath resolves which config file (if any) loadConfigSources
// should read: an explicit --config flag overrides discovery entirely;
// otherwise the CURRENT WORKING DIRECTORY is searched for a project-local
// ".marprc.*" file, first match wins. There is deliberately NO global/XDG
// config path in v1 (research Open Question 5, RESOLVED) -- matches Marp
// CLI's own project-local-only behavior. A global path is a documented
// later addition, not a silent v1 inclusion.
func discoverConfigPath(cmd *cobra.Command) string {
	if c, _ := cmd.Flags().GetString("config"); c != "" {
		return c
	}
	for _, name := range []string{".marprc.yml", ".marprc.yaml", ".marprc.json", ".marprc.toml"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}
