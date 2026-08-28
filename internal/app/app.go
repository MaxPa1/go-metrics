package app

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MaxPa1/go-metrics/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type App struct {
	cfg config.ServConfig
	db  *sql.DB
}

func New(cfg config.ServConfig) (*App, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &App{
		cfg: cfg,
		db:  db,
	}, nil
}

func (app *App) CheckDb(ctx context.Context) error {
	err := app.db.PingContext(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (app *App) Close() error {
	if app.db != nil {
		return app.db.Close()
	}
	return nil
}
