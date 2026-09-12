package logger

import (
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/avatar31/halmidi/config"
	osfileutils "github.com/avatar31/halmidi/internal/fileutils/os_file_utils"
)

// Creates and returns a new Zap logger instance
// Its user responsibility to call Sync() on the returned logger
func NewZapLogger(module string) *zap.Logger {
	cfg := config.GetConfig()
	writer := zapcore.AddSync(getLogWriter(module))

	var zapErrWriter zapcore.WriteSyncer
	if file, err := osfileutils.CreateFileIfNotExists(filepath.Join(cfg.Logging.Path, "zap.log")); err == nil {
		zapErrWriter = zapcore.AddSync(file)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.MessageKey = "message"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		writer,
		zap.DebugLevel,
	)

	fiedsOption := zap.Fields(zap.String("module", module))
	stachTraceOption := zap.AddStacktrace(zap.ErrorLevel)
	callerOption := zap.AddCaller()
	if zapErrWriter == nil {
		return zap.New(core, callerOption, stachTraceOption, fiedsOption)
	}

	return zap.New(core, callerOption, stachTraceOption, fiedsOption, zap.ErrorOutput(zapErrWriter))
}
