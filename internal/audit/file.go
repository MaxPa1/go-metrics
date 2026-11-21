package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type FileObserver struct {
	file *os.File
	enc  *json.Encoder
}

func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: f, enc: json.NewEncoder(f)}, nil
}

func (o *FileObserver) Notify(_ context.Context, e Event) error {
	if err := o.enc.Encode(e); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func (o *FileObserver) Close() error {
	return o.file.Close()
}
