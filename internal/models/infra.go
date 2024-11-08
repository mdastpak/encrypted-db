package models

import (
	"encrypted-db/internal/db"
	"encrypted-db/internal/rabbitmq"
)

type InfraServices struct {
	Postgres *db.PostgresService
	Redis    *db.RedisService
	RabbitMQ *rabbitmq.RabbitMQService
}
