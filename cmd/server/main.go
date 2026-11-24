// Команда server принимает метрики по HTTP и сохраняет их в хранилище.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaxPa1/go-metrics/internal/app"
	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/logger"
	"github.com/MaxPa1/go-metrics/internal/middleware"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadServConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	zapLog, err := logger.Initialize(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	defer zapLog.Sync() // сбросить буфер логов перед выходом

	application, err := app.New(ctx, *cfg, zapLog)
	if err != nil {
		return fmt.Errorf("app: %w", err)
	}
	defer closeApp(application, zapLog)

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: newRouter(application, cfg.Key, zapLog),
	}
	return serve(ctx, srv, zapLog)
}

func newRouter(application *app.App, key string, zapLog *zap.SugaredLogger) http.Handler {
	metricService := service.NewMetricsService(application.Storage())
	auditor := application.Auditor()

	router := chi.NewRouter()
	router.Use(middleware.GzipMiddleware, middleware.RequestLogger(zapLog), middleware.HashMiddleware(key))

	router.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(metricService, auditor))
	router.Post("/update/", handler.MetricsV2Handler(metricService, auditor))
	router.Post("/update", handler.MetricsV2Handler(metricService, auditor))

	router.Post("/updates/", handler.MetricsListHandler(metricService, auditor))
	router.Post("/updates", handler.MetricsListHandler(metricService, auditor))

	router.Get("/value/{metricsType}/{metricsName}", handler.GetMetricsHandler(metricService))
	router.Post("/value", handler.GetMetricsV2Handler(metricService, zapLog))
	router.Post("/value/", handler.GetMetricsV2Handler(metricService, zapLog))

	router.Get("/", handler.GetAllMetricsHandler(metricService))
	router.Get("/ping", handler.PingHandler(application))

	return router
}

func serve(ctx context.Context, srv *http.Server, zapLog *zap.SugaredLogger) error {
	serveErr := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	zapLog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	return nil
}

func closeApp(application *app.App, zapLog *zap.SugaredLogger) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := application.Close(ctx); err != nil {
		zapLog.Errorw("failed to close application", "error", err)
	}
}
