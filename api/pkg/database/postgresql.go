package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (p *PostgreSQL) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	commandTag, err := p.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresResult{commandTag: commandTag}, nil
}

func (p *PostgreSQL) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &postgresRows{rows: rows}, nil
}

func (p *PostgreSQL) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	row := p.pool.QueryRow(ctx, query, args...)
	return &postgresRow{row: row}
}

func (p *PostgreSQL) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Adapter types to bridge pgx and database/sql interfaces

type postgresResult struct {
	commandTag pgconn.CommandTag
}

func (r *postgresResult) LastInsertId() (int64, error) {
	return 0, fmt.Errorf("LastInsertId is not supported by this driver")
}

func (r *postgresResult) RowsAffected() (int64, error) {
	return r.commandTag.RowsAffected(), nil
}

type postgresRows struct {
	rows pgx.Rows
}

func (r *postgresRows) Close() error {
	r.rows.Close()
	return nil
}

func (r *postgresRows) ColumnTypes() ([]*sql.ColumnType, error) {
	return nil, fmt.Errorf("ColumnTypes is not implemented")
}

func (r *postgresRows) Columns() ([]string, error) {
	fieldDescs := r.rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = fd.Name
	}
	return columns, nil
}

func (r *postgresRows) Err() error {
	return r.rows.Err()
}

func (r *postgresRows) Next() bool {
	return r.rows.Next()
}

func (r *postgresRows) NextResultSet() bool {
	return false
}

func (r *postgresRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

type postgresRow struct {
	row pgx.Row
}

func (r *postgresRow) Err() error {
	return nil
}

func (r *postgresRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}
