// internal/models/currency.go
package models

import "time"

// Currency model for the currencies table
type Currency struct {
	ID     int    `json:"id"`
	HK     string `json:"hk"`
	Status string `json:"status"`
	Info   struct {
		FaName string `json:"fa_name"`
		EnName string `json:"en_name"`
		Symbol string `json:"symbol"`
		IfFiat bool   `json:"if_fiat"`
	} `json:"info"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
