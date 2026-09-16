package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"encrypted-db/internal/db"
	"encrypted-db/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupUserRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err())

	t.Cleanup(func() {
		client.FlushDB(context.Background())
		client.Close()
	})

	return client
}

func performGetActiveCurrencies(h *UserHandler) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/currencies", h.GetActiveCurrencies)

	req := httptest.NewRequest(http.MethodGet, "/currencies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestGetActiveCurrencies_Empty(t *testing.T) {
	client := setupUserRedis(t)
	h := &UserHandler{RedisClient: &db.RedisService{Client: client}}

	w := performGetActiveCurrencies(h)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Active currencies retrieved successfully.", resp.Message)
	assert.Empty(t, resp.Info)
}

func TestGetActiveCurrencies_ReturnsCachedCurrencies(t *testing.T) {
	client := setupUserRedis(t)
	h := &UserHandler{RedisClient: &db.RedisService{Client: client}}

	currencies := []models.Currency{
		{HK: "hk-1", Status: "approved", Info: models.CurrencyInfo{EnName: "Bitcoin", Symbol: "BTC"}},
		{HK: "hk-2", Status: "approved", Info: models.CurrencyInfo{EnName: "Ethereum", Symbol: "ETH"}},
	}

	ctx := context.Background()
	for _, c := range currencies {
		data, err := json.Marshal(c)
		require.NoError(t, err)
		key := fmt.Sprintf("base_definitions:currency:%s", c.HK)
		require.NoError(t, client.Set(ctx, key, data, 0).Err())
	}

	w := performGetActiveCurrencies(h)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Active currencies retrieved successfully.", resp.Message)

	got := resp.Info.([]interface{})
	assert.Len(t, got, 2)
}

func TestGetActiveCurrencies_UnmarshalErrorReturns500(t *testing.T) {
	client := setupUserRedis(t)
	h := &UserHandler{RedisClient: &db.RedisService{Client: client}}

	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "base_definitions:currency:broken", "not-json", 0).Err())

	w := performGetActiveCurrencies(h)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp models.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Failed to process currency information.", resp.Message)
}

func TestUpdateProfile_ReturnsNotImplemented(t *testing.T) {
	h := &UserHandler{}
	router := gin.New()
	router.PUT("/profile", h.UpdateProfile)

	req := httptest.NewRequest(http.MethodPut, "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)

	var resp models.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Profile update is not implemented yet.", resp.Message)
}
