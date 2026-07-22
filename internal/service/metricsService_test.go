package service

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) UpdateGauge(name string, value float64) {
	m.Called(name, value)
}

func (m *MockStorage) UpdateCounter(name string, value int64) {
	m.Called(name, value)
}

func TestMetricsService_RecordGauge(t *testing.T) {
	mockRepo := new(MockStorage)
	service := NewMetricsService(mockRepo)

	mockRepo.On("UpdateGauge", "cpu", 75.5).Return()

	service.RecordGauge("cpu", 75.5)

	mockRepo.AssertCalled(t, "UpdateGauge", "cpu", 75.5)
}

func TestMetricsService_RecordCounter(t *testing.T) {
	mockRepo := new(MockStorage)
	service := NewMetricsService(mockRepo)

	mockRepo.On("UpdateCounter", "call", int64(3)).Return()

	service.RecordCounter("call", int64(3))

	mockRepo.AssertCalled(t, "UpdateCounter", "call", int64(3))
}
