package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/antflydb/shopify-app-template-go/internal/entity"
	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/pkg/database"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
)

type sessionStorage struct {
	database.Database
}

var _ service.SessionStorage = (*sessionStorage)(nil)

func NewSessionStorage(db database.Database) *sessionStorage {
	return &sessionStorage{db}
}

func (s *sessionStorage) Get(ctx context.Context, sessionID string) (*entity.Session, error) {
	sb := sqlbuilder.NewSelectBuilder()
	query, args := sb.
		Select("session_id", "store_id").
		From("sessions").
		Where(sb.Equal("session_id", sessionID)).
		Build()

	var session entity.Session
	err := s.Pool().QueryRow(ctx, query, args...).Scan(
		&session.SessionID,
		&session.StoreID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (s *sessionStorage) Create(ctx context.Context, session *entity.Session) (*entity.Session, error) {
	sb := sqlbuilder.NewInsertBuilder()
	query, args := sb.
		InsertInto("sessions").
		Cols("session_id", "store_id").
		Values(session.SessionID, session.StoreID).
		Build()

	_, err := s.Pool().Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (s *sessionStorage) Delete(ctx context.Context, sessionID string) error {
	sb := sqlbuilder.NewDeleteBuilder()
	query, args := sb.
		DeleteFrom("sessions").
		Where(sb.Equal("session_id", sessionID)).
		Build()

	_, err := s.Pool().Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}
