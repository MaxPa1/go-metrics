package repository

import (
	"context"
	"sync"
	"testing"

	models "github.com/MaxPa1/go-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_UpdateCounter(t *testing.T) {
	type fields struct {
		data map[string]int64
	}
	type args struct {
		name  string
		delta int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
	}{
		{
			name:   "Update existing counter",
			fields: fields{data: map[string]int64{"calls": int64(4)}},
			args:   args{"calls", int64(1)},
			want:   int64(5),
		},
		{
			name:   "Add new counter",
			fields: fields{data: map[string]int64{}},
			args:   args{"calls", int64(1)},
			want:   int64(1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex:      &sync.Mutex{},
				counterMap: tt.fields.data,
			}
			err := m.UpdateCounter(context.Background(), tt.args.name, tt.args.delta)
			require.NoError(t, err)
			val, ok := m.counterMap[tt.args.name]
			require.True(t, ok)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	type fields struct {
		data map[string]float64
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
	}{
		{
			name:   "Update existing gauge",
			fields: fields{data: map[string]float64{"cpu": 0.32}},
			args:   args{"cpu", 0.76},
			want:   0.76,
		},
		{
			name:   "Add new gauge",
			fields: fields{data: map[string]float64{}},
			args:   args{"memory", 33.2},
			want:   33.2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex:    &sync.Mutex{},
				gaugeMap: tt.fields.data,
			}
			err := m.UpdateGauge(context.Background(), tt.args.name, tt.args.value)
			require.NoError(t, err)
			val, ok := m.gaugeMap[tt.args.name]
			require.True(t, ok)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestMemStorage_UpdateBatch(t *testing.T) {
	m := &MemStorage{
		mutex:      &sync.Mutex{},
		counterMap: map[string]int64{"calls": 4},
		gaugeMap:   map[string]float64{"cpu": 0.32},
	}

	gaugeValue := 0.76
	counterDelta := int64(1)
	err := m.UpdateBatch(context.Background(), []models.Metrics{
		{ID: "cpu", MType: models.Gauge, Value: &gaugeValue},
		{ID: "calls", MType: models.Counter, Delta: &counterDelta},
		{ID: "memory", MType: models.Gauge, Value: nil},
	})
	require.NoError(t, err)

	assert.Equal(t, 0.76, m.gaugeMap["cpu"])
	assert.Equal(t, int64(5), m.counterMap["calls"])
	_, ok := m.gaugeMap["memory"]
	assert.False(t, ok)
}

func TestMemStorage_FindGauge(t *testing.T) {
	type fields struct {
		data map[string]float64
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
		want1  bool
	}{
		{
			name:   "Find existing gauge",
			fields: fields{data: map[string]float64{"cpu": 0.32}},
			args:   args{"cpu"},
			want:   0.32,
			want1:  true,
		},
		{
			name:   "Find not existing gauge",
			fields: fields{data: map[string]float64{}},
			args:   args{"SYS"},
			want:   0,
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex:    &sync.Mutex{},
				gaugeMap: tt.fields.data,
			}
			got, got1, err := m.FindGauge(context.Background(), tt.args.name)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "FindGauge(%v)", tt.args.name)
			assert.Equalf(t, tt.want1, got1, "FindGauge(%v)", tt.args.name)
		})
	}
}

func TestMemStorage_FindCounter(t *testing.T) {
	type fields struct {
		data map[string]int64
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
		want1  bool
	}{
		{
			name:   "Find existing counter",
			fields: fields{data: map[string]int64{"Counter": int64(36)}},
			args:   args{"Counter"},
			want:   int64(36),
			want1:  true,
		},
		{
			name:   "Find not existing counter",
			fields: fields{data: map[string]int64{}},
			args:   args{"Counter"},
			want:   0,
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex:      &sync.Mutex{},
				counterMap: tt.fields.data,
			}
			got, got1, err := m.FindCounter(context.Background(), tt.args.name)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "FindCounter(%v)", tt.args.name)
			assert.Equalf(t, tt.want1, got1, "FindCounter(%v)", tt.args.name)
		})
	}
}
