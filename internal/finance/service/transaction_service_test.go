package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeTransactionRepository struct {
	created      []model.Transaction
	transactions []model.Transaction
	lastFilter   repository.TransactionFilter
	createErr    error
	findErr      error
}

func (fake *fakeTransactionRepository) Create(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	if fake.createErr != nil {
		return model.Transaction{}, fake.createErr
	}

	if transaction.ID == bson.NilObjectID {
		transaction.ID = bson.NewObjectID()
	}

	fake.created = append(fake.created, transaction)

	return transaction, nil
}

func (fake *fakeTransactionRepository) Find(ctx context.Context, filter repository.TransactionFilter) ([]model.Transaction, error) {
	fake.lastFilter = filter
	if fake.findErr != nil {
		return nil, fake.findErr
	}

	transactions := make([]model.Transaction, 0, len(fake.transactions))
	for _, transaction := range fake.transactions {
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

func (fake *fakeTransactionRepository) FindByID(ctx context.Context, id bson.ObjectID) (model.Transaction, error) {
	for _, transaction := range fake.transactions {
		if transaction.ID == id {
			return transaction, nil
		}
	}

	return model.Transaction{}, mongo.ErrNoDocuments
}

func (fake *fakeTransactionRepository) Update(ctx context.Context, transaction model.Transaction) (model.Transaction, error) {
	for index, currentTransaction := range fake.transactions {
		if currentTransaction.ID == transaction.ID {
			fake.transactions[index] = transaction
			return transaction, nil
		}
	}

	return model.Transaction{}, mongo.ErrNoDocuments
}

func newTestFinanceService(transactionRepository repository.TransactionRepository, goalRepositories ...repository.GoalRepository) *FinanceService {
	var goalRepository repository.GoalRepository
	if len(goalRepositories) > 0 {
		goalRepository = goalRepositories[0]
	}

	return NewFinanceService(FinanceServiceDependencies{
		TransactionRepository: transactionRepository,
		GoalRepository:        goalRepository,
	})
}

func TestCreateTransactionAcceptsValidIncomeAndExpense(t *testing.T) {
	tests := []struct {
		name            string
		transactionType string
		amount          int64
	}{
		{name: "income", transactionType: model.TransactionTypeIncome, amount: 500000},
		{name: "expense", transactionType: model.TransactionTypeExpense, amount: 2590},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeTransactionRepository{}
			service := newTestFinanceService(repository)

			response, err := service.CreateTransaction(context.Background(), dto.CreateTransactionRequest{
				Description:     "Monthly transaction",
				Amount:          test.amount,
				Type:            test.transactionType,
				Category:        "general",
				TransactionDate: "2026-05-22",
			})
			if err != nil {
				t.Fatalf("CreateTransaction returned error: %v", err)
			}

			if len(repository.created) != 1 {
				t.Fatalf("expected 1 created transaction, got %d", len(repository.created))
			}

			created := repository.created[0]
			if created.Description != "Monthly transaction" {
				t.Fatalf("expected trimmed description to be saved")
			}
			if created.Amount != test.amount {
				t.Fatalf("expected amount %d, got %d", test.amount, created.Amount)
			}
			if created.Type != test.transactionType {
				t.Fatalf("expected type %q, got %q", test.transactionType, created.Type)
			}
			if created.TransactionDate.Format(dto.DateLayout) != "2026-05-22" {
				t.Fatalf("expected transaction date 2026-05-22, got %s", created.TransactionDate.Format(dto.DateLayout))
			}
			if created.CreatedAt.IsZero() {
				t.Fatalf("expected createdAt to be set")
			}
			if response.Amount != test.amount {
				t.Fatalf("expected response amount %d, got %d", test.amount, response.Amount)
			}
		})
	}
}

