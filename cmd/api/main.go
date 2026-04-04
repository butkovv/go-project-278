package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"url-shortener/config"
	"url-shortener/db"
	"url-shortener/handlers"
)

//go:embed db/migrations/*sql
var embedMigrations embed.FS

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
	defer pool.Close()

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	err = pool.Ping()
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
