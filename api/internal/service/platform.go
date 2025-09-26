package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/antflydb/shopify-app-template-go/config"
	"github.com/antflydb/shopify-app-template-go/internal/entity"
	"github.com/antflydb/shopify-app-template-go/pkg/errs"
	"github.com/antflydb/shopify-app-template-go/pkg/logging"
)

// platformService service implements PlatformService interface.
type platformService struct {
	apis     APIs
	storages Storages
	config   *config.Config
	logger   logging.Logger
}

var _ PlatformService = (*platformService)(nil)

func NewPlatformService(opts *Options) *platformService {
	return &platformService{
		apis:     opts.Apis,
		storages: opts.Storages,
		config:   opts.Config,
		logger:   opts.Logger.Named("Platform"),
	}
}

func (s *platformService) Handle(ctx context.Context, storeName, installationURL string) (string, error) {
	logger := s.logger.Named("Handle").WithContext(ctx)

	// Check if store is not already installed
	store, err := s.storages.Store.Get(ctx, storeName)
	if err != nil {
		logger.Error("failed to get store from storage", "err", err)
		return "", fmt.Errorf("failed to get store from storage: %w", err)
	}
	logger = logger.With("store", store)
	logger.Debug("got store")

	if store != nil && store.Installed {
		logger.Info("store is already installed")
		return fmt.Sprintf("https://%s/admin/apps/%s/exit-iframe", storeName, s.config.Shopify.ApiKey), nil
	}

	res, err := s.apis.Platform.HandleInstall(HandleInstallOptions{
		InstallationURL: installationURL,
		RedirectURL:     s.config.App.BaseURL + "/auth/callback",
		StoreName:       storeName,
	})
	if err != nil {
		logger.Info(err.Error())
		return "", err
	}
	logger = logger.With("res", res)
	logger.Debug("handled install on api side")

	// Create new instance of a store in db or update existing one with nonce
	if store == nil {
		logger.Info("creating new store", "storeName", storeName, "nonce", res.Nonce)
		createdStore, err := s.storages.Store.Create(ctx, &entity.Store{
			Name:      storeName,
			Nonce:     res.Nonce,
			Installed: false,
		})
		if err != nil {
			logger.Error("failed to create store in storage", "err", err)
			return "", fmt.Errorf("failed to create store in storage: %w", err)
		}
		logger = logger.With("createdStore", createdStore)
		logger.Info("successfully created store", "storeId", createdStore.ID, "storeName", createdStore.Name)
	} else {
		logger.Info("updating existing store with new nonce", "storeName", storeName, "oldNonce", store.Nonce, "newNonce", res.Nonce)
		updatedStore, err := s.storages.Store.Update(ctx, &entity.Store{
			Name:      storeName,
			Nonce:     res.Nonce,
			Installed: false,
		})
		if err != nil {
			logger.Error("failed to updated store in storage", "err", err)
			return "", fmt.Errorf("failed to create store in storage: %w", err)
		}
		logger = logger.With("updatedStore", updatedStore)
		logger.Info("successfully updated store with new nonce", "storeId", updatedStore.ID, "storeName", updatedStore.Name)
	}
	logger.Info("got redirect url and saved store's nonce into db")

	return res.RedirectURL, nil
}

func (s *platformService) HandleRedirect(ctx context.Context, opts ServiceHandleRedirectOptions) error {
	logger := s.logger.
		Named("HandleRedirect").
		With("opts", opts)

	// Check if store exists
	store, err := s.storages.Store.Get(ctx, opts.StoreName)
	if err != nil {
		logger.Error("failed to get store from storage", "err", err)
		return fmt.Errorf("failed to get store from storage: %w", err)
	}
	if store == nil {
		logger.Info("store is not found")
		return ErrHandleRedirectStoreNotFound
	}
	logger = logger.With("store", store)
	logger.Debug("got store for redirect")

	accessToken, err := s.apis.Platform.HandleRedirect(APIHandleRedirectOptions{
		Nonce:         store.Nonce,
		RedirectedURL: opts.RedirectedURL,
		StoreName:     opts.StoreName,
	})
	if err != nil {
		if errs.IsExpected(err) {
			logger.Info(err.Error())
			return err
		}
		logger.Error("failed to handle redirect at API ", "err", err)
		return fmt.Errorf("failed to retrieve access token from api: %w", err)
	}
	logger.Debug("got access token")

	err = s.apis.Platform.SubscribeToAppUninstallWebhook(SubscribeToAppUninstallWebhookOptions{
		RedirectURL: fmt.Sprintf("%s/uninstall?shop=%s", s.config.App.BaseURL, opts.StoreName),
		StoreName:   opts.StoreName,
		AccessToken: accessToken,
	})
	if err != nil {
		logger.Error("failed to subscribe to app uninstalled webhook", "err", err)
		return fmt.Errorf("failed to subscribe to app uninstalled webhook: %w", err)
	}
	logger.Debug("subscribed to webhook")

	logger.Info("marking store as installed", "storeName", opts.StoreName)
	updatedStore, err := s.storages.Store.Update(ctx, &entity.Store{
		Name:        opts.StoreName,
		AccessToken: accessToken,
		Installed:   true,
	})
	if err != nil {
		logger.Error("failed to update store in storage", "err", err)
		return fmt.Errorf("failed to update store in storage: %w", err)
	}
	logger = logger.With("updatedStore", updatedStore)
	logger.Info("successfully marked store as installed", "storeId", updatedStore.ID, "storeName", updatedStore.Name, "installed", updatedStore.Installed)

	return nil
}

