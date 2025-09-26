package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/pkg/errs"
)

type platformRoutes struct {
	RouterContext
}

// bindQuery binds query parameters to a struct
func bindQuery(values url.Values, target any) error {
	// Simple implementation - in production you might want to use a library like gorilla/schema
	switch v := target.(type) {
	case *handlerRequestQuery:
		v.StoreName = values.Get("shop")
		if v.StoreName == "" {
			return fmt.Errorf("required parameter 'shop' is missing")
		}
	case *redirectHandlerRequestQuery:
		v.StoreName = values.Get("shop")
		if v.StoreName == "" {
			return fmt.Errorf("required parameter 'shop' is missing")
		}
	case *uninstallHandlerRequestQuery:
		v.StoreName = values.Get("shop")
		if v.StoreName == "" {
			return fmt.Errorf("required parameter 'shop' is missing")
		}
	}
	return nil
}

func newPlatformRoutes(options RouterOptions) {
	r := &platformRoutes{RouterContext{
		services: options.Services,
		storages: options.Storages,
		logger:   options.Logger.Named("platformRoutes"),
		cfg:      options.Config,
	}}

	options.Handler.HandleFunc("GET /", wrapHandler(options, r.handler))
	options.Handler.HandleFunc("GET /auth/callback", wrapHandler(options, r.redirectHandler))
	options.Handler.HandleFunc("POST /uninstall", wrapHandler(options, r.uninstallHandler))
	options.Handler.HandleFunc("GET /api/products/count", wrapHandler(options, r.getProductsCount))
	options.Handler.HandleFunc("GET /api/products/create", wrapHandler(options, r.createProducts))
}

type handlerRequestQuery struct {
	StoreName string `form:"shop" binding:"required"`
}

func (r *platformRoutes) handler(c *RequestContext) (any, *httpErr) {
	logger := r.logger.Named("handler").WithContext(c.Context())

	var requestQuery handlerRequestQuery
	err := bindQuery(c.Request.URL.Query(), &requestQuery)
	if err != nil {
		logger.Info("failed to parse request query", "err", err)
		return nil, &httpErr{Type: ErrorTypeClient, Message: "invalid request query", Details: err}
	}
	logger = logger.With("requestQuery", requestQuery)

	redirectURL, err := r.services.Platform.Handle(c.Context(), requestQuery.StoreName, c.Request.URL.String())
	if err != nil {
		if errs.IsExpected(err) {
			logger.Info(err.Error())
			return nil, &httpErr{Type: ErrorTypeClient, Message: err.Error()}
		}
		logger.Error("failed to handle call", "err", err)
		return nil, &httpErr{
			Type:    ErrorTypeServer,
			Message: "failed to handle call",
			Details: err,
		}
	}
	logger = logger.With("redirectURL", redirectURL)

	c.Redirect(http.StatusFound, redirectURL)

	logger.Info("successfully handled call")
	return nil, nil
}

type redirectHandlerRequestQuery struct {
	StoreName string `form:"shop" binding:"required"`
}

func (r *platformRoutes) redirectHandler(c *RequestContext) (any, *httpErr) {
	logger := r.logger.Named("redirectHandler").WithContext(c.Context())

	var requestQuery redirectHandlerRequestQuery
	err := bindQuery(c.Request.URL.Query(), &requestQuery)
	if err != nil {
		logger.Info("failed to parse request query", "err", err)
		return nil, &httpErr{Type: ErrorTypeClient, Message: "invalid request query", Details: err}
	}
	logger = logger.With("requestQuery", requestQuery)

	err = r.services.Platform.HandleRedirect(c.Context(), service.ServiceHandleRedirectOptions{
		StoreName:     requestQuery.StoreName,
		RedirectedURL: c.Request.URL.String(),
	})
	if err != nil {
		if errs.IsExpected(err) {
			logger.Info(err.Error())
			return nil, &httpErr{Type: ErrorTypeClient, Message: err.Error()}
		}
		logger.Error("failed to handle oauth2 redirect call", "err", err)
		return nil, &httpErr{
			Type:    ErrorTypeServer,
			Message: "failed to handle oauth2 redirect call",
			Details: err,
		}
	}
	// After successful handling of redirect call, redirect user to app's UI at their platform store
	redirectURL := fmt.Sprintf("https://%s/admin/apps/%s", requestQuery.StoreName, r.cfg.Shopify.ApiKey)
	logger.Info("redirecting to app UI", "redirectURL", redirectURL, "apiKey", r.cfg.Shopify.ApiKey)
	c.Redirect(http.StatusFound, redirectURL)

	logger.Info("successfully handled redirect call")
	return nil, nil
}

type uninstallHandlerRequestQuery struct {
	StoreName string `form:"shop" binding:"required"`
}

func (r *platformRoutes) uninstallHandler(c *RequestContext) (any, *httpErr) {
	logger := r.logger.
		Named("uninstallHandler").
		WithContext(c.Context())

	var requestQuery uninstallHandlerRequestQuery
	err := bindQuery(c.Request.URL.Query(), &requestQuery)
	if err != nil {
		logger.Info("failed to parse request query", "err", err)
		return nil, &httpErr{Type: ErrorTypeClient, Message: "invalid request query", Details: err}
	}
	logger = logger.With("requestQuery", requestQuery)

	err = r.services.Platform.HandleUninstall(c.Context(), requestQuery.StoreName)
	if err != nil {
		if errs.IsExpected(err) {
			logger.Info(err.Error())
			return nil, &httpErr{Type: ErrorTypeClient, Message: err.Error()}
		}
		logger.Error("failed to uninstall app", "err", err)
		return nil, &httpErr{
			Type:    ErrorTypeServer,
			Message: "failed to failed to uninstall app",
			Details: err,
		}
	}

	logger.Info("successfully uninstalled the application")
	return nil, nil
}

type getProductsCountResponse struct {
	Count int `json:"count"`
}

func (r *platformRoutes) getProductsCount(c *RequestContext) (any, *httpErr) {
	logger := r.logger.Named("getProductsCount")

	// Set authorization in context - create a new context with the auth header
	ctx := c.Context()
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		ctx = context.WithValue(ctx, "Authorization", auth)
		c.WithContext(ctx)
	}

	count, err := r.services.Platform.GetProductsCount(c.Context())
	if err != nil {
		// TODO: return custom errors to client, instead of 500
		logger.Error("failed to get products count", "err", err)
		return nil, &httpErr{
			Type:    ErrorTypeServer,
			Message: "failed to get products count",
			Details: err,
		}
	}
	logger = logger.With("count", count)

	logger.Info("successfully got products count")
	return getProductsCountResponse{Count: count}, nil
}

func (r *platformRoutes) createProducts(c *RequestContext) (any, *httpErr) {
	logger := r.logger.Named("createProducts")

	// Set authorization in context - create a new context with the auth header
	ctx := c.Context()
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		ctx = context.WithValue(ctx, "Authorization", auth)
		c.WithContext(ctx)
	}

	err := r.services.Platform.CreateProducts(c.Context())
	if err != nil {
		// TODO: return custom errors to client, instead of 500
		logger.Error("failed to create products", "err", err)
		return nil, &httpErr{
			Type:    ErrorTypeServer,
			Message: "failed to create products",
			Details: err,
		}
	}

	logger.Info("successfully created products")
	return "", nil
}
