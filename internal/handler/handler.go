package handler

import (
	"net/http"
	"strconv"

	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/service"
)

func MetricsHandler(metricService service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		metricsType := r.PathValue("metricsType")
		metricsName := r.PathValue("metricsName")
		metricsValue := r.PathValue("metricsValue")

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
