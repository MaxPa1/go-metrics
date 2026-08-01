package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/mocks"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

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
			mockService := mocks.NewMetricsService(t)

			if tt.expectRecordCall {
				if tt.expectRecordType == "gauge" {
					mockService.EXPECT().RecordGauge(tt.expectedMetricName, tt.expectedValue).Return()
				} else if tt.expectRecordType == "counter" {
					mockService.EXPECT().RecordCounter(tt.expectedMetricName, tt.expectedValue).Return()
				}
			}

			r := chi.NewRouter()
			r.Post("/update/{metricsType}/{metricsName}/{metricsValue}",
				MetricsHandler(mockService))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetMetricsHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		mockSetup      func(m *mocks.MetricsService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "get existing gauge",
			path: "/value/gauge/cpu",
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetGauge("cpu").Return(75.5, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "75.5",
		},
		{
			name: "get existing counter",
			path: "/value/counter/requests",
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetCounter("requests").Return(int64(42), nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "42",
		},
		{
			name: "gauge not found",
			path: "/value/gauge/missing",
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetGauge("missing").Return(0.0, service.ErrMetricNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name: "counter not found",
			path: "/value/counter/missing",
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetCounter("missing").Return(int64(0), service.ErrMetricNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "unknown metric type",
			path:           "/value/histogram/latency",
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
		{
			name:           "empty metric name",
			path:           "/value/gauge/",
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMetricsService(t)
			tt.mockSetup(mockService)

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
		})
	}
}

func TestGetAllMetricsHandler(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(m *mocks.MetricsService)
		expectedStatus int
		contentType    string
		checkBody      func(t *testing.T, body string)
	}{
		{
			name: "get all metrics OK",
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetAll().Return([]string{"counter: 36", "gauge: 0.75"})
			},
			expectedStatus: http.StatusOK,
			contentType:    "text/html; charset=utf-8",
			checkBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "<li>counter: 36</li>")
				assert.Contains(t, body, "<li>gauge: 0.75</li>")
				assert.Contains(t, body, "<ul>")
				assert.Contains(t, body, "</ul>")
			},
		},
		{
			name: "get empty metrics OK",
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetAll().Return([]string{})
			},
			expectedStatus: http.StatusOK,
			contentType:    "text/html; charset=utf-8",
			checkBody: func(t *testing.T, body string) {
				assert.NotContains(t, body, "<li>")
				assert.Contains(t, body, "<ul>")
				assert.Contains(t, body, "</ul>")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMetricsService(t)
			tt.mockSetup(mockService)

			handler := GetAllMetricsHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/", nil)

			w := httptest.NewRecorder()
			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}
