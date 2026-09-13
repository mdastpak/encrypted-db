package currencies

import (
	"encoding/json"
	"encrypted-db/config"
	"encrypted-db/internal/helpers"
	"encrypted-db/internal/models"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
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
func (h *CurrenciesAdminHandler) CreateCurrency(c *gin.Context) {
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
	err = h.Handler.Postgres.Pool.QueryRow(
		h.Handler.Ctx,
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
	if err := h.Handler.RabbitMQService.Publish(config.Config.RabbitMQ.Exchanges.Currency, currencyCreateMessage); err != nil {
		log.Printf("Failed to publish currency create: %v", err)
	}

	helpers.SendResponse(c, http.StatusCreated, "Currency created successfully.", nil)
}
