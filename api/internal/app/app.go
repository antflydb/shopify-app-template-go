package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/antflydb/shopify-app-template-go/config"
	"github.com/antflydb/shopify-app-template-go/internal/api/shopify"
	httpcontroller "github.com/antflydb/shopify-app-template-go/internal/controller/http"
	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/internal/storage"
	"github.com/antflydb/shopify-app-template-go/pkg/database"
	"github.com/antflydb/shopify-app-template-go/pkg/httpserver"
	"github.com/antflydb/shopify-app-template-go/pkg/logging"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Run(cfg *config.Config) {
	logger := logging.NewZap(cfg.Log.Level)
	logger.Info("loaded configuration", "apiKey", cfg.Shopify.ApiKey, "apiKeyLength", len(cfg.Shopify.ApiKey), "baseURL", cfg.App.BaseURL)
	ctx := context.Background()

	// Init db
	sql, err := database.NewDatabase(ctx, cfg)
	if err != nil {
		logger.Fatal("failed to connect to database", "err", err)
	}

	// Run migrations
	err = runMigrations(cfg, logger)
	if err != nil {
		logger.Fatal("migration failed", "err", err)
	}

	storages := service.Storages{
		Store: storage.NewStoreStorage(sql),
	}

	apis := service.APIs{
		Platform: shopify.NewAPI(shopify.Options{
			Config: cfg,
			Logger: logger,
		}),
	}

	serviceOptions := &service.Options{
		Apis:     apis,
		Storages: storages,
		Config:   cfg,
		Logger:   logger,
	}

	services := service.Services{
		Platform: service.NewPlatformService(serviceOptions),
	}

	// Init native HTTP handler
	mux := http.NewServeMux()

	httpcontroller.New(&httpcontroller.Options{
		Handler:  mux,
		Services: services,
		Storages: storages,
		Logger:   logger,
		Config:   cfg,
	})

	httpServer := httpserver.New(
		mux,
		httpserver.Port(cfg.HTTP.Port),
		httpserver.ReadTimeout(120*time.Second),
		httpserver.WriteTimeout(120*time.Second),
		httpserver.ShutdownTimeout(30*time.Second),
	)

	// Waiting for a signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		logger.Info("app - Run - signal: " + s.String())

	case err = <-httpServer.Notify():
		logger.Error("app - Run - httpServer.Notify", "err", err)
	}

	// Shutdown HTTP server
	err = httpServer.Shutdown()
	if err != nil {
		logger.Error("app - Run - httpServer.Shutdown", "err", err)
	}

	// Close database connection
	sql.Close()
}

func runMigrations(cfg *config.Config, logger logging.Logger) error {
	var databaseURL, migrationsPath string

	switch strings.ToLower(cfg.Database.Type) {
	case "postgres", "postgresql":
		databaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s/%s?sslmode=disable",
			cfg.Database.Postgres.User,
			cfg.Database.Postgres.Password,
			cfg.Database.Postgres.Host,
			cfg.Database.Postgres.Database,
		)
		migrationsPath = "file://migrations"
	case "sqlite", "sqlite3":
		databaseURL = fmt.Sprintf("sqlite3://%s", cfg.Database.SQLite.Path)
		migrationsPath = "file://migrations/sqlite"
	default:
		return fmt.Errorf("unsupported database type for migrations: %s", cfg.Database.Type)
	}

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("migrations completed successfully")
	return nil
}
