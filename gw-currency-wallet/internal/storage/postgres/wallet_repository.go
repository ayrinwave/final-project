package postgres

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepository interface {
	GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error)
	UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error
	OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error)
	CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error
	CreateWalletTx(ctx context.Context, tx pgx.Tx, wallet *models.Wallet) error

	GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)
	GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency) (*models.Wallet, error)
	GetAllUserWallets(ctx context.Context, userID uuid.UUID) ([]*models.Wallet, error)

	ExchangeOperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error)
	CreateExchangeOperationTx(ctx context.Context, tx pgx.Tx, op models.ExchangeOperation) error
}
type PgWalletRepository struct {
	db *pgxpool.Pool
}

func NewWalletRepository(db *pgxpool.Pool) WalletRepository {
	return &PgWalletRepository{db: db}
}

func (r *PgWalletRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	const op = "storage.GetByID"
	var wallet models.Wallet
	err := r.db.QueryRow(ctx, storage.GetWalletByIDQuery, id).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.Version,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wallet, nil
}

func (r *PgWalletRepository) GetByUserAndCurrency(ctx context.Context, userID uuid.UUID, currency models.Currency) (*models.Wallet, error) {
	const op = "storage.GetByUserAndCurrency"
	var wallet models.Wallet
	err := r.db.QueryRow(ctx, storage.GetWalletByUserAndCurrencyQuery, userID, currency).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.Version,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wallet, nil
}

func (r *PgWalletRepository) GetAllUserWallets(ctx context.Context, userID uuid.UUID) ([]*models.Wallet, error) {
	const op = "storage.GetAllUserWallets"

	rows, err := r.db.Query(ctx, storage.GetAllUserWalletsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var wallets []*models.Wallet
	for rows.Next() {
		var wallet models.Wallet
		err := rows.Scan(
			&wallet.ID,
			&wallet.UserID,
			&wallet.Currency,
			&wallet.Balance,
			&wallet.Version,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: scan error: %w", op, err)
		}
		wallets = append(wallets, &wallet)
	}
	return wallets, nil
}
