package dto

import (
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const DateLayout = "2006-01-02"

type CreateTransactionRequest struct {
	Description     string `json:"description"`
	Amount          int64  `json:"amount"`
	Type            string `json:"type"`
	Category        string `json:"category"`
	TransactionDate string `json:"transactionDate"`
}

type TransactionResponse struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	Amount          int64  `json:"amount"`
	Type            string `json:"type"`
	Category        string `json:"category"`
	TransactionDate string `json:"transactionDate"`
	CreatedAt       string `json:"createdAt"`
}

type FinancialSummaryResponse struct {
	Period           string `json:"period"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
	TotalIncome      int64  `json:"totalIncome"`
	TotalExpense     int64  `json:"totalExpense"`
	Balance          int64  `json:"balance"`
	TransactionCount int    `json:"transactionCount"`
}

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

type APIResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func NewTransactionResponse(transaction model.Transaction) TransactionResponse {
	id := ""
	if transaction.ID != bson.NilObjectID {
		id = transaction.ID.Hex()
	}

	return TransactionResponse{
		ID:              id,
		Description:     transaction.Description,
		Amount:          transaction.Amount,
		Type:            transaction.Type,
		Category:        transaction.Category,
		TransactionDate: formatTime(transaction.TransactionDate, DateLayout),
		CreatedAt:       formatTime(transaction.CreatedAt, time.RFC3339),
	}
}

func NewTransactionResponses(transactions []model.Transaction) []TransactionResponse {
	responses := make([]TransactionResponse, 0, len(transactions))
	for _, transaction := range transactions {
		responses = append(responses, NewTransactionResponse(transaction))
	}

	return responses
}

func formatTime(value time.Time, layout string) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(layout)
}
