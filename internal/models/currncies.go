// internal/models/currency.go
package models

// CurrencyInfo defines the structure of the `info` field in the Currency model
type CurrencyInfo struct {
	EnName   string `json:"en_name"`
	FaName   string `json:"fa_name"`
	IfFiat   bool   `json:"if_fiat"`
	Symbol   string `json:"symbol"`
	IsToken  bool   `json:"is_token"`
	Contract string `json:"contract"`
}

// Currency defines the structure of the currency model
type Currency struct {
	ID        int          `json:"id"`
	HK        string       `json:"hk"`
	Status    string       `json:"status"`
	Info      CurrencyInfo `json:"info"`
	CreatedAt string       `json:"created_at"`
	UpdatedAt string       `json:"updated_at"`
	DeletedAt *string      `json:"deleted_at,omitempty"`
}
