package repository

import (
	"fmt"
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

func (m *MemStorage) FindGauge(name string) (float64, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.data[name].(float64)
	return v, ok
}

func (m *MemStorage) FindCounter(name string) (int64, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.data[name].(int64)
	return v, ok
}

func (m *MemStorage) FindAll() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	list := make([]string, 0, len(m.data))
	for k, v := range m.data {
		list = append(list, fmt.Sprintf("%s: %v", k, v))
	}
	return list
}
