# crap

CRAP scores for Go functions, in one binary.

```
CRAP = CC² × (1 − coverage)³ + CC
```

`CC` is cyclomatic complexity (the same count as [gocyclo](https://github.com/fzipp/gocyclo)), `coverage` is the share of the function's statements that ran under `go test`. A complex function with no tests scores high; the same function fully tested scores just its complexity. The usual limit is 10.

The metric is from Alberto Savoia and Bob Evans; this tool follows the Go part of Uncle Bob's [crapper](https://github.com/unclebob/crapper), used by his [ub-mode](https://github.com/unclebob/ub-mode) agent skill, without the Python runtime.

## Install

```sh
go install github.com/tggo/crap@latest
```

## Use

From a module root:

```sh
crap                      # go test ./... with coverage, then a table, worst first
crap -top 20 ./internal/...
crap -max 10              # exit 1 when any function is over 10 (for CI and agents)
crap -json                # machine-readable
crap -profile cover.out   # reuse an existing profile instead of running go test
crap -tags integration    # build tags for go test and go list
```

```
    CRAP  CC  Cov%                     Function             Location
  1260.0  35   0.0  (*server).youtubeTranscript  cmd/hub/youtube.go:489
   702.0  26   0.0                      collect  cmd/agent/telemetry.go:103
     9.0   9 100.0                   complexity  crap.go:34
```

Exit codes: `0` ok, `1` a function is over `-max`, `2` an error (including failing tests).

## How it counts

- **Complexity:** 1, plus one for each `if`, `for`, `range`, non-default `case`, non-default `select` case, `&&` and `||`. Function literals count toward the function that holds them.
- **Coverage:** profile blocks inside the function's lines, weighted by statements. Files are matched by import path (`module/dir/file.go`), so the repo can live anywhere, not only in a GOPATH-style tree.
- A function with no statements counts as covered. A file that isn't in the profile counts as uncovered.
- Coverage comes from each package's own tests (plain `go test -coverprofile`). To credit cross-package tests, build a profile with `-coverpkg=./...` and pass it with `-profile`.

## License

MIT
