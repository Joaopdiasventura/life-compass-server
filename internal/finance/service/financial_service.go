package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
)

const (
	defaultCategoryExpenseShareLimit = 40
	goalDueSoonDays                  = 7
)

type alertOptions struct {
	MonthlyExpenseLimit       int64
	CategoryExpenseShareLimit float64
}

type chartBucket struct {
	Label string
	Start time.Time
	End   time.Time
}

func (service *FinanceService) GetFinancialDashboard(ctx context.Context, period string, date string, monthlyExpenseLimit string, categoryExpenseShareLimit string) (dto.FinancialDashboardResponse, error) {
	periodRange, normalizedPeriod, err := buildPeriodRange(period, date)
	if err != nil {
		return dto.FinancialDashboardResponse{}, err
	}

	transactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
		StartDate: &periodRange.Start,
		EndDate:   &periodRange.End,
	})
	if err != nil {
		return dto.FinancialDashboardResponse{}, err
	}

	totalIncome, totalExpense := calculateTotals(transactions)
	balance, err := service.GetTotalBalance(ctx)
	if err != nil {
		return dto.FinancialDashboardResponse{}, err
	}

	alerts, err := service.GetFinancialAlerts(ctx, date, monthlyExpenseLimit, categoryExpenseShareLimit)
	if err != nil {
		return dto.FinancialDashboardResponse{}, err
	}

	startDate, endDate := formatPeriodBounds(periodRange)

	return dto.FinancialDashboardResponse{
		Period:             normalizedPeriod,
		StartDate:          startDate,
		EndDate:            endDate,
		Balance:            balance.Balance,
		TotalIncome:        totalIncome,
		TotalExpense:       totalExpense,
		Savings:            totalIncome - totalExpense,
		TransactionCount:   len(transactions),
		TopExpenseCategory: topExpenseCategory(transactions),
		Alerts:             alerts,
	}, nil
}

func (service *FinanceService) GetCategoryExpensesChart(ctx context.Context, period string, date string) (dto.CategoryExpensesChartResponse, error) {
	periodRange, normalizedPeriod, transactions, err := service.findTransactionsForPeriod(ctx, period, date)
	if err != nil {
		return dto.CategoryExpensesChartResponse{}, err
	}

	startDate, endDate := formatPeriodBounds(periodRange)

	return dto.CategoryExpensesChartResponse{
		Period:     normalizedPeriod,
		StartDate:  startDate,
		EndDate:    endDate,
		Categories: expenseCategories(transactions),
	}, nil
}

func (service *FinanceService) GetIncomeExpenseChart(ctx context.Context, period string, date string) (dto.IncomeExpenseChartResponse, error) {
	periodRange, normalizedPeriod, transactions, err := service.findTransactionsForPeriod(ctx, period, date)
	if err != nil {
		return dto.IncomeExpenseChartResponse{}, err
	}

	buckets := buildChartBuckets(periodRange, normalizedPeriod)
	points := make([]dto.IncomeExpensePointResponse, 0, len(buckets))
	for _, bucket := range buckets {
		totalIncome, totalExpense := calculateTotals(filterTransactionsByRange(transactions, bucket.Start, bucket.End))
		points = append(points, dto.IncomeExpensePointResponse{
			Label:        bucket.Label,
			StartDate:    bucket.Start.Format(dto.DateLayout),
			EndDate:      bucket.End.AddDate(0, 0, -1).Format(dto.DateLayout),
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
			Balance:      totalIncome - totalExpense,
		})
	}

	startDate, endDate := formatPeriodBounds(periodRange)

	return dto.IncomeExpenseChartResponse{
		Period:    normalizedPeriod,
		StartDate: startDate,
		EndDate:   endDate,
		Points:    points,
	}, nil
}

