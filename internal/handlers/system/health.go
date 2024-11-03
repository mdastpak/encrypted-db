package system

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

// HealthCheckHandler checks the health of each service and returns a JSON response
func (h *SystemHandler) HealthCheckHandler(c *gin.Context) {
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

	// Return JSON response
	c.JSON(http.StatusOK, healthStatus)
}