func TestCreateTransactionValidation(t *testing.T) {
	baseRequest := dto.CreateTransactionRequest{
		Description:     "Transaction",
		Amount:          1000,
		Type:            model.TransactionTypeIncome,
		Category:        "general",
		TransactionDate: "2026-05-22",
	}

	tests := []struct {
		name   string
		change func(*dto.CreateTransactionRequest)
	}{
		{name: "missing category", change: func(request *dto.CreateTransactionRequest) { request.Category = " " }},
		{name: "zero amount", change: func(request *dto.CreateTransactionRequest) { request.Amount = 0 }},
		{name: "negative amount", change: func(request *dto.CreateTransactionRequest) { request.Amount = -1 }},
		{name: "missing type", change: func(request *dto.CreateTransactionRequest) { request.Type = "" }},
		{name: "invalid type", change: func(request *dto.CreateTransactionRequest) { request.Type = "transfer" }},
		{name: "missing transaction date", change: func(request *dto.CreateTransactionRequest) { request.TransactionDate = "" }},
		{name: "invalid transaction date", change: func(request *dto.CreateTransactionRequest) { request.TransactionDate = "22/05/2026" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := baseRequest
			test.change(&request)

			repository := &fakeTransactionRepository{}
			service := newTestFinanceService(repository)

			_, err := service.CreateTransaction(context.Background(), request)
			if err == nil {
				t.Fatalf("expected validation error")
			}
			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T", err)
			}
			if len(repository.created) != 0 {
				t.Fatalf("expected no transaction to be created")
			}
		})
	}
}

func TestCreateTransactionAllowsEmptyDescription(t *testing.T) {
	repository := &fakeTransactionRepository{}
	service := newTestFinanceService(repository)

	response, err := service.CreateTransaction(context.Background(), dto.CreateTransactionRequest{
		Description:     " ",
		Amount:          1000,
		Type:            model.TransactionTypeExpense,
		Category:        "Alimentação",
		TransactionDate: "2026-05-22",
	})
	if err != nil {
		t.Fatalf("CreateTransaction returned error: %v", err)
	}

	if response.Description != "" {
		t.Fatalf("expected empty description, got %q", response.Description)
	}
	if len(repository.created) != 1 || repository.created[0].Description != "" {
		t.Fatalf("expected saved transaction with empty description")
	}
}

func TestListTransactionsRejectsInvalidTypeFilters(t *testing.T) {
	repository := &fakeTransactionRepository{}
	service := newTestFinanceService(repository)

	if _, err := service.ListTransactions(context.Background(), "transfer"); !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid list type, got %v", err)
	}

	if _, err := service.ListTransactionsByPeriod(context.Background(), PeriodDaily, "2026-05-22", "transfer"); !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid period list type, got %v", err)
	}
}

func TestGetAndUpdateTransaction(t *testing.T) {
	transactionID := bson.NewObjectID()
	repository := &fakeTransactionRepository{
		transactions: []model.Transaction{
			{
				ID:              transactionID,
				Description:     "Café",
				Amount:          2590,
				Type:            model.TransactionTypeExpense,
				Category:        "Alimentação",
				TransactionDate: mustDate("2026-05-22"),
				CreatedAt:       mustDate("2026-05-22"),
			},
		},
	}
	service := newTestFinanceService(repository)

	transaction, err := service.GetTransaction(context.Background(), transactionID.Hex())
	if err != nil {
		t.Fatalf("GetTransaction returned error: %v", err)
	}
	if transaction.Description != "Café" {
		t.Fatalf("expected transaction to be loaded, got %+v", transaction)
	}

	updatedTransaction, err := service.UpdateTransaction(context.Background(), transactionID.Hex(), dto.UpdateTransactionRequest{
		Description:     "Almoço",
		Amount:          4290,
		Type:            model.TransactionTypeExpense,
		Category:        "Alimentação",
		TransactionDate: "2026-05-23",
	})
	if err != nil {
		t.Fatalf("UpdateTransaction returned error: %v", err)
	}
	if updatedTransaction.Description != "Almoço" || updatedTransaction.Amount != 4290 {
		t.Fatalf("expected updated transaction, got %+v", updatedTransaction)
	}
	if repository.transactions[0].CreatedAt.IsZero() {
		t.Fatalf("expected createdAt to be preserved")
	}
}

