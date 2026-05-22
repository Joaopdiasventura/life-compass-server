package repository

import (
	"context"
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const TransactionCollectionName = "transactions"

type TransactionFilter struct {
	Type      string
	StartDate *time.Time
	EndDate   *time.Time
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction model.Transaction) (model.Transaction, error)
	Find(ctx context.Context, filter TransactionFilter) ([]model.Transaction, error)
}

type MongoTransactionRepository struct {
	collection *mongo.Collection
}

func NewMongoTransactionRepository(database *mongo.Database) *MongoTransactionRepository {
	return &MongoTransactionRepository{
		collection: database.Collection(TransactionCollectionName),
	}
}

func (repository *MongoTransactionRepository) Create(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	if transaction.ID == bson.NilObjectID {
		transaction.ID = bson.NewObjectID()
	}

	_, err := repository.collection.InsertOne(ctx, transaction)
	if err != nil {
		return model.Transaction{}, err
	}

	return transaction, nil
}

func (repository *MongoTransactionRepository) Find(ctx context.Context, filter TransactionFilter) ([]model.Transaction, error) {
	mongoFilter := bson.M{}

	if filter.Type != "" {
		mongoFilter["type"] = filter.Type
	}

	if filter.StartDate != nil || filter.EndDate != nil {
		dateFilter := bson.M{}
		if filter.StartDate != nil {
			dateFilter["$gte"] = *filter.StartDate
		}
		if filter.EndDate != nil {
			dateFilter["$lt"] = *filter.EndDate
		}

		mongoFilter["transactionDate"] = dateFilter
	}

	cursor, err := repository.collection.Find(
		ctx,
		mongoFilter,
		options.Find().SetSort(bson.D{
			{Key: "transactionDate", Value: -1},
			{Key: "createdAt", Value: -1},
		}),
	)
	if err != nil {
		return nil, err
	}

	var transactions []model.Transaction
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}

	if transactions == nil {
		return []model.Transaction{}, nil
	}

	return transactions, nil
}
