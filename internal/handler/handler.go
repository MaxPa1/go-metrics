package handler

import (
	"encoding/json"
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

func MetricsV2Handler(metricService MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request models.Metrics
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		switch request.MType {
		case models.Gauge:
			if request.Value == nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			metricService.RecordGauge(request.ID, *request.Value)
		case models.Counter:
			if request.Delta == nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			metricService.RecordCounter(request.ID, *request.Delta)
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetMetricsV2Handler(metricService MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request models.Metrics
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if request.ID == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var response models.Metrics
		switch request.MType {
		case models.Gauge:
			value, err := metricService.GetGauge(request.ID)
			if err != nil {
				writeError(w, err)
				return
			}
			response.Value = &value
		case models.Counter:
			value, err := metricService.GetCounter(request.ID)
			if err != nil {
				writeError(w, err)
				return
			}
			response.Delta = &value
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response.ID = request.ID
		response.MType = request.MType
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
