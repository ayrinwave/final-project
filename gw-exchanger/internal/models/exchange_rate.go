package models

import "time"

type ExchangeRate struct {
	ID        int       `db:"id"`
	Currency  string    `db:"currency"`
	Rate      float64   `db:"rate"`
	UpdatedAt time.Time `db:"updated_at"`
}
