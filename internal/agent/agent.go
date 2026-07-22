package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"

	"github.com/MaxPa1/go-metrics/internal/model"
)

type Agent interface {
	SendMetrics(client *http.Client)
}

type MetricAgent struct {
	pollCount   int64
	randomValue float64
	gauges      map[string]float64
	baseURL     string
}

func NewMetricAgent(baseURL string) *MetricAgent {
	return &MetricAgent{
		gauges:  make(map[string]float64),
		baseURL: baseURL,
	}
}

func (m *MetricAgent) SendMetrics(client *http.Client) {
	for name, value := range m.gauges {
		sendMetric(client, m.baseURL, models.Gauge, name, fmt.Sprintf("%v", value))
	}
	sendMetric(client, m.baseURL, models.Gauge, "randomValue", fmt.Sprintf("%v", m.randomValue))
	sendMetric(client, m.baseURL, models.Counter, "pollCount", fmt.Sprintf("%d", m.pollCount))
}

func sendMetric(client *http.Client, baseURL, mType, name, value string) {
	url := fmt.Sprintf("%s/%s/%s/%s", baseURL, mType, name, value)
	resp, err := client.Post(url, "text/plain", http.NoBody)
	if err != nil {
		log.Printf("Error sending %s %s: %v", mType, name, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK status for %s %s: %d", mType, name, resp.StatusCode)
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
