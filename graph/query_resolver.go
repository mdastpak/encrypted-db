package graph

import (
	"context"
	"encrypted-db/graph/model"
	"time"
)

func (r *Resolver) Healthcheck(ctx context.Context) (*model.HealthStatus, error) {
	// Check services' health status
	status := &model.HealthStatus{Status: "ok", Details: []*model.ServiceStatus{}}

	postgresStatus := "ok"
	if err := r.Postgres.DB.Ping(); err != nil {
		postgresStatus = "error"
		status.Status = "error"
	}
	status.Details = append(status.Details, &model.ServiceStatus{
		Service:   "Postgres",
		Status:    postgresStatus,
		LastCheck: time.Now().Format(time.RFC3339),
	})

	redisStatus := "ok"
	if _, err := r.Redis.Client.Ping(ctx).Result(); err != nil {
		redisStatus = "error"
		status.Status = "error"
	}
	status.Details = append(status.Details, &model.ServiceStatus{
		Service:   "Redis",
		Status:    redisStatus,
		LastCheck: time.Now().Format(time.RFC3339),
	})

	return status, nil
}

func (r *Resolver) Messages(ctx context.Context) ([]*model.Message, error) {
	// Retrieve messages from database or memory
	// Placeholder implementation
	return []*model.Message{}, nil
}
