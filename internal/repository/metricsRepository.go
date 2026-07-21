package repository

import (
	"sync"
)

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

func (m *MemStorage) UpdateCounter(name string, delta int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	current, _ := m.data[name].(int64)
	current += delta
	m.data[name] = current
}
