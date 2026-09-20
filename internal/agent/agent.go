package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"

	"github.com/MaxPa1/go-metrics/internal/compress"
	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/retry"
	"github.com/go-resty/resty/v2"
)

type MetricAgent struct {
	pollCount   int64
	randomValue float64
	gauges      map[string]float64
	updatesURL  string
}

func NewMetricAgent(cfg *config.AgentConfig) *MetricAgent {
	return &MetricAgent{
		gauges:     make(map[string]float64),
		updatesURL: "http://" + cfg.Address + "/updates/",
	}
}

func (m *MetricAgent) SendMetrics(ctx context.Context, client *resty.Client) {
	batch := m.collectMetrics()
	if len(batch) == 0 {
		return
	}

	if err := sendBatch(ctx, client, m.updatesURL, batch); err != nil {
		log.Printf("Error sending metrics batch: %s\n", err)
		return
	}
	m.pollCount = 0
}

func (m *MetricAgent) collectMetrics() []models.Metrics {
	batch := make([]models.Metrics, 0, len(m.gauges)+2)

	for name, value := range m.gauges {
		batch = append(batch, models.Metrics{ID: name, MType: models.Gauge, Value: &value})
	}

	randomValue := m.randomValue
	batch = append(batch, models.Metrics{ID: "RandomValue", MType: models.Gauge, Value: &randomValue})

	pollCount := m.pollCount
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount})

	return batch
}

func sendBatch(ctx context.Context, client *resty.Client, url string, batch []models.Metrics) error {
	jsonBody, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal metrics batch: %w", err)
	}

	compressed, err := compress.Compress(jsonBody)
	if err != nil {
		return fmt.Errorf("compress metrics batch: %w", err)
	}

	return retry.Do(ctx, retry.IsRetriableNetError, func() error {
		resp, err := client.R().
			SetBody(compressed).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			Post(url)
		if err != nil {
			return fmt.Errorf("post metrics batch: %w", err)
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("unexpected status for metrics batch: %d", resp.StatusCode())
		}
		return nil
	})
}

func (m *MetricAgent) UpdateMetrics() {
	ms := new(runtime.MemStats)
	runtime.ReadMemStats(ms)

	m.gauges = map[string]float64{
		"Alloc":         float64(ms.Alloc),
		"BuckHashSys":   float64(ms.BuckHashSys),
		"Frees":         float64(ms.Frees),
		"GCCPUFraction": ms.GCCPUFraction,
		"GCSys":         float64(ms.GCSys),
		"HeapAlloc":     float64(ms.HeapAlloc),
		"HeapIdle":      float64(ms.HeapIdle),
		"HeapInuse":     float64(ms.HeapInuse),
		"HeapObjects":   float64(ms.HeapObjects),
		"HeapReleased":  float64(ms.HeapReleased),
		"HeapSys":       float64(ms.HeapSys),
		"LastGC":        float64(ms.LastGC),
		"Lookups":       float64(ms.Lookups),
		"MCacheInuse":   float64(ms.MCacheInuse),
		"MCacheSys":     float64(ms.MCacheSys),
		"MSpanInuse":    float64(ms.MSpanInuse),
		"MSpanSys":      float64(ms.MSpanSys),
		"Mallocs":       float64(ms.Mallocs),
		"NextGC":        float64(ms.NextGC),
		"NumForcedGC":   float64(ms.NumForcedGC),
		"NumGC":         float64(ms.NumGC),
		"OtherSys":      float64(ms.OtherSys),
		"PauseTotalNs":  float64(ms.PauseTotalNs),
		"StackInuse":    float64(ms.StackInuse),
		"StackSys":      float64(ms.StackSys),
		"Sys":           float64(ms.Sys),
		"TotalAlloc":    float64(ms.TotalAlloc),
	}
	m.pollCount++
	m.randomValue = rand.Float64()
}
