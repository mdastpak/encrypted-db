package db

import (
	"database/sql"
	"encrypted-db/config"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type PostgresService struct {
	DB *sql.DB
}

// NewPostgresService sets up the PostgreSQL connection and applies migrations
func NewPostgresService() *PostgresService {
	// Create the connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Config.Postgres.Host, config.Config.Postgres.Port,
		config.Config.Postgres.User, config.Config.Postgres.Password,
		config.Config.Postgres.DBName)

	// Connect to the PostgreSQL database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening PostgreSQL database: %v", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to PostgreSQL database: %v", err)
	}

	// Apply migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Error running migrations: %v", err)
	}

	log.Println("PostgreSQL connected and migrations applied successfully.")

	return &PostgresService{
		DB: db,
	}
}

// runMigrations applies migrations from the migrations directory
func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not start migration driver: %v", err)
	}

	// Replace "path/to/migrations" with the actual path to your migrations folder
	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/db/migrations", // Path to migrations folder
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("migration instance creation failed: %v", err)
	}

	// Apply all up migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}
