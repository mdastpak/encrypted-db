package currencies

import (
	"database/sql"
	"encoding/json"
	"encrypted-db/config"
	"encrypted-db/internal/helpers"
	"encrypted-db/internal/models"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

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
func (h *CurrenciesAdminHandler) UpdateCurrency(c *gin.Context) {
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

	err := h.Handler.Postgres.Pool.QueryRow(
		h.Handler.Ctx,
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
	_, err = h.Handler.Postgres.Pool.Exec(
		h.Handler.Ctx,
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
	if err := h.Handler.RabbitMQService.Publish(config.Config.RabbitMQ.Exchanges.Currency, currencyUpdateMessage); err != nil {
		log.Printf("Failed to publish currency update: %v", err)
	}

	helpers.SendResponse(c, http.StatusOK, "Currency updated successfully.", nil)
}
