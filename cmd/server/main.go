package main

import (
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	storage := repository.NewMemStorage()
	metricService := service.NewMetricsService(storage)

	mux := http.NewServeMux()
	mux.HandleFunc("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(metricService))
	return http.ListenAndServe(":8080", mux)
}
