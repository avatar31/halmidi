## Benchmark

### Go benchmark stat
```sh
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Capture baseline
go test ./internal/core/iam/... -bench=. -benchmem -count=5 > before.txt

# After your changes
go test ./internal/core/iam/... -bench=. -benchmem -count=5 > after.txt

# Compare
benchstat before.txt after.txt
```
