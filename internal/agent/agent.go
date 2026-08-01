package agent

import (
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strconv"

	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/go-resty/resty/v2"
)

type MetricAgent struct {
	pollCount   int64
	randomValue float64
	gauges      map[string]float64
	baseURL     string
}

func NewMetricAgent(cfg *Config) *MetricAgent {
	return &MetricAgent{
		gauges:  make(map[string]float64),
		baseURL: "http://" + cfg.Address + "/update/{metricsType}/{metricsName}/{metricsValue}",
	}
}

func (m *MetricAgent) SendMetrics(client *resty.Client) {
	for name, value := range m.gauges {
		sendMetric(client, m.baseURL, models.Gauge, name, strconv.FormatFloat(value, 'g', -1, 64))
	}
	sendMetric(client, m.baseURL, models.Gauge, "randomValue", strconv.FormatFloat(m.randomValue, 'g', -1, 64))
	sendMetric(client, m.baseURL, models.Counter, "pollCount", strconv.FormatInt(m.pollCount, 10))
	m.pollCount = 0
}

func sendMetric(client *resty.Client, url, mType, name, value string) {
	resp, err := client.R().
		SetPathParams(map[string]string{
			"metricsType":  mType,
			"metricsName":  name,
			"metricsValue": value,
		}).
		SetHeader("content-type", "text/plain").
		Post(url)

	if err != nil {
		log.Printf("Error sending %s %s: %v", mType, name, err)
		return
	}
	if resp.StatusCode() != http.StatusOK {
		log.Printf("Non-OK status for %s %s: %d", mType, name, resp.StatusCode())
	}
}

func (m *MetricAgent) UpdateMetrics() {
	ms := new(runtime.MemStats)
	runtime.ReadMemStats(ms)

	m.gauges = map[string]float64{
		"Alloc":         float64(ms.Alloc),
		"BuckHashSys":   float64(ms.BuckHashSys),
		"Frees":         float64(ms.Frees),
		"GCCPUFraction": ms.GCCPUFraction,
		"GCSys":         float64(ms.GCSys),
		"HeapAlloc":     float64(ms.HeapAlloc),
		"HeapIdle":      float64(ms.HeapIdle),
		"HeapInuse":     float64(ms.HeapInuse),
		"HeapObjects":   float64(ms.HeapObjects),
		"HeapReleased":  float64(ms.HeapReleased),
		"HeapSys":       float64(ms.HeapSys),
		"LastGC":        float64(ms.LastGC),
		"Lookups":       float64(ms.Lookups),
		"MCacheInuse":   float64(ms.MCacheInuse),
		"MCacheSys":     float64(ms.MCacheSys),
		"MSpanInuse":    float64(ms.MSpanInuse),
		"MSpanSys":      float64(ms.MSpanSys),
		"Mallocs":       float64(ms.Mallocs),
		"NextGC":        float64(ms.NextGC),
		"NumForcedGC":   float64(ms.NumForcedGC),
		"NumGC":         float64(ms.NumGC),
		"OtherSys":      float64(ms.OtherSys),
		"PauseTotalNs":  float64(ms.PauseTotalNs),
		"StackInuse":    float64(ms.StackInuse),
		"StackSys":      float64(ms.StackSys),
		"Sys":           float64(ms.Sys),
		"TotalAlloc":    float64(ms.TotalAlloc),
	}
	m.pollCount++
	m.randomValue = rand.Float64()
}
