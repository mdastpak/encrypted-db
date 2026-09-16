package system

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthStatus struct {
	Status  string                   `json:"status"`
	Details map[string]ServiceStatus `json:"details"`
}

type ServiceStatus struct {
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
	LatencyMs int64     `json:"latency_ms,omitempty"`
}

func (h *SystemHandler) HealthCheckHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	healthStatus := HealthStatus{
		Status:  "ok",
		Details: make(map[string]ServiceStatus),
	}

	start := time.Now()
	if err := h.Postgres.Ping(ctx); err != nil {
		healthStatus.Status = "degraded"
		healthStatus.Details["postgres"] = ServiceStatus{
			Status:    "error",
			LastCheck: time.Now(),
			LatencyMs: time.Since(start).Milliseconds(),
		}
	} else {
		healthStatus.Details["postgres"] = ServiceStatus{
			Status:    "ok",
			LastCheck: time.Now(),
			LatencyMs: time.Since(start).Milliseconds(),
		}
	}

	start = time.Now()
	if err := h.Redis.Ping(ctx); err != nil {
		healthStatus.Status = "degraded"
		healthStatus.Details["redis"] = ServiceStatus{
			Status:    "error",
			LastCheck: time.Now(),
			LatencyMs: time.Since(start).Milliseconds(),
		}
	} else {
		healthStatus.Details["redis"] = ServiceStatus{
			Status:    "ok",
			LastCheck: time.Now(),
			LatencyMs: time.Since(start).Milliseconds(),
		}
	}

	start = time.Now()
	if h.RabbitMQService == nil || !h.RabbitMQService.IsHealthy() {
		healthStatus.Status = "degraded"
		healthStatus.Details["rabbitmq"] = ServiceStatus{
			Status:    "error",
			LastCheck: time.Now(),
			LatencyMs: time.Since(start).Milliseconds(),
		}
	} else {
		healthStatus.Details["rabbitmq"] = ServiceStatus{
			Status:    "ok",
			LastCheck: time.Now(),
			LatencyMs: time.Since(start).Milliseconds(),
		}
	}

	c.JSON(http.StatusOK, healthStatus)
}

func (h *SystemHandler) PingPongHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong", "timestamp": time.Now().UTC().Format(time.RFC3339)})
}
