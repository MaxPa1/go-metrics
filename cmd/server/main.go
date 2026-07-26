package main

import (
	"flag"
	"net/http"

	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

var serverAddress string

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	storage := repository.NewMemStorage()
	metricService := service.NewMetricsService(storage)

	router := chi.NewRouter()
	router.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(metricService))
	router.Get("/value/{metricsType}/{metricsName}", handler.GetMetricsHandler(metricService))
	router.Get("/", handler.GetAllMetricsHandler(metricService))

	return http.ListenAndServe(serverAddress, router)
}

func init() {
	flag.StringVar(&serverAddress, "a", "localhost:8080", "server address")
}
