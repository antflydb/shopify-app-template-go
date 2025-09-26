package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Run(cfg *config.Config) {
	logger := logging.NewZap(cfg.Log.Level)
	ctx := context.Background()

	// Init db
	sql, err := database.NewPostgreSQL(ctx, &database.PostgreSQLConfig{
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Host:     cfg.Postgres.Host,
		Database: cfg.Postgres.Database,
	})
	if err != nil {
		logger.Fatal("failed to connect to PostgreSQL", "err", err)
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

	// Init HTTP framework of choice
	httpHandler := gin.New()

	httpcontroller.New(&httpcontroller.Options{
		Handler:  httpHandler,
		Services: services,
		Storages: storages,
		Logger:   logger,
		Config:   cfg,
	})

	httpServer := httpserver.New(
		httpHandler,
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
	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Database,
	)

	m, err := migrate.New("file://migrations", databaseURL)
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
