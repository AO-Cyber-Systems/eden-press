---
schema: 1
id: "eden-press"
extends: "go"
components: [{ path: "bind/dart/", profile: "flutter" }, { path: "internal/latex2mathml/", profile: "go" }]
commands:
  test: { run: "make test" }
provenance:
  reviewed: "2026-09-29"
  sources: ["Makefile", ".github/workflows/ci.yml", ".github/workflows/dart-native.yml", "tools/corpus-gen/package.json"]
---

# Stack Profile: eden-press

<!-- Drafted by `df-tools stack init`. Add no `## ` heading below unless this project genuinely diverges from `go`: an empty section would replace the parent's. Recognized sections: Principles, Idioms, Avoid, Layout & architecture, Testing, Dependencies, Generated code, Security, UI. -->

<!-- stack init notes (see .planning/STACK-REPORT.md):
- tools/corpus-gen/ stack — info (unsupported area (node): no tier-2 profile, so it is not a component; only its own commands are noted)
- tools/corpus-gen/ codegen: npm run gen — sub_area (runs in tools/corpus-gen/: not the primary stack; never a root command)
- root build: docker build --build-arg CHROME_VERSION="${CHROME_VERSION}" -f Dockerfile.export-ci -t eden-press-export-ci:local . — off_stack (tool stack docker does not match extends go)
- root test: make export-test — narrow (not the repo-wide test (single-path: tests one package or path, not the whole repo))
-->
