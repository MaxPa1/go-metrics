package handler

import (
	"errors"
	"html/template"
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

func MetricsHandler(metricService service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
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

func GetMetricsHandler(metricService service.MetricsService) http.HandlerFunc {
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
		w.Write([]byte(body))
	}
}

func GetAllMetricsHandler(metricService service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := template.Must(template.New("metrics").Parse(tmpl))

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		metrics := metricService.GetAll()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		t.Execute(w, metrics)
	}
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrMetricNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
}
