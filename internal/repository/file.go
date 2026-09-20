package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	models "github.com/MaxPa1/go-metrics/internal/model"
)

type FileStorage struct {
	*MemStorage
	path     string
	syncSave bool
	ticker   *time.Ticker
	cancel   context.CancelFunc
}

func NewFileStorage(ctx context.Context, path string, interval time.Duration, restore bool) (*FileStorage, error) {
	bgCtx, cancel := context.WithCancel(context.Background())
	fs := &FileStorage{
		MemStorage: NewMemStorage(),
		path:       path,
		syncSave:   interval == 0,
		cancel:     cancel,
	}

	if restore {
		if err := loadFromFile(ctx, path, fs.MemStorage); err != nil {
			cancel()
			return nil, err
		}
	}

	if !fs.syncSave {
		fs.ticker = time.NewTicker(interval)
		go fs.runTicker(bgCtx)
	}

	return fs, nil
}

func (fs *FileStorage) runTicker(ctx context.Context) {
	defer fs.ticker.Stop()
	for {
		select {
		case <-fs.ticker.C:
			fs.Save(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (fs *FileStorage) Close(ctx context.Context) error {
	fs.cancel()
	if err := saveToFile(ctx, fs.path, fs.MemStorage); err != nil {
		return fmt.Errorf("final save: %w", err)
	}
	return nil
}

func (fs *FileStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	err := fs.MemStorage.UpdateGauge(ctx, name, value)
	if err != nil {
		return err
	}
	if fs.syncSave {
		fs.Save(ctx)
	}
	return nil
}

func (fs *FileStorage) UpdateCounter(ctx context.Context, name string, delta int64) error {
	err := fs.MemStorage.UpdateCounter(ctx, name, delta)
	if err != nil {
		return err
	}
	if fs.syncSave {
		fs.Save(ctx)
	}
	return nil
}

func (fs *FileStorage) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	err := fs.MemStorage.UpdateBatch(ctx, metrics)
	if err != nil {
		return err
	}
	if fs.syncSave {
		fs.Save(ctx)
	}
	return nil
}

func (fs *FileStorage) Save(ctx context.Context) {
	if err := saveToFile(ctx, fs.path, fs.MemStorage); err != nil {
		log.Printf("save metrics: %v", err)
	}
}

func saveToFile(ctx context.Context, path string, storage *MemStorage) error {
	counters, gauges, err := storage.FindAll(ctx)
	if err != nil {
		return err
	}
	metrics := append(toCounterMetrics(counters), toGaugeMetrics(gauges)...)
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func loadFromFile(ctx context.Context, path string, storage *MemStorage) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var metrics []models.Metrics
	if err := json.NewDecoder(file).Decode(&metrics); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	for _, m := range metrics {
		switch m.MType {
		case models.Counter:
			if m.Delta != nil {
				err := storage.UpdateCounter(ctx, m.ID, *m.Delta)
				if err != nil {
					return err
				}
			}
		case models.Gauge:
			if m.Value != nil {
				err := storage.UpdateGauge(ctx, m.ID, *m.Value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func toCounterMetrics(counters map[string]int64) []models.Metrics {
	metrics := make([]models.Metrics, 0, len(counters))
	for key, value := range counters {
		metric := models.Metrics{
			ID:    key,
			MType: models.Counter,
			Delta: &value,
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func toGaugeMetrics(gauges map[string]float64) []models.Metrics {
	metrics := make([]models.Metrics, 0, len(gauges))
	for key, value := range gauges {
		metric := models.Metrics{
			ID:    key,
			MType: models.Gauge,
			Value: &value,
		}
		metrics = append(metrics, metric)
	}
	return metrics
}
