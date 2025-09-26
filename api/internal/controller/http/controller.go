package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/DataDog/gostackparse"
	"github.com/antflydb/shopify-app-template-go/config"
	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/pkg/logging"
)

// Options is used to create HTTP controller.
type Options struct {
	Handler  *http.ServeMux
	Services service.Services
	Storages service.Storages
	Logger   logging.Logger
	Config   *config.Config
}

// RouterOptions provides shared options for all routers.
type RouterOptions struct {
	Handler  *http.ServeMux
	Services service.Services
	Storages service.Storages
	Logger   logging.Logger
	Config   *config.Config
}

// RouterContext provides a shared context for all routers.
type RouterContext struct {
	services service.Services
	storages service.Storages
	logger   logging.Logger
	cfg      *config.Config
}

func New(options *Options) {
	routerOptions := RouterOptions{
		Handler:  options.Handler,
		Services: options.Services,
		Storages: options.Storages,
		Logger:   options.Logger.Named("HTTPController"),
		Config:   options.Config,
	}

	// K8S probe
	options.Handler.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Routers
	{
		newPlatformRoutes(routerOptions)
	}
}

// httpErr provides a base error type for all http controller errors.
type httpErr struct {
	Type             httpErrType    `json:"-"`
	Code             int            `json:"-"`
	Message          string         `json:"message"`
	Details          any            `json:"details,omitempty"`
	ValidationErrors map[string]any `json:"validationErrors,omitempty"`
}

// httpErrType is used to define error type.
type httpErrType string

const (
	// ErrorTypeServer is an "unexpected" internal server error.
	ErrorTypeServer httpErrType = "server"
	// ErrorTypeClient is an "expected" business error.
	ErrorTypeClient httpErrType = "client"
)

// Error is used to convert an error to a string.
func (e *httpErr) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// RequestContext provides context for HTTP handlers
type RequestContext struct {
	Request  *http.Request
	Writer   http.ResponseWriter
	Logger   logging.Logger
	Config   *config.Config
	Services service.Services
	Storages service.Storages
	ctx      context.Context
}

// Context returns the request context
func (r *RequestContext) Context() context.Context {
	return r.ctx
}

// WithContext sets the context
func (r *RequestContext) WithContext(ctx context.Context) {
	r.ctx = ctx
}

// JSON writes JSON response
func (r *RequestContext) JSON(status int, data any) error {
	r.Writer.Header().Set("Content-Type", "application/json")
	r.Writer.WriteHeader(status)
	return json.NewEncoder(r.Writer).Encode(data)
}

// Redirect sends an HTTP redirect response
func (r *RequestContext) Redirect(status int, url string) {
	http.Redirect(r.Writer, r.Request, url, status)
}

// wrapHandler provides unified error handling for all handlers.
func wrapHandler(options RouterOptions, handler func(r *RequestContext) (any, *httpErr)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := options.Logger.Named("wrapHandler")

		// Add CORS middleware
		corsMiddleware(w, r)

		// handle panics
		defer func() {
			if err := recover(); err != nil {
				// get stacktrace
				stacktrace, errors := gostackparse.Parse(bytes.NewReader(debug.Stack()))
				if len(errors) > 0 || len(stacktrace) == 0 {
					logger.Error("get stacktrace errors", "stacktraceErrors", errors, "stacktrace", "unknown", "err", err)
				} else {
					logger.Error("unhandled error", "err", err, "stacktrace", stacktrace)
				}

				// return error
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("Internal server error: %v", err)
			}
		}()

		reqCtx := &RequestContext{
			Request:  r,
			Writer:   w,
			Logger:   logger,
			Config:   options.Config,
			Services: options.Services,
			Storages: options.Storages,
			ctx:      r.Context(),
		}

		// execute handler
		body, err := handler(reqCtx)

		// check if middleware
		if body == nil && err == nil {
			return
		}
		logger = logger.With("body", body).With("err", err)

		// check error
		if err != nil {
			if err.Type == ErrorTypeServer {
				logger.Error("internal server error")

				// whether to send error to the client
				if options.Config.HTTP.SendDetailsOnInternalError {
					// send error to the client
					reqCtx.JSON(http.StatusInternalServerError, err)
				} else {
					// don't send error to the client
					w.WriteHeader(http.StatusInternalServerError)
					logger.Info("aborted with error")
				}
			} else {
				logger.Info("client error")
				reqCtx.JSON(http.StatusUnprocessableEntity, err)
			}
			return
		}
		logger.Info("request handled")
		reqCtx.JSON(http.StatusOK, body)
	}
}

// corsMiddleware is used to allow incoming cross-origin requests.
func corsMiddleware(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
}
