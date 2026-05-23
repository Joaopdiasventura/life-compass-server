package service

import "github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"

type FinanceServiceDependencies struct {
	TransactionRepository repository.TransactionRepository
	GoalRepository        repository.GoalRepository
}

type FinanceService struct {
	transactionRepository repository.TransactionRepository
	goalRepository        repository.GoalRepository
}

func NewFinanceService(dependencies FinanceServiceDependencies) *FinanceService {
	return &FinanceService{
		transactionRepository: dependencies.TransactionRepository,
		goalRepository:        dependencies.GoalRepository,
	}
}
