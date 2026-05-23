package service

import (
	"context"
	"testing"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeGoalRepository struct {
	goals     []model.Goal
	createErr error
	findErr   error
	updateErr error
}

func (fake *fakeGoalRepository) Create(ctx context.Context, goal model.Goal) (model.Goal, error) {
	if fake.createErr != nil {
		return model.Goal{}, fake.createErr
	}

	if goal.ID == bson.NilObjectID {
		goal.ID = bson.NewObjectID()
	}

	fake.goals = append(fake.goals, goal)

	return goal, nil
}

func (fake *fakeGoalRepository) Find(ctx context.Context, filter repository.GoalFilter) ([]model.Goal, error) {
	if fake.findErr != nil {
		return nil, fake.findErr
	}

	goals := make([]model.Goal, 0, len(fake.goals))
	for _, goal := range fake.goals {
		if filter.Status != "" && goal.Status != filter.Status {
			continue
		}

		goals = append(goals, goal)
	}

	return goals, nil
}

func (fake *fakeGoalRepository) FindByID(ctx context.Context, id bson.ObjectID) (model.Goal, error) {
	for _, goal := range fake.goals {
		if goal.ID == id {
			return goal, nil
		}
	}

	return model.Goal{}, mongo.ErrNoDocuments
}

func (fake *fakeGoalRepository) Update(ctx context.Context, goal model.Goal) (model.Goal, error) {
	if fake.updateErr != nil {
		return model.Goal{}, fake.updateErr
	}

	for index, currentGoal := range fake.goals {
		if currentGoal.ID == goal.ID {
			fake.goals[index] = goal
			return goal, nil
		}
	}

	return model.Goal{}, mongo.ErrNoDocuments
}

func TestGoalLifecycleValidationAndCompletion(t *testing.T) {
	transactionRepository := &fakeTransactionRepository{}
	goalRepository := &fakeGoalRepository{}
	service := newTestFinanceService(transactionRepository, goalRepository)

	createdGoal, err := service.CreateGoal(context.Background(), dto.CreateGoalRequest{
		Description:   "Reserva de emergência",
		TargetAmount:  100000,
		CurrentAmount: 25000,
		DueDate:       "2026-06-30",
	})
	if err != nil {
		t.Fatalf("CreateGoal returned error: %v", err)
	}
	if createdGoal.Status != model.GoalStatusActive {
		t.Fatalf("expected active goal, got %q", createdGoal.Status)
	}

	updatedGoal, err := service.UpdateGoal(context.Background(), createdGoal.ID, dto.UpdateGoalRequest{
		Description:   "Reserva completa",
		TargetAmount:  120000,
		CurrentAmount: 80000,
		DueDate:       "2026-07-15",
	})
	if err != nil {
		t.Fatalf("UpdateGoal returned error: %v", err)
	}
	if updatedGoal.Description != "Reserva completa" {
		t.Fatalf("expected updated description, got %q", updatedGoal.Description)
	}

	completedGoal, err := service.CompleteGoal(context.Background(), createdGoal.ID)
	if err != nil {
		t.Fatalf("CompleteGoal returned error: %v", err)
	}
	if completedGoal.Status != model.GoalStatusCompleted {
		t.Fatalf("expected completed goal, got %q", completedGoal.Status)
	}
	if completedGoal.CurrentAmount != completedGoal.TargetAmount {
		t.Fatalf("expected current amount to reach target on completion")
	}
	if completedGoal.CompletedAt == "" {
		t.Fatalf("expected completedAt to be set")
	}
}

func TestGoalValidationRejectsInvalidValues(t *testing.T) {
	service := newTestFinanceService(&fakeTransactionRepository{}, &fakeGoalRepository{})

	tests := []dto.CreateGoalRequest{
		{Description: " ", TargetAmount: 1000, CurrentAmount: 0, DueDate: "2026-06-30"},
		{Description: "Meta", TargetAmount: 0, CurrentAmount: 0, DueDate: "2026-06-30"},
		{Description: "Meta", TargetAmount: 1000, CurrentAmount: -1, DueDate: "2026-06-30"},
		{Description: "Meta", TargetAmount: 1000, CurrentAmount: 0, DueDate: "30/06/2026"},
	}

	for _, request := range tests {
		if _, err := service.CreateGoal(context.Background(), request); !IsValidationError(err) {
			t.Fatalf("expected validation error for request %+v, got %v", request, err)
		}
	}
}

func TestFinancialDashboardReturnsConsolidatedCardsAndAlerts(t *testing.T) {
	goalID := bson.NewObjectID()
	transactionRepository := &fakeTransactionRepository{
		transactions: []model.Transaction{
			{Amount: 200000, Type: model.TransactionTypeIncome, Category: "Salário", TransactionDate: mustDate("2026-05-01")},
			{Amount: 70000, Type: model.TransactionTypeExpense, Category: "Moradia", TransactionDate: mustDate("2026-05-03")},
			{Amount: 30000, Type: model.TransactionTypeExpense, Category: "Alimentação", TransactionDate: mustDate("2026-05-08")},
		},
	}
	goalRepository := &fakeGoalRepository{
		goals: []model.Goal{
			{
				ID:            goalID,
				Description:   "Viagem",
				TargetAmount:  100000,
				CurrentAmount: 60000,
				DueDate:       mustDate("2026-05-25"),
				Status:        model.GoalStatusActive,
				CreatedAt:     mustDate("2026-05-01"),
				UpdatedAt:     mustDate("2026-05-01"),
			},
		},
	}
	service := newTestFinanceService(transactionRepository, goalRepository)

	dashboard, err := service.GetFinancialDashboard(context.Background(), PeriodMonthly, "2026-05-22", "90000", "50")
	if err != nil {
		t.Fatalf("GetFinancialDashboard returned error: %v", err)
	}

	if dashboard.Balance != 100000 {
		t.Fatalf("expected total balance 100000, got %d", dashboard.Balance)
	}
	if dashboard.TotalIncome != 200000 || dashboard.TotalExpense != 100000 || dashboard.Savings != 100000 {
		t.Fatalf("expected monthly totals to be calculated, got %+v", dashboard)
	}
	if dashboard.TopExpenseCategory == nil || dashboard.TopExpenseCategory.Category != "Moradia" {
		t.Fatalf("expected Moradia as top category, got %+v", dashboard.TopExpenseCategory)
	}
	if len(dashboard.Alerts) != 4 {
		t.Fatalf("expected 4 alerts, got %d: %+v", len(dashboard.Alerts), dashboard.Alerts)
	}
}

func TestFinancialChartsCalculateBucketsAndBalanceEvolution(t *testing.T) {
	transactionRepository := &fakeTransactionRepository{
		transactions: []model.Transaction{
			{Amount: 100000, Type: model.TransactionTypeIncome, Category: "Salário", TransactionDate: mustDate("2026-04-15")},
			{Amount: 200000, Type: model.TransactionTypeIncome, Category: "Salário", TransactionDate: mustDate("2026-05-01")},
			{Amount: 50000, Type: model.TransactionTypeExpense, Category: "Moradia", TransactionDate: mustDate("2026-05-08")},
			{Amount: 25000, Type: model.TransactionTypeExpense, Category: "Alimentação", TransactionDate: mustDate("2026-05-15")},
		},
	}
	service := newTestFinanceService(transactionRepository, &fakeGoalRepository{})

	categories, err := service.GetCategoryExpensesChart(context.Background(), PeriodMonthly, "2026-05-22")
	if err != nil {
		t.Fatalf("GetCategoryExpensesChart returned error: %v", err)
	}
	if len(categories.Categories) != 2 || categories.Categories[0].Category != "Moradia" {
		t.Fatalf("expected category expenses sorted by total, got %+v", categories.Categories)
	}

	incomeExpense, err := service.GetIncomeExpenseChart(context.Background(), PeriodMonthly, "2026-05-22")
	if err != nil {
		t.Fatalf("GetIncomeExpenseChart returned error: %v", err)
	}
	if len(incomeExpense.Points) != 5 {
		t.Fatalf("expected 5 weekly buckets for May 2026, got %d", len(incomeExpense.Points))
	}
	if incomeExpense.Points[0].TotalIncome != 200000 || incomeExpense.Points[1].TotalExpense != 50000 {
		t.Fatalf("expected transactions to fall into weekly buckets, got %+v", incomeExpense.Points)
	}

	evolution, err := service.GetBalanceEvolutionChart(context.Background(), PeriodMonthly, "2026-05-22")
	if err != nil {
		t.Fatalf("GetBalanceEvolutionChart returned error: %v", err)
	}
	if evolution.Points[0].Balance != 300000 {
		t.Fatalf("expected first balance to include prior balance and first bucket income, got %d", evolution.Points[0].Balance)
	}
	if evolution.Points[2].Balance != 225000 {
		t.Fatalf("expected balance after third bucket to be 225000, got %d", evolution.Points[2].Balance)
	}
}
