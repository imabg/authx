package config

import (
	"log"

	"github.com/spf13/viper"
)

type ApplicationConfig struct {
	App struct {
		PORT string `mapstructure:"port"`
		ENV  string `mapstructure:"env"`
	} `mapstructure:"app"`
	Database struct {
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
	return config
}
