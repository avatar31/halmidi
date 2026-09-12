package logger

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/avatar31/dotfs/fileutils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func NewZapLogger(module string) *zap.Logger {
	cfg := config.GetConfig()
	writer := zapcore.AddSync(getLogWriter(module))

	var zapErrWriter zapcore.WriteSyncer
	if file, err := fileutils.CreateFileIfNotExists(filepath.Join(cfg.Logging.Path, "zap.log")); err == nil {
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
