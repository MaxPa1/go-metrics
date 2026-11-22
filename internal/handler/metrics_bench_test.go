package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/MaxPa1/go-metrics/internal/audit"
	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type noopAuditor struct{}

func (noopAuditor) Notify(audit.Event) {}

func benchBatchJSON(b *testing.B) ([]models.Metrics, []byte) {
	b.Helper()
	batch := make([]models.Metrics, 0, 39)
	for i := range 38 {
		v := float64(i) * 1.5
		batch = append(batch, models.Metrics{ID: "Gauge" + strconv.Itoa(i), MType: models.Gauge, Value: &v})
	}
	d := int64(5)
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d})
	data, err := json.Marshal(batch)
	if err != nil {
		b.Fatal(err)
	}
	return batch, data
}

func benchRouter(b *testing.B) (http.Handler, []byte) {
	b.Helper()
	batch, data := benchBatchJSON(b)
	svc := service.NewMetricsService(repository.NewMemStorage())
	if err := svc.RecordBatch(context.Background(), batch); err != nil {
		b.Fatal(err)
	}

	r := chi.NewRouter()
	r.Post("/updates/", MetricsListHandler(svc, noopAuditor{}))
	r.Post("/update/", MetricsV2Handler(svc, noopAuditor{}))
	r.Post("/value/", GetMetricsV2Handler(svc, zap.NewNop().Sugar()))
	r.Get("/value/{metricsType}/{metricsName}", GetMetricsHandler(svc))
	r.Get("/", GetAllMetricsHandler(svc))
	return r, data
}

func BenchmarkHandlers(b *testing.B) {
	router, batch := benchRouter(b)

	cases := []struct {
		name   string
		method string
		target string
		body   []byte
	}{
		{name: "POST /updates/ batch", method: http.MethodPost, target: "/updates/", body: batch},
		{name: "POST /update/ gauge", method: http.MethodPost, target: "/update/", body: []byte(`{"id":"Gauge1","type":"gauge","value":1.5}`)},
		{name: "POST /value/ gauge", method: http.MethodPost, target: "/value/", body: []byte(`{"id":"Gauge1","type":"gauge"}`)},
		{name: "GET /value/counter", method: http.MethodGet, target: "/value/counter/PollCount"},
		{name: "GET / all metrics", method: http.MethodGet, target: "/"},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				req := httptest.NewRequest(tc.method, tc.target, bytes.NewReader(tc.body))
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					b.Fatalf("unexpected status %d", rec.Code)
				}
			}
		})
	}
}
