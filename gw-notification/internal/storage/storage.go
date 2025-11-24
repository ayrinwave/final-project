package storage

import (
	"context"
	"gw-notification/internal/models"
)

type Storage interface {
	SaveNotification(ctx context.Context, notification *models.LargeTransferNotification) error
	GetNotificationByTransactionID(ctx context.Context, transactionID string) (*models.LargeTransferNotification, error)
	Close() error
}
