package service

import (
	"context"
	"fmt"
	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/grpc_client"
	"gw-currency-wallet/internal/kafka"
	"gw-currency-wallet/internal/models"
	"gw-currency-wallet/internal/storage/postgres"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CachedRate struct {
	Rate      float64
	Timestamp time.Time
}

type Exchange interface {
	GetExchangeRates(ctx context.Context) (map[string]float64, error)
	ExchangeCurrency(ctx context.Context, userID uuid.UUID, req models.ExchangeRequest) (*models.ExchangeResponse, error)
}

type ExchangeService struct {
	walletRepo    postgres.WalletRepository
	txManager     TxManager
	grpcClient    grpc_client.ExchangerClient
	kafkaProducer kafka.Producer

	cache         map[string]CachedRate
	allRatesCache *AllRatesCache
	cacheMutex    sync.RWMutex

	cacheExpiration time.Duration
	log             *slog.Logger

	eventQueue chan models.LargeTransferEvent
	wg         sync.WaitGroup
	stopCh     chan struct{}
}
type AllRatesCache struct {
	Rates     map[string]float64
	Timestamp time.Time
}

func NewExchangeService(
	walletRepo postgres.WalletRepository,
	txManager TxManager,
	grpcClient grpc_client.ExchangerClient,
	kafkaProducer kafka.Producer,
	cacheExpiration time.Duration,
	log *slog.Logger,
) *ExchangeService {
	svc := &ExchangeService{
		walletRepo:      walletRepo,
		txManager:       txManager,
		grpcClient:      grpcClient,
		kafkaProducer:   kafkaProducer,
		cache:           make(map[string]CachedRate),
		cacheExpiration: cacheExpiration,
		eventQueue:      make(chan models.LargeTransferEvent, 100),
		stopCh:          make(chan struct{}),
		log:             log,
	}

	for i := 0; i < 5; i++ {
		svc.wg.Add(1)
		go svc.kafkaWorker(i)
	}

	return svc
}
func (s *ExchangeService) kafkaWorker(id int) {
	defer s.wg.Done()
	s.log.Info("kafka worker started", slog.Int("worker_id", id))

	for {
		select {
		case event := <-s.eventQueue:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := s.kafkaProducer.SendLargeTransferEvent(ctx, event); err != nil {
				s.log.Error("kafka send failed",
					slog.Int("worker_id", id),
					slog.String("tx_id", event.TransactionID),
					slog.String("error", err.Error()))
			} else {
				s.log.Info("event sent to kafka",
					slog.Int("worker_id", id),
					slog.String("tx_id", event.TransactionID))
			}
			cancel()

		case <-s.stopCh:
			s.log.Info("kafka worker stopping", slog.Int("worker_id", id))
			return
		}
	}
}

func (s *ExchangeService) Shutdown(ctx context.Context) error {
	s.log.Info("shutting down exchange service")

	close(s.stopCh)

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("all kafka workers stopped")
		return nil
	case <-ctx.Done():
		s.log.Warn("shutdown timeout exceeded")
		return ctx.Err()
	}
}

func (s *ExchangeService) GetExchangeRates(ctx context.Context) (map[string]float64, error) {
	s.cacheMutex.RLock()
	if s.allRatesCache != nil && time.Since(s.allRatesCache.Timestamp) < s.cacheExpiration {
		rates := make(map[string]float64, len(s.allRatesCache.Rates))
		for k, v := range s.allRatesCache.Rates {
			rates[k] = v
		}
		s.cacheMutex.RUnlock()
		return rates, nil
	}
	s.cacheMutex.RUnlock()

	resp, err := s.grpcClient.GetExchangeRates(ctx)
	if err != nil {
		return nil, err
	}

	s.cacheMutex.Lock()
	s.allRatesCache = &AllRatesCache{
		Rates:     resp.Rates,
		Timestamp: time.Now(),
	}
	s.cacheMutex.Unlock()

	return resp.Rates, nil
}

func (s *ExchangeService) getExchangeRate(ctx context.Context, from, to string) (float64, error) {
	const op = "service.getExchangeRate"

	cacheKey := fmt.Sprintf("%s_%s", from, to)

	s.cacheMutex.RLock()
	if cached, ok := s.cache[cacheKey]; ok {
		if time.Since(cached.Timestamp) < s.cacheExpiration {
			rate := cached.Rate
			s.cacheMutex.RUnlock()
			s.log.Debug("курс взят из кэша",
				slog.String("from", from),
				slog.String("to", to),
				slog.Float64("rate", rate))
			return rate, nil
		}
	}
	s.cacheMutex.RUnlock()

	s.log.Debug("запрос курса у exchanger сервиса",
		slog.String("from", from),
		slog.String("to", to))

	resp, err := s.grpcClient.GetExchangeRateForCurrency(ctx, from, to)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	s.cacheMutex.Lock()
	s.cache[cacheKey] = CachedRate{
		Rate:      resp.Rate,
		Timestamp: time.Now(),
	}
	s.cacheMutex.Unlock()

	s.log.Debug("курс обновлен в кэше",
		slog.String("from", from),
		slog.String("to", to),
		slog.Float64("rate", resp.Rate))

	return resp.Rate, nil
}

