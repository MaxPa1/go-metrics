package main

import (
	"log"
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.ParseFlags()

	storage := repository.NewMemStorage()
	metricService := service.NewMetricsService(storage)

	router := chi.NewRouter()
	router.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(metricService))
	router.Get("/value/{metricsType}/{metricsName}", handler.GetMetricsHandler(metricService))
	router.Get("/", handler.GetAllMetricsHandler(metricService))

	return http.ListenAndServe(cfg.Address, router)
}
