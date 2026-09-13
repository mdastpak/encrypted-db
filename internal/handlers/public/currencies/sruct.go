package currencies

import "encrypted-db/internal/models"

// CurrencyResponse represents the custom output format for each currency
type CurrencyResponse struct {
	HK     string              `json:"hk"`
	Info   models.CurrencyInfo `json:"info"`
	Status string              `json:"status"`
}
