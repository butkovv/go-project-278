package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func NewPostgreSQLDB(ctx context.Context, dsn string) (*sql.DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	db := stdlib.OpenDBFromPool(pool)

	if err != nil {
		return nil, fmt.Errorf("error creating pg pool: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("db unavailable: %w", err)
	}

	return db, nil
}
