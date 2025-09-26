package database

import (
	"context"
	"time"
)

type Row interface {
	Scan(dest ...interface{}) error
}

type Rows interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...interface{}) error
}

type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type Database interface {
	// Exec executes a query without returning any rows.
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	// Query executes a query that returns rows.
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	// QueryRow executes a query that is expected to return at most one row.
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	// Close is used to close database connection.
	Close()
}

// Model provides base fields for all database models.
type Model struct {
	CreatedAt time.Time  `json:"-"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-"`
}
