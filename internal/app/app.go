package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/MaxPa1/go-metrics/migrations"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type App struct {
	cfg     config.ServConfig
	db      *sql.DB
	storage service.MetricsStorage
}

func New(ctx context.Context, cfg config.ServConfig) (*App, error) {
	app := &App{cfg: cfg}

	switch {
	case cfg.DatabaseDSN != "":
		db, err := sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("open db: %w", err)
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(15)
		db.SetConnMaxLifetime(10 * time.Minute)
		db.SetConnMaxIdleTime(10 * time.Minute)

		if err := db.PingContext(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("ping db: %w", err)
		}

		if err := runMigrations(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("run migrations: %w", err)
		}

		app.db = db
		app.storage = repository.NewDBStorage(db)

	case cfg.FileStoragePath != "":
		fs, err := repository.NewFileStorage(ctx, cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
		if err != nil {
			return nil, fmt.Errorf("file storage: %w", err)
		}
		app.storage = fs

	default:
		app.storage = repository.NewMemStorage()
	}

	return app, nil
}

func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func (app *App) CheckDB(ctx context.Context) error {
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

func (app *App) Storage() service.MetricsStorage {
	return app.storage
}
