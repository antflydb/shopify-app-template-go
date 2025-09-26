package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/antflydb/shopify-app-template-go/internal/entity"
	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/pkg/database"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
)

type storeStorage struct {
	database.Database
}

var _ service.StoreStorage = (*storeStorage)(nil)

func NewStoreStorage(db database.Database) *storeStorage {
	return &storeStorage{db}
}

func (s *storeStorage) Get(ctx context.Context, storeName string) (*entity.Store, error) {
	sb := sqlbuilder.NewSelectBuilder()
	query, args := sb.
		Select("id", "name", "nonce", "access_token", "installed", "created_at", "updated_at", "deleted_at").
		From("stores").
		Where(sb.Equal("name", storeName)).
		Where(sb.IsNull("deleted_at")).
		Build()

	var store entity.Store
	err := s.Pool().QueryRow(ctx, query, args...).Scan(
		&store.ID,
		&store.Name,
		&store.Nonce,
		&store.AccessToken,
		&store.Installed,
		&store.CreatedAt,
		&store.UpdatedAt,
		&store.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}

	return &store, nil
}

func (s *storeStorage) Update(ctx context.Context, store *entity.Store) (*entity.Store, error) {
	now := time.Now()
	store.UpdatedAt = now

	sb := sqlbuilder.NewUpdateBuilder()
	query, args := sb.
		Update("stores").
		Set(
			sb.Assign("nonce", store.Nonce),
			sb.Assign("access_token", store.AccessToken),
			sb.Assign("installed", store.Installed),
			sb.Assign("updated_at", now),
		).
		Where(sb.Equal("name", store.Name)).
		Where(sb.IsNull("deleted_at")).
		Build()

	_, err := s.Pool().Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update store: %w", err)
	}

	updatedStore, err := s.Get(ctx, store.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated store: %w", err)
	}

	return updatedStore, nil
}

func (s *storeStorage) Create(ctx context.Context, store *entity.Store) (*entity.Store, error) {
	now := time.Now()
	store.CreatedAt = now
	store.UpdatedAt = now

	sb := sqlbuilder.NewInsertBuilder()
	query, args := sb.
		InsertInto("stores").
		Cols("id", "name", "nonce", "access_token", "installed", "created_at", "updated_at").
		Values(store.ID, store.Name, store.Nonce, store.AccessToken, store.Installed, store.CreatedAt, store.UpdatedAt).
		Build()

	_, err := s.Pool().Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return store, nil
}

func (s *storeStorage) Delete(ctx context.Context, storeName string) error {
	now := time.Now()

	sb := sqlbuilder.NewUpdateBuilder()
	query, args := sb.
		Update("stores").
		Set(sb.Assign("deleted_at", now)).
		Where(sb.Equal("name", storeName)).
		Where(sb.IsNull("deleted_at")).
		Build()

	_, err := s.Pool().Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete store: %w", err)
	}

	return nil
}
