package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/service"
)

type FinanceController struct {
	service *service.FinanceService
}

func NewFinanceController(service *service.FinanceService) *FinanceController {
	return &FinanceController{
		service: service,
	}
}

func (controller *FinanceController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/transactions", controller.handleTransactions)
	mux.HandleFunc("/transactions/daily", controller.handleDailyTransactions)
	mux.HandleFunc("/transactions/weekly", controller.handleWeeklyTransactions)
	mux.HandleFunc("/transactions/monthly", controller.handleMonthlyTransactions)
	mux.HandleFunc("/transactions/annual", controller.handleAnnualTransactions)
	mux.HandleFunc("/transactions/", controller.handleTransactionByID)
	mux.HandleFunc("/financial-summary", controller.handleFinancialSummary)
	mux.HandleFunc("/financial-dashboard", controller.handleFinancialDashboard)
	mux.HandleFunc("/financial-charts/category-expenses", controller.handleCategoryExpensesChart)
	mux.HandleFunc("/financial-charts/income-expense", controller.handleIncomeExpenseChart)
	mux.HandleFunc("/financial-charts/balance-evolution", controller.handleBalanceEvolutionChart)
	mux.HandleFunc("/financial-charts/period-expenses", controller.handlePeriodExpensesChart)
	mux.HandleFunc("/financial-alerts", controller.handleFinancialAlerts)
	mux.HandleFunc("/balance", controller.handleBalance)
	mux.HandleFunc("/goals", controller.handleGoals)
	mux.HandleFunc("/goals/", controller.handleGoalByID)
	mux.HandleFunc("/", controller.handleNotFound)
}

func (controller *FinanceController) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		controller.createTransaction(w, r)
	case http.MethodGet:
		controller.listTransactions(w, r)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (controller *FinanceController) createTransaction(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateTransactionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Corpo da requisição inválido.")
		return
	}

	transaction, err := controller.service.CreateTransaction(r.Context(), request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.APIResponse{
		Message: "Transação criada com sucesso.",
		Data:    transaction,
	})
}

func (controller *FinanceController) listTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := controller.service.ListTransactions(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Transações consultadas com sucesso.",
		Data:    transactions,
	})
}

func (controller *FinanceController) handleTransactionByID(w http.ResponseWriter, r *http.Request) {
	transactionPath := strings.TrimPrefix(r.URL.Path, "/transactions/")
	segments := strings.Split(strings.Trim(transactionPath, "/"), "/")
	if len(segments) != 1 || segments[0] == "" {
		writeError(w, http.StatusNotFound, "Rota não encontrada.")
		return
	}

	switch r.Method {
	case http.MethodGet:
		controller.getTransaction(w, r, segments[0])
	case http.MethodPut:
		controller.updateTransaction(w, r, segments[0])
	default:
		writeMethodNotAllowed(w, "GET, PUT")
	}
}

func (controller *FinanceController) getTransaction(w http.ResponseWriter, r *http.Request, id string) {
	transaction, err := controller.service.GetTransaction(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Transação consultada com sucesso.",
		Data:    transaction,
	})
}

func (controller *FinanceController) updateTransaction(w http.ResponseWriter, r *http.Request, id string) {
	var request dto.UpdateTransactionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Corpo da requisição inválido.")
		return
	}

	transaction, err := controller.service.UpdateTransaction(r.Context(), id, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Transação atualizada com sucesso.",
		Data:    transaction,
	})
}

func (controller *FinanceController) handleDailyTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodDaily, "Transações diárias consultadas com sucesso.")
}

func (controller *FinanceController) handleWeeklyTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodWeekly, "Transações semanais consultadas com sucesso.")
}

func (controller *FinanceController) handleMonthlyTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodMonthly, "Transações mensais consultadas com sucesso.")
}

func (controller *FinanceController) handleAnnualTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodAnnual, "Transações anuais consultadas com sucesso.")
}

func (controller *FinanceController) listTransactionsByPeriod(w http.ResponseWriter, r *http.Request, period string, message string) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	transactions, err := controller.service.ListTransactionsByPeriod(
		r.Context(),
		period,
		r.URL.Query().Get("date"),
		r.URL.Query().Get("type"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: message,
		Data:    transactions,
	})
}

func (controller *FinanceController) handleFinancialSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	summary, err := controller.service.GetFinancialSummary(
		r.Context(),
		r.URL.Query().Get("period"),
		r.URL.Query().Get("date"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Resumo financeiro consultado com sucesso.",
		Data:    summary,
	})
}

func (controller *FinanceController) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	balance, err := controller.service.GetTotalBalance(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Saldo total consultado com sucesso.",
		Data:    balance,
	})
}

