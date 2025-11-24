package grpc_client

import (
	"context"
	"fmt"
	pb "gw-exchanger/proto-exchange"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ExchangeRatesResponse struct {
	Rates map[string]float64
}

type ExchangeRateResponse struct {
	FromCurrency string
	ToCurrency   string
	Rate         float64
}

type ExchangerClient interface {
	GetExchangeRates(ctx context.Context) (*ExchangeRatesResponse, error)
	GetExchangeRateForCurrency(ctx context.Context, from, to string) (*ExchangeRateResponse, error)
	Close() error
}

type grpcExchangerClient struct {
	conn    *grpc.ClientConn
	client  pb.ExchangeServiceClient
	timeout time.Duration
	log     *slog.Logger
}

func NewExchangerClient(addr string, timeout time.Duration, log *slog.Logger) (ExchangerClient, error) {
	const op = "grpc_client.NewExchangerClient"

	log.Info("подключение к gRPC exchanger сервису", slog.String("addr", addr))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect: %w", op, err)
	}

	client := pb.NewExchangeServiceClient(conn)

	log.Info("успешное подключение к exchanger сервису")

	return &grpcExchangerClient{
		conn:    conn,
		client:  client,
		timeout: timeout,
		log:     log,
	}, nil
}

func (c *grpcExchangerClient) GetExchangeRates(ctx context.Context) (*ExchangeRatesResponse, error) {
	const op = "grpc_client.GetExchangeRates"

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetExchangeRates(ctx, &pb.Empty{})
	if err != nil {
		c.log.Error("ошибка получения курсов", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		c.log.Warn("медленный gRPC запрос",
			slog.String("op", op),
			slog.Duration("duration", duration))
	}

	rates := make(map[string]float64)
	for currency, rate := range resp.Rates {
		rates[currency] = float64(rate)
	}

	return &ExchangeRatesResponse{
		Rates: rates,
	}, nil
}

func (c *grpcExchangerClient) GetExchangeRateForCurrency(ctx context.Context, from, to string) (*ExchangeRateResponse, error) {
	const op = "grpc_client.GetExchangeRateForCurrency"

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.log.Debug("запрос курса валюты",
		slog.String("from", from),
		slog.String("to", to))

	resp, err := c.client.GetExchangeRateForCurrency(ctx, &pb.CurrencyRequest{
		FromCurrency: from,
		ToCurrency:   to,
	})
	if err != nil {
		c.log.Error("ошибка получения курса",
			slog.String("op", op),
			slog.String("from", from),
			slog.String("to", to),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	c.log.Debug("получен курс валюты",
		slog.String("from", resp.FromCurrency),
		slog.String("to", resp.ToCurrency),
		slog.Float64("rate", float64(resp.Rate)))

	return &ExchangeRateResponse{
		FromCurrency: resp.FromCurrency,
		ToCurrency:   resp.ToCurrency,
		Rate:         float64(resp.Rate),
	}, nil
}

func (c *grpcExchangerClient) Close() error {
	if c.conn == nil {
		return nil
	}
	c.log.Info("закрытие соединения с exchanger сервисом")
	return c.conn.Close()
}