func (s *platformService) HandleUninstall(ctx context.Context, storeName string) error {
	logger := s.logger.
		Named("HandleUninstall").
		With("storeName", storeName)

	store, err := s.storages.Store.Get(ctx, storeName)
	if err != nil {
		logger.Error("failed to get store from storage", "err", err)
		return fmt.Errorf("failed to get store from storage: %w", err)
	}
	if store == nil {
		logger.Info("store is not found")
		return ErrHandleUninstallStoreNotFound
	}
	logger = logger.With("store", store)
	logger.Debug("got store")

	err = s.storages.Store.Delete(ctx, storeName)
	if err != nil {
		logger.Error("failed to delete store from storage", "err", err)
		return fmt.Errorf("failed to delete store from storage: %w", err)
	}

	logger.Info("successfully deleted store's config")
	return nil
}

func (s *platformService) CreateProducts(ctx context.Context) error {
	logger := s.logger.Named("CreateProducts").WithContext(ctx)

	output, err := s.apis.Platform.VerifySession(ctx)
	if err != nil {
		logger.Error("failed to verify session", "err", err)
		return fmt.Errorf("failed to verify session: %w", err)
	}
	if !output.IsVerified {
		logger.Info("invalid session")
		return errors.New("invalid session")
	}

	store, err := s.storages.Store.Get(ctx, output.StoreName)
	if err != nil {
		logger.Error("failed to get store from storage", "err", err)
		return fmt.Errorf("failed to get store from storage: %w", err)
	}
	if store == nil {
		logger.Info("store not found, creating new store", "store_name", output.StoreName)
		store, err = s.storages.Store.Create(ctx, &entity.Store{
			Name:      output.StoreName,
			Installed: true,
		})
		if err != nil {
			logger.Error("failed to create store", "err", err)
			return fmt.Errorf("failed to create store: %w", err)
		}
		logger.Info("successfully created store", "storeId", store.ID, "storeName", store.Name)
	}

	// Get session token from context for API calls
	sessionToken := getSessionTokenFromContext(ctx)
	if sessionToken == "" {
		logger.Error("missing session token for API call")
		return fmt.Errorf("missing session token for API call")
	}

	err = s.apis.Platform.WithSessionToken(ctx, store, sessionToken).CreateProducts(ctx)
	if err != nil {
		logger.Error("failed to create products", "err", err)
		return fmt.Errorf("failed to create products: %w", err)
	}

	return nil
}

func (s *platformService) GetProductsCount(ctx context.Context) (int, error) {
	logger := s.logger.Named("GetProductsCount").WithContext(ctx)

	output, err := s.apis.Platform.VerifySession(ctx)
	if err != nil {
		logger.Error("failed to verify session", "err", err)
		return 0, fmt.Errorf("failed to verify session: %w", err)
	}
	if !output.IsVerified {
		logger.Info("invalid session")
		return 0, errors.New("invalid session")
	}

	store, err := s.storages.Store.Get(ctx, output.StoreName)
	if err != nil {
		logger.Error("failed to get store from storage", "err", err)
		return 0, fmt.Errorf("failed to get store from storage: %w", err)
	}
	if store == nil {
		logger.Info("store not found, creating new store", "store_name", output.StoreName)
		store, err = s.storages.Store.Create(ctx, &entity.Store{
			Name:      output.StoreName,
			Installed: true,
		})
		if err != nil {
			logger.Error("failed to create store", "err", err)
			return 0, fmt.Errorf("failed to create store: %w", err)
		}
		logger.Info("successfully created store", "storeId", store.ID, "storeName", store.Name)
	}

	// Get session token from context for API calls
	sessionToken := getSessionTokenFromContext(ctx)
	if sessionToken == "" {
		logger.Error("missing session token for API call")
		return 0, fmt.Errorf("missing session token for API call")
	}

	count, err := s.apis.Platform.WithSessionToken(ctx, store, sessionToken).GetProductsCount(ctx)
	if err != nil {
		logger.Error("failed to get product count", "err", err)
		return 0, fmt.Errorf("failed to get product count: %w", err)
	}

	return count, nil
}

// getSessionTokenFromContext extracts the session token from the authorization header
func getSessionTokenFromContext(ctx context.Context) string {
	authHeader := ctx.Value("Authorization")
	if authHeader == nil {
		return ""
	}

	authStr, ok := authHeader.(string)
	if !ok {
		return ""
	}

	// Extract token from "Bearer <token>" format
	parts := strings.Split(authStr, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
