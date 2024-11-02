// internal/handlers/admin/currencies.go
package admin

import (
	"encoding/json"
	"net/http"

	"encrypted-db/internal/db"
	"encrypted-db/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"golang.org/x/net/context"
)

// CurrenciesAdminHandler struct to hold dependencies for admin routes
type CurrenciesAdminHandler struct {
	PostgresDB  *db.PostgresService
	RedisClient *db.RedisService
}

// NewHandler function to initialize CurrenciesAdminHandler with dependencies
func NewHandler(postgres *db.PostgresService, redis *db.RedisService) *CurrenciesAdminHandler {
	return &CurrenciesAdminHandler{
		PostgresDB:  postgres,
		RedisClient: redis,
	}
}

// CreateCurrency godoc
// @Summary Create a new currency
// @Description Create a new currency in the system
// @Tags Admin
// @Accept json
// @Produce json
// @Param currency body models.Currency true "Currency to create"
// @Success 201 {object} models.Currency
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/currencies [post]
func (h *CurrenciesAdminHandler) CreateCurrency(c *gin.Context) {
	var currency models.Currency
	if err := c.ShouldBindJSON(&currency); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := h.PostgresDB.DB.Exec(
		"INSERT INTO currencies (status, info) VALUES ($1, $2::jsonb)",
		currency.Status, currency.Info,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create currency"})
		return
	}

	h.updateRedisCache()
	c.JSON(http.StatusCreated, gin.H{"message": "Currency created"})
}

// UpdateCurrency godoc
// @Summary Update a currency
// @Description Update the information or status of an existing currency
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Currency ID"
// @Param currency body models.Currency true "Currency data to update"
// @Success 200 {string} string "Currency updated"
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/currencies/{id} [put]
func (h *CurrenciesAdminHandler) UpdateCurrency(c *gin.Context) {
	id := chi.URLParam(c.Request, "id")
	var currency models.Currency
	if err := c.ShouldBindJSON(&currency); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := h.PostgresDB.DB.Exec(
		"UPDATE currencies SET status=$1, info=$2::jsonb, updated_at=NOW() WHERE id=$3 AND deleted_at IS NULL",
		currency.Status, currency.Info, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update currency"})
		return
	}

	h.updateSingleCurrencyCache(id)
	c.JSON(http.StatusOK, gin.H{"message": "Currency updated"})
}

// DeleteCurrency godoc
// @Summary Delete a currency (logical delete)
// @Description Logically delete a currency by setting deleted_at
// @Tags Admin
// @Param id path string true "Currency ID"
// @Success 200 {string} string "Currency deleted"
// @Failure 500 {object} map[string]interface{}
// @Router /admin/currencies/{id} [delete]
func (h *CurrenciesAdminHandler) DeleteCurrency(c *gin.Context) {
	id := chi.URLParam(c.Request, "id")

	_, err := h.PostgresDB.DB.Exec("UPDATE currencies SET deleted_at=NOW() WHERE id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete currency"})
		return
	}

	h.RedisClient.Client.HDel(context.Background(), "currencies", id)
	c.JSON(http.StatusOK, gin.H{"message": "Currency deleted"})
}

// updateRedisCache updates the entire cache of active currencies
func (h *CurrenciesAdminHandler) updateRedisCache() {
	rows, err := h.PostgresDB.DB.Query("SELECT id, hk, status, info FROM currencies WHERE status='approved' AND deleted_at IS NULL")
	if err != nil {
		return
	}
	defer rows.Close()

	var currencies []models.Currency
	for rows.Next() {
		var currency models.Currency
		if err := rows.Scan(&currency.ID, &currency.HK, &currency.Status, &currency.Info); err != nil {
			return
		}
		currencies = append(currencies, currency)
	}

	data, _ := json.Marshal(currencies)
	h.RedisClient.Client.Set(context.Background(), "active_currencies", data, 0)
}

// updateSingleCurrencyCache updates the cache for a single currency by its ID
func (h *CurrenciesAdminHandler) updateSingleCurrencyCache(id string) {
	row := h.PostgresDB.DB.QueryRow("SELECT id, hk, status, info FROM currencies WHERE id=$1 AND status='approved' AND deleted_at IS NULL", id)
	var currency models.Currency
	if err := row.Scan(&currency.ID, &currency.HK, &currency.Status, &currency.Info); err != nil {
		return
	}

	data, _ := json.Marshal(currency)
	h.RedisClient.Client.HSet(context.Background(), "currencies", id, data)
}
