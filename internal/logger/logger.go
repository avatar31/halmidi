package logger

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	zerolog_log "github.com/rs/zerolog/log"

	"github.com/avatar31/halmidi/config"
	fileutils "github.com/avatar31/halmidi/internal/fileutils/os_file_utils"
	"github.com/avatar31/halmidi/utils"
)

type Logger interface {
	Trace(v ...any)
	Tracef(format string, v ...any)

	Debug(...any)
	Debugf(format string, v ...any)

	Info(v ...any)
	Infof(format string, v ...any)

	Error(...any)
	Errorf(format string, v ...any)

	Warning(...any)
	Warningf(format string, v ...any)

	Panic(v ...any)
	Panicf(format string, v ...any)

	Fatal(...any)
	Fatalf(format string, v ...any)

	WithError(err error) Logger
	WithField(key string, value any) Logger
	WithFields(fields map[string]any) Logger
	WithLogger(ctx context.Context) context.Context
}

// TODO: Implement password masking

type defaultLogger struct {
	log zerolog.Logger
}

var (
	appLogger defaultLogger
	once      sync.Once
)

func InitLogger(ctx context.Context) {
	once.Do(func() {
		cfg := config.GetConfig()
		if err := fileutils.CreateDirIfNotExists(cfg.Logging.Path); err != nil {
			zerolog_log.Fatal().Msgf("failed to create logger directory %s: %v", cfg.Logging.Path, err)
		}

		appLogger = defaultLogger{log: NewZeroLogger("halmidi")}
	})
}

func NewSubModuleLogger(module string) Logger {
	// Since we are creating logger for sub-module, better to set level to Debug
	zLog := defaultLogger{log: NewZeroLogger(module).Level(zerolog.DebugLevel)}
	return &zLog
}

func GetLogger(ctx context.Context) Logger {
	logger, ok := ctx.Value(utils.LogCtxKey).(Logger)
	if ok {
		return logger
	}

	return &appLogger
}

func (l *defaultLogger) Trace(v ...any) {
	l.log.Trace().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Tracef(format string, v ...any) {
	l.log.Trace().Msgf(format, v...)
}

func (l *defaultLogger) Debug(v ...any) {
	l.log.Debug().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Debugf(format string, v ...any) {
	l.log.Debug().Msgf(format, v...)
}

func (l *defaultLogger) Info(v ...any) {
	l.log.Info().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Infof(format string, v ...any) {
	l.log.Info().Msgf(format, v...)
}

func (l *defaultLogger) Error(v ...any) {
	l.log.Error().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Errorf(format string, v ...any) {
	l.log.Error().Msgf(format, v...)
}

func (l *defaultLogger) Warning(v ...any) {
	l.log.Warn().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Warningf(format string, v ...any) {
	l.log.Warn().Msgf(format, v...)
}

func (l *defaultLogger) Panic(v ...any) {
	l.log.Panic().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Panicf(format string, v ...any) {
	l.log.Panic().Msgf(format, v...)
}

func (l *defaultLogger) Fatal(v ...any) {
	l.log.Fatal().Msg(fmt.Sprint(v...))
}

func (l *defaultLogger) Fatalf(format string, v ...any) {
	l.log.Fatal().Msgf(format, v...)
}

func (l *defaultLogger) WithError(err error) Logger {
	return &defaultLogger{log: l.log.With().Err(err).Logger()}
}

func (l *defaultLogger) WithField(key string, value any) Logger {
	return &defaultLogger{log: l.log.With().Any(key, value).Logger()}
}

func (l *defaultLogger) WithFields(fields map[string]any) Logger {
	return &defaultLogger{log: l.log.With().Fields(fields).Logger()}
}

func (l *defaultLogger) WithLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, utils.LogCtxKey, l)
}
