# Eden Press

Eden Press is a **Go (+ Dart/Flutter client) framework and CLI for generating documents
from Markdown**, Markdown-compatible with [Marp](https://marp.app) but implemented
clean-room with **zero JavaScript in the backend**. It is a developer-facing library first
(`press.Render(md)` embedded in any Go service), a CLI second (`eden-press`), and part of
the **Eden open-source platform**. The v1 target is Marp-compatible slide decks; the
architecture is deliberately format-agnostic (an output-**profile** abstraction) so paged
documents/reports, single-page articles, and EPUB can follow as additional profiles without
a rewrite. See [`PROPOSAL.md`](./PROPOSAL.md) for the full architecture and design source.

> **Core value:** render Marp-compatible documents from Markdown inside a Go service or CLI
> with no JavaScript runtime, no Node, and no browser for HTML/structured output — while
> emitting the document as structured data, not just HTML.

## Status

Objective 0 (Conformance Corpus, Acceptance Gate & Attribution Bootstrap). This repository
currently establishes the Go module, the conformance-test skeleton, and the day-one
licensing/attribution mechanism (LICENSE, NOTICE, per-file MIT headers, a per-PR
NOTICE-update checklist, and a scheduled upstream-drift CI check). Engine code and vendored
theme assets land in later objectives.

## Acknowledgments

Eden Press is inspired by and Markdown-compatible with [Marp](https://marp.app)
(Marpit, Marp Core, Marp CLI), implemented clean-room in Go with zero JavaScript in the
backend. Eden Press is **not affiliated with, endorsed by, or sponsored by the Marp team**.
See [NOTICE](./NOTICE) for full third-party credits.

## License

[MIT](./LICENSE) — Copyright (c) 2026 AO Cyber Systems. Verbatim-reused third-party assets
retain their original license and copyright; see [NOTICE](./NOTICE) and per-file headers.
