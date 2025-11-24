package grpc_server

import (
	"context"
	"gw-exchanger/internal/storage"
	pb "gw-exchanger/proto-exchange"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ExchangeServer struct {
	pb.UnimplementedExchangeServiceServer
	storage storage.Storage
	log     *slog.Logger
}

var supportedCurrencies = map[string]bool{
	"USD": true,
	"RUB": true,
	"EUR": true,
}

func NewExchangeServer(storage storage.Storage, log *slog.Logger) *ExchangeServer {
	return &ExchangeServer{
		storage: storage,
		log:     log,
	}
}

func (s *ExchangeServer) GetExchangeRates(ctx context.Context, req *pb.Empty) (*pb.ExchangeRatesResponse, error) {
	const op = "grpc_server.GetExchangeRates"

	rates, err := s.storage.GetAllRates(ctx)
	if err != nil {
		s.log.Error("failed to get rates", slog.String("op", op), slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get exchange rates")
	}

	ratesMap := make(map[string]float64, len(rates))
	for _, rate := range rates {
		ratesMap[rate.Currency] = rate.Rate
	}

	return &pb.ExchangeRatesResponse{
		Rates: ratesMap,
	}, nil
}

func (s *ExchangeServer) GetExchangeRateForCurrency(ctx context.Context, req *pb.CurrencyRequest) (*pb.ExchangeRateResponse, error) {
	const op = "grpc_server.GetExchangeRateForCurrency"

	if !supportedCurrencies[req.FromCurrency] {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported from_currency: %s", req.FromCurrency)
	}
	if !supportedCurrencies[req.ToCurrency] {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported to_currency: %s", req.ToCurrency)
	}
	if req.FromCurrency == req.ToCurrency {
		return nil, status.Error(codes.InvalidArgument, "from_currency and to_currency must be different")
	}

	s.log.Info("получен запрос на курс валюты",
		slog.String("op", op),
		slog.String("from", req.FromCurrency),
		slog.String("to", req.ToCurrency))

	fromRate, err := s.storage.GetRateByCurrency(ctx, req.FromCurrency)
	if err != nil {
		s.log.Error("ошибка получения курса исходной валюты",
			slog.String("op", op),
			slog.String("currency", req.FromCurrency),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "%s: failed to get from_currency rate", op)
	}

	toRate, err := s.storage.GetRateByCurrency(ctx, req.ToCurrency)
	if err != nil {
		s.log.Error("ошибка получения курса целевой валюты",
			slog.String("op", op),
			slog.String("currency", req.ToCurrency),
			slog.String("error", err.Error()))
		return nil, status.Errorf(codes.Internal, "%s: failed to get to_currency rate", op)
	}

	if fromRate.Rate == 0 || toRate.Rate == 0 {
		s.log.Error("invalid rate in database",
			slog.String("from", req.FromCurrency),
			slog.Float64("from_rate", fromRate.Rate),
			slog.String("to", req.ToCurrency),
			slog.Float64("to_rate", toRate.Rate))
		return nil, status.Error(codes.Internal, "invalid exchange rate data")
	}

	rate := toRate.Rate / fromRate.Rate

	s.log.Info("отправлен курс обмена",
		slog.String("op", op),
		slog.String("from", req.FromCurrency),
		slog.String("to", req.ToCurrency),
		slog.Float64("rate", rate))

	return &pb.ExchangeRateResponse{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Rate:         rate,
	}, nil
}
