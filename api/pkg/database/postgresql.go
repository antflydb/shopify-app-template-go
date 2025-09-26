package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLConfig struct {
	User     string
	Password string
	Host     string
	Database string
}

type PostgreSQL struct {
	pool *pgxpool.Pool
}

// Check if implements the interface.
var _ Database = (*PostgreSQL)(nil)

// NewPostgreSQL is used to create new instance of PostgreSQL.
func NewPostgreSQL(ctx context.Context, cfg *PostgreSQLConfig) (*PostgreSQL, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Database,
	)

	// Connect to the database
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgresql: %w", err)
	}

	// Test the connection
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgresql: %w", err)
	}

	// Create UUID extension
	_, err = pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	return &PostgreSQL{pool: pool}, nil
}

func (p *PostgreSQL) Pool() *pgxpool.Pool {
	return p.pool
}

func (p *PostgreSQL) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}
