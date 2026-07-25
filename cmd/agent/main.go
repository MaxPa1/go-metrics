package main

import (
	"time"

	"github.com/MaxPa1/go-metrics/internal/agent"
	"github.com/go-resty/resty/v2"
)

const (
	pullInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
)

func main() {
	client := resty.New().
		SetTimeout(5 * time.Second)

	metricAgent := agent.NewMetricAgent(agent.PostMetricsUrl)

	lastReport := time.Now()

	for {
		metricAgent.UpdateMetrics()
		time.Sleep(pullInterval)

		if time.Since(lastReport) >= reportInterval {
			metricAgent.SendMetrics(client)
			lastReport = time.Now()
		}
	}
}
