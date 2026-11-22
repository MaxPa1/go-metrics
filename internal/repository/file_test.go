package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func readMetricsFile(path string) (map[string]int64, map[string]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, nil, err
	}
	counters := map[string]int64{}
	gauges := map[string]float64{}
	for _, m := range metrics {
		switch {
		case m.MType == models.Counter && m.Delta != nil:
			counters[m.ID] = *m.Delta
		case m.MType == models.Gauge && m.Value != nil:
			gauges[m.ID] = *m.Value
		}
	}
	return counters, gauges, nil
}

func TestFileStorage_Save(t *testing.T) {
	wantCounters := map[string]int64{"PollCount": 5}
	wantGauges := map[string]float64{"Alloc": 12.5, "HeapAlloc": 7.25}

	tests := []struct {
		name            string
		subdir          string
		createSubdir    bool
		interval        time.Duration
		wantBeforeClose bool
		wantCloseErr    bool
	}{
		{
			name:            "sync save on every update",
			interval:        0,
			wantBeforeClose: true,
		},
		{
			name:            "periodic save by ticker",
			interval:        20 * time.Millisecond,
			wantBeforeClose: true,
		},
		{
			name:     "long interval saves only on close",
			interval: time.Hour,
		},
		{
			name:         "missing directory",
			subdir:       "missing-dir",
			interval:     0,
			wantCloseErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), tt.subdir, "storage.json")

			fs, err := NewFileStorage(ctx, path, tt.interval, false)
			require.NoError(t, err)

			require.NoError(t, fs.UpdateGauge(ctx, "Alloc", 12.5))
			require.NoError(t, fs.UpdateCounter(ctx, "PollCount", 3))
			require.NoError(t, fs.UpdateBatch(ctx, []models.Metrics{
				{ID: "PollCount", MType: models.Counter, Delta: ptr(int64(2))},
				{ID: "HeapAlloc", MType: models.Gauge, Value: ptr(7.25)},
			}))

			if tt.wantBeforeClose {
				assert.Eventually(t, func() bool {
					counters, gauges, err := readMetricsFile(path)
					return err == nil && assert.ObjectsAreEqual(wantCounters, counters) &&
						assert.ObjectsAreEqual(wantGauges, gauges)
				}, 2*time.Second, 10*time.Millisecond)
			} else if !tt.wantCloseErr {
				_, err := os.Stat(path)
				assert.ErrorIs(t, err, os.ErrNotExist)
			}

			err = fs.Close(ctx)
			if tt.wantCloseErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			counters, gauges, err := readMetricsFile(path)
			require.NoError(t, err)
			assert.Equal(t, wantCounters, counters)
			assert.Equal(t, wantGauges, gauges)
		})
	}
}

func TestFileStorage_Restore(t *testing.T) {
	tests := []struct {
		name         string
		content      *string
		wantErr      bool
		wantCounters map[string]int64
		wantGauges   map[string]float64
	}{
		{
			name:         "missing file",
			content:      nil,
			wantCounters: map[string]int64{},
			wantGauges:   map[string]float64{},
		},
		{
			name:         "empty file",
			content:      ptr(""),
			wantCounters: map[string]int64{},
			wantGauges:   map[string]float64{},
		},
		{
			name:    "broken file",
			content: ptr("{broken"),
			wantErr: true,
		},
		{
			name:         "valid metrics",
			content:      ptr(`[{"id":"PollCount","type":"counter","delta":5},{"id":"Alloc","type":"gauge","value":12.5}]`),
			wantCounters: map[string]int64{"PollCount": 5},
			wantGauges:   map[string]float64{"Alloc": 12.5},
		},
		{
			name:         "skips metrics without values and unknown types",
			content:      ptr(`[{"id":"a","type":"counter"},{"id":"b","type":"gauge"},{"id":"c","type":"unknown","delta":1}]`),
			wantCounters: map[string]int64{},
			wantGauges:   map[string]float64{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "storage.json")
			if tt.content != nil {
				require.NoError(t, os.WriteFile(path, []byte(*tt.content), 0600))
			}

			fs, err := NewFileStorage(ctx, path, 0, true)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			counters, gauges, err := fs.FindAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCounters, counters)
			assert.Equal(t, tt.wantGauges, gauges)
		})
	}
}

func TestMemStorage_FindAll(t *testing.T) {
	tests := []struct {
		name     string
		counters map[string]int64
		gauges   map[string]float64
	}{
		{
			name:     "empty storage",
			counters: map[string]int64{},
			gauges:   map[string]float64{},
		},
		{
			name:     "filled storage",
			counters: map[string]int64{"PollCount": 1, "Requests": 10},
			gauges:   map[string]float64{"Alloc": 2, "HeapAlloc": 3.5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := NewMemStorage()
			for k, v := range tt.counters {
				require.NoError(t, m.UpdateCounter(ctx, k, v))
			}
			for k, v := range tt.gauges {
				require.NoError(t, m.UpdateGauge(ctx, k, v))
			}

			counters, gauges, err := m.FindAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.counters, counters)
			assert.Equal(t, tt.gauges, gauges)

			counters["mutated"] = 100
			gauges["mutated"] = 100
			again, againGauges, err := m.FindAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.counters, again, "FindAll должен возвращать копию")
			assert.Equal(t, tt.gauges, againGauges, "FindAll должен возвращать копию")
		})
	}
}
