package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"encrypted-db/config"
	"encrypted-db/internal/helpers"
	"encrypted-db/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/net/context"
)

// CreateCurrency godoc
// @Summary Create a new currency
// @Description Create a new currency in the system
// @Tags Admin
// @Accept json
// @Produce json
// @Param currency body models.Currency true "Currency to create"
// @Success 201 {object} models.APIResponse "Currency created successfully"
// @Failure 400 {object} models.APIResponse "Invalid input or duplicate symbol/contract"
// @Failure 500 {object} models.APIResponse "Failed to create currency due to an unexpected error"
// @Failure 0 {object} models.APIResponse "CORS Error: Unauthorized domain access"
// @Failure 0 {object} models.APIResponse "Network Failure: Check your connection"
// @Failure 0 {object} models.APIResponse "URL Scheme Error: Ensure URL scheme is 'http' or 'https'"
// @Router /admin/currencies [post]
func (h *AdminHandler) CreateCurrency(c *gin.Context) {
	var currency models.Currency
	if err := c.ShouldBindJSON(&currency); err != nil {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid input: Please check the provided data format.", nil)
		return
	}

	// Convert CurrencyInfo to JSON for JSONB storage in PostgreSQL
	infoJSON, err := json.Marshal(currency.Info)
	if err != nil {
		log.Printf("Error marshalling CurrencyInfo to JSON: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Internal error: Failed to process currency information.", nil)
		return
	}

	// Insert the currency into the database and return the generated HK
	var generatedHK string
	err = h.PostgresDB.DB.QueryRow(
		"INSERT INTO currencies (status, info) VALUES ($1, $2::jsonb) RETURNING hk",
		currency.Status, infoJSON,
	).Scan(&generatedHK)
	if err != nil {
		// Log detailed error
		log.Printf("Error creating currency in database: %v\n", err)

		// Check if the error is due to unique constraint violation
		if pgError, ok := err.(*pq.Error); ok {
			switch pgError.Code.Name() {
			case "unique_violation":
				helpers.SendResponse(c, http.StatusBadRequest, "Currency with this symbol or contract already exists.", nil)
				return
			case "check_violation":
				helpers.SendResponse(c, http.StatusBadRequest, "Invalid currency structure: Missing required fields.", nil)
				return
			}
		}

		// General error response
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to create currency due to an unexpected error.", nil)
		return
	}

	// Set the generated HK to currency struct
	currency.HK = generatedHK

	// Inside CreateCurrency and UpdateCurrency, after successful database insert/update
	err = h.AddOrUpdateCurrencyInCache(currency)
	if err != nil {
		log.Printf("Error caching currency after database update: %v\n", err)
	}

	// Publish the update to RabbitMQ for WebSocket listeners
	currencyCreateMessage := fmt.Sprintf(`{"entity": "currencies", "action": "create", "hk": "%s", "info": %s}`, generatedHK, string(infoJSON))
	if err := h.RabbitMQService.Publish(config.Config.RabbitMQ.Exchanges.Currency, currencyCreateMessage); err != nil {
		log.Printf("Failed to publish currency create: %v", err)
	}

	helpers.SendResponse(c, http.StatusCreated, "Currency created successfully.", nil)
}

