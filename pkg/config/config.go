package config

import (
	"fmt"
	"log"

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
}

func NewConfig() ApplicationConfig {
	// setup viper
	viper.SetConfigFile("config.yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file %s", err)
	}
	var config ApplicationConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	config.Database.URI = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Name)
	return config
}
