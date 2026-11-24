// Package models описывает формат метрик для передачи и хранения.
package models

// Типы метрик.
const (
	Counter = "counter"
	Gauge   = "gauge"
)

// Metrics — метрика в формате JSON.
// Delta задаётся для counter, Value — для gauge; указатели отличают ноль от отсутствия значения.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
