package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

type dateRange struct {
	Start time.Time
	End   time.Time
}

func (service *FinanceService) CreateTransaction(ctx context.Context, request dto.CreateTransactionRequest) (dto.TransactionResponse, error) {
	description, amount, transactionType, category, transactionDate, err := validateTransactionFields(
		request.Description,
		request.Amount,
		request.Type,
		request.Category,
		request.TransactionDate,
	)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	transaction := model.Transaction{
		Description:     description,
		Amount:          amount,
		Type:            transactionType,
		Category:        category,
		TransactionDate: transactionDate,
		CreatedAt:       time.Now().UTC(),
	}

	createdTransaction, err := service.transactionRepository.Create(ctx, transaction)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	return dto.NewTransactionResponse(createdTransaction), nil
}

func (service *FinanceService) GetTransaction(ctx context.Context, id string) (dto.TransactionResponse, error) {
	transactionID, err := parseTransactionID(id)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	transaction, err := service.transactionRepository.FindByID(ctx, transactionID)
	if err != nil {
		return dto.TransactionResponse{}, mapTransactionNotFoundError(err)
	}

	return dto.NewTransactionResponse(transaction), nil
}

func (service *FinanceService) UpdateTransaction(ctx context.Context, id string, request dto.UpdateTransactionRequest) (dto.TransactionResponse, error) {
	transactionID, err := parseTransactionID(id)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	description, amount, transactionType, category, transactionDate, err := validateTransactionFields(
		request.Description,
		request.Amount,
		request.Type,
		request.Category,
		request.TransactionDate,
	)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	transaction, err := service.transactionRepository.FindByID(ctx, transactionID)
	if err != nil {
		return dto.TransactionResponse{}, mapTransactionNotFoundError(err)
	}

	transaction.Description = description
	transaction.Amount = amount
	transaction.Type = transactionType
	transaction.Category = category
	transaction.TransactionDate = transactionDate

	updatedTransaction, err := service.transactionRepository.Update(ctx, transaction)
	if err != nil {
		return dto.TransactionResponse{}, err
	}

	return dto.NewTransactionResponse(updatedTransaction), nil
}

func (service *FinanceService) ListTransactions(ctx context.Context, transactionType string) ([]dto.TransactionResponse, error) {
	normalizedType, err := validateOptionalTransactionType(transactionType)
	if err != nil {
		return nil, err
	}

	transactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
		Type: normalizedType,
	})
	if err != nil {
		return nil, err
	}

	return dto.NewTransactionResponses(transactions), nil
}

func (service *FinanceService) ListTransactionsByPeriod(ctx context.Context, period string, date string, transactionType string) ([]dto.TransactionResponse, error) {
	normalizedType, err := validateOptionalTransactionType(transactionType)
	if err != nil {
		return nil, err
	}

	periodRange, _, err := buildPeriodRange(period, date)
	if err != nil {
		return nil, err
	}

	transactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
		Type:      normalizedType,
		StartDate: &periodRange.Start,
		EndDate:   &periodRange.End,
	})
	if err != nil {
		return nil, err
	}

	return dto.NewTransactionResponses(transactions), nil
}

func (service *FinanceService) GetFinancialSummary(ctx context.Context, period string, date string) (dto.FinancialSummaryResponse, error) {
	periodRange, normalizedPeriod, err := buildPeriodRange(period, date)
	if err != nil {
		return dto.FinancialSummaryResponse{}, err
	}

	transactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
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

func (service *FinanceService) GetTotalBalance(ctx context.Context) (dto.BalanceResponse, error) {
	transactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{})
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

func validateTransactionFields(description string, amount int64, transactionType string, category string, transactionDateValue string) (string, int64, string, string, time.Time, error) {
	trimmedDescription := strings.TrimSpace(description)

	trimmedCategory := strings.TrimSpace(category)
	if trimmedCategory == "" {
		return "", 0, "", "", time.Time{}, newValidationError("Categoria é obrigatória.")
	}

	if amount <= 0 {
		return "", 0, "", "", time.Time{}, newValidationError("O valor da transação deve ser maior que zero.")
	}

	normalizedType, err := validateRequiredTransactionType(transactionType)
	if err != nil {
		return "", 0, "", "", time.Time{}, err
	}

	transactionDate, err := parseRequiredDate(transactionDateValue, "Data da transação é obrigatória.")
	if err != nil {
		return "", 0, "", "", time.Time{}, err
	}

	return trimmedDescription, amount, normalizedType, trimmedCategory, transactionDate, nil
}

func parseTransactionID(id string) (bson.ObjectID, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return bson.NilObjectID, newValidationError("ID da transação é obrigatório.")
	}

	transactionID, err := bson.ObjectIDFromHex(trimmedID)
	if err != nil {
		return bson.NilObjectID, newValidationError("ID da transação inválido.")
	}

	return transactionID, nil
}

func mapTransactionNotFoundError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return newValidationError("Transação não encontrada.")
	}

	return err
}

func newValidationError(message string) error {
	return &ValidationError{Message: message}
}
