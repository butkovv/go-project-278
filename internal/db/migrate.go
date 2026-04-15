package db

import (
	"database/sql"
	"fmt"
	"url-shortener/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	DefaultDatabaseURL   = "postgres://test:test@localhost:5432/usdb?sslmode=disable"
	DefaultMigrationsDir = "internal/migrations/sql"
)

type MigrationOptions struct {
	DatabaseURL   string
	MigrationsDir string
	Dialect       string
}

func DefaultMigrationOptions() *MigrationOptions {
	return &MigrationOptions{
		DatabaseURL:   DefaultDatabaseURL,
		MigrationsDir: DefaultMigrationsDir,
		Dialect:       "postgres",
	}
}

func openDB(opts *MigrationOptions) (*sql.DB, error) {
	err := goose.SetDialect(opts.Dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	databaseURL := opts.DatabaseURL
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: unable to get config: %v\n", err)
	} else if cfg.DatabaseUrl != "" {
		databaseURL = cfg.DatabaseUrl
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	if _, err := goose.EnsureDBVersion(db); err != nil {
		fmt.Printf("Warning: failed to ensure goose_db_version table (will be created by first migration): %v\n", err)
	}

	return db, nil
}

func MigrateUp(opts *MigrationOptions) error {
	if opts == nil {
		opts = DefaultMigrationOptions()
	}

	db, err := openDB(opts)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", closeErr)
		}
	}()

	return goose.Up(db, opts.MigrationsDir)
}

func MigrateDown(opts *MigrationOptions) error {
	if opts == nil {
		opts = DefaultMigrationOptions()
	}

	db, err := openDB(opts)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", closeErr)
		}
	}()

	return goose.Down(db, opts.MigrationsDir)
}

func MigrateStatus(opts *MigrationOptions) error {
	if opts == nil {
		opts = DefaultMigrationOptions()
	}

	db, err := openDB(opts)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", closeErr)
		}
	}()

	return goose.Status(db, opts.MigrationsDir)
}

func MigrateReset(opts *MigrationOptions) error {
	if opts == nil {
		opts = DefaultMigrationOptions()
	}

	db, err := openDB(opts)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close DB: %v\n", closeErr)
		}
	}()

	err = goose.DownTo(db, opts.MigrationsDir, 0)
	if err != nil {
		return err
	}

	return MigrateUp(opts)
}
