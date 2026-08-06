package repository

import (
	"maps"
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

func (m *MemStorage) FindAll() (map[string]int64, map[string]float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	copyCounters := make(map[string]int64, len(m.counterMap))
	maps.Copy(copyCounters, m.counterMap)

	copyGauges := make(map[string]float64, len(m.gaugeMap))
	maps.Copy(copyGauges, m.gaugeMap)
	return copyCounters, copyGauges
}
