package repository

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/model"
)

func benchBatch(n int) []models.Metrics {
	batch := make([]models.Metrics, 0, n+1)
	for i := range n {
		batch = append(batch, models.Metrics{ID: "Gauge" + strconv.Itoa(i), MType: models.Gauge, Value: ptr(float64(i) * 1.5)})
	}
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: ptr(int64(5))})
	return batch
}

func BenchmarkMemStorage_UpdateBatch(b *testing.B) {
	ctx := context.Background()
	m := NewMemStorage()
	batch := benchBatch(38)

	b.ReportAllocs()
	for b.Loop() {
		if err := m.UpdateBatch(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemStorage_UpdateBatchParallel(b *testing.B) {
	ctx := context.Background()
	m := NewMemStorage()
	batch := benchBatch(38)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := m.UpdateBatch(ctx, batch); err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkMemStorage_FindGauge(b *testing.B) {
	ctx := context.Background()
	m := NewMemStorage()
	if err := m.UpdateBatch(ctx, benchBatch(38)); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := m.FindGauge(ctx, "Gauge10"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemStorage_FindAll(b *testing.B) {
	ctx := context.Background()
	m := NewMemStorage()
	if err := m.UpdateBatch(ctx, benchBatch(38)); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := m.FindAll(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorage_Save(b *testing.B) {
	ctx := context.Background()
	m := NewMemStorage()
	if err := m.UpdateBatch(ctx, benchBatch(38)); err != nil {
		b.Fatal(err)
	}
	path := filepath.Join(b.TempDir(), "storage.json")

	b.ReportAllocs()
	for b.Loop() {
		if err := saveToFile(ctx, path, m); err != nil {
			b.Fatal(err)
		}
	}
}
