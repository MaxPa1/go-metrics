package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/signature"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricAgent(t *testing.T) {
	cfg := &config.AgentConfig{
		Address:        "localhost:8080",
		Key:            "secret",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		RateLimit:      3,
	}
	agent := NewMetricAgent(cfg, resty.New())

	require.NotNil(t, agent)
	assert.Equal(t, "secret", agent.key)
	assert.Equal(t, "http://localhost:8080/updates/", agent.updatesURL)
	assert.Equal(t, 2*time.Second, agent.pollInterval)
	assert.Equal(t, 10*time.Second, agent.reportInterval)
	assert.Equal(t, 3, agent.rateLimit)
	require.NotNil(t, agent.store)
}

func TestMetricAgent_Run(t *testing.T) {
	var (
		mu      sync.Mutex
		batches [][]models.Metrics
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		assert.NoError(t, err)
		var batch []models.Metrics
		assert.NoError(t, json.Unmarshal(body, &batch))

		mu.Lock()
		batches = append(batches, batch)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewMetricAgent(&config.AgentConfig{
		Address:        strings.TrimPrefix(server.URL, "http://"),
		PollInterval:   10 * time.Millisecond,
		ReportInterval: 50 * time.Millisecond,
		RateLimit:      2,
	}, resty.NewWithClient(server.Client()))

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		m.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, batches)

	var totalPollCount int64
	ids := make(map[string]struct{})
	for _, batch := range batches {
		for _, metric := range batch {
			ids[metric.ID] = struct{}{}
			if metric.ID == "PollCount" {
				require.NotNil(t, metric.Delta)
				totalPollCount += *metric.Delta
			}
		}
	}
	for _, id := range []string{"Alloc", "RandomValue", "PollCount", "TotalMemory", "FreeMemory", "CPUutilization1"} {
		assert.Contains(t, ids, id)
	}
	assert.Positive(t, totalPollCount)
}

func TestMetricAgent_Send_RestoresPollCountOnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	m := &MetricAgent{
		client:     resty.NewWithClient(server.Client()),
		updatesURL: server.URL + "/updates/",
		store:      newStore(),
	}
	m.store.addPollCount(5)

	_, pollCount := m.store.read()
	m.store.addPollCount(-pollCount)
	m.send(context.Background(), report{metrics: buildBatch(nil, pollCount), pollCount: pollCount})

	_, pollCount = m.store.read()
	assert.Equal(t, int64(5), pollCount, "unsent PollCount must be returned to the store")
}

func TestStore_Read(t *testing.T) {
	s := newStore()
	s.setGauges(map[string]float64{"Alloc": 1, "HeapSys": 2})
	s.setGauges(map[string]float64{"TotalMemory": 3, "Alloc": 4})
	s.addPollCount(1)
	s.addPollCount(2)

	gauges, pollCount := s.read()

	assert.Equal(t, int64(3), pollCount)
	require.Len(t, gauges, 3)
	assert.InDelta(t, 4, gauges["Alloc"], 0.0001, "later write must win")
	assert.InDelta(t, 2, gauges["HeapSys"], 0.0001, "gauges from other collectors must be kept")
	assert.InDelta(t, 3, gauges["TotalMemory"], 0.0001)

	_, pollCount = s.read()
	assert.Equal(t, int64(3), pollCount, "read must not reset pollCount as a side effect")

	gauges["Injected"] = 99
	freshGauges, _ := s.read()
	assert.NotContains(t, freshGauges, "Injected", "read must return a copy, not the internal map")
}

func TestBuildBatch(t *testing.T) {
	batch := buildBatch(map[string]float64{"Alloc": 4, "HeapSys": 2}, 3)

	byID := make(map[string]models.Metrics)
	for _, metric := range batch {
		byID[metric.ID] = metric
	}
	require.Len(t, byID, 3)
	assert.Equal(t, models.Gauge, byID["Alloc"].MType)
	assert.InDelta(t, 4, *byID["Alloc"].Value, 0.0001)
	assert.InDelta(t, 2, *byID["HeapSys"].Value, 0.0001)
	assert.Equal(t, models.Counter, byID["PollCount"].MType)
	assert.Equal(t, int64(3), *byID["PollCount"].Delta)
}

func TestRunWorkers_LimitsConcurrency(t *testing.T) {
	const (
		limit = 3
		total = 20
	)

	var inFlight, maxInFlight, processed atomic.Int64
	jobs := make(chan report)
	go func() {
		defer close(jobs)
		for range total {
			jobs <- report{}
		}
	}()

	runWorkers(context.Background(), limit, jobs, func(context.Context, report) {
		cur := inFlight.Add(1)
		for {
			prev := maxInFlight.Load()
			if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		inFlight.Add(-1)
		processed.Add(1)
	})

	assert.Equal(t, int64(total), processed.Load())
	assert.LessOrEqual(t, maxInFlight.Load(), int64(limit))
}

func TestCollectRuntimeMetrics(t *testing.T) {
	expectedKeys := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	gauges := collectRuntimeMetrics()

	assert.Len(t, gauges, len(expectedKeys))
	for _, key := range expectedKeys {
		assert.Contains(t, gauges, key)
	}
	assert.True(t, gauges["RandomValue"] >= 0 && gauges["RandomValue"] < 1)
}

func TestCollectSystemMetrics(t *testing.T) {
	gauges, err := collectSystemMetrics(context.Background())
	require.NoError(t, err)

	assert.Positive(t, gauges["TotalMemory"])
	assert.Contains(t, gauges, "FreeMemory")

	cpuCount := len(gauges) - 2
	require.Positive(t, cpuCount)
	for i := 1; i <= cpuCount; i++ {
		assert.Contains(t, gauges, fmt.Sprintf("CPUutilization%d", i))
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

	err := sendBatch(context.Background(), restyClient, server.URL+"/updates/", "", batch)
	require.NoError(t, err)

	assert.Equal(t, "/updates/", receivedPath)
	assert.Equal(t, "application/json", receivedContentType)
	assert.Equal(t, "gzip", receivedContentEncoding)
	require.Len(t, receivedBatch, 2)
	assert.ElementsMatch(t, []string{"PollCount", "RandomValue"}, []string{receivedBatch[0].ID, receivedBatch[1].ID})
}

func TestSendBatch_Signature(t *testing.T) {
	var (
		headerPresent bool
		gotHeader     string
		gotBody       []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerPresent = len(r.Header.Values(signature.Header)) > 0
		gotHeader = r.Header.Get(signature.Header)
		var err error
		gotBody, err = readBody(r)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restyClient := resty.NewWithClient(server.Client())
	delta := int64(1)
	batch := []models.Metrics{{ID: "PollCount", MType: "counter", Delta: &delta}}

	t.Run("with key", func(t *testing.T) {
		require.NoError(t, sendBatch(context.Background(), restyClient, server.URL, "secret", batch))

		assert.True(t, headerPresent)
		assert.True(t, signature.Valid("secret", gotBody, gotHeader), "header must be HMAC of the uncompressed JSON body")
	})

	t.Run("without key", func(t *testing.T) {
		require.NoError(t, sendBatch(context.Background(), restyClient, server.URL, "", batch))

		assert.False(t, headerPresent, "header must not be sent without a key")
	})
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