func (controller *FinanceController) handleFinancialDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	dashboard, err := controller.service.GetFinancialDashboard(
		r.Context(),
		r.URL.Query().Get("period"),
		r.URL.Query().Get("date"),
		r.URL.Query().Get("monthlyExpenseLimit"),
		r.URL.Query().Get("categoryExpenseShareLimit"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Dashboard financeiro consultado com sucesso.",
		Data:    dashboard,
	})
}

func (controller *FinanceController) handleCategoryExpensesChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	chart, err := controller.service.GetCategoryExpensesChart(
		r.Context(),
		r.URL.Query().Get("period"),
		r.URL.Query().Get("date"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Gráfico de gastos por categoria consultado com sucesso.",
		Data:    chart,
	})
}

func (controller *FinanceController) handleIncomeExpenseChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	chart, err := controller.service.GetIncomeExpenseChart(
		r.Context(),
		r.URL.Query().Get("period"),
		r.URL.Query().Get("date"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Gráfico de entradas e gastos consultado com sucesso.",
		Data:    chart,
	})
}

func (controller *FinanceController) handleBalanceEvolutionChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	chart, err := controller.service.GetBalanceEvolutionChart(
		r.Context(),
		r.URL.Query().Get("period"),
		r.URL.Query().Get("date"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Evolução do saldo consultada com sucesso.",
		Data:    chart,
	})
}

func (controller *FinanceController) handlePeriodExpensesChart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	chart, err := controller.service.GetPeriodExpensesChart(
		r.Context(),
		r.URL.Query().Get("period"),
		r.URL.Query().Get("date"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Gastos por período consultados com sucesso.",
		Data:    chart,
	})
}

func (controller *FinanceController) handleFinancialAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	alerts, err := controller.service.GetFinancialAlerts(
		r.Context(),
		r.URL.Query().Get("date"),
		r.URL.Query().Get("monthlyExpenseLimit"),
		r.URL.Query().Get("categoryExpenseShareLimit"),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Alertas financeiros consultados com sucesso.",
		Data:    alerts,
	})
}

func (controller *FinanceController) handleGoals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		controller.createGoal(w, r)
	case http.MethodGet:
		controller.listGoals(w, r)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (controller *FinanceController) createGoal(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateGoalRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Corpo da requisição inválido.")
		return
	}

	goal, err := controller.service.CreateGoal(r.Context(), request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.APIResponse{
		Message: "Meta criada com sucesso.",
		Data:    goal,
	})
}

func (controller *FinanceController) listGoals(w http.ResponseWriter, r *http.Request) {
	goals, err := controller.service.ListGoals(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Metas consultadas com sucesso.",
		Data:    goals,
	})
}

func (controller *FinanceController) handleGoalByID(w http.ResponseWriter, r *http.Request) {
	goalPath := strings.TrimPrefix(r.URL.Path, "/goals/")
	segments := strings.Split(strings.Trim(goalPath, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		writeError(w, http.StatusNotFound, "Rota não encontrada.")
		return
	}

	if len(segments) == 1 && r.Method == http.MethodPut {
		controller.updateGoal(w, r, segments[0])
		return
	}

	if len(segments) == 2 && segments[1] == "complete" && r.Method == http.MethodPatch {
		controller.completeGoal(w, r, segments[0])
		return
	}

	if len(segments) == 1 {
		writeMethodNotAllowed(w, "PUT")
		return
	}

	if len(segments) == 2 && segments[1] == "complete" {
		writeMethodNotAllowed(w, "PATCH")
		return
	}

	writeError(w, http.StatusNotFound, "Rota não encontrada.")
}

func (controller *FinanceController) updateGoal(w http.ResponseWriter, r *http.Request, id string) {
	var request dto.UpdateGoalRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Corpo da requisição inválido.")
		return
	}

	goal, err := controller.service.UpdateGoal(r.Context(), id, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Meta atualizada com sucesso.",
		Data:    goal,
	})
}

func (controller *FinanceController) completeGoal(w http.ResponseWriter, r *http.Request, id string) {
	goal, err := controller.service.CompleteGoal(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.APIResponse{
		Message: "Meta concluída com sucesso.",
		Data:    goal,
	})
}

func (controller *FinanceController) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "Rota não encontrada.")
}

func decodeJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(value); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return err
	}

	return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	if service.IsValidationError(err) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeError(w, http.StatusInternalServerError, "Não foi possível processar a solicitação.")
}

func writeMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, http.StatusMethodNotAllowed, "Método não permitido.")
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, dto.ErrorResponse{
		Message: message,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