func (service *FinanceService) GetBalanceEvolutionChart(ctx context.Context, period string, date string) (dto.BalanceEvolutionChartResponse, error) {
	periodRange, normalizedPeriod, transactions, err := service.findTransactionsForPeriod(ctx, period, date)
	if err != nil {
		return dto.BalanceEvolutionChartResponse{}, err
	}

	previousTransactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{EndDate: &periodRange.Start})
	if err != nil {
		return dto.BalanceEvolutionChartResponse{}, err
	}

	previousIncome, previousExpense := calculateTotals(previousTransactions)
	currentBalance := previousIncome - previousExpense

	buckets := buildChartBuckets(periodRange, normalizedPeriod)
	points := make([]dto.BalanceEvolutionPointResponse, 0, len(buckets))
	for _, bucket := range buckets {
		totalIncome, totalExpense := calculateTotals(filterTransactionsByRange(transactions, bucket.Start, bucket.End))
		currentBalance += totalIncome - totalExpense
		points = append(points, dto.BalanceEvolutionPointResponse{
			Label:        bucket.Label,
			Date:         bucket.End.AddDate(0, 0, -1).Format(dto.DateLayout),
			Balance:      currentBalance,
			TotalIncome:  totalIncome,
			TotalExpense: totalExpense,
		})
	}

	startDate, endDate := formatPeriodBounds(periodRange)

	return dto.BalanceEvolutionChartResponse{
		Period:    normalizedPeriod,
		StartDate: startDate,
		EndDate:   endDate,
		Points:    points,
	}, nil
}

func (service *FinanceService) GetPeriodExpensesChart(ctx context.Context, period string, date string) (dto.PeriodExpensesChartResponse, error) {
	periodRange, normalizedPeriod, transactions, err := service.findTransactionsForPeriod(ctx, period, date)
	if err != nil {
		return dto.PeriodExpensesChartResponse{}, err
	}

	buckets := buildChartBuckets(periodRange, normalizedPeriod)
	points := make([]dto.PeriodExpensePointResponse, 0, len(buckets))
	for _, bucket := range buckets {
		_, totalExpense := calculateTotals(filterTransactionsByRange(transactions, bucket.Start, bucket.End))
		points = append(points, dto.PeriodExpensePointResponse{
			Label:        bucket.Label,
			StartDate:    bucket.Start.Format(dto.DateLayout),
			EndDate:      bucket.End.AddDate(0, 0, -1).Format(dto.DateLayout),
			TotalExpense: totalExpense,
		})
	}

	startDate, endDate := formatPeriodBounds(periodRange)

	return dto.PeriodExpensesChartResponse{
		Period:    normalizedPeriod,
		StartDate: startDate,
		EndDate:   endDate,
		Points:    points,
	}, nil
}

func (service *FinanceService) GetFinancialAlerts(ctx context.Context, date string, monthlyExpenseLimit string, categoryExpenseShareLimit string) ([]dto.FinancialAlertResponse, error) {
	parsedDate, err := parseRequiredDate(date, "Data é obrigatória.")
	if err != nil {
		return nil, err
	}

	options, err := parseAlertOptions(monthlyExpenseLimit, categoryExpenseShareLimit)
	if err != nil {
		return nil, err
	}

	monthlyRange, _, err := buildPeriodRange(PeriodMonthly, date)
	if err != nil {
		return nil, err
	}

	dailyRange := dateRange{
		Start: parsedDate,
		End:   parsedDate.AddDate(0, 0, 1),
	}

	monthlyTransactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
		StartDate: &monthlyRange.Start,
		EndDate:   &monthlyRange.End,
	})
	if err != nil {
		return nil, err
	}

	dailyTransactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
		StartDate: &dailyRange.Start,
		EndDate:   &dailyRange.End,
	})
	if err != nil {
		return nil, err
	}

	activeGoals, err := service.findActiveGoals(ctx)
	if err != nil {
		return nil, err
	}

	_, monthlyExpense := calculateTotals(monthlyTransactions)
	alerts := make([]dto.FinancialAlertResponse, 0)

	if options.MonthlyExpenseLimit > 0 && monthlyExpense > options.MonthlyExpenseLimit {
		alerts = append(alerts, dto.FinancialAlertResponse{
			ID:       "monthly-expense-limit",
			Type:     "monthly_expense_limit",
			Severity: "warning",
			Title:    "Gasto mensal acima do limite",
			Message:  "Suas despesas do mês passaram do limite definido.",
		})
	}

	if monthlyExpense > 0 {
		for _, category := range expenseCategories(monthlyTransactions) {
			if category.Percentage >= options.CategoryExpenseShareLimit {
				alerts = append(alerts, dto.FinancialAlertResponse{
					ID:       "category-share-" + normalizeAlertID(category.Category),
					Type:     "category_expense_share",
					Severity: "warning",
					Title:    "Categoria consumindo muito",
					Message:  fmt.Sprintf("%s concentra %.0f%% dos seus gastos do mês.", category.Category, category.Percentage),
				})
				break
			}
		}
	}

	if len(dailyTransactions) == 0 {
		alerts = append(alerts, dto.FinancialAlertResponse{
			ID:       "daily-update-missing",
			Type:     "daily_update_missing",
			Severity: "info",
			Title:    "Atualização do dia pendente",
			Message:  "Registre suas entradas e gastos de hoje para manter o controle em dia.",
		})
	}

	alerts = append(alerts, goalDeadlineAlerts(activeGoals, parsedDate)...)

	return alerts, nil
}

