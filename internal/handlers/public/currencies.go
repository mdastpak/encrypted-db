package public

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"encrypted-db/internal/db"
	"encrypted-db/internal/helpers"
	"encrypted-db/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

// CurrenciesPublicHandler struct to hold dependencies for public routes
type CurrenciesPublicHandler struct {
	PostgresDB  *db.PostgresService
	RedisClient *db.RedisService
	RedisIndex  string // Redis index for base definitions
}

// NewHandler function to initialize CurrenciesPublicHandler with dependencies
func NewHandler(postgres *db.PostgresService, redis *db.RedisService) *CurrenciesPublicHandler {
	return &CurrenciesPublicHandler{
		PostgresDB:  postgres,
		RedisClient: redis,
		RedisIndex:  "base_definitions", // Setting index for base definitions
	}
}

// CurrencyResponse represents the custom output format for each currency
type CurrencyResponse struct {
	HK     string              `json:"hk"`
	Info   models.CurrencyInfo `json:"info"`
	Status string              `json:"status"`
}

// GetActiveCurrencies godoc
// @Summary Get all active currencies
// @Description Retrieve all active currencies from Redis cache
// @Tags Public
// @Produce json
// @Success 200 {array} models.Currency "List of active currencies"
// @Failure 500 {object} models.APIResponse "Failed to retrieve currencies from cache"
// @Router /public/currencies [get]
func (h *CurrenciesPublicHandler) GetActiveCurrencies(c *gin.Context) {
	// Define the pattern to match all active currency keys
	pattern := "base_definitions:currency:*"

	// Use Redis SCAN command to retrieve all matching keys
	var currencies []models.Currency
	iter := h.RedisClient.Client.Scan(context.Background(), 0, pattern, 0).Iterator()
	for iter.Next(context.Background()) {
		// Get the value of each currency key from Redis
		data, err := h.RedisClient.Client.Get(context.Background(), iter.Val()).Result()
		if err != nil {
			log.Printf("Error retrieving currency from cache for key %s: %v\n", iter.Val(), err)
			helpers.SendResponse(c, http.StatusInternalServerError, "Failed to retrieve currencies from cache.", nil)
			return
		}

		// Unmarshal the JSON data into a Currency struct
		var currency models.Currency
		if err := json.Unmarshal([]byte(data), &currency); err != nil {
			log.Printf("Error unmarshalling currency data for key %s: %v\n", iter.Val(), err)
			helpers.SendResponse(c, http.StatusInternalServerError, "Failed to process currency information.", nil)
			return
		}

		// Append the currency to the list
		currencies = append(currencies, currency)
	}

	// Check for any error during key iteration
	if err := iter.Err(); err != nil {
		log.Printf("Error during Redis key iteration: %v\n", err)
		helpers.SendResponse(c, http.StatusInternalServerError, "Failed to retrieve currencies from cache.", nil)
		return
	}

	// Send the list of active currencies as a response
	helpers.SendResponse(c, http.StatusOK, "Active currencies retrieved successfully.", currencies)
}

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
	data, err := h.RedisClient.Client.Get(context.Background(), cacheKey).Result()
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
