package hash

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSum(t *testing.T) {
	got := Sum("Jefe", []byte("what do ya want for nothing?"))
	assert.Equal(t, "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843", got)
}

func TestValid(t *testing.T) {
	data := []byte(`[{"id":"PollCount","type":"counter","delta":1}]`)
	sum := Sum("key", data)

	tests := []struct {
		name string
		key  string
		data []byte
		got  string
		want bool
	}{
		{"correct", "key", data, sum, true},
		{"uppercase hex", "key", data, strings.ToUpper(sum), true},
		{"wrong key", "other", data, sum, false},
		{"tampered data", "key", []byte("tampered"), sum, false},
		{"empty signature", "key", data, "", false},
		{"not hex", "key", data, "zzzz", false},
		{"truncated", "key", data, sum[:10], false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Valid(tt.key, tt.data, tt.got))
		})
	}
}
