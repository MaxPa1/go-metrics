package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MaxPa1/go-metrics/internal/audit"
	"github.com/MaxPa1/go-metrics/internal/config"
	"github.com/MaxPa1/go-metrics/internal/repository"
	"github.com/MaxPa1/go-metrics/internal/retry"
	"github.com/MaxPa1/go-metrics/internal/service"
	"github.com/MaxPa1/go-metrics/migrations"

	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type storageCloser interface {
	Close(ctx context.Context) error
}

type App struct {
	cfg     config.ServConfig
	db      *sql.DB
	storage service.MetricsStorage
	auditor *audit.Auditor
}

func New(ctx context.Context, cfg config.ServConfig, log *zap.SugaredLogger) (_ *App, err error) {
	auditor, err := newAuditor(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}
	defer func() {
		if err != nil {
			_ = auditor.Close(context.Background())
		}
	}()

	app := &App{cfg: cfg, auditor: auditor}

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

		if err := retry.Do(ctx, retry.IsRetriablePgError, func() error {
			return db.PingContext(ctx)
		}); err != nil {
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

func newAuditor(cfg config.ServConfig, log *zap.SugaredLogger) (*audit.Auditor, error) {
	var observers []audit.Observer

	if cfg.AuditURL != "" {
		ho, err := audit.NewHTTPObserver(cfg.AuditURL, nil)
		if err != nil {
			return nil, err
		}
		observers = append(observers, ho)
	}

	if cfg.AuditFile != "" {
		fo, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, err
		}
		observers = append(observers, fo)
	}

	return audit.NewAuditor(log, observers...), nil
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
	return retry.Do(ctx, retry.IsRetriablePgError, func() error {
		return app.db.PingContext(ctx)
	})
}

func (app *App) Close(ctx context.Context) error {
	var errs []error

	if err := app.auditor.Close(ctx); err != nil {
		errs = append(errs, fmt.Errorf("close auditor: %w", err))
	}

	if closer, ok := app.storage.(storageCloser); ok {
		if err := closer.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close storage: %w", err))
		}
	}

	if app.db != nil {
		if err := app.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close db: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (app *App) Storage() service.MetricsStorage {
	return app.storage
}

func (app *App) Auditor() *audit.Auditor {
	return app.auditor
}
