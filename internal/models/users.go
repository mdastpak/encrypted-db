package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the database
type User struct {
	ID        int
	HK        uuid.UUID
	Status    string
	Info      map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullString
}
