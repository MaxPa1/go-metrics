package config

import (
	"flag"
	"os"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Address  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.Parse()

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = address
	}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
