package db

import (
	"context"
	"encrypted-db/config"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib" // Import for pgx-stdlib compatibility
)

type PostgresService struct {
	Pool *pgxpool.Pool
}

// NewPostgresService initializes the PostgreSQL connection pool and sets up a default context with timeout
func NewPostgresService() *PostgresService {
	// Create the connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Config.Postgres.Host, config.Config.Postgres.Port,
		config.Config.Postgres.User, config.Config.Postgres.Password,
		config.Config.Postgres.DBName)

	// Configure connection pool settings
	pgxConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Unable to parse connection string: %v", err)
	}
	pgxConfig.MinConns = 5
	pgxConfig.MaxConns = 10
	pgxConfig.MaxConnIdleTime = time.Minute * 5
	pgxConfig.HealthCheckPeriod = time.Minute * 1

	// Connect to the database
	pool, err := pgxpool.ConnectConfig(context.Background(), pgxConfig)
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL database: %v", err)
	}

	return &PostgresService{
		Pool: pool,
	}
}

// Close closes the PostgreSQL connection pool and cancels the default context
func (p *PostgresService) Close() {
	p.Pool.Close()
}

func GetNewContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return ctx
}

// runMigrations applies migrations using a pgx-compatible database/sql connection
func runMigrations(connStr string) error {
	// Parse pgx config for stdlib
	connConfig, err := pgx.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("unable to parse connection string for migration: %v", err)
	}

	// Use pgx-stdlib to open a database/sql connection
	db := stdlib.OpenDB(*connConfig)
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not start migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/db/migrations", // Path to migrations folder
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("migration instance creation failed: %v", err)
	}

	// Apply migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %v", err)
	}

	return nil
}
