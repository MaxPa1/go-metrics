package retry

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withFastDelays(t *testing.T) {
	t.Helper()
	original := Delays
	Delays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { Delays = original })
}

func TestDo_SucceedsWithoutRetry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func(error) bool { return true }, func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_RetriesUntilSuccess(t *testing.T) {
	withFastDelays(t)

	calls := 0
	retriableErr := errors.New("temporary")
	err := Do(context.Background(), func(error) bool { return true }, func() error {
		calls++
		if calls < 3 {
			return retriableErr
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_StopsOnNonRetriableError(t *testing.T) {
	withFastDelays(t)

	calls := 0
	nonRetriable := errors.New("fatal")
	err := Do(context.Background(), func(error) bool { return false }, func() error {
		calls++
		return nonRetriable
	})
	require.ErrorIs(t, err, nonRetriable)
	assert.Equal(t, 1, calls)
}

func TestDo_StopsAfterMaxAttempts(t *testing.T) {
	withFastDelays(t)

	calls := 0
	retriableErr := errors.New("still failing")
	err := Do(context.Background(), func(error) bool { return true }, func() error {
		calls++
		return retriableErr
	})
	require.ErrorIs(t, err, retriableErr)
	assert.Equal(t, len(Delays)+1, calls)
}

func TestDo_StopsOnContextCancellation(t *testing.T) {
	original := Delays
	Delays = []time.Duration{time.Hour}
	defer func() { Delays = original }()

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	retriableErr := errors.New("still failing")
	done := make(chan error, 1)
	go func() {
		done <- Do(ctx, func(error) bool { return true }, func() error {
			calls++
			return retriableErr
		})
	}()

	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, retriableErr)
	case <-time.After(time.Second):
		t.Fatal("Do did not return after context cancellation")
	}
	assert.Equal(t, 1, calls)
}

func TestIsRetriablePgError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection exception",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionException},
			want: true,
		},
		{
			name: "connection failure",
			err:  &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			want: true,
		},
		{
			name: "unique violation is not retriable",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			want: false,
		},
		{
			name: "wrapped connection exception",
			err:  errWrap(&pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist}),
			want: true,
		},
		{
			name: "network error before reaching postgres",
			err:  &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetriablePgError(tt.err))
		})
	}
}

func TestIsRetriableNetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection refused",
			err:  &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			want: true,
		},
		{
			name: "wrapped connection reset",
			err:  errWrap(syscall.ECONNRESET),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetriableNetError(tt.err))
		})
	}
}

func errWrap(err error) error {
	return errors.Join(err)
}
