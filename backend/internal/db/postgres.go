package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

func Connect(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database url: %w", err)
	}

	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.MinConns = cfg.DBMinConns
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	logger.Info("Connected to PostgreSQL successfully")

	return &DB{
		Pool:   pool,
		logger: logger,
	}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.logger.Info("Closing PostgreSQL connection pool")
		db.Pool.Close()
	}
}

func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func RunMigrations(dbURL string, logger *slog.Logger) error {
	d, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer m.Close()

	logger.Info("Running database migrations...")
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("Database schema is up to date")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("Database migrations applied successfully")
	return nil
}
