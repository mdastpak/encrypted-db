// internal/handlers/public/currencies.go
package public

import (
	"encoding/json"
	"net/http"

	"encrypted-db/internal/db"
	"encrypted-db/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

// CurrenciesPublicHandler struct to hold dependencies for public routes
type CurrenciesPublicHandler struct {
	PostgresDB  *db.PostgresService
	RedisClient *db.RedisService
}

// NewHandler function to initialize CurrenciesPublicHandler with dependencies
func NewHandler(postgres *db.PostgresService, redis *db.RedisService) *CurrenciesPublicHandler {
	return &CurrenciesPublicHandler{
		PostgresDB:  postgres,
		RedisClient: redis,
	}
}

// GetActiveCurrencies godoc
// @Summary Get all active currencies
// @Description Retrieve all active currencies from the system
// @Tags Public
// @Produce json
// @Success 200 {array} models.Currency
// @Failure 500 {object} map[string]interface{}
// @Router /public/currencies [get]
func (h *CurrenciesPublicHandler) GetActiveCurrencies(c *gin.Context) {
	// Attempt to retrieve data from Redis cache
	result, err := h.RedisClient.Client.Get(context.Background(), "active_currencies").Result()
	if err == redis.Nil || result == "" {
		// Cache miss - Load from PostgreSQL
		rows, err := h.PostgresDB.DB.Query("SELECT id, hk, status, info FROM currencies WHERE status='approved' AND deleted_at IS NULL")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch currencies"})
			return
		}
		defer rows.Close()

		var currencies []models.Currency
		for rows.Next() {
			var currency models.Currency
			if err := rows.Scan(&currency.ID, &currency.HK, &currency.Status, &currency.Info); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse currencies"})
				return
			}
			currencies = append(currencies, currency)
		}

		// Cache data in Redis
		data, _ := json.Marshal(currencies)
		h.RedisClient.Client.Set(context.Background(), "active_currencies", data, 0)
		c.JSON(http.StatusOK, currencies)
		return
	}

	// Cache hit - Return data from Redis
	var currencies []models.Currency
	if err := json.Unmarshal([]byte(result), &currencies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse cache data"})
		return
	}
	c.JSON(http.StatusOK, currencies)
}
