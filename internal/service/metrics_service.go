// Package service содержит бизнес-логику работы с метриками.
package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/MaxPa1/go-metrics/internal/model"
)

// ErrMetricNotFound возвращается, если метрика не найдена.
var ErrMetricNotFound = errors.New("metric not found")

// MetricsStorage описывает хранилище метрик.
type MetricsStorage interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	FindGauge(ctx context.Context, name string) (float64, bool, error)
	UpdateCounter(ctx context.Context, name string, value int64) error
	FindCounter(ctx context.Context, name string) (int64, bool, error)
	FindAll(ctx context.Context) (map[string]int64, map[string]float64, error)
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

// MetricsServiceImpl реализует бизнес-логику поверх MetricsStorage.
type MetricsServiceImpl struct {
	repository MetricsStorage
}

// NewMetricsService создаёт сервис поверх хранилища.
func NewMetricsService(repository MetricsStorage) *MetricsServiceImpl {
	return &MetricsServiceImpl{
		repository: repository,
	}
}

// RecordGauge сохраняет значение gauge.
func (s *MetricsServiceImpl) RecordGauge(ctx context.Context, name string, value float64) error {
	err := s.repository.UpdateGauge(ctx, name, value)
	if err != nil {
		return fmt.Errorf("record gauge: %w", err)
	}
	return nil
}

// RecordCounter прибавляет value к counter.
func (s *MetricsServiceImpl) RecordCounter(ctx context.Context, name string, value int64) error {
	err := s.repository.UpdateCounter(ctx, name, value)
	if err != nil {
		return fmt.Errorf("record counter: %w", err)
	}
	return nil
}

// GetGauge возвращает значение gauge или ErrMetricNotFound.
func (s *MetricsServiceImpl) GetGauge(ctx context.Context, name string) (float64, error) {
	value, ok, err := s.repository.FindGauge(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("get gauge %q: %w", name, err)
	}
	if !ok {
		return 0, fmt.Errorf("get gauge %q: %w", name, ErrMetricNotFound)
	}
	return value, nil
}

// GetCounter возвращает значение counter или ErrMetricNotFound.
func (s *MetricsServiceImpl) GetCounter(ctx context.Context, name string) (int64, error) {
	value, ok, err := s.repository.FindCounter(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("get counter: %w", err)
	}
	if !ok {
		return 0, fmt.Errorf("get counter: %w", ErrMetricNotFound)
	}
	return value, nil
}

// RecordBatch сохраняет набор метрик.
func (s *MetricsServiceImpl) RecordBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	if err := s.repository.UpdateBatch(ctx, metrics); err != nil {
		return fmt.Errorf("record batch: %w", err)
	}
	return nil
}

// GetAll возвращает все метрики в виде строк «имя: значение».
func (s *MetricsServiceImpl) GetAll(ctx context.Context) ([]string, error) {
	counters, gauges, err := s.repository.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all: %w", err)
	}

	list := make([]string, 0, len(counters)+len(gauges))

	var buf [128]byte
	for k, v := range counters {
		b := append(append(buf[:0], k...), ": "...)
		list = append(list, string(strconv.AppendInt(b, v, 10)))
	}
	for k, v := range gauges {
		b := append(append(buf[:0], k...), ": "...)
		list = append(list, string(strconv.AppendFloat(b, v, 'f', -1, 64)))
	}
	return list, nil
}
