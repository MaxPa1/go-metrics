package config

import (
	"flag"
)

type Config struct {
	Address string
}

func ParseFlags() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.Parse()
	return cfg
}
