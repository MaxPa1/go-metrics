package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/MaxPa1/go-metrics/internal/model"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricAgent(t *testing.T) {
	agent := NewMetricAgent(&Config{Address: "localhost:8080"})

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.gauges, "gauges must not be nil")
	assert.Empty(t, agent.gauges)
	assert.Equal(t, int64(0), agent.pollCount)
	assert.Equal(t, float64(0), agent.randomValue)
	assert.Equal(t, "http://localhost:8080/update", agent.baseURL)
}

func TestMetricAgent_SendMetrics(t *testing.T) {
	type requestData struct {
		path        string
		contentType string
		metric      models.Metrics
	}

	var requests []requestData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "failed to read request body")

		var m models.Metrics
		err = json.Unmarshal(body, &m)
		require.NoError(t, err, "failed to unmarshal JSON")

		requests = append(requests, requestData{
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			metric:      m,
		})
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restyClient := resty.NewWithClient(server.Client())

	tests := []struct {
		name        string
		pollCount   int64
		randomValue float64
		gauges      map[string]float64
	}{
		{
			name:        "1",
			pollCount:   3,
			randomValue: 52.2,
			gauges: map[string]float64{
				"cpu":    72.2,
				"memory": 23.1,
			},
		},
		{
			name:        "2",
			pollCount:   7,
			randomValue: 2.2,
			gauges: map[string]float64{
				"SYS":       7.1,
				"LastGC":    11.1,
				"HeapInuse": 41.2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests = nil

			m := &MetricAgent{
				pollCount:   tt.pollCount,
				randomValue: tt.randomValue,
				gauges:      tt.gauges,
				baseURL:     server.URL + "/update",
			}
			m.SendMetrics(restyClient)

			expectedCount := len(tt.gauges) + 2
			require.Len(t, requests, expectedCount)

			metricsByID := make(map[string]models.Metrics)
			for _, req := range requests {
				assert.Equal(t, "/update", req.path)
				assert.Equal(t, "application/json", req.contentType)
				metricsByID[req.metric.ID] = req.metric
			}

			for name, value := range tt.gauges {
				metric, ok := metricsByID[name]
				require.Truef(t, ok, "missing gauge metric %s", name)
				assert.Equal(t, "gauge", metric.MType, "metric %s should be gauge", name)
				require.NotNil(t, metric.Value, "gauge %s should have Value", name)
				assert.InDelta(t, value, *metric.Value, 0.0001, "value mismatch for gauge %s", name)
				assert.Nil(t, metric.Delta, "gauge %s should not have Delta", name)
			}

			randomMetric, ok := metricsByID["RandomValue"]
			require.True(t, ok, "missing gauge RandomValue")
			assert.Equal(t, "gauge", randomMetric.MType)
			require.NotNil(t, randomMetric.Value)
			assert.InDelta(t, tt.randomValue, *randomMetric.Value, 0.0001)

			pollMetric, ok := metricsByID["PollCount"]
			require.True(t, ok, "missing counter PollCount")
			assert.Equal(t, "counter", pollMetric.MType)
			require.NotNil(t, pollMetric.Delta)
			assert.Equal(t, tt.pollCount, *pollMetric.Delta)
			assert.Nil(t, pollMetric.Value)

			assert.Equal(t, int64(0), m.pollCount)
		})
	}
}

func TestSendMetric(t *testing.T) {
	var (
		receivedPath    string
		receivedBody    models.Metrics
		receivedContent string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedContent = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedBody); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restyClient := resty.NewWithClient(server.Client())

	tests := []struct {
		name        string
		mType       string
		metricID    string
		delta       int64
		value       float64
		expectDelta bool
		expectValue bool
	}{
		{
			name:        "counter",
			mType:       "counter",
			metricID:    "PollCount",
			delta:       355,
			value:       0,
			expectDelta: true,
			expectValue: false,
		},
		{
			name:        "gauge",
			mType:       "gauge",
			metricID:    "RandomValue",
			delta:       0,
			value:       333.6,
			expectDelta: false,
			expectValue: true,
		},
		{
			name:        "gauge2",
			mType:       "gauge",
			metricID:    "Cpu",
			delta:       0,
			value:       0.75,
			expectDelta: false,
			expectValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedPath = ""
			receivedContent = ""
			receivedBody = models.Metrics{}

			m := &MetricAgent{
				baseURL: server.URL + "/update",
			}

			err := sendMetric(restyClient, m.baseURL, tt.mType, tt.metricID, tt.delta, tt.value)
			require.NoError(t, err)

			assert.Equal(t, "/update", receivedPath)
			assert.Equal(t, "application/json", receivedContent)
			assert.Equal(t, tt.metricID, receivedBody.ID)
			assert.Equal(t, tt.mType, receivedBody.MType)

			if tt.expectDelta {
				require.NotNil(t, receivedBody.Delta)
				assert.Equal(t, tt.delta, *receivedBody.Delta)
				assert.Nil(t, receivedBody.Value)
			} else {
				assert.Nil(t, receivedBody.Delta)
				require.NotNil(t, receivedBody.Value)
				assert.InDelta(t, tt.value, *receivedBody.Value, 0.0001)
			}
		})
	}
}

func TestMetricAgent_UpdateMetrics(t *testing.T) {
	m := &MetricAgent{
		gauges: make(map[string]float64),
	}

	initialPollCount := m.pollCount
	expectedKeys := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc",
	}

	m.UpdateMetrics()

	assert.Len(t, m.gauges, len(expectedKeys))
	assert.Equal(t, initialPollCount+1, m.pollCount)
	assert.True(t, m.randomValue >= 0.0 && m.randomValue < 1.0)
	for _, key := range expectedKeys {
		assert.Contains(t, m.gauges, key, "gauges should contain key %s", key)
	}

	m.UpdateMetrics()
	assert.Equal(t, initialPollCount+2, m.pollCount, "pollCount should increment again")
}
