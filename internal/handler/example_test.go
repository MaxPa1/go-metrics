package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/MaxPa1/go-metrics/internal/audit"
	"github.com/MaxPa1/go-metrics/internal/handler"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func newExampleRouter() http.Handler {
	svc := service.NewMetricsService(repository.NewMemStorage())
	auditor := audit.NewAuditor(zap.NewNop().Sugar())
	log := zap.NewNop().Sugar()

	r := chi.NewRouter()
	r.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.MetricsHandler(svc, auditor))
	r.Post("/update", handler.MetricsV2Handler(svc, auditor))
	r.Post("/updates", handler.MetricsListHandler(svc, auditor))
	r.Get("/value/{metricsType}/{metricsName}", handler.GetMetricsHandler(svc))
	r.Post("/value", handler.GetMetricsV2Handler(svc, log))
	r.Get("/", handler.GetAllMetricsHandler(svc))
	return r
}

func do(h http.Handler, method, target, body string) (int, string) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	b, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, strings.TrimSpace(string(b))
}

func ExampleMetricsHandler() {
	r := newExampleRouter()

	code, _ := do(r, http.MethodPost, "/update/counter/PollCount/5", "")
	fmt.Println(code)

	code, _ = do(r, http.MethodPost, "/update/gauge/Alloc/abc", "")
	fmt.Println(code)

	// Output:
	// 200
	// 400
}

func ExampleGetMetricsHandler() {
	r := newExampleRouter()
	do(r, http.MethodPost, "/update/gauge/Alloc/12.5", "")

	code, body := do(r, http.MethodGet, "/value/gauge/Alloc", "")
	fmt.Println(code, body)

	code, _ = do(r, http.MethodGet, "/value/gauge/Unknown", "")
	fmt.Println(code)

	// Output:
	// 200 12.5
	// 404
}

func ExampleMetricsV2Handler() {
	r := newExampleRouter()

	code, _ := do(r, http.MethodPost, "/update", `{"id":"Alloc","type":"gauge","value":3.14}`)
	fmt.Println(code)

	// Output:
	// 200
}

func ExampleGetMetricsV2Handler() {
	r := newExampleRouter()
	do(r, http.MethodPost, "/update", `{"id":"PollCount","type":"counter","delta":2}`)
	do(r, http.MethodPost, "/update", `{"id":"PollCount","type":"counter","delta":3}`)

	code, body := do(r, http.MethodPost, "/value", `{"id":"PollCount","type":"counter"}`)
	fmt.Println(code, body)

	// Output:
	// 200 {"id":"PollCount","type":"counter","delta":5}
}

func ExampleMetricsListHandler() {
	r := newExampleRouter()

	code, _ := do(r, http.MethodPost, "/updates", `[
		{"id":"Alloc","type":"gauge","value":1.5},
		{"id":"PollCount","type":"counter","delta":10}
	]`)
	fmt.Println(code)

	_, body := do(r, http.MethodGet, "/value/counter/PollCount", "")
	fmt.Println(body)

	// Output:
	// 200
	// 10
}

func ExampleGetAllMetricsHandler() {
	r := newExampleRouter()
	do(r, http.MethodPost, "/update/gauge/Alloc/7", "")

	code, body := do(r, http.MethodGet, "/", "")
	fmt.Println(code, strings.Contains(body, "<li>Alloc: 7</li>"))

	// Output:
	// 200 true
}
