package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			path:             "/update/gauge//75.5", // пустое имя
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
				req.SetPathValue("metricsType", parts[1])
				req.SetPathValue("metricsName", parts[2])
				req.SetPathValue("metricsValue", parts[3])
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
