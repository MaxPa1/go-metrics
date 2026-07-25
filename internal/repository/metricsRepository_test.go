package repository

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_UpdateCounter(t *testing.T) {
	type fields struct {
		mutex sync.RWMutex
		data  map[string]interface{}
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
			fields: fields{data: map[string]interface{}{"calls": int64(4)}},
			args:   args{"calls", int64(1)},
			want:   int64(5),
		},
		{
			name:   "Add new counter",
			fields: fields{data: map[string]interface{}{}},
			args:   args{"calls", int64(1)},
			want:   int64(1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex: tt.fields.mutex,
				data:  tt.fields.data,
			}
			m.UpdateCounter(tt.args.name, tt.args.delta)
			val, ok := m.data[tt.args.name].(int64)
			require.True(t, ok, "value should be int64")
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	type fields struct {
		mutex sync.RWMutex
		data  map[string]interface{}
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
			fields: fields{data: map[string]interface{}{"cpu": 0.32}},
			args:   args{"cpu", 0.76},
			want:   0.76,
		},
		{
			name:   "Add new gauge",
			fields: fields{data: map[string]interface{}{}},
			args:   args{"memory", 33.2},
			want:   33.2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex: tt.fields.mutex,
				data:  tt.fields.data,
			}
			m.UpdateGauge(tt.args.name, tt.args.value)

			val, ok := m.data[tt.args.name].(float64)
			require.True(t, ok, "value should be float64")
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestMemStorage_FindGauge(t *testing.T) {
	type fields struct {
		mutex sync.RWMutex
		data  map[string]interface{}
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
			fields: fields{data: map[string]interface{}{"cpu": 0.32}},
			args:   args{"cpu"},
			want:   0.32,
			want1:  true,
		},
		{
			name:   "Find not existing gauge",
			fields: fields{data: map[string]interface{}{}},
			args:   args{"SYS"},
			want:   0,
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex: tt.fields.mutex,
				data:  tt.fields.data,
			}
			got, got1 := m.FindGauge(tt.args.name)
			assert.Equalf(t, tt.want, got, "FindGauge(%v)", tt.args.name)
			assert.Equalf(t, tt.want1, got1, "FindGauge(%v)", tt.args.name)
		})
	}
}

func TestMemStorage_FindCounter(t *testing.T) {
	type fields struct {
		mutex sync.RWMutex
		data  map[string]interface{}
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
			fields: fields{data: map[string]interface{}{"Counter": int64(36)}},
			args:   args{"Counter"},
			want:   int64(36),
			want1:  true,
		},
		{
			name:   "Find not existing counter",
			fields: fields{data: map[string]interface{}{}},
			args:   args{"Counter"},
			want:   0,
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				mutex: tt.fields.mutex,
				data:  tt.fields.data,
			}
			got, got1 := m.FindCounter(tt.args.name)
			assert.Equalf(t, tt.want, got, "FindCounter(%v)", tt.args.name)
			assert.Equalf(t, tt.want1, got1, "FindCounter(%v)", tt.args.name)
		})
	}
}
