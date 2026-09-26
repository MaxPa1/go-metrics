package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/MaxPa1/go-metrics/internal/compress"
	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/retry"
	"github.com/MaxPa1/go-metrics/internal/signature"
	"github.com/go-resty/resty/v2"
)

type MetricAgent struct {
	client         *resty.Client
	updatesURL     string
	key            string
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
	store          *store
}

func NewMetricAgent(cfg *config.AgentConfig, client *resty.Client) *MetricAgent {
	return &MetricAgent{
		client:         client,
		updatesURL:     "http://" + cfg.Address + "/updates/",
		key:            cfg.Key,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		rateLimit:      cfg.RateLimit,
		store:          newStore(),
	}
}

func (m *MetricAgent) Run(ctx context.Context) {
	jobs := make(chan report, 1)

	var wg sync.WaitGroup
	wg.Go(func() { pollPeriodically(ctx, m.pollInterval, m.pollRuntime) })
	wg.Go(func() { pollPeriodically(ctx, m.pollInterval, m.pollSystem) })
	wg.Go(func() { m.schedule(ctx, jobs) })
	wg.Go(func() { runWorkers(ctx, m.rateLimit, jobs, m.send) })
	wg.Wait()
}

func (m *MetricAgent) pollRuntime(context.Context) {
	m.store.setGauges(collectRuntimeMetrics())
	m.store.addPollCount(1)
}

func (m *MetricAgent) pollSystem(ctx context.Context) {
	gauges, err := collectSystemMetrics(ctx)
	if err != nil {
		log.Printf("Error collecting system metrics: %s\n", err)
		return
	}
	m.store.setGauges(gauges)
}

func (m *MetricAgent) schedule(ctx context.Context, jobs chan<- report) {
	defer close(jobs)

	ticker := time.NewTicker(m.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gauges, pollCount := m.store.read()
			r := report{metrics: buildBatch(gauges, pollCount), pollCount: pollCount}
			select {
			case jobs <- r:
				m.store.addPollCount(-pollCount)
			default:
				log.Printf("All %d senders are busy, skipping report\n", m.rateLimit)
			}
		}
	}
}

func (m *MetricAgent) send(ctx context.Context, r report) {
	if err := sendBatch(ctx, m.client, m.updatesURL, m.key, r.metrics); err != nil {
		log.Printf("Error sending metrics batch: %s\n", err)
		m.store.addPollCount(r.pollCount)
	}
}

type report struct {
	metrics   []models.Metrics
	pollCount int64
}

func buildBatch(gauges map[string]float64, pollCount int64) []models.Metrics {
	batch := make([]models.Metrics, 0, len(gauges)+1)
	for name, value := range gauges {
		batch = append(batch, models.Metrics{ID: name, MType: models.Gauge, Value: &value})
	}
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount})
	return batch
}

func runWorkers(ctx context.Context, n int, jobs <-chan report, handle func(context.Context, report)) {
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			for job := range jobs {
				handle(ctx, job)
			}
		})
	}
	wg.Wait()
}

func pollPeriodically(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		fn(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func sendBatch(ctx context.Context, client *resty.Client, url, key string, batch []models.Metrics) error {
	jsonBody, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal metrics batch: %w", err)
	}

	var sign string
	if key != "" {
		sign = signature.Sum(key, jsonBody)
	}

	compressed, err := compress.Compress(jsonBody)
	if err != nil {
		return fmt.Errorf("compress metrics batch: %w", err)
	}

	return retry.Do(ctx, retry.IsRetriableNetError, func() error {
		req := client.R().
			SetContext(ctx).
			SetBody(compressed).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip")
		if sign != "" {
			req.SetHeader(signature.Header, sign)
		}

		resp, err := req.Post(url)
		if err != nil {
			return fmt.Errorf("post metrics batch: %w", err)
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("unexpected status for metrics batch: %d", resp.StatusCode())
		}
		return nil
	})
}
