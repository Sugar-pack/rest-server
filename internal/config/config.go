package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// API contains api settings.
type API struct {
	Bind string `mapstructure:"bind"`
}

type Service struct {
	Address string `mapstructure:"address"`
}

type Server struct {
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// Config is a container for handler config.
type Config struct {
	User   *Service `mapstructure:"user_api"`
	Order  *Service `mapstructure:"order_api"`
	App    *API     `mapstructure:"app_api"`
	Server *Server  `mapstructure:"server"`
}

// GetConfig returns *Config.
func GetConfig() (*Config, error) {
	viper.SetConfigName("config") // hardcoded config name
	viper.AddConfigPath(".")      // hardcoded configfile path
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read config from file: %w", err)
	}
	viper.AutomaticEnv()

	config := new(Config)
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %w", err)
	}

	return config, nil
}
