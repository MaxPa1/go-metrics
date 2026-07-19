package service

import (
	"github.com/MaxPa1/go-metrics/internal/repository"
)

type MetricsService interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, value int64) error
}

type MetricsServiceImpl struct {
	repository repository.MetricsStorage
}

func NewMetricsService(repository repository.MetricsStorage) *MetricsServiceImpl {
	return &MetricsServiceImpl{
		repository: repository,
	}
}

func (s *MetricsServiceImpl) UpdateGauge(name string, value float64) error {
	s.repository.UpdateGauge(name, value)
	return nil
}

func (s *MetricsServiceImpl) UpdateCounter(name string, value int64) error {
	s.repository.UpdateCounter(name, value)
	return nil
}
