package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{name: "debug", level: "debug"},
		{name: "info", level: "info"},
		{name: "error", level: "error"},
		{name: "invalid level", level: "not-a-level", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := Initialize(tt.level)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, l)
		})
	}
}

func TestLoggingResponseWriter(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		writes     []string
		wantStatus int
		wantSize   int
	}{
		{
			name:       "implicit status",
			writes:     []string{"abc", "de"},
			wantStatus: http.StatusOK,
			wantSize:   5,
		},
		{
			name:       "explicit status without body",
			status:     http.StatusNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "explicit status with body",
			status:     http.StatusCreated,
			writes:     []string{"created"},
			wantStatus: http.StatusCreated,
			wantSize:   7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			info := &ResponseLogInfo{}
			w := &LoggingResponseWriter{ResponseWriter: rec, ResponseInfo: info}

			if tt.status != 0 {
				w.WriteHeader(tt.status)
			}
			for _, s := range tt.writes {
				n, err := w.Write([]byte(s))
				require.NoError(t, err)
				assert.Equal(t, len(s), n)
			}

			assert.Equal(t, tt.wantStatus, info.Status)
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantSize, info.Size)
		})
	}
}
