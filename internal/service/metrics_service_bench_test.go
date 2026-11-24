package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/repository"
)

func benchService(b *testing.B) (*MetricsServiceImpl, []models.Metrics) {
	b.Helper()
	batch := make([]models.Metrics, 0, 39)
	for i := range 38 {
		v := float64(i) * 1.5
		batch = append(batch, models.Metrics{ID: "Gauge" + strconv.Itoa(i), MType: models.Gauge, Value: &v})
	}
	d := int64(5)
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d})

	s := NewMetricsService(repository.NewMemStorage())
	if err := s.RecordBatch(context.Background(), batch); err != nil {
		b.Fatal(err)
	}
	return s, batch
}

func BenchmarkMetricsService_RecordBatch(b *testing.B) {
	s, batch := benchService(b)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if err := s.RecordBatch(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMetricsService_GetGauge(b *testing.B) {
	s, _ := benchService(b)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.GetGauge(ctx, "Gauge10"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMetricsService_GetAll(b *testing.B) {
	s, _ := benchService(b)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.GetAll(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
