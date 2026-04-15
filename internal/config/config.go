package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppHost     string `env:"APP_HOST" envDefault:"http://localhost:8080/"`
	AppPort     string `env:"APP_PORT" envDefault:"8080"`
	DatabaseUrl string `env:"DATABASE_URL,required"`
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		slog.Warn(".env file not found, applying system defaults")
	}
	cfg := &Config{}
	err = env.Parse(cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%s", strings.TrimSpace(c.AppPort))
}
