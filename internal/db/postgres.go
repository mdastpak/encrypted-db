package db

import (
	"context"
	"database/sql"
	"encrypted-db/config"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresService struct {
	DB      *sql.DB
	connStr string
}

func NewPostgresService() (*PostgresService, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Config.Postgres.Host, config.Config.Postgres.Port,
		config.Config.Postgres.User, config.Config.Postgres.Password,
		config.Config.Postgres.DBName, config.Config.Postgres.SSLMode)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening PostgreSQL database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to PostgreSQL database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	db.SetMaxOpenConns(config.Config.Postgres.MaxOpenConns)
	db.SetMaxIdleConns(config.Config.Postgres.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(config.Config.Postgres.ConnMaxLifetime) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(config.Config.Postgres.ConnMaxIdleTime) * time.Minute)

	log.Println("PostgreSQL connected and migrations applied successfully.")

	return &PostgresService{
		DB:      db,
		connStr: connStr,
	}, nil
}

func (p *PostgresService) Close() error {
	if p.DB != nil {
		if err := p.DB.Close(); err != nil {
			log.Printf("Error closing PostgreSQL connection: %v", err)
			return err
		}
	}
	return nil
}

func (p *PostgresService) Ping(ctx context.Context) error {
	if p.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return p.DB.PingContext(ctx)
}

func (p *PostgresService) Stats() sql.DBStats {
	if p.DB == nil {
		return sql.DBStats{}
	}
	return p.DB.Stats()
}

// NewTestPostgresService creates a PostgresService for testing without running migrations
func NewTestPostgresService(host, port, user, password, dbname, sslmode string) (*PostgresService, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening PostgreSQL database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to PostgreSQL database: %w", err)
	}

	// Create users table with correct structure for tests
	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS users;
		DROP TYPE IF EXISTS USER_STATUS;
		CREATE TYPE USER_STATUS AS ENUM ('disabled', 'deleted', 'unverified', 'suspend', 'approved');
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			hk UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
			status USER_STATUS DEFAULT 'unverified',
			info JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			deleted_at TIMESTAMP NULL
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create test users table: %w", err)
	}

	log.Println("Test PostgreSQL connected successfully (no migrations).")

	return &PostgresService{
		DB:      db,
		connStr: connStr,
	}, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not start migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		config.MigrationsPath(),
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("migration instance creation failed: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
