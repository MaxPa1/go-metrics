package repository

import (
	"sync"
)

type MetricsStorage interface {
	UpdateGauge(name string, value float64)
	Gauge(name string) (float64, bool)

	UpdateCounter(name string, value int64)
	Counter(name string) (int64, bool)
}

type MemStorage struct {
	mutex sync.RWMutex
	data  map[string]interface{}
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]interface{}),
	}
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[name] = value
}

func (m *MemStorage) Gauge(name string) (float64, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.data[name].(float64)
	return v, ok
}

func (m *MemStorage) UpdateCounter(name string, delta int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	current, _ := m.data[name].(int64)
	current += delta
	m.data[name] = current
}

func (m *MemStorage) Counter(name string) (int64, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.data[name].(int64)
	return v, ok
}