func (s *ExchangeService) ExchangeCurrency(ctx context.Context, userID uuid.UUID, req models.ExchangeRequest) (*models.ExchangeResponse, error) {
	const op = "service.ExchangeCurrency"

	if !req.FromCurrency.IsValid() || !req.ToCurrency.IsValid() {
		return nil, custom_err.ErrInvalidCurrency
	}
	if req.Amount <= 0 {
		return nil, custom_err.ErrInvalidAmount
	}
	if req.FromCurrency == req.ToCurrency {
		return nil, fmt.Errorf("%s: cannot exchange same currency", op)
	}
	if req.RequestID == "" {
		return nil, custom_err.ErrInvalidInput
	}

	rate, err := s.getExchangeRate(ctx, string(req.FromCurrency), string(req.ToCurrency))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get exchange rate: %w", op, err)
	}

	exchangedAmount := req.Amount * rate

	s.log.Info("обмен валют",
		slog.String("user_id", userID.String()),
		slog.String("from", string(req.FromCurrency)),
		slog.String("to", string(req.ToCurrency)),
		slog.Float64("amount", req.Amount),
		slog.Float64("rate", rate),
		slog.Float64("exchanged_amount", exchangedAmount))

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {

		exists, err := s.walletRepo.ExchangeOperationExistsTx(ctx, tx, req.RequestID)
		if err != nil {
			return fmt.Errorf("failed to check exchange operation: %w", err)
		}
		if exists {
			return custom_err.ErrDuplicateRequest
		}

		fromWallet, err := s.walletRepo.GetByUserAndCurrency(ctx, userID, req.FromCurrency)
		if err != nil {
			return fmt.Errorf("failed to get source wallet: %w", err)
		}

		toWallet, err := s.walletRepo.GetByUserAndCurrency(ctx, userID, req.ToCurrency)
		if err != nil {
			return fmt.Errorf("failed to get destination wallet: %w", err)
		}

		amountInMinorUnits := models.AmountToMinorUnits(req.Amount)
		exchangedAmountInMinorUnits := models.AmountToMinorUnits(exchangedAmount)

		fromBalance, err := s.walletRepo.GetWalletBalanceForUpdateTx(ctx, tx, fromWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to get source balance: %w", err)
		}

		newFromBalance := fromBalance - amountInMinorUnits
		if newFromBalance < 0 {
			return custom_err.ErrInsufficientFunds
		}

		if err := s.walletRepo.UpdateBalanceTx(ctx, tx, fromWallet.ID, newFromBalance); err != nil {
			return fmt.Errorf("failed to update source balance: %w", err)
		}

		toBalance, err := s.walletRepo.GetWalletBalanceForUpdateTx(ctx, tx, toWallet.ID)
		if err != nil {
			return fmt.Errorf("failed to get destination balance: %w", err)
		}

		newToBalance := toBalance + exchangedAmountInMinorUnits

		if err := s.walletRepo.UpdateBalanceTx(ctx, tx, toWallet.ID, newToBalance); err != nil {
			return fmt.Errorf("failed to update destination balance: %w", err)
		}

		err = s.walletRepo.CreateExchangeOperationTx(ctx, tx, models.ExchangeOperation{
			UserID:          userID,
			FromCurrency:    string(req.FromCurrency),
			ToCurrency:      string(req.ToCurrency),
			Amount:          amountInMinorUnits,
			ExchangedAmount: exchangedAmountInMinorUnits,
			Rate:            rate,
			RequestID:       req.RequestID,
		})
		if err != nil {
			return fmt.Errorf("failed to create exchange operation: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	const largeTransferThreshold = 30000.0
	if req.Amount >= largeTransferThreshold || exchangedAmount >= largeTransferThreshold {
		event := models.LargeTransferEvent{
			TransactionID: req.RequestID,
			UserID:        userID,
			FromCurrency:  string(req.FromCurrency),
			ToCurrency:    string(req.ToCurrency),
			Amount:        req.Amount,
			ExchangedAmt:  exchangedAmount,
			Rate:          rate,
			Timestamp:     time.Now(),
		}

		select {
		case s.eventQueue <- event:
			s.log.Debug("событие о крупном переводе добавлено в очередь", slog.String("transaction_id", req.RequestID))
		default:
			s.log.Error("очередь событий переполнена, событие отброшено",
				slog.String("transaction_id", req.RequestID),
				slog.Float64("amount", req.Amount))
		}
	}

	return &models.ExchangeResponse{
		Message:         "Exchange successful",
		ExchangedAmount: exchangedAmount,
		Rate:            rate,
	}, nil
}
