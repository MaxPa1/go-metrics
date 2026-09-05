package service

import (
	"context"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/mocks"
	models "github.com/MaxPa1/go-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMetricsService_RecordGauge(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().
		UpdateGauge(mock.Anything, "cpu", 75.5).
		Return(nil)

	err := service.RecordGauge(context.Background(), "cpu", 75.5)
	assert.NoError(t, err)
}

func TestMetricsService_RecordCounter(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().
		UpdateCounter(mock.Anything, "call", int64(3)).
		Return(nil)

	err := service.RecordCounter(context.Background(), "call", int64(3))
	assert.NoError(t, err)
}

func TestMetricsService_GetCounter(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().
		FindCounter(mock.Anything, "counter").
		Return(int64(3), true, nil)
	value1, err1 := service.GetCounter(context.Background(), "counter")
	assert.NoError(t, err1)
	assert.Equal(t, int64(3), value1)

	mockRepo.EXPECT().
		FindCounter(mock.Anything, "requestCount").
		Return(int64(0), false, nil)
	value2, err2 := service.GetCounter(context.Background(), "requestCount")
	assert.ErrorIs(t, err2, ErrMetricNotFound)
	assert.Equal(t, int64(0), value2)
}

func TestMetricsService_RecordBatch(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	gaugeValue := 75.5
	counterDelta := int64(3)
	batch := []models.Metrics{
		{ID: "cpu", MType: models.Gauge, Value: &gaugeValue},
		{ID: "call", MType: models.Counter, Delta: &counterDelta},
	}

	mockRepo.EXPECT().
		UpdateBatch(mock.Anything, batch).
		Return(nil)

	err := service.RecordBatch(context.Background(), batch)
	assert.NoError(t, err)
}

func TestMetricsService_GetGauge(t *testing.T) {
	mockRepo := mocks.NewMetricsStorage(t)
	service := NewMetricsService(mockRepo)

	mockRepo.EXPECT().
		FindGauge(mock.Anything, "temperature").
		Return(36.6, true, nil)
	val1, err1 := service.GetGauge(context.Background(), "temperature")
	assert.NoError(t, err1)
	assert.Equal(t, 36.6, val1)

	mockRepo.EXPECT().
		FindGauge(mock.Anything, "unknown").
		Return(0.0, false, nil)
	val2, err2 := service.GetGauge(context.Background(), "unknown")
	assert.ErrorIs(t, err2, ErrMetricNotFound)
	assert.Equal(t, 0.0, val2)
}
