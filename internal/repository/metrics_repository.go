package repository

import (
	"sync"
)

type MemStorage struct {
	mutex      *sync.Mutex
	counterMap map[string]int64
	gaugeMap   map[string]float64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mutex:      &sync.Mutex{},
		counterMap: make(map[string]int64),
		gaugeMap:   make(map[string]float64),
	}
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.gaugeMap[name] = value
}

func (m *MemStorage) UpdateCounter(name string, delta int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	current, _ := m.counterMap[name]
	current += delta
	m.counterMap[name] = current
}

func (m *MemStorage) FindGauge(name string) (float64, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.gaugeMap[name]
	return v, ok
}

func (m *MemStorage) FindCounter(name string) (int64, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.counterMap[name]
	return v, ok
}

func (m *MemStorage) FindAll() (counters map[string]int64, gauges map[string]float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	counters = make(map[string]int64, len(m.counterMap))
	for k, v := range m.counterMap {
		counters[k] = v
	}

	gauges = make(map[string]float64, len(m.gaugeMap))
	for k, v := range m.gaugeMap {
		gauges[k] = v
	}
	return
}
