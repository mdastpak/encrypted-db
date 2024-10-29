package db

import (
	"database/sql"
	"encrypted-db/config"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type PostgresService struct {
	DB *sql.DB
}

// NewPostgresService sets up the PostgreSQL connection
func NewPostgresService() *PostgresService {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Config.Postgres.Host, config.Config.Postgres.Port,
		config.Config.Postgres.User, config.Config.Postgres.Password,
		config.Config.Postgres.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening PostgreSQL database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to PostgreSQL database: %v", err)
	}

	return &PostgresService{
		DB: db,
	}
}
