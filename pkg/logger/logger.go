package logger

import (
	"context"
	"log"
	"sync"

	"github.com/avatar31/dotfs/fileutils"
	"go.uber.org/zap"

	"github.com/avatar31/halmidi/config"
)

type ctxKey struct{}

var (
	logKey    ctxKey
	once      sync.Once
	appLogger = zap.NewNop()
)

func InitLogger() {
	once.Do(func() {
		cfg := config.GetConfig()
		if err := fileutils.CreateDirIfNotExists(cfg.Logging.Path); err != nil {
			log.Fatalf("failed to create logger directory %s: %v", cfg.Logging.Path, err)
		}

		appLogger = NewZapLogger(config.APP_NAME)
	})
}

func GetLogger(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(logKey).(*zap.Logger)
	if ok && logger != nil {
		return logger
	}

	return appLogger
}

func WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, logKey, appLogger)
}
