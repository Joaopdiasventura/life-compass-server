package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
)

const (
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
	PeriodAnnual  = "annual"
)

type ValidationError struct {
	Message string
}

func (err *ValidationError) Error() string {
	return err.Message
}

func IsValidationError(err error) bool {
	var validationError *ValidationError
	return errors.As(err, &validationError)
}

type TransactionService struct {
	repository repository.TransactionRepository
}

type dateRange struct {
	Start time.Time
	End   time.Time
}

func NewTransactionService(repository repository.TransactionRepository) *TransactionService {
	return &TransactionService{
		repository: repository,
	}
}

func (service *TransactionService) CreateTransaction(ctx context.Context, request dto.CreateTransactionRequest) (dto.TransactionResponse, error) {
	description := strings.TrimSpace(request.Description)
	if description == "" {
		return dto.TransactionResponse{}, newValidationError("Descrição é obrigatória.")
	}

	category := strings.TrimSpace(request.Category)
	if category == "" {
		return dto.TransactionResponse{}, newValidationError("Categoria é obrigatória.")
	}

	if request.Amount <= 0 {
		return dto.TransactionResponse{}, newValidationError("O valor da transação deve ser maior que zero.")
	}

	transactionType, err := validateRequiredTransactionType(request.Type)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	transactionDate, err := parseRequiredDate(request.TransactionDate, "Data da transação é obrigatória.")
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	transaction := model.Transaction{
		Description:     description,
		Amount:          request.Amount,
		Type:            transactionType,
		Category:        category,
		TransactionDate: transactionDate,
		CreatedAt:       time.Now().UTC(),
	}

	createdTransaction, err := service.repository.Create(ctx, transaction)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	return dto.NewTransactionResponse(createdTransaction), nil
}

func (service *TransactionService) ListTransactions(ctx context.Context, transactionType string) ([]dto.TransactionResponse, error) {
	normalizedType, err := validateOptionalTransactionType(transactionType)
	if err != nil {
		return nil, err
	}

	transactions, err := service.repository.Find(ctx, repository.TransactionFilter{
		Type: normalizedType,
	})
	if err != nil {
		return nil, err
	}

	return dto.NewTransactionResponses(transactions), nil
}

func (service *TransactionService) ListTransactionsByPeriod(ctx context.Context, period string, date string, transactionType string) ([]dto.TransactionResponse, error) {
	normalizedType, err := validateOptionalTransactionType(transactionType)
	if err != nil {
		return nil, err
	}

	periodRange, _, err := buildPeriodRange(period, date)
	if err != nil {
		return nil, err
	}

	transactions, err := service.repository.Find(ctx, repository.TransactionFilter{
		Type:      normalizedType,
		StartDate: &periodRange.Start,
		EndDate:   &periodRange.End,
	})
	if err != nil {
		return nil, err
	}

	return dto.NewTransactionResponses(transactions), nil
}

func (service *TransactionService) GetFinancialSummary(ctx context.Context, period string, date string) (dto.FinancialSummaryResponse, error) {
	periodRange, normalizedPeriod, err := buildPeriodRange(period, date)
	if err != nil {
		return dto.FinancialSummaryResponse{}, err
	}

	transactions, err := service.repository.Find(ctx, repository.TransactionFilter{
		StartDate: &periodRange.Start,
		EndDate:   &periodRange.End,
	})
	if err != nil {
		return dto.FinancialSummaryResponse{}, err
	}

	totalIncome, totalExpense := calculateTotals(transactions)

	return dto.FinancialSummaryResponse{
		Period:           normalizedPeriod,
		StartDate:        periodRange.Start.Format(dto.DateLayout),
		EndDate:          periodRange.End.AddDate(0, 0, -1).Format(dto.DateLayout),
		TotalIncome:      totalIncome,
		TotalExpense:     totalExpense,
		Balance:          totalIncome - totalExpense,
		TransactionCount: len(transactions),
	}, nil
}

