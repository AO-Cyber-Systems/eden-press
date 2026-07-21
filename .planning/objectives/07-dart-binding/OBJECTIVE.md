---
work: feature
requirements: [DART-01, DART-02, DART-03, DART-04, DART-05]
depends_on: [3]
---
# Dart/Flutter Binding (bind/capi + bind/dart)
## Goal
Expose press.Render() to Flutter via a single Go C-ABI core (JSON-in/JSON-out) built three ways: Android .so (c-shared), iOS .a (c-archive), Web .wasm (GOOS=js) — no gomobile bind. Gated only on Obj-3 API stability (now met).
## Decision gate (resolve before any WASM-specific code)
standard Go vs. TinyGo for the WASM target — compatibility audit of goldmark, yaml.v3 front-matter, JSON-AST emitter against TinyGo's partial reflection/encoding/json. Functional-correctness risk, not just size. If TinyGo: pin its wasm_exec.js to the exact compiler version.
## Requirements
DART-01..05 (see .planning/REQUIREMENTS.md)
---
*Created: 2026-07-21 (/devflow:build parallel workstreams)*
