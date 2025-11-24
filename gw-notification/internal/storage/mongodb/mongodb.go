package mongodb

import (
	"context"
	"errors"
	"fmt"
	"gw-notification/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

func NewMongoStorage(ctx context.Context, uri, database, collection string, timeout time.Duration) (*MongoStorage, error) {
	clientOpts := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(timeout).
		SetServerSelectionTimeout(timeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(ctxPing, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)
	coll := db.Collection(collection)

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "transaction_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	ctxIndex, cancelIndex := context.WithTimeout(ctx, timeout)
	defer cancelIndex()

	if _, err := coll.Indexes().CreateOne(ctxIndex, indexModel); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &MongoStorage{
		client:     client,
		database:   db,
		collection: coll,
	}, nil
}

func (s *MongoStorage) SaveNotification(ctx context.Context, notification *models.LargeTransferNotification) error {

	notification.ProcessedAt = time.Now()

	_, err := s.collection.InsertOne(ctx, notification)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("failed to save notification: %w", err)
	}

	return nil
}

var ErrNotificationNotFound = errors.New("notification not found")

func (s *MongoStorage) GetNotificationByTransactionID(ctx context.Context, transactionID string) (*models.LargeTransferNotification, error) {
	var notification models.LargeTransferNotification

	filter := bson.M{"transaction_id": transactionID}
	err := s.collection.FindOne(ctx, filter).Decode(&notification)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}
func (s *MongoStorage) Close() error {
	if s.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.client.Disconnect(ctx)
}
