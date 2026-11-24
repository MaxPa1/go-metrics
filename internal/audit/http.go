package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultHTTPTimeout = 5 * time.Second

// HTTPObserver отправляет события аудита POST-запросом на заданный URL.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт наблюдателя; при client == nil используется клиент с таймаутом по умолчанию.
func NewHTTPObserver(rawURL string, client *http.Client) (*HTTPObserver, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse audit url: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("audit url must be absolute http(s) URL, got %q", rawURL)
	}
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &HTTPObserver{url: u.String(), client: client}, nil
}

// Notify отправляет событие на сервер аудита.
func (o *HTTPObserver) Notify(ctx context.Context, e Event) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit event: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("audit server responded with status %d", resp.StatusCode)
	}
	return nil
}
