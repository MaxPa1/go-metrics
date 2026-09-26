package agent

import (
	"maps"
	"sync"
)

type store struct {
	mu        sync.Mutex
	gauges    map[string]float64
	pollCount int64
}

func newStore() *store {
	return &store{gauges: make(map[string]float64)}
}

func (s *store) setGauges(gauges map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	maps.Copy(s.gauges, gauges)
}

func (s *store) addPollCount(delta int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollCount += delta
}

func (s *store) read() (gauges map[string]float64, pollCount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.gauges), s.pollCount
}
