# Performance Baseline

**Status:** Accepted
**Date:** 2026-06-02  
**Hardware:** Intel Core i5-4460 @ 3.20GHz, 4 cores  
**Go:** run `go version` to confirm  
**OS:** Linux (amd64)  
**Command:** `go test -tags=integration -bench=. -benchmem -count=3 ./tests/`

## Results

```
goos: linux
goarch: amd64
pkg: github.com/SergioLacerda/strategist-skill/tests
cpu: Intel(R) Core(TM) i5-4460  CPU @ 3.20GHz

BenchmarkInstallAndCompile_RealEmbed-4    394    2843083 ns/op    3777053 B/op    6467 allocs/op
BenchmarkInstallAndCompile_RealEmbed-4    505    2761427 ns/op    3776469 B/op    6467 allocs/op
BenchmarkInstallAndCompile_RealEmbed-4    506    2468633 ns/op    3778817 B/op    6469 allocs/op

BenchmarkStaleCheck_CacheMiss-4        685926       1734 ns/op       352 B/op        3 allocs/op
BenchmarkStaleCheck_CacheMiss-4        689688       1746 ns/op       352 B/op        3 allocs/op
BenchmarkStaleCheck_CacheMiss-4        697648       1856 ns/op       352 B/op        3 allocs/op
```

## Summary

| Benchmark | Mean ns/op | Notes |
|-----------|-----------|-------|
| `BenchmarkInstallAndCompile_RealEmbed` | ~2.7 ms | Full embed extract + CompileAll, cold .compiled/ |
| `BenchmarkStaleCheck_CacheMiss` | ~1.8 µs | IsStale on absent artifact (first-run path) |

## Existing benchmarks (`./internal/compile/` and `./internal/stale/`)

Run separately with `go test -bench=. ./internal/compile/ ./internal/stale/`.  
These use `testutil.MinimalRoot` — representative of minimal fixture overhead, not real-embed load.

## Updating this baseline

Re-run the command above after significant changes to the compile or stale paths.
Paste the new output below this section with its date and hardware context.
