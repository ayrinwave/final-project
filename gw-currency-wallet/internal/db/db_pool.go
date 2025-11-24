package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolConfig struct {
	MaxConns          int
	MinConns          int
	HealthCheckPeriod time.Duration
	PoolTimeout       time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
	ApplicationName   string
}

func NewPool(ctx context.Context, dsn string, cfg PoolConfig, log *slog.Logger) (*pgxpool.Pool, error) {
	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить DSN: %w", err)
	}

	conf.MaxConns = int32(cfg.MaxConns)
	conf.MinConns = int32(cfg.MinConns)
	conf.HealthCheckPeriod = cfg.HealthCheckPeriod
	conf.MaxConnLifetime = 30 * time.Minute
	conf.MaxConnIdleTime = 5 * time.Minute
	if cfg.ApplicationName != "" {
		conf.ConnConfig.RuntimeParams["application_name"] = cfg.ApplicationName
	}
	conf.ConnConfig.ConnectTimeout = cfg.PoolTimeout

	var pool *pgxpool.Pool
	for i := 0; i < cfg.RetryAttempts; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, conf)
		if err != nil {
			log.Warn("не удалось создать пул соединений",
				slog.Int("attempt", i+1),
				slog.Int("max_attempts", cfg.RetryAttempts),
				slog.String("error", err.Error()))
			time.Sleep(cfg.RetryDelay * time.Duration(1<<i))
			continue
		}

		if err = pool.Ping(ctx); err != nil {
			log.Warn("ping БД не удался",
				slog.Int("attempt", i+1),
				slog.String("error", err.Error()))
			pool.Close()
			time.Sleep(cfg.RetryDelay * time.Duration(1<<i))
			continue
		}

		log.Info("подключение к базе данных успешно")
		return pool, nil
	}

	return nil, fmt.Errorf("не удалось создать пул соединений после %d попыток: %w", cfg.RetryAttempts, err)
}
