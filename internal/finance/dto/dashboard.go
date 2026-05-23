package dto

type CategoryAmountResponse struct {
	Category   string  `json:"category"`
	Total      int64   `json:"total"`
	Percentage float64 `json:"percentage"`
}

type FinancialAlertResponse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}

type FinancialDashboardResponse struct {
	Period             string                   `json:"period"`
	StartDate          string                   `json:"startDate"`
	EndDate            string                   `json:"endDate"`
	Balance            int64                    `json:"balance"`
	TotalIncome        int64                    `json:"totalIncome"`
	TotalExpense       int64                    `json:"totalExpense"`
	Savings            int64                    `json:"savings"`
	TransactionCount   int                      `json:"transactionCount"`
	TopExpenseCategory *CategoryAmountResponse  `json:"topExpenseCategory"`
	Alerts             []FinancialAlertResponse `json:"alerts"`
}

type CategoryExpensesChartResponse struct {
	Period     string                   `json:"period"`
	StartDate  string                   `json:"startDate"`
	EndDate    string                   `json:"endDate"`
	Categories []CategoryAmountResponse `json:"categories"`
}

type IncomeExpensePointResponse struct {
	Label        string `json:"label"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	TotalIncome  int64  `json:"totalIncome"`
	TotalExpense int64  `json:"totalExpense"`
	Balance      int64  `json:"balance"`
}

type IncomeExpenseChartResponse struct {
	Period    string                       `json:"period"`
	StartDate string                       `json:"startDate"`
	EndDate   string                       `json:"endDate"`
	Points    []IncomeExpensePointResponse `json:"points"`
}

type BalanceEvolutionPointResponse struct {
	Label        string `json:"label"`
	Date         string `json:"date"`
	Balance      int64  `json:"balance"`
	TotalIncome  int64  `json:"totalIncome"`
	TotalExpense int64  `json:"totalExpense"`
}

type BalanceEvolutionChartResponse struct {
	Period    string                          `json:"period"`
	StartDate string                          `json:"startDate"`
	EndDate   string                          `json:"endDate"`
	Points    []BalanceEvolutionPointResponse `json:"points"`
}

type PeriodExpensePointResponse struct {
	Label        string `json:"label"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
	TotalExpense int64  `json:"totalExpense"`
}

type PeriodExpensesChartResponse struct {
	Period    string                       `json:"period"`
	StartDate string                       `json:"startDate"`
	EndDate   string                       `json:"endDate"`
	Points    []PeriodExpensePointResponse `json:"points"`
}
