package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	GoalStatusActive    = "active"
	GoalStatusCompleted = "completed"
)

type Goal struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	Description   string        `bson:"description"`
	TargetAmount  int64         `bson:"targetAmount"`
	CurrentAmount int64         `bson:"currentAmount"`
	DueDate       time.Time     `bson:"dueDate"`
	Status        string        `bson:"status"`
	CompletedAt   *time.Time    `bson:"completedAt,omitempty"`
	CreatedAt     time.Time     `bson:"createdAt"`
	UpdatedAt     time.Time     `bson:"updatedAt"`
}
