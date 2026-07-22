package main

import (
	"net/http"
	"time"

	"github.com/MaxPa1/go-metrics/internal/agent"
)

const (
	pullInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
)

func main() {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	metricAgent := agent.NewMetricAgent("http://localhost:8080/update")

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
