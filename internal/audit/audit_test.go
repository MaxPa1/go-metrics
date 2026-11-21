package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingObserver struct {
	mu     sync.Mutex
	events []Event
	block  chan struct{}
	closed bool
}

func (o *recordingObserver) Notify(ctx context.Context, e Event) error {
	if o.block != nil {
		select {
		case <-o.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, e)
	return nil
}

func (o *recordingObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.closed = true
	return nil
}

func (o *recordingObserver) snapshot() ([]Event, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Event(nil), o.events...), o.closed
}

func TestAuditor_NoObservers(t *testing.T) {
	a := NewAuditor(zap.NewNop().Sugar())
	a.Notify(NewEvent([]string{"Alloc"}, "127.0.0.1"))
	require.NoError(t, a.Close(context.Background()))
	require.NoError(t, a.Close(context.Background()))
}

func TestAuditor_FanOutAndDrainOnClose(t *testing.T) {
	o1, o2 := &recordingObserver{}, &recordingObserver{}
	a := NewAuditor(zap.NewNop().Sugar(), o1, o2)

	for i := 0; i < 10; i++ {
		a.Notify(NewEvent([]string{"m"}, "127.0.0.1"))
	}
	require.NoError(t, a.Close(context.Background()))

	for _, o := range []*recordingObserver{o1, o2} {
		events, closed := o.snapshot()
		assert.Len(t, events, 10)
		assert.True(t, closed)
	}
}

func TestAuditor_SlowObserverDoesNotBlock(t *testing.T) {
	slow := &recordingObserver{block: make(chan struct{})}
	a := NewAuditor(zap.NewNop().Sugar(), slow)

	const total = queueSize * 2
	done := make(chan struct{})
	go func() {
		for i := 0; i < total; i++ {
			a.Notify(NewEvent([]string{"m"}, "127.0.0.1"))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Notify blocked on slow observer")
	}

	close(slow.block)
	require.NoError(t, a.Close(context.Background()))

	events, _ := slow.snapshot()
	assert.Less(t, len(events), total, "overflowing events must be dropped")
}

func TestAuditor_CloseTimeoutCancelsDelivery(t *testing.T) {
	stuck := &recordingObserver{block: make(chan struct{})}
	a := NewAuditor(zap.NewNop().Sugar(), stuck)
	a.Notify(NewEvent([]string{"m"}, "127.0.0.1"))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	require.ErrorIs(t, a.Close(ctx), context.DeadlineExceeded)
	_, closed := stuck.snapshot()
	assert.True(t, closed)
}

func TestFileObserver_AppendsLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	require.NoError(t, os.WriteFile(path, []byte(`{"existing":true}`+"\n"), 0o600))

	o, err := NewFileObserver(path)
	require.NoError(t, err)

	e1 := Event{TS: 1, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"}
	e2 := Event{TS: 2, Metrics: []string{"PollCount"}, IPAddress: "10.0.0.1"}
	require.NoError(t, o.Notify(context.Background(), e1))
	require.NoError(t, o.Notify(context.Background(), e2))
	require.NoError(t, o.Close())

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	require.NoError(t, sc.Err())
	require.Len(t, lines, 3)
	assert.Equal(t, `{"existing":true}`, lines[0])
	assert.JSONEq(t, `{"ts":1,"metrics":["Alloc","Frees"],"ip_address":"192.168.0.42"}`, lines[1])
	assert.JSONEq(t, `{"ts":2,"metrics":["PollCount"],"ip_address":"10.0.0.1"}`, lines[2])
}

func TestFileObserver_InvalidPath(t *testing.T) {
	_, err := NewFileObserver(filepath.Join(t.TempDir(), "no-such-dir", "audit.log"))
	assert.Error(t, err)
}

func TestHTTPObserver_Post(t *testing.T) {
	var got Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	o, err := NewHTTPObserver(srv.URL+"/audit", nil)
	require.NoError(t, err)

	e := Event{TS: 42, Metrics: []string{"Alloc"}, IPAddress: "192.168.0.42"}
	require.NoError(t, o.Notify(context.Background(), e))
	assert.Equal(t, e, got)
}

func TestHTTPObserver_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	o, err := NewHTTPObserver(srv.URL, nil)
	require.NoError(t, err)
	assert.Error(t, o.Notify(context.Background(), Event{}))
}

func TestNewHTTPObserver_InvalidURL(t *testing.T) {
	for _, u := range []string{"", "localhost:8080", "ftp://example.com", "/relative/path"} {
		_, err := NewHTTPObserver(u, nil)
		assert.Error(t, err, u)
	}
}
