package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestBuildConfigDev(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	cfg := buildConfig(true)
	if !cfg.Development {
		t.Fatal("expected development mode")
	}
	if cfg.DisableCaller {
		t.Fatal("expected caller in dev")
	}
	if cfg.DisableStacktrace {
		t.Fatal("expected stacktraces in dev")
	}
	if cfg.Level.Level() != zap.DebugLevel {
		t.Fatalf("level = %s, want debug", cfg.Level.Level())
	}
	if len(cfg.OutputPaths) != 1 || cfg.OutputPaths[0] != "stdout" {
		t.Fatalf("output paths = %v", cfg.OutputPaths)
	}
}

func TestBuildConfigNonDev(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	cfg := buildConfig(false)
	if cfg.Development {
		t.Fatal("expected production mode")
	}
	if !cfg.DisableCaller {
		t.Fatal("expected no caller outside dev")
	}
	if !cfg.DisableStacktrace {
		t.Fatal("expected no stacktraces outside dev")
	}
	if cfg.Level.Level() != zap.InfoLevel {
		t.Fatalf("level = %s, want info", cfg.Level.Level())
	}
}

func TestLogLevelEnvOverride(t *testing.T) {
	t.Setenv("LOG_LEVEL", "error")
	if got := logLevel(true); got != zap.ErrorLevel {
		t.Fatalf("dev override = %s, want error", got)
	}
	if got := logLevel(false); got != zap.ErrorLevel {
		t.Fatalf("non-dev override = %s, want error", got)
	}
}
