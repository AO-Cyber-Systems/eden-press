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
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

// loadConfigSources is the config-source pipeline applyConfig runs through.
// THIS IS THE BASELINE STUB: today it loads ONLY the posflag provider
// (command-line flags), so the CLI works flags-only with no config file
// involved. TRD 04-04 (koanf config loading, CLI-06) PREPENDS file+env
// providers BEFORE this posflag load, establishing flags > env > file
// precedence -- it does not change this function's final posflag.Load call,
// only what runs before it.
//
// Pitfall 5 (research): posflag.Provider is called with the THREE-arg form,
// passing the live koanf instance k as the third argument. This is
// load-bearing: it lets posflag see which keys are ALREADY set (by a config
// file or env, once 04-04 lands) and skip merging a flag's unchanged
// DEFAULT value over them. A two-arg call (or a fresh koanf.New passed as
// the third arg instead of the real instance) would let an unset flag's
// zero-value default silently stomp file/env-provided values.
func loadConfigSources(k *koanf.Koanf, cmd *cobra.Command) error {
	return k.Load(posflag.Provider(cmd.Flags(), ".", k), nil)
}
