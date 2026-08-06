package agent

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricAgent(t *testing.T) {
	agent := NewMetricAgent(&Config{Address: "localhost:8080"})

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.gauges, "gauges must not be nil")
	assert.Empty(t, agent.gauges)
	assert.Equal(t, int64(0), agent.pollCount)
	assert.Equal(t, float64(0), agent.randomValue)
	assert.Equal(t, "http://localhost:8080/update/{metricsType}/{metricsName}/{metricsValue}", agent.baseURL)
}

func TestMetricAgent_SendMetrics(t *testing.T) {
	var receivedPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))
		receivedPaths = append(receivedPaths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restyClient := resty.NewWithClient(server.Client())

	type fields struct {
		pollCount   int64
		randomValue float64
		gauges      map[string]float64
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"1",
			fields{3, 52.2, map[string]float64{"cpu": 72.2, "memory": 23.1}},
		},
		{
			"2",
			fields{7, 2.2, map[string]float64{"SYS": 7.1, "LastGC": 11.1, "HeapInuse": 41.2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedPaths = nil

			m := &MetricAgent{
				pollCount:   tt.fields.pollCount,
				randomValue: tt.fields.randomValue,
				gauges:      tt.fields.gauges,
				baseURL:     server.URL + "/update/{metricsType}/{metricsName}/{metricsValue}",
			}
			m.SendMetrics(restyClient)

			expectedCount := len(m.gauges) + 2
			assert.Len(t, receivedPaths, expectedCount, "should send all metrics")

			for name, value := range tt.fields.gauges {
				expectedPath := fmt.Sprintf("/update/gauge/%s/%v", name, value)
				assert.Contains(t, receivedPaths, expectedPath, "missing gauge metric %s", name)
			}

			expectedRandomPath := fmt.Sprintf("/update/gauge/randomValue/%v", tt.fields.randomValue)
			assert.Contains(t, receivedPaths, expectedRandomPath)

			expectedPollPath := fmt.Sprintf("/update/counter/pollCount/%d", tt.fields.pollCount)
			assert.Contains(t, receivedPaths, expectedPollPath)
		})
	}
}

func TestSendMetric(t *testing.T) {
	var receivedPaths string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))
		assert.Empty(t, body)
		receivedPaths = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restyClient := resty.NewWithClient(server.Client())

	type args struct {
		mType string
		name  string
		value string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"1",
			args{"counter", "pollCount", "355"},
		},
		{
			"2",
			args{"gauge", "randomValue", "333.6"},
		},
		{
			"3",
			args{"gauge", "Cpu", "0.75"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricAgent{
				baseURL: server.URL + "/update/{metricsType}/{metricsName}/{metricsValue}",
			}
			sendMetric(restyClient, m.baseURL, tt.args.mType, tt.args.name, tt.args.value)

			expectedPath := fmt.Sprintf("/update/%s/%s/%s", tt.args.mType, tt.args.name, tt.args.value)
			assert.Equal(t, expectedPath, receivedPaths, "should send metric")
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
