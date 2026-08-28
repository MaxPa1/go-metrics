package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/app"
	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/logger"
	"github.com/MaxPa1/go-metrics/internal/middleware"
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
	cfg, err := config.LoadServConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	zapLog, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}

	fileStorage, err := repository.NewFileStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	application, err := app.New(*cfg)
	if err != nil {
		return fmt.Errorf("app: %w", err)
	}
	defer application.Close()
	metricService := service.NewMetricsService(fileStorage)

	router := chi.NewRouter()

	router.Use(middleware.GzipMiddleware, middleware.RequestLogger(zapLog))

	router.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(metricService))
	router.Post("/update/", handler.MetricsV2Handler(metricService))
	router.Post("/update", handler.MetricsV2Handler(metricService))

	router.Get("/value/{metricsType}/{metricsName}", handler.GetMetricsHandler(metricService))
	router.Post("/value", handler.GetMetricsV2Handler(metricService, zapLog))
	router.Post("/value/", handler.GetMetricsV2Handler(metricService, zapLog))

	router.Get("/", handler.GetAllMetricsHandler(metricService))
	router.Get("/ping", handler.PingHandler(application))

	return http.ListenAndServe(cfg.Address, router)
}
