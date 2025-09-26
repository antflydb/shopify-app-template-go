package main

import (
	"github.com/antflydb/shopify-app-template-go/config"
	"github.com/antflydb/shopify-app-template-go/internal/app"
	"github.com/antflydb/shopify-app-template-go/pkg/logging"
)

func main() {
	logger := logging.NewZap("main")

	cfg := config.Get()
	logger.Info("read config", "config", cfg)

	app.Run(cfg)
}
