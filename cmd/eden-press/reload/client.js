// Copyright (c) 2026 AO Cyber Systems
// SPDX-License-Identifier: MIT
//
// Eden-authored EventSource reload client (~5 lines, NOT a Marp asset).
// Spliced into a watch/serve-session document via htmldoc.go's
// InjectScripts seam, with the "%s" placeholder Sprintf'd to the Hub's SSE
// endpoint URL by ClientJS (server.go). EventSource reconnects on its own
// (research Pattern 4) -- no hand-written retry/backoff loop needed.
new EventSource("%s").addEventListener("reload", () => location.reload());
