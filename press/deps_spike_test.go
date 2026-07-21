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

package press

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
)

// TestGoldmarkEmojiCompat is research riskiest-item #3: the build+run gate
// that PROVES github.com/yuin/goldmark-emoji v1.0.6 compiles and runs against
// this repo's github.com/yuin/goldmark v1.8.4, BEFORE any battery TRD (CORE-06,
// TRD 03-04) commits to reusing it.
//
// goldmark-emoji's own go.mod floors goldmark at v1.7.10 while this repo pins
// v1.8.4; under Go's minimal version selection the higher v1.8.4 is selected,
// so a mere version-floor bump is expected and fine. What this test guards
// against is the OTHER failure mode: an actual API break between v1.7.10's and
// v1.8.4's goldmark.Extender / parser / renderer contracts that would stop the
// emoji extension from BUILDING or from producing output at all. If the reuse
// plan for CORE-06 were to collapse, it would surface right here.
//
// It builds a goldmark engine with emoji.New(emoji.WithRenderingMethod(
// emoji.Twemoji)), converts ":smile:", and asserts the output is a twemoji
// <img> tag (emoji.DefaultTwemojiTemplate) rather than the untouched literal.
func TestGoldmarkEmojiCompat(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			emoji.New(emoji.WithRenderingMethod(emoji.Twemoji)),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(":smile:"), &buf); err != nil {
		t.Fatalf("goldmark-emoji v1.0.6 Convert against goldmark v1.8.4 failed: %v", err)
	}

	got := buf.String()

	// A twemoji render is an <img class="emoji" ... src="...twemoji..."> tag.
	// If the shortcode were left unrendered, the output would still literally
	// contain ":smile:" and carry no <img.
	if !strings.Contains(got, "<img") {
		t.Fatalf("emoji extension did not render :smile: to an <img> tag; got %q", got)
	}
	if !strings.Contains(got, `class="emoji"`) {
		t.Fatalf("twemoji output missing the expected class=\"emoji\"; got %q", got)
	}
	if !strings.Contains(got, "twemoji") {
		t.Fatalf("twemoji output missing the twemoji CDN src; got %q", got)
	}
	if strings.Contains(got, ":smile:") {
		t.Fatalf("shortcode :smile: was left unrendered; got %q", got)
	}
}
