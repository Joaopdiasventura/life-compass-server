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

func (service *FinanceService) CreateGoal(ctx context.Context, request dto.CreateGoalRequest) (dto.GoalResponse, error) {
	if service.goalRepository == nil {
		return dto.GoalResponse{}, errors.New("goal repository is not configured")
	}

	description, targetAmount, currentAmount, dueDate, err := validateGoalFields(
		request.Description,
		request.TargetAmount,
		request.CurrentAmount,
		request.DueDate,
	)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	now := time.Now().UTC()
	goal := model.Goal{
		Description:   description,
		TargetAmount:  targetAmount,
		CurrentAmount: currentAmount,
		DueDate:       dueDate,
		Status:        model.GoalStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	createdGoal, err := service.goalRepository.Create(ctx, goal)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	return dto.NewGoalResponse(createdGoal), nil
}

func (service *FinanceService) ListGoals(ctx context.Context, status string) ([]dto.GoalResponse, error) {
	if service.goalRepository == nil {
		return nil, errors.New("goal repository is not configured")
	}

	normalizedStatus, err := validateOptionalGoalStatus(status)
	if err != nil {
		return nil, err
	}

	goals, err := service.goalRepository.Find(ctx, repository.GoalFilter{Status: normalizedStatus})
	if err != nil {
		return nil, err
	}

	return dto.NewGoalResponses(goals), nil
}

func (service *FinanceService) UpdateGoal(ctx context.Context, id string, request dto.UpdateGoalRequest) (dto.GoalResponse, error) {
	if service.goalRepository == nil {
		return dto.GoalResponse{}, errors.New("goal repository is not configured")
	}

	goalID, err := parseGoalID(id)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	description, targetAmount, currentAmount, dueDate, err := validateGoalFields(
		request.Description,
		request.TargetAmount,
		request.CurrentAmount,
		request.DueDate,
	)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	goal, err := service.goalRepository.FindByID(ctx, goalID)
	if err != nil {
		return dto.GoalResponse{}, mapGoalNotFoundError(err)
	}

	goal.Description = description
	goal.TargetAmount = targetAmount
	goal.CurrentAmount = currentAmount
	goal.DueDate = dueDate
	goal.UpdatedAt = time.Now().UTC()

	updatedGoal, err := service.goalRepository.Update(ctx, goal)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	return dto.NewGoalResponse(updatedGoal), nil
}

func (service *FinanceService) CompleteGoal(ctx context.Context, id string) (dto.GoalResponse, error) {
	if service.goalRepository == nil {
		return dto.GoalResponse{}, errors.New("goal repository is not configured")
	}

	goalID, err := parseGoalID(id)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	goal, err := service.goalRepository.FindByID(ctx, goalID)
	if err != nil {
		return dto.GoalResponse{}, mapGoalNotFoundError(err)
	}

	now := time.Now().UTC()
	goal.Status = model.GoalStatusCompleted
	if goal.CurrentAmount < goal.TargetAmount {
		goal.CurrentAmount = goal.TargetAmount
	}
	goal.CompletedAt = &now
	goal.UpdatedAt = now

	updatedGoal, err := service.goalRepository.Update(ctx, goal)
	if err != nil {
		return dto.GoalResponse{}, err
	}

	return dto.NewGoalResponse(updatedGoal), nil
}

func validateGoalFields(description string, targetAmount int64, currentAmount int64, dueDate string) (string, int64, int64, time.Time, error) {
	trimmedDescription := strings.TrimSpace(description)
	if trimmedDescription == "" {
		return "", 0, 0, time.Time{}, newValidationError("Descrição da meta é obrigatória.")
	}

	if targetAmount <= 0 {
		return "", 0, 0, time.Time{}, newValidationError("O valor alvo da meta deve ser maior que zero.")
	}

	if currentAmount < 0 {
		return "", 0, 0, time.Time{}, newValidationError("O valor atual da meta não pode ser negativo.")
	}

	parsedDueDate, err := parseRequiredDate(dueDate, "Prazo da meta é obrigatório.")
	if err != nil {
		return "", 0, 0, time.Time{}, err
	}

	return trimmedDescription, targetAmount, currentAmount, parsedDueDate, nil
}

func validateOptionalGoalStatus(status string) (string, error) {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	if normalizedStatus == "" {
		return "", nil
	}

	switch normalizedStatus {
	case model.GoalStatusActive, model.GoalStatusCompleted:
		return normalizedStatus, nil
	default:
		return "", newValidationError("Status da meta inválido. Use active ou completed.")
	}
}

func parseGoalID(id string) (bson.ObjectID, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return bson.NilObjectID, newValidationError("ID da meta é obrigatório.")
	}

	goalID, err := bson.ObjectIDFromHex(trimmedID)
	if err != nil {
		return bson.NilObjectID, newValidationError("ID da meta inválido.")
	}

	return goalID, nil
}

func mapGoalNotFoundError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return newValidationError("Meta não encontrada.")
	}

	return err
}
