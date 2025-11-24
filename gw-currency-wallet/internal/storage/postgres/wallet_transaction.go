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
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PgWalletRepository) GetWalletBalanceForUpdateTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID) (int64, error) {
	var balance int64
	err := tx.QueryRow(ctx, storage.GetWalletStateQuery, walletID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, custom_err.ErrNotFound
		}
		return 0, err
	}
	return balance, nil
}
func (r *PgWalletRepository) UpdateBalanceTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, newBalance int64) error {
	res, err := tx.Exec(ctx,
		storage.UpdateWalletBalanceQuery,
		newBalance,
		walletID,
	)
	if err != nil {

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return custom_err.ErrInsufficientFunds
		}
		return err
	}

	if res.RowsAffected() == 0 {
		return custom_err.ErrNotFound
	}

	return nil
}

func (r *PgWalletRepository) OperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, storage.CheckOperationExistsQuery, requestID).Scan(&exists)
	return exists, err
}

func (r *PgWalletRepository) ExchangeOperationExistsTx(ctx context.Context, tx pgx.Tx, requestID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		storage.ExchangeOperationExistsQuery,
		requestID,
	).Scan(&exists)
	return exists, err
}

func (r *PgWalletRepository) RequestIDExists(ctx context.Context, q pgxQueryer, table, requestID string) (bool, error) {
	query := fmt.Sprintf(storage.CheckOperationExistsQuery, table)
	var exists bool
	err := q.QueryRow(ctx, query, requestID).Scan(&exists)
	return exists, err
}
func (r *PgWalletRepository) CreateOperationTx(ctx context.Context, tx pgx.Tx, walletID uuid.UUID, amount int64, requestID string) error {
	_, err := tx.Exec(ctx, storage.CreateOperationQuery, walletID, amount, requestID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return custom_err.ErrDuplicateRequest
		}
		return err
	}
	return nil
}

func (r *PgWalletRepository) CreateWalletTx(ctx context.Context, tx pgx.Tx, wallet *models.Wallet) error {
	_, err := tx.Exec(ctx, storage.CreateWalletQuery,
		wallet.ID, wallet.UserID, wallet.Currency, wallet.Balance)
	return err
}

func (r *PgWalletRepository) CreateExchangeOperationTx(ctx context.Context, tx pgx.Tx, op models.ExchangeOperation) error {
	_, err := tx.Exec(ctx, storage.CreateExchangeOperationQuery,
		op.UserID, op.FromCurrency, op.ToCurrency,
		op.Amount, op.ExchangedAmount, op.Rate, op.RequestID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return custom_err.ErrDuplicateRequest
		}
		return err
	}
	return nil
}