func TestListTransactionsByPeriodBuildsDateRanges(t *testing.T) {
	tests := []struct {
		name      string
		period    string
		startDate string
		endDate   string
	}{
		{name: "daily", period: PeriodDaily, startDate: "2026-05-22", endDate: "2026-05-23"},
		{name: "weekly", period: PeriodWeekly, startDate: "2026-05-18", endDate: "2026-05-25"},
		{name: "monthly", period: PeriodMonthly, startDate: "2026-05-01", endDate: "2026-06-01"},
		{name: "annual", period: PeriodAnnual, startDate: "2026-01-01", endDate: "2027-01-01"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeTransactionRepository{}
			service := newTestFinanceService(repository)

			_, err := service.ListTransactionsByPeriod(context.Background(), test.period, "2026-05-22", model.TransactionTypeIncome)
			if err != nil {
				t.Fatalf("ListTransactionsByPeriod returned error: %v", err)
			}

			assertDate(t, repository.lastFilter.StartDate, test.startDate)
			assertDate(t, repository.lastFilter.EndDate, test.endDate)
			if repository.lastFilter.Type != model.TransactionTypeIncome {
				t.Fatalf("expected type filter %q, got %q", model.TransactionTypeIncome, repository.lastFilter.Type)
			}
		})
	}
}

func TestGetFinancialSummaryCalculatesTotalsForPeriod(t *testing.T) {
	repository := &fakeTransactionRepository{
		transactions: []model.Transaction{
			{Amount: 10000, Type: model.TransactionTypeIncome, TransactionDate: mustDate("2026-05-01")},
			{Amount: 5000, Type: model.TransactionTypeIncome, TransactionDate: mustDate("2026-05-31")},
			{Amount: 2500, Type: model.TransactionTypeExpense, TransactionDate: mustDate("2026-05-22")},
			{Amount: 9999, Type: model.TransactionTypeExpense, TransactionDate: mustDate("2026-06-01")},
		},
	}
	service := newTestFinanceService(repository)

	summary, err := service.GetFinancialSummary(context.Background(), PeriodMonthly, "2026-05-22")
	if err != nil {
		t.Fatalf("GetFinancialSummary returned error: %v", err)
	}

	if summary.Period != PeriodMonthly {
		t.Fatalf("expected period %q, got %q", PeriodMonthly, summary.Period)
	}
	if summary.StartDate != "2026-05-01" {
		t.Fatalf("expected start date 2026-05-01, got %s", summary.StartDate)
	}
	if summary.EndDate != "2026-05-31" {
		t.Fatalf("expected end date 2026-05-31, got %s", summary.EndDate)
	}
	if summary.TotalIncome != 15000 {
		t.Fatalf("expected total income 15000, got %d", summary.TotalIncome)
	}
	if summary.TotalExpense != 2500 {
		t.Fatalf("expected total expense 2500, got %d", summary.TotalExpense)
	}
	if summary.Balance != 12500 {
		t.Fatalf("expected balance 12500, got %d", summary.Balance)
	}
	if summary.TransactionCount != 3 {
		t.Fatalf("expected transaction count 3, got %d", summary.TransactionCount)
	}
}

func TestGetTotalBalanceCalculatesConsolidatedBalance(t *testing.T) {
	repository := &fakeTransactionRepository{
		transactions: []model.Transaction{
			{Amount: 1000, Type: model.TransactionTypeIncome, TransactionDate: mustDate("2026-05-20")},
			{Amount: 250, Type: model.TransactionTypeExpense, TransactionDate: mustDate("2026-05-21")},
			{Amount: 50, Type: model.TransactionTypeExpense, TransactionDate: mustDate("2026-05-22")},
		},
	}
	service := newTestFinanceService(repository)

	balance, err := service.GetTotalBalance(context.Background())
	if err != nil {
		t.Fatalf("GetTotalBalance returned error: %v", err)
	}

	if balance.Balance != 700 {
		t.Fatalf("expected balance 700, got %d", balance.Balance)
	}
}

func TestRepositoryErrorsAreReturned(t *testing.T) {
	expectedErr := errors.New("repository error")
	repository := &fakeTransactionRepository{findErr: expectedErr}
	service := newTestFinanceService(repository)

	_, err := service.GetTotalBalance(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func assertDate(t *testing.T, date *time.Time, expected string) {
	t.Helper()

	if date == nil {
		t.Fatalf("expected date %s, got nil", expected)
	}
	if date.Format(dto.DateLayout) != expected {
		t.Fatalf("expected date %s, got %s", expected, date.Format(dto.DateLayout))
	}
}

func mustDate(value string) time.Time {
	parsedDate, err := time.ParseInLocation(dto.DateLayout, value, time.UTC)
	if err != nil {
		panic(err)
	}

	return parsedDate
}