// UpdateCurrency godoc
// @Summary Update a currency
// @Description Update the information or status of an existing currency using its HK (UUID) as identifier
// @Tags Admin
// @Accept json
// @Produce json
// @Param hk path string true "Currency HK"
// @Param currency body models.Currency true "Currency data to update"
// @Success 200 {object} models.APIResponse "Currency updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid input or duplicate symbol/contract"
// @Failure 404 {object} models.APIResponse "Currency not found"
// @Failure 500 {object} models.APIResponse "Failed to update currency due to an unexpected error"
// @Router /admin/currencies/{hk} [put]
func (h *AdminHandler) UpdateCurrency(c *gin.Context) {
	hk := c.Param("hk")

	// Log and validate HK as a UUID
	log.Printf("Received HK value: %s\n", hk)
	if _, err := uuid.Parse(hk); err != nil {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid UUID format for HK", nil)
		return
	}

	var updatedData models.Currency
	if err := c.ShouldBindJSON(&updatedData); err != nil {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid input: Please check the provided data format.", nil)
		return
	}
	updatedData.HK = hk

	// Retrieve the current data for the currency
	var currentStatus string
	var infoData []byte
	err := h.PostgresDB.DB.QueryRow(
		"SELECT status, info FROM currencies WHERE hk = $1 AND deleted_at IS NULL", hk,
	).Scan(&currentStatus, &infoData)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.SendResponse(c, http.StatusNotFound, "Currency not found", nil)
			return
		}
		log.Printf("Error retrieving currency data: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to retrieve currency data.", nil)
		return
	}

	// Unmarshal the info JSONB data into the CurrencyInfo struct
	var currentInfo models.CurrencyInfo
	if err := json.Unmarshal(infoData, &currentInfo); err != nil {
		log.Printf("Error unmarshalling CurrencyInfo: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to process currency information.", nil)
		return
	}

	// Merge new data with existing data
	if updatedData.Status != "" {
		currentStatus = updatedData.Status
	}
	if updatedData.Info.EnName != "" {
		currentInfo.EnName = updatedData.Info.EnName
	}
	if updatedData.Info.FaName != "" {
		currentInfo.FaName = updatedData.Info.FaName
	}
	if updatedData.Info.Symbol != "" {
		currentInfo.Symbol = updatedData.Info.Symbol
	}
	if updatedData.Info.Contract != "" {
		currentInfo.Contract = updatedData.Info.Contract
	}
	if updatedData.Info.IfFiat {
		currentInfo.IfFiat = updatedData.Info.IfFiat
	}
	if updatedData.Info.IsToken {
		currentInfo.IsToken = updatedData.Info.IsToken
	}

	// Convert merged data to JSON for JSONB storage
	infoJSON, err := json.Marshal(currentInfo)
	if err != nil {
		log.Printf("Error marshalling CurrencyInfo to JSON: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Internal error: Failed to process currency information.", nil)
		return
	}

	// Update the currency in the database
	_, err = h.PostgresDB.DB.Exec(
		"UPDATE currencies SET status=$1, info=$2::jsonb, updated_at=NOW() WHERE hk=$3 AND deleted_at IS NULL",
		currentStatus, infoJSON, hk,
	)
	if err != nil {
		log.Printf("Error updating currency in database: %v\n", err)

		if pgError, ok := err.(*pq.Error); ok {
			switch pgError.Code.Name() {
			case "unique_violation":
				helpers.SendResponse(c, http.StatusBadRequest, "Currency with this symbol or contract already exists.", nil)
				return
			case "check_violation":
				helpers.SendResponse(c, http.StatusBadRequest, "Invalid currency structure: Missing required fields.", nil)
				return
			}
		}

		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to update currency due to an unexpected error.", nil)
		return
	}

	// Inside CreateCurrency and UpdateCurrency, after successful database insert/update
	err = h.AddOrUpdateCurrencyInCache(updatedData)
	if err != nil {
		log.Printf("Error caching currency after database update: %v\n", err)
	}

	// Publish the update to RabbitMQ for WebSocket listeners
	currencyUpdateMessage := fmt.Sprintf(`{"entity": "currencies", "action": "update", "hk": "%s", "info": %s}`, hk, string(infoJSON))
	if err := h.RabbitMQService.Publish(config.Config.RabbitMQ.Exchanges.Currency, currencyUpdateMessage); err != nil {
		log.Printf("Failed to publish currency update: %v", err)
	}

	helpers.SendResponse(c, http.StatusOK, "Currency updated successfully.", nil)
}

