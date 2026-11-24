package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// FileObserver записывает события аудита в файл в формате JSON Lines.
type FileObserver struct {
	file *os.File
	enc  *json.Encoder
}

// NewFileObserver открывает файл для дозаписи событий.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: f, enc: json.NewEncoder(f)}, nil
}

// Notify записывает событие в файл.
func (o *FileObserver) Notify(_ context.Context, e Event) error {
	if err := o.enc.Encode(e); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

// Close закрывает файл.
func (o *FileObserver) Close() error {
	return o.file.Close()
}
