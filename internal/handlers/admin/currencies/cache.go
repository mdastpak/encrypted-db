package currencies

import (
	"context"
	"encoding/json"
	"encrypted-db/internal/models"
	"fmt"
	"log"
)

// AddOrUpdateCurrencyInCache adds or updates a currency in Redis cache
func (h *CurrenciesAdminHandler) AddOrUpdateCurrencyInCache(currency models.Currency) error {
	// Convert currency data to JSON for storage in Redis
	data, err := json.Marshal(currency)
	if err != nil {
		log.Printf("Error marshalling currency data for cache: %v\n", err)
		return err
	}

	// Use a specific key for each currency based on hk
	cacheKey := fmt.Sprintf("base_definitions:currency:%s", currency.HK)

	// Set data in Redis with no expiration
	err = h.Handler.Redis.Client.Set(context.Background(), cacheKey, data, 0).Err()
	if err != nil {
		log.Printf("Error updating currency in cache: %v\n", err)
		return err
	}

	log.Printf("Currency with HK %s has been added/updated in cache.", currency.HK)
	return nil
}

// DeleteCurrencyFromCache removes a currency from Redis cache by its HK
func (h *CurrenciesAdminHandler) DeleteCurrencyFromCache(hk string) error {
	// Define the cache key based on HK
	cacheKey := fmt.Sprintf("base_definitions:currency:%s", hk)

	// Delete the key from Redis
	err := h.Handler.Redis.Client.Del(context.Background(), cacheKey).Err()
	if err != nil {
		log.Printf("Error deleting currency from cache: %v\n", err)
		return err
	}

	log.Printf("Currency with HK %s has been deleted from cache.", hk)
	return nil
}

// LoadAndCacheCurrencies loads all active currencies from the database and caches them in Redis
func (h *CurrenciesAdminHandler) LoadAndCacheCurrencies() error {
	rows, err := h.Handler.Postgres.Pool.Query(
		h.Handler.Ctx,
		"SELECT id, hk, status, info FROM currencies WHERE status='approved' AND deleted_at IS NULL",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var currency models.Currency
		var infoData []byte // Temporary variable to hold the JSONB data

		// Scan each column into respective fields
		if err := rows.Scan(&currency.ID, &currency.HK, &currency.Status, &infoData); err != nil {
			return err
		}

		// Unmarshal the info JSONB data into the CurrencyInfo struct
		if err := json.Unmarshal(infoData, &currency.Info); err != nil {
			return err
		}

		// Add each currency to cache
		if err := h.AddOrUpdateCurrencyInCache(currency); err != nil {
			return err
		}
	}

	log.Println("All active currencies have been loaded and cached successfully.")
	return nil
}
