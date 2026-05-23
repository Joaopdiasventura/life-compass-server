package repository

import (
	"context"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const GoalCollectionName = "goals"

type GoalFilter struct {
	Status string
}

type GoalRepository interface {
	Create(ctx context.Context, goal model.Goal) (model.Goal, error)
	Find(ctx context.Context, filter GoalFilter) ([]model.Goal, error)
	FindByID(ctx context.Context, id bson.ObjectID) (model.Goal, error)
	Update(ctx context.Context, goal model.Goal) (model.Goal, error)
}

type MongoGoalRepository struct {
	collection *mongo.Collection
}

func NewMongoGoalRepository(database *mongo.Database) *MongoGoalRepository {
	return &MongoGoalRepository{
		collection: database.Collection(GoalCollectionName),
	}
}

func (repository *MongoGoalRepository) Create(ctx context.Context, goal model.Goal) (model.Goal, error) {
	if goal.ID == bson.NilObjectID {
		goal.ID = bson.NewObjectID()
	}

	_, err := repository.collection.InsertOne(ctx, goal)
	if err != nil {
		return model.Goal{}, err
	}

	return goal, nil
}

func (repository *MongoGoalRepository) Find(ctx context.Context, filter GoalFilter) ([]model.Goal, error) {
	mongoFilter := bson.M{}
	if filter.Status != "" {
		mongoFilter["status"] = filter.Status
	}

	cursor, err := repository.collection.Find(
		ctx,
		mongoFilter,
		options.Find().SetSort(bson.D{
			{Key: "status", Value: 1},
			{Key: "dueDate", Value: 1},
			{Key: "createdAt", Value: -1},
		}),
	)
	if err != nil {
		return nil, err
	}

	var goals []model.Goal
	if err := cursor.All(ctx, &goals); err != nil {
		return nil, err
	}

	if goals == nil {
		return []model.Goal{}, nil
	}

	return goals, nil
}

func (repository *MongoGoalRepository) FindByID(ctx context.Context, id bson.ObjectID) (model.Goal, error) {
	var goal model.Goal
	err := repository.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&goal)
	if err != nil {
		return model.Goal{}, err
	}

	return goal, nil
}

func (repository *MongoGoalRepository) Update(ctx context.Context, goal model.Goal) (model.Goal, error) {
	_, err := repository.collection.ReplaceOne(ctx, bson.M{"_id": goal.ID}, goal)
	if err != nil {
		return model.Goal{}, err
	}

	return goal, nil
}
