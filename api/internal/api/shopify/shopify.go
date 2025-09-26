package shopify

import (
	"context"
	"fmt"
	"time"

	"github.com/antflydb/shopify-app-template-go/config"
	"github.com/antflydb/shopify-app-template-go/internal/entity"
	"github.com/antflydb/shopify-app-template-go/internal/service"
	"github.com/antflydb/shopify-app-template-go/pkg/logging"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-retryablehttp"
)

type Options struct {
	Config *config.Config
	Logger logging.Logger
}

var _ service.PlatformAPI = (*shopifyAPI)(nil)

type shopifyAPI struct {
	client  *resty.Client
	logger  logging.Logger
	cfg     *config.Config
	retries int
}

func NewAPI(opts Options) *shopifyAPI {
	restyClient := resty.New()

	return &shopifyAPI{
		logger: opts.Logger.Named("shopifyAPI"),
		cfg:    opts.Config,
		client: restyClient,
	}
}
func (s *shopifyAPI) WithConfig(ctx context.Context, store *entity.Store) service.PlatformAPI {
	var h *resty.Client

	if s.retries != 0 {
		c := retryablehttp.NewClient()
		c.RetryMax = s.retries
		c.RetryWaitMax = time.Second * 30
		h = resty.NewWithClient(c.StandardClient())
	} else {
		h = resty.New()
	}

	h = h.
		SetBaseURL(fmt.Sprintf(`https://%s`, store.Name)).
		SetHeader("X-Shopify-Access-Token", store.AccessToken).
		SetHeader("Content-Type", "application/json")

	return &shopifyAPI{
		client:  h,
		logger:  s.logger,
		cfg:     s.cfg,
		retries: s.retries,
	}
}

func (s *shopifyAPI) WithSessionToken(ctx context.Context, store *entity.Store, sessionToken string) service.PlatformAPI {
	var h *resty.Client

	if s.retries != 0 {
		c := retryablehttp.NewClient()
		c.RetryMax = s.retries
		c.RetryWaitMax = time.Second * 30
		h = resty.NewWithClient(c.StandardClient())
	} else {
		h = resty.New()
	}

	// Exchange session token for access token
	accessToken, err := s.exchangeSessionToken(ctx, store.Name, sessionToken)
	if err != nil {
		s.logger.Error("failed to exchange session token", "err", err)
		// Return client without access token - requests will fail but won't crash
		h = h.
			SetBaseURL(fmt.Sprintf(`https://%s`, store.Name)).
			SetHeader("Content-Type", "application/json")
	} else {
		h = h.
			SetBaseURL(fmt.Sprintf(`https://%s`, store.Name)).
			SetHeader("X-Shopify-Access-Token", accessToken).
			SetHeader("Content-Type", "application/json")
	}

	return &shopifyAPI{
		client:  h,
		logger:  s.logger,
		cfg:     s.cfg,
		retries: s.retries,
	}
}

// exchangeSessionToken exchanges a session token for an access token
// https://shopify.dev/docs/apps/auth/oauth/session-tokens/getting-started#step-3-make-authenticated-requests
func (s *shopifyAPI) exchangeSessionToken(ctx context.Context, storeName, sessionToken string) (string, error) {
	logger := s.logger.Named("exchangeSessionToken").WithContext(ctx)

	type tokenExchangeRequest struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
		SubjectToken string `json:"subject_token"`
		SubjectTokenType string `json:"subject_token_type"`
	}

	type tokenExchangeResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	requestBody := tokenExchangeRequest{
		ClientID:         s.cfg.Shopify.ApiKey,
		ClientSecret:     s.cfg.Shopify.ApiSecret,
		GrantType:        "urn:ietf:params:oauth:grant-type:token-exchange",
		SubjectToken:     sessionToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:id_token",
	}

	var responseBody tokenExchangeResponse

	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(requestBody).
		SetResult(&responseBody).
		Post(fmt.Sprintf("https://%s/admin/oauth/access_token", storeName))

	if err != nil {
		logger.Error("failed to make token exchange request", "err", err)
		return "", fmt.Errorf("failed to exchange session token: %w", err)
	}

	if resp.StatusCode() != 200 {
		logger.Error("token exchange failed", "status", resp.StatusCode(), "response", string(resp.Body()))
		return "", fmt.Errorf("token exchange failed with status %d", resp.StatusCode())
	}

	logger.Info("successfully exchanged session token for access token")
	return responseBody.AccessToken, nil
}
