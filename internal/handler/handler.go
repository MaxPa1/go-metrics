package handler

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

const tmpl = `
<!DOCTYPE html>
<html>
<head><title>Metrics</title></head>
<body>
  <h1>Metrics List</h1>
  <ul>
    {{range .}}<li>{{.}}</li>{{end}}
  </ul>
</body>
</html>`

type MetricsService interface {
	RecordGauge(name string, value float64)
	GetGauge(name string) (float64, error)
	RecordCounter(name string, value int64)
	GetCounter(name string) (int64, error)
	GetAll() []string
}

func MetricsHandler(metricService MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricsType := chi.URLParam(r, "metricsType")
		metricsName := chi.URLParam(r, "metricsName")
		metricsValue := chi.URLParam(r, "metricsValue")

		if metricsName == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch metricsType {
		case models.Gauge:
			val, err := strconv.ParseFloat(metricsValue, 64)
			if err != nil {
				http.Error(w, "invalid gauge value", http.StatusBadRequest)
				return
			}
			metricService.RecordGauge(metricsName, val)
		case models.Counter:
			val, err := strconv.ParseInt(metricsValue, 10, 64)
			if err != nil {
				http.Error(w, "invalid counter value", http.StatusBadRequest)
				return
			}
			metricService.RecordCounter(metricsName, val)
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetMetricsHandler(metricService MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricsType := chi.URLParam(r, "metricsType")
		metricsName := chi.URLParam(r, "metricsName")

		if metricsName == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body string
		switch metricsType {
		case models.Gauge:
			value, err := metricService.GetGauge(metricsName)
			if err != nil {
				writeError(w, err)
				return
			}
			body = strconv.FormatFloat(value, 'f', -1, 64)
		case models.Counter:
			value, err := metricService.GetCounter(metricsName)
			if err != nil {
				writeError(w, err)
				return
			}
			body = strconv.FormatInt(value, 10)
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(body))
		if err != nil {
			log.Printf("failed to write response body: %v", err)
			return
		}
	}
}

func GetAllMetricsHandler(metricService MetricsService) http.HandlerFunc {
	t, err := template.New("metrics").Parse(tmpl)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Printf("failed to parse template: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := metricService.GetAll()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		err := t.Execute(w, metrics)
		if err != nil {
			log.Printf("failed to execute template: %v", err)
			return
		}
	}
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrMetricNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
}
