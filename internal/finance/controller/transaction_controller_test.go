package controller

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/service"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type controllerTransactionRepository struct {
	transactions []model.Transaction
}

func (repository *controllerTransactionRepository) Create(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	if transaction.ID == bson.NilObjectID {
		transaction.ID = bson.NewObjectID()
	}

	repository.transactions = append(repository.transactions, transaction)

	return transaction, nil
}

func (repository *controllerTransactionRepository) Find(ctx context.Context, filter repository.TransactionFilter) ([]model.Transaction, error) {
	transactions := make([]model.Transaction, 0, len(repository.transactions))
	for _, transaction := range repository.transactions {
		if filter.Type != "" && transaction.Type != filter.Type {
			continue
		}
		if filter.StartDate != nil && transaction.TransactionDate.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && !transaction.TransactionDate.Before(*filter.EndDate) {
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (repository *controllerTransactionRepository) FindByID(ctx context.Context, id bson.ObjectID) (model.Transaction, error) {
	for _, transaction := range repository.transactions {
		if transaction.ID == id {
			return transaction, nil
		}
	}

	return model.Transaction{}, mongo.ErrNoDocuments
}

func (repository *controllerTransactionRepository) Update(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	for index, currentTransaction := range repository.transactions {
		if currentTransaction.ID == transaction.ID {
			repository.transactions[index] = transaction
			return transaction, nil
		}
	}

	return model.Transaction{}, mongo.ErrNoDocuments
}

type controllerGoalRepository struct {
	goals []model.Goal
}

func (repository *controllerGoalRepository) Create(ctx context.Context, goal model.Goal) (model.Goal, error) {
	if goal.ID == bson.NilObjectID {
		goal.ID = bson.NewObjectID()
	}

	repository.goals = append(repository.goals, goal)

	return goal, nil
}

func (repository *controllerGoalRepository) Find(ctx context.Context, filter repository.GoalFilter) ([]model.Goal, error) {
	goals := make([]model.Goal, 0, len(repository.goals))
	for _, goal := range repository.goals {
		if filter.Status != "" && goal.Status != filter.Status {
			continue
		}

		goals = append(goals, goal)
	}

	return goals, nil
}

func (repository *controllerGoalRepository) FindByID(ctx context.Context, id bson.ObjectID) (model.Goal, error) {
	for _, goal := range repository.goals {
		if goal.ID == id {
			return goal, nil
		}
	}

	return model.Goal{}, mongo.ErrNoDocuments
}

func (repository *controllerGoalRepository) Update(ctx context.Context, goal model.Goal) (model.Goal, error) {
	for index, currentGoal := range repository.goals {
		if currentGoal.ID == goal.ID {
			repository.goals[index] = goal
			return goal, nil
		}
	}

	return model.Goal{}, mongo.ErrNoDocuments
}

func TestControllerServesFinancialDashboardEndpoint(t *testing.T) {
	mux := newTestMux(
		&controllerTransactionRepository{
			transactions: []model.Transaction{
				{Amount: 500000, Type: model.TransactionTypeIncome, Category: "Salário", TransactionDate: mustControllerDate("2026-05-01")},
				{Amount: 125000, Type: model.TransactionTypeExpense, Category: "Casa", TransactionDate: mustControllerDate("2026-05-03")},
			},
		},
		&controllerGoalRepository{},
	)
	request := httptest.NewRequest(http.MethodGet, "/financial-dashboard?period=monthly&date=2026-05-22&monthlyExpenseLimit=100000", nil)
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Dashboard financeiro consultado com sucesso.") {
		t.Fatalf("expected dashboard success message, got %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "\"topExpenseCategory\"") {
		t.Fatalf("expected dashboard payload, got %s", response.Body.String())
	}
}

func TestControllerCreatesAndCompletesGoals(t *testing.T) {
	goalRepository := &controllerGoalRepository{}
	mux := newTestMux(&controllerTransactionRepository{}, goalRepository)
	request := httptest.NewRequest(http.MethodPost, "/goals", bytes.NewBufferString(`{
		"description": "Reserva",
		"targetAmount": 100000,
		"currentAmount": 10000,
		"dueDate": "2026-06-30"
	}`))
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if len(goalRepository.goals) != 1 {
		t.Fatalf("expected goal to be persisted")
	}

	goalID := goalRepository.goals[0].ID.Hex()
	completeRequest := httptest.NewRequest(http.MethodPatch, "/goals/"+goalID+"/complete", nil)
	completeResponse := httptest.NewRecorder()

	mux.ServeHTTP(completeResponse, completeRequest)

	if completeResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", completeResponse.Code, completeResponse.Body.String())
	}
	if goalRepository.goals[0].Status != model.GoalStatusCompleted {
		t.Fatalf("expected goal to be completed")
	}
}

func TestControllerUpdatesTransactions(t *testing.T) {
	transactionID := bson.NewObjectID()
	transactionRepository := &controllerTransactionRepository{
		transactions: []model.Transaction{
			{
				ID:              transactionID,
				Description:     "Café",
				Amount:          2500,
				Type:            model.TransactionTypeExpense,
				Category:        "Alimentação",
				TransactionDate: mustControllerDate("2026-05-22"),
				CreatedAt:       mustControllerDate("2026-05-22"),
			},
		},
	}
	mux := newTestMux(transactionRepository, &controllerGoalRepository{})
	request := httptest.NewRequest(http.MethodPut, "/transactions/"+transactionID.Hex(), bytes.NewBufferString(`{
		"description": "Almoço",
		"amount": 4290,
		"type": "expense",
		"category": "Alimentação",
		"transactionDate": "2026-05-23"
	}`))
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if transactionRepository.transactions[0].Description != "Almoço" {
		t.Fatalf("expected transaction to be updated")
	}
	if !strings.Contains(response.Body.String(), "Transação atualizada com sucesso.") {
		t.Fatalf("expected update success message, got %s", response.Body.String())
	}
}

func newTestMux(transactionRepository repository.TransactionRepository, goalRepository repository.GoalRepository) *http.ServeMux {
	financeService := service.NewFinanceService(service.FinanceServiceDependencies{
		TransactionRepository: transactionRepository,
		GoalRepository:        goalRepository,
	})
	financeController := NewFinanceController(financeService)
	mux := http.NewServeMux()
	financeController.RegisterRoutes(mux)

	return mux
}

func mustControllerDate(value string) time.Time {
	parsedDate, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil {
		panic(err)
	}

	return parsedDate
}
