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

package reload

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHubBroadcastDeliversReloadEvent is test-list case 5: an httptest
// subscriber to the reload Hub receives an "event: reload" SSE frame after
// hub.Broadcast("reload").
//
// Hub.ServeHTTP subscribes BEFORE writing/flushing response headers (see
// server.go), so by the time http.Get below returns (headers received),
// the subscription is already registered -- no sleep/poll needed to avoid
// the race between "client connected" and "server broadcasts".
func TestHubBroadcastDeliversReloadEvent(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want %q", got, "text/event-stream")
	}

	h.Broadcast("reload")

	frameCh := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(resp.Body)
		var b strings.Builder
		for {
			line, err := reader.ReadString('\n')
			b.WriteString(line)
			if err != nil {
				frameCh <- b.String()
				return
			}
			if strings.Contains(b.String(), "\n\n") {
				frameCh <- b.String()
				return
			}
		}
	}()

	select {
	case frame := <-frameCh:
		if !strings.Contains(frame, "event: reload") {
			t.Errorf("frame = %q, want it to contain %q", frame, "event: reload")
		}
		if !strings.Contains(frame, "data: reload") {
			t.Errorf("frame = %q, want it to contain %q", frame, "data: reload")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the broadcast reload frame")
	}
}

// TestHubBroadcastReachesMultipleSubscribers proves Broadcast fans out to
// every currently-subscribed connection, not just the first one -- the
// shape watch (this TRD) and serve (04-07) both depend on when more than one
// browser tab is open against the same session.
func TestHubBroadcastReachesMultipleSubscribers(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	const n = 3
	bodies := make([]*http.Response, n)
	for i := range n {
		resp, err := http.Get(srv.URL)
		if err != nil {
			t.Fatalf("http.Get[%d]: %v", i, err)
		}
		bodies[i] = resp
		t.Cleanup(func() { resp.Body.Close() })
	}

	h.Broadcast("reload")

	for i, resp := range bodies {
		reader := bufio.NewReader(resp.Body)
		frameCh := make(chan string, 1)
		go func() {
			var b strings.Builder
			for {
				line, err := reader.ReadString('\n')
				b.WriteString(line)
				if err != nil || strings.Contains(b.String(), "\n\n") {
					frameCh <- b.String()
					return
				}
			}
		}()
		select {
		case frame := <-frameCh:
			if !strings.Contains(frame, "event: reload") {
				t.Errorf("subscriber[%d] frame = %q, want event: reload", i, frame)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("subscriber[%d]: timed out waiting for broadcast frame", i)
		}
	}
}

// TestClientJSSplicesEndpointURL proves ClientJS Sprintf's the given SSE
// endpoint URL into the embedded EventSource snippet verbatim -- the exact
// string rebuildOnce (watch.go) passes through htmldoc.go's InjectScripts
// seam.
func TestClientJSSplicesEndpointURL(t *testing.T) {
	got := ClientJS("http://127.0.0.1:12345/")

	if !strings.Contains(got, "http://127.0.0.1:12345/") {
		t.Errorf("ClientJS output = %q, want it to contain the endpoint URL", got)
	}
	if !strings.Contains(got, "EventSource") {
		t.Errorf("ClientJS output = %q, want it to contain %q", got, "EventSource")
	}
	if !strings.Contains(got, "reload") {
		t.Errorf("ClientJS output = %q, want it to reference the %q event", got, "reload")
	}
}

// TestHubURLBeforeStartIsEmpty documents Hub.URL()'s zero-value contract:
// "" until Start has been called, rather than panicking on a nil listener.
func TestHubURLBeforeStartIsEmpty(t *testing.T) {
	h := NewHub()
	if got := h.URL(); got != "" {
		t.Errorf("URL() before Start = %q, want empty", got)
	}
}

// TestHubStartBindsLoopbackNon8080Port proves Start() binds a real loopback
// listener (never :8080 -- HARD constraint) and that URL() reflects it once
// bound; Close() then tears it down cleanly.
func TestHubStartBindsLoopbackNon8080Port(t *testing.T) {
	h := NewHub()
	url, err := h.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { h.Close() })

	if url == "" {
		t.Fatal("Start returned an empty URL")
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Errorf("URL = %q, want it to start with http://127.0.0.1:", url)
	}
	if strings.Contains(url, ":8080") {
		t.Fatalf("URL = %q, bound the forbidden port 8080", url)
	}
	if got := h.URL(); got != url {
		t.Errorf("URL() = %q, want it to match Start()'s returned %q", got, url)
	}
}
