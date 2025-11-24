package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"gw-notification/internal/models"
	"gw-notification/internal/storage"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	storage       storage.Storage
	topic         string
	workers       int
	log           *slog.Logger
	wg            sync.WaitGroup
}

func NewConsumer(brokers []string, groupID, topic string, workers int, storage storage.Storage, log *slog.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	log.Info("kafka consumer создан",
		slog.String("group_id", groupID),
		slog.String("topic", topic),
		slog.Int("workers", workers))

	return &Consumer{
		consumerGroup: consumerGroup,
		storage:       storage,
		topic:         topic,
		workers:       workers,
		log:           log,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("запуск kafka consumer")

	handler := &consumerGroupHandler{
		storage: c.storage,
		log:     c.log,
	}

	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go func(workerID int) {
			defer c.wg.Done()
			c.log.Info("воркер запущен", slog.Int("worker_id", workerID))

			for {

				if err := c.consumerGroup.Consume(ctx, []string{c.topic}, handler); err != nil {
					c.log.Error("ошибка consume",
						slog.Int("worker_id", workerID),
						slog.String("error", err.Error()))
					return
				}

				if ctx.Err() != nil {
					return
				}
			}
		}(i)
	}

	go func() {
		for err := range c.consumerGroup.Errors() {
			c.log.Error("ошибка consumer group", slog.String("error", err.Error()))
		}
	}()

	return nil
}

func (c *Consumer) Close(ctx context.Context) error {
	c.log.Info("закрытие kafka consumer")

	done := make(chan struct{})
	go func() {
		if err := c.consumerGroup.Close(); err != nil {
			c.log.Error("failed to close consumer group", slog.String("error", err.Error()))
		}
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.log.Info("kafka consumer закрыт")
		return nil
	case <-ctx.Done():
		c.log.Warn("kafka consumer close timeout")
		return ctx.Err()
	}
}

type consumerGroupHandler struct {
	storage storage.Storage
	log     *slog.Logger
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := h.processMessage(session.Context(), message); err != nil {
			h.log.Error("failed to process message",
				slog.String("topic", message.Topic),
				slog.Int64("offset", message.Offset),
				slog.String("error", err.Error()))

			continue
		}
		session.MarkMessage(message, "")
	}
	return nil
}

func (h *consumerGroupHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	h.log.Debug("получено сообщение из kafka",
		slog.String("topic", message.Topic),
		slog.Int("partition", int(message.Partition)),
		slog.Int64("offset", message.Offset))

	var kafkaMsg models.KafkaMessage
	if err := json.Unmarshal(message.Value, &kafkaMsg); err != nil {
		h.log.Error("ошибка десериализации сообщения",
			slog.String("error", err.Error()),
			slog.String("raw_message", string(message.Value)))

		return nil
	}

	notification := &models.LargeTransferNotification{
		TransactionID: kafkaMsg.TransactionID,
		UserID:        kafkaMsg.UserID,
		FromCurrency:  kafkaMsg.FromCurrency,
		ToCurrency:    kafkaMsg.ToCurrency,
		Amount:        kafkaMsg.Amount,
		ExchangedAmt:  kafkaMsg.ExchangedAmt,
		Rate:          kafkaMsg.Rate,
		Timestamp:     kafkaMsg.Timestamp,
		ProcessedAt:   time.Now(),
	}

	if err := h.storage.SaveNotification(ctx, notification); err != nil {
		h.log.Error("ошибка сохранения уведомления",
			slog.String("transaction_id", notification.TransactionID),
			slog.String("error", err.Error()))
		return err
	}

	h.log.Info("уведомление успешно сохранено",
		slog.String("transaction_id", notification.TransactionID),
		slog.String("user_id", notification.UserID),
		slog.Float64("amount", notification.Amount))

	return nil
}
