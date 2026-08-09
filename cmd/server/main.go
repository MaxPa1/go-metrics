package main

import (
	"log"
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/logger"
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
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	storage := repository.NewMemStorage()
	metricService := service.NewMetricsService(storage)

	router := chi.NewRouter()

	router.Use(logger.RequestLogger)

	router.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(metricService))
	router.Get("/value/{metricsType}/{metricsName}", handler.GetMetricsHandler(metricService))
	router.Get("/", handler.GetAllMetricsHandler(metricService))

	return http.ListenAndServe(cfg.Address, router)
}
