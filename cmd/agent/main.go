package main

import (
	"time"

	"github.com/MaxPa1/go-metrics/internal/agent"
	"github.com/go-resty/resty/v2"
)

func main() {
	config := agent.ParseFlags()

	client := resty.New().
		SetTimeout(5 * time.Second)

	metricAgent := agent.NewMetricAgent(config)

	lastReport := time.Now()

	for {
		metricAgent.UpdateMetrics()
		time.Sleep(config.PollInterval)

		if time.Since(lastReport) >= config.ReportInterval {
			metricAgent.SendMetrics(client)
			lastReport = time.Now()
		}
	}
}
