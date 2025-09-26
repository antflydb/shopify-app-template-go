package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteConfig struct {
	Path string
}

type SQLite struct {
	db *sql.DB
}

var _ Database = (*SQLite)(nil)

func NewSQLite(ctx context.Context, cfg *SQLiteConfig) (*SQLite, error) {
	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	return &SQLite{db: db}, nil
}

func (s *SQLite) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteResult{result: result}, nil
}

func (s *SQLite) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqliteRows{rows: rows}, nil
}

func (s *SQLite) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := s.db.QueryRowContext(ctx, query, args...)
	return &sqliteRow{row: row}
}

func (s *SQLite) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// Adapter types for SQLite

type sqliteResult struct {
	result sql.Result
}

func (r *sqliteResult) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *sqliteResult) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}

type sqliteRows struct {
	rows *sql.Rows
}

func (r *sqliteRows) Close() error {
	return r.rows.Close()
}

func (r *sqliteRows) Columns() ([]string, error) {
	return r.rows.Columns()
}

func (r *sqliteRows) Err() error {
	return r.rows.Err()
}

func (r *sqliteRows) Next() bool {
	return r.rows.Next()
}

func (r *sqliteRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

type sqliteRow struct {
	row *sql.Row
}

func (r *sqliteRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}