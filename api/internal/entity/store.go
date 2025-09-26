package entity

import (
	"github.com/antflydb/shopify-app-template-go/pkg/database"
)

// Store model represents model of platform store.
type Store struct {
	database.Model
	ID   string `json:"id"`
	Name string `json:"name"`

	// Shopify
	Nonce       string `json:"nonce"`
	AccessToken string `json:"access_token"`
	Installed   bool   `json:"installed"`
}

type Session struct {
	SessionID string `json:"session_id"`
	StoreID   string `json:"store_id"`
}
