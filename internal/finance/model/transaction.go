package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	TransactionTypeIncome  = "income"
	TransactionTypeExpense = "expense"
)

type Transaction struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	Description     string        `bson:"description"`
	Amount          int64         `bson:"amount"`
	Type            string        `bson:"type"`
	Category        string        `bson:"category"`
	TransactionDate time.Time     `bson:"transactionDate"`
	CreatedAt       time.Time     `bson:"createdAt"`
}
