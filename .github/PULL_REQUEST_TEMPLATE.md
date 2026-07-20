# Pull Request

## Summary

<!-- What does this PR change and why? -->

## Attribution checklist

- [ ] If this PR vendors/embeds a new third-party asset (theme, script, font, spec/corpus
      data, generated fixture), **NOTICE has been updated** with its source, license, and
      copyright.
- [ ] Verbatim-reused files carry their **ORIGINAL** per-file license header
      (Marp assets keep `Copyright (c) 2018 Marp team (marp-team@marp.app)`);
      `addlicense -check` is green.
- [ ] Eden Press original `.go` files carry the Eden MIT header
      (`Copyright (c) 2026 AO Cyber Systems`, SPDX `MIT`).
- [ ] If a vendored asset pins an upstream Marp release, `UPSTREAM-VERSIONS.txt` is consistent.

## Dependency checklist

- [ ] Any new module was added as a pinned `require` + `go mod download` (**no `go mod tidy`** —
      see CONTRIBUTING.md).

## Verification

- [ ] `go build ./... && go vet ./... && go test ./...` pass locally.
