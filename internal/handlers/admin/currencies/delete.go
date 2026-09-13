package currencies

import (
	"context"
	"database/sql"
	"encrypted-db/config"
	"encrypted-db/internal/helpers"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeleteCurrency godoc
// @Summary Delete a currency
// @Description Logically delete a currency by setting deleted_at and changing its status to 'deleted'.
// @Tags Admin
// @Param hk path string true "Currency HK"
// @Success 200 {object} models.APIResponse "Currency deleted successfully"
// @Failure 404 {object} models.APIResponse "Currency not found or already deleted"
// @Failure 500 {object} models.APIResponse "Failed to delete currency due to an unexpected error"
// @Router /admin/currencies/{hk} [delete]
func (h *CurrenciesAdminHandler) DeleteCurrency(c *gin.Context) {
	hk := c.Param("hk")

	// Validate if HK is a valid UUID
	if _, err := uuid.Parse(hk); err != nil {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid UUID format for HK", nil)
		return
	}

	// Check if the currency exists and is not already deleted
	var isDeleted bool
	err := h.Handler.Postgres.Pool.QueryRow(
		h.Handler.Ctx,
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
	_, err = h.Handler.Postgres.Pool.Exec(
		h.Handler.Ctx,
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
	if err := h.Handler.RabbitMQService.Publish(config.Config.RabbitMQ.Exchanges.Currency, currencyDeleteMessage); err != nil {
		log.Printf("Failed to publish currency delete: %v", err)
	}

	helpers.SendResponse(c, http.StatusOK, "Currency deleted successfully.", nil)
}

// CleanupBaseDefinitions removes all currency-related base definition entries from Redis
func (h *CurrenciesAdminHandler) CleanupBaseDefinitions() error {
	// Define the pattern to match all currency keys
	pattern := "base_definitions:currency:*"

	// Use Redis SCAN command to iterate over matching keys
	iter := h.Handler.Redis.Client.Scan(context.Background(), 0, pattern, 0).Iterator()
	for iter.Next(context.Background()) {
		// For each matching key, delete it from Redis
		if err := h.Handler.Redis.Client.Del(context.Background(), iter.Val()).Err(); err != nil {
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
