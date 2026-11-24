package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// ServConfig содержит настройки сервера.
type ServConfig struct {
	Address         string        `env:"ADDRESS"`
	LogLevel        string        `env:"LOG_LEVEL"`
	StoreInterval   time.Duration `env:"STORE_INTERVAL"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH"`
	Restore         bool          `env:"RESTORE"`
	DatabaseDSN     string        `env:"DATABASE_DSN"`
	Key             string        `env:"KEY"`
	AuditFile       string        `env:"AUDIT_FILE"`
	AuditURL        string        `env:"AUDIT_URL"`
}

// LoadServConfig читает флаги и переменные окружения; переменные окружения имеют приоритет.
func LoadServConfig() (*ServConfig, error) {
	cfg := &ServConfig{}
	var storeInterval int

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.IntVar(&storeInterval, "i", 300, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", "storage.json", "file storage path")
	flag.BoolVar(&cfg.Restore, "r", false, "restore")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database DSN")
	flag.StringVar(&cfg.Key, "k", "", "signing key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "URL to POST audit events to")
	flag.Parse()

	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	if err := env.ParseWithOptions(cfg, options); err != nil {
		return nil, fmt.Errorf("env parse error: %w", err)
	}
	return cfg, nil
}
