package repository

import (
	"strconv"
	"strings"
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

func (m *MemStorage) FindAll() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	list := make([]string, 0, len(m.counterMap)+len(m.gaugeMap))

	var sb strings.Builder

	for k, v := range m.counterMap {
		sb.Reset()
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.Write(strconv.AppendInt(nil, v, 10))
		list = append(list, sb.String())
	}
	for k, v := range m.gaugeMap {
		sb.Reset()
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.Write(strconv.AppendFloat(nil, v, 'f', -1, 64))
		list = append(list, sb.String())
	}
	return list
}
