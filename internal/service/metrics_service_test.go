package service

import (
	"testing"

	"github.com/MaxPa1/go-metrics/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestMetricsService_RecordGauge(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().UpdateGauge("cpu", 75.5).Return()

	service.RecordGauge("cpu", 75.5)
}

func TestMetricsService_RecordCounter(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().UpdateCounter("call", int64(3)).Return()

	service.RecordCounter("call", int64(3))
}

func TestMetricsService_GetCounter(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().FindCounter("counter").Return(int64(3), true)
	value1, err1 := service.GetCounter("counter")
	assert.NoError(t, err1)
	assert.Equal(t, int64(3), value1)

	mockRepo.EXPECT().FindCounter("requestCount").Return(int64(0), false)
	value2, err2 := service.GetCounter("requestCount")
	assert.ErrorIs(t, err2, ErrMetricNotFound)
	assert.Equal(t, int64(0), value2)
}

func TestMetricsService_GetGauge(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().FindGauge("temperature").Return(36.6, true)
	val1, err1 := service.GetGauge("temperature")
	assert.NoError(t, err1)
	assert.Equal(t, 36.6, val1)

	mockRepo.EXPECT().FindGauge("unknown").Return(0.0, false)
	val2, err2 := service.GetGauge("unknown")
	assert.ErrorIs(t, err2, ErrMetricNotFound)
	assert.Equal(t, 0.0, val2)
}
