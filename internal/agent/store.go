package agent

import (
	"maps"
	"sync"

	models "github.com/MaxPa1/go-metrics/internal/model"
)

type report struct {
	metrics   []models.Metrics
	pollCount int64
}

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

func (s *store) snapshot() report {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch := make([]models.Metrics, 0, len(s.gauges)+1)
	for name, value := range s.gauges {
		batch = append(batch, models.Metrics{ID: name, MType: models.Gauge, Value: &value})
	}

	pollCount := s.pollCount
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount})
	s.pollCount = 0

	return report{metrics: batch, pollCount: pollCount}
}
