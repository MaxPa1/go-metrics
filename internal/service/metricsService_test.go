package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func (m *MockStorage) FindGauge(name string) (float64, bool) {
	args := m.Called(name)
	return args.Get(0).(float64), args.Bool(1)
}

func (m *MockStorage) FindCounter(name string) (int64, bool) {
	args := m.Called(name)
	return args.Get(0).(int64), args.Bool(1)
}

func (m *MockStorage) FindAll() []string {
	args := m.Called()
	return args.Get(0).([]string)
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

func TestMetricsService_GetCounter(t *testing.T) {
	mockRepo := new(MockStorage)
	service := NewMetricsService(mockRepo)

	mockRepo.On("FindCounter", "counter").Return(int64(3), true)
	value1, err1 := service.GetCounter("counter")
	assert.NoError(t, err1)
	assert.Equal(t, int64(3), value1)
	mockRepo.AssertCalled(t, "FindCounter", "counter")

	mockRepo.On("FindCounter", "requestCount").Return(int64(0), false)
	value2, err2 := service.GetCounter("requestCount")
	assert.ErrorIs(t, err2, ErrMetricNotFound)
	assert.Equal(t, int64(0), value2)
	mockRepo.AssertCalled(t, "FindCounter", "requestCount")

	mockRepo.AssertExpectations(t)
}

func TestMetricsService_GetGauge(t *testing.T) {
	mockRepo := new(MockStorage)
	service := NewMetricsService(mockRepo)

	mockRepo.On("FindGauge", "temperature").Return(36.6, true)
	val1, err1 := service.GetGauge("temperature")
	assert.NoError(t, err1)
	assert.Equal(t, 36.6, val1)
	mockRepo.AssertCalled(t, "FindGauge", "temperature")

	mockRepo.On("FindGauge", "unknown").Return(0.0, false)
	val2, err2 := service.GetGauge("unknown")
	assert.ErrorIs(t, err2, ErrMetricNotFound)
	assert.Equal(t, 0.0, val2)
	mockRepo.AssertCalled(t, "FindGauge", "unknown")

	mockRepo.AssertExpectations(t)
}
