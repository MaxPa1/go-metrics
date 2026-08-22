package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type ServConfig struct {
	Address         string        `env:"ADDRESS"`
	LogLevel        string        `env:"LOG_LEVEL"`
	StoreInterval   time.Duration `env:"STORE_INTERVAL"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH"`
	Restore         bool          `env:"RESTORE"`
}

func LoadServConfig() (*ServConfig, error) {
	cfg := &ServConfig{}
	var storeInterval int

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&storeInterval, "i", 300, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "storage.json", "file storage path")
	flag.BoolVar(&cfg.Restore, "r", false, "restore")
	flag.Parse()

	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	if err := env.ParseWithOptions(cfg, options); err != nil {
		return nil, fmt.Errorf("env parse error: %w", err)
	}
	return cfg, nil
}
