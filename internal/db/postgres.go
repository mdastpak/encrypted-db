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

// migrationLockID is a fixed Postgres advisory lock key used to serialize
// schema migrations against a database. This guards against multiple
// concurrently-starting replicas of this service (or, in this repo's test
// suite, NewTestPostgresService and the full-migration tests) racing
// DROP/CREATE DDL against the same database at once.
const migrationLockID int64 = 918273645

// withMigrationLock runs fn while holding a session-scoped Postgres
// advisory lock, ensuring exclusive access to schema-mutating DDL.
func withMigrationLock(ctx context.Context, db *sql.DB, fn func() error) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for migration lock: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)

	return fn()
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

	if err := pingWithRetry(db, config.Config.Postgres.ConnectRetries, time.Duration(config.Config.Postgres.ConnectRetryDelayMs)*time.Millisecond); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to PostgreSQL database: %w", err)
	}

	// Serialize migrations via a Postgres advisory lock. This guards against
	// two hazards: multiple concurrently-starting replicas of this service
	// migrating the same database simultaneously in production, and (in the
	// test suite) different packages' tests racing full-schema resets and
	// migrations against the same shared test database under `go test ./...`.
	if err := withMigrationLock(context.Background(), db, func() error {
		return runMigrations(db)
	}); err != nil {
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

	// Create users table with correct structure for tests. Serialized via an
	// advisory lock: go test runs different packages in parallel and many
	// packages call this function against the same shared test database, so
	// without mutual exclusion here concurrent DROP/CREATE TYPE statements
	// from other packages (or this package's full-schema-reset tests) race
	// and fail with "type already exists" / "schema concurrently dropped".
	err = withMigrationLock(ctx, db, func() error {
		_, execErr := db.ExecContext(ctx, `
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
		return execErr
	})
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

// pingWithRetry pings the database up to maxAttempts times with an exponential
// backoff between attempts, so transient startup ordering issues (e.g. the
// database container not yet accepting connections) do not fail the service.
// A maxAttempts value <= 1 performs a single attempt with no retry.
func pingWithRetry(db *sql.DB, maxAttempts int, delay time.Duration) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		lastErr = db.PingContext(ctx)
		cancel()
		if lastErr == nil {
			return nil
		}
		if attempt < maxAttempts {
			log.Printf("PostgreSQL ping attempt %d/%d failed: %v, retrying in %v", attempt, maxAttempts, lastErr, delay)
			time.Sleep(delay)
			delay *= 2
		}
	}
	return lastErr
}