func (service *FinanceService) findTransactionsForPeriod(ctx context.Context, period string, date string) (dateRange, string, []model.Transaction, error) {
	periodRange, normalizedPeriod, err := buildPeriodRange(period, date)
	if err != nil {
		return dateRange{}, "", nil, err
	}

	transactions, err := service.transactionRepository.Find(ctx, repository.TransactionFilter{
		StartDate: &periodRange.Start,
		EndDate:   &periodRange.End,
	})
	if err != nil {
		return dateRange{}, "", nil, err
	}

	return periodRange, normalizedPeriod, transactions, nil
}

func (service *FinanceService) findActiveGoals(ctx context.Context) ([]model.Goal, error) {
	if service.goalRepository == nil {
		return []model.Goal{}, nil
	}

	return service.goalRepository.Find(ctx, repository.GoalFilter{Status: model.GoalStatusActive})
}

func parseAlertOptions(monthlyExpenseLimit string, categoryExpenseShareLimit string) (alertOptions, error) {
	monthlyLimit, err := parseOptionalInt64(monthlyExpenseLimit, "O limite mensal deve ser um valor em centavos maior ou igual a zero.")
	if err != nil {
		return alertOptions{}, err
	}

	categoryLimit := float64(defaultCategoryExpenseShareLimit)
	trimmedCategoryLimit := strings.TrimSpace(categoryExpenseShareLimit)
	if trimmedCategoryLimit != "" {
		parsedCategoryLimit, err := strconv.ParseFloat(trimmedCategoryLimit, 64)
		if err != nil || parsedCategoryLimit <= 0 || parsedCategoryLimit > 100 {
			return alertOptions{}, newValidationError("O percentual por categoria deve estar entre 1 e 100.")
		}
		categoryLimit = parsedCategoryLimit
	}

	return alertOptions{
		MonthlyExpenseLimit:       monthlyLimit,
		CategoryExpenseShareLimit: categoryLimit,
	}, nil
}

func parseOptionalInt64(value string, message string) (int64, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return 0, nil
	}

	parsedValue, err := strconv.ParseInt(trimmedValue, 10, 64)
	if err != nil || parsedValue < 0 {
		return 0, newValidationError(message)
	}

	return parsedValue, nil
}

func expenseCategories(transactions []model.Transaction) []dto.CategoryAmountResponse {
	categoryTotals := map[string]int64{}
	var totalExpense int64

	for _, transaction := range transactions {
		if transaction.Type != model.TransactionTypeExpense {
			continue
		}

		category := strings.TrimSpace(transaction.Category)
		if category == "" {
			category = "Sem categoria"
		}

		categoryTotals[category] += transaction.Amount
		totalExpense += transaction.Amount
	}

	categories := make([]dto.CategoryAmountResponse, 0, len(categoryTotals))
	for category, total := range categoryTotals {
		percentage := 0.0
		if totalExpense > 0 {
			percentage = float64(total) / float64(totalExpense) * 100
		}

		categories = append(categories, dto.CategoryAmountResponse{
			Category:   category,
			Total:      total,
			Percentage: percentage,
		})
	}

	sort.Slice(categories, func(i int, j int) bool {
		if categories[i].Total == categories[j].Total {
			return categories[i].Category < categories[j].Category
		}

		return categories[i].Total > categories[j].Total
	})

	return categories
}

