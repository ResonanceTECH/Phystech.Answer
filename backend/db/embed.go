// Package db ships the SQL migrations from db/migrations as embedded files.
package db

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS
