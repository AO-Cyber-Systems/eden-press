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

// Package reload implements the CLI's live-reload channel: a stdlib-only
// (no websocket dependency) Server-Sent-Events broadcast Hub, plus a tiny
// embedded EventSource client snippet. This is the SHARED plumbing watch
// (04-06, this TRD) and serve (04-07) both reuse -- one reload mechanism,
// not two (research Pattern 4).
//
// The channel is deliberately one-directional (server tells the browser
// "reload," full stop): SSE over a plain net/http handler, not a
// websocket. The browser side (client.js) uses the built-in EventSource
// API, which auto-reconnects on its own -- no hand-written retry/backoff
// loop, unlike a websocket client would need.
package reload

import (
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"sync"
)

//go:embed client.js
var clientJSTemplate string

// ClientJS returns the embedded EventSource reload snippet (client.js) with
// url Sprintf'd into its "%s" placeholder -- the exact string a caller
// splices into an assembled document's own <script> block via htmldoc.go's
// InjectScripts seam (watch-session output only; the default zero-JS
// convert output never calls this).
func ClientJS(url string) string {
	return fmt.Sprintf(clientJSTemplate, url)
}

// Hub is a Server-Sent-Events broadcast channel: any number of subscribers
// (open browser EventSource connections) each receive every message passed
// to Broadcast. A Hub is also an http.Handler (ServeHTTP) usable standalone
// -- watch (this TRD) calls Start to have the Hub own its own loopback
// listener, since watch has no other HTTP server to mount onto; serve
// (04-07) can instead mount ServeHTTP directly on its own existing mux,
// reusing the identical broadcast/subscribe machinery either way.
type Hub struct {
	mu   sync.Mutex
	subs map[chan string]struct{}

	ln  net.Listener
	srv *http.Server
}

// NewHub returns a ready-to-use Hub with no subscribers and no listener
// bound yet (see Start).
func NewHub() *Hub {
	return &Hub{subs: make(map[chan string]struct{})}
}

// subscribe registers a new subscriber channel and returns it. The channel
// is buffered so Broadcast (below) never blocks delivering to OTHER
// subscribers because one particular subscriber's connection is slow.
func (h *Hub) subscribe() chan string {
	ch := make(chan string, 4)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// unsubscribe removes and closes ch. Safe to call exactly once per
// subscribe (ServeHTTP defers it).
func (h *Hub) unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

// Broadcast sends msg to every currently-subscribed connection.
// Non-blocking per subscriber (select/default): a subscriber whose 4-slot
// buffer is already full is skipped for THIS message rather than stalling
// every other subscriber or the caller (rebuildOnce in watch.go calls this
// synchronously after every rebuild, so it must never block).
func (h *Hub) Broadcast(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

// ServeHTTP implements http.Handler: it holds the connection open as a
// text/event-stream and writes "event: <msg>\ndata: <msg>\n\n" + Flush for
// every Broadcast message delivered to this subscriber, until the request
// context is done (the client disconnects) or the Hub's channel is closed.
//
// Subscribing happens BEFORE the response headers are written/flushed, so
// that by the time a client sees the response headers, its subscription is
// already registered -- a Broadcast call issued right after a client's
// request returns is never lost to a subscribe/broadcast race.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch := h.subscribe()
	defer h.unsubscribe(ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	_ = rc.Flush() // land headers on the client immediately, even before the first message

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg, msg); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

// Start binds a loopback-only listener on an EPHEMERAL port (never 8080 --
// HARD constraint) and begins serving this Hub's own ServeHTTP handler in
// the background. Returns the resolved endpoint URL (see URL). Safe to call
// once per Hub; watch.go calls this exactly once per watch session.
func (h *Hub) Start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("reload: listen: %w", err)
	}
	h.ln = ln
	h.srv = &http.Server{Handler: h}
	go h.srv.Serve(ln) //nolint:errcheck // Serve's only error on a clean Close is http.ErrServerClosed, expected

	return h.URL(), nil
}

// URL returns this Hub's SSE endpoint ("http://127.0.0.1:PORT/"). Returns ""
// until Start has bound a listener.
func (h *Hub) URL() string {
	if h.ln == nil {
		return ""
	}
	return "http://" + h.ln.Addr().String() + "/"
}

// Close shuts down the Hub's HTTP listener, if Start was ever called. Safe
// to call on a Hub that was never Started.
func (h *Hub) Close() error {
	if h.srv == nil {
		return nil
	}
	return h.srv.Close()
}
