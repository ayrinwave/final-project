package app

import (
	"context"
	"fmt"
	"gw-exchanger/internal/config"
	"gw-exchanger/internal/db"
	"gw-exchanger/internal/grpc_server"
	"gw-exchanger/internal/storage"
	"gw-exchanger/internal/storage/postgres"
	"gw-exchanger/pkg/logger"
	pb "gw-exchanger/proto-exchange"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	cfg        *config.Config
	pool       *pgxpool.Pool
	storage    storage.Storage
	grpcServer *grpc.Server
	listener   net.Listener
}

func NewApp() (*App, error) {
	loggerWithFile := logger.NewLoggerWithFile("exchanger.log")
	log := loggerWithFile.Logger

	log.Info("инициализация gw-exchanger приложения")

	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}
	log.Info("конфигурация загружена", slog.String("grpc_port", cfg.GRPCPort))

	log.Info("выполнение миграций базы данных")
	if err := db.RunMigrations(cfg.DB.MigrationURL(), "migrations"); err != nil {
		return nil, fmt.Errorf("ошибка выполнения миграций: %w", err)
	}
	log.Info("миграции успешно применены")

	poolCfg := db.PoolConfig{
		MaxConns:          20,
		MinConns:          5,
		HealthCheckPeriod: 30 * time.Second,
		PoolTimeout:       5 * time.Second,
		RetryAttempts:     5,
		RetryDelay:        1 * time.Second,
	}

	pool, err := db.NewPool(context.Background(), cfg.DB.DSN(), poolCfg, log)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}
	log.Info("подключение к базе данных установлено")

	storage := postgres.NewPostgresStorage(pool)
	exchangeServer := grpc_server.NewExchangeServer(storage, log)

	grpcServer := grpc.NewServer()
	pb.RegisterExchangeServiceServer(grpcServer, exchangeServer)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ошибка создания listener: %w", err)
	}

	log.Info("gRPC сервер инициализирован", slog.String("port", cfg.GRPCPort))

	return &App{
		log:        log,
		cfg:        cfg,
		pool:       pool,
		storage:    storage,
		grpcServer: grpcServer,
		listener:   listener,
	}, nil
}

func (a *App) Run() error {
	a.log.Info("gRPC сервер запускается", slog.String("port", a.cfg.GRPCPort))

	defer a.listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		if err := a.grpcServer.Serve(a.listener); err != nil {
			serverErr <- fmt.Errorf("ошибка запуска gRPC сервера: %w", err)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-shutdownChan:
		a.log.Info("получен сигнал завершения", slog.String("signal", sig.String()))
	}

	a.log.Info("приложение останавливается")
	done := make(chan struct{})
	go func() {
		a.log.Info("остановка gRPC сервера")
		a.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		a.log.Info("gRPC сервер остановлен")
	case <-time.After(30 * time.Second):
		a.log.Warn("timeout graceful shutdown, force stop")
		a.grpcServer.Stop()
	}

	a.log.Info("закрытие соединения с базой данных")
	a.pool.Close()

	a.log.Info("закрытие storage")
	a.storage.Close()

	a.log.Info("приложение остановлено")
	return nil
}
