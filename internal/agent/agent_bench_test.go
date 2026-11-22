package agent

import (
	"encoding/json"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/compress"
)

func BenchmarkCollectRuntimeMetrics(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		collectRuntimeMetrics()
	}
}

func BenchmarkBuildBatch(b *testing.B) {
	gauges := collectRuntimeMetrics()

	b.ReportAllocs()
	for b.Loop() {
		buildBatch(gauges, 5)
	}
}

func BenchmarkStore(b *testing.B) {
	s := newStore()
	gauges := collectRuntimeMetrics()

	b.ReportAllocs()
	for b.Loop() {
		s.setGauges(gauges)
		s.addPollCount(1)
		s.read()
	}
}

func BenchmarkPrepareBatchBody(b *testing.B) {
	batch := buildBatch(collectRuntimeMetrics(), 5)

	b.ReportAllocs()
	for b.Loop() {
		data, err := json.Marshal(batch)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := compress.Compress(data); err != nil {
			b.Fatal(err)
		}
	}
}
