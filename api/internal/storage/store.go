package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/antflydb/shopify-app-template-go/internal/entity"
	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/pkg/database"
	"github.com/antflydb/shopify-app-template-go/pkg/logging"
	"github.com/huandu/go-sqlbuilder"
)

type storeStorage struct {
	database.Database
	logger logging.Logger
}

var _ service.StoreStorage = (*storeStorage)(nil)

func NewStoreStorage(db database.Database) *storeStorage {
	return &storeStorage{
		Database: db,
		logger:   logging.NewZap("info").Named("StoreStorage"),
	}
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
	err := s.QueryRow(ctx, query, args...).Scan(
		&store.ID,
		&store.Name,
		&store.Nonce,
		&store.AccessToken,
		&store.Installed,
		&store.CreatedAt,
		&store.UpdatedAt,
		&store.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) || (err != nil && strings.Contains(err.Error(), "no rows in result set")) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get store: %w", err)
	}

	return &store, nil
}

func (s *storeStorage) Update(ctx context.Context, store *entity.Store) (*entity.Store, error) {
	logger := s.logger.Named("Update").WithContext(ctx).With("storeName", store.Name)
	logger.Info("attempting to update store in database", "installed", store.Installed, "hasAccessToken", store.AccessToken != "")

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

	logger.Debug("executing update query", "query", query)
	_, err := s.Exec(ctx, query, args...)
	if err != nil {
		logger.Error("failed to execute update query", "err", err)
		return nil, fmt.Errorf("failed to update store: %w", err)
	}

	updatedStore, err := s.Get(ctx, store.Name)
	if err != nil {
		logger.Error("failed to get updated store after update", "err", err)
		return nil, fmt.Errorf("failed to get updated store: %w", err)
	}

	logger.Info("successfully updated store in database", "storeId", updatedStore.ID, "storeName", updatedStore.Name, "installed", updatedStore.Installed)
	return updatedStore, nil
}

func (s *storeStorage) Create(ctx context.Context, store *entity.Store) (*entity.Store, error) {
	logger := s.logger.Named("Create").WithContext(ctx).With("storeName", store.Name)
	logger.Info("attempting to create store in database", "storeId", store.ID, "installed", store.Installed)

	now := time.Now()
	store.CreatedAt = now
	store.UpdatedAt = now

	sb := sqlbuilder.NewInsertBuilder()
	query, args := sb.
		InsertInto("stores").
		Cols("name", "nonce", "access_token", "installed", "created_at", "updated_at").
		Values(store.Name, store.Nonce, store.AccessToken, store.Installed, store.CreatedAt, store.UpdatedAt).
		Build()

	logger.Debug("executing create query", "query", query, "args", args)
	_, err := s.Exec(ctx, query, args...)
	if err != nil {
		logger.Error("failed to execute create query", "err", err)
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	logger.Info("successfully created store in database", "storeId", store.ID, "storeName", store.Name)
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

	_, err := s.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete store: %w", err)
	}

	return nil
}
