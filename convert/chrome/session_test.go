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

package chrome

import (
	"errors"
	"testing"

	"github.com/chromedp/chromedp"

	"github.com/AO-Cyber-Systems/eden-press/convert"
)

// TestSessionMultiTab is Test-list case 6: one Session/browser serves
// multiple tabs. Gated on Chrome presence -- t.Skip cleanly (never fail) if
// Discover cannot find a Chrome/Chromium executable anywhere, which is
// exactly the EXP-04 no-system-Chrome case 05-05 hardens in CI.
func TestSessionMultiTab(t *testing.T) {
	if _, _, err := Discover(DiscoverOptions{}); errors.Is(err, ErrChromeNotFound) {
		t.Skip("no Chrome discovered")
	}

	sess, err := New(convert.Options{})
	if err != nil {
		if errors.Is(err, ErrChromeNotFound) {
			t.Skip("no Chrome discovered")
		}
		t.Fatalf("New: %v", err)
	}
	defer sess.Close()

	tab1, cancel1 := sess.NewTab()
	defer cancel1()
	if err := chromedp.Run(tab1, chromedp.Navigate("about:blank")); err != nil {
		t.Fatalf("tab1 Navigate: %v", err)
	}

	tab2, cancel2 := sess.NewTab()
	defer cancel2()
	if err := chromedp.Run(tab2, chromedp.Navigate("about:blank")); err != nil {
		t.Fatalf("tab2 Navigate: %v", err)
	}

	c1 := chromedp.FromContext(tab1)
	c2 := chromedp.FromContext(tab2)
	if c1.Browser == nil || c2.Browser == nil {
		t.Fatal("expected both tabs to have a non-nil Browser")
	}
	if c1.Browser != c2.Browser {
		t.Error("expected both tabs to share the SAME Browser (one browser, many tabs)")
	}
	if c1.Target == c2.Target {
		t.Error("expected each tab to have its OWN Target (distinct tabs)")
	}
}
