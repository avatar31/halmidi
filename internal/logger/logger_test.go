package logger

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/avatar31/halmidi/utils"
)

// Benchmark Test Run Date: 24/12/2025
//
// goos: linux
// goarch: amd64
// pkg: github.com/avatar31/halmidi/internal/logger
// cpu: Intel(R) Xeon(R) CPU E5-2683 v4 @ 2.10GHz
// BenchmarkLoggers/DefaultLogger-16                  95233             11621 ns/op            2790 B/op         20 allocs/op
// BenchmarkLoggers/ZeroLogger-16                    203772              9278 ns/op             521 B/op         11 allocs/op
//
// | Column                              | Meaning                                                |
// | ----------------------------------- | ------------------------------------------------------ |
// | BenchmarkLoggers/DefaultLogger-16   | Benchmark name and GOMAXPROCS (`-16` = 16 CPU threads) |
// | 95233                               | Number of iterations the benchmark ran (`b.N`)         |
// | 11621 ns/op                         | Average time per operation                             |
// | 2790 B/op                           | Bytes allocated per operation                          |
// | 20 allocs/op                        | Allocations per operation                              |
//
// Detailed Analysis:
// ns/op: Average time per call
// DefaultLogger: 11621 ns (~11.6 µs)
// ZeroLogger: 9278 ns (~9.3 µs) → ~20% faster
//
// B/op: Bytes allocated per operation
// DefaultLogger: 2790 B
// ZeroLogger: 521 B → ~5x smaller
//
// allocs/op: Heap allocations
// DefaultLogger: 20
// ZeroLogger: 11 → ~2x fewer allocations
//
// Observation: DefaultLogger is slower and heavier in terms of memory usage compared to ZeroLogger.

func BenchmarkLoggers(b *testing.B) {
	b.Run("DefaultLogger", func(b *testing.B) {
		zeroLogger := createZeroLogger("test", io.Discard)
		logger := defaultLogger{log: zeroLogger}

		getLogger := func(ctx context.Context) Logger {
			loggerWithCtx, ok := ctx.Value(utils.LogCtxKey).(Logger)
			if ok {
				return loggerWithCtx
			}

			return &logger
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getLogger(context.Background()).
				WithError(errors.New("dummy error")).
				WithField("key", "value").
				WithFields(map[string]any{"key1": "value1", "key2": 2}).
				Errorf("Benchmarking default logger. Iteration %d", i)
		}
	})

	b.Run("ZeroLogger", func(b *testing.B) {
		zeroLogger := createZeroLogger("test", io.Discard)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			zeroLogger.
				Err(errors.New("dummy error")).
				Any("key", "value").
				Fields(map[string]any{"key1": "value1", "key2": 2}).
				Msgf("Benchmarking zero logger. Iteration %d", i)
		}
	})
}
