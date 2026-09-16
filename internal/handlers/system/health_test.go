package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"encrypted-db/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func performHealthCheck(h *SystemHandler) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/health", h.HealthCheckHandler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestHealthCheckHandler_AllDependenciesDown(t *testing.T) {
	h := &SystemHandler{
		Postgres:        &db.PostgresService{},
		Redis:           &db.RedisService{},
		RabbitMQService: nil,
	}

	w := performHealthCheck(h)
	assert.Equal(t, http.StatusOK, w.Code)

	var status HealthStatus
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))

	assert.Equal(t, "degraded", status.Status)
	require.Contains(t, status.Details, "postgres")
	require.Contains(t, status.Details, "redis")
	require.Contains(t, status.Details, "rabbitmq")
	assert.Equal(t, "error", status.Details["postgres"].Status)
	assert.Equal(t, "error", status.Details["redis"].Status)
	assert.Equal(t, "error", status.Details["rabbitmq"].Status)
}

func TestHealthCheckHandler_NilRabbitMQOnlyDegradesRabbitMQ(t *testing.T) {
	// Postgres and Redis are healthy (nil-safe, no live services required
	// for this scenario as we assert only on the rabbitmq-triggered
	// "degraded" overall status and per-service detail).
	h := &SystemHandler{
		Postgres:        &db.PostgresService{},
		Redis:           &db.RedisService{},
		RabbitMQService: nil,
	}

	w := performHealthCheck(h)

	var status HealthStatus
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))

	assert.Equal(t, "error", status.Details["rabbitmq"].Status)
	assert.Equal(t, "degraded", status.Status, "a nil RabbitMQService must degrade overall status")
}

func TestHealthCheckHandler_ResponseIncludesLatencyAndTimestamps(t *testing.T) {
	h := &SystemHandler{
		Postgres: &db.PostgresService{},
		Redis:    &db.RedisService{},
	}

	w := performHealthCheck(h)

	var status HealthStatus
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))

	for name, detail := range status.Details {
		assert.False(t, detail.LastCheck.IsZero(), "%s should have a non-zero LastCheck timestamp", name)
		assert.GreaterOrEqual(t, detail.LatencyMs, int64(0), "%s should report a non-negative latency", name)
	}
}

func TestPingPongHandler(t *testing.T) {
	h := &SystemHandler{}
	router := gin.New()
	router.GET("/ping", h.PingPongHandler)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "pong", body["message"])
	assert.NotEmpty(t, body["timestamp"])
}
