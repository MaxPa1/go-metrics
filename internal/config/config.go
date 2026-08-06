package config

import (
	"flag"
	"os"
)

type Config struct {
	Address string
}

func ParseFlags() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.Parse()

	if address, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = address
	}
	return cfg
}
