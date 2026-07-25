package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMetricsService struct {
	mock.Mock
}

func (m *MockMetricsService) RecordGauge(name string, value float64) {
	m.Called(name, value)
}

func (m *MockMetricsService) RecordCounter(name string, value int64) {
	m.Called(name, value)
}

func (m *MockMetricsService) GetGauge(name string) (float64, error) {
	args := m.Called(name)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsService) GetCounter(name string) (int64, error) {
	args := m.Called(name)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricsService) GetAll() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestMetricsHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		path               string
		expectedStatus     int
		expectRecordCall   bool
		expectRecordType   string
		expectedMetricName string
		expectedValue      interface{}
	}{
		{
			name:               "valid gauge",
			method:             http.MethodPost,
			path:               "/update/gauge/cpu/75.5",
			expectedStatus:     http.StatusOK,
			expectRecordCall:   true,
			expectRecordType:   "gauge",
			expectedMetricName: "cpu",
			expectedValue:      75.5,
		},
		{
			name:               "valid counter",
			method:             http.MethodPost,
			path:               "/update/counter/requests/42",
			expectedStatus:     http.StatusOK,
			expectRecordCall:   true,
			expectRecordType:   "counter",
			expectedMetricName: "requests",
			expectedValue:      int64(42),
		},
		{
			name:             "method not allowed (GET)",
			method:           http.MethodGet,
			path:             "/update/gauge/cpu/75.5",
			expectedStatus:   http.StatusMethodNotAllowed,
			expectRecordCall: false,
		},
		{
			name:             "missing metric name",
			method:           http.MethodPost,
			path:             "/update/gauge//75.5",
			expectedStatus:   http.StatusNotFound,
			expectRecordCall: false,
		},
		{
			name:             "invalid gauge value",
			method:           http.MethodPost,
			path:             "/update/gauge/cpu/abc",
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "invalid counter value",
			method:           http.MethodPost,
			path:             "/update/counter/requests/12.5",
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "unknown metric type",
			method:           http.MethodPost,
			path:             "/update/histogram/cpu/1.0",
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMetricsService)
			handler := MetricsHandler(mockService)
			mockService.On("RecordCounter", tt.expectedMetricName, tt.expectedValue).Return()
			mockService.On("RecordGauge", tt.expectedMetricName, tt.expectedValue).Return()

			req := httptest.NewRequest(tt.method, tt.path, nil)

			parts := strings.Split(strings.Trim(tt.path, "/"), "/")
			if len(parts) == 4 && parts[0] == "update" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("metricsType", parts[1])
				rctx.URLParams.Add("metricsName", parts[2])
				rctx.URLParams.Add("metricsValue", parts[3])
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectRecordCall {
				if tt.expectRecordType == "gauge" {
					mockService.AssertCalled(t, "RecordGauge", tt.expectedMetricName, tt.expectedValue)
					mockService.AssertNotCalled(t, "RecordCounter")
				} else if tt.expectRecordType == "counter" {
					mockService.AssertCalled(t, "RecordCounter", tt.expectedMetricName, tt.expectedValue)
					mockService.AssertNotCalled(t, "RecordGauge")
				}
			} else {
				mockService.AssertNotCalled(t, "RecordGauge")
				mockService.AssertNotCalled(t, "RecordCounter")
			}
		})
	}
}

func TestGetMetricsHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		mockSetup      func(m *MockMetricsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "get existing gauge",
			path: "/value/gauge/cpu",
			mockSetup: func(m *MockMetricsService) {
				m.On("GetGauge", "cpu").Return(75.5, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "75.5",
		},
		{
			name: "get existing counter",
			path: "/value/counter/requests",
			mockSetup: func(m *MockMetricsService) {
				m.On("GetCounter", "requests").Return(int64(42), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "42",
		},
		{
			name: "gauge not found",
			path: "/value/gauge/missing",
			mockSetup: func(m *MockMetricsService) {
				m.On("GetGauge", "missing").Return(0.0, service.ErrMetricNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name: "counter not found",
			path: "/value/counter/missing",
			mockSetup: func(m *MockMetricsService) {
				m.On("GetCounter", "missing").Return(int64(0), service.ErrMetricNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "unknown metric type",
			path:           "/value/histogram/latency",
			mockSetup:      func(m *MockMetricsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
		{
			name:           "empty metric name",
			path:           "/value/gauge/",
			mockSetup:      func(m *MockMetricsService) {},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMetricsService)
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			handler := GetMetricsHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			parts := strings.Split(strings.Trim(tt.path, "/"), "/")
			if len(parts) >= 3 && parts[0] == "value" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("metricsType", parts[1])
				if len(parts) > 2 {
					rctx.URLParams.Add("metricsName", parts[2])
				}
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			} else {
				assert.Empty(t, w.Body.String())
			}

			mockService.AssertExpectations(t)
		})
	}
}
