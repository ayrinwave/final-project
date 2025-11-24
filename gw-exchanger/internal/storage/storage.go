package storage

import (
	"context"
	"gw-exchanger/internal/models"
)

type Storage interface {
	GetAllRates(ctx context.Context) ([]models.ExchangeRate, error)
	GetRateByCurrency(ctx context.Context, currency string) (*models.ExchangeRate, error)
	Close()
}
