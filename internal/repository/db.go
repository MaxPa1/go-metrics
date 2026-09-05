package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
)

type DbStorage struct {
	db *sql.DB
}

func NewDbStorage(db *sql.DB) *DbStorage {
	return &DbStorage{
		db: db,
	}
}

func (d *DbStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	_, err := d.db.ExecContext(ctx, updateGauge, name, value)
	if err != nil {
		return fmt.Errorf("update gauge %q: %w", name, err)
	}
	return nil
}

func (d *DbStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	_, err := d.db.ExecContext(ctx, updateCounter, name, value)
	if err != nil {
		return fmt.Errorf("update counter %q: %w", name, err)
	}
	return nil
}

func (d *DbStorage) FindGauge(ctx context.Context, name string) (float64, bool, error) {
	var value float64
	err := d.db.QueryRowContext(ctx, findGauge, name).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find gauge %q: %w", name, err)
	}
	return value, true, nil
}

func (d *DbStorage) FindCounter(ctx context.Context, name string) (int64, bool, error) {
	var delta int64
	err := d.db.QueryRowContext(ctx, findCounter, name).Scan(&delta)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find counter %q: %w", name, err)
	}
	return delta, true, nil
}

func (d *DbStorage) FindAll(ctx context.Context) (map[string]int64, map[string]float64, error) {
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
