package service

import (
	"errors"
	"fmt"
)

var ErrMetricNotFound = errors.New("metric not found")

type MetricsStorage interface {
	UpdateGauge(name string, value float64)
	FindGauge(name string) (float64, bool)
	UpdateCounter(name string, value int64)
	FindCounter(name string) (int64, bool)
	FindAll() []string
}

type MetricsServiceImpl struct {
	repository MetricsStorage
}

func NewMetricsService(repository MetricsStorage) *MetricsServiceImpl {
	return &MetricsServiceImpl{
		repository: repository,
	}
}

func (s *MetricsServiceImpl) RecordGauge(name string, value float64) {
	s.repository.UpdateGauge(name, value)
}

func (s *MetricsServiceImpl) RecordCounter(name string, value int64) {
	s.repository.UpdateCounter(name, value)
}

func (s *MetricsServiceImpl) GetGauge(name string) (float64, error) {
	value, ok := s.repository.FindGauge(name)
	if !ok {
		return 0.0, fmt.Errorf("gauge %q: %w", name, ErrMetricNotFound)
	}
	return value, nil
}

func (s *MetricsServiceImpl) GetCounter(name string) (int64, error) {
	value, ok := s.repository.FindCounter(name)
	if !ok {
		return 0, fmt.Errorf("counter %q: %w", name, ErrMetricNotFound)
	}
	return value, nil
}

func (s *MetricsServiceImpl) GetAll() []string {
	return s.repository.FindAll()
}
