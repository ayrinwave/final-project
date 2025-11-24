package service

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Wallet interface {
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)

	GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalanceResponse, error)
	Deposit(ctx context.Context, userID uuid.UUID, req models.DepositRequest) (*models.BalanceOperationResponse, error)
	Withdraw(ctx context.Context, userID uuid.UUID, req models.WithdrawRequest) (*models.BalanceOperationResponse, error)
}

type WalletService struct {
	repo      postgres.WalletRepository
	txManager TxManager
}

func NewWalletService(repo postgres.WalletRepository, txManager TxManager) Wallet {
	return &WalletService{
		repo:      repo,
		txManager: txManager,
	}
}

func (s *WalletService) UpdateBalance(ctx context.Context, req models.WalletOperationRequest) error {
	const op = "service.UpdateBalance"

	return s.txManager.WithTx(ctx, func(tx pgx.Tx) error {

		exists, err := s.repo.OperationExistsTx(ctx, tx, req.RequestID)
		if err != nil {
			return fmt.Errorf("%s: failed to check operation: %w", op, err)
		}
		if exists {
			return custom_err.ErrDuplicateRequest
		}

		currentBalance, err := s.repo.GetWalletBalanceForUpdateTx(ctx, tx, req.WalletID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return custom_err.ErrNotFound
			}
			return fmt.Errorf("%s: failed to get balance: %w", op, err)
		}

		var newBalance int64
		switch req.OperationType {
		case models.OperationDeposit:
			newBalance = currentBalance + req.Amount
		case models.OperationWithdraw:
			newBalance = currentBalance - req.Amount
			if newBalance < 0 {
				return custom_err.ErrInsufficientFunds
			}
		default:
			return fmt.Errorf("%s: invalid operation type", op)
		}

		if err := s.repo.UpdateBalanceTx(ctx, tx, req.WalletID, newBalance); err != nil {
			return fmt.Errorf("%s: failed to update balance: %w", op, err)
		}

		if err := s.repo.CreateOperationTx(ctx, tx, req.WalletID, req.Amount, req.RequestID); err != nil {
			return fmt.Errorf("%s: failed to create operation: %w", op, err)
		}

		return nil
	})
}

func (s *WalletService) GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	const op = "service.GetWalletByID"

	wallet, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return wallet, nil
}

func (s *WalletService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*models.UserBalanceResponse, error) {
	const op = "service.GetUserBalance"

	wallets, err := s.repo.GetAllUserWallets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	response := &models.UserBalanceResponse{
		USD: 0.0,
		RUB: 0.0,
		EUR: 0.0,
	}

	for _, wallet := range wallets {
		balance := models.AmountFromMinorUnits(wallet.Balance)
		switch models.Currency(wallet.Currency) {
		case models.CurrencyUSD:
			response.USD = balance
		case models.CurrencyRUB:
			response.RUB = balance
		case models.CurrencyEUR:
			response.EUR = balance
		}
	}

	return response, nil
}

func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, req models.DepositRequest) (*models.BalanceOperationResponse, error) {
	return s.performOperation(ctx, userID, req.Currency, req.Amount, req.RequestID, models.OperationDeposit, "Account topped up successfully")
}

func (s *WalletService) Withdraw(ctx context.Context, userID uuid.UUID, req models.WithdrawRequest) (*models.BalanceOperationResponse, error) {
	return s.performOperation(ctx, userID, req.Currency, req.Amount, req.RequestID, models.OperationWithdraw, "Withdrawal successful")
}

func (s *WalletService) performOperation(
	ctx context.Context,
	userID uuid.UUID,
	currency models.Currency,
	amount float64,
	requestID string,
	opType models.OperationType,
	successMsg string,
) (*models.BalanceOperationResponse, error) {
	const op = "service.performOperation"

	if !currency.IsValid() {
		return nil, custom_err.ErrInvalidCurrency
	}
	if amount <= 0 {
		return nil, custom_err.ErrInvalidAmount
	}
	if requestID == "" {
		return nil, custom_err.ErrInvalidInput
	}

	wallet, err := s.repo.GetByUserAndCurrency(ctx, userID, currency)
	if err != nil {
		if errors.Is(err, custom_err.ErrNotFound) {
			return nil, custom_err.ErrNotFound
		}
		return nil, fmt.Errorf("%s: failed to get wallet: %w", op, err)
	}

	amountInMinorUnits := models.AmountToMinorUnits(amount)

	updateReq := models.WalletOperationRequest{
		WalletID:      wallet.ID,
		OperationType: opType,
		Amount:        amountInMinorUnits,
		RequestID:     requestID,
	}

	if err := s.UpdateBalance(ctx, updateReq); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	balances, err := s.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.BalanceOperationResponse{
		Message:    successMsg,
		NewBalance: *balances,
	}, nil
}
