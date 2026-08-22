package config

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

type AgentConfig struct {
	Address        string        `env:"ADDRESS"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
}

func LoadAgentConfig() (*AgentConfig, error) {
	cfg := &AgentConfig{}
	var pollSeconds, reportSeconds int

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "server address")
	flag.IntVar(&pollSeconds, "p", 2, "poll interval in seconds")
	flag.IntVar(&reportSeconds, "r", 10, "report interval in seconds")
	flag.Parse()

	cfg.PollInterval = time.Duration(pollSeconds) * time.Second
	cfg.ReportInterval = time.Duration(reportSeconds) * time.Second

	if err := env.ParseWithOptions(cfg, options); err != nil {
		return nil, fmt.Errorf("env parse error: %w", err)
	}
	return cfg, nil
}

var options = env.Options{
	FuncMap: map[reflect.Type]env.ParserFunc{
		reflect.TypeOf(time.Duration(0)): func(s string) (interface{}, error) {
			sec, err := strconv.Atoi(s)
			if err != nil {
				return nil, fmt.Errorf("parse duration as integer seconds: %w", err)
			}
			return time.Duration(sec) * time.Second, nil
		},
	},
}
