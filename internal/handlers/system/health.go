package system

import (
	"encoding/json"
	"encrypted-db/internal/db"
	"net/http"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status  string                   `json:"status"`
	Details map[string]ServiceStatus `json:"details"`
}

// ServiceStatus represents the health status of an individual service
type ServiceStatus struct {
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
}

// Handler struct for system-related endpoints
type Handler struct {
	Postgres *db.PostgresService
	Redis    *db.RedisService
}

// NewHandler creates a new system handler with injected dependencies
func NewHandler(postgres *db.PostgresService, redis *db.RedisService) *Handler {
	return &Handler{Postgres: postgres, Redis: redis}
}

// HealthCheckHandler checks the health of each service and returns a JSON response
func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Initialize health status response
	healthStatus := HealthStatus{
		Status:  "ok",
		Details: make(map[string]ServiceStatus),
	}

	// Check PostgreSQL status
	postgresStatus := "ok"
	if err := h.Postgres.DB.Ping(); err != nil {
		postgresStatus = "error"
		healthStatus.Status = "error"
	}
	healthStatus.Details["postgres"] = ServiceStatus{
		Status:    postgresStatus,
		LastCheck: time.Now(),
	}

	// Check Redis status
	redisStatus := "ok"
	if _, err := h.Redis.Client.Ping(h.Redis.Ctx).Result(); err != nil {
		redisStatus = "error"
		healthStatus.Status = "error"
	}
	healthStatus.Details["redis"] = ServiceStatus{
		Status:    redisStatus,
		LastCheck: time.Now(),
	}

	// Set response headers and return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthStatus)
}
