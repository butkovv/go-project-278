package main

import (
	"context"
	"log/slog"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/db"
	"url-shortener/internal/handlers"
	migrations "url-shortener/internal/migrations"

	"github.com/pressly/goose/v3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	dsn := cfg.DatabaseUrl
	if dsn == "" {
		slog.Error("DATABASE_URL not set")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.NewPostgreSQLDB(ctx, dsn)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer func() {
		err := pool.Close()
		if err != nil {
			slog.Error("failed to close database connection", "error", err)
		}
	}()

	err = pool.Ping()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	goose.SetBaseFS(migrations.EmbedMigrations)

	err = goose.Up(pool, "sql")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	router := handlers.SetupRouter(pool)

	slog.Info("server started", "address", cfg.Addr())

	err = router.Run(cfg.Addr())
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
