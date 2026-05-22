package controller

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/dto"
	"github.com/Joaopdiasventura/life-compass-server/internal/finance/service"
)

type TransactionController struct {
	service *service.TransactionService
}

func NewTransactionController(service *service.TransactionService) *TransactionController {
	return &TransactionController{
		service: service,
	}
}

func (controller *TransactionController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/transactions", controller.handleTransactions)
	mux.HandleFunc("/transactions/daily", controller.handleDailyTransactions)
	mux.HandleFunc("/transactions/weekly", controller.handleWeeklyTransactions)
	mux.HandleFunc("/transactions/monthly", controller.handleMonthlyTransactions)
	mux.HandleFunc("/transactions/annual", controller.handleAnnualTransactions)
	mux.HandleFunc("/financial-summary", controller.handleFinancialSummary)
	mux.HandleFunc("/balance", controller.handleBalance)
	mux.HandleFunc("/", controller.handleNotFound)
}

func (controller *TransactionController) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		controller.createTransaction(w, r)
	case http.MethodGet:
		controller.listTransactions(w, r)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (controller *TransactionController) createTransaction(w http.ResponseWriter, r *http.Request) {
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

func (controller *TransactionController) listTransactions(w http.ResponseWriter, r *http.Request) {
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

func (controller *TransactionController) handleDailyTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodDaily, "Transações diárias consultadas com sucesso.")
}

func (controller *TransactionController) handleWeeklyTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodWeekly, "Transações semanais consultadas com sucesso.")
}

func (controller *TransactionController) handleMonthlyTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodMonthly, "Transações mensais consultadas com sucesso.")
}

func (controller *TransactionController) handleAnnualTransactions(w http.ResponseWriter, r *http.Request) {
	controller.listTransactionsByPeriod(w, r, service.PeriodAnnual, "Transações anuais consultadas com sucesso.")
}

func (controller *TransactionController) listTransactionsByPeriod(w http.ResponseWriter, r *http.Request, period string, message string) {
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

func (controller *TransactionController) handleFinancialSummary(w http.ResponseWriter, r *http.Request) {
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

func (controller *TransactionController) handleBalance(w http.ResponseWriter, r *http.Request) {
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

func (controller *TransactionController) handleNotFound(w http.ResponseWriter, r *http.Request) {
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
