package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createLogger() *zap.Logger {
	encodeCfg := zap.NewProductionEncoderConfig()
	encodeCfg.TimeKey = "timestamp"
	encodeCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	config := zap.Config {
		Level: zap.NewAtomicLevelAt(getLogLevelFromEnv()),
		Development: false,
		DisableCaller: false,
		DisableStacktrace: false,
		Sampling: nil,
		Encoding: "json",
		EncoderConfig: encodeCfg,
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
	return  zap.Must(config.Build())
}

func getLogLevelFromEnv() zapcore.Level {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func Setup() {
	logger := createLogger()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)
}