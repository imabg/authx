package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type ApplicationConfig struct {
	App struct {
		PORT string `mapstructure:"port"`
		ENV  string `mapstructure:"env"`
	} `mapstructure:"app"`
	Database struct {
		Host         string `mapstructure:"host"`
		Port         int    `mapstructure:"port"`
		User         string `mapstructure:"user"`
		Password     string `mapstructure:"password"`
		Name         string `mapstructure:"name"`
		SSLMode      string `mapstructure:"sslmode"`
		URI          string
		MigrationURI string `mapstructure:"migrationURI"`
	} `mapstructure:"database"`
	JWT struct {
		Secret string `mapstructure:"secret"`
		Issuer string `mapstructure:"issuer"`
	} `mapstructure:"jwt"`
	AdminAPIKey   string `mapstructure:"admin_api_key"`
	PublicBaseURL string `mapstructure:"public_base_url"`
	Mail          struct {
		Driver string `mapstructure:"driver"`
	} `mapstructure:"mail"`
}

// IsDevelopment reports whether env is a local/dev environment.
// Accepts "dev" and "development" (case-insensitive). Any other value is treated as non-dev.
func IsDevelopment(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "dev", "development":
		return true
	default:
		return false
	}
}

func (c ApplicationConfig) IsDevelopment() bool {
	return IsDevelopment(c.App.ENV)
}

func NewConfig() ApplicationConfig {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	for _, dir := range configSearchDirs() {
		viper.AddConfigPath(dir)
	}
	viper.AutomaticEnv()
	viper.SetDefault("app.port", ":8080")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.migrationURI", "file://migrations")
	viper.SetDefault("jwt.issuer", "authx")
	viper.SetDefault("public_base_url", "http://localhost:3000")
	viper.SetDefault("mail.driver", "log")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file %s", err)
	}
	var config ApplicationConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	sslMode := config.Database.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	config.Database.URI = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Database.User,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Name,
		sslMode,
	)
	return config
}

func configSearchDirs() []string {
	wd, err := os.Getwd()
	if err != nil {
		return []string{"."}
	}
	var dirs []string
	dir := wd
	for {
		dirs = append(dirs, dir)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dirs
}
