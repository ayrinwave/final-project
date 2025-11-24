package models

import (
	"time"

	"github.com/google/uuid"
)

type ExchangeRate struct {
	ID        uuid.UUID `db:"id"`
	Currency  string    `db:"currency"`
	Rate      float64   `db:"rate"`
	UpdatedAt time.Time `db:"updated_at"`
}
