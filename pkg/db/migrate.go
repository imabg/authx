package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/imabg/authx/pkg/config"
	"go.uber.org/zap"
)

func RunMigrations(cfg config.ApplicationConfig) error {
	source, err := migrationSource(cfg.Database.MigrationURI)
	if err != nil {
		zap.L().Error("failed to resolve migration path", zap.Error(err))
		return err
	}
	m, err := migrate.New(source, cfg.Database.URI)
	if err != nil {
		zap.L().Error("failed to initialize migrations", zap.Error(err))
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		zap.L().Error("failed to run migrations", zap.Error(err))
		return err
	}
	zap.L().Info("database migrations are up to date")
	return nil
}

func migrationSource(uri string) (string, error) {
	if uri == "" {
		uri = "migrations"
	}
	path := strings.TrimPrefix(uri, "file://")
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if isDir(abs) {
		return "file://" + abs, nil
	}
	found, err := findMigrationsDir()
	if err != nil {
		return "", fmt.Errorf("migrations directory not found at %s: %w", abs, err)
	}
	return "file://" + found, nil
}

func findMigrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, "migrations")
		if isDir(candidate) {
			return candidate, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return "", errors.New("no migrations directory in module")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("reached filesystem root")
		}
		dir = parent
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