func topExpenseCategory(transactions []model.Transaction) *dto.CategoryAmountResponse {
	categories := expenseCategories(transactions)
	if len(categories) == 0 {
		return nil
	}

	topCategory := categories[0]
	return &topCategory
}

func buildChartBuckets(periodRange dateRange, period string) []chartBucket {
	switch period {
	case PeriodWeekly:
		return buildDailyBuckets(periodRange)
	case PeriodMonthly:
		return buildWeeklyBuckets(periodRange)
	case PeriodAnnual:
		return buildMonthlyBuckets(periodRange)
	default:
		return []chartBucket{{
			Label: "Dia",
			Start: periodRange.Start,
			End:   periodRange.End,
		}}
	}
}

func buildDailyBuckets(periodRange dateRange) []chartBucket {
	buckets := make([]chartBucket, 0, int(periodRange.End.Sub(periodRange.Start).Hours()/24))
	for start := periodRange.Start; start.Before(periodRange.End); start = start.AddDate(0, 0, 1) {
		buckets = append(buckets, chartBucket{
			Label: start.Format("02/01"),
			Start: start,
			End:   start.AddDate(0, 0, 1),
		})
	}

	return buckets
}

func buildWeeklyBuckets(periodRange dateRange) []chartBucket {
	buckets := make([]chartBucket, 0, 5)
	index := 1
	for start := periodRange.Start; start.Before(periodRange.End); start = start.AddDate(0, 0, 7) {
		end := start.AddDate(0, 0, 7)
		if end.After(periodRange.End) {
			end = periodRange.End
		}

		buckets = append(buckets, chartBucket{
			Label: fmt.Sprintf("Semana %d", index),
			Start: start,
			End:   end,
		})
		index++
	}

	return buckets
}

func buildMonthlyBuckets(periodRange dateRange) []chartBucket {
	buckets := make([]chartBucket, 0, 12)
	for start := periodRange.Start; start.Before(periodRange.End); start = start.AddDate(0, 1, 0) {
		buckets = append(buckets, chartBucket{
			Label: start.Format("01/2006"),
			Start: start,
			End:   start.AddDate(0, 1, 0),
		})
	}

	return buckets
}

func filterTransactionsByRange(transactions []model.Transaction, start time.Time, end time.Time) []model.Transaction {
	filteredTransactions := make([]model.Transaction, 0)
	for _, transaction := range transactions {
		if transaction.TransactionDate.Before(start) || !transaction.TransactionDate.Before(end) {
			continue
		}

		filteredTransactions = append(filteredTransactions, transaction)
	}

	return filteredTransactions
}

func goalDeadlineAlerts(goals []model.Goal, referenceDate time.Time) []dto.FinancialAlertResponse {
	alerts := make([]dto.FinancialAlertResponse, 0)
	dueSoonLimit := referenceDate.AddDate(0, 0, goalDueSoonDays)

	for _, goal := range goals {
		if goal.DueDate.Before(referenceDate) {
			alerts = append(alerts, dto.FinancialAlertResponse{
				ID:       "goal-overdue-" + goal.ID.Hex(),
				Type:     "goal_overdue",
				Severity: "danger",
				Title:    "Meta vencida",
				Message:  fmt.Sprintf("A meta %q passou do prazo.", goal.Description),
			})
			continue
		}

		if !goal.DueDate.After(dueSoonLimit) {
			alerts = append(alerts, dto.FinancialAlertResponse{
				ID:       "goal-due-soon-" + goal.ID.Hex(),
				Type:     "goal_due_soon",
				Severity: "warning",
				Title:    "Meta perto do prazo",
				Message:  fmt.Sprintf("A meta %q vence em breve.", goal.Description),
			})
		}
	}

	return alerts
}

func normalizeAlertID(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "/", "-")

	return normalized
}

func formatPeriodBounds(periodRange dateRange) (string, string) {
	return periodRange.Start.Format(dto.DateLayout), periodRange.End.AddDate(0, 0, -1).Format(dto.DateLayout)
}
