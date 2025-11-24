package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"gw-currency-wallet/internal/models"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

type Producer interface {
	SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error
	Close() error
}

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	log      *slog.Logger
}

func NewKafkaProducer(brokers []string, topic string, log *slog.Logger) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Timeout = 5 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	log.Info("kafka producer создан", slog.String("topic", topic), slog.Any("brokers", brokers))

	return &KafkaProducer{
		producer: producer,
		topic:    topic,
		log:      log,
	}, nil
}

func (p *KafkaProducer) SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.TransactionID),
		Value: sarama.ByteEncoder(eventData),
	}

	type result struct {
		partition int32
		offset    int64
		err       error
	}

	resultCh := make(chan result, 1)

	go func() {
		partition, offset, err := p.producer.SendMessage(msg)
		resultCh <- result{partition, offset, err}
	}()

	select {
	case res := <-resultCh:
		if res.err != nil {
			p.log.Error("kafka send failed",
				slog.String("tx_id", event.TransactionID),
				slog.String("error", res.err.Error()))
			return res.err
		}
		p.log.Debug("kafka send success",
			slog.String("tx_id", event.TransactionID),
			slog.Int("partition", int(res.partition)),
			slog.Int64("offset", res.offset))
		return nil

	case <-ctx.Done():
		p.log.Warn("kafka send cancelled",
			slog.String("tx_id", event.TransactionID))
		return ctx.Err()
	}
}

func (p *KafkaProducer) Close() error {
	if p.producer == nil {
		return nil
	}
	p.log.Info("закрытие kafka producer")
	return p.producer.Close()
}

type NoOpProducer struct {
	log *slog.Logger
}

func NewNoOpProducer(log *slog.Logger) Producer {
	return &NoOpProducer{log: log}
}

func (p *NoOpProducer) SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error {
	p.log.Debug("kafka отключен, событие не отправлено",
		slog.String("transaction_id", event.TransactionID))
	return nil
}

func (p *NoOpProducer) Close() error {
	return nil
}
