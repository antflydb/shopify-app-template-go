package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/antflydb/shopify-app-template-go/config"
)

func NewDatabase(ctx context.Context, cfg *config.Config) (Database, error) {
	switch strings.ToLower(cfg.Database.Type) {
	case "postgres", "postgresql":
		return NewPostgreSQL(ctx, &PostgreSQLConfig{
			User:     cfg.Database.Postgres.User,
			Password: cfg.Database.Postgres.Password,
			Host:     cfg.Database.Postgres.Host,
			Database: cfg.Database.Postgres.Database,
		})
	case "sqlite", "sqlite3":
		return NewSQLite(ctx, &SQLiteConfig{
			Path: cfg.Database.SQLite.Path,
		})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}