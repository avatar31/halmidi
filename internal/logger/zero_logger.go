package logger

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/avatar31/halmidi/config"
)

func getLogRotator(module string) io.Writer {
	cfg := config.GetConfig()

	// Log rotation setup
	return &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Logging.Path, fmt.Sprintf("%s.log", module)),
		MaxSize:    20, // MB
		MaxAge:     30, // days
		MaxBackups: 20, // old log files to keep
		LocalTime:  true,
		Compress:   true, // gzip old files
	}
}

func getLogWriter(module string) io.Writer {
	// multi := io.MultiWriter(os.Stdout, getLogRotator(module))
	multi := io.MultiWriter(getLogRotator(module))
	return multi
}

func NewZeroLogger(module string) zerolog.Logger {
	cfg := config.GetConfig()

	zerolog.CallerMarshalFunc = callerMarshalFunc
	zerolog.SetGlobalLevel(cfg.Logging.Level)
	return createZeroLogger(module, getLogWriter(module))
}

func createZeroLogger(module string, writer io.Writer) zerolog.Logger {
	return zerolog.New(writer).With().CallerWithSkipFrameCount(3).Timestamp().Str("module", module).Logger()
}

// Custom caller formatter: show only last 2 elements of path
func callerMarshalFunc(pc uintptr, file string, line int) string {
	parts := strings.Split(filepath.ToSlash(file), "/")
	n := len(parts)
	if n >= 2 {
		return fmt.Sprintf("%s:%d", strings.Join(parts[n-2:], "/"), line)
	}

	return fmt.Sprintf("%s:%d", file, line)
}
