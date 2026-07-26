package agent

import (
	"flag"
	"time"
)

type Config struct {
	Address        string
	PollInterval   time.Duration
	ReportInterval time.Duration
}

func ParseFlags() *Config {
	cfg := &Config{}
	var pollSeconds, reportSeconds int

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.IntVar(&pollSeconds, "p", 2, "poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", 10, "report interval in seconds")
	flag.Parse()

	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second
	return cfg
}
