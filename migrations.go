package migrations // или package migrations

import "embed"

//go:embed db/migrations/*.sql
var EmbedMigrations embed.FS
