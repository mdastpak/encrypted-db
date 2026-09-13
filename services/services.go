package services

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/rabbitmq"
)

type Services struct {
	Postgres *db.PostgresService
	Redis    *db.RedisService
	RabbitMQ *rabbitmq.RabbitMQService
}

// initializeServices initializes all required services and handles errors
func InitializeServices() (*Services, func(), error) {
	postgresService := db.NewPostgresService()
	redisService := db.NewRedisService()
	rabbitMQService := rabbitmq.NewRabbitMQService()

	cleanup := func() {
		postgresService.Close()
		redisService.Close()
		rabbitMQService.Close()
	}

	return &Services{
		Postgres: postgresService,
		Redis:    redisService,
		RabbitMQ: rabbitMQService,
	}, cleanup, nil
}
