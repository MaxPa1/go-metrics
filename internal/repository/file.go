package repository

import (
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
}

func NewFileStorage(path string, interval time.Duration, restore bool) (*FileStorage, error) {
	fs := &FileStorage{
		MemStorage: NewMemStorage(),
		path:       path,
		syncSave:   interval == 0,
	}

	if restore {
		if err := loadFromFile(path, fs.MemStorage); err != nil {
			return nil, err
		}
	}

	if !fs.syncSave {
		go fs.runTicker(interval)
	}

	return fs, nil
}

func (fs *FileStorage) runTicker(interval time.Duration) {
	for range time.Tick(interval) {
		fs.Save()
	}
}

func (fs *FileStorage) UpdateGauge(name string, value float64) {
	fs.MemStorage.UpdateGauge(name, value)
	if fs.syncSave {
		fs.Save()
	}
}

func (fs *FileStorage) UpdateCounter(name string, delta int64) {
	fs.MemStorage.UpdateCounter(name, delta)
	if fs.syncSave {
		fs.Save()
	}
}

func (fs *FileStorage) Save() {
	if err := saveToFile(fs.path, fs.MemStorage); err != nil {
		log.Printf("save metrics: %v", err)
	}
}

func saveToFile(path string, storage *MemStorage) error {
	counters, gauges := storage.FindAll()
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

func loadFromFile(path string, storage *MemStorage) error {
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
				storage.UpdateCounter(m.ID, *m.Delta)
			}
		case models.Gauge:
			if m.Value != nil {
				storage.UpdateGauge(m.ID, *m.Value)
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
