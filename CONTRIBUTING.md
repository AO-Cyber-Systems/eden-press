# Contributing to Eden Press

Eden Press is MIT-licensed (Copyright (c) 2026 AO Cyber Systems) and is a clean-room,
JavaScript-free Go reimplementation that is **not affiliated with, endorsed by, or sponsored
by the Marp team**. Attribution discipline is a first-class, CI-enforced part of this project —
please read the header and NOTICE rules below before opening a PR.

## Build & test

```sh
go build ./...
go vet ./...
go test ./...
```

`go test ./...` automatically covers `conformance/...` as those packages land. CI
(`.github/workflows/ci.yml`) runs exactly this plus an `addlicense -check`, and is the
acceptance gate for every engine objective.

## Dependency management — additive-only, NEVER `go mod tidy`

`go.mod` and `go.sum` are **hand-authored, complete, and version-pinned in TRD 00-01** and are
the **single source of truth** for the whole objective. Every downstream change is
**additive-only**:

- To add a module, add its pinned `require` line and run **`go mod download <module>`** to
  record checksums in `go.sum`.
- **Do NOT run `go mod tidy`.** In a parallel-worktree workflow, `tidy` prunes requires that a
  sibling branch legitimately needs (its importing `.go` file may not have merged yet),
  producing a broken `go.mod` / merge conflict that fails the 00-01 CI gate. A `require` with
  no importing `.go` file yet is expected and correct — leave it. Any final reconciliation is a
  single post-merge step, never a parallel-job step.

## Per-file license headers

Every `.go` file (and every vendored source asset) must carry a license header, enforced by
[`google/addlicense`](https://github.com/google/addlicense) in `-check` mode in CI. There are
**two** header templates depending on provenance:

### 1. Eden Press original files → Eden Press MIT header

Stamp with:

```sh
addlicense -l mit -s -c "AO Cyber Systems" -y 2026 -v <files-or-dir>
```

This writes the MIT header with `Copyright (c) 2026 AO Cyber Systems` and an
`SPDX-License-Identifier: MIT` line.

### 2. Verbatim-reused Marp assets → PRESERVE the original Marp copyright

The three vendored themes (`default`, `gaia`, `uncover`) and the browser fit/polyfill script
(arriving in Objective 3) are reused **verbatim** and MUST preserve Marp's original copyright
and year — **do not relabel them 2026 or "AO Cyber Systems"**:

```sh
addlicense -l mit -s -c "Marp team (marp-team@marp.app)" -y 2018 -v \
  themes/default.scss themes/gaia.scss themes/uncover.scss themes/browser-fit.js
```

The Marp copyright year is **2018** (Marpit `2018-`, Marp Core / Marp CLI `2018`). Preserve it
exactly.

### Verify

```sh
addlicense -l mit -s -c "AO Cyber Systems" -check .
```

## New vendored / verbatim asset checklist

When a PR adds ANY new third-party asset (theme, script, font, spec/corpus data, generated
fixture) — mirrored in `.github/PULL_REQUEST_TEMPLATE.md`:

- [ ] **NOTICE updated** with the asset's source URL, license, and copyright line.
- [ ] Verbatim-reused files carry their **ORIGINAL** per-file license header (template #2),
      `addlicense -check` green.
- [ ] Provenance recorded (upstream tag / commit / npm tarball integrity hash where relevant).
- [ ] If the asset pins an upstream Marp release, `UPSTREAM-VERSIONS.txt` is consistent with it.
- [ ] `go build ./... && go vet ./... && go test ./...` pass locally.
