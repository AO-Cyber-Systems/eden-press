# conformance/spec — vendored CommonMark + GFM spec suites

This directory vendors generated `spec.json` fixture files for the CONF-04
acceptance gate (`conformance/runner/spec_test.go`). They are pre-generated and
`go:embed`-ed (see `embed.go`) so the test suite is hermetic — no network fetch
or `spec.txt` parsing happens at `go test` time.

## Sources

- **CommonMark** (`commonmark/spec.json`): `github.com/commonmark/commonmark-spec`,
  tag `0.31.2`.
- **GFM** (`gfm/spec.json` + `gfm/extensions.json`): `github.com/github/cmark-gfm`,
  live `master` branch (`test/spec.txt` + `test/extensions.txt`). This is
  deliberately the live split-file source, NOT the frozen 2019 `0.29.0.gfm.13`
  tag that backs the public `https://github.github.com/gfm/` page — see
  00-RESEARCH.md Pattern 2 / Anti-Patterns.

Exact pinned commits + measured example counts are recorded in `VERSIONS.txt`.

## Regeneration commands

Run these on a machine with network access + Python 3, then commit the
resulting JSON (and update `VERSIONS.txt` + `NOTICE` with the new tag/counts):

```bash
# CommonMark
git clone --branch 0.31.2 --depth 1 https://github.com/commonmark/commonmark-spec.git
python3 commonmark-spec/test/spec_tests.py --dump-tests \
  --spec commonmark-spec/spec.txt > conformance/spec/commonmark/spec.json

# GFM (live cmark-gfm master — same spec_tests.py mechanism, forked from commonmark-spec)
git clone --depth 1 https://github.com/github/cmark-gfm.git
python3 cmark-gfm/test/spec_tests.py --dump-tests \
  --spec cmark-gfm/test/spec.txt       > conformance/spec/gfm/spec.json
python3 cmark-gfm/test/spec_tests.py --dump-tests \
  --spec cmark-gfm/test/extensions.txt > conformance/spec/gfm/extensions.json
```

Note: `spec_tests.py --dump-tests` reads the `--spec`/`-s` path argument, not
stdin — pass the file explicitly (piping `< spec.txt` on its own fails with
`FileNotFoundError: spec.txt`, since the script's default path is relative to
the current working directory, not stdin).

Each example in the dumped JSON follows this schema (mirrors goldmark's own
`commonmark_test.go` shape):

```go
type specExample struct {
    Markdown  string `json:"markdown"`
    HTML      string `json:"html"`
    Example   int    `json:"example"`
    StartLine int    `json:"start_line"`
    EndLine   int    `json:"end_line"`
    Section   string `json:"section"`
}
```

## When to re-vendor

Re-run the commands above (and update `VERSIONS.txt` + `NOTICE` in the same
change) whenever:

- CommonMark cuts a new spec tag, or
- cmark-gfm's `master` `test/spec.txt`/`test/extensions.txt` meaningfully
  changes (e.g. a new GFM extension section is added).

This mirrors the spirit of the project's upstream-drift mechanism
(`.github/workflows/upstream-drift.yml`), applied to spec data rather than the
Marp packages it tracks.
