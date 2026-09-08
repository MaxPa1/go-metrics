package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/config"
	models "github.com/MaxPa1/go-metrics/internal/model"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricAgent(t *testing.T) {
	agent := NewMetricAgent(&config.AgentConfig{Address: "localhost:8080"})

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.gauges, "gauges must not be nil")
	assert.Empty(t, agent.gauges)
	assert.Equal(t, int64(0), agent.pollCount)
	assert.Equal(t, float64(0), agent.randomValue)
	assert.Equal(t, "http://localhost:8080/updates/", agent.updatesURL)
}

func TestMetricAgent_SendMetrics(t *testing.T) {
	type requestData struct {
		path            string
		contentType     string
		contentEncoding string
		batch           []models.Metrics
	}

	var requests []requestData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		require.NoError(t, err, "failed to read request body")

		var batch []models.Metrics
		err = json.Unmarshal(body, &batch)
		require.NoError(t, err, "failed to unmarshal JSON")

		requests = append(requests, requestData{
			path:            r.URL.Path,
			contentType:     r.Header.Get("Content-Type"),
			contentEncoding: r.Header.Get("Content-Encoding"),
			batch:           batch,
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
				updatesURL:  server.URL + "/updates/",
			}
			m.SendMetrics(context.Background(), restyClient)

			require.Len(t, requests, 1)
			req := requests[0]
			assert.Equal(t, "/updates/", req.path)
			assert.Equal(t, "application/json", req.contentType)
			assert.Equal(t, "gzip", req.contentEncoding)

			expectedCount := len(tt.gauges) + 2
			require.Len(t, req.batch, expectedCount)

			metricsByID := make(map[string]models.Metrics)
			for _, metric := range req.batch {
				metricsByID[metric.ID] = metric
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

func TestSendBatch(t *testing.T) {
	var (
		receivedPath            string
		receivedContentType     string
		receivedContentEncoding string
		receivedBatch           []models.Metrics
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedContentType = r.Header.Get("Content-Type")
		receivedContentEncoding = r.Header.Get("Content-Encoding")
		body, err := readBody(r)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedBatch); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restyClient := resty.NewWithClient(server.Client())

	delta := int64(355)
	value := 333.6
	batch := []models.Metrics{
		{ID: "PollCount", MType: "counter", Delta: &delta},
		{ID: "RandomValue", MType: "gauge", Value: &value},
	}

	err := sendBatch(context.Background(), restyClient, server.URL+"/updates/", batch)
	require.NoError(t, err)

	assert.Equal(t, "/updates/", receivedPath)
	assert.Equal(t, "application/json", receivedContentType)
	assert.Equal(t, "gzip", receivedContentEncoding)
	require.Len(t, receivedBatch, 2)
	assert.ElementsMatch(t, []string{"PollCount", "RandomValue"}, []string{receivedBatch[0].ID, receivedBatch[1].ID})
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

func readBody(r *http.Request) ([]byte, error) {
	var reader io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		reader = gz
	}
	return io.ReadAll(reader)
}
