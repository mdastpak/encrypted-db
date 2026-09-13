package currencies

import (
	"context"
	"encoding/json"
	"encrypted-db/internal/helpers"
	"encrypted-db/internal/models"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// GetCurrencyByHK godoc
// @Summary Get a single currency by HK
// @Description Retrieve a single currency from Redis cache using its HK
// @Tags Public
// @Param hk path string true "Currency HK"
// @Produce json
// @Success 200 {object} models.Currency "Currency data"
// @Failure 404 {object} models.APIResponse "Currency not found"
// @Failure 500 {object} models.APIResponse "Failed to retrieve currency from cache"
// @Router /public/currencies/{hk} [get]
func (h *CurrenciesPublicHandler) GetCurrencyByHK(c *gin.Context) {
	// Retrieve the HK from the URL
	hk := c.Param("hk")

	// Define the cache key for the specific currency
	cacheKey := fmt.Sprintf("base_definitions:currency:%s", hk)

	// Attempt to retrieve the currency data from Redis
	data, err := h.Handler.Redis.Client.Get(context.Background(), cacheKey).Result()
	if err == redis.Nil {
		// If the currency is not found in the cache, return a 404 error
		helpers.SendResponse(c, http.StatusNotFound, "Currency not found", nil)
		return
	} else if err != nil {
		// If there is any other error, return a 500 error
		log.Printf("Error retrieving currency from cache for HK %s: %v\n", hk, err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to retrieve currency from cache.", nil)
		return
	}

	// Unmarshal the JSON data into a Currency struct
	var currency models.Currency
	if err := json.Unmarshal([]byte(data), &currency); err != nil {
		log.Printf("Error unmarshalling currency data for HK %s: %v\n", hk, err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to process currency information.", nil)
		return
	}

	// Return the currency data as a response
	helpers.SendResponse(c, http.StatusOK, "Currency retrieved successfully.", currency)
}
