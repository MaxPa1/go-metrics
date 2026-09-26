package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MaxPa1/go-metrics/internal/model"
	"github.com/MaxPa1/go-metrics/internal/retry"
)

const (
	updateGauge = `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, 'gauge', $2)
		ON CONFLICT (id, type) DO UPDATE SET value = EXCLUDED.value
	`

	updateCounter = `
		INSERT INTO metrics (id, type, delta)
		VALUES ($1, 'counter', $2)
		ON CONFLICT (id, type) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
	`

	findGauge = `
		SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'
	`

	findCounter = `
		SELECT delta FROM metrics WHERE id = $1 AND type = 'counter'
	`

	findAll = `
		SELECT id, type, delta, value FROM metrics
	`

	updateGaugeBatch = `
		INSERT INTO metrics (id, type, value)
		SELECT unnest($1::text[]), 'gauge', unnest($2::double precision[])
		ON CONFLICT (id, type) DO UPDATE SET value = EXCLUDED.value
	`

	updateCounterBatch = `
		INSERT INTO metrics (id, type, delta)
		SELECT unnest($1::text[]), 'counter', unnest($2::bigint[])
		ON CONFLICT (id, type) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
	`
)

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{
		db: db,
	}
}

func (d *DBStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	err := retry.Do(ctx, retry.IsRetriablePgError, func() error {
		_, err := d.db.ExecContext(ctx, updateGauge, name, value)
		return err
	})
	if err != nil {
		return fmt.Errorf("update gauge %q: %w", name, err)
	}
	return nil
}

func (d *DBStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	err := retry.Do(ctx, retry.IsRetriablePgError, func() error {
		_, err := d.db.ExecContext(ctx, updateCounter, name, value)
		return err
	})
	if err != nil {
		return fmt.Errorf("update counter %q: %w", name, err)
	}
	return nil
}

func (d *DBStorage) FindGauge(ctx context.Context, name string) (float64, bool, error) {
	var value float64
	err := retry.Do(ctx, retry.IsRetriablePgError, func() error {
		return d.db.QueryRowContext(ctx, findGauge, name).Scan(&value)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find gauge %q: %w", name, err)
	}
	return value, true, nil
}

func (d *DBStorage) FindCounter(ctx context.Context, name string) (int64, bool, error) {
	var delta int64
	err := retry.Do(ctx, retry.IsRetriablePgError, func() error {
		return d.db.QueryRowContext(ctx, findCounter, name).Scan(&delta)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find counter %q: %w", name, err)
	}
	return delta, true, nil
}

func (d *DBStorage) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	return retry.Do(ctx, retry.IsRetriablePgError, func() error {
		return d.updateBatchOnce(ctx, metrics)
	})
}

func (d *DBStorage) updateBatchOnce(ctx context.Context, metrics []models.Metrics) error {
	gauges, counters := splitBatch(metrics)

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if len(gauges.ids) > 0 {
		if _, err := tx.ExecContext(ctx, updateGaugeBatch, gauges.ids, gauges.values); err != nil {
			return fmt.Errorf("update gauges batch: %w", err)
		}
	}

	if len(counters.ids) > 0 {
		if _, err := tx.ExecContext(ctx, updateCounterBatch, counters.ids, counters.deltas); err != nil {
			return fmt.Errorf("update counters batch: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

type gaugeBatch struct {
	ids    []string
	values []float64
}

type counterBatch struct {
	ids    []string
	deltas []int64
}

func splitBatch(metrics []models.Metrics) (gauges gaugeBatch, counters counterBatch) {
	gaugeIdx := make(map[string]int, len(metrics))
	counterIdx := make(map[string]int, len(metrics))

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				continue
			}
			if idx, ok := gaugeIdx[metric.ID]; ok {
				gauges.values[idx] = *metric.Value
				continue
			}
			gaugeIdx[metric.ID] = len(gauges.ids)
			gauges.ids = append(gauges.ids, metric.ID)
			gauges.values = append(gauges.values, *metric.Value)
		case models.Counter:
			if metric.Delta == nil {
				continue
			}
			if idx, ok := counterIdx[metric.ID]; ok {
				counters.deltas[idx] += *metric.Delta
				continue
			}
			counterIdx[metric.ID] = len(counters.ids)
			counters.ids = append(counters.ids, metric.ID)
			counters.deltas = append(counters.deltas, *metric.Delta)
		}
	}
	return gauges, counters
}

func (d *DBStorage) FindAll(ctx context.Context) (map[string]int64, map[string]float64, error) {
	var counters map[string]int64
	var gauges map[string]float64

	err := retry.Do(ctx, retry.IsRetriablePgError, func() error {
		var err error
		counters, gauges, err = d.findAllOnce(ctx)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return counters, gauges, nil
}

func (d *DBStorage) findAllOnce(ctx context.Context) (map[string]int64, map[string]float64, error) {
	rows, err := d.db.QueryContext(ctx, findAll)
	if err != nil {
		return nil, nil, fmt.Errorf("find all metrics: %w", err)
	}
	defer rows.Close()

	counters := make(map[string]int64)
	gauges := make(map[string]float64)

	for rows.Next() {
		var (
			id    string
			mType string
			delta sql.NullInt64
			value sql.NullFloat64
		)
		if err := rows.Scan(&id, &mType, &delta, &value); err != nil {
			return nil, nil, fmt.Errorf("scan row: %w", err)
		}

		switch mType {
		case "counter":
			if delta.Valid {
				counters[id] = delta.Int64
			}
		case "gauge":
			if value.Valid {
				gauges[id] = value.Float64
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate rows: %w", err)
	}

	return counters, gauges, nil
}
