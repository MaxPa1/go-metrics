package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrMetricNotFound = errors.New("metric not found")

type MetricsStorage interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	FindGauge(ctx context.Context, name string) (float64, bool, error)
	UpdateCounter(ctx context.Context, name string, value int64) error
	FindCounter(ctx context.Context, name string) (int64, bool, error)
	FindAll(ctx context.Context) (map[string]int64, map[string]float64, error)
}

type MetricsServiceImpl struct {
	repository MetricsStorage
}

func NewMetricsService(repository MetricsStorage) *MetricsServiceImpl {
	return &MetricsServiceImpl{
		repository: repository,
	}
}

func (s *MetricsServiceImpl) RecordGauge(ctx context.Context, name string, value float64) error {
	err := s.repository.UpdateGauge(ctx, name, value)
	if err != nil {
		return fmt.Errorf("record gauge: %w", err)
	}
	return nil
}

func (s *MetricsServiceImpl) RecordCounter(ctx context.Context, name string, value int64) error {
	err := s.repository.UpdateCounter(ctx, name, value)
	if err != nil {
		return fmt.Errorf("record counter: %w", err)
	}
	return nil
}

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

func (s *MetricsServiceImpl) GetAll(ctx context.Context) ([]string, error) {
	counters, gauges, err := s.repository.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all: %w", err)
	}

	list := make([]string, 0, len(counters)+len(gauges))

	var sb strings.Builder

	for k, v := range counters {
		sb.Reset()
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.Write(strconv.AppendInt(nil, v, 10))
		list = append(list, sb.String())
	}
	for k, v := range gauges {
		sb.Reset()
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.Write(strconv.AppendFloat(nil, v, 'f', -1, 64))
		list = append(list, sb.String())
	}
	return list, nil
}
