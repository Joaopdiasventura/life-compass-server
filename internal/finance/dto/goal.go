package dto

import (
	"time"

	"github.com/Joaopdiasventura/life-compass-server/internal/finance/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type CreateGoalRequest struct {
	Description   string `json:"description"`
	TargetAmount  int64  `json:"targetAmount"`
	CurrentAmount int64  `json:"currentAmount"`
	DueDate       string `json:"dueDate"`
}

type UpdateGoalRequest struct {
	Description   string `json:"description"`
	TargetAmount  int64  `json:"targetAmount"`
	CurrentAmount int64  `json:"currentAmount"`
	DueDate       string `json:"dueDate"`
}

type GoalResponse struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	TargetAmount  int64  `json:"targetAmount"`
	CurrentAmount int64  `json:"currentAmount"`
	DueDate       string `json:"dueDate"`
	Status        string `json:"status"`
	CompletedAt   string `json:"completedAt,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

func NewGoalResponse(goal model.Goal) GoalResponse {
	id := ""
	if goal.ID != bson.NilObjectID {
		id = goal.ID.Hex()
	}

	completedAt := ""
	if goal.CompletedAt != nil {
		completedAt = formatTime(*goal.CompletedAt, time.RFC3339)
	}

	return GoalResponse{
		ID:            id,
		Description:   goal.Description,
		TargetAmount:  goal.TargetAmount,
		CurrentAmount: goal.CurrentAmount,
		DueDate:       formatTime(goal.DueDate, DateLayout),
		Status:        goal.Status,
		CompletedAt:   completedAt,
		CreatedAt:     formatTime(goal.CreatedAt, time.RFC3339),
		UpdatedAt:     formatTime(goal.UpdatedAt, time.RFC3339),
	}
}

func NewGoalResponses(goals []model.Goal) []GoalResponse {
	responses := make([]GoalResponse, 0, len(goals))
	for _, goal := range goals {
		responses = append(responses, NewGoalResponse(goal))
	}

	return responses
}
