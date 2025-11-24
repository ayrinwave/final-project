package models

import (
	"time"

	"github.com/google/uuid"
)

// ExchangeRequest запрос на обмен валют
type ExchangeRequest struct {
	FromCurrency Currency `json:"from_currency"`
	ToCurrency   Currency `json:"to_currency"`
	Amount       float64  `json:"amount"`
	RequestID    string   `json:"requestID"`
}

// ExchangeResponse ответ на обмен валют
type ExchangeResponse struct {
	Message         string  `json:"message"`
	ExchangedAmount float64 `json:"exchanged_amount"`
	Rate            float64 `json:"rate,omitempty"`
}

// ExchangeRatesResponse ответ с курсами валют
type ExchangeRatesResponse struct {
	Rates map[string]float64 `json:"rates"`
}
type ExchangeOperation struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	FromCurrency    string
	ToCurrency      string
	Amount          int64
	ExchangedAmount int64
	Rate            float64
	RequestID       string
	CreatedAt       time.Time
}
