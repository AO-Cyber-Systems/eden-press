---
generated: "2026-09-29"
profile: "go"
profile_source: file
components: ["bind/dart/", "internal/latex2mathml/"]
unsupported_areas: ["tools/corpus-gen/"]
counts: { gap: 10, weak: 3, info: 12 }
---

# Stack Report: eden-press

Proposals only — nothing here has been applied. Review, then change CI/runners yourself.

## Gaps

| ID | Component | Finding | Evidence | Proposal |
|---|---|---|---|---|
| GO-FMT | (root) | CI has no Go format check | go.mod | Fail CI on unformatted files; a bare `gofmt -l` lists them and exits 0. `test -z "$(gofmt -l .)"` |
| GO-RACE | (root) | no `go test -race` in CI | go.mod | Run the tests with the race detector (needs CGO). `go test -race ./...` |
| GO-VULN | (root) | CI has no govulncheck | go.mod | Scan dependencies for known vulnerabilities in CI, on a schedule as well as on push. `govulncheck ./...` |
| DART-FORMAT | bind/dart/ | CI has no Dart format check | bind/dart/pubspec.yaml | Fail CI on unformatted Dart. `dart format --output=none --set-exit-if-changed .` |
| DART-TEST | bind/dart/ | CI has no Dart/Flutter test run | bind/dart/pubspec.yaml | Run the tests in CI. `flutter test` |
| GO-FMT | internal/latex2mathml/ | CI has no Go format check | internal/latex2mathml/go.mod | Fail CI on unformatted files; a bare `gofmt -l` lists them and exits 0. `test -z "$(gofmt -l .)"` |
| GO-RACE | internal/latex2mathml/ | no `go test -race` in CI | internal/latex2mathml/go.mod | Run the tests with the race detector (needs CGO). `go test -race ./...` |
| GO-VULN | internal/latex2mathml/ | CI has no govulncheck | internal/latex2mathml/go.mod | Scan dependencies for known vulnerabilities in CI, on a schedule as well as on push. `govulncheck ./...` |
| JS-AUDIT | tools/corpus-gen/ | CI has no dependency audit | tools/corpus-gen/package.json | Audit dependencies in CI. `npm audit --audit-level=high` |
| JS-LINT | tools/corpus-gen/ | CI has no JS/TS linter | tools/corpus-gen/package.json | Lint in CI. `npx eslint .` |

## Weak

| ID | Component | Finding | Evidence | Proposal |
|---|---|---|---|---|
| GO-TIDY | (root) | CI has no go.mod tidy check | go.mod | Fail CI when go.mod/go.sum are not tidy (Go >= 1.23). `go mod tidy -diff` |
| DART-ANALYZE | bind/dart/ | no `dart analyze` / `flutter analyze` in CI | bind/dart/pubspec.yaml | Make analyzer infos and warnings fail the build. `flutter analyze` |
| GO-TIDY | internal/latex2mathml/ | CI has no go.mod tidy check | internal/latex2mathml/go.mod | Fail CI when go.mod/go.sum are not tidy (Go >= 1.23). `go mod tidy -diff` |

## Info

| ID | Component | Finding | Evidence | Proposal |
|---|---|---|---|---|
| CI-HYGIENE | (root) | workflow hygiene: ci.yml (3 actions not pinned to a commit SHA, no timeout-minutes, no concurrency); dart-native.yml (7 actions not pinned to a commit SHA, no timeout-minutes, no concurrency); upstream-drift.yml (1 action not pinned to a commit SHA, no timeout-minutes, no concurrency) | .github/workflows/ci.yml; .github/workflows/dart-native.yml; .github/workflows/upstream-drift.yml | Set a least-privilege `permissions:` block, pin actions by SHA, and add `timeout-minutes` and a `concurrency` group. |
| GO-COVER | (root) | CI runs Go tests without coverage | go.mod | Record coverage (no threshold is implied). `go test -coverprofile=coverage.out ./...` |
| GO-FIX | (root) | CI has no `go fix -diff` | go.mod | Report pending modernizer fixes; `go fix -diff` exits non-zero on a diff. `go fix -diff ./...` |
| GO-LINT | (root) | CI has no Go linter (golangci-lint / staticcheck) | go.mod | Add a linter beyond vet; staticcheck needs no config. `staticcheck ./...` |
| GO-SAST | (root) | CI has no SAST pass (gosec / CodeQL) | go.mod | Optional: add a SAST pass. `gosec ./...` |
| GO-COVER | internal/latex2mathml/ | CI runs Go tests without coverage | internal/latex2mathml/go.mod | Record coverage (no threshold is implied). `go test -coverprofile=coverage.out ./...` |
| GO-LINT | internal/latex2mathml/ | CI has no Go linter (golangci-lint / staticcheck) | internal/latex2mathml/go.mod | Add a linter beyond vet; staticcheck needs no config. `staticcheck ./...` |
| GO-SAST | internal/latex2mathml/ | CI has no SAST pass (gosec / CodeQL) | internal/latex2mathml/go.mod | Optional: add a SAST pass. `gosec ./...` |

## Draft notes

| Key | Candidate | Status | Source |
|---|---|---|---|
| build | `docker build --build-arg CHROME_VERSION="${CHROME_VERSION}" -f Dockerfile.export-ci -t eden-press-export-ci:local .` | off_stack | ci |
| test | `make export-test` | narrow | runner |
| — | — | info | — |
| codegen | `npm run gen` | sub_area | manifest |
