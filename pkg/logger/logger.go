package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createLogger(dev bool) *zap.Logger {
	return zap.Must(buildConfig(dev).Build())
}

func buildConfig(dev bool) zap.Config {
	encodeCfg := zap.NewProductionEncoderConfig()
	encodeCfg.TimeKey = "timestamp"
	encodeCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	return zap.Config{
		Level:             zap.NewAtomicLevelAt(logLevel(dev)),
		Development:       dev,
		DisableCaller:     !dev,
		DisableStacktrace: !dev,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encodeCfg,
		OutputPaths: []string{
			"stdout",
		},
		ErrorOutputPaths: []string{
			"stdout",
		},
		InitialFields: map[string]any{
			"pid": os.Getpid(),
		},
	}
}

func logLevel(dev bool) zapcore.Level {
	if level, ok := parseLogLevel(os.Getenv("LOG_LEVEL")); ok {
		return level
	}
	if dev {
		return zap.DebugLevel
	}
	return zap.InfoLevel
}

func parseLogLevel(levelStr string) (zapcore.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return zap.DebugLevel, true
	case "info":
		return zap.InfoLevel, true
	case "warn":
		return zap.WarnLevel, true
	case "error":
		return zap.ErrorLevel, true
	case "dpanic":
		return zap.DPanicLevel, true
	case "panic":
		return zap.PanicLevel, true
	case "fatal":
		return zap.FatalLevel, true
	default:
		return zap.InfoLevel, false
	}
}

func Setup(dev bool) {
	logger := createLogger(dev)
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)
}
