package database

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database interface {
	// Pool is used to get database connection pool.
	Pool() *pgxpool.Pool
	// Close is used to close database connection.
	Close()
}

// Model provides base fields for all database models.
type Model struct {
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-"`
}
