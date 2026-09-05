package repository

import (
	"context"
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

func (m *MemStorage) UpdateGauge(_ context.Context, name string, value float64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.gaugeMap[name] = value
	return nil
}

func (m *MemStorage) UpdateCounter(_ context.Context, name string, delta int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.counterMap[name] += delta
	return nil
}

func (m *MemStorage) FindGauge(_ context.Context, name string) (float64, bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.gaugeMap[name]
	return v, ok, nil
}

func (m *MemStorage) FindCounter(_ context.Context, name string) (int64, bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.counterMap[name]
	return v, ok, nil
}

func (m *MemStorage) FindAll(_ context.Context) (map[string]int64, map[string]float64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	copyCounters := make(map[string]int64, len(m.counterMap))
	maps.Copy(copyCounters, m.counterMap)

	copyGauges := make(map[string]float64, len(m.gaugeMap))
	maps.Copy(copyGauges, m.gaugeMap)

	return copyCounters, copyGauges, nil
}
