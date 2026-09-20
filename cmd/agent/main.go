package main

import (
	"context"
	"log"
	"time"

	"github.com/MaxPa1/go-metrics/internal/agent"
	"github.com/MaxPa1/go-metrics/internal/config"

	"github.com/go-resty/resty/v2"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := resty.New().
		SetTimeout(5 * time.Second)

	metricAgent := agent.NewMetricAgent(cfg)

	lastReport := time.Now()

	for {
		metricAgent.UpdateMetrics()
		time.Sleep(cfg.PollInterval)

		if time.Since(lastReport) >= cfg.ReportInterval {
			metricAgent.SendMetrics(ctx, client)
			lastReport = time.Now()
		}
	}
}
