package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/mocks"
	models "github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMetricsV2Handler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		body               string
		expectedStatus     int
		expectRecordCall   bool
		expectRecordType   string
		expectedMetricName string
		expectedValue      interface{}
	}{
		{
			name:               "valid gauge",
			method:             http.MethodPost,
			body:               `{"id":"cpu","type":"gauge","value":75.5}`,
			expectedStatus:     http.StatusOK,
			expectRecordCall:   true,
			expectRecordType:   "gauge",
			expectedMetricName: "cpu",
			expectedValue:      75.5,
		},
		{
			name:               "valid counter",
			method:             http.MethodPost,
			body:               `{"id":"requests","type":"counter","delta":42}`,
			expectedStatus:     http.StatusOK,
			expectRecordCall:   true,
			expectRecordType:   "counter",
			expectedMetricName: "requests",
			expectedValue:      int64(42),
		},
		{
			name:             "method not allowed (GET)",
			method:           http.MethodGet,
			body:             `{"id":"cpu","type":"gauge","value":75.5}`,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectRecordCall: false,
		},
		{
			name:             "invalid json",
			method:           http.MethodPost,
			body:             `{"id":"cpu","type":"gauge","value":}`,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "empty body",
			method:           http.MethodPost,
			body:             ``,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "gauge without value",
			method:           http.MethodPost,
			body:             `{"id":"cpu","type":"gauge"}`,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "counter without delta",
			method:           http.MethodPost,
			body:             `{"id":"requests","type":"counter"}`,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "gauge with wrong value type",
			method:           http.MethodPost,
			body:             `{"id":"cpu","type":"gauge","value":"abc"}`,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "counter with float delta",
			method:           http.MethodPost,
			body:             `{"id":"requests","type":"counter","delta":12.5}`,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
		{
			name:             "unknown metric type",
			method:           http.MethodPost,
			body:             `{"id":"cpu","type":"histogram","value":1.0}`,
			expectedStatus:   http.StatusBadRequest,
			expectRecordCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMetricsService(t)

			if tt.expectRecordCall {
				switch tt.expectRecordType {
				case "gauge":
					mockService.EXPECT().RecordGauge(tt.expectedMetricName, tt.expectedValue).Return()
				case "counter":
					mockService.EXPECT().RecordCounter(tt.expectedMetricName, tt.expectedValue).Return()
				}
			}

			r := chi.NewRouter()
			r.Post("/update/", MetricsV2Handler(mockService))

			req := httptest.NewRequest(tt.method, "/update/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetMetricsV2Handler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *mocks.MetricsService)
		expectedStatus int
		checkBody      bool
		expectedID     string
		expectedType   string
		expectedValue  float64
		expectedDelta  int64
	}{
		{
			name: "get existing gauge",
			body: `{"id":"cpu","type":"gauge"}`,
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetGauge("cpu").Return(75.5, nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      true,
			expectedID:     "cpu",
			expectedType:   models.Gauge,
			expectedValue:  75.5,
		},
		{
			name: "get existing counter",
			body: `{"id":"requests","type":"counter"}`,
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetCounter("requests").Return(int64(42), nil)
			},
			expectedStatus: http.StatusOK,
			checkBody:      true,
			expectedID:     "requests",
			expectedType:   models.Counter,
			expectedDelta:  42,
		},
		{
			name: "gauge not found",
			body: `{"id":"missing","type":"gauge"}`,
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetGauge("missing").Return(0.0, service.ErrMetricNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "counter not found",
			body: `{"id":"missing","type":"counter"}`,
			mockSetup: func(m *mocks.MetricsService) {
				m.EXPECT().GetCounter("missing").Return(int64(0), service.ErrMetricNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unknown metric type",
			body:           `{"id":"latency","type":"histogram"}`,
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty metric name",
			body:           `{"id":"","type":"gauge"}`,
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing id field",
			body:           `{"type":"gauge"}`,
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid json",
			body:           `{"id":"cpu","type":}`,
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           ``,
			mockSetup:      func(m *mocks.MetricsService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMetricsService(t)
			tt.mockSetup(mockService)

			handler := GetMetricsV2Handler(mockService)

			req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.checkBody {
				return
			}

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var got models.Metrics
			require.NoError(t, json.NewDecoder(w.Body).Decode(&got))

			assert.Equal(t, tt.expectedID, got.ID)
			assert.Equal(t, tt.expectedType, got.MType)

			switch tt.expectedType {
			case models.Gauge:
				require.NotNil(t, got.Value)
				assert.Equal(t, tt.expectedValue, *got.Value)
				assert.Nil(t, got.Delta)
			case models.Counter:
				require.NotNil(t, got.Delta)
				assert.Equal(t, tt.expectedDelta, *got.Delta)
				assert.Nil(t, got.Value)
			}
		})
	}
}
