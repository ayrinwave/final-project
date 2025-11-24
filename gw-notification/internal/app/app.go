package app

import (
	"context"
	"fmt"
	"gw-notification/internal/config"
	"gw-notification/internal/kafka"
	"gw-notification/internal/storage"
	"gw-notification/internal/storage/mongodb"
	"gw-notification/pkg/logger"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	log      *slog.Logger
	logFile  *os.File
	cfg      *config.Config
	consumer *kafka.Consumer
	storage  storage.Storage
}

func NewApp() (*App, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	loggerWithFile := logger.NewLoggerWithFile("notification.log")
	log := loggerWithFile.Logger

	log.Info("инициализация gw-notification приложения")
	log.Info("конфигурация загружена",
		slog.String("kafka_topic", cfg.Kafka.Topic),
		slog.String("mongo_database", cfg.MongoDB.Database))

	log.Info("подключение к MongoDB", slog.String("uri", cfg.MongoDB.URI))
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoDB.Timeout)
	defer cancel()

	storage, err := mongodb.NewMongoStorage(
		ctx,
		cfg.MongoDB.URI,
		cfg.MongoDB.Database,
		cfg.MongoDB.Collection,
		cfg.MongoDB.Timeout,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к MongoDB: %w", err)
	}
	log.Info("подключение к MongoDB установлено")

	log.Info("инициализация kafka consumer")
	consumer, err := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.GroupID,
		cfg.Kafka.Topic,
		cfg.Kafka.Workers,
		storage,
		log,
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания kafka consumer: %w", err)
	}

	return &App{
		log:      log,
		logFile:  loggerWithFile.LogFile,
		cfg:      cfg,
		consumer: consumer,
		storage:  storage,
	}, nil
}

func (a *App) Run() error {
	a.log.Info("приложение запускается")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.consumer.Start(ctx); err != nil {
		return fmt.Errorf("ошибка запуска consumer: %w", err)
	}

	a.log.Info("kafka consumer запущен, ожидание сообщений...")

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-shutdownChan
	a.log.Info("получен сигнал завершения", slog.String("signal", sig.String()))

	a.log.Info("приложение останавливается")

	cancel()

	ctxClose, cancelClose := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelClose()

	a.log.Info("закрытие kafka consumer")
	if err := a.consumer.Close(ctxClose); err != nil {
		a.log.Error("ошибка при закрытии kafka consumer", slog.String("error", err.Error()))
	}

	a.log.Info("закрытие соединения с MongoDB")
	if err := a.storage.Close(); err != nil {
		a.log.Error("ошибка при закрытии MongoDB", slog.String("error", err.Error()))
	}

	a.log.Info("закрытие файла логов")
	if a.logFile != nil {
		if err := a.logFile.Close(); err != nil {
			a.log.Error("ошибка при закрытии файла логов", slog.String("error", err.Error()))
		}
	}

	a.log.Info("приложение остановлено корректно")
	return nil
}