// DeleteCurrency godoc
// @Summary Delete a currency
// @Description Logically delete a currency by setting deleted_at and changing its status to 'deleted'.
// @Tags Admin
// @Param hk path string true "Currency HK"
// @Success 200 {object} models.APIResponse "Currency deleted successfully"
// @Failure 404 {object} models.APIResponse "Currency not found or already deleted"
// @Failure 500 {object} models.APIResponse "Failed to delete currency due to an unexpected error"
// @Router /admin/currencies/{hk} [delete]
func (h *AdminHandler) DeleteCurrency(c *gin.Context) {
	hk := c.Param("hk")

	// Validate if HK is a valid UUID
	if _, err := uuid.Parse(hk); err != nil {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid UUID format for HK", nil)
		return
	}

	// Check if the currency exists and is not already deleted
	var isDeleted bool
	err := h.PostgresDB.DB.QueryRow(
		"SELECT (deleted_at IS NOT NULL) FROM currencies WHERE hk = $1", hk,
	).Scan(&isDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			helpers.SendResponse(c, http.StatusNotFound, "Currency not found", nil)
		} else {
			log.Printf("Error checking currency in database: %v\n", err)
			helpers.SendResponse(c, http.StatusInternalServerError, "Failed to delete currency due to an unexpected error.", nil)
		}
		return
	}

	// If the currency is already deleted, return a 404 error
	if isDeleted {
		helpers.SendResponse(c, http.StatusNotFound, "Currency not found or already deleted", nil)
		return
	}

	// Proceed with logical deletion: set deleted_at and update status to 'deleted'
	_, err = h.PostgresDB.DB.Exec(
		"UPDATE currencies SET deleted_at=NOW(), status='deleted' WHERE hk=$1", hk,
	)
	if err != nil {
		log.Printf("Error deleting currency in database: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to delete currency due to an unexpected error.", nil)
		return
	}

	// Inside DeleteCurrency, after successful database deletion
	err = h.DeleteCurrencyFromCache(hk)
	if err != nil {
		log.Printf("Error deleting currency from cache after database deletion: %v\n", err)
	}

	// Publish the delete to RabbitMQ for WebSocket listeners
	currencyDeleteMessage := fmt.Sprintf(`{"entity": "currencies", "action": "delete", "hk": "%s", "info": {"status": "deleted"}}`, hk)
	if err := h.RabbitMQService.Publish(config.Config.RabbitMQ.Exchanges.Currency, currencyDeleteMessage); err != nil {
		log.Printf("Failed to publish currency delete: %v", err)
	}

	helpers.SendResponse(c, http.StatusOK, "Currency deleted successfully.", nil)
}

// CleanupBaseDefinitions removes all currency-related base definition entries from Redis
func (h *AdminHandler) CleanupBaseDefinitions() error {
	// Define the pattern to match all currency keys
	pattern := "base_definitions:currency:*"

	// Use Redis SCAN command to iterate over matching keys
	iter := h.RedisClient.Client.Scan(context.Background(), 0, pattern, 0).Iterator()
	for iter.Next(context.Background()) {
		// For each matching key, delete it from Redis
		if err := h.RedisClient.Client.Del(context.Background(), iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete key %s from Redis: %v\n", iter.Val(), err)
			return err
		}
	}

	// Check if iteration encountered any errors
	if err := iter.Err(); err != nil {
		log.Printf("Error during Redis key iteration: %v\n", err)
		return err
	}

	log.Println("All base definitions for currencies have been removed from Redis.")
	return nil
}

// AddOrUpdateCurrencyInCache adds or updates a currency in Redis cache
func (h *AdminHandler) AddOrUpdateCurrencyInCache(currency models.Currency) error {
	// Convert currency data to JSON for storage in Redis
	data, err := json.Marshal(currency)
	if err != nil {
		log.Printf("Error marshalling currency data for cache: %v\n", err)
		return err
	}

	// Use a specific key for each currency based on hk
	cacheKey := fmt.Sprintf("base_definitions:currency:%s", currency.HK)

	// Set data in Redis with no expiration
	err = h.RedisClient.Client.Set(context.Background(), cacheKey, data, 0).Err()
	if err != nil {
		log.Printf("Error updating currency in cache: %v\n", err)
		return err
	}

	log.Printf("Currency with HK %s has been added/updated in cache.", currency.HK)
	return nil
}

// DeleteCurrencyFromCache removes a currency from Redis cache by its HK
func (h *AdminHandler) DeleteCurrencyFromCache(hk string) error {
	// Define the cache key based on HK
	cacheKey := fmt.Sprintf("base_definitions:currency:%s", hk)

	// Delete the key from Redis
	err := h.RedisClient.Client.Del(context.Background(), cacheKey).Err()
	if err != nil {
		log.Printf("Error deleting currency from cache: %v\n", err)
		return err
	}

	log.Printf("Currency with HK %s has been deleted from cache.", hk)
	return nil
}

// LoadAndCacheCurrencies loads all active currencies from the database and caches them in Redis
func (h *AdminHandler) LoadAndCacheCurrencies() error {
	rows, err := h.PostgresDB.DB.Query("SELECT id, hk, status, info FROM currencies WHERE status='approved' AND deleted_at IS NULL")
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