func (service *TransactionService) GetTotalBalance(ctx context.Context) (dto.BalanceResponse, error) {
	transactions, err := service.repository.Find(ctx, repository.TransactionFilter{})
	if err != nil {
		return dto.BalanceResponse{}, err
	}

	totalIncome, totalExpense := calculateTotals(transactions)

	return dto.BalanceResponse{
		Balance: totalIncome - totalExpense,
	}, nil
}

func calculateTotals(transactions []model.Transaction) (int64, int64) {
	var totalIncome int64
	var totalExpense int64

	for _, transaction := range transactions {
		switch transaction.Type {
		case model.TransactionTypeIncome:
			totalIncome += transaction.Amount
		case model.TransactionTypeExpense:
			totalExpense += transaction.Amount
		}
	}

	return totalIncome, totalExpense
}

func buildPeriodRange(period string, date string) (dateRange, string, error) {
	normalizedPeriod, err := validatePeriod(period)
	if err != nil {
		return dateRange{}, "", err
	}

	parsedDate, err := parseRequiredDate(date, "Data é obrigatória.")
	if err != nil {
		return dateRange{}, "", err
	}

	switch normalizedPeriod {
	case PeriodDaily:
		return dateRange{
			Start: parsedDate,
			End:   parsedDate.AddDate(0, 0, 1),
		}, normalizedPeriod, nil
	case PeriodWeekly:
		weekday := int(parsedDate.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := parsedDate.AddDate(0, 0, -(weekday - 1))

		return dateRange{
			Start: start,
			End:   start.AddDate(0, 0, 7),
		}, normalizedPeriod, nil
	case PeriodMonthly:
		start := time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, time.UTC)

		return dateRange{
			Start: start,
			End:   start.AddDate(0, 1, 0),
		}, normalizedPeriod, nil
	case PeriodAnnual:
		start := time.Date(parsedDate.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)

		return dateRange{
			Start: start,
			End:   start.AddDate(1, 0, 0),
		}, normalizedPeriod, nil
	default:
		return dateRange{}, "", newValidationError("Período inválido. Use daily, weekly, monthly ou annual.")
	}
}

func parseRequiredDate(value string, requiredMessage string) (time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return time.Time{}, newValidationError(requiredMessage)
	}

	parsedDate, err := time.ParseInLocation(dto.DateLayout, trimmedValue, time.UTC)
	if err != nil {
		return time.Time{}, newValidationError("Data inválida. Use o formato YYYY-MM-DD.")
	}

	return parsedDate, nil
}

func validatePeriod(period string) (string, error) {
	normalizedPeriod := strings.ToLower(strings.TrimSpace(period))
	if normalizedPeriod == "" {
		return "", newValidationError("Período é obrigatório.")
	}

	switch normalizedPeriod {
	case PeriodDaily, PeriodWeekly, PeriodMonthly, PeriodAnnual:
		return normalizedPeriod, nil
	default:
		return "", newValidationError("Período inválido. Use daily, weekly, monthly ou annual.")
	}
}

func validateRequiredTransactionType(transactionType string) (string, error) {
	normalizedType := strings.ToLower(strings.TrimSpace(transactionType))
	if normalizedType == "" {
		return "", newValidationError("Tipo da transação é obrigatório.")
	}

	return validateTransactionType(normalizedType)
}

func validateOptionalTransactionType(transactionType string) (string, error) {
	normalizedType := strings.ToLower(strings.TrimSpace(transactionType))
	if normalizedType == "" {
		return "", nil
	}

	return validateTransactionType(normalizedType)
}

func validateTransactionType(transactionType string) (string, error) {
	switch transactionType {
	case model.TransactionTypeIncome, model.TransactionTypeExpense:
		return transactionType, nil
	default:
		return "", newValidationError("Tipo de transação inválido. Use income ou expense.")
	}
}

func newValidationError(message string) error {
	return &ValidationError{Message: message}
}
