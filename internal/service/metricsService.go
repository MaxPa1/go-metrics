package service

type MetricsStorage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
}

type MetricsService interface {
	RecordGauge(name string, value float64)
	RecordCounter(name string, value int64)
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
